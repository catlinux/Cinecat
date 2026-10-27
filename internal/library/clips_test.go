package library

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestClipTitle(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{name: "2019-08-15_Cumple.de.Ana.mp4", want: "2019-08-15 Cumple de Ana"},
		{name: "Clip__con...espacios.mkv", want: "Clip con espacios"},
		{name: "Clip Ñandú 🎬.MP4", want: "Clip Ñandú 🎬"},
		{name: "12345.mp4", want: "12345"},
		{name: "sin-extension", want: "sin-extension"},
	}
	for _, test := range cases {
		if got := clipTitle(test.name); got != test.want {
			t.Errorf("clipTitle(%q) = %q, se esperaba %q", test.name, got, test.want)
		}
	}
}

func createClipLibrary(t *testing.T, db *sql.DB, path string) int64 {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO libraries (path, name, kind) VALUES (?, ?, 'clips')`, path, "Vídeos")
	if err != nil {
		t.Fatalf("crear biblioteca de clips: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("id de biblioteca de clips: %v", err)
	}
	return id
}

func TestScannerClipsSinIdentificacion(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	rootA := t.TempDir()
	rootB := t.TempDir()
	fileTitle := filepath.Join(rootA, "2019-08-15_Cumple.de.Ana.mp4")
	fileRemove := filepath.Join(rootA, "Clip Ñandú 🎬.mkv")
	fileEmptyRoot := filepath.Join(rootB, "12345.mp4")
	writeFile(t, fileTitle)
	writeFile(t, fileRemove)
	writeFile(t, fileEmptyRoot)

	libraryID := createClipLibrary(t, db, rootA)
	addScannerLibraryPath(t, db, libraryID, rootA)
	addScannerLibraryPath(t, db, libraryID, rootB)

	var omdbCalls atomic.Int32
	omdbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		omdbCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":"True","Title":"No debería consultarse","Plot":"x"}`))
	}))
	t.Cleanup(omdbServer.Close)

	fake := &fakeTMDB{}
	scanner := New(db).WithTMDB(fake).WithOMDb(newMetadataTestClient(t, omdbServer.URL))
	res, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Files != 3 || res.Upserted != 3 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if res.Identified != 0 || res.Unidentified != 0 || res.SeriesIdentified != 0 ||
		res.SeriesUnidentified != 0 {
		t.Fatalf("los clips no se identifican: %+v", res)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("avisos inesperados: %v", res.Warnings)
	}
	if fake.searchCalls != 0 || fake.detailsCalls != 0 || fake.tvSearchCalls != 0 ||
		fake.tvDetailsCalls != 0 || omdbCalls.Load() != 0 {
		t.Fatalf("no debe haber peticiones externas: tmdb=%d/%d/%d/%d omdb=%d",
			fake.searchCalls, fake.detailsCalls, fake.tvSearchCalls, fake.tvDetailsCalls,
			omdbCalls.Load())
	}

	movie := readMediaRow(t, db, fileTitle)
	if movie.title != "2019-08-15 Cumple de Ana" || movie.kind != KindMovie ||
		movie.year.Valid || movie.season.Valid || movie.episode.Valid {
		t.Fatalf("clip inesperado: %+v", movie)
	}
	var source sql.NullString
	var status sql.NullString
	if err := db.QueryRow(
		`SELECT metadata_source, match_status FROM media_items WHERE path = ?`, fileTitle,
	).Scan(&source, &status); err != nil {
		t.Fatalf("leer metadatos del clip: %v", err)
	}
	if source.Valid || !status.Valid || status.String != "none" {
		t.Fatalf("metadatos del clip inesperados: source=%+v status=%+v", source, status)
	}
	if title := readMediaRow(t, db, fileEmptyRoot).title; title != "12345" {
		t.Fatalf("título numérico = %q", title)
	}

	// Una carpeta que se queda vacía conserva sus fichas (posible disco sin
	// montar), aunque el archivo ya no esté.
	if err := os.Remove(fileEmptyRoot); err != nil {
		t.Fatalf("vaciar la segunda carpeta: %v", err)
	}
	res, err = scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan con carpeta vacía: %v", err)
	}
	if count := countRows(t, &libraryTestServer{db: db},
		`SELECT count(*) FROM media_items WHERE path = ?`, fileEmptyRoot); count != 1 {
		t.Fatalf("el clip de la carpeta vacía debe conservarse: %d", count)
	}
	foundWarning := false
	for _, warning := range res.Warnings {
		if strings.Contains(warning, rootB) && strings.Contains(warning, "está vacía pero tiene fichas") {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Fatalf("falta el aviso de carpeta vacía: %v", res.Warnings)
	}

	// Un archivo retirado de una carpeta recorrida con éxito sí se poda.
	if err := os.Remove(fileRemove); err != nil {
		t.Fatalf("retirar archivo: %v", err)
	}
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("Scan tras retirar archivo: %v", err)
	}
	if count := countRows(t, &libraryTestServer{db: db},
		`SELECT count(*) FROM media_items WHERE path = ?`, fileRemove); count != 0 {
		t.Fatalf("el clip ausente debe podarse: %d", count)
	}
	if count := countRows(t, &libraryTestServer{db: db},
		`SELECT count(*) FROM media_items WHERE path = ?`, fileTitle); count != 1 {
		t.Fatalf("el otro clip debe conservarse: %d", count)
	}
	if fake.searchCalls != 0 || fake.detailsCalls != 0 || omdbCalls.Load() != 0 {
		t.Fatalf("los escaneos posteriores tampoco deben consultar metadatos")
	}
}
