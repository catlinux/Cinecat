package library

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mediaItemID(t *testing.T, db *sql.DB, path string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(`SELECT id FROM media_items WHERE path = ?`, path).Scan(&id); err != nil {
		t.Fatalf("id de %q: %v", path, err)
	}
	return id
}

func TestScannerReenlazaPeliculaRenombradaConservandoProgreso(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	oldPath := filepath.Join(root, "Pelicula.mp4")
	newPath := filepath.Join(root, "Pelicula.mkv")
	writeFile(t, oldPath)

	libraryID := createLibrary(t, db, root)
	addScannerLibraryPath(t, db, libraryID, root)
	scanner := New(db).WithStableAge(0)
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("primer Scan: %v", err)
	}
	oldID := mediaItemID(t, db, oldPath)

	if _, err := db.Exec(`INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`,
		"ana", "x", "user"); err != nil {
		t.Fatalf("crear usuario: %v", err)
	}
	userID := progressUserID(t, db, "ana")
	setProgressRow(t, db, userID, oldID, 120, 600, "2026-10-26T10:00:00Z")

	if err := os.Remove(oldPath); err != nil {
		t.Fatalf("borrar original: %v", err)
	}
	writeFile(t, newPath)

	res, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan tras renombrar: %v", err)
	}
	if res.Relinked != 1 {
		t.Fatalf("Relinked = %d, se esperaba 1", res.Relinked)
	}
	if got := mediaItemID(t, db, newPath); got != oldID {
		t.Fatalf("id tras reenlace = %d, se esperaba %d", got, oldID)
	}
	if count := countRows(t, &libraryTestServer{db: db},
		`SELECT count(*) FROM playback_progress WHERE media_item_id = ?`, oldID); count != 1 {
		t.Fatalf("el progreso debe seguir ligado al id %d: %d", oldID, count)
	}
}

func TestScannerReenlazaMismoTituloYAno(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	oldPath := filepath.Join(root, "Pelicula (2010) x264.mp4")
	newPath := filepath.Join(root, "Pelicula (2010) x265.mkv")
	writeFile(t, oldPath)

	libraryID := createLibrary(t, db, root)
	addScannerLibraryPath(t, db, libraryID, root)
	scanner := New(db).WithStableAge(0)
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("primer Scan: %v", err)
	}
	oldID := mediaItemID(t, db, oldPath)

	if err := os.Remove(oldPath); err != nil {
		t.Fatalf("borrar original: %v", err)
	}
	writeFile(t, newPath)

	res, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan tras cambiar códec: %v", err)
	}
	if res.Relinked != 1 {
		t.Fatalf("Relinked = %d, se esperaba 1", res.Relinked)
	}
	if got := mediaItemID(t, db, newPath); got != oldID {
		t.Fatalf("id tras reenlace = %d, se esperaba %d", got, oldID)
	}
}

func TestScannerReenlazaEpisodioRenombrado(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	oldPath := filepath.Join(root, "Serie", "Season 1", "Serie - 1x02 - Piloto.mp4")
	newPath := filepath.Join(root, "Serie", "Season 1", "Serie - S01E02 - Piloto.mkv")
	writeFile(t, oldPath)

	libraryID := createLibrary(t, db, root)
	addScannerLibraryPath(t, db, libraryID, root)
	scanner := New(db).WithStableAge(0)
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("primer Scan: %v", err)
	}
	oldID := mediaItemID(t, db, oldPath)

	if err := os.Remove(oldPath); err != nil {
		t.Fatalf("borrar original: %v", err)
	}
	writeFile(t, newPath)

	res, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan tras renombrar episodio: %v", err)
	}
	if res.Relinked != 1 {
		t.Fatalf("Relinked = %d, se esperaba 1", res.Relinked)
	}
	if got := mediaItemID(t, db, newPath); got != oldID {
		t.Fatalf("id tras reenlace = %d, se esperaba %d", got, oldID)
	}
}

func TestScannerNoReenlazaConCandidatasAmbiguas(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	oldA := filepath.Join(root, "Pelicula (2010) a.mp4")
	oldB := filepath.Join(root, "Pelicula (2010) b.mp4")
	newPath := filepath.Join(root, "Pelicula (2010) c.mkv")
	writeFile(t, oldA)
	writeFile(t, oldB)

	libraryID := createLibrary(t, db, root)
	addScannerLibraryPath(t, db, libraryID, root)
	scanner := New(db).WithStableAge(0)
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("primer Scan: %v", err)
	}
	idA := mediaItemID(t, db, oldA)
	idB := mediaItemID(t, db, oldB)

	if err := os.Remove(oldA); err != nil {
		t.Fatalf("borrar candidata A: %v", err)
	}
	if err := os.Remove(oldB); err != nil {
		t.Fatalf("borrar candidata B: %v", err)
	}
	writeFile(t, newPath)

	res, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan ambiguo: %v", err)
	}
	if res.Relinked != 0 {
		t.Fatalf("Relinked = %d, con dos candidatas no debe reenlazar", res.Relinked)
	}
	newID := mediaItemID(t, db, newPath)
	if newID == idA || newID == idB {
		t.Fatalf("la ficha nueva no debe reutilizar una candidata ambigua: %d (%d, %d)", newID, idA, idB)
	}
}

func TestScannerIgnoraArchivoRecienteYNoPodaSuCarpeta(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	oldPath := filepath.Join(root, "Pelicula (2010).mp4")
	otherPath := filepath.Join(root, "Otra (2011).mkv")
	newPath := filepath.Join(root, "Pelicula (2010).mkv")
	writeFile(t, oldPath)
	writeFile(t, otherPath)

	libraryID := createLibrary(t, db, root)
	addScannerLibraryPath(t, db, libraryID, root)
	if _, err := New(db).WithStableAge(0).Scan(ctx); err != nil {
		t.Fatalf("primer Scan: %v", err)
	}
	oldID := mediaItemID(t, db, oldPath)

	if err := os.Remove(oldPath); err != nil {
		t.Fatalf("borrar original: %v", err)
	}
	if err := os.WriteFile(newPath, []byte("archivo en conversión"), 0o644); err != nil {
		t.Fatalf("crear archivo reciente: %v", err)
	}

	res, err := New(db).WithStableAge(time.Hour).Scan(ctx)
	if err != nil {
		t.Fatalf("Scan con archivo reciente: %v", err)
	}
	if res.Relinked != 0 {
		t.Fatalf("Relinked = %d, un archivo reciente no debe reenlazarse", res.Relinked)
	}
	if count := countRows(t, &libraryTestServer{db: db},
		`SELECT count(*) FROM media_items WHERE path = ?`, newPath); count != 0 {
		t.Fatalf("el archivo reciente no debe indexarse: %d", count)
	}
	if count := countRows(t, &libraryTestServer{db: db},
		`SELECT count(*) FROM media_items WHERE id = ?`, oldID); count != 1 {
		t.Fatalf("la ficha antigua de la carpeta inestable no debe podarse: %d", count)
	}
}

func TestScannerNoReenlazaSiElOriginalSigueExistiendo(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	oldPath := filepath.Join(root, "Pelicula (2010) x264.mp4")
	newPath := filepath.Join(root, "Pelicula (2010) x265.mkv")
	writeFile(t, oldPath)

	libraryID := createLibrary(t, db, root)
	addScannerLibraryPath(t, db, libraryID, root)
	scanner := New(db).WithStableAge(0)
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("primer Scan: %v", err)
	}
	oldID := mediaItemID(t, db, oldPath)

	writeFile(t, newPath)
	res, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan con original y copia: %v", err)
	}
	if res.Relinked != 0 {
		t.Fatalf("Relinked = %d, con el original presente no debe reenlazar", res.Relinked)
	}
	if got := mediaItemID(t, db, oldPath); got != oldID {
		t.Fatalf("la ficha original cambió: %d, se esperaba %d", got, oldID)
	}
	newID := mediaItemID(t, db, newPath)
	if newID == oldID {
		t.Fatalf("la copia debe tener una ficha nueva, no reutilizar %d", oldID)
	}
}

func TestScannerTrasladaEstadoCuandoConvivenYSePodaElOriginal(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	oldPath := filepath.Join(root, "Pelicula (2010) x264.mp4")
	newPath := filepath.Join(root, "Pelicula (2010) x265.mkv")
	writeFile(t, oldPath)

	libraryID := createLibrary(t, db, root)
	addScannerLibraryPath(t, db, libraryID, root)
	scanner := New(db).WithStableAge(0)
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("primer Scan: %v", err)
	}
	oldID := mediaItemID(t, db, oldPath)
	if _, err := db.Exec(`INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`,
		"ana", "x", "user"); err != nil {
		t.Fatalf("crear usuario: %v", err)
	}
	userID := progressUserID(t, db, "ana")
	setProgressRow(t, db, userID, oldID, 120, 600, "2026-10-26T10:00:00Z")

	writeFile(t, newPath)
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("Scan con ambos archivos: %v", err)
	}
	newID := mediaItemID(t, db, newPath)
	if err := os.Remove(oldPath); err != nil {
		t.Fatalf("borrar original: %v", err)
	}
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("Scan tras borrar el original: %v", err)
	}
	if count := countRows(t, &libraryTestServer{db: db},
		`SELECT count(*) FROM playback_progress WHERE media_item_id = ?`, newID); count != 1 {
		t.Fatalf("el progreso debe pasar a la ficha nueva %d: %d", newID, count)
	}
}
