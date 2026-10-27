package library

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func collectionHTTPMovie(t *testing.T, ts *libraryTestServer, title string, year int, poster, kind string) int64 {
	t.Helper()
	if kind == "" {
		kind = KindMovie
	}
	libraryID := createLibrary(t, ts.db, t.TempDir())
	return insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      filepath.Join(t.TempDir(), title+".mkv"),
		title:     title,
		year:      year,
		kind:      kind,
		posterURL: poster,
	})
}

func collectionHTTPSeed(t *testing.T, db *sql.DB, name, source string, tmdbID any, poster string, edited, deleted int) int64 {
	t.Helper()
	var posterValue any
	if poster != "" {
		posterValue = poster
	}
	result, err := db.Exec(`
		INSERT INTO collections (
			name, poster_url, tmdb_collection_id, source, edited, deleted
		) VALUES (?, ?, ?, ?, ?, ?)`,
		name, posterValue, tmdbID, source, edited, deleted)
	if err != nil {
		t.Fatalf("crear colección %q: %v", name, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de colección %q: %v", name, err)
	}
	return id
}

func collectionHTTPMember(t *testing.T, db *sql.DB, collectionID, itemID int64, position any, excluded int) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO collection_items (collection_id, media_item_id, position, excluded)
		VALUES (?, ?, ?, ?)`, collectionID, itemID, position, excluded); err != nil {
		t.Fatalf("añadir miembro: %v", err)
	}
}

func collectionHTTPURL(ts *libraryTestServer, id int64) string {
	return ts.URL + "/api/v1/collections/" + strconv.FormatInt(id, 10)
}

func TestCollectionsListVisibilidadYPoster(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	manual := collectionHTTPSeed(t, ts.db, "Manual vacía", "manual", nil, "", 0, 0)
	auto := collectionHTTPSeed(t, ts.db, "Saga visible", "tmdb", 900, "", 0, 0)
	singleCollection := collectionHTTPSeed(t, ts.db, "Saga de una", "tmdb", 901, "", 0, 0)
	excluded := collectionHTTPSeed(t, ts.db, "Saga excluida", "tmdb", 902, "", 0, 0)
	deleted := collectionHTTPSeed(t, ts.db, "Borrada", "manual", nil, "", 0, 1)

	old := collectionHTTPMovie(t, ts, "Antigua", 1999, "https://images.example/antigua.jpg", KindMovie)
	recent := collectionHTTPMovie(t, ts, "Reciente", 2001, "https://images.example/reciente.jpg", KindMovie)
	noPoster := collectionHTTPMovie(t, ts, "Sin cartel", 1990, "", KindMovie)
	collectionHTTPMember(t, ts.db, auto, old, nil, 0)
	collectionHTTPMember(t, ts.db, auto, recent, nil, 0)
	collectionHTTPMember(t, ts.db, auto, noPoster, nil, 0)
	collectionHTTPMember(t, ts.db, excluded, old, nil, 0)
	collectionHTTPMember(t, ts.db, excluded, recent, nil, 1)
	single := collectionHTTPMovie(t, ts, "Sola", 2005, "", KindMovie)
	collectionHTTPMember(t, ts.db, singleCollection, single, nil, 0)

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/collections?limit=200", ts.token)
	if status != http.StatusOK {
		t.Fatalf("listar colecciones: código = %d, cuerpo = %s", status, body)
	}
	var response collectionsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("respuesta: %v", err)
	}
	if response.Total != 2 || len(response.Collections) != 2 {
		t.Fatalf("colecciones visibles inesperadas: %+v", response)
	}
	byID := make(map[int64]collectionDTO)
	for _, collection := range response.Collections {
		byID[collection.ID] = collection
	}
	for _, hiddenID := range []int64{singleCollection, excluded, deleted} {
		if _, hidden := byID[hiddenID]; hidden {
			t.Fatalf("la colección %d no debe aparecer", hiddenID)
		}
	}
	manualItem, ok := byID[manual]
	if !ok || manualItem.ItemCount != 0 || manualItem.Source != "manual" {
		t.Fatalf("la colección manual vacía debe verse: %+v", manualItem)
	}
	autoItem, ok := byID[auto]
	if !ok || autoItem.ItemCount != 3 || autoItem.Source != "tmdb" {
		t.Fatalf("colección automática inesperada: %+v", autoItem)
	}
	if autoItem.PosterURL == nil || *autoItem.PosterURL != "https://images.example/antigua.jpg" {
		t.Fatalf("cartel de respaldo = %+v", autoItem.PosterURL)
	}
}

func TestCollectionsListPaginacionBusquedaYOrden(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	movieA := collectionHTTPMovie(t, ts, "A", 2000, "", KindMovie)
	movieB := collectionHTTPMovie(t, ts, "B", 2001, "", KindMovie)
	movieC := collectionHTTPMovie(t, ts, "C", 2002, "", KindMovie)

	alfa := collectionHTTPSeed(t, ts.db, "Saga Alfa", "manual", nil, "", 0, 0)
	beta := collectionHTTPSeed(t, ts.db, "Saga Beta", "manual", nil, "", 0, 0)
	otra := collectionHTTPSeed(t, ts.db, "Otra", "manual", nil, "", 0, 0)
	collectionHTTPMember(t, ts.db, alfa, movieA, 0, 0)
	collectionHTTPMember(t, ts.db, alfa, movieB, 1, 0)
	collectionHTTPMember(t, ts.db, beta, movieA, 0, 0)
	collectionHTTPMember(t, ts.db, otra, movieA, 0, 0)
	collectionHTTPMember(t, ts.db, otra, movieB, 1, 0)
	collectionHTTPMember(t, ts.db, otra, movieC, 2, 0)

	status, body := libraryHTTPGet(t,
		ts.URL+"/api/v1/collections?q=saga&sort=size&limit=1&offset=1", ts.token)
	if status != http.StatusOK {
		t.Fatalf("página filtrada: código = %d, cuerpo = %s", status, body)
	}
	var page collectionsResponse
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatalf("página filtrada: %v", err)
	}
	if page.Total != 2 || page.Limit != 1 || page.Offset != 1 ||
		len(page.Collections) != 1 || page.Collections[0].Name != "Saga Beta" {
		t.Fatalf("página inesperada: %+v", page)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/collections?sort=added", ts.token)
	if status != http.StatusOK {
		t.Fatalf("orden added: código = %d, cuerpo = %s", status, body)
	}
	var added collectionsResponse
	if err := json.Unmarshal(body, &added); err != nil {
		t.Fatalf("orden added: %v", err)
	}
	if len(added.Collections) != 3 || added.Collections[0].Name != "Otra" {
		t.Fatalf("orden added inesperado: %+v", added.Collections)
	}
	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/collections", ts.token)
	if status != http.StatusOK {
		t.Fatalf("orden name: código = %d, cuerpo = %s", status, body)
	}
	var byName collectionsResponse
	if err := json.Unmarshal(body, &byName); err != nil {
		t.Fatalf("orden name: %v", err)
	}
	if len(byName.Collections) != 3 || byName.Collections[0].Name != "Otra" ||
		byName.Collections[1].Name != "Saga Alfa" || byName.Collections[2].Name != "Saga Beta" {
		t.Fatalf("orden name inesperado: %+v", byName.Collections)
	}

	for _, query := range []string{"sort=otro", "limit=0", "limit=201", "offset=-1"} {
		if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/collections?"+query, ts.token); status != http.StatusBadRequest {
			t.Fatalf("%s: código = %d, cuerpo = %s", query, status, body)
		}
	}
}

func TestCollectionDetailOrdenYEstadoUsuario(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	collectionID := collectionHTTPSeed(t, ts.db, "Saga manual", "manual", nil, "", 0, 0)
	first := collectionHTTPMovie(t, ts, "Primera", 2000, "", KindMovie)
	second := collectionHTTPMovie(t, ts, "Segunda", 1999, "", KindMovie)
	third := collectionHTTPMovie(t, ts, "Tercera", 2001, "", KindMovie)
	collectionHTTPMember(t, ts.db, collectionID, first, 2, 0)
	collectionHTTPMember(t, ts.db, collectionID, second, 0, 0)
	collectionHTTPMember(t, ts.db, collectionID, third, 1, 0)

	userID := progressUserID(t, ts.db, "ana")
	setWatchState(t, ts.db, userID, second)
	setProgressRow(t, ts.db, userID, third, 120, 600, "2026-10-26T10:00:00Z")

	status, body := libraryHTTPGet(t, collectionHTTPURL(ts, collectionID), ts.token)
	if status != http.StatusOK {
		t.Fatalf("detalle: código = %d, cuerpo = %s", status, body)
	}
	var detail collectionDetailDTO
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("detalle: %v", err)
	}
	if len(detail.Items) != 3 || detail.Items[0].ID != second ||
		detail.Items[1].ID != third || detail.Items[2].ID != first {
		t.Fatalf("orden por posición inesperado: %+v", detail.Items)
	}
	if !detail.Items[0].Watched || detail.Items[1].Progress == nil {
		t.Fatalf("estado de usuario ausente: %+v", detail.Items)
	}
	if len(detail.Items[0].Collections) == 0 {
		t.Fatal("las películas del detalle deben incluir sus colecciones")
	}
}

func TestCollectionDetailOrdenPorAnioYOcultas(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	auto := collectionHTTPSeed(t, ts.db, "Saga automática", "tmdb", 700, "", 0, 0)
	newer := collectionHTTPMovie(t, ts, "Nueva", 2010, "", KindMovie)
	older := collectionHTTPMovie(t, ts, "Vieja", 1990, "", KindMovie)
	collectionHTTPMember(t, ts.db, auto, newer, nil, 0)
	collectionHTTPMember(t, ts.db, auto, older, nil, 0)

	status, body := libraryHTTPGet(t, collectionHTTPURL(ts, auto), ts.token)
	if status != http.StatusOK {
		t.Fatalf("detalle automático: código = %d, cuerpo = %s", status, body)
	}
	var detail collectionDetailDTO
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("detalle automático: %v", err)
	}
	if len(detail.Items) != 2 || detail.Items[0].ID != older || detail.Items[1].ID != newer {
		t.Fatalf("orden por año inesperado: %+v", detail.Items)
	}

	hidden := collectionHTTPSeed(t, ts.db, "Saga oculta", "tmdb", 701, "", 0, 0)
	collectionHTTPMember(t, ts.db, hidden, older, nil, 0)
	deleted := collectionHTTPSeed(t, ts.db, "Borrada", "manual", nil, "", 0, 1)
	for _, id := range []int64{hidden, deleted, 999999} {
		if status, body := libraryHTTPGet(t, collectionHTTPURL(ts, id), ts.token); status != http.StatusNotFound {
			t.Fatalf("colección %d: código = %d, cuerpo = %s", id, status, body)
		}
	}
}

func TestCollectionCreateYPatch(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)

	for name, request := range map[string]map[string]any{
		"sin nombre":     {},
		"nombre vacío":   {"name": "   "},
		"nombre largo":   {"name": strings.Repeat("a", 121)},
		"overview largo": {"name": "Saga", "overview": strings.Repeat("a", 2001)},
	} {
		t.Run(name, func(t *testing.T) {
			status, body := libraryHTTPDo(t, http.MethodPost, ts.URL+"/api/v1/collections",
				ts.token, request)
			if status != http.StatusBadRequest {
				t.Fatalf("código = %d, cuerpo = %s", status, body)
			}
		})
	}

	status, body := libraryHTTPDo(t, http.MethodPost, ts.URL+"/api/v1/collections", ts.token,
		map[string]any{"name": "  Mi saga  ", "overview": "Una sinopsis"})
	if status != http.StatusCreated {
		t.Fatalf("crear: código = %d, cuerpo = %s", status, body)
	}
	var created collectionDTO
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("crear: %v", err)
	}
	if created.Name != "Mi saga" || created.Source != "manual" || created.ItemCount != 0 ||
		created.Edited || created.Overview == nil || *created.Overview != "Una sinopsis" {
		t.Fatalf("colección creada inesperada: %+v", created)
	}

	status, body = libraryHTTPDo(t, http.MethodPatch, collectionHTTPURL(ts, created.ID), ts.token,
		map[string]any{
			"name":         "Nombre nuevo",
			"overview":     "",
			"poster_url":   "https://images.example/p.jpg",
			"backdrop_url": "https://images.example/b.jpg",
		})
	if status != http.StatusOK {
		t.Fatalf("patch: código = %d, cuerpo = %s", status, body)
	}
	var patched collectionDTO
	if err := json.Unmarshal(body, &patched); err != nil {
		t.Fatalf("patch: %v", err)
	}
	if patched.Name != "Nombre nuevo" || patched.Overview != nil ||
		patched.PosterURL == nil || *patched.PosterURL != "https://images.example/p.jpg" ||
		patched.Edited {
		t.Fatalf("colección parcheada inesperada: %+v", patched)
	}

	status, body = libraryHTTPDo(t, http.MethodPatch, collectionHTTPURL(ts, created.ID), ts.token,
		map[string]any{"poster_url": "", "backdrop_url": ""})
	if status != http.StatusOK {
		t.Fatalf("limpiar urls: código = %d, cuerpo = %s", status, body)
	}
	patched = collectionDTO{}
	if err := json.Unmarshal(body, &patched); err != nil {
		t.Fatalf("limpiar urls: %v", err)
	}
	if patched.PosterURL != nil || patched.BackdropURL != nil {
		t.Fatalf("las URLs debían quedar vacías: %+v", patched)
	}

	for name, request := range map[string]map[string]any{
		"sin campos":     {},
		"nombre vacío":   {"name": ""},
		"url inválida":   {"poster_url": "ftp://images.example/p.jpg"},
		"overview largo": {"overview": strings.Repeat("a", 2001)},
	} {
		t.Run("patch "+name, func(t *testing.T) {
			status, body := libraryHTTPDo(t, http.MethodPatch, collectionHTTPURL(ts, created.ID),
				ts.token, request)
			if status != http.StatusBadRequest {
				t.Fatalf("código = %d, cuerpo = %s", status, body)
			}
		})
	}

	tmdbID := collectionHTTPSeed(t, ts.db, "Saga TMDB", "tmdb", 800, "", 0, 0)
	status, body = libraryHTTPDo(t, http.MethodPatch, collectionHTTPURL(ts, tmdbID), ts.token,
		map[string]any{"name": "Cambiada"})
	if status != http.StatusOK {
		t.Fatalf("patch tmdb: código = %d, cuerpo = %s", status, body)
	}
	var tmdbCollection collectionDTO
	if err := json.Unmarshal(body, &tmdbCollection); err != nil {
		t.Fatalf("patch tmdb: %v", err)
	}
	if !tmdbCollection.Edited || tmdbCollection.Name != "Cambiada" {
		t.Fatalf("el patch de TMDB debe editar: %+v", tmdbCollection)
	}
}

func TestCollectionDeleteManualYTMDB(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	movie := collectionHTTPMovie(t, ts, "Pelicula", 2000, "", KindMovie)

	manual := collectionHTTPSeed(t, ts.db, "Manual", "manual", nil, "", 0, 0)
	collectionHTTPMember(t, ts.db, manual, movie, 0, 0)
	status, body := libraryHTTPDo(t, http.MethodDelete, collectionHTTPURL(ts, manual), ts.token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("borrar manual: código = %d, cuerpo = %s", status, body)
	}
	if _, _, ok := readMembership(t, ts.db, manual, movie); ok {
		t.Fatal("la pertenencia manual debía borrarse en cascada")
	}
	if status, _ := libraryHTTPGet(t, collectionHTTPURL(ts, manual), ts.token); status != http.StatusNotFound {
		t.Fatalf("la colección manual borrada debe dar 404: %d", status)
	}

	tmdb := collectionHTTPSeed(t, ts.db, "TMDB", "tmdb", 810, "", 0, 0)
	status, body = libraryHTTPDo(t, http.MethodDelete, collectionHTTPURL(ts, tmdb), ts.token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("borrar tmdb: código = %d, cuerpo = %s", status, body)
	}
	var deleted int
	if err := ts.db.QueryRow(`SELECT deleted FROM collections WHERE id = ?`, tmdb).Scan(&deleted); err != nil {
		t.Fatalf("leer deleted: %v", err)
	}
	if deleted != 1 {
		t.Fatal("la colección de TMDB debe quedar marcada como borrada")
	}
	if status, _ := libraryHTTPGet(t, collectionHTTPURL(ts, tmdb), ts.token); status != http.StatusNotFound {
		t.Fatalf("la colección de TMDB borrada debe dar 404: %d", status)
	}
	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/collections?limit=200", ts.token)
	if status != http.StatusOK {
		t.Fatalf("listado tras borrar TMDB: código = %d, cuerpo = %s", status, body)
	}
	var list collectionsResponse
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("listado tras borrar TMDB: %v", err)
	}
	for _, collection := range list.Collections {
		if collection.ID == tmdb {
			t.Fatal("la colección de TMDB borrada no debe reaparecer")
		}
	}
}

func TestCollectionItemsAddRemoveManualYTMDB(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	first := collectionHTTPMovie(t, ts, "Primera", 2000, "", KindMovie)
	second := collectionHTTPMovie(t, ts, "Segunda", 2001, "", KindMovie)
	episode := collectionHTTPMovie(t, ts, "Episodio", 2002, "", KindEpisode)

	manual := collectionHTTPSeed(t, ts.db, "Manual", "manual", nil, "", 0, 0)
	for i := 0; i < 2; i++ {
		status, body := libraryHTTPDo(t, http.MethodPost, collectionHTTPURL(ts, manual)+"/items",
			ts.token, map[string]any{"media_item_id": first})
		if status != http.StatusNoContent {
			t.Fatalf("añadir manual %d: código = %d, cuerpo = %s", i+1, status, body)
		}
	}
	position, excluded, ok := readMembership(t, ts.db, manual, first)
	if !ok || excluded != 0 || !position.Valid || position.Int64 != 0 {
		t.Fatalf("pertenencia manual inesperada: %v %v %v", position, excluded, ok)
	}
	if status, body := libraryHTTPDo(t, http.MethodPost, collectionHTTPURL(ts, manual)+"/items",
		ts.token, map[string]any{"media_item_id": second}); status != http.StatusNoContent {
		t.Fatalf("añadir segunda manual: código = %d, cuerpo = %s", status, body)
	}
	if position, _, _ := readMembership(t, ts.db, manual, second); !position.Valid || position.Int64 != 1 {
		t.Fatalf("la segunda película manual debe ir al final: %v", position)
	}

	for name, itemID := range map[string]int64{"inexistente": 999999, "episodio": episode} {
		t.Run("añadir "+name, func(t *testing.T) {
			status, body := libraryHTTPDo(t, http.MethodPost, collectionHTTPURL(ts, manual)+"/items",
				ts.token, map[string]any{"media_item_id": itemID})
			expected := http.StatusNotFound
			if name == "episodio" {
				expected = http.StatusBadRequest
			}
			if status != expected {
				t.Fatalf("código = %d, cuerpo = %s", status, body)
			}
		})
	}

	status, body := libraryHTTPDo(t, http.MethodDelete,
		collectionHTTPURL(ts, manual)+"/items/"+strconv.FormatInt(first, 10), ts.token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("quitar manual: código = %d, cuerpo = %s", status, body)
	}
	if _, _, ok := readMembership(t, ts.db, manual, first); ok {
		t.Fatal("la pertenencia manual debía borrarse")
	}
	if status, _ := libraryHTTPDo(t, http.MethodDelete,
		collectionHTTPURL(ts, manual)+"/items/"+strconv.FormatInt(first, 10), ts.token, nil); status != http.StatusNoContent {
		t.Fatalf("quitar manual idempotente: %d", status)
	}
	if status, body := libraryHTTPDo(t, http.MethodDelete,
		collectionHTTPURL(ts, manual)+"/items/999999", ts.token, nil); status != http.StatusNotFound {
		t.Fatalf("quitar inexistente: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodDelete,
		collectionHTTPURL(ts, manual)+"/items/"+strconv.FormatInt(episode, 10), ts.token, nil); status != http.StatusBadRequest {
		t.Fatalf("quitar episodio: código = %d, cuerpo = %s", status, body)
	}

	tmdb := collectionHTTPSeed(t, ts.db, "TMDB", "tmdb", 820, "", 0, 0)
	collectionHTTPMember(t, ts.db, tmdb, second, nil, 0)
	if status, body := libraryHTTPDo(t, http.MethodPost, collectionHTTPURL(ts, tmdb)+"/items",
		ts.token, map[string]any{"media_item_id": first}); status != http.StatusNoContent {
		t.Fatalf("añadir tmdb: código = %d, cuerpo = %s", status, body)
	}
	_, excluded, ok = readMembership(t, ts.db, tmdb, first)
	if !ok || excluded != 0 {
		t.Fatalf("alta TMDB inesperada: %v %v", excluded, ok)
	}
	var edited int
	if err := ts.db.QueryRow(`SELECT edited FROM collections WHERE id = ?`, tmdb).Scan(&edited); err != nil {
		t.Fatalf("leer edited: %v", err)
	}
	if edited != 1 {
		t.Fatal("añadir a una colección TMDB debe marcarla editada")
	}

	if status, body := libraryHTTPDo(t, http.MethodDelete,
		collectionHTTPURL(ts, tmdb)+"/items/"+strconv.FormatInt(first, 10), ts.token, nil); status != http.StatusNoContent {
		t.Fatalf("quitar tmdb: código = %d, cuerpo = %s", status, body)
	}
	_, excluded, ok = readMembership(t, ts.db, tmdb, first)
	if !ok || excluded != 1 {
		t.Fatalf("baja TMDB inesperada: %v %v", excluded, ok)
	}
	if status, body := libraryHTTPDo(t, http.MethodPost, collectionHTTPURL(ts, tmdb)+"/items",
		ts.token, map[string]any{"media_item_id": first}); status != http.StatusNoContent {
		t.Fatalf("reponer tmdb: código = %d, cuerpo = %s", status, body)
	}
	_, excluded, ok = readMembership(t, ts.db, tmdb, first)
	if !ok || excluded != 0 {
		t.Fatalf("reposición TMDB inesperada: %v %v", excluded, ok)
	}
}

func TestCollectionOrder(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	a := collectionHTTPMovie(t, ts, "A", 2000, "", KindMovie)
	b := collectionHTTPMovie(t, ts, "B", 2001, "", KindMovie)
	c := collectionHTTPMovie(t, ts, "C", 2002, "", KindMovie)
	foreign := collectionHTTPMovie(t, ts, "Ajena", 2003, "", KindMovie)

	manual := collectionHTTPSeed(t, ts.db, "Manual", "manual", nil, "", 0, 0)
	collectionHTTPMember(t, ts.db, manual, a, 0, 0)
	collectionHTTPMember(t, ts.db, manual, b, 1, 0)
	collectionHTTPMember(t, ts.db, manual, c, 2, 0)
	if status, body := libraryHTTPDo(t, http.MethodPut, collectionHTTPURL(ts, manual)+"/order",
		ts.token, map[string]any{"media_item_ids": []int64{c, b, a}}); status != http.StatusNoContent {
		t.Fatalf("orden manual: código = %d, cuerpo = %s", status, body)
	}
	expected := map[int64]int64{c: 0, b: 1, a: 2}
	for itemID, want := range expected {
		position, _, ok := readMembership(t, ts.db, manual, itemID)
		if !ok || !position.Valid || position.Int64 != want {
			t.Fatalf("posición de %d = %v (esperada %d)", itemID, position, want)
		}
	}

	for name, ids := range map[string][]int64{
		"faltan":  {a, b},
		"repiten": {a, b, c, c},
		"ajena":   {a, b, foreign},
	} {
		t.Run(name, func(t *testing.T) {
			status, body := libraryHTTPDo(t, http.MethodPut, collectionHTTPURL(ts, manual)+"/order",
				ts.token, map[string]any{"media_item_ids": ids})
			if status != http.StatusBadRequest {
				t.Fatalf("código = %d, cuerpo = %s", status, body)
			}
		})
	}

	tmdb := collectionHTTPSeed(t, ts.db, "TMDB", "tmdb", 830, "", 0, 0)
	collectionHTTPMember(t, ts.db, tmdb, a, nil, 0)
	collectionHTTPMember(t, ts.db, tmdb, b, nil, 0)
	if status, body := libraryHTTPDo(t, http.MethodPut, collectionHTTPURL(ts, tmdb)+"/order",
		ts.token, map[string]any{"media_item_ids": []int64{b, a}}); status != http.StatusNoContent {
		t.Fatalf("orden tmdb: código = %d, cuerpo = %s", status, body)
	}
	if position, _, _ := readMembership(t, ts.db, tmdb, b); !position.Valid || position.Int64 != 0 {
		t.Fatalf("orden TMDB no aplicado: %v", position)
	}
	var edited int
	if err := ts.db.QueryRow(`SELECT edited FROM collections WHERE id = ?`, tmdb).Scan(&edited); err != nil {
		t.Fatalf("leer edited: %v", err)
	}
	if edited != 1 {
		t.Fatal("ordenar una colección TMDB debe marcarla editada")
	}
}

func TestCollectionsGestionPermisos(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	movie := collectionHTTPMovie(t, ts, "Pelicula", 2000, "", KindMovie)
	collectionID := collectionHTTPSeed(t, ts.db, "Saga", "manual", nil, "", 0, 0)
	collectionHTTPMember(t, ts.db, collectionID, movie, 0, 0)
	userToken := nonAdminToken(t, ts)

	requests := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"crear", http.MethodPost, "/api/v1/collections", map[string]any{"name": "Nueva"}},
		{"patch", http.MethodPatch, "/api/v1/collections/" + strconv.FormatInt(collectionID, 10), map[string]any{"name": "Otra"}},
		{"delete", http.MethodDelete, "/api/v1/collections/" + strconv.FormatInt(collectionID, 10), nil},
		{"añadir", http.MethodPost, "/api/v1/collections/" + strconv.FormatInt(collectionID, 10) + "/items", map[string]any{"media_item_id": movie}},
		{"quitar", http.MethodDelete, "/api/v1/collections/" + strconv.FormatInt(collectionID, 10) + "/items/" + strconv.FormatInt(movie, 10), nil},
		{"orden", http.MethodPut, "/api/v1/collections/" + strconv.FormatInt(collectionID, 10) + "/order", map[string]any{"media_item_ids": []int64{movie}}},
	}
	for _, request := range requests {
		t.Run(request.name+" sin admin", func(t *testing.T) {
			status, body := libraryHTTPDo(t, request.method, ts.URL+request.path, userToken, request.body)
			if status != http.StatusForbidden {
				t.Fatalf("código = %d, cuerpo = %s", status, body)
			}
		})
		t.Run(request.name+" sin sesión", func(t *testing.T) {
			status, body := libraryHTTPDo(t, request.method, ts.URL+request.path, "", request.body)
			if status != http.StatusUnauthorized {
				t.Fatalf("código = %d, cuerpo = %s", status, body)
			}
		})
	}
}

func TestItemCollectionsYFiltro(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	a := collectionHTTPMovie(t, ts, "A", 2000, "", KindMovie)
	b := collectionHTTPMovie(t, ts, "B", 2001, "", KindMovie)
	c := collectionHTTPMovie(t, ts, "C", 2002, "", KindMovie)
	d := collectionHTTPMovie(t, ts, "D", 2003, "", KindMovie)

	manual := collectionHTTPSeed(t, ts.db, "Manual", "manual", nil, "", 0, 0)
	auto := collectionHTTPSeed(t, ts.db, "Automática", "tmdb", 900, "", 0, 0)
	hidden := collectionHTTPSeed(t, ts.db, "Oculta", "tmdb", 901, "", 0, 0)
	collectionHTTPMember(t, ts.db, manual, a, 0, 0)
	collectionHTTPMember(t, ts.db, auto, a, nil, 0)
	collectionHTTPMember(t, ts.db, auto, b, nil, 0)
	collectionHTTPMember(t, ts.db, hidden, c, nil, 0)

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(a, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("ítem A: código = %d, cuerpo = %s", status, body)
	}
	var item mediaItemDTO
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("ítem A: %v", err)
	}
	if item.Collections == nil || len(item.Collections) != 2 {
		t.Fatalf("colecciones del ítem A: %+v", item.Collections)
	}
	foundCollections := map[int64]bool{}
	for _, collection := range item.Collections {
		foundCollections[collection.ID] = true
	}
	if !foundCollections[manual] || !foundCollections[auto] {
		t.Fatalf("faltan colecciones en el ítem A: %+v", item.Collections)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(c, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("ítem C: código = %d", status)
	}
	var hiddenItem mediaItemDTO
	if err := json.Unmarshal(body, &hiddenItem); err != nil {
		t.Fatalf("ítem C: %v", err)
	}
	if len(hiddenItem.Collections) != 0 {
		t.Fatalf("la colección automática oculta no debe aparecer: %+v", hiddenItem.Collections)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(d, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("ítem D: código = %d", status)
	}
	item = mediaItemDTO{}
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("ítem D: %v", err)
	}
	if item.Collections == nil || len(item.Collections) != 0 {
		t.Fatalf("las colecciones vacías deben ser []: %#v", item.Collections)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items?collection_id="+strconv.FormatInt(auto, 10)+"&limit=200", ts.token)
	if status != http.StatusOK {
		t.Fatalf("filtro auto: código = %d, cuerpo = %s", status, body)
	}
	var list itemsResponse
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("filtro auto: %v", err)
	}
	if list.Total != 2 || len(list.Items) != 2 || list.Items[0].ID != a || list.Items[1].ID != b {
		t.Fatalf("filtro auto inesperado: %+v", list)
	}
	if len(list.Items[0].Collections) == 0 {
		t.Fatal("la lista debe incluir las colecciones de cada ítem")
	}
	if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items?collection_id=no", ts.token); status != http.StatusBadRequest {
		t.Fatalf("filtro inválido: código = %d, cuerpo = %s", status, body)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items?collection=any&limit=200", ts.token)
	if status != http.StatusOK {
		t.Fatalf("filtro any: código = %d, cuerpo = %s", status, body)
	}
	var anyList itemsResponse
	if err := json.Unmarshal(body, &anyList); err != nil {
		t.Fatalf("filtro any: %v", err)
	}
	if anyList.Total != 2 || len(anyList.Items) != 2 ||
		anyList.Items[0].ID != a || anyList.Items[1].ID != b {
		t.Fatalf("filtro any inesperado: %+v", anyList)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items?collection=none&limit=200", ts.token)
	if status != http.StatusOK {
		t.Fatalf("filtro none: código = %d, cuerpo = %s", status, body)
	}
	var noneList itemsResponse
	if err := json.Unmarshal(body, &noneList); err != nil {
		t.Fatalf("filtro none: %v", err)
	}
	if noneList.Total != 1 || len(noneList.Items) != 1 || noneList.Items[0].ID != d {
		t.Fatalf("filtro none inesperado: %+v", noneList)
	}

	if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items?collection=otra", ts.token); status != http.StatusBadRequest {
		t.Fatalf("filtro de colección inválido: código = %d, cuerpo = %s", status, body)
	}

	episode := collectionHTTPMovie(t, ts, "Episodio", 2004, "", KindEpisode)
	for _, value := range []string{"any", "none"} {
		status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items?kind=episode&collection="+value, ts.token)
		if status != http.StatusOK {
			t.Fatalf("filtro %s en episodios: código = %d, cuerpo = %s", value, status, body)
		}
		var episodes itemsResponse
		if err := json.Unmarshal(body, &episodes); err != nil {
			t.Fatalf("filtro %s en episodios: %v", value, err)
		}
		if episodes.Total != 1 || len(episodes.Items) != 1 || episodes.Items[0].ID != episode {
			t.Fatalf("el filtro no debe afectar a episodios (%s): %+v", value, episodes)
		}
	}
}

func TestCollectionsListAllAdmin(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	movie := collectionHTTPMovie(t, ts, "Única", 2000, "", KindMovie)
	hidden := collectionHTTPSeed(t, ts.db, "Alfa oculta", "tmdb", 910, "", 0, 0)
	manual := collectionHTTPSeed(t, ts.db, "Zeta manual", "manual", nil, "", 0, 0)
	deleted := collectionHTTPSeed(t, ts.db, "Borrada", "manual", nil, "", 0, 1)
	collectionHTTPMember(t, ts.db, hidden, movie, nil, 0)

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/collections?all=1", ts.token)
	if status != http.StatusOK {
		t.Fatalf("listar todas: código = %d, cuerpo = %s", status, body)
	}
	var response collectionsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("listar todas: %v", err)
	}
	if response.Total != 2 || len(response.Collections) != 2 {
		t.Fatalf("colecciones inesperadas: %+v", response)
	}
	if response.Collections[0].ID != hidden || response.Collections[0].ItemCount != 1 ||
		response.Collections[0].Source != "tmdb" {
		t.Fatalf("primera colección inesperada: %+v", response.Collections[0])
	}
	if response.Collections[1].ID != manual || response.Collections[1].ItemCount != 0 ||
		response.Collections[1].Source != "manual" {
		t.Fatalf("segunda colección inesperada: %+v", response.Collections[1])
	}
	for _, collection := range response.Collections {
		if collection.ID == deleted {
			t.Fatal("las colecciones borradas no deben aparecer")
		}
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/collections?all=1", nonAdminToken(t, ts))
	if status != http.StatusForbidden {
		t.Fatalf("all=1 sin admin: código = %d, cuerpo = %s", status, body)
	}
}
