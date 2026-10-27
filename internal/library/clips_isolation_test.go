package library

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
)

func TestClipsAisladosDePeliculasYIdentificacionBloqueada(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	videoRoot := t.TempDir()
	clipRoot := t.TempDir()
	videoLibrary := createLibrary(t, ts.db, videoRoot)
	clipLibrary := createClipLibrary(t, ts.db, clipRoot)

	video := insertItemFixture(t, ts.db, itemFixture{
		libraryID: videoLibrary, path: filepath.Join(videoRoot, "Pelicula.mkv"),
		title: "Pelicula", year: 2000, genres: `["Drama"]`, matchStatus: "auto",
		metadataSource: "tmdb",
	})
	candidate := insertItemFixture(t, ts.db, itemFixture{
		libraryID: videoLibrary, path: filepath.Join(videoRoot, "Candidata.mkv"),
		title: "Candidata", year: 2001, genres: `["Drama"]`,
		posterURL: "https://images.example/candidate.jpg", matchStatus: "auto",
		metadataSource: "tmdb",
	})
	clip := insertItemFixture(t, ts.db, itemFixture{
		libraryID: clipLibrary, path: filepath.Join(clipRoot, "Clip.mp4"),
		title: "Clip", genres: `["Drama"]`,
		posterURL: "https://images.example/clip.jpg", matchStatus: "none",
	})
	anaID := progressUserID(t, ts.db, "ana")
	setProgressRow(t, ts.db, anaID, clip, 30, 100, "2026-10-27T10:00:00Z")

	get := func(path string) (int, []byte) {
		t.Helper()
		return libraryHTTPGet(t, ts.URL+path, ts.token)
	}

	status, body := get("/api/v1/libraries")
	if status != http.StatusOK {
		t.Fatalf("bibliotecas: código = %d, cuerpo = %s", status, body)
	}
	var libraries librariesResponse
	if err := json.Unmarshal(body, &libraries); err != nil {
		t.Fatalf("bibliotecas: %v", err)
	}
	for _, library := range libraries.Libraries {
		switch library.ID {
		case clipLibrary:
			if library.ItemCount != 1 || library.Kind != "clips" {
				t.Fatalf("recuento de la biblioteca de clips: %+v", library)
			}
		case videoLibrary:
			if library.ItemCount != 2 || library.Kind != "video" {
				t.Fatalf("recuento de la biblioteca de vídeo: %+v", library)
			}
		}
	}

	status, body = get("/api/v1/items?kind=movie&limit=200")
	if status != http.StatusOK {
		t.Fatalf("listado: código = %d, cuerpo = %s", status, body)
	}
	var items itemsResponse
	if err := json.Unmarshal(body, &items); err != nil {
		t.Fatalf("listado: %v", err)
	}
	if items.Total != 2 || len(items.Items) != 2 {
		t.Fatalf("el listado solo debe tener películas de vídeo: %+v", items)
	}
	for _, item := range items.Items {
		if item.ID == clip || item.LibraryKind != "video" {
			t.Fatalf("clip colado en el listado: %+v", item)
		}
	}

	status, body = get("/api/v1/items?q=Clip&limit=200")
	if status != http.StatusOK {
		t.Fatalf("búsqueda: código = %d, cuerpo = %s", status, body)
	}
	if err := json.Unmarshal(body, &items); err != nil || items.Total != 0 {
		t.Fatalf("el clip aparece en la búsqueda: %s (%v)", body, err)
	}

	status, body = get("/api/v1/items?unmatched=true&limit=200")
	if status != http.StatusOK {
		t.Fatalf("sin identificar: código = %d, cuerpo = %s", status, body)
	}
	if err := json.Unmarshal(body, &items); err != nil || items.Total != 0 {
		t.Fatalf("el clip aparece como película sin identificar: %s (%v)", body, err)
	}

	status, body = get("/api/v1/genres?kind=movie")
	if status != http.StatusOK {
		t.Fatalf("géneros: código = %d, cuerpo = %s", status, body)
	}
	var genres genresResponse
	if err := json.Unmarshal(body, &genres); err != nil || len(genres.Genres) != 1 ||
		genres.Genres[0].Count != 2 {
		t.Fatalf("los géneros incluyen clips: %s (%v)", body, err)
	}

	status, body = get("/api/v1/items/" + strconv.FormatInt(video, 10) + "/similar")
	if status != http.StatusOK {
		t.Fatalf("similares: código = %d, cuerpo = %s", status, body)
	}
	var similar similarItemsResponse
	if err := json.Unmarshal(body, &similar); err != nil {
		t.Fatalf("similares: %v", err)
	}
	if len(similar.Items) != 1 || similar.Items[0].ID != candidate {
		t.Fatalf("similares con clips: %+v", similar.Items)
	}

	status, body = get("/api/v1/items/" + strconv.FormatInt(clip, 10))
	if status != http.StatusOK {
		t.Fatalf("ficha del clip: código = %d, cuerpo = %s", status, body)
	}
	var clipItem mediaItemDTO
	if err := json.Unmarshal(body, &clipItem); err != nil {
		t.Fatalf("ficha del clip: %v", err)
	}
	if clipItem.LibraryKind != "clips" || clipItem.Progress == nil {
		t.Fatalf("ficha del clip inesperada: %+v", clipItem)
	}
	status, body = get("/api/v1/items/" + strconv.FormatInt(clip, 10) + "/similar")
	if status != http.StatusOK {
		t.Fatalf("similares del clip: código = %d, cuerpo = %s", status, body)
	}
	var clipSimilar similarItemsResponse
	if err := json.Unmarshal(body, &clipSimilar); err != nil || len(clipSimilar.Items) != 0 {
		t.Fatalf("similares del clip inesperados: %s (%v)", body, err)
	}

	for _, request := range []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodGet, "/candidates", nil},
		{http.MethodPut, "/match", map[string]any{"tmdb_id": 42}},
		{http.MethodDelete, "/match", nil},
		{http.MethodGet, "/images", nil},
		{http.MethodPut, "/images/poster", map[string]any{"url": "https://images.example/x.jpg"}},
	} {
		status, body := libraryHTTPDo(t, request.method,
			ts.URL+"/api/v1/items/"+strconv.FormatInt(clip, 10)+request.path,
			ts.token, request.body)
		if status != http.StatusBadRequest {
			t.Fatalf("%s %s: código = %d, cuerpo = %s", request.method, request.path, status, body)
		}
		var response map[string]string
		if err := json.Unmarshal(body, &response); err != nil ||
			response["error"] != "Los vídeos no se identifican" {
			t.Fatalf("%s %s: error inesperado: %s", request.method, request.path, body)
		}
	}

	result, err := ts.db.Exec(`INSERT INTO collections (name, source) VALUES ('Saga', 'manual')`)
	if err != nil {
		t.Fatalf("crear colección: %v", err)
	}
	collectionID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de colección: %v", err)
	}
	if _, err := ts.db.Exec(`
		INSERT INTO collection_items (collection_id, media_item_id, position)
		VALUES (?, ?, 0), (?, ?, 1)`, collectionID, video, collectionID, clip); err != nil {
		t.Fatalf("membresías de colección: %v", err)
	}

	status, body = get("/api/v1/collections/" + strconv.FormatInt(collectionID, 10))
	if status != http.StatusOK {
		t.Fatalf("colección: código = %d, cuerpo = %s", status, body)
	}
	var collection collectionDetailDTO
	if err := json.Unmarshal(body, &collection); err != nil {
		t.Fatalf("colección: %v", err)
	}
	if collection.ItemCount != 1 || len(collection.Items) != 1 || collection.Items[0].ID != video {
		t.Fatalf("la colección incluye el clip: %+v", collection)
	}
	if collection.PosterURL != nil {
		t.Fatalf("el cartel de la colección no debe salir de un clip: %+v", collection.PosterURL)
	}

	status, body = get("/api/v1/collections?limit=200")
	if status != http.StatusOK {
		t.Fatalf("colecciones: código = %d, cuerpo = %s", status, body)
	}
	var list collectionsResponse
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("colecciones: %v", err)
	}
	for _, item := range list.Collections {
		if item.ID == collectionID && item.ItemCount != 1 {
			t.Fatalf("recuento de colección con clip: %+v", item)
		}
	}
	status, body = libraryHTTPDo(t, http.MethodPost,
		ts.URL+"/api/v1/collections/"+strconv.FormatInt(collectionID, 10)+"/items",
		ts.token, map[string]any{"media_item_id": clip})
	if status != http.StatusBadRequest {
		t.Fatalf("añadir clip a colección: código = %d, cuerpo = %s", status, body)
	}
	var failure map[string]string
	if err := json.Unmarshal(body, &failure); err != nil ||
		failure["error"] != "Los vídeos no se identifican" {
		t.Fatalf("error al añadir clip: %s", body)
	}

	status, body = get("/api/v1/home")
	if status != http.StatusOK {
		t.Fatalf("home: código = %d, cuerpo = %s", status, body)
	}
	var home homeResponse
	if err := json.Unmarshal(body, &home); err != nil {
		t.Fatalf("home: %v", err)
	}
	if !home.HasClips {
		t.Fatal("home debe indicar que hay clips")
	}
	if home.Counts.Movies != 2 {
		t.Fatalf("el recuento de películas incluye clips: %+v", home.Counts)
	}
	for _, item := range home.RecentMovies {
		if item.ID == clip {
			t.Fatalf("clip en películas recientes: %+v", home.RecentMovies)
		}
	}
	if len(home.Continue) != 1 || home.Continue[0].ID != clip {
		t.Fatalf("continuar viendo debe incluir el clip: %+v", home.Continue)
	}
}
