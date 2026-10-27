package library

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/catlinux/cinecat/internal/metadata"
)

func TestPersonaSeparaBibliotecaYResto(t *testing.T) {
	tmdbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/person/123":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 123, "name": "Persona", "biography": "Biografía",
				"birthday": "1970-01-02", "place_of_birth": "Madrid",
				"profile_path": "/persona.jpg", "known_for_department": "Acting",
				"combined_credits": map[string]any{
					"cast": []map[string]any{
						{"id": 603, "media_type": "movie", "title": "Matrix", "release_date": "1999-03-31", "character": "Neo"},
						{"id": 1399, "media_type": "tv", "name": "Juego de tronos", "first_air_date": "2011-04-17", "character": "Tyrion"},
						{"id": 604, "media_type": "movie", "title": "Otra", "release_date": "2001-01-01", "character": "Extra"},
					},
					"crew": []map[string]any{
						{"id": 604, "media_type": "movie", "title": "Otra", "release_date": "2001-01-01", "job": "Director"},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(tmdbServer.Close)
	tmdb, err := metadata.New(metadata.Options{
		APIKey:       "clave",
		BaseURL:      tmdbServer.URL,
		ImageBaseURL: tmdbServer.URL + "/img",
		CacheDir:     t.TempDir(),
		HTTPClient:   tmdbServer.Client(),
	})
	if err != nil {
		t.Fatalf("metadata.New: %v", err)
	}
	ts := newLibraryHTTPTestServer(t, tmdb)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	clipLibrary, err := ts.db.Exec(`INSERT INTO libraries (path, name, kind) VALUES (?, 'Clips', 'clips')`, t.TempDir())
	if err != nil {
		t.Fatalf("crear biblioteca de clips: %v", err)
	}
	clipLibraryID, _ := clipLibrary.LastInsertId()

	movieByTMDB := insertPersonMovie(t, ts, libraryID, 603,
		`{"cast":[{"name":"Persona","character":"Neo","photo":""}],"crew":[]}`)
	movieByCredit := insertPersonMovie(t, ts, libraryID, 605,
		`{"cast":[{"id":123,"name":"Persona","character":"Protagonista","photo":""}],"crew":[]}`)
	insertPersonMovie(t, ts, libraryID, 700,
		`{"cast":[{"id":999,"name":"Otra","character":"Nadie","photo":""}],"crew":[]}`)
	insertPersonMovie(t, ts, clipLibraryID, 603,
		`{"cast":[{"id":123,"name":"Persona","character":"Clip","photo":""}],"crew":[]}`)
	seriesResult, err := ts.db.Exec(`
		INSERT INTO series (library_id, folder, title, year, tmdb_id, credits)
		VALUES (?, 'Juego de tronos', 'Juego de tronos', 2011, 1399,
		        '{"cast":[{"name":"Persona","character":"Tyrion","photo":""}],"crew":[]}')`,
		libraryID)
	if err != nil {
		t.Fatalf("crear serie: %v", err)
	}
	seriesID, _ := seriesResult.LastInsertId()

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/people/123", ts.token)
	if status != http.StatusOK {
		t.Fatalf("persona: código = %d, cuerpo = %s", status, body)
	}
	var response personMediaResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("respuesta de persona: %v", err)
	}
	if response.Person.ID != 123 || response.Person.Name != "Persona" ||
		response.Person.Biography != "Biografía" || response.Person.PhotoURL == "" {
		t.Fatalf("persona inesperada: %+v", response.Person)
	}
	if len(response.InLibrary) != 3 {
		t.Fatalf("en biblioteca = %d, se esperaban 3: %+v", len(response.InLibrary), response.InLibrary)
	}
	byID := make(map[string]personMediaDTO)
	for _, item := range response.InLibrary {
		byID[item.Type+":"+strconv.FormatInt(item.ID, 10)] = item
	}
	if movie := byID["movie:"+strconv.FormatInt(movieByTMDB, 10)]; movie.Type != "movie" || movie.Title != "Peli" ||
		len(movie.Roles) != 1 || movie.Roles[0] != "Neo" {
		t.Fatalf("respaldo por tmdb_id inesperado: %+v", movie)
	}
	if movie := byID["movie:"+strconv.FormatInt(movieByCredit, 10)]; movie.Type != "movie" || len(movie.Roles) != 1 ||
		movie.Roles[0] != "Protagonista" {
		t.Fatalf("crédito local inesperado: %+v", movie)
	}
	if series := byID["series:"+strconv.FormatInt(seriesID, 10)]; series.Type != "series" || series.Title != "Juego de tronos" ||
		len(series.Roles) != 1 || series.Roles[0] != "Tyrion" {
		t.Fatalf("serie inesperada: %+v", series)
	}
	if len(response.NotInLibrary) != 1 {
		t.Fatalf("fuera de biblioteca = %d, se esperaba 1: %+v", len(response.NotInLibrary), response.NotInLibrary)
	}
	outside := response.NotInLibrary[0]
	if outside.Type != "movie" || outside.TMDBID != 604 || outside.Title != "Otra" ||
		len(outside.Roles) != 2 || outside.Roles[0] != "Extra" || outside.Roles[1] != "Director" {
		t.Fatalf("crédito fuera inesperado: %+v", outside)
	}
}

func insertPersonMovie(t *testing.T, ts *libraryTestServer, libraryID, tmdbID int64, credits string) int64 {
	t.Helper()
	result, err := ts.db.Exec(`
		INSERT INTO media_items
			(library_id, path, title, year, kind, tmdb_id, credits, rating, poster_url)
		VALUES (?, ?, ?, 2000, 'movie', ?, ?, 7.8, ?)`,
		libraryID, t.TempDir(), "Peli", tmdbID, credits, "/poster.jpg")
	if err != nil {
		t.Fatalf("crear película de persona: %v", err)
	}
	id, _ := result.LastInsertId()
	return id
}

func TestPersonaSinSesionSinTMDBYNoEncontrada(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	if status, _ := libraryHTTPGet(t, ts.URL+"/api/v1/people/123", ""); status != http.StatusUnauthorized {
		t.Fatalf("sin sesión: código = %d", status)
	}
	if status, _ := libraryHTTPGet(t, ts.URL+"/api/v1/people/123", ts.token); status != http.StatusNotFound {
		t.Fatalf("sin TMDB: código = %d", status)
	}

	tmdbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	t.Cleanup(tmdbServer.Close)
	tmdb, err := metadata.New(metadata.Options{
		APIKey: "clave", BaseURL: tmdbServer.URL, ImageBaseURL: tmdbServer.URL + "/img",
		CacheDir: t.TempDir(), HTTPClient: tmdbServer.Client(),
	})
	if err != nil {
		t.Fatalf("metadata.New: %v", err)
	}
	withTMDB := newLibraryHTTPTestServer(t, tmdb)
	if status, _ := libraryHTTPGet(t, withTMDB.URL+"/api/v1/people/999", withTMDB.token); status != http.StatusNotFound {
		t.Fatalf("persona inexistente: código = %d", status)
	}
}
