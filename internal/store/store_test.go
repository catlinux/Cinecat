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
		"schema_migrations", "users", "sessions", "libraries", "library_paths", "media_items",
		"playback_state", "playback_progress", "collections", "collection_items", "series",
		"series_seasons", "settings", "item_preferences", "music_artists", "music_albums",
		"music_tracks", "item_lists", "series_lists", "collection_lists", "trash_items",
	} {
		if !got[name] {
			t.Errorf("falta la tabla %q", name)
		}
	}
}

func TestMigracionLibraryPathsConDatosPrevios(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "cinecat.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	initial, err := migrations.ReadFile("migrations/0001_init.sql")
	if err != nil {
		t.Fatalf("leer migración inicial: %v", err)
	}
	if _, err := s.DB().ExecContext(ctx, string(initial)); err != nil {
		t.Fatalf("aplicar esquema inicial: %v", err)
	}
	if _, err := s.DB().ExecContext(ctx,
		`INSERT INTO libraries (path, name) VALUES ('/media/vieja', 'Vieja')`); err != nil {
		t.Fatalf("crear biblioteca previa: %v", err)
	}
	migration, err := migrations.ReadFile("migrations/0015_library_paths.sql")
	if err != nil {
		t.Fatalf("leer migración 0015: %v", err)
	}
	if _, err := s.DB().ExecContext(ctx, string(migration)); err != nil {
		t.Fatalf("aplicar migración 0015: %v", err)
	}

	var id int64
	var kind, path string
	if err := s.DB().QueryRowContext(ctx,
		`SELECT id, kind, path FROM libraries WHERE name = 'Vieja'`).
		Scan(&id, &kind, &path); err != nil {
		t.Fatalf("leer biblioteca migrada: %v", err)
	}
	if kind != "video" || path != "/media/vieja" {
		t.Fatalf("biblioteca migrada inesperada: kind=%q path=%q", kind, path)
	}
	var copied string
	if err := s.DB().QueryRowContext(ctx,
		`SELECT path FROM library_paths WHERE library_id = ?`, id).Scan(&copied); err != nil {
		t.Fatalf("leer carpeta migrada: %v", err)
	}
	if copied != path {
		t.Fatalf("carpeta migrada = %q, se esperaba %q", copied, path)
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

func TestMigracionPersonasYTrailers(t *testing.T) {
	s, ctx := openTestStore(t, filepath.Join(t.TempDir(), "cinecat.db"))

	for table, expected := range map[string]string{
		"media_items": "trailer_url",
		"series":      "trailer_url",
	} {
		columns := columns(t, ctx, s, table)
		if !columns[expected] || !columns["trailer_language"] {
			t.Fatalf("%s: faltan las columnas de tráiler: %v", table, columns)
		}
	}
}

func TestMigracionPapelera(t *testing.T) {
	s, ctx := openTestStore(t, filepath.Join(t.TempDir(), "cinecat.db"))

	got := columns(t, ctx, s, "trash_items")
	for _, name := range []string{
		"id", "library_id", "original_path", "trash_path", "moved_kind", "extra_paths",
		"title", "year", "kind", "snapshot", "deleted_by", "deleted_at",
	} {
		if !got[name] {
			t.Errorf("trash_items: falta la columna %q", name)
		}
	}
	result, err := s.DB().ExecContext(ctx,
		`INSERT INTO libraries (path, name) VALUES ('/media/papelera', 'Papelera')`)
	if err != nil {
		t.Fatalf("crear biblioteca: %v", err)
	}
	libraryID, _ := result.LastInsertId()
	if _, err := s.DB().ExecContext(ctx, `
		INSERT INTO trash_items
			(library_id, original_path, trash_path, moved_kind, title, kind, snapshot, deleted_at)
		VALUES (?, '/media/papelera/peli.mkv', '/media/papelera/.cinecat-papelera/peli.mkv',
		        'file', 'Peli', 'movie', '{}', '2026-10-27T10:00:00Z')`, libraryID); err != nil {
		t.Fatalf("crear elemento de papelera: %v", err)
	}
	if _, err := s.DB().ExecContext(ctx, `DELETE FROM libraries WHERE id = ?`, libraryID); err != nil {
		t.Fatalf("borrar biblioteca: %v", err)
	}
	var count int
	if err := s.DB().QueryRowContext(ctx, `SELECT count(*) FROM trash_items`).Scan(&count); err != nil {
		t.Fatalf("contar papelera: %v", err)
	}
	if count != 0 {
		t.Fatalf("la papelera debería caer en cascada, quedan %d filas", count)
	}
}

func TestMigracionPreferencias(t *testing.T) {
	s, ctx := openTestStore(t, filepath.Join(t.TempDir(), "cinecat.db"))

	columns := columns(t, ctx, s, "item_preferences")
	for _, name := range []string{
		"user_id", "media_item_id", "audio_index", "subtitle_id", "quality", "updated_at",
	} {
		if !columns[name] {
			t.Errorf("item_preferences: falta la columna %q", name)
		}
	}
	var script string
	if err := s.DB().QueryRowContext(ctx,
		`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'item_preferences'`).
		Scan(&script); err != nil {
		t.Fatalf("leer definición de item_preferences: %v", err)
	}
	if !strings.Contains(script, "STRICT") {
		t.Fatalf("item_preferences debe ser STRICT: %s", script)
	}
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

func TestMigracionRefillTMDBCredits(t *testing.T) {
	s, ctx := openTestStore(t, filepath.Join(t.TempDir(), "cinecat.db"))

	library, err := s.DB().ExecContext(ctx,
		`INSERT INTO libraries (path, name) VALUES ('/refill', 'Refill')`)
	if err != nil {
		t.Fatalf("crear biblioteca: %v", err)
	}
	libraryID, err := library.LastInsertId()
	if err != nil {
		t.Fatalf("id de biblioteca: %v", err)
	}

	if _, err := s.DB().ExecContext(ctx, `
		INSERT INTO media_items (
			library_id, path, title, kind, metadata_source, credits, collection_checked
		) VALUES
			(?, '/tmdb.mkv', 'TMDB', 'movie', 'tmdb', '{"cast":[]}', 1),
			(?, '/omdb.mkv', 'OMDB', 'movie', 'omdb', '{"cast":[]}', 1)`,
		libraryID, libraryID); err != nil {
		t.Fatalf("insertar películas: %v", err)
	}
	if _, err := s.DB().ExecContext(ctx, `
		INSERT INTO series (library_id, folder, title, metadata_source, match_status, credits)
		VALUES
			(?, 'TMDB', 'TMDB', 'tmdb', 'auto', '{"cast":[]}'),
			(?, 'Manual', 'Manual', NULL, 'auto', '{"cast":[]}')`,
		libraryID, libraryID); err != nil {
		t.Fatalf("insertar series: %v", err)
	}

	script, err := migrations.ReadFile("migrations/0013_refill_tmdb_metadata.sql")
	if err != nil {
		t.Fatalf("leer migración 0013: %v", err)
	}
	if _, err := s.DB().ExecContext(ctx, string(script)); err != nil {
		t.Fatalf("aplicar migración 0013: %v", err)
	}

	rows, err := s.DB().QueryContext(ctx, `
		SELECT path, metadata_source, credits, collection_checked
		  FROM media_items ORDER BY path`)
	if err != nil {
		t.Fatalf("leer películas: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var path, source string
		var credits sql.NullString
		var checked int
		if err := rows.Scan(&path, &source, &credits, &checked); err != nil {
			t.Fatalf("leer película: %v", err)
		}
		if source == "tmdb" {
			if credits.Valid || checked != 0 {
				t.Fatalf("%s: se esperaba credits NULL y collection_checked 0", path)
			}
		} else if !credits.Valid || checked != 1 {
			t.Fatalf("%s: la migración tocó una fila que no era de TMDB", path)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("recorrer películas: %v", err)
	}

	seriesRows, err := s.DB().QueryContext(ctx, `
		SELECT folder, metadata_source, credits FROM series ORDER BY folder`)
	if err != nil {
		t.Fatalf("leer series: %v", err)
	}
	defer seriesRows.Close()
	for seriesRows.Next() {
		var folder string
		var source sql.NullString
		var credits sql.NullString
		if err := seriesRows.Scan(&folder, &source, &credits); err != nil {
			t.Fatalf("leer serie: %v", err)
		}
		if source.Valid && source.String == "tmdb" {
			if credits.Valid {
				t.Fatalf("%s: la serie de TMDB debe quedar sin credits", folder)
			}
		} else if !credits.Valid {
			t.Fatalf("%s: la migración tocó una serie que no era de TMDB", folder)
		}
	}
	if err := seriesRows.Err(); err != nil {
		t.Fatalf("recorrer series: %v", err)
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

func TestMigracionMusicaRestriccionesYCascada(t *testing.T) {
	s, ctx := openTestStore(t, filepath.Join(t.TempDir(), "cinecat.db"))
	db := s.DB()
	res, err := db.ExecContext(ctx,
		`INSERT INTO libraries (path, name, kind) VALUES ('/musica', 'Música', 'music')`)
	if err != nil {
		t.Fatalf("crear biblioteca: %v", err)
	}
	libraryID, _ := res.LastInsertId()
	res, err = db.ExecContext(ctx,
		`INSERT INTO music_artists (name, sort_name) VALUES ('The Beatles', 'Beatles')`)
	if err != nil {
		t.Fatalf("crear artista: %v", err)
	}
	artistID, _ := res.LastInsertId()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO music_artists (name, sort_name) VALUES ('the beatles', 'beatles')`); err == nil {
		t.Fatal("el nombre de artista debe ser único sin distinguir mayúsculas")
	}
	res, err = db.ExecContext(ctx, `
		INSERT INTO music_albums (library_id, artist_id, title, year)
		VALUES (?, ?, 'Abbey Road', 1969)`, libraryID, artistID)
	if err != nil {
		t.Fatalf("crear álbum: %v", err)
	}
	albumID, _ := res.LastInsertId()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO music_albums (library_id, artist_id, title)
		VALUES (?, ?, 'abbey road')`, libraryID, artistID); err == nil {
		t.Fatal("el título del álbum debe ser único sin distinguir mayúsculas")
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO music_tracks (
			library_id, album_id, artist_id, path, title, size_bytes, mtime_unix
		) VALUES (?, ?, ?, '/musica/01.mp3', 'Come Together', 10, 20)`,
		libraryID, albumID, artistID); err != nil {
		t.Fatalf("crear pista: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`DELETE FROM libraries WHERE id = ?`, libraryID); err != nil {
		t.Fatalf("borrar biblioteca: %v", err)
	}
	for _, table := range []string{"music_tracks", "music_albums"} {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM `+table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s tras cascada = %d, %v", table, count, err)
		}
	}
}

func TestMigracionUserListsCascada(t *testing.T) {
	s, ctx := openTestStore(t, filepath.Join(t.TempDir(), "cinecat.db"))
	db := s.DB()
	res, err := db.ExecContext(ctx, `INSERT INTO users (username, password_hash) VALUES ('ana', 'hash')`)
	if err != nil {
		t.Fatalf("crear usuario: %v", err)
	}
	userID, _ := res.LastInsertId()
	res, err = db.ExecContext(ctx, `INSERT INTO libraries (path, name) VALUES ('/media/lista', 'Lista')`)
	if err != nil {
		t.Fatalf("crear biblioteca: %v", err)
	}
	libraryID, _ := res.LastInsertId()
	res, err = db.ExecContext(ctx, `
		INSERT INTO media_items (library_id, path, title, kind)
		VALUES (?, '/media/lista/pelicula.mkv', 'Película', 'movie')`, libraryID)
	if err != nil {
		t.Fatalf("crear ítem: %v", err)
	}
	itemID, _ := res.LastInsertId()
	res, err = db.ExecContext(ctx, `
		INSERT INTO series (library_id, folder, title) VALUES (?, 'Serie', 'Serie')`, libraryID)
	if err != nil {
		t.Fatalf("crear serie: %v", err)
	}
	seriesID, _ := res.LastInsertId()
	res, err = db.ExecContext(ctx, `INSERT INTO collections (name, source) VALUES ('Colección', 'manual')`)
	if err != nil {
		t.Fatalf("crear colección: %v", err)
	}
	collectionID, _ := res.LastInsertId()
	for _, statement := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO item_lists (user_id, media_item_id, list) VALUES (?, ?, 'favorite')`, []any{userID, itemID}},
		{`INSERT INTO series_lists (user_id, series_id, list) VALUES (?, ?, 'favorite')`, []any{userID, seriesID}},
		{`INSERT INTO collection_lists (user_id, collection_id, list) VALUES (?, ?, 'favorite')`, []any{userID, collectionID}},
	} {
		if _, err := db.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Fatalf("crear marca: %v", err)
		}
	}

	if _, err := db.ExecContext(ctx, `DELETE FROM media_items WHERE id = ?`, itemID); err != nil {
		t.Fatalf("borrar ítem: %v", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM item_lists`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("marcas de ítem tras cascada = %d, %v", count, err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM series WHERE id = ?`, seriesID); err != nil {
		t.Fatalf("borrar serie: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM series_lists`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("marcas de serie tras cascada = %d, %v", count, err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM collections WHERE id = ?`, collectionID); err != nil {
		t.Fatalf("borrar colección: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM collection_lists`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("marcas de colección tras cascada = %d, %v", count, err)
	}

	res, err = db.ExecContext(ctx, `
		INSERT INTO media_items (library_id, path, title, kind)
		VALUES (?, '/media/lista/otra.mkv', 'Otra', 'movie')`, libraryID)
	if err != nil {
		t.Fatalf("crear segundo ítem: %v", err)
	}
	secondItemID, _ := res.LastInsertId()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO item_lists (user_id, media_item_id, list) VALUES (?, ?, 'watchlist')`,
		userID, secondItemID); err != nil {
		t.Fatalf("crear segunda marca: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, userID); err != nil {
		t.Fatalf("borrar usuario: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM item_lists`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("marcas tras borrar usuario = %d, %v", count, err)
	}
}
