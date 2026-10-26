package library

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/catlinux/cinecat/internal/auth"
	"github.com/catlinux/cinecat/internal/metadata"
)

func nullableTestString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableTestInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func nullableTestFloat(value float64) any {
	if value == 0 {
		return nil
	}
	return value
}

type itemFixture struct {
	libraryID        int64
	path             string
	title            string
	displayTitle     string
	originalTitle    string
	year             int
	kind             string
	season           int
	episode          int
	tmdbID           int
	overview         string
	posterURL        string
	imdbID           string
	tagline          string
	genres           string
	runtime          int
	rating           float64
	directors        string
	cast             string
	backdropURL      string
	overviewLanguage string
	metadataSource   string
	matchStatus      string
	matchScore       float64
	seriesID         int64
	episodeTitle     string
	stillURL         string
}

func insertItemFixture(t *testing.T, db *sql.DB, item itemFixture) int64 {
	t.Helper()
	kind := item.kind
	if kind == "" {
		kind = KindMovie
	}
	res, err := db.Exec(`
		INSERT INTO media_items (
			library_id, path, title, display_title, original_title, year,
			kind, season, episode, tmdb_id, overview, poster_url, imdb_id,
			tagline, genres, runtime_minutes, rating, directors, cast_names,
			backdrop_url, overview_language, metadata_source, match_status,
			match_score, series_id, episode_title, still_url
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.libraryID, item.path, item.title,
		nullableTestString(item.displayTitle), nullableTestString(item.originalTitle),
		nullableTestInt(item.year), kind, nullableTestInt(item.season),
		nullableTestInt(item.episode), nullableTestInt(item.tmdbID),
		nullableTestString(item.overview), nullableTestString(item.posterURL),
		nullableTestString(item.imdbID), nullableTestString(item.tagline),
		nullableTestString(item.genres), nullableTestInt(item.runtime),
		nullableTestFloat(item.rating), nullableTestString(item.directors),
		nullableTestString(item.cast), nullableTestString(item.backdropURL),
		nullableTestString(item.overviewLanguage), nullableTestString(item.metadataSource),
		nullableTestString(item.matchStatus), nullableTestFloat(item.matchScore),
		nullableTestInt(int(item.seriesID)), nullableTestString(item.episodeTitle),
		nullableTestString(item.stillURL),
	)
	if err != nil {
		t.Fatalf("insertar ítem %q: %v", item.path, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("id del ítem %q: %v", item.path, err)
	}
	return id
}

func libraryHTTPDo(t *testing.T, method, rawURL, token string, body any) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, rawURL, reader)
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, rawURL, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leer respuesta: %v", err)
	}
	return resp.StatusCode, data
}

func nonAdminToken(t *testing.T, ts *libraryTestServer) string {
	t.Helper()
	hash, err := auth.HashPassword("Clave-segura-2")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if _, err := ts.db.Exec(
		`INSERT INTO users (username, password_hash, role) VALUES ('bea', ?, 'user')`,
		hash,
	); err != nil {
		t.Fatalf("crear usuario no admin: %v", err)
	}
	token, _, err := ts.auth.Login(context.Background(), "bea", "Clave-segura-2")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	return token
}

func TestItemsFiltrosYOrden(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())

	alpha := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/Alpha (2000).mkv", title: "Alpha",
		displayTitle: "Alfa", originalTitle: "Alpha original", year: 2000,
		matchStatus: "auto", genres: `["Drama"]`, directors: `["Directora"]`,
		cast: `["Actor"]`, runtime: 100, rating: 8.1,
	})
	beta := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/Beta (2010).mkv", title: "Beta",
		year: 2010, matchStatus: "none",
	})
	hundred := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/100% Cine.mkv", title: "100% Cine",
	})
	underscore := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/A_B (1990).mkv", title: "A_B",
		year: 1990, matchStatus: "manual",
	})
	episode := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/Serie/S01E01.mkv", title: "Capítulo",
		year: 2020, kind: KindEpisode,
	})

	get := func(query string) itemsResponse {
		t.Helper()
		status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items?"+query, ts.token)
		if status != http.StatusOK {
			t.Fatalf("%s: código = %d, cuerpo = %s", query, status, body)
		}
		var response itemsResponse
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("%s: respuesta: %v", query, err)
		}
		if response.Total != int64(len(response.Items)) {
			t.Fatalf("%s: total = %d, ítems = %d", query, response.Total, len(response.Items))
		}
		return response
	}
	ids := func(response itemsResponse) []int64 {
		values := make([]int64, 0, len(response.Items))
		for _, item := range response.Items {
			values = append(values, item.ID)
		}
		return values
	}

	movies := get("kind=movie")
	if movies.Total != 4 || len(movies.Items) != 4 {
		t.Fatalf("kind=movie: %+v", movies)
	}
	episodes := get("kind=episode")
	if episodes.Total != 1 || episodes.Items[0].ID != episode {
		t.Fatalf("kind=episode: %+v", episodes)
	}

	byDisplay := get("q=ALFA")
	if byDisplay.Total != 1 || byDisplay.Items[0].ID != alpha {
		t.Fatalf("q por display_title: %+v", byDisplay)
	}
	byOriginal := get("q=alpha+original")
	if byOriginal.Total != 1 || byOriginal.Items[0].ID != alpha {
		t.Fatalf("q por original_title: %+v", byOriginal)
	}
	byPercent := get("q=" + url.QueryEscape("100%"))
	if byPercent.Total != 1 || byPercent.Items[0].ID != hundred {
		t.Fatalf("q con %% literal: %+v", byPercent)
	}
	byUnderscore := get("q=_")
	if byUnderscore.Total != 1 || byUnderscore.Items[0].ID != underscore {
		t.Fatalf("q con _ literal: %+v", byUnderscore)
	}

	unmatched := get("unmatched=true")
	if unmatched.Total != 2 {
		t.Fatalf("unmatched: %+v", unmatched)
	}
	unmatchedIDs := map[int64]bool{}
	for _, item := range unmatched.Items {
		unmatchedIDs[item.ID] = true
	}
	if !unmatchedIDs[beta] || !unmatchedIDs[hundred] {
		t.Fatalf("unmatched inesperado: %+v", unmatched)
	}

	if got := ids(get("")); len(got) != 5 || got[0] != alpha || got[4] != episode {
		t.Fatalf("orden por id: %v", got)
	}
	if got := ids(get("sort=title")); len(got) != 5 || got[0] != hundred || got[1] != underscore {
		t.Fatalf("orden por título: %v", got)
	}
	if got := ids(get("sort=year")); len(got) != 5 || got[0] != episode || got[1] != beta || got[4] != hundred {
		t.Fatalf("orden por año: %v", got)
	}
	if got := ids(get("sort=added")); len(got) != 5 || got[0] != episode || got[4] != alpha {
		t.Fatalf("orden por añadido: %v", got)
	}

	for _, invalid := range []string{"kind=serie", "sort=foo", "unmatched=quiz", "library_id=abc"} {
		status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items?"+invalid, ts.token)
		if status != http.StatusBadRequest {
			t.Fatalf("%s: código = %d, cuerpo = %s", invalid, status, body)
		}
	}
}

func TestItemsDTOFichaCompleta(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	id := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/Pelicula (2019).mkv", title: "Pelicula",
		displayTitle: "Título visible", originalTitle: "Original", year: 2019,
		tmdbID: 42, overview: "Sinopsis", posterURL: "poster", imdbID: "tt1",
		tagline: "Lema", genres: `["Drama"]`, runtime: 100, rating: 8.2,
		directors: `["Directora"]`, cast: `["Actor"]`, backdropURL: "backdrop",
		overviewLanguage: "es-ES", metadataSource: "tmdb", matchStatus: "auto",
		matchScore: 0.91,
	})
	empty := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/Otra.mkv", title: "Otra",
		genres: "no es json", directors: "[]", cast: "null",
	})

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(id, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("ficha: código = %d, cuerpo = %s", status, body)
	}
	var item mediaItemDTO
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("respuesta: %v", err)
	}
	if item.Title != "Título visible" || item.FileTitle != "Pelicula" {
		t.Fatalf("títulos = %q / %q", item.Title, item.FileTitle)
	}
	if item.OriginalTitle == nil || *item.OriginalTitle != "Original" ||
		item.IMDbID == nil || *item.IMDbID != "tt1" ||
		item.Tagline == nil || *item.Tagline != "Lema" ||
		item.RuntimeMinutes == nil || *item.RuntimeMinutes != 100 ||
		item.Rating == nil || *item.Rating != 8.2 ||
		item.MatchStatus == nil || *item.MatchStatus != "auto" ||
		item.MatchScore == nil || *item.MatchScore != 0.91 {
		t.Fatalf("ficha incompleta: %+v", item)
	}
	if len(item.Genres) != 1 || item.Genres[0] != "Drama" ||
		len(item.Directors) != 1 || item.Directors[0] != "Directora" ||
		len(item.Cast) != 1 || item.Cast[0] != "Actor" {
		t.Fatalf("arrays = %+v / %+v / %+v", item.Genres, item.Directors, item.Cast)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(empty, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("ficha vacía: código = %d", status)
	}
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("respuesta vacía: %v", err)
	}
	if item.Genres == nil || item.Directors == nil || item.Cast == nil ||
		len(item.Genres) != 0 || len(item.Directors) != 0 || len(item.Cast) != 0 {
		t.Fatalf("arrays vacíos deben ser []: %+v / %+v / %+v", item.Genres, item.Directors, item.Cast)
	}
}

func TestCorreccionCandidatos(t *testing.T) {
	fake := &fakeTMDB{details: &metadata.Details{TMDBID: 42, Title: "Ficha"}}
	ts := newLibraryHTTPTestServer(t, fake)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	id := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/Pelicula (2019).mkv", title: "Pelicula",
	})
	base := ts.URL + "/api/v1/items/" + strconv.FormatInt(id, 10) + "/candidates"

	status, body := libraryHTTPGet(t, base, ts.token)
	if status != http.StatusOK {
		t.Fatalf("candidatos: código = %d, cuerpo = %s", status, body)
	}
	if fake.lastQuery != "Pelicula" || fake.lastYear != 0 {
		t.Fatalf("búsqueda por defecto = %q, %d", fake.lastQuery, fake.lastYear)
	}
	var response struct {
		Candidates []metadata.Candidate `json:"candidates"`
	}
	if err := json.Unmarshal(body, &response); err != nil || len(response.Candidates) != 1 {
		t.Fatalf("candidatos inesperados: %s (%v)", body, err)
	}

	status, body = libraryHTTPGet(t, base+"?q=Otra&year=2020", ts.token)
	if status != http.StatusOK {
		t.Fatalf("candidatos con q/year: código = %d", status)
	}
	if fake.lastQuery != "Otra" || fake.lastYear != 2020 {
		t.Fatalf("búsqueda = %q, %d", fake.lastQuery, fake.lastYear)
	}

	fake.searchErr = metadata.ErrNotFound
	status, body = libraryHTTPGet(t, base+"?q=nada", ts.token)
	if status != http.StatusOK {
		t.Fatalf("sin resultados: código = %d", status)
	}
	response.Candidates = nil
	if err := json.Unmarshal(body, &response); err != nil || response.Candidates == nil || len(response.Candidates) != 0 {
		t.Fatalf("candidatos vacíos inesperados: %s", body)
	}

	for _, invalid := range []string{"?year=abc", "?year=-1"} {
		if status, body := libraryHTTPGet(t, base+invalid, ts.token); status != http.StatusBadRequest {
			t.Fatalf("%s: código = %d, cuerpo = %s", invalid, status, body)
		}
	}

	fake.searchErr = errors.New("red de prueba")
	status, body = libraryHTTPGet(t, base+"?q=fallo", ts.token)
	if status != http.StatusBadGateway {
		t.Fatalf("error de TMDB: código = %d, cuerpo = %s", status, body)
	}
	var failure map[string]string
	if err := json.Unmarshal(body, &failure); err != nil || failure["error"] != "no se pudo consultar TMDB" {
		t.Fatalf("error de TMDB inesperado: %s", body)
	}
}

func TestCorreccionMatch(t *testing.T) {
	fake := &fakeTMDB{details: &metadata.Details{
		TMDBID: 123, Title: "Ficha manual", OriginalTitle: "Original",
		Year: 2020, Overview: "Sinopsis", Genres: []string{"Drama"},
		Directors: []string{"Directora"}, Cast: []string{"Actor"},
	}}
	ts := newLibraryHTTPTestServer(t, fake)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	id := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/Pelicula (2019).mkv", title: "Pelicula",
	})
	base := ts.URL + "/api/v1/items/" + strconv.FormatInt(id, 10) + "/match"

	status, body := libraryHTTPDo(t, http.MethodPut, base, ts.token, map[string]any{"tmdb_id": 123})
	if status != http.StatusOK {
		t.Fatalf("PUT match: código = %d, cuerpo = %s", status, body)
	}
	var item mediaItemDTO
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("respuesta: %v", err)
	}
	if item.Title != "Ficha manual" || item.FileTitle != "Pelicula" ||
		item.MatchStatus == nil || *item.MatchStatus != "manual" ||
		item.MatchScore == nil || *item.MatchScore != 1 ||
		item.TMDBID == nil || *item.TMDBID != 123 {
		t.Fatalf("ítem corregido inesperado: %+v", item)
	}

	status, body = libraryHTTPDo(t, http.MethodPut, base, ts.token, map[string]any{"tmdb_id": nil})
	if status != http.StatusOK {
		t.Fatalf("PUT null: código = %d, cuerpo = %s", status, body)
	}
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("respuesta null: %v", err)
	}
	if item.MatchStatus == nil || *item.MatchStatus != "none" ||
		item.TMDBID != nil || item.Title != "Pelicula" ||
		item.Genres == nil || len(item.Genres) != 0 {
		t.Fatalf("ítem sin coincidencia inesperado: %+v", item)
	}

	for _, invalid := range []any{
		map[string]any{},
		map[string]any{"tmdb_id": 0},
		map[string]any{"tmdb_id": "abc"},
	} {
		status, body := libraryHTTPDo(t, http.MethodPut, base, ts.token, invalid)
		if status != http.StatusBadRequest {
			t.Fatalf("cuerpo %v: código = %d, cuerpo = %s", invalid, status, body)
		}
	}

	fake.detailsErr = errors.New("red de prueba")
	status, body = libraryHTTPDo(t, http.MethodPut, base, ts.token, map[string]any{"tmdb_id": 123})
	if status != http.StatusBadGateway {
		t.Fatalf("error de TMDB: código = %d, cuerpo = %s", status, body)
	}
}

func TestCorreccionDeleteMatch(t *testing.T) {
	fake := &fakeTMDB{details: &metadata.Details{
		TMDBID: 42, Title: "Ficha automática", Year: 2019,
		Overview: "Sinopsis", Genres: []string{"Drama"},
	}}
	ts := newLibraryHTTPTestServer(t, fake)
	root := t.TempDir()
	libraryID := createLibrary(t, ts.db, root)
	id := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(root, "Pelicula (2019).mkv"),
		title: "Pelicula", displayTitle: "Ficha anterior", year: 2025,
		tmdbID: 9, metadataSource: "tmdb", matchStatus: "manual", matchScore: 1,
		genres: `["Antiguo"]`,
	})
	base := ts.URL + "/api/v1/items/" + strconv.FormatInt(id, 10) + "/match"

	status, body := libraryHTTPDo(t, http.MethodDelete, base, ts.token, nil)
	if status != http.StatusOK {
		t.Fatalf("DELETE match: código = %d, cuerpo = %s", status, body)
	}
	var item mediaItemDTO
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("respuesta: %v", err)
	}
	if item.MatchStatus == nil || *item.MatchStatus != "auto" ||
		item.TMDBID == nil || *item.TMDBID != 42 ||
		item.Title != "Ficha automática" || item.FileTitle != "Pelicula" {
		t.Fatalf("ítem reidentificado inesperado: %+v", item)
	}

	// Sin coincidencia se restaura el año del archivo, no el de TMDB.
	noMatch := &fakeTMDB{searchErr: metadata.ErrNotFound}
	ts2 := newLibraryHTTPTestServer(t, noMatch)
	root2 := filepath.Join(t.TempDir(), "Pelicula (1999)")
	libraryID2 := createLibrary(t, ts2.db, root2)
	id2 := insertItemFixture(t, ts2.db, itemFixture{
		libraryID: libraryID2, path: filepath.Join(root2, "Foo (2010)", "foo.mkv"),
		title: "foo", displayTitle: "Ficha anterior", year: 2025,
		tmdbID: 9, metadataSource: "tmdb", matchStatus: "manual", matchScore: 1,
	})
	base2 := ts2.URL + "/api/v1/items/" + strconv.FormatInt(id2, 10) + "/match"

	status, body = libraryHTTPDo(t, http.MethodDelete, base2, ts2.token, nil)
	if status != http.StatusOK {
		t.Fatalf("DELETE sin coincidencia: código = %d, cuerpo = %s", status, body)
	}
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("respuesta sin coincidencia: %v", err)
	}
	if item.Year == nil || *item.Year != 2010 {
		t.Fatalf("año restaurado = %+v", item.Year)
	}
	if item.MatchStatus != nil || item.TMDBID != nil || item.Title != "foo" ||
		item.Genres == nil || len(item.Genres) != 0 {
		t.Fatalf("ítem sin identificar inesperado: %+v", item)
	}
}

func TestCorreccionDeleteUsaRutaRelativa(t *testing.T) {
	noMatch := &fakeTMDB{searchErr: metadata.ErrNotFound}
	ts := newLibraryHTTPTestServer(t, noMatch)
	root := filepath.Join(t.TempDir(), "foo (1999)")
	libraryID := createLibrary(t, ts.db, root)
	id := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(root, "foo.mkv"),
		title: "foo", displayTitle: "Ficha anterior", year: 2025,
		tmdbID: 9, metadataSource: "tmdb", matchStatus: "manual", matchScore: 1,
	})
	base := ts.URL + "/api/v1/items/" + strconv.FormatInt(id, 10) + "/match"

	status, body := libraryHTTPDo(t, http.MethodDelete, base, ts.token, nil)
	if status != http.StatusOK {
		t.Fatalf("DELETE match: código = %d, cuerpo = %s", status, body)
	}
	var item mediaItemDTO
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("respuesta: %v", err)
	}
	if item.Year != nil {
		t.Fatalf("no debe tomar el año de la raíz de la biblioteca: %+v", item.Year)
	}
}

func TestCorreccionPermisosYErrores(t *testing.T) {
	fake := &fakeTMDB{details: &metadata.Details{TMDBID: 42, Title: "Ficha"}}
	ts := newLibraryHTTPTestServer(t, fake)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	movie := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/Pelicula.mkv", title: "Pelicula",
	})
	episode := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/Serie/S01E01.mkv", title: "Capítulo",
		kind: KindEpisode,
	})
	userToken := nonAdminToken(t, ts)

	movieBase := ts.URL + "/api/v1/items/" + strconv.FormatInt(movie, 10)
	episodeBase := ts.URL + "/api/v1/items/" + strconv.FormatInt(episode, 10)
	for _, request := range []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodGet, movieBase + "/candidates", nil},
		{http.MethodPut, movieBase + "/match", map[string]any{"tmdb_id": 42}},
		{http.MethodDelete, movieBase + "/match", nil},
	} {
		if status, body := libraryHTTPDo(t, request.method, request.path, userToken, request.body); status != http.StatusForbidden {
			t.Fatalf("%s %s sin admin: código = %d, cuerpo = %s", request.method, request.path, status, body)
		}
		if status, body := libraryHTTPDo(t, request.method, request.path, "", request.body); status != http.StatusUnauthorized {
			t.Fatalf("%s %s sin sesión: código = %d, cuerpo = %s", request.method, request.path, status, body)
		}
		for _, missing := range []string{
			ts.URL + "/api/v1/items/999999/candidates",
			episodeBase + "/candidates",
		} {
			if request.method == http.MethodGet {
				if status, body := libraryHTTPDo(t, request.method, missing, ts.token, nil); status != http.StatusNotFound {
					t.Fatalf("%s: código = %d, cuerpo = %s", missing, status, body)
				}
			}
		}
		if request.method != http.MethodGet {
			if status, body := libraryHTTPDo(t, request.method, episodeBase+"/match", ts.token, request.body); status != http.StatusNotFound {
				t.Fatalf("%s episodio: código = %d, cuerpo = %s", request.method, status, body)
			}
			if status, body := libraryHTTPDo(t, request.method, ts.URL+"/api/v1/items/999999/match", ts.token, request.body); status != http.StatusNotFound {
				t.Fatalf("%s inexistente: código = %d, cuerpo = %s", request.method, status, body)
			}
		}
	}

	withoutTMDB := newLibraryHTTPTestServer(t)
	libraryID = createLibrary(t, withoutTMDB.db, t.TempDir())
	id := insertItemFixture(t, withoutTMDB.db, itemFixture{
		libraryID: libraryID, path: "/media/Pelicula.mkv", title: "Pelicula",
	})
	base := withoutTMDB.URL + "/api/v1/items/" + strconv.FormatInt(id, 10)
	for _, request := range []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodGet, base + "/candidates", nil},
		{http.MethodPut, base + "/match", map[string]any{"tmdb_id": 42}},
		{http.MethodDelete, base + "/match", nil},
	} {
		status, body := libraryHTTPDo(t, request.method, request.path, withoutTMDB.token, request.body)
		if status != http.StatusServiceUnavailable {
			t.Fatalf("%s %s sin TMDB: código = %d, cuerpo = %s", request.method, request.path, status, body)
		}
		var response map[string]string
		if err := json.Unmarshal(body, &response); err != nil || response["error"] != "TMDB no está configurado" {
			t.Fatalf("error sin TMDB inesperado: %s", body)
		}
	}
}
