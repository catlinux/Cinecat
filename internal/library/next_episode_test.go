package library

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
)

func nextEpisodePlay(t *testing.T, ts *libraryTestServer, itemID int64, token string) playResponse {
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

func insertNextEpisodeFixture(
	t *testing.T,
	ts *libraryTestServer,
	libraryID, seriesID int64,
	root, name, title, episodeTitle string,
	season, episode int,
) int64 {
	t.Helper()
	path := filepath.Join(root, name)
	writeFile(t, path)
	return insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: path, title: title, kind: KindEpisode,
		seriesID: seriesID, season: season, episode: episode, episodeTitle: episodeTitle,
	})
}

func TestPlayDevuelveElSiguienteEpisodio(t *testing.T) {
	ts := newLeakHTTPServer(t)
	root := t.TempDir()
	libraryID := createLibrary(t, ts.db, root)
	seriesID := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Serie siguiente", title: "Serie siguiente",
	})

	first := insertNextEpisodeFixture(t, ts, libraryID, seriesID, root,
		"t1e01.mp4", "1x01", "Uno", 1, 1)
	third := insertNextEpisodeFixture(t, ts, libraryID, seriesID, root,
		"t1e03.mp4", "1x03", "Tres", 1, 3)
	fourth := insertNextEpisodeFixture(t, ts, libraryID, seriesID, root,
		"t1e04.mp4", "1x04", "Cuatro", 1, 4)
	secondSeason := insertNextEpisodeFixture(t, ts, libraryID, seriesID, root,
		"t2e01.mp4", "2x01", "Dos uno", 2, 1)

	moviePath := filepath.Join(root, "pelicula.mp4")
	writeFile(t, moviePath)
	movie := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: moviePath, title: "Película", kind: KindMovie,
	})

	tests := []struct {
		name    string
		itemID  int64
		wantID  int64
		wantNil bool
		season  int
		episode int
		title   string
	}{
		{name: "salta huecos de numeración", itemID: first, wantID: third, season: 1, episode: 3, title: "Tres"},
		{name: "siguiente dentro de la temporada", itemID: third, wantID: fourth, season: 1, episode: 4, title: "Cuatro"},
		{name: "salto de temporada", itemID: fourth, wantID: secondSeason, season: 2, episode: 1, title: "Dos uno"},
		{name: "último episodio", itemID: secondSeason, wantNil: true},
		{name: "película", itemID: movie, wantNil: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := nextEpisodePlay(t, ts, test.itemID, ts.token)
			if test.wantNil {
				if response.NextEpisode != nil {
					t.Fatalf("next_episode = %+v, se esperaba null", response.NextEpisode)
				}
				return
			}
			next := response.NextEpisode
			if next == nil {
				t.Fatal("next_episode = null, se esperaba un episodio")
			}
			if next.ID != test.wantID || next.Season != test.season ||
				next.Episode != test.episode || next.Title != test.title {
				t.Fatalf("next_episode = %+v", next)
			}
		})
	}
}

func TestPlayNoRevelaUnSiguienteEpisodioOculto(t *testing.T) {
	ts := newLeakHTTPServer(t)
	visibleRoot := t.TempDir()
	hiddenRoot := t.TempDir()
	visibleLibrary := createLibrary(t, ts.db, visibleRoot)
	hiddenLibrary := createLibrary(t, ts.db, hiddenRoot)
	seriesID := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: visibleLibrary, folder: "Serie parcial", title: "Serie parcial",
	})

	first := insertNextEpisodeFixture(t, ts, visibleLibrary, seriesID, visibleRoot,
		"t1e01.mp4", "1x01", "Uno", 1, 1)
	hiddenNext := insertNextEpisodeFixture(t, ts, hiddenLibrary, seriesID, hiddenRoot,
		"t1e02.mp4", "1x02", "Dos", 1, 2)

	userToken, _ := partialAccessToken(t, ts, visibleLibrary)
	if response := nextEpisodePlay(t, ts, first, userToken); response.NextEpisode != nil {
		t.Fatalf("next_episode visible para un usuario parcial: %+v", response.NextEpisode)
	}
	if response := nextEpisodePlay(t, ts, first, ts.token); response.NextEpisode == nil ||
		response.NextEpisode.ID != hiddenNext {
		t.Fatalf("next_episode del administrador = %+v", response.NextEpisode)
	}
}
