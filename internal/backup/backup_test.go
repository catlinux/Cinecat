package backup

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/catlinux/cinecat/internal/store"
)

func openBackupDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data", "cinecat.db")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("crear carpeta de datos: %v", err)
	}
	st, err := store.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close: %v", err)
		}
	})
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if _, err := st.DB().Exec(`CREATE TABLE prueba_copia (valor TEXT NOT NULL)`); err != nil {
		t.Fatalf("crear tabla: %v", err)
	}
	if _, err := st.DB().Exec(`INSERT INTO prueba_copia (valor) VALUES ('contenido')`); err != nil {
		t.Fatalf("insertar: %v", err)
	}
	return st.DB(), path
}

func TestCreateCopiaValida(t *testing.T) {
	db, path := openBackupDB(t)
	now := time.Date(2026, time.February, 3, 10, 11, 12, 0, time.UTC)
	manager := New(db, path).WithClock(func() time.Time { return now })

	copy, err := manager.Create(context.Background())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if copy.Name != "cinecat-20260203-101112.db" {
		t.Fatalf("nombre = %q", copy.Name)
	}
	if copy.SizeBytes <= 0 {
		t.Fatalf("tamaño = %d", copy.SizeBytes)
	}
	if copy.CreatedAt != now.Format(time.RFC3339) {
		t.Fatalf("created_at = %q", copy.CreatedAt)
	}

	backupPath := filepath.Join(manager.Dir(), copy.Name)
	info, err := os.Stat(backupPath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("permisos = %o", info.Mode().Perm())
	}
	if runtime.GOOS != "windows" {
		dirInfo, err := os.Stat(manager.Dir())
		if err != nil {
			t.Fatalf("Stat directorio: %v", err)
		}
		if dirInfo.Mode().Perm() != 0o700 {
			t.Fatalf("permisos del directorio = %o", dirInfo.Mode().Perm())
		}
	}

	backupStore, err := store.Open(context.Background(), backupPath)
	if err != nil {
		t.Fatalf("abrir copia: %v", err)
	}
	defer backupStore.Close()
	var value string
	if err := backupStore.DB().QueryRow(`SELECT valor FROM prueba_copia`).Scan(&value); err != nil {
		t.Fatalf("leer copia: %v", err)
	}
	if value != "contenido" {
		t.Fatalf("valor = %q", value)
	}

	list, err := manager.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].Name != copy.Name {
		t.Fatalf("lista = %+v", list)
	}
	if manager.LastError() != "" {
		t.Fatalf("last_error = %q", manager.LastError())
	}
}

func TestCreateSoloUnaCopiaALaVez(t *testing.T) {
	db, path := openBackupDB(t)
	manager := New(db, path)

	manager.runMu.Lock()
	_, err := manager.Create(context.Background())
	manager.runMu.Unlock()
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("Create con otra copia en curso = %v", err)
	}

	if _, err := manager.Create(context.Background()); err != nil {
		t.Fatalf("Create tras liberar: %v", err)
	}
}

func TestRetencionConservaLaMasReciente(t *testing.T) {
	db, path := openBackupDB(t)
	now := time.Date(2026, time.February, 10, 12, 0, 0, 0, time.UTC)
	manager := New(db, path).WithClock(func() time.Time { return now }).
		WithSettings(func() bool { return true }, func() int { return 7 })

	writeBackup := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(manager.Dir(), name), []byte(content), 0o600); err != nil {
			t.Fatalf("escribir %s: %v", name, err)
		}
	}
	writeBackup("cinecat-20260101-120000.db", "antigua")
	writeBackup("cinecat-20260205-120000.db", "reciente")

	copy, err := manager.Create(context.Background())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := os.Stat(filepath.Join(manager.Dir(), "cinecat-20260101-120000.db")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("la copia antigua sigue existiendo: %v", err)
	}
	if _, err := os.Stat(filepath.Join(manager.Dir(), "cinecat-20260205-120000.db")); err != nil {
		t.Fatalf("la copia reciente se borró: %v", err)
	}
	if _, err := os.Stat(filepath.Join(manager.Dir(), copy.Name)); err != nil {
		t.Fatalf("la copia nueva no existe: %v", err)
	}
}

func TestRetencionNuncaBorraLaMasRecienteAunqueSeaAntigua(t *testing.T) {
	db, path := openBackupDB(t)
	now := time.Date(2026, time.February, 10, 12, 0, 0, 0, time.UTC)
	manager := New(db, path).WithClock(func() time.Time { return now }).
		WithSettings(func() bool { return true }, func() int { return 1 })

	for _, name := range []string{"cinecat-20260101-120000.db", "cinecat-20260102-120000.db"} {
		if err := os.WriteFile(filepath.Join(manager.Dir(), name), []byte("x"), 0o600); err != nil {
			t.Fatalf("escribir %s: %v", name, err)
		}
	}
	if err := manager.applyRetention(""); err != nil {
		t.Fatalf("applyRetention: %v", err)
	}
	if _, err := os.Stat(filepath.Join(manager.Dir(), "cinecat-20260102-120000.db")); err != nil {
		t.Fatalf("se borró la copia más reciente: %v", err)
	}
	if _, err := os.Stat(filepath.Join(manager.Dir(), "cinecat-20260101-120000.db")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("la copia antigua sigue existiendo: %v", err)
	}
}

func TestBackupDue(t *testing.T) {
	db, path := openBackupDB(t)
	now := time.Date(2026, time.February, 10, 12, 0, 0, 0, time.UTC)
	manager := New(db, path).WithClock(func() time.Time { return now })

	due, wait := manager.backupDue(24 * time.Hour)
	if !due || wait != 0 {
		t.Fatalf("sin copias: due=%v wait=%v", due, wait)
	}
	for _, name := range []string{
		"cinecat-20260210-110000.db",
		"cinecat-20260209-110000.db",
	} {
		if err := os.WriteFile(filepath.Join(manager.Dir(), name), []byte("x"), 0o600); err != nil {
			t.Fatalf("escribir %s: %v", name, err)
		}
	}
	due, wait = manager.backupDue(24 * time.Hour)
	if due || wait != 23*time.Hour {
		t.Fatalf("copia reciente: due=%v wait=%v", due, wait)
	}

	now = now.Add(25 * time.Hour)
	due, _ = manager.backupDue(24 * time.Hour)
	if !due {
		t.Fatal("la copia de hace 25 horas debería estar vencida")
	}
}

func TestRunBackgroundCopiaYEsperaCambios(t *testing.T) {
	db, path := openBackupDB(t)
	var mu sync.Mutex
	enabled := false
	manager := New(db, path).WithSettings(
		func() bool {
			mu.Lock()
			defer mu.Unlock()
			return enabled
		},
		func() int { return 7 },
	)

	changes := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go manager.RunBackground(ctx, changes, time.Hour, 24*time.Hour, time.Hour)

	time.Sleep(20 * time.Millisecond)
	if list, err := manager.List(); err != nil || len(list) != 0 {
		t.Fatalf("copió con el ajuste desactivado: %v, %v", list, err)
	}

	mu.Lock()
	enabled = true
	mu.Unlock()
	changes <- struct{}{}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		list, err := manager.List()
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(list) == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("no se hizo la copia al activar el ajuste")
}
