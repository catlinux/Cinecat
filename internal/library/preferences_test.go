package library

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
)

func itemPreferencesURL(ts *libraryTestServer, id int64) string {
	return ts.URL + "/api/v1/items/" + strconv.FormatInt(id, 10) + "/preferences"
}

func TestPreferenciasGetPutDeleteYValidaciones(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	id := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      filepath.Join(t.TempDir(), "peli.mkv"),
		title:     "Peli",
	})

	status, body := libraryHTTPGet(t, itemPreferencesURL(ts, id), ts.token)
	if status != http.StatusOK {
		t.Fatalf("GET inicial: %d %s", status, body)
	}
	var preferences itemPreferencesDTO
	if err := json.Unmarshal(body, &preferences); err != nil {
		t.Fatalf("GET inicial: %v", err)
	}
	if preferences.AudioIndex != nil || preferences.SubtitleID != nil || preferences.Quality != nil {
		t.Fatalf("preferencias iniciales inesperadas: %+v", preferences)
	}

	status, body = libraryHTTPDo(t, http.MethodPut, itemPreferencesURL(ts, id), ts.token,
		map[string]any{"audio_index": 1})
	if status != http.StatusNoContent {
		t.Fatalf("PUT audio: %d %s", status, body)
	}
	status, body = libraryHTTPDo(t, http.MethodPut, itemPreferencesURL(ts, id), ts.token,
		map[string]any{"subtitle_id": "e0", "quality": "720"})
	if status != http.StatusNoContent {
		t.Fatalf("PUT parcial: %d %s", status, body)
	}
	status, body = libraryHTTPGet(t, itemPreferencesURL(ts, id), ts.token)
	if err := json.Unmarshal(body, &preferences); err != nil || status != http.StatusOK {
		t.Fatalf("GET tras PUT: %d %s (%v)", status, body, err)
	}
	if preferences.AudioIndex == nil || *preferences.AudioIndex != 1 ||
		preferences.SubtitleID == nil || *preferences.SubtitleID != "e0" ||
		preferences.Quality == nil || *preferences.Quality != "720" {
		t.Fatalf("preferencias guardadas inesperadas: %+v", preferences)
	}

	status, body = libraryHTTPDo(t, http.MethodPut, itemPreferencesURL(ts, id), ts.token,
		map[string]any{"audio_index": nil})
	if status != http.StatusNoContent {
		t.Fatalf("PUT null: %d %s", status, body)
	}
	status, body = libraryHTTPGet(t, itemPreferencesURL(ts, id), ts.token)
	if err := json.Unmarshal(body, &preferences); err != nil || status != http.StatusOK {
		t.Fatalf("GET tras null: %d %s (%v)", status, body, err)
	}
	if preferences.AudioIndex != nil || preferences.SubtitleID == nil || *preferences.SubtitleID != "e0" {
		t.Fatalf("null no borró solo audio: %+v", preferences)
	}

	for _, invalid := range []map[string]any{
		{"audio_index": -1},
		{"quality": "4k"},
		{"subtitle_id": "srt-1"},
		{"desconocido": 1},
	} {
		if status, body := libraryHTTPDo(t, http.MethodPut, itemPreferencesURL(ts, id),
			ts.token, invalid); status != http.StatusBadRequest {
			t.Fatalf("PUT inválido %v: %d %s", invalid, status, body)
		}
	}
	if status, _ := libraryHTTPDo(t, http.MethodPut, itemPreferencesURL(ts, 999999),
		ts.token, map[string]any{"audio_index": 0}); status != http.StatusNotFound {
		t.Fatalf("ítem inexistente: %d", status)
	}
	if status, _ := libraryHTTPGet(t, itemPreferencesURL(ts, id), ""); status != http.StatusUnauthorized {
		t.Fatalf("sin sesión: %d", status)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(id, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("GET ítem: %d %s", status, body)
	}
	var item mediaItemDTO
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("GET ítem: %v", err)
	}
	if item.Preferences == nil || item.Preferences.SubtitleID == nil || *item.Preferences.SubtitleID != "e0" {
		t.Fatalf("el ítem no incluye preferences: %+v", item.Preferences)
	}

	otherToken := nonAdminToken(t, ts)
	status, body = libraryHTTPGet(t, itemPreferencesURL(ts, id), otherToken)
	if err := json.Unmarshal(body, &preferences); err != nil || status != http.StatusOK {
		t.Fatalf("GET otro usuario: %d %s (%v)", status, body, err)
	}
	if preferences.AudioIndex != nil || preferences.SubtitleID != nil || preferences.Quality != nil {
		t.Fatalf("las preferencias se filtraron entre usuarios: %+v", preferences)
	}

	status, body = libraryHTTPDo(t, http.MethodDelete, itemPreferencesURL(ts, id), ts.token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("DELETE: %d %s", status, body)
	}
	status, body = libraryHTTPGet(t, itemPreferencesURL(ts, id), ts.token)
	if err := json.Unmarshal(body, &preferences); err != nil || status != http.StatusOK {
		t.Fatalf("GET tras DELETE: %d %s (%v)", status, body, err)
	}
	if preferences.AudioIndex != nil || preferences.SubtitleID != nil || preferences.Quality != nil {
		t.Fatalf("DELETE no borró las preferencias: %+v", preferences)
	}
}
