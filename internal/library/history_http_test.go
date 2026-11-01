package library

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistorySoloDelUsuarioYConAcceso(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	visibleLibrary := createLibrary(t, ts.db, t.TempDir())
	hiddenLibrary := createLibrary(t, ts.db, t.TempDir())
	token, userID := partialAccessToken(t, ts, visibleLibrary)
	otherToken := nonAdminToken(t, ts)
	otherID := progressUserID(t, ts.db, "bea")

	movieID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: visibleLibrary,
		path:      filepath.Join(t.TempDir(), "propia.mkv"),
		title:     "Propia",
		posterURL: "propia.jpg",
		kind:      KindMovie,
	})
	seenID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: visibleLibrary,
		path:      filepath.Join(t.TempDir(), "vista.mkv"),
		title:     "Vista",
		kind:      KindMovie,
	})
	hiddenID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: hiddenLibrary,
		path:      filepath.Join(t.TempDir(), "oculta.mkv"),
		title:     "Oculta",
		kind:      KindMovie,
	})
	seriesID := activitySeries(t, ts.db, visibleLibrary)
	episodeID := insertItemFixture(t, ts.db, itemFixture{
		libraryID:    visibleLibrary,
		path:         filepath.Join(t.TempDir(), "episodio.mkv"),
		title:        "Episodio",
		episodeTitle: "El episodio",
		seriesID:     seriesID,
		season:       2,
		episode:      3,
		kind:         KindEpisode,
	})

	setProgressRow(t, ts.db, userID, movieID, 120, 600, "2026-10-31T10:00:00Z")
	setActivityWatch(t, ts.db, userID, seenID, "2026-10-31T11:00:00Z")
	setProgressRow(t, ts.db, userID, episodeID, 30, 1800, "2026-10-31T12:00:00Z")
	setProgressRow(t, ts.db, userID, hiddenID, 50, 600, "2026-10-31T13:00:00Z")
	setProgressRow(t, ts.db, otherID, seenID, 60, 600, "2026-10-31T14:00:00Z")

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/history", token)
	if status != http.StatusOK {
		t.Fatalf("historial: código = %d, cuerpo = %s", status, body)
	}
	if strings.Contains(string(body), `"user_id"`) || strings.Contains(string(body), `"username"`) {
		t.Fatalf("el historial no debe incluir datos de usuario: %s", body)
	}
	var response historyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("historial: %v", err)
	}
	if len(response.Entries) != 3 {
		t.Fatalf("entradas = %d, se esperaban 3: %+v", len(response.Entries), response.Entries)
	}
	for _, entry := range response.Entries {
		if entry.ItemID == hiddenID {
			t.Fatalf("aparece un ítem de una biblioteca no accesible: %+v", entry)
		}
	}
	episode := response.Entries[0]
	if episode.ItemID != episodeID || episode.Title != "El episodio" ||
		episode.SeriesTitle == nil || *episode.SeriesTitle != "Serie de prueba" ||
		episode.Season == nil || *episode.Season != 2 ||
		episode.Episode == nil || *episode.Episode != 3 {
		t.Fatalf("entrada de episodio inesperada: %+v", episode)
	}
	movie := response.Entries[1]
	if movie.ItemID != seenID || !movie.Watched || movie.PosterURL != nil {
		t.Fatalf("entrada vista inesperada: %+v", movie)
	}
	progressMovie := response.Entries[2]
	if progressMovie.ItemID != movieID || progressMovie.PosterURL == nil ||
		*progressMovie.PosterURL != "propia.jpg" ||
		progressMovie.PositionSeconds != 120 || progressMovie.DurationSeconds != 600 ||
		progressMovie.Watched {
		t.Fatalf("entrada con progreso inesperada: %+v", progressMovie)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/history?limit=1", token)
	if status != http.StatusOK {
		t.Fatalf("límite de historial: código = %d, cuerpo = %s", status, body)
	}
	var limited historyResponse
	if err := json.Unmarshal(body, &limited); err != nil {
		t.Fatalf("límite de historial: %v", err)
	}
	if len(limited.Entries) != 1 || limited.Entries[0].ItemID != episodeID {
		t.Fatalf("límite de historial inesperado: %+v", limited.Entries)
	}

	// El otro usuario conserva su propia actividad.
	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/history", otherToken)
	if status != http.StatusOK {
		t.Fatalf("historial ajeno: código = %d, cuerpo = %s", status, body)
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("historial ajeno: %v", err)
	}
	if len(response.Entries) != 1 || response.Entries[0].ItemID != seenID {
		t.Fatalf("historial ajeno inesperado: %+v", response.Entries)
	}
}

func TestHistoryDeleteQuitaSoloLoDelUsuario(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	visibleLibrary := createLibrary(t, ts.db, t.TempDir())
	hiddenLibrary := createLibrary(t, ts.db, t.TempDir())
	token, userID := partialAccessToken(t, ts, visibleLibrary)
	otherToken := nonAdminToken(t, ts)
	otherID := progressUserID(t, ts.db, "bea")

	itemID := progressItem(t, ts, visibleLibrary, "Borrar historial", KindMovie)
	hiddenID := progressItem(t, ts, hiddenLibrary, "No accesible", KindMovie)
	setProgressRow(t, ts.db, userID, itemID, 10, 100, "2026-10-31T10:00:00Z")
	setProgressRow(t, ts.db, otherID, itemID, 20, 100, "2026-10-31T11:00:00Z")
	setActivityWatch(t, ts.db, userID, itemID, "2026-10-31T10:00:00Z")
	setActivityWatch(t, ts.db, otherID, itemID, "2026-10-31T11:00:00Z")
	setProgressRow(t, ts.db, userID, hiddenID, 30, 100, "2026-10-31T12:00:00Z")
	for _, target := range []struct {
		userID int64
		itemID int64
	}{
		{userID, itemID},
		{otherID, itemID},
	} {
		if _, err := ts.db.Exec(`
			INSERT INTO playback_state (user_id, media_item_id, position_seconds)
			VALUES (?, ?, 5)`, target.userID, target.itemID); err != nil {
			t.Fatalf("crear estado de reproducción: %v", err)
		}
	}

	if status, body := libraryHTTPDo(t, http.MethodDelete,
		fmt.Sprintf("%s/api/v1/history/%d", ts.URL, itemID), "", nil); status != http.StatusUnauthorized {
		t.Fatalf("sin sesión: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodDelete,
		fmt.Sprintf("%s/api/v1/history/%d", ts.URL, hiddenID), token, nil); status != http.StatusNotFound {
		t.Fatalf("ítem no accesible: código = %d, cuerpo = %s", status, body)
	}

	status, body := libraryHTTPDo(t, http.MethodDelete,
		fmt.Sprintf("%s/api/v1/history/%d", ts.URL, itemID), token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("borrar historial: código = %d, cuerpo = %s", status, body)
	}
	ctx := context.Background()
	var count int
	for _, query := range []string{
		`SELECT count(*) FROM playback_progress WHERE user_id = ? AND media_item_id = ?`,
		`SELECT count(*) FROM playback_state WHERE user_id = ? AND media_item_id = ?`,
		`SELECT count(*) FROM watch_state WHERE user_id = ? AND media_item_id = ?`,
	} {
		if err := ts.db.QueryRowContext(ctx, query, userID, itemID).Scan(&count); err != nil {
			t.Fatalf("comprobar historial borrado: %v", err)
		}
		if count != 0 {
			t.Fatalf("quedan %d filas del historial propio: %s", count, query)
		}
		if err := ts.db.QueryRowContext(ctx, query, otherID, itemID).Scan(&count); err != nil {
			t.Fatalf("comprobar historial ajeno: %v", err)
		}
		if count != 1 {
			t.Fatalf("el historial ajeno cambió: quedan %d filas: %s", count, query)
		}
	}
	var hiddenCount int
	if err := ts.db.QueryRowContext(ctx, `
		SELECT count(*) FROM playback_progress WHERE user_id = ? AND media_item_id = ?`,
		userID, hiddenID).Scan(&hiddenCount); err != nil {
		t.Fatalf("comprobar historial oculto: %v", err)
	}
	if hiddenCount != 1 {
		t.Fatal("el 404 no debe borrar el historial de un ítem no accesible")
	}

	// El otro usuario sigue viendo su entrada.
	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/history", otherToken)
	if status != http.StatusOK {
		t.Fatalf("historial ajeno: código = %d, cuerpo = %s", status, body)
	}
	var response historyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("historial ajeno: %v", err)
	}
	if len(response.Entries) != 1 || response.Entries[0].ItemID != itemID {
		t.Fatalf("el otro usuario perdió su historial: %+v", response.Entries)
	}
}
