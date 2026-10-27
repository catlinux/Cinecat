package library

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func guidedLibrary(t *testing.T, ts *libraryTestServer, name, kind string, roots ...string) int64 {
	t.Helper()
	if len(roots) == 0 {
		t.Fatal("la biblioteca necesita al menos una raíz")
	}
	result, err := ts.db.Exec(
		`INSERT INTO libraries (path, name, kind) VALUES (?, ?, ?)`, roots[0], name, kind)
	if err != nil {
		t.Fatalf("crear biblioteca %q: %v", name, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de la biblioteca %q: %v", name, err)
	}
	for _, root := range roots {
		if _, err := ts.db.Exec(
			`INSERT INTO library_paths (library_id, path) VALUES (?, ?)`, id, root); err != nil {
			t.Fatalf("crear raíz %q: %v", root, err)
		}
	}
	return id
}

func guidedMovie(t *testing.T, db *sql.DB, libraryID int64, path string) int64 {
	t.Helper()
	result, err := db.Exec(`
		INSERT INTO media_items (library_id, path, title, kind)
		VALUES (?, ?, ?, 'movie')`, libraryID, path, filepath.Base(path))
	if err != nil {
		t.Fatalf("crear película %q: %v", path, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de la película %q: %v", path, err)
	}
	return id
}

func guidedSeriesFixture(t *testing.T, db *sql.DB, libraryID int64, folder string) int64 {
	t.Helper()
	result, err := db.Exec(
		`INSERT INTO series (library_id, folder, title) VALUES (?, ?, ?)`, libraryID, folder, folder)
	if err != nil {
		t.Fatalf("crear serie %q: %v", folder, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de la serie %q: %v", folder, err)
	}
	return id
}

func guidedEpisodeFixture(t *testing.T, db *sql.DB, libraryID, seriesID int64, path string, season, episode int) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO media_items (library_id, path, title, kind, series_id, season, episode)
		VALUES (?, ?, ?, 'episode', ?, ?, ?)`,
		libraryID, path, filepath.Base(path), seriesID, season, episode); err != nil {
		t.Fatalf("crear episodio %q: %v", path, err)
	}
}

func guidedUploadURL(ts *libraryTestServer, seriesID int64, season int, episode int, name string) string {
	values := url.Values{
		"season":  {strconv.Itoa(season)},
		"episode": {strconv.Itoa(episode)},
		"name":    {name},
	}
	return fmt.Sprintf("%s/api/v1/series/%d/files?%s", ts.URL, seriesID, values.Encode())
}

func guidedUpload(t *testing.T, ts *libraryTestServer, seriesID int64, season, episode int, name, content, token string) (int, []byte) {
	t.Helper()
	return uploadRequest(t, guidedUploadURL(ts, seriesID, season, episode, name),
		token, bytes.NewReader([]byte(content)), int64(len(content)))
}

func TestUploadTargetsCuentaPeliculasYSeries(t *testing.T) {
	ts := newTrashTestServer(t)
	setAllowFileChanges(t, ts, true)
	base := t.TempDir()
	rootA := filepath.Join(base, "a")
	rootB := filepath.Join(base, "b")
	clipsRoot := filepath.Join(base, "clips")
	for _, root := range []string{rootA, rootB, clipsRoot} {
		if err := os.MkdirAll(root, 0o775); err != nil {
			t.Fatalf("crear raíz %s: %v", root, err)
		}
	}
	videoID := guidedLibrary(t, ts, "Beta", "video", rootA, rootB)
	clipsID := guidedLibrary(t, ts, "Alfa", "clips", clipsRoot)
	musicID := guidedLibrary(t, ts, "Música", "music", filepath.Join(base, "music"))

	guidedMovie(t, ts.db, videoID, filepath.Join(rootA, "Uno.mkv"))
	guidedMovie(t, ts.db, videoID, filepath.Join(rootA, "sub", "Dos.mkv"))
	guidedMovie(t, ts.db, videoID, filepath.Join(rootB, "Tres.mkv"))
	guidedMovie(t, ts.db, clipsID, filepath.Join(clipsRoot, "Clip.mkv"))
	guidedMovie(t, ts.db, musicID, filepath.Join(base, "music", "No cuenta.mkv"))

	serieA := guidedSeriesFixture(t, ts.db, videoID, "Serie A")
	guidedEpisodeFixture(t, ts.db, videoID, serieA,
		filepath.Join(rootA, "Serie A", "Temporada 1", "Serie A S01E01.mkv"), 1, 1)
	serieB := guidedSeriesFixture(t, ts.db, videoID, "Serie B")
	guidedEpisodeFixture(t, ts.db, videoID, serieB,
		filepath.Join(rootB, "Serie B", "Serie B S01E01.mkv"), 1, 1)
	serieSinEpisodios := guidedSeriesFixture(t, ts.db, videoID, "Sin episodios")
	_ = serieSinEpisodios

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/uploads/targets", ts.token)
	if status != http.StatusOK {
		t.Fatalf("destinos: código = %d, cuerpo = %s", status, body)
	}
	var response uploadTargetsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("destinos: %v", err)
	}
	if !response.Allowed {
		t.Fatal("allowed debe ser true")
	}
	if len(response.Roots) != 3 {
		t.Fatalf("raíces inesperadas: %+v", response.Roots)
	}
	byPath := make(map[string]uploadRootTarget)
	for _, root := range response.Roots {
		byPath[root.Path] = root
	}
	clips, ok := byPath[clipsRoot]
	if !ok || clips.LibraryName != "Alfa" || clips.LibraryKind != "clips" ||
		clips.MovieCount != 1 || clips.SeriesCount != 0 {
		t.Fatalf("raíz de clips inesperada: %+v", clips)
	}
	a, ok := byPath[rootA]
	if !ok || a.LibraryID != videoID || a.MovieCount != 2 || a.SeriesCount != 1 {
		t.Fatalf("raíz A inesperada: %+v", a)
	}
	b, ok := byPath[rootB]
	if !ok || b.MovieCount != 1 || b.SeriesCount != 1 {
		t.Fatalf("raíz B inesperada: %+v", b)
	}
	if response.Roots[0].LibraryName != "Alfa" ||
		response.Roots[1].Path != rootA || response.Roots[2].Path != rootB {
		t.Fatalf("orden de raíces inesperado: %+v", response.Roots)
	}
	if _, ok := byPath[filepath.Join(base, "music")]; ok {
		t.Fatal("la biblioteca de música no debe aparecer")
	}

	setAllowFileChanges(t, ts, false)
	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/uploads/targets", ts.token)
	if status != http.StatusOK {
		t.Fatalf("destinos apagados: código = %d, cuerpo = %s", status, body)
	}
	var disabled uploadTargetsResponse
	if err := json.Unmarshal(body, &disabled); err != nil {
		t.Fatalf("destinos apagados: %v", err)
	}
	if disabled.Allowed {
		t.Fatal("allowed debe ser false")
	}

	userToken := nonAdminToken(t, ts)
	if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/uploads/targets", userToken); status != http.StatusForbidden {
		t.Fatalf("destinos sin admin: código = %d, cuerpo = %s", status, body)
	}
}

func TestUploadSerieCapituloSinTemporadas(t *testing.T) {
	ts := newTrashTestServer(t)
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := guidedLibrary(t, ts, "Series", "video", root)
	seriesID := guidedSeriesFixture(t, ts.db, libraryID, "Serie plana")
	seriesDir := filepath.Join(root, "Serie plana")
	if err := os.MkdirAll(seriesDir, 0o775); err != nil {
		t.Fatalf("crear serie: %v", err)
	}
	guidedEpisodeFixture(t, ts.db, libraryID, seriesID,
		filepath.Join(seriesDir, "Capitulo viejo.mkv"), 1, 1)

	status, body := guidedUpload(t, ts, seriesID, 1, 2, "Capitulo.nuevo.MKV", "contenido", ts.token)
	if status != http.StatusCreated {
		t.Fatalf("subida plana: código = %d, cuerpo = %s", status, body)
	}
	var result guidedUploadResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("respuesta de subida: %v", err)
	}
	expected := filepath.Join(seriesDir, "Serie plana S01E02.mkv")
	if result.Path != expected || result.Season != 1 || result.Episode != 2 {
		t.Fatalf("resultado = %+v, se esperaba %s", result, expected)
	}
	stored, err := os.ReadFile(expected)
	if err != nil || string(stored) != "contenido" {
		t.Fatalf("contenido final = %q, %v", stored, err)
	}
}

func TestUploadSerieTemporadaExistenteYNueva(t *testing.T) {
	ts := newTrashTestServer(t)
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := guidedLibrary(t, ts, "Series", "video", root)
	seriesID := guidedSeriesFixture(t, ts.db, libraryID, "Con temporadas")
	seriesDir := filepath.Join(root, "Con temporadas")
	seasonOne := filepath.Join(seriesDir, "Temporada 1")
	if err := os.MkdirAll(seasonOne, 0o775); err != nil {
		t.Fatalf("crear temporada: %v", err)
	}
	guidedEpisodeFixture(t, ts.db, libraryID, seriesID,
		filepath.Join(seasonOne, "Con temporadas S01E01.mkv"), 1, 1)
	guidedEpisodeFixture(t, ts.db, libraryID, seriesID,
		filepath.Join(seasonOne, "Con temporadas S01E02.mkv"), 1, 2)

	status, body := guidedUpload(t, ts, seriesID, 1, 3, "Capitulo.mkv", "tres", ts.token)
	if status != http.StatusCreated {
		t.Fatalf("temporada existente: código = %d, cuerpo = %s", status, body)
	}
	existing := filepath.Join(seasonOne, "Con temporadas S01E03.mkv")
	if _, err := os.Stat(existing); err != nil {
		t.Fatalf("archivo en temporada existente: %v", err)
	}

	status, body = guidedUpload(t, ts, seriesID, 2, 1, "Capitulo.mkv", "nueva", ts.token)
	if status != http.StatusCreated {
		t.Fatalf("temporada nueva: código = %d, cuerpo = %s", status, body)
	}
	seasonTwo := filepath.Join(seriesDir, "Temporada 2")
	newFile := filepath.Join(seasonTwo, "Con temporadas S02E01.mkv")
	if _, err := os.Stat(newFile); err != nil {
		t.Fatalf("archivo en temporada nueva: %v", err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(seasonTwo)
		if err != nil || info.Mode().Perm() != 0o775 {
			t.Fatalf("permisos de temporada nueva = %v, %v", info, err)
		}
	}
}

func TestUploadSerieSubtituloConservaIdioma(t *testing.T) {
	ts := newTrashTestServer(t)
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := guidedLibrary(t, ts, "Series", "video", root)
	seriesID := guidedSeriesFixture(t, ts.db, libraryID, "Con subtítulos")
	seriesDir := filepath.Join(root, "Con subtítulos")
	if err := os.MkdirAll(seriesDir, 0o775); err != nil {
		t.Fatalf("crear serie: %v", err)
	}
	guidedEpisodeFixture(t, ts.db, libraryID, seriesID,
		filepath.Join(seriesDir, "Capitulo.mkv"), 1, 1)

	status, body := guidedUpload(t, ts, seriesID, 1, 5, "Pelicula.ES.SRT", "subtítulos", ts.token)
	if status != http.StatusCreated {
		t.Fatalf("subtítulo: código = %d, cuerpo = %s", status, body)
	}
	expected := filepath.Join(seriesDir, "Con subtítulos S01E05.es.srt")
	if stored, err := os.ReadFile(expected); err != nil || string(stored) != "subtítulos" {
		t.Fatalf("subtítulo final = %q, %v", stored, err)
	}
}

func TestUploadSerieValidacionesYConflictos(t *testing.T) {
	ts := newTrashTestServer(t)
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := guidedLibrary(t, ts, "Series", "video", root)
	seriesID := guidedSeriesFixture(t, ts.db, libraryID, "Validaciones")
	seriesDir := filepath.Join(root, "Validaciones")
	if err := os.MkdirAll(seriesDir, 0o775); err != nil {
		t.Fatalf("crear serie: %v", err)
	}
	guidedEpisodeFixture(t, ts.db, libraryID, seriesID,
		filepath.Join(seriesDir, "Capitulo.mkv"), 1, 1)

	for _, test := range []struct {
		name     string
		season   int
		episode  int
		fileName string
	}{
		{"temporada negativa", -1, 1, "Capitulo.mkv"},
		{"temporada alta", 100, 1, "Capitulo.mkv"},
		{"episodio cero", 1, 0, "Capitulo.mkv"},
		{"episodio alto", 1, 1000, "Capitulo.mkv"},
		{"extensión no válida", 1, 1, "Capitulo.txt"},
	} {
		t.Run(test.name, func(t *testing.T) {
			status, body := guidedUpload(t, ts, seriesID, test.season, test.episode,
				test.fileName, "x", ts.token)
			if status != http.StatusBadRequest {
				t.Fatalf("código = %d, cuerpo = %s", status, body)
			}
		})
	}

	existing := filepath.Join(seriesDir, "Validaciones S01E06.mkv")
	writeTestFile(t, existing, "anterior")
	status, body := guidedUpload(t, ts, seriesID, 1, 6, "Capitulo.mkv", "nuevo", ts.token)
	if status != http.StatusConflict {
		t.Fatalf("existente: código = %d, cuerpo = %s", status, body)
	}
	if stored, _ := os.ReadFile(existing); string(stored) != "anterior" {
		t.Fatalf("se sobrescribió el archivo: %q", stored)
	}

	status, body = guidedUpload(t, ts, 999999, 1, 1, "Capitulo.mkv", "x", ts.token)
	if status != http.StatusNotFound {
		t.Fatalf("serie inexistente: código = %d, cuerpo = %s", status, body)
	}

	emptySeries := guidedSeriesFixture(t, ts.db, libraryID, "Vacía")
	status, body = guidedUpload(t, ts, emptySeries, 1, 1, "Capitulo.mkv", "x", ts.token)
	if status != http.StatusConflict {
		t.Fatalf("serie sin episodios: código = %d, cuerpo = %s", status, body)
	}
}

func TestUploadSerieAjusteApagadoYNoAdmin(t *testing.T) {
	ts := newTrashTestServer(t)
	root := t.TempDir()
	libraryID := guidedLibrary(t, ts, "Series", "video", root)
	seriesID := guidedSeriesFixture(t, ts.db, libraryID, "Permisos")
	seriesDir := filepath.Join(root, "Permisos")
	if err := os.MkdirAll(seriesDir, 0o775); err != nil {
		t.Fatalf("crear serie: %v", err)
	}
	guidedEpisodeFixture(t, ts.db, libraryID, seriesID, filepath.Join(seriesDir, "Capitulo.mkv"), 1, 1)

	status, body := guidedUpload(t, ts, seriesID, 1, 2, "Capitulo.mkv", "x", ts.token)
	if status != http.StatusForbidden {
		t.Fatalf("ajuste apagado: código = %d, cuerpo = %s", status, body)
	}

	setAllowFileChanges(t, ts, true)
	userToken := nonAdminToken(t, ts)
	status, body = guidedUpload(t, ts, seriesID, 1, 2, "Capitulo.mkv", "x", userToken)
	if status != http.StatusForbidden {
		t.Fatalf("no admin: código = %d, cuerpo = %s", status, body)
	}
}

func TestUploadSerieLanzaEscaneo(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	ts.api.scanJob = ts.job
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := guidedLibrary(t, ts, "Series", "video", root)
	seriesID := guidedSeriesFixture(t, ts.db, libraryID, "Escaneo")
	seriesDir := filepath.Join(root, "Escaneo")
	if err := os.MkdirAll(seriesDir, 0o775); err != nil {
		t.Fatalf("crear serie: %v", err)
	}
	guidedEpisodeFixture(t, ts.db, libraryID, seriesID, filepath.Join(seriesDir, "Capitulo.mkv"), 1, 1)

	started := make(chan struct{})
	ts.job.scan = func(context.Context) (Result, error) {
		close(started)
		return Result{}, nil
	}
	status, body := guidedUpload(t, ts, seriesID, 1, 2, "Capitulo.mkv", "x", ts.token)
	if status != http.StatusCreated {
		t.Fatalf("subida: código = %d, cuerpo = %s", status, body)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("la subida de serie no lanzó el escaneo")
	}
}
