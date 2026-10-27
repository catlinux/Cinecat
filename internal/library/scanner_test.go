package library

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catlinux/cinecat/internal/store"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "cinecat.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close: %v", err)
		}
	})
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return st.DB()
}

func createLibrary(t *testing.T, db *sql.DB, path string) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO libraries (path, name) VALUES (?, ?)`, path, "Pruebas")
	if err != nil {
		t.Fatalf("crear biblioteca: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("id de biblioteca: %v", err)
	}
	return id
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("crear carpeta: %v", err)
	}
	if err := os.WriteFile(path, []byte("contenido de prueba"), 0o644); err != nil {
		t.Fatalf("crear archivo: %v", err)
	}
}

type mediaRow struct {
	title   string
	year    sql.NullInt64
	kind    string
	season  sql.NullInt64
	episode sql.NullInt64
}

func readMediaRow(t *testing.T, db *sql.DB, path string) mediaRow {
	t.Helper()
	var row mediaRow
	if err := db.QueryRow(
		`SELECT title, year, kind, season, episode FROM media_items WHERE path = ?`, path,
	).Scan(&row.title, &row.year, &row.kind, &row.season, &row.episode); err != nil {
		t.Fatalf("leer media_items %q: %v", path, err)
	}
	return row
}

func addScannerLibraryPath(t *testing.T, db *sql.DB, libraryID int64, path string) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO library_paths (library_id, path) VALUES (?, ?)`, libraryID, path); err != nil {
		t.Fatalf("añadir carpeta %q: %v", path, err)
	}
}

func TestScannerVariasCarpetasYArchivoRetirado(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	rootA := t.TempDir()
	rootB := t.TempDir()
	fileA := filepath.Join(rootA, "A (2001).mkv")
	fileA2 := filepath.Join(rootA, "A2 (2003).mkv")
	fileB := filepath.Join(rootB, "B (2002).mkv")
	writeFile(t, fileA)
	writeFile(t, fileA2)
	writeFile(t, fileB)

	libraryID := createLibrary(t, db, rootA)
	addScannerLibraryPath(t, db, libraryID, rootA)
	addScannerLibraryPath(t, db, libraryID, rootB)

	scanner := New(db)
	res, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Files != 3 || res.Upserted != 3 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if err := os.Remove(fileA); err != nil {
		t.Fatalf("retirar archivo: %v", err)
	}
	res, err = scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan tras retirar archivo: %v", err)
	}
	if res.Files != 2 {
		t.Fatalf("archivos tras poda = %d, se esperaba 2", res.Files)
	}
	if count := countRows(t, &libraryTestServer{db: db}, `SELECT count(*) FROM media_items WHERE path = ?`, fileA); count != 0 {
		t.Fatalf("el archivo ausente debe podarse: %d", count)
	}
	if count := countRows(t, &libraryTestServer{db: db}, `SELECT count(*) FROM media_items WHERE path = ?`, fileB); count != 1 {
		t.Fatalf("el archivo de la otra carpeta debe conservarse: %d", count)
	}
}

func TestScannerCarpetaNoDisponibleConservaFichasYProgreso(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	rootA := t.TempDir()
	rootB := t.TempDir()
	fileA := filepath.Join(rootA, "A (2001).mkv")
	fileB := filepath.Join(rootB, "B (2002).mkv")
	writeFile(t, fileA)
	writeFile(t, fileB)

	libraryID := createLibrary(t, db, rootA)
	addScannerLibraryPath(t, db, libraryID, rootA)
	addScannerLibraryPath(t, db, libraryID, rootB)
	scanner := New(db)
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("primer Scan: %v", err)
	}

	var itemID int64
	if err := db.QueryRow(`SELECT id FROM media_items WHERE path = ?`, fileB).Scan(&itemID); err != nil {
		t.Fatalf("leer ítem B: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`,
		"ana", "x", "user"); err != nil {
		t.Fatalf("crear usuario: %v", err)
	}
	anaID := progressUserID(t, db, "ana")
	setWatchState(t, db, anaID, itemID)
	setProgressRow(t, db, anaID, itemID, 30, 100, "2026-10-26T10:00:00Z")
	if err := os.RemoveAll(rootB); err != nil {
		t.Fatalf("retirar carpeta B: %v", err)
	}

	res, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan sin carpeta B: %v", err)
	}
	foundWarning := false
	for _, warning := range res.Warnings {
		if strings.Contains(warning, rootB) && strings.Contains(warning, "se conservan sus fichas") {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Fatalf("falta el aviso de carpeta no disponible: %v", res.Warnings)
	}
	server := &libraryTestServer{db: db}
	if count := countRows(t, server, `SELECT count(*) FROM media_items WHERE id = ?`, itemID); count != 1 {
		t.Fatalf("el ítem de la carpeta ausente debe conservarse: %d", count)
	}
	if count := countRows(t, server, `SELECT count(*) FROM watch_state WHERE media_item_id = ?`, itemID); count != 1 {
		t.Fatalf("el visto debe conservarse: %d", count)
	}
	if count := countRows(t, server, `SELECT count(*) FROM playback_progress WHERE media_item_id = ?`, itemID); count != 1 {
		t.Fatalf("el progreso debe conservarse: %d", count)
	}

	writeFile(t, fileB)
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("Scan con carpeta B restaurada: %v", err)
	}
	if count := countRows(t, server, `SELECT count(*) FROM media_items WHERE id = ?`, itemID); count != 1 {
		t.Fatalf("el ítem debe seguir al volver la carpeta: %d", count)
	}
}

func TestScannerSaltaMusica(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "music.mkv"))
	result, err := db.Exec(
		`INSERT INTO libraries (path, name, kind) VALUES (?, ?, 'music')`, root, "music")
	if err != nil {
		t.Fatalf("crear biblioteca music: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de biblioteca music: %v", err)
	}
	addScannerLibraryPath(t, db, id, root)

	res, err := New(db).Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Files != 0 {
		t.Fatalf("música no debe escanearse todavía: %+v", res)
	}
	found := false
	for _, warning := range res.Warnings {
		if strings.Contains(warning, "tipo music pendiente de implementar") {
			found = true
		}
	}
	if !found {
		t.Fatalf("falta el aviso para music: %v", res.Warnings)
	}
}

func TestScannerPueblaMediaItemsSinDuplicar(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	root := t.TempDir()
	files := []string{
		"Pelicula Española (2019).mkv",
		"Another.Movie.2010.1080p.WEB-DL.x264-GRUPO.mp4",
		"Series/Season 1/Series - 1x01 - Piloto.mkv",
		"Series/Season 1/04 - Cuarto.avi",
		"Mayusculas.MKV",
		"notas.txt",
		"portada.jpg",
		".oculto.mkv",
	}
	for _, name := range files {
		writeFile(t, filepath.Join(root, name))
	}
	createLibrary(t, db, root)

	scanner := New(db)
	res, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Libraries != 1 || res.Files != 5 || res.Upserted != 5 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("avisos inesperados: %v", res.Warnings)
	}

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM media_items`).Scan(&count); err != nil {
		t.Fatalf("contar entradas: %v", err)
	}
	if count != 5 {
		t.Fatalf("media_items tiene %d filas, se esperaban 5", count)
	}

	movie := readMediaRow(t, db, filepath.Join(root, "Pelicula Española (2019).mkv"))
	if movie.title != "Pelicula Española" || movie.kind != KindMovie ||
		!movie.year.Valid || movie.year.Int64 != 2019 {
		t.Fatalf("película inesperada: %+v", movie)
	}
	if movie.season.Valid || movie.episode.Valid {
		t.Fatalf("una película no debe tener temporada ni episodio: %+v", movie)
	}

	episode := readMediaRow(t, db, filepath.Join(root, "Series", "Season 1", "Series - 1x01 - Piloto.mkv"))
	if episode.title != "Series" || episode.kind != KindEpisode ||
		!episode.season.Valid || episode.season.Int64 != 1 ||
		!episode.episode.Valid || episode.episode.Int64 != 1 {
		t.Fatalf("episodio inesperado: %+v", episode)
	}

	leading := readMediaRow(t, db, filepath.Join(root, "Series", "Season 1", "04 - Cuarto.avi"))
	if leading.title != "Series" || leading.kind != KindEpisode ||
		!leading.season.Valid || leading.season.Int64 != 1 ||
		!leading.episode.Valid || leading.episode.Int64 != 4 {
		t.Fatalf("episodio numerado inesperado: %+v", leading)
	}

	upper := readMediaRow(t, db, filepath.Join(root, "Mayusculas.MKV"))
	if upper.title != "Mayusculas" || upper.kind != KindMovie {
		t.Fatalf("archivo con extensión en mayúsculas inesperado: %+v", upper)
	}

	// Un segundo escaneo actualiza pero no duplica.
	if _, err := db.Exec(`UPDATE media_items SET tmdb_id = 42 WHERE path = ?`,
		filepath.Join(root, "Pelicula Española (2019).mkv")); err != nil {
		t.Fatalf("marcar tmdb_id: %v", err)
	}

	res, err = scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("segundo Scan: %v", err)
	}
	if res.Files != 5 || res.Upserted != 5 {
		t.Fatalf("segundo escaneo inesperado: %+v", res)
	}
	if err := db.QueryRow(`SELECT count(*) FROM media_items`).Scan(&count); err != nil {
		t.Fatalf("contar entradas tras el segundo escaneo: %v", err)
	}
	if count != 5 {
		t.Fatalf("tras el segundo escaneo hay %d filas, se esperaban 5", count)
	}

	var tmdbID sql.NullInt64
	if err := db.QueryRow(`SELECT tmdb_id FROM media_items WHERE path = ?`,
		filepath.Join(root, "Pelicula Española (2019).mkv")).Scan(&tmdbID); err != nil {
		t.Fatalf("leer tmdb_id: %v", err)
	}
	if !tmdbID.Valid || tmdbID.Int64 != 42 {
		t.Fatalf("el segundo escaneo no debe borrar tmdb_id: %+v", tmdbID)
	}
}

func TestScannerBibliotecaInexistente(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	createLibrary(t, db, filepath.Join(t.TempDir(), "no-existe"))

	res, err := New(db).Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Libraries != 1 || res.Files != 0 || len(res.Warnings) != 1 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
}

func TestScannerCarpetaVaciaConFichasNoSePoda(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	rootA := t.TempDir()
	rootB := t.TempDir()
	fileA := filepath.Join(rootA, "A (2001).mkv")
	fileB := filepath.Join(rootB, "B (2002).mkv")
	writeFile(t, fileA)
	writeFile(t, fileB)

	libraryID := createLibrary(t, db, rootA)
	addScannerLibraryPath(t, db, libraryID, rootA)
	addScannerLibraryPath(t, db, libraryID, rootB)
	scanner := New(db)
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("primer Scan: %v", err)
	}

	// Un disco sin montar deja su punto de montaje como carpeta vacía.
	if err := os.Remove(fileB); err != nil {
		t.Fatalf("vaciar carpeta B: %v", err)
	}
	res, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan con carpeta B vacía: %v", err)
	}
	server := &libraryTestServer{db: db}
	if count := countRows(t, server, `SELECT count(*) FROM media_items WHERE path = ?`, fileB); count != 1 {
		t.Fatalf("la ficha de una carpeta vacía debe conservarse: %d", count)
	}
	found := false
	for _, warning := range res.Warnings {
		if strings.Contains(warning, rootB) && strings.Contains(warning, "está vacía pero tiene fichas") {
			found = true
		}
	}
	if !found {
		t.Fatalf("falta el aviso de carpeta vacía: %v", res.Warnings)
	}
	if count := countRows(t, server, `SELECT count(*) FROM media_items WHERE path = ?`, fileA); count != 1 {
		t.Fatalf("la otra carpeta no debe verse afectada: %d", count)
	}
}
