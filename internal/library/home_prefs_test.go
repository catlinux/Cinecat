package library

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func homePreferenceByID(response homePreferencesResponse, id string) (homePreferenceRow, bool) {
	for _, row := range response.Rows {
		if row.ID == id {
			return row, true
		}
	}
	return homePreferenceRow{}, false
}

func TestHomePreferencesFiltraPorUsuario(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	movieID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      filepath.Join(t.TempDir(), "reciente.mkv"),
		title:     "Reciente",
		kind:      KindMovie,
	})
	seriesID := activitySeries(t, ts.db, libraryID)
	episodeID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      filepath.Join(t.TempDir(), "episodio.mkv"),
		title:     "Episodio",
		seriesID:  seriesID,
		season:    1,
		episode:   1,
		kind:      KindEpisode,
	})
	userToken := nonAdminToken(t, ts)
	userID := progressUserID(t, ts.db, "bea")
	setProgressRow(t, ts.db, userID, movieID, 60, 600, "2026-10-31T10:00:00Z")
	setProgressRow(t, ts.db, userID, episodeID, 30, 1200, "2026-10-31T11:00:00Z")

	if status, body := libraryHTTPDo(t, http.MethodPut,
		ts.URL+"/api/v1/home/preferences", userToken,
		map[string]any{"hidden": []string{"recent_movies", "continue_movies"}},
	); status != http.StatusNoContent {
		t.Fatalf("guardar preferencias: código = %d, cuerpo = %s", status, body)
	}

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/home/preferences", userToken)
	if status != http.StatusOK {
		t.Fatalf("preferencias: código = %d, cuerpo = %s", status, body)
	}
	var preferences homePreferencesResponse
	if err := json.Unmarshal(body, &preferences); err != nil {
		t.Fatalf("preferencias: %v", err)
	}
	if len(preferences.Rows) != 9 {
		t.Fatalf("filas = %d, se esperaban 9: %+v", len(preferences.Rows), preferences.Rows)
	}
	for _, id := range []string{"recent_movies", "continue_movies"} {
		row, ok := homePreferenceByID(preferences, id)
		if !ok || !row.Hidden || row.Title == "" {
			t.Fatalf("fila %q mal guardada: %+v", id, row)
		}
	}
	if row, _ := homePreferenceByID(preferences, "continue_series"); row.Hidden {
		t.Fatalf("continue_series no debía ocultarse: %+v", row)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/home", userToken)
	if status != http.StatusOK {
		t.Fatalf("portal del usuario: código = %d, cuerpo = %s", status, body)
	}
	var home homeResponse
	if err := json.Unmarshal(body, &home); err != nil {
		t.Fatalf("portal del usuario: %v", err)
	}
	if len(home.RecentMovies) != 0 {
		t.Fatalf("recent_movies debería estar oculto: %+v", home.RecentMovies)
	}
	if len(home.Continue) != 1 || home.Continue[0].ID != episodeID {
		t.Fatalf("continuar viendo debe conservar solo el episodio: %+v", home.Continue)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/home", ts.token)
	if status != http.StatusOK {
		t.Fatalf("portal del administrador: código = %d, cuerpo = %s", status, body)
	}
	if err := json.Unmarshal(body, &home); err != nil {
		t.Fatalf("portal del administrador: %v", err)
	}
	if len(home.RecentMovies) != 1 {
		t.Fatalf("otro usuario debe seguir viendo la fila reciente: %+v", home)
	}

	status, body = libraryHTTPDo(t, http.MethodPut,
		ts.URL+"/api/v1/home/preferences", userToken,
		map[string]any{"hidden": []string{"fila_inexistente"}},
	)
	if status != http.StatusBadRequest || !strings.Contains(string(body), "desconocida") {
		t.Fatalf("id desconocido: código = %d, cuerpo = %s", status, body)
	}

	status, body = libraryHTTPDo(t, http.MethodPut,
		ts.URL+"/api/v1/home/preferences", userToken,
		map[string]any{"hidden": []string{}},
	)
	if status != http.StatusNoContent {
		t.Fatalf("restaurar filas: código = %d, cuerpo = %s", status, body)
	}
	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/home", userToken)
	if status != http.StatusOK {
		t.Fatalf("portal restaurado: código = %d, cuerpo = %s", status, body)
	}
	if err := json.Unmarshal(body, &home); err != nil {
		t.Fatalf("portal restaurado: %v", err)
	}
	if len(home.RecentMovies) != 1 || len(home.Continue) != 2 {
		t.Fatalf("las filas restauradas deben volver: %+v", home)
	}
}

func TestHomePreferencesValidaSesionYJSON(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/home/preferences", ""); status != http.StatusUnauthorized {
		t.Fatalf("sin sesión: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodPut,
		ts.URL+"/api/v1/home/preferences", ts.token, map[string]any{"otra": []string{}},
	); status != http.StatusBadRequest {
		t.Fatalf("cuerpo desconocido: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodPut,
		ts.URL+"/api/v1/home/preferences", ts.token, map[string]any{},
	); status != http.StatusBadRequest {
		t.Fatalf("falta hidden: código = %d, cuerpo = %s", status, body)
	}
}
