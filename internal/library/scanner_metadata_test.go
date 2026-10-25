package library

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/catlinux/cinecat/internal/metadata"
)

func newMetadataTestClient(t *testing.T, baseURL string) *metadata.OMDbClient {
	t.Helper()
	client, err := metadata.NewOMDbClient(metadata.OMDbOptions{
		APIKey:   "clave-de-prueba",
		BaseURL:  baseURL,
		CacheDir: filepath.Join(t.TempDir(), "cache"),
	})
	if err != nil {
		t.Fatalf("NewOMDbClient: %v", err)
	}
	return client
}

func readMetadata(t *testing.T, db *sql.DB, path string) (sql.NullString, sql.NullString) {
	t.Helper()
	var overview, poster sql.NullString
	if err := db.QueryRow(
		`SELECT overview, poster_url FROM media_items WHERE path = ?`, path,
	).Scan(&overview, &poster); err != nil {
		t.Fatalf("leer metadatos de %q: %v", path, err)
	}
	return overview, poster
}

func TestScannerConOMDbRellenaYNoRepite(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	root := t.TempDir()
	file := filepath.Join(root, "Pelicula (2019).mkv")
	writeFile(t, file)
	createLibrary(t, db, root)

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"Title": "Pelicula",
			"Year": "2019",
			"Plot": "Una sinopsis de prueba.",
			"Poster": "https://images.example/pelicula.jpg",
			"Response": "True"
		}`))
	}))
	t.Cleanup(server.Close)

	scanner := New(db).WithOMDb(newMetadataTestClient(t, server.URL))
	res, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Files != 1 || res.Upserted != 1 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("avisos inesperados: %v", res.Warnings)
	}

	overview, poster := readMetadata(t, db, file)
	if !overview.Valid || overview.String != "Una sinopsis de prueba." {
		t.Fatalf("overview = %+v", overview)
	}
	if !poster.Valid || poster.String != "https://images.example/pelicula.jpg" {
		t.Fatalf("poster_url = %+v", poster)
	}
	if calls.Load() != 1 {
		t.Fatalf("peticiones = %d, se esperaba 1", calls.Load())
	}

	// Un segundo escaneo no vuelve a consultar OMDb.
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("segundo Scan: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("el segundo escaneo hizo %d peticiones a OMDb", calls.Load())
	}
}

func TestScannerConOMDbNoEncontrado(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	root := t.TempDir()
	file := filepath.Join(root, "Desconocida (1999).mkv")
	writeFile(t, file)
	createLibrary(t, db, root)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":"False","Error":"Movie not found!"}`))
	}))
	t.Cleanup(server.Close)

	res, err := New(db).WithOMDb(newMetadataTestClient(t, server.URL)).Scan(ctx)
	if err != nil {
		t.Fatalf("el escaneo no debe fallar: %v", err)
	}
	if res.Files != 1 {
		t.Fatalf("archivos = %d", res.Files)
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("avisos = %v, se esperaba uno", res.Warnings)
	}

	overview, poster := readMetadata(t, db, file)
	if overview.Valid || poster.Valid {
		t.Fatalf("no debe haber metadatos: %+v, %+v", overview, poster)
	}
}

func TestScannerConOMDbIgnoraSeriesYEpisodios(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Serie", "Season 1", "Serie - 1x01.mkv"))
	createLibrary(t, db, root)

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":"True","Title":"Serie","Plot":"x"}`))
	}))
	t.Cleanup(server.Close)

	res, err := New(db).WithOMDb(newMetadataTestClient(t, server.URL)).Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Files != 1 {
		t.Fatalf("archivos = %d", res.Files)
	}
	if calls.Load() != 0 {
		t.Fatalf("no se debe consultar OMDb para episodios: %d peticiones", calls.Load())
	}
}
