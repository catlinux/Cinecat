package store

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"strings"
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
		"playback_progress", "collections", "collection_items", "series", "series_seasons",
		"settings",
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
	expected, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		t.Fatalf("listar migraciones embebidas: %v", err)
	}
	if applied != len(expected) {
		t.Fatalf("migraciones registradas = %d, se esperaban %d", applied, len(expected))
	}
}

func TestMigracionBloqueosImagen(t *testing.T) {
	s, ctx := openTestStore(t, filepath.Join(t.TempDir(), "cinecat.db"))

	rows, err := s.DB().QueryContext(ctx,
		`SELECT name, "notnull", dflt_value FROM pragma_table_info('media_items')
		  WHERE name IN ('poster_locked', 'backdrop_locked') ORDER BY name`)
	if err != nil {
		t.Fatalf("consultar columnas de bloqueo: %v", err)
	}
	defer rows.Close()

	found := 0
	for rows.Next() {
		var name, defaultValue string
		var notNull int
		if err := rows.Scan(&name, &notNull, &defaultValue); err != nil {
			t.Fatalf("leer columna de bloqueo: %v", err)
		}
		found++
		if notNull != 1 || defaultValue != "0" {
			t.Errorf("%s: notnull = %d, default = %q; se esperaba 1 y 0", name, notNull, defaultValue)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("recorrer columnas de bloqueo: %v", err)
	}
	if found != 2 {
		t.Fatalf("columnas de bloqueo = %d, se esperaban 2", found)
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

func TestMigracionSettings(t *testing.T) {
	s, ctx := openTestStore(t, filepath.Join(t.TempDir(), "cinecat.db"))

	columns := columns(t, ctx, s, "settings")
	for _, name := range []string{"key", "value", "updated_at"} {
		if !columns[name] {
			t.Errorf("settings: falta la columna %q", name)
		}
	}

	var script string
	if err := s.DB().QueryRowContext(ctx,
		`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'settings'`).Scan(&script); err != nil {
		t.Fatalf("leer definición de settings: %v", err)
	}
	if !strings.Contains(script, "STRICT") {
		t.Fatalf("settings debe ser STRICT: %s", script)
	}
}

func TestMigracionSeries(t *testing.T) {
	s, ctx := openTestStore(t, filepath.Join(t.TempDir(), "cinecat.db"))

	mediaColumns := columns(t, ctx, s, "media_items")
	for _, name := range []string{"series_id", "episode_title", "still_url", "credits"} {
		if !mediaColumns[name] {
			t.Errorf("media_items: falta la columna %q", name)
		}
	}

	seriesColumns := columns(t, ctx, s, "series")
	for _, name := range []string{
		"library_id", "folder", "title", "display_title", "tmdb_id",
		"match_status", "sort_title", "sort_letter", "poster_locked", "backdrop_locked",
		"credits",
	} {
		if !seriesColumns[name] {
			t.Errorf("series: falta la columna %q", name)
		}
	}

	seasonColumns := columns(t, ctx, s, "series_seasons")
	for _, name := range []string{"series_id", "season", "name", "overview", "poster_url"} {
		if !seasonColumns[name] {
			t.Errorf("series_seasons: falta la columna %q", name)
		}
	}
}

func TestBorradoDeSerieEnCascadaYDesasociaEpisodios(t *testing.T) {
	s, ctx := openTestStore(t, filepath.Join(t.TempDir(), "cinecat.db"))

	library, err := s.DB().ExecContext(ctx,
		`INSERT INTO libraries (path, name) VALUES ('/series', 'Series')`)
	if err != nil {
		t.Fatalf("crear biblioteca: %v", err)
	}
	libraryID, err := library.LastInsertId()
	if err != nil {
		t.Fatalf("id de biblioteca: %v", err)
	}

	created, err := s.DB().ExecContext(ctx,
		`INSERT INTO series (library_id, folder, title) VALUES (?, 'Serie', 'Serie')`,
		libraryID)
	if err != nil {
		t.Fatalf("crear serie: %v", err)
	}
	seriesID, err := created.LastInsertId()
	if err != nil {
		t.Fatalf("id de serie: %v", err)
	}

	if _, err := s.DB().ExecContext(ctx,
		`INSERT INTO series_seasons (series_id, season, name) VALUES (?, 1, 'Temporada 1')`,
		seriesID); err != nil {
		t.Fatalf("crear temporada: %v", err)
	}
	if _, err := s.DB().ExecContext(ctx, `
		INSERT INTO media_items (library_id, path, title, kind, series_id)
		VALUES (?, '/series/Serie/Temporada 1/1x01.mkv', 'Serie', 'episode', ?)`,
		libraryID, seriesID); err != nil {
		t.Fatalf("crear episodio: %v", err)
	}

	if _, err := s.DB().ExecContext(ctx, `DELETE FROM series WHERE id = ?`, seriesID); err != nil {
		t.Fatalf("borrar serie: %v", err)
	}

	var seasons int
	if err := s.DB().QueryRowContext(ctx,
		`SELECT count(*) FROM series_seasons WHERE series_id = ?`, seriesID).Scan(&seasons); err != nil {
		t.Fatalf("contar temporadas: %v", err)
	}
	if seasons != 0 {
		t.Fatalf("quedaron %d temporadas tras borrar la serie", seasons)
	}

	var linked sql.NullInt64
	if err := s.DB().QueryRowContext(ctx,
		`SELECT series_id FROM media_items WHERE path = '/series/Serie/Temporada 1/1x01.mkv'`).
		Scan(&linked); err != nil {
		t.Fatalf("leer episodio: %v", err)
	}
	if linked.Valid {
		t.Fatalf("el episodio sigue asociado a la serie %d", linked.Int64)
	}
}

func TestClavesForaneasActivas(t *testing.T) {
	s, ctx := openTestStore(t, filepath.Join(t.TempDir(), "cinecat.db"))

	_, err := s.DB().ExecContext(ctx,
		`INSERT INTO media_items (library_id, path, title, kind) VALUES (999, '/x.mkv', 'X', 'movie')`)
	if err == nil {
		t.Fatal("se esperaba un error de clave foránea con library_id inexistente")
	}
}
