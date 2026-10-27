package library

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
)

func listURL(ts *libraryTestServer, list string) string {
	return ts.URL + "/api/v1/lists/" + list
}

func itemListURL(ts *libraryTestServer, itemID int64, list string) string {
	return ts.URL + "/api/v1/items/" + strconv.FormatInt(itemID, 10) + "/lists/" + list
}

func seriesListURL(ts *libraryTestServer, seriesID int64, list string) string {
	return ts.URL + "/api/v1/series/" + strconv.FormatInt(seriesID, 10) + "/lists/" + list
}

func collectionListURL(ts *libraryTestServer, collectionID int64, list string) string {
	return ts.URL + "/api/v1/collections/" + strconv.FormatInt(collectionID, 10) + "/lists/" + list
}

func TestListsMarcarDesmarcarIdempotenciaYAislamiento(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	itemID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      "/media/lista/pelicula.mkv",
		title:     "Película",
	})
	bea := nonAdminToken(t, ts)

	for i := 0; i < 2; i++ {
		if status, body := libraryHTTPDo(t, http.MethodPut, itemListURL(ts, itemID, "favorite"), ts.token, nil); status != http.StatusNoContent {
			t.Fatalf("marcar favorito %d: %d %s", i, status, body)
		}
	}
	var count int
	if err := ts.db.QueryRow(`SELECT count(*) FROM item_lists WHERE user_id = ? AND media_item_id = ?`,
		progressUserID(t, ts.db, "ana"), itemID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("marca duplicada: %d, %v", count, err)
	}
	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("leer ítem marcado: %d %s", status, body)
	}
	var item mediaItemDTO
	if err := json.Unmarshal(body, &item); err != nil || !item.Favorite || item.Watchlist {
		t.Fatalf("indicadores del ítem: %+v, %v", item, err)
	}
	if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10), bea); status != http.StatusOK {
		t.Fatalf("leer ítem como bea: %d %s", status, body)
	} else {
		var other mediaItemDTO
		if err := json.Unmarshal(body, &other); err != nil || other.Favorite || other.Watchlist {
			t.Fatalf("bea no debe ver las marcas de ana: %+v, %v", other, err)
		}
	}
	status, body = libraryHTTPGet(t,
		ts.URL+"/api/v1/items?library_id="+strconv.FormatInt(libraryID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("listar ítems: %d %s", status, body)
	}
	var items itemsResponse
	if err := json.Unmarshal(body, &items); err != nil || len(items.Items) != 1 ||
		!items.Items[0].Favorite || items.Items[0].Watchlist {
		t.Fatalf("indicadores en el listado: %+v, %v", items, err)
	}

	status, body = libraryHTTPGet(t, listURL(ts, "favorite")+"?kind=items", ts.token)
	if status != http.StatusOK {
		t.Fatalf("listar favoritos: %d %s", status, body)
	}
	var listed listsResponse
	if err := json.Unmarshal(body, &listed); err != nil || listed.Total == nil || *listed.Total != 1 || len(listed.Items) != 1 || listed.Items[0].ID != itemID {
		t.Fatalf("favoritos inesperados: %+v, %v", listed, err)
	}
	if status, body := libraryHTTPGet(t, listURL(ts, "favorite")+"?kind=items", bea); status != http.StatusOK {
		t.Fatalf("favoritos de bea: %d %s", status, body)
	} else if err := json.Unmarshal(body, &listed); err != nil || len(listed.Items) != 0 {
		t.Fatalf("bea no debe ver favoritos de ana: %+v, %v", listed, err)
	}

	for i := 0; i < 2; i++ {
		if status, body := libraryHTTPDo(t, http.MethodDelete, itemListURL(ts, itemID, "favorite"), ts.token, nil); status != http.StatusNoContent {
			t.Fatalf("desmarcar %d: %d %s", i, status, body)
		}
	}
	if err := ts.db.QueryRow(`SELECT count(*) FROM item_lists WHERE media_item_id = ?`, itemID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("desmarcar no idempotente: %d, %v", count, err)
	}
}

func TestListsSeriesColeccionesIndicadoresYErrores(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	seriesID := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Serie", title: "Serie", displayTitle: "Serie",
	})
	collectionID := collectionHTTPSeed(t, ts.db, "Colección", "manual", nil, "", 0, 0)

	for _, url := range []string{
		seriesListURL(ts, seriesID, "favorite"),
		collectionListURL(ts, collectionID, "watchlist"),
	} {
		if status, body := libraryHTTPDo(t, http.MethodPut, url, ts.token, nil); status != http.StatusNoContent {
			t.Fatalf("marcar %s: %d %s", url, status, body)
		}
	}

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/series", ts.token)
	if status != http.StatusOK {
		t.Fatalf("listar series: %d %s", status, body)
	}
	var series seriesResponse
	if err := json.Unmarshal(body, &series); err != nil || len(series.Series) != 1 || !series.Series[0].Favorite || series.Series[0].Watchlist {
		t.Fatalf("indicadores de serie en listado: %+v, %v", series, err)
	}
	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/series/"+strconv.FormatInt(seriesID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("detalle de serie: %d %s", status, body)
	}
	var detail seriesDetailDTO
	if err := json.Unmarshal(body, &detail); err != nil || !detail.Favorite || detail.Watchlist {
		t.Fatalf("indicadores de serie en detalle: %+v, %v", detail, err)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/collections", ts.token)
	if status != http.StatusOK {
		t.Fatalf("listar colecciones: %d %s", status, body)
	}
	var collections collectionsResponse
	if err := json.Unmarshal(body, &collections); err != nil || len(collections.Collections) != 1 || collections.Collections[0].Favorite || !collections.Collections[0].Watchlist {
		t.Fatalf("indicadores de colección en listado: %+v, %v", collections, err)
	}
	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/collections/"+strconv.FormatInt(collectionID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("detalle de colección: %d %s", status, body)
	}
	var collection collectionDetailDTO
	if err := json.Unmarshal(body, &collection); err != nil || collection.Favorite || !collection.Watchlist {
		t.Fatalf("indicadores de colección en detalle: %+v, %v", collection, err)
	}

	for _, test := range []struct {
		name string
		url  string
	}{
		{"lista inválida", itemListURL(ts, 1, "otra")},
		{"ítem inexistente", itemListURL(ts, 99999, "favorite")},
		{"serie inexistente", seriesListURL(ts, 99999, "favorite")},
		{"colección inexistente", collectionListURL(ts, 99999, "favorite")},
	} {
		t.Run(test.name, func(t *testing.T) {
			if status, body := libraryHTTPDo(t, http.MethodPut, test.url, ts.token, nil); status != http.StatusNotFound {
				t.Fatalf("código = %d, cuerpo = %s", status, body)
			}
		})
	}
}

func TestListsPaginacionOrdenYBloques(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	userID := progressUserID(t, ts.db, "ana")
	ids := make([]int64, 0, 3)
	for i := 0; i < 3; i++ {
		id := insertItemFixture(t, ts.db, itemFixture{
			libraryID: libraryID,
			path:      fmt.Sprintf("/media/lista/%d.mkv", i),
			title:     fmt.Sprintf("Item %d", i),
		})
		ids = append(ids, id)
		if status, body := libraryHTTPDo(t, http.MethodPut, itemListURL(ts, id, "favorite"), ts.token, nil); status != http.StatusNoContent {
			t.Fatalf("marcar %d: %d %s", i, status, body)
		}
		if _, err := ts.db.Exec(`UPDATE item_lists SET added_at = ? WHERE user_id = ? AND media_item_id = ? AND list = 'favorite'`,
			fmt.Sprintf("2026-10-26T10:0%d:00Z", i), userID, id); err != nil {
			t.Fatalf("ordenar marca: %v", err)
		}
	}
	// Una serie y una colección para comprobar que el bloque sin kind devuelve
	// los tres tipos y sus totales.
	seriesID := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Serie", title: "Serie", displayTitle: "Serie",
	})
	collectionID := collectionHTTPSeed(t, ts.db, "Colección", "manual", nil, "", 0, 0)
	if status, body := libraryHTTPDo(t, http.MethodPut, seriesListURL(ts, seriesID, "favorite"), ts.token, nil); status != http.StatusNoContent {
		t.Fatalf("marcar serie: %d %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodPut, collectionListURL(ts, collectionID, "favorite"), ts.token, nil); status != http.StatusNoContent {
		t.Fatalf("marcar colección: %d %s", status, body)
	}

	status, body := libraryHTTPGet(t, listURL(ts, "favorite")+"?kind=items&limit=2&offset=0", ts.token)
	if status != http.StatusOK {
		t.Fatalf("página de ítems: %d %s", status, body)
	}
	var page listsResponse
	if err := json.Unmarshal(body, &page); err != nil || page.Total == nil || *page.Total != 3 || len(page.Items) != 2 {
		t.Fatalf("página inesperada: %+v, %v", page, err)
	}
	if page.Items[0].ID != ids[2] || page.Items[1].ID != ids[1] {
		t.Fatalf("orden inesperado: %+v", page.Items)
	}

	status, body = libraryHTTPGet(t, listURL(ts, "favorite"), ts.token)
	if status != http.StatusOK {
		t.Fatalf("bloques: %d %s", status, body)
	}
	page = listsResponse{}
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatalf("bloques JSON: %v", err)
	}
	if page.Totals.Items != 3 || page.Totals.Series != 1 || page.Totals.Collections != 1 ||
		len(page.Items) != 3 || len(page.Series) != 1 || len(page.Collections) != 1 || page.Total != nil {
		t.Fatalf("bloques inesperados: %+v", page)
	}
}

func TestHomeListasYQuitarWatchlistAlVer(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	itemID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/lista/pelicula.mkv", title: "Película",
	})
	seriesID := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Serie", title: "Serie", displayTitle: "Serie",
	})
	episodeOne := insertSeriesEpisodeHTTP(t, ts, libraryID, seriesID, "Episodio 1", "Episodio 1", 1, 1, "", "", "")
	episodeTwo := insertSeriesEpisodeHTTP(t, ts, libraryID, seriesID, "Episodio 2", "Episodio 2", 1, 2, "", "", "")

	for _, url := range []string{
		itemListURL(ts, itemID, "watchlist"),
		itemListURL(ts, itemID, "favorite"),
		seriesListURL(ts, seriesID, "watchlist"),
		seriesListURL(ts, seriesID, "favorite"),
	} {
		if status, body := libraryHTTPDo(t, http.MethodPut, url, ts.token, nil); status != http.StatusNoContent {
			t.Fatalf("marcar %s: %d %s", url, status, body)
		}
	}

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/home", ts.token)
	if status != http.StatusOK {
		t.Fatalf("home: %d %s", status, body)
	}
	var home homeResponse
	if err := json.Unmarshal(body, &home); err != nil {
		t.Fatalf("home JSON: %v", err)
	}
	if len(home.Watchlist) != 2 || len(home.Favorites) != 2 {
		t.Fatalf("listas del portal: %+v", home)
	}
	kinds := map[string]bool{}
	for _, entry := range home.Watchlist {
		kinds[entry.Kind] = true
	}
	if !kinds["item"] || !kinds["series"] {
		t.Fatalf("el portal debe mezclar ítems y series: %+v", home.Watchlist)
	}

	if status, body := libraryHTTPDo(t, http.MethodPut,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10)+"/watched", ts.token, nil); status != http.StatusNoContent {
		t.Fatalf("marcar ítem visto: %d %s", status, body)
	}
	var count int
	userID := progressUserID(t, ts.db, "ana")
	if err := ts.db.QueryRow(`SELECT count(*) FROM item_lists WHERE user_id = ? AND media_item_id = ? AND list = 'watchlist'`,
		userID, itemID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("watchlist del ítem visto: %d, %v", count, err)
	}
	if err := ts.db.QueryRow(`SELECT count(*) FROM item_lists WHERE user_id = ? AND media_item_id = ? AND list = 'favorite'`,
		userID, itemID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("el favorito debe conservarse: %d, %v", count, err)
	}

	if status, body := libraryHTTPDo(t, http.MethodPut,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(episodeOne, 10)+"/watched", ts.token, nil); status != http.StatusNoContent {
		t.Fatalf("marcar primer episodio: %d %s", status, body)
	}
	if err := ts.db.QueryRow(`SELECT count(*) FROM series_lists WHERE user_id = ? AND series_id = ? AND list = 'watchlist'`,
		userID, seriesID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("la serie parcial no debe quitarse: %d, %v", count, err)
	}
	if status, body := libraryHTTPDo(t, http.MethodPut,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(episodeTwo, 10)+"/watched", ts.token, nil); status != http.StatusNoContent {
		t.Fatalf("marcar segundo episodio: %d %s", status, body)
	}
	if err := ts.db.QueryRow(`SELECT count(*) FROM series_lists WHERE user_id = ? AND series_id = ? AND list = 'watchlist'`,
		userID, seriesID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("la serie completa debe quitarse de próximas: %d, %v", count, err)
	}
}
