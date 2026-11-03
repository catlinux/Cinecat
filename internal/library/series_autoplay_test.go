package library

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

type autoplayTestResponse struct {
	Enabled bool `json:"enabled"`
}

func autoplayURL(ts *libraryTestServer, seriesID int64) string {
	return ts.URL + "/api/v1/series/" + strconv.FormatInt(seriesID, 10) + "/autoplay"
}

func autoplayGet(t *testing.T, ts *libraryTestServer, seriesID int64, token string) autoplayTestResponse {
	t.Helper()
	status, body := libraryHTTPGet(t, autoplayURL(ts, seriesID), token)
	if status != http.StatusOK {
		t.Fatalf("GET autoplay de %d: código = %d, cuerpo = %s", seriesID, status, body)
	}
	var response autoplayTestResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("GET autoplay de %d: %v", seriesID, err)
	}
	return response
}

func autoplayPut(t *testing.T, ts *libraryTestServer, seriesID int64, token string, enabled bool) autoplayTestResponse {
	t.Helper()
	status, body := libraryHTTPDo(t, http.MethodPut, autoplayURL(ts, seriesID), token,
		map[string]any{"enabled": enabled})
	if status != http.StatusOK {
		t.Fatalf("PUT autoplay de %d: código = %d, cuerpo = %s", seriesID, status, body)
	}
	var response autoplayTestResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("PUT autoplay de %d: %v", seriesID, err)
	}
	return response
}

func autoplaySeriesDetail(t *testing.T, ts *libraryTestServer, seriesID int64, token string) seriesDetailDTO {
	t.Helper()
	status, body := libraryHTTPGet(t,
		ts.URL+"/api/v1/series/"+strconv.FormatInt(seriesID, 10), token)
	if status != http.StatusOK {
		t.Fatalf("detalle de la serie %d: código = %d, cuerpo = %s", seriesID, status, body)
	}
	var detail seriesDetailDTO
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("detalle de la serie %d: %v", seriesID, err)
	}
	return detail
}

func autoplayPlay(t *testing.T, ts *libraryTestServer, itemID int64, token string) playResponse {
	t.Helper()
	status, body := libraryHTTPGet(t,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10)+"/play?caps=mp4", token)
	if status != http.StatusOK {
		t.Fatalf("play del ítem %d: código = %d, cuerpo = %s", itemID, status, body)
	}
	var response playResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("play del ítem %d: %v", itemID, err)
	}
	return response
}

func writeAutoplayMedia(t *testing.T, root, name string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte("video de prueba"), 0o644); err != nil {
		t.Fatalf("crear vídeo %q: %v", name, err)
	}
	return path
}

func TestSeriesAutoplayPorDefectoYCambios(t *testing.T) {
	ts := newLeakHTTPServer(t)
	root := t.TempDir()
	libraryID := createLibrary(t, ts.db, root)
	seriesID := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Serie autoplay", title: "Serie autoplay",
	})
	episodePath := writeAutoplayMedia(t, root, "episodio.mp4")
	episodeID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: episodePath, title: "Episodio",
		kind: KindEpisode, seriesID: seriesID, season: 1, episode: 1,
		episodeTitle: "Piloto",
	})
	moviePath := writeAutoplayMedia(t, root, "pelicula.mp4")
	movieID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: moviePath, title: "Película", kind: KindMovie,
	})

	if response := autoplayGet(t, ts, seriesID, ts.token); !response.Enabled {
		t.Fatal("el ajuste por defecto debe estar activado")
	}
	if detail := autoplaySeriesDetail(t, ts, seriesID, ts.token); !detail.AutoplayNext {
		t.Fatal("la ficha debe indicar autoplay_next=true por defecto")
	}
	if response := autoplayPlay(t, ts, episodeID, ts.token); !response.AutoplayNext {
		t.Fatal("el episodio debe indicar autoplay_next=true por defecto")
	}
	if response := autoplayPlay(t, ts, movieID, ts.token); !response.AutoplayNext {
		t.Fatal("una película debe indicar autoplay_next=true")
	}

	if response := autoplayPut(t, ts, seriesID, ts.token, false); response.Enabled {
		t.Fatal("PUT false debe devolver enabled=false")
	}
	if response := autoplayGet(t, ts, seriesID, ts.token); response.Enabled {
		t.Fatal("GET debe conservar enabled=false")
	}
	if detail := autoplaySeriesDetail(t, ts, seriesID, ts.token); detail.AutoplayNext {
		t.Fatal("la ficha debe reflejar autoplay_next=false")
	}
	if response := autoplayPlay(t, ts, episodeID, ts.token); response.AutoplayNext {
		t.Fatal("el episodio debe reflejar autoplay_next=false")
	}

	if response := autoplayPut(t, ts, seriesID, ts.token, true); !response.Enabled {
		t.Fatal("PUT true debe restablecer el ajuste")
	}
	if response := autoplayGet(t, ts, seriesID, ts.token); !response.Enabled {
		t.Fatal("GET debe reflejar el restablecimiento")
	}
	if response := autoplayPlay(t, ts, episodeID, ts.token); !response.AutoplayNext {
		t.Fatal("el episodio debe volver a autoplay_next=true")
	}
}

func TestSeriesAutoplayUsuariosIndependientesYErrores(t *testing.T) {
	ts := newLeakHTTPServer(t)
	visibleRoot := t.TempDir()
	hiddenRoot := t.TempDir()
	visibleLibrary := createLibrary(t, ts.db, visibleRoot)
	hiddenLibrary := createLibrary(t, ts.db, hiddenRoot)
	seriesID := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: visibleLibrary, folder: "Serie visible", title: "Serie visible",
	})
	hiddenSeries := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: hiddenLibrary, folder: "Serie oculta", title: "Serie oculta",
	})
	userToken, _ := partialAccessToken(t, ts, visibleLibrary)

	if response := autoplayPut(t, ts, seriesID, ts.token, false); response.Enabled {
		t.Fatal("el administrador debe guardar false")
	}
	if response := autoplayGet(t, ts, seriesID, userToken); !response.Enabled {
		t.Fatal("otro usuario debe conservar el valor por defecto")
	}
	if response := autoplayPut(t, ts, seriesID, userToken, false); response.Enabled {
		t.Fatal("el segundo usuario debe guardar false")
	}
	if response := autoplayGet(t, ts, seriesID, ts.token); response.Enabled {
		t.Fatal("el ajuste del administrador no debe cambiar")
	}

	for _, request := range []struct {
		name   string
		method string
		url    string
		token  string
		body   any
		want   int
	}{
		{name: "oculta GET", method: http.MethodGet, url: autoplayURL(ts, hiddenSeries), token: userToken, want: http.StatusNotFound},
		{name: "oculta PUT", method: http.MethodPut, url: autoplayURL(ts, hiddenSeries), token: userToken, body: map[string]any{"enabled": true}, want: http.StatusNotFound},
		{name: "no es serie", method: http.MethodGet, url: ts.URL + "/api/v1/series/" + strconv.FormatInt(seriesID+1000, 10) + "/autoplay", token: userToken, want: http.StatusNotFound},
		{name: "sin sesión", method: http.MethodGet, url: autoplayURL(ts, seriesID), token: "", want: http.StatusUnauthorized},
		{name: "cuerpo vacío", method: http.MethodPut, url: autoplayURL(ts, seriesID), token: userToken, body: map[string]any{}, want: http.StatusBadRequest},
		{name: "tipo no válido", method: http.MethodPut, url: autoplayURL(ts, seriesID), token: userToken, body: map[string]any{"enabled": "no"}, want: http.StatusBadRequest},
	} {
		status, body := libraryHTTPDo(t, request.method, request.url, request.token, request.body)
		if status != request.want {
			t.Fatalf("%s: código = %d, se esperaba %d; cuerpo = %s",
				request.name, status, request.want, body)
		}
	}
}
