package library

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/catlinux/cinecat/internal/metadata"
)

func insertImageTestItem(t *testing.T, ts *libraryTestServer, tmdbID int, kind string) int64 {
	t.Helper()
	libraryID := createLibrary(t, ts.db, t.TempDir())
	return insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      filepath.Join(t.TempDir(), "Pelicula.mkv"),
		title:     "Pelicula",
		kind:      kind,
		tmdbID:    tmdbID,
	})
}

func imageTestURL(ts *libraryTestServer, id int64, suffix string) string {
	return ts.URL + "/api/v1/items/" + strconv.FormatInt(id, 10) + suffix
}

func TestItemImagesGet(t *testing.T) {
	fake := &fakeTMDB{images: &metadata.ImageOptions{
		Posters: []metadata.ImageOption{{
			URL: "https://images.example/w342/poster.jpg", FullURL: "https://images.example/w780/poster.jpg",
			Language: "es", VoteAverage: 7, Width: 342, Height: 513,
		}},
		Backdrops: []metadata.ImageOption{{
			URL: "https://images.example/w780/backdrop.jpg", FullURL: "https://images.example/w1280/backdrop.jpg",
			Language: "en", Width: 1280, Height: 720,
		}},
	}}
	ts := newLibraryHTTPTestServer(t, fake)
	id := insertImageTestItem(t, ts, 603, KindMovie)

	status, body := libraryHTTPGet(t, imageTestURL(ts, id, "/images"), ts.token)
	if status != http.StatusOK {
		t.Fatalf("imágenes: código = %d, cuerpo = %s", status, body)
	}
	var options metadata.ImageOptions
	if err := json.Unmarshal(body, &options); err != nil {
		t.Fatalf("respuesta de imágenes: %v", err)
	}
	if len(options.Posters) != 1 || len(options.Backdrops) != 1 ||
		options.Posters[0].Language != "es" || options.Backdrops[0].Width != 1280 {
		t.Fatalf("imágenes inesperadas: %+v", options)
	}
	if fake.imagesCalls != 1 || fake.lastImagesID != 603 {
		t.Fatalf("MovieImages = %d llamadas, id %d", fake.imagesCalls, fake.lastImagesID)
	}

	unidentified := insertImageTestItem(t, ts, 0, KindMovie)
	if status, body := libraryHTTPGet(t, imageTestURL(ts, unidentified, "/images"), ts.token); status != http.StatusConflict ||
		!strings.Contains(string(body), `"la película no está identificada"`) {
		t.Fatalf("sin tmdb_id: código = %d, cuerpo = %s", status, body)
	}

	noClient := newLibraryHTTPTestServer(t)
	identified := insertImageTestItem(t, noClient, 603, KindMovie)
	if status, body := libraryHTTPGet(t, imageTestURL(noClient, identified, "/images"), noClient.token); status != http.StatusServiceUnavailable {
		t.Fatalf("sin cliente TMDB: código = %d, cuerpo = %s", status, body)
	}

	fake.imagesErr = errors.New("red caída")
	if status, body := libraryHTTPGet(t, imageTestURL(ts, id, "/images"), ts.token); status != http.StatusBadGateway {
		t.Fatalf("error de TMDB: código = %d, cuerpo = %s", status, body)
	}
}

func TestItemImagesGetPermisosYTipo(t *testing.T) {
	fake := &fakeTMDB{}
	ts := newLibraryHTTPTestServer(t, fake)
	id := insertImageTestItem(t, ts, 603, KindMovie)
	episode := insertImageTestItem(t, ts, 0, KindEpisode)

	userToken := nonAdminToken(t, ts)
	if status, body := libraryHTTPGet(t, imageTestURL(ts, id, "/images"), userToken); status != http.StatusForbidden {
		t.Fatalf("sin admin: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPGet(t, imageTestURL(ts, episode, "/images"), ts.token); status != http.StatusNotFound {
		t.Fatalf("episodio: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPGet(t, imageTestURL(ts, 999999, "/images"), ts.token); status != http.StatusNotFound {
		t.Fatalf("ítem inexistente: código = %d, cuerpo = %s", status, body)
	}
}

func TestItemImagePut(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	id := insertImageTestItem(t, ts, 0, KindMovie)

	status, body := libraryHTTPDo(t, http.MethodPut, imageTestURL(ts, id, "/images/poster"), ts.token,
		map[string]string{"url": "https://images.example/propio.jpg"})
	if status != http.StatusOK {
		t.Fatalf("fijar póster: código = %d, cuerpo = %s", status, body)
	}
	var item mediaItemDTO
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("respuesta: %v", err)
	}
	if item.PosterURL == nil || *item.PosterURL != "https://images.example/propio.jpg" || !item.PosterCustom {
		t.Fatalf("póster fijado inesperado: %+v", item)
	}

	status, body = libraryHTTPDo(t, http.MethodPut, imageTestURL(ts, id, "/images/backdrop"), ts.token,
		map[string]string{"url": "https://images.example/fondo.jpg?x=1"})
	if status != http.StatusOK {
		t.Fatalf("fijar fondo: código = %d, cuerpo = %s", status, body)
	}
	item = mediaItemDTO{}
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("respuesta del fondo: %v", err)
	}
	if item.BackdropURL == nil || *item.BackdropURL != "https://images.example/fondo.jpg?x=1" ||
		!item.BackdropCustom {
		t.Fatalf("fondo fijado inesperado: %+v", item)
	}
}

func TestItemImagePutValidaURL(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	id := insertImageTestItem(t, ts, 0, KindMovie)
	url := imageTestURL(ts, id, "/images/poster")

	for name, value := range map[string]string{
		"vacía":           "",
		"sin esquema":     "images.example/poster.jpg",
		"ftp":             "ftp://images.example/poster.jpg",
		"sin host":        "https:///poster.jpg",
		"con espacio":     "https://images.example/pos ter.jpg",
		"demasiado larga": "https://images.example/" + strings.Repeat("a", 2050),
	} {
		t.Run(name, func(t *testing.T) {
			status, body := libraryHTTPDo(t, http.MethodPut, url, ts.token, map[string]string{"url": value})
			if status != http.StatusBadRequest {
				t.Fatalf("código = %d, cuerpo = %s", status, body)
			}
		})
	}
}

func TestItemImagePutErrores(t *testing.T) {
	fake := &fakeTMDB{}
	ts := newLibraryHTTPTestServer(t, fake)
	id := insertImageTestItem(t, ts, 0, KindMovie)
	episode := insertImageTestItem(t, ts, 0, KindEpisode)
	url := imageTestURL(ts, id, "/images/poster")

	if status, body := libraryHTTPDo(t, http.MethodPut, imageTestURL(ts, id, "/images/otra"), ts.token,
		map[string]string{"url": "https://images.example/p.jpg"}); status != http.StatusNotFound {
		t.Fatalf("tipo inválido: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodPut, imageTestURL(ts, episode, "/images/poster"), ts.token,
		map[string]string{"url": "https://images.example/p.jpg"}); status != http.StatusNotFound {
		t.Fatalf("episodio: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodPut, imageTestURL(ts, 999999, "/images/poster"), ts.token,
		map[string]string{"url": "https://images.example/p.jpg"}); status != http.StatusNotFound {
		t.Fatalf("ítem inexistente: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodPut, url, ts.token,
		map[string]string{"url": "https://images.example/p.jpg", "otro": "x"}); status != http.StatusBadRequest {
		t.Fatalf("campo desconocido: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodPut, url, nonAdminToken(t, ts),
		map[string]string{"url": "https://images.example/p.jpg"}); status != http.StatusForbidden {
		t.Fatalf("sin admin: código = %d, cuerpo = %s", status, body)
	}
}

func setImageLocked(t *testing.T, ts *libraryTestServer, id int64, poster, backdrop string) {
	t.Helper()
	if _, err := ts.db.Exec(`
		UPDATE media_items
		   SET poster_url = ?, poster_locked = 1,
		       backdrop_url = ?, backdrop_locked = 1
		 WHERE id = ?`, poster, backdrop, id); err != nil {
		t.Fatalf("fijar imágenes de prueba: %v", err)
	}
}

func TestItemImageDeleteRestauraTMDB(t *testing.T) {
	fake := &fakeTMDB{details: &metadata.Details{
		TMDBID:      603,
		PosterURL:   "https://images.example/poster-tmdb.jpg",
		BackdropURL: "https://images.example/fondo-tmdb.jpg",
	}}
	ts := newLibraryHTTPTestServer(t, fake)
	id := insertImageTestItem(t, ts, 603, KindMovie)
	setImageLocked(t, ts, id, "https://images.example/poster-propio.jpg", "https://images.example/fondo-propio.jpg")

	status, body := libraryHTTPDo(t, http.MethodDelete, imageTestURL(ts, id, "/images/poster"), ts.token, nil)
	if status != http.StatusOK {
		t.Fatalf("restaurar póster: código = %d, cuerpo = %s", status, body)
	}
	var item mediaItemDTO
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("respuesta: %v", err)
	}
	if item.PosterURL == nil || *item.PosterURL != "https://images.example/poster-tmdb.jpg" ||
		item.PosterCustom {
		t.Fatalf("póster restaurado inesperado: %+v", item)
	}
	if item.BackdropURL == nil || *item.BackdropURL != "https://images.example/fondo-propio.jpg" ||
		!item.BackdropCustom {
		t.Fatalf("el fondo fijado no debía cambiar: %+v", item)
	}

	status, body = libraryHTTPDo(t, http.MethodDelete, imageTestURL(ts, id, "/images/backdrop"), ts.token, nil)
	if status != http.StatusOK {
		t.Fatalf("restaurar fondo: código = %d, cuerpo = %s", status, body)
	}
	item = mediaItemDTO{}
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("respuesta del fondo: %v", err)
	}
	if item.BackdropURL == nil || *item.BackdropURL != "https://images.example/fondo-tmdb.jpg" ||
		item.BackdropCustom {
		t.Fatalf("fondo restaurado inesperado: %+v", item)
	}
}

func TestItemImageDeleteSinIdentificarNoNecesitaTMDB(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	id := insertImageTestItem(t, ts, 0, KindMovie)
	setImageLocked(t, ts, id, "https://images.example/poster-propio.jpg", "https://images.example/fondo-propio.jpg")

	status, body := libraryHTTPDo(t, http.MethodDelete, imageTestURL(ts, id, "/images/poster"), ts.token, nil)
	if status != http.StatusOK {
		t.Fatalf("restaurar póster sin TMDB: código = %d, cuerpo = %s", status, body)
	}
	var item mediaItemDTO
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("respuesta: %v", err)
	}
	if item.PosterURL != nil || item.PosterCustom {
		t.Fatalf("el póster debía quedar vacío: %+v", item)
	}
}

func TestItemImageDeleteErrores(t *testing.T) {
	idServer := newLibraryHTTPTestServer(t)
	id := insertImageTestItem(t, idServer, 603, KindMovie)
	if status, body := libraryHTTPDo(t, http.MethodDelete, imageTestURL(idServer, id, "/images/poster"), idServer.token, nil); status != http.StatusServiceUnavailable {
		t.Fatalf("sin cliente TMDB: código = %d, cuerpo = %s", status, body)
	}

	fake := &fakeTMDB{detailsErr: errors.New("red caída")}
	errServer := newLibraryHTTPTestServer(t, fake)
	errID := insertImageTestItem(t, errServer, 603, KindMovie)
	if status, body := libraryHTTPDo(t, http.MethodDelete, imageTestURL(errServer, errID, "/images/backdrop"), errServer.token, nil); status != http.StatusBadGateway {
		t.Fatalf("error de TMDB: código = %d, cuerpo = %s", status, body)
	}

	okFake := &fakeTMDB{details: &metadata.Details{TMDBID: 603}}
	ts := newLibraryHTTPTestServer(t, okFake)
	movie := insertImageTestItem(t, ts, 603, KindMovie)
	episode := insertImageTestItem(t, ts, 0, KindEpisode)
	if status, body := libraryHTTPDo(t, http.MethodDelete, imageTestURL(ts, movie, "/images/otra"), ts.token, nil); status != http.StatusNotFound {
		t.Fatalf("tipo inválido: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodDelete, imageTestURL(ts, episode, "/images/poster"), ts.token, nil); status != http.StatusNotFound {
		t.Fatalf("episodio: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodDelete, imageTestURL(ts, movie, "/images/poster"), nonAdminToken(t, ts), nil); status != http.StatusForbidden {
		t.Fatalf("sin admin: código = %d, cuerpo = %s", status, body)
	}
}
