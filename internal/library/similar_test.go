package library

import (
	"encoding/json"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
)

func TestSimilarScoreControlado(t *testing.T) {
	current := similarProfile{
		id:          1,
		genres:      []string{"Drama", "Acción"},
		directors:   []string{"Nolan"},
		cast:        []string{"Actor A", "Actor B"},
		collections: map[int64]bool{10: true},
		year:        2001,
		rating:      8,
		hasRating:   true,
	}
	candidate := similarProfile{
		id:          2,
		genres:      []string{"Drama", "Comedia"},
		directors:   []string{"Nolan"},
		cast:        []string{"Actor A", "Actor C"},
		collections: map[int64]bool{10: true},
		year:        2005,
		rating:      8.5,
		hasRating:   true,
	}

	score, ok := similarScore(current, candidate)
	if !ok {
		t.Fatal("el candidato debería ser similar")
	}
	want := 4.0/3.0 + 3 + 2 + 0.4 + 0.5 + 0.3
	if math.Abs(score-want) > 1e-9 {
		t.Fatalf("puntuación = %.6f, se esperaba %.6f", score, want)
	}

	onlySharedFacts := similarProfile{
		id: 3, genres: []string{"Comedia"}, collections: map[int64]bool{10: true},
		year: 2001, rating: 8, hasRating: true,
	}
	if _, ok := similarScore(current, onlySharedFacts); ok {
		t.Fatal("sin género, dirección ni actor compartido no debe ser similar")
	}

	if _, ok := similarScore(similarProfile{id: 4}, candidate); ok {
		t.Fatal("un ítem sin metadatos no debe tener similares")
	}

	manyActors := similarProfile{
		id: 5, genres: []string{"Drama"},
		cast: []string{"Actor A", "Actor B", "Actor C", "Actor D", "Actor E", "Actor F"},
	}
	score, ok = similarScore(
		similarProfile{id: 1, genres: []string{"Drama"}, cast: manyActors.cast},
		manyActors,
	)
	if !ok || math.Abs(score-(4+2)) > 1e-9 {
		t.Fatalf("la puntuación de actores debe estar limitada a 2: %.6f", score)
	}
}

func TestRankSimilarDesempataPorNotaYID(t *testing.T) {
	current := similarProfile{genres: []string{"Drama"}}
	candidates := []similarProfile{
		{id: 3, genres: []string{"Drama"}, rating: 7, hasRating: true},
		{id: 2, genres: []string{"Drama"}, rating: 8, hasRating: true},
		{id: 1, genres: []string{"Drama"}, rating: 8, hasRating: true},
		{id: 4, genres: []string{"Drama"}},
	}
	ranked := rankSimilar(current, candidates)
	wantIDs := []int64{1, 2, 3, 4}
	for index, want := range wantIDs {
		if ranked[index].id != want {
			t.Fatalf("orden = %v, se esperaba %v", ranked, wantIDs)
		}
	}
}

func TestSimilarItemsHTTP(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())

	current := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      filepath.Join(t.TempDir(), "actual.mkv"),
		title:     "Actual",
		kind:      KindMovie,
		posterURL: "https://images.example/actual.jpg",
		genres:    `["Drama","Acción"]`,
		directors: `["Nolan"]`,
		cast:      `["Actor A","Actor B"]`,
		year:      2001,
		rating:    8,
	})
	first := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      filepath.Join(t.TempDir(), "primera.mkv"),
		title:     "Primera",
		kind:      KindMovie,
		posterURL: "https://images.example/primera.jpg",
		genres:    `["Drama"]`,
		directors: `["Nolan"]`,
		cast:      `["Actor A"]`,
		year:      2005,
		rating:    8.5,
	})
	second := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      filepath.Join(t.TempDir(), "segunda.mkv"),
		title:     "Segunda",
		kind:      KindMovie,
		posterURL: "https://images.example/segunda.jpg",
		genres:    `["Drama"]`,
		cast:      `["Actor A"]`,
		year:      2010,
		rating:    7,
	})
	insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      filepath.Join(t.TempDir(), "sinposter.mkv"),
		title:     "Sin cartel",
		kind:      KindMovie,
		genres:    `["Drama"]`,
		cast:      `["Actor A"]`,
	})
	insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      filepath.Join(t.TempDir(), "sinrelacion.mkv"),
		title:     "Sin relación",
		kind:      KindMovie,
		posterURL: "https://images.example/otra.jpg",
		genres:    `["Comedia"]`,
	})

	userID := progressUserID(t, ts.db, "ana")
	setWatchState(t, ts.db, userID, first)
	setProgressRow(t, ts.db, userID, first, 120, 600, "2026-10-26T10:00:00Z")
	if _, err := ts.db.Exec(`UPDATE media_items SET credits = ? WHERE id = ?`,
		`{"cast":[{"name":"Actor A","character":"Heroína","photo":"https://images.example/a.jpg"}],"crew":[{"name":"Nolan","job":"Director","photo":""}]}`,
		first); err != nil {
		t.Fatalf("credits: %v", err)
	}

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(current, 10)+"/similar?limit=1", ts.token)
	if status != http.StatusOK {
		t.Fatalf("similares: código = %d, cuerpo = %s", status, body)
	}
	var response similarItemsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("similares: %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].ID != first {
		t.Fatalf("similares inesperados: %+v", response.Items)
	}
	item := response.Items[0]
	if !item.Watched || item.Progress == nil || item.Progress.PositionSeconds != 120 {
		t.Fatalf("estado del usuario ausente: %+v", item)
	}
	if len(item.Credits.Cast) != 0 || len(item.Credits.Crew) != 0 {
		t.Fatalf("los similares no deben incluir credits: %+v", item.Credits)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(current, 10)+"/similar", ts.token)
	if status != http.StatusOK {
		t.Fatalf("similares por defecto: código = %d", status)
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("similares por defecto: %v", err)
	}
	if len(response.Items) != 2 || response.Items[0].ID != first || response.Items[1].ID != second {
		t.Fatalf("orden de similares inesperado: %+v", response.Items)
	}

	if status, _ := libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(current, 10)+"/similar?limit=25", ts.token); status != http.StatusBadRequest {
		t.Fatalf("límite excesivo: código = %d", status)
	}
	if status, _ := libraryHTTPGet(t, ts.URL+"/api/v1/items/999999/similar", ts.token); status != http.StatusNotFound {
		t.Fatalf("inexistente: código = %d", status)
	}
	if status, _ := libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(first, 10)+"/similar", ""); status != http.StatusUnauthorized {
		t.Fatalf("sin sesión: código = %d", status)
	}

	episode := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      filepath.Join(t.TempDir(), "episodio.mkv"),
		title:     "Episodio",
		kind:      KindEpisode,
		posterURL: "https://images.example/episodio.jpg",
		genres:    `["Drama"]`,
	})
	if status, _ := libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(episode, 10)+"/similar", ts.token); status != http.StatusNotFound {
		t.Fatalf("episodio: código = %d", status)
	}
}

func TestSimilarSinMetadatosYArraysVacios(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	current := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(t.TempDir(), "sin-meta.mkv"),
		title: "Sin metadatos", kind: KindMovie, posterURL: "https://images.example/p.jpg",
	})

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(current, 10)+"/similar", ts.token)
	if status != http.StatusOK {
		t.Fatalf("sin metadatos: código = %d, cuerpo = %s", status, body)
	}
	if string(body) == "" {
		t.Fatal("respuesta vacía")
	}
	var response similarItemsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("respuesta: %v", err)
	}
	if response.Items == nil || len(response.Items) != 0 {
		t.Fatalf("items debe ser un array vacío: %+v", response.Items)
	}
}

func TestDTODeCreditsSiemprePresente(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	movie := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(t.TempDir(), "creditos-vacios.mkv"),
		title: "Sin créditos", kind: KindMovie,
	})
	episode := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(t.TempDir(), "episodio.mkv"),
		title: "Episodio", kind: KindEpisode,
	})
	seriesID := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Sin créditos", title: "Sin créditos",
		displayTitle: "Sin créditos", year: 2000,
	})

	for _, url := range []string{
		ts.URL + "/api/v1/items/" + strconv.FormatInt(movie, 10),
		ts.URL + "/api/v1/items/" + strconv.FormatInt(episode, 10),
	} {
		status, body := libraryHTTPGet(t, url, ts.token)
		if status != http.StatusOK {
			t.Fatalf("%s: código = %d", url, status)
		}
		var item mediaItemDTO
		if err := json.Unmarshal(body, &item); err != nil {
			t.Fatalf("%s: %v", url, err)
		}
		if item.Credits.Cast == nil || item.Credits.Crew == nil ||
			len(item.Credits.Cast) != 0 || len(item.Credits.Crew) != 0 {
			t.Fatalf("%s: credits vacíos inesperados: %+v", url, item.Credits)
		}
	}

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/series/"+strconv.FormatInt(seriesID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("serie: código = %d, cuerpo = %s", status, body)
	}
	var series seriesDetailDTO
	if err := json.Unmarshal(body, &series); err != nil {
		t.Fatalf("serie: %v", err)
	}
	if series.Credits.Cast == nil || series.Credits.Crew == nil {
		t.Fatalf("credits de serie deben ser arrays: %+v", series.Credits)
	}

	if _, err := ts.db.Exec(`UPDATE media_items SET credits = ? WHERE id = ?`,
		`{"cast":[{"name":"Actor","character":"Papel","photo":"https://images.example/a.jpg"}],"crew":[{"name":"Dirección","job":"Director","photo":""}]}`,
		movie); err != nil {
		t.Fatalf("credits: %v", err)
	}
	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(movie, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("ítem con credits: código = %d", status)
	}
	var item mediaItemDTO
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("ítem con credits: %v", err)
	}
	if len(item.Credits.Cast) != 1 || item.Credits.Cast[0].PhotoURL != "https://images.example/a.jpg" ||
		len(item.Credits.Crew) != 1 || item.Credits.Crew[0].Job != "Director" {
		t.Fatalf("credits del DTO inesperados: %+v", item.Credits)
	}
}

func TestSimilarSeriesHTTP(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	current := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Actual", title: "Actual",
		displayTitle: "Actual", year: 2001, posterURL: "https://images.example/serie-actual.jpg",
		genres: `["Drama"]`, directors: `["Creadora"]`, cast: `["Actriz A"]`,
	})
	first := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Primera", title: "Primera",
		displayTitle: "Primera", year: 2004, posterURL: "https://images.example/serie-primera.jpg",
		genres: `["Drama"]`, directors: `["Creadora"]`, cast: `["Actriz A"]`,
	})
	second := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Segunda", title: "Segunda",
		displayTitle: "Segunda", year: 2010, posterURL: "https://images.example/serie-segunda.jpg",
		genres: `["Drama"]`, cast: `["Actriz A"]`,
	})
	insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Sin cartel", title: "Sin cartel",
		displayTitle: "Sin cartel", year: 2002, genres: `["Drama"]`, cast: `["Actriz A"]`,
	})
	insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Sin relación", title: "Sin relación",
		displayTitle: "Sin relación", year: 2002, posterURL: "https://images.example/otra-serie.jpg",
		genres: `["Comedia"]`,
	})

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/series/"+strconv.FormatInt(current, 10)+"/similar?limit=1", ts.token)
	if status != http.StatusOK {
		t.Fatalf("series similares: código = %d, cuerpo = %s", status, body)
	}
	var response similarSeriesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("series similares: %v", err)
	}
	if len(response.Series) != 1 || response.Series[0].ID != first {
		t.Fatalf("series similares inesperadas: %+v", response.Series)
	}
	if response.Series[0].Title != "Primera" || response.Series[0].PosterURL == nil {
		t.Fatalf("resumen de serie inesperado: %+v", response.Series[0])
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/series/"+strconv.FormatInt(current, 10)+"/similar", ts.token)
	if status != http.StatusOK {
		t.Fatalf("series similares por defecto: código = %d", status)
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("series similares por defecto: %v", err)
	}
	if len(response.Series) != 2 || response.Series[0].ID != first || response.Series[1].ID != second {
		t.Fatalf("orden de series similares inesperado: %+v", response.Series)
	}

	if status, _ := libraryHTTPGet(t, ts.URL+"/api/v1/series/999999/similar", ts.token); status != http.StatusNotFound {
		t.Fatalf("serie inexistente: código = %d", status)
	}
	if status, _ := libraryHTTPGet(t, ts.URL+"/api/v1/series/"+strconv.FormatInt(current, 10)+"/similar?limit=25", ts.token); status != http.StatusBadRequest {
		t.Fatalf("serie con límite excesivo: código = %d", status)
	}
	if status, _ := libraryHTTPGet(t, ts.URL+"/api/v1/series/"+strconv.FormatInt(current, 10)+"/similar", ""); status != http.StatusUnauthorized {
		t.Fatalf("serie sin sesión: código = %d", status)
	}

	empty := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Sin metadatos", title: "Sin metadatos",
		displayTitle: "Sin metadatos", year: 2000, posterURL: "https://images.example/sin-meta.jpg",
	})
	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/series/"+strconv.FormatInt(empty, 10)+"/similar", ts.token)
	if status != http.StatusOK {
		t.Fatalf("serie sin metadatos: código = %d", status)
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("serie sin metadatos: %v", err)
	}
	if response.Series == nil || len(response.Series) != 0 {
		t.Fatalf("serie sin metadatos debe devolver un array vacío: %+v", response.Series)
	}
}
