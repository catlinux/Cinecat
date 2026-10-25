package store

import (
	"context"
	"path/filepath"
	"testing"
)

func openTestStore(t *testing.T, path string) (*Store, context.Context) {
	t.Helper()
	ctx := context.Background()

	s, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return s, ctx
}

func tables(t *testing.T, ctx context.Context, s *Store) map[string]bool {
	t.Helper()
	rows, err := s.DB().QueryContext(ctx,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Fatalf("consultar tablas: %v", err)
	}
	defer rows.Close()

	got := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("leer tabla: %v", err)
		}
		got[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("recorrer tablas: %v", err)
	}
	return got
}

func TestMigrateCreaTablas(t *testing.T) {
	s, ctx := openTestStore(t, filepath.Join(t.TempDir(), "cinecat.db"))

	got := tables(t, ctx, s)
	for _, name := range []string{
		"schema_migrations", "users", "sessions", "libraries", "media_items", "playback_state",
	} {
		if !got[name] {
			t.Errorf("falta la tabla %q", name)
		}
	}
}

func TestMigrateEnMemoria(t *testing.T) {
	s, ctx := openTestStore(t, ":memory:")

	if !tables(t, ctx, s)["media_items"] {
		t.Fatal("falta la tabla media_items en una base en memoria")
	}
}

func TestMigrateEsIdempotente(t *testing.T) {
	s, ctx := openTestStore(t, filepath.Join(t.TempDir(), "cinecat.db"))

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("segunda migración: %v", err)
	}

	var applied int
	if err := s.DB().QueryRowContext(ctx,
		`SELECT count(*) FROM schema_migrations`).Scan(&applied); err != nil {
		t.Fatalf("contar migraciones: %v", err)
	}
	if applied != 1 {
		t.Fatalf("migraciones registradas = %d, se esperaba 1", applied)
	}
}

func TestNoGuardaSecretosEnClaro(t *testing.T) {
	s, ctx := openTestStore(t, filepath.Join(t.TempDir(), "cinecat.db"))

	for table, expected := range map[string]string{
		"users":    "password_hash",
		"sessions": "token_hash",
	} {
		columns := columns(t, ctx, s, table)
		if !columns[expected] {
			t.Errorf("%s: falta la columna %q", table, expected)
		}
		if columns["password"] || columns["token"] {
			t.Errorf("%s: no debe haber columnas password/token en claro", table)
		}
	}
}

func columns(t *testing.T, ctx context.Context, s *Store, table string) map[string]bool {
	t.Helper()
	rows, err := s.DB().QueryContext(ctx, `SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		t.Fatalf("consultar columnas de %s: %v", table, err)
	}
	defer rows.Close()

	got := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("leer columna de %s: %v", table, err)
		}
		got[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("recorrer columnas de %s: %v", table, err)
	}
	return got
}

func TestClavesForaneasActivas(t *testing.T) {
	s, ctx := openTestStore(t, filepath.Join(t.TempDir(), "cinecat.db"))

	_, err := s.DB().ExecContext(ctx,
		`INSERT INTO media_items (library_id, path, title, kind) VALUES (999, '/x.mkv', 'X', 'movie')`)
	if err == nil {
		t.Fatal("se esperaba un error de clave foránea con library_id inexistente")
	}
}
