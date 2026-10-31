package library

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func activitySeries(t *testing.T, db *sql.DB, libraryID int64) int64 {
	t.Helper()
	result, err := db.Exec(`
		INSERT INTO series (library_id, folder, title)
		VALUES (?, 'Serie de prueba', 'Serie de prueba')`, libraryID)
	if err != nil {
		t.Fatalf("crear serie de actividad: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de serie de actividad: %v", err)
	}
	return id
}

func setActivityWatch(t *testing.T, db *sql.DB, userID, itemID int64, watchedAt string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO watch_state (user_id, media_item_id, watched_at)
		VALUES (?, ?, ?)
		ON CONFLICT (user_id, media_item_id) DO UPDATE SET watched_at = excluded.watched_at`,
		userID, itemID, watchedAt); err != nil {
		t.Fatalf("guardar visionado de actividad: %v", err)
	}
}

func TestActivityAdminListaOrdenFiltroYTipos(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	movieID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(t.TempDir(), "padrino.mkv"),
		title: "El Padrino", displayTitle: "El Padrino", year: 1972, kind: KindMovie,
	})
	seriesID := activitySeries(t, ts.db, libraryID)
	episodeID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(t.TempDir(), "episodio.mkv"),
		title: "Episodio de prueba", episodeTitle: "El episodio",
		seriesID: seriesID, season: 2, episode: 3, kind: KindEpisode,
	})
	watchOnlyID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(t.TempDir(), "visto.mkv"),
		title: "Solo visto", displayTitle: "Solo visto", year: 2001, kind: KindMovie,
	})

	adminID := progressUserID(t, ts.db, "ana")
	nonAdminToken(t, ts)
	otherID := progressUserID(t, ts.db, "bea")

	setProgressRow(t, ts.db, adminID, movieID, 600, 7200, "2026-10-30T10:00:00Z")
	setActivityWatch(t, ts.db, adminID, movieID, "2026-10-31T07:00:00Z")
	setActivityWatch(t, ts.db, adminID, watchOnlyID, "2026-10-31T08:00:00Z")
	setProgressRow(t, ts.db, otherID, episodeID, 120, 2400, "2026-10-31T09:00:00Z")

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/activity", ts.token)
	if status != http.StatusOK {
		t.Fatalf("actividad: código = %d, cuerpo = %s", status, body)
	}
	var response activityResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("actividad: %v", err)
	}
	if len(response.Entries) != 3 {
		t.Fatalf("entradas = %d, se esperaban 3: %+v", len(response.Entries), response.Entries)
	}

	episode := response.Entries[0]
	if episode.UserID != otherID || episode.Username != "bea" ||
		episode.ItemID != episodeID || episode.Title != "El episodio" ||
		episode.Kind != KindEpisode || episode.SeriesTitle == nil ||
		*episode.SeriesTitle != "Serie de prueba" ||
		episode.Season == nil || *episode.Season != 2 ||
		episode.Episode == nil || *episode.Episode != 3 ||
		episode.PositionSeconds != 120 || episode.DurationSeconds != 2400 ||
		episode.Watched || episode.UpdatedAt != "2026-10-31T09:00:00Z" {
		t.Fatalf("entrada de episodio inesperada: %+v", episode)
	}

	watchOnly := response.Entries[1]
	if watchOnly.UserID != adminID || watchOnly.ItemID != watchOnlyID ||
		watchOnly.Title != "Solo visto" || watchOnly.Kind != KindMovie ||
		watchOnly.SeriesTitle != nil || watchOnly.Season != nil || watchOnly.Episode != nil ||
		watchOnly.PositionSeconds != 0 || watchOnly.DurationSeconds != 0 ||
		!watchOnly.Watched || watchOnly.UpdatedAt != "2026-10-31T08:00:00Z" {
		t.Fatalf("entrada solo vista inesperada: %+v", watchOnly)
	}

	movie := response.Entries[2]
	if movie.UserID != adminID || movie.ItemID != movieID || movie.Title != "El Padrino" ||
		!movie.Watched || movie.PositionSeconds != 600 ||
		movie.UpdatedAt != "2026-10-31T07:00:00Z" {
		t.Fatalf("entrada de película inesperada: %+v", movie)
	}

	status, body = libraryHTTPGet(t,
		fmt.Sprintf("%s/api/v1/activity?user_id=%d", ts.URL, adminID), ts.token)
	if status != http.StatusOK {
		t.Fatalf("filtro de actividad: código = %d, cuerpo = %s", status, body)
	}
	var filtered activityResponse
	if err := json.Unmarshal(body, &filtered); err != nil {
		t.Fatalf("filtro de actividad: %v", err)
	}
	if len(filtered.Entries) != 2 {
		t.Fatalf("entradas filtradas = %d, se esperaban 2", len(filtered.Entries))
	}
	for _, entry := range filtered.Entries {
		if entry.UserID != adminID {
			t.Fatalf("entrada de otro usuario en el filtro: %+v", entry)
		}
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/activity?limit=1", ts.token)
	if status != http.StatusOK {
		t.Fatalf("límite de actividad: código = %d, cuerpo = %s", status, body)
	}
	if err := json.Unmarshal(body, &filtered); err != nil {
		t.Fatalf("límite de actividad: %v", err)
	}
	if len(filtered.Entries) != 1 || filtered.Entries[0].ItemID != episodeID {
		t.Fatalf("límite de actividad inesperado: %+v", filtered.Entries)
	}
}

func TestActivityPermisosYParametros(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/activity", ""); status != http.StatusUnauthorized {
		t.Fatalf("sin sesión: código = %d, cuerpo = %s", status, body)
	}
	userToken := nonAdminToken(t, ts)
	if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/activity", userToken); status != http.StatusForbidden {
		t.Fatalf("sin admin: código = %d, cuerpo = %s", status, body)
	}
	for _, query := range []string{"limit=0", "limit=abc", "user_id=0", "user_id=abc"} {
		if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/activity?"+query, ts.token); status != http.StatusBadRequest {
			t.Fatalf("%s: código = %d, cuerpo = %s", query, status, body)
		}
	}
	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/activity?user_id=999999", ts.token)
	if status != http.StatusOK {
		t.Fatalf("usuario sin actividad: código = %d, cuerpo = %s", status, body)
	}
	var response activityResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("usuario sin actividad: %v", err)
	}
	if response.Entries == nil || len(response.Entries) != 0 {
		t.Fatalf("el listado vacío debe ser un array vacío: %+v", response.Entries)
	}
}

func TestActivityLimitRecortaYValida(t *testing.T) {
	tests := []struct {
		raw     string
		want    int
		wantErr bool
	}{
		{raw: "", want: defaultActivityLimit},
		{raw: "25", want: 25},
		{raw: "200", want: 200},
		{raw: "201", want: maxActivityLimit},
		{raw: "9999", want: maxActivityLimit},
		{raw: "0", wantErr: true},
		{raw: "-1", wantErr: true},
		{raw: "abc", wantErr: true},
	}
	for _, test := range tests {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/activity?limit="+test.raw, nil)
		got, err := activityLimit(request)
		if test.wantErr {
			if err == nil {
				t.Fatalf("limit=%q debía fallar", test.raw)
			}
			continue
		}
		if err != nil || got != test.want {
			t.Fatalf("limit=%q: got %d, err %v; want %d", test.raw, got, err, test.want)
		}
	}
}
