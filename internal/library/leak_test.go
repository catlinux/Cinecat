package library

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/catlinux/cinecat/internal/auth"
	"github.com/catlinux/cinecat/internal/metadata"
	"github.com/catlinux/cinecat/internal/playback"
)

const (
	leakMarker   = "OCULTO-7f3a"
	leakPersonID = 777
	leakCredits  = `{"cast":[{"id":777,"name":"Persona","character":"Protagonista","photo":""}],"crew":[]}`
)

type leakFixture struct {
	userID                              int64
	visibleVideo, hiddenVideo           int64
	visibleClips, hiddenClips           int64
	visibleMusic, hiddenMusic           int64
	visibleMovie, hiddenMovie           int64
	visibleClip, hiddenClip             int64
	visibleSeries, hiddenSeries         int64
	visibleEpisode, hiddenEpisode       int64
	visibleEpisode2, hiddenEpisode2     int64
	visibleCollection, hiddenCollection int64
	visibleArtist, hiddenArtist         int64
	visibleAlbum, hiddenAlbum           int64
	visibleTrack, hiddenTrack           int64
}

type leakCase struct {
	name       string
	pattern    string
	path       string
	method     string
	body       any
	wantStatus int
	notFound   bool
	wantURL    bool
	adminMarca bool
}

func leakExec(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	result, err := db.Exec(query, args...)
	if err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId: %v", err)
	}
	return id
}

func leakLibrary(t *testing.T, db *sql.DB, path, name, kind string) (int64, int64) {
	t.Helper()
	id := leakExec(t, db,
		`INSERT INTO libraries (path, name, kind) VALUES (?, ?, ?)`, path, name, kind)
	root := leakExec(t, db, `INSERT INTO library_paths (library_id, path) VALUES (?, ?)`, id, path)
	return id, root
}

func leakItem(t *testing.T, db *sql.DB, libraryID int64, path, title, kind, genres string, seriesID *int64, season, episode *int) int64 {
	t.Helper()
	var series any
	if seriesID != nil {
		series = *seriesID
	}
	var seasonValue, episodeValue any
	if season != nil {
		seasonValue = *season
	}
	if episode != nil {
		episodeValue = *episode
	}
	return leakExec(t, db, `
		INSERT INTO media_items (library_id, path, title, kind, genres, series_id, season, episode)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		libraryID, path, title, kind, genres, series, seasonValue, episodeValue)
}

func newLeakFixture(t *testing.T, ts *libraryTestServer) (leakFixture, string) {
	t.Helper()
	var f leakFixture
	visibleVideoRoot := t.TempDir()
	hiddenVideoRoot := filepath.Join(t.TempDir(), leakMarker+"-video")
	visibleClipsRoot := t.TempDir()
	hiddenClipsRoot := filepath.Join(t.TempDir(), leakMarker+"-clips")
	visibleMusicRoot := t.TempDir()
	hiddenMusicRoot := filepath.Join(t.TempDir(), leakMarker+"-music")
	f.visibleVideo, _ = leakLibrary(t, ts.db, visibleVideoRoot, "Vídeo visible", "video")
	f.hiddenVideo, _ = leakLibrary(t, ts.db, hiddenVideoRoot, leakMarker+" vídeo", "video")
	f.visibleClips, _ = leakLibrary(t, ts.db, visibleClipsRoot, "Clips visibles", "clips")
	f.hiddenClips, _ = leakLibrary(t, ts.db, hiddenClipsRoot, leakMarker+" clips", "clips")
	f.visibleMusic, _ = leakLibrary(t, ts.db, visibleMusicRoot, "Música visible", "music")
	f.hiddenMusic, _ = leakLibrary(t, ts.db, hiddenMusicRoot, leakMarker+" música", "music")

	visibleMoviePath := filepath.Join(visibleVideoRoot, "pelicula-visible.mp4")
	writeFile(t, visibleMoviePath)
	visibleClipPath := filepath.Join(visibleClipsRoot, "video-visible.mkv")
	writeFile(t, visibleClipPath)
	f.visibleMovie = leakItem(t, ts.db, f.visibleVideo,
		visibleMoviePath, "Película visible", "movie", `["Drama"]`, nil, nil, nil)
	f.hiddenMovie = leakItem(t, ts.db, f.hiddenVideo,
		filepath.Join(hiddenVideoRoot, leakMarker+"-pelicula.mkv"), leakMarker+" película", "movie",
		`["Drama","OCULTO-7f3a"]`, nil, nil, nil)
	f.visibleClip = leakItem(t, ts.db, f.visibleClips,
		visibleClipPath, "Vídeo visible", "movie", `[]`, nil, nil, nil)
	f.hiddenClip = leakItem(t, ts.db, f.hiddenClips,
		filepath.Join(hiddenClipsRoot, leakMarker+"-video.mkv"), leakMarker+" vídeo", "movie", `[]`, nil, nil, nil)

	f.visibleSeries = leakExec(t, ts.db, `
		INSERT INTO series (library_id, folder, title, display_title, genres, poster_url, match_status)
		VALUES (?, ?, ?, ?, '["Drama"]', 'https://img/visible.jpg', 'auto')`,
		f.visibleVideo, "visible-serie", "Serie visible", "Serie visible")
	f.hiddenSeries = leakExec(t, ts.db, `
		INSERT INTO series (library_id, folder, title, display_title, genres, poster_url, match_status)
		VALUES (?, ?, ?, ?, '["OCULTO-7f3a"]', 'https://img/OCULTO-7f3a.jpg', 'auto')`,
		f.hiddenVideo, leakMarker+"-serie", leakMarker+" serie", leakMarker+" serie")
	leakExec(t, ts.db, `UPDATE media_items SET tmdb_id = ?, credits = ? WHERE id = ?`,
		1001, leakCredits, f.visibleMovie)
	leakExec(t, ts.db, `UPDATE media_items SET tmdb_id = ?, credits = ? WHERE id = ?`,
		1002, leakCredits, f.hiddenMovie)
	leakExec(t, ts.db, `UPDATE series SET tmdb_id = ?, credits = ? WHERE id = ?`,
		2001, leakCredits, f.visibleSeries)
	leakExec(t, ts.db, `UPDATE series SET tmdb_id = ?, credits = ? WHERE id = ?`,
		2002, leakCredits, f.hiddenSeries)
	season, episode := 1, 1
	f.visibleEpisode = leakItem(t, ts.db, f.visibleVideo,
		"/media/visible-video/visible-serie/S01E01.mkv", "Episodio visible", "episode", `[]`, &f.visibleSeries, &season, &episode)
	episode = 2
	f.visibleEpisode2 = leakItem(t, ts.db, f.visibleVideo,
		"/media/visible-video/visible-serie/S01E02.mkv", "Episodio visible 2", "episode", `[]`, &f.visibleSeries, &season, &episode)
	episode = 1
	f.hiddenEpisode = leakItem(t, ts.db, f.hiddenVideo,
		"/media/OCULTO-7f3a-video/OCULTO-7f3a-serie/S01E01.mkv", leakMarker+" episodio", "episode", `[]`, &f.hiddenSeries, &season, &episode)
	episode = 2
	f.hiddenEpisode2 = leakItem(t, ts.db, f.hiddenVideo,
		"/media/OCULTO-7f3a-video/OCULTO-7f3a-serie/S01E02.mkv", leakMarker+" episodio 2", "episode", `[]`, &f.hiddenSeries, &season, &episode)

	f.visibleCollection = leakExec(t, ts.db,
		`INSERT INTO collections (name, source) VALUES (?, 'manual')`, "Colección visible")
	f.hiddenCollection = leakExec(t, ts.db,
		`INSERT INTO collections (name, source) VALUES (?, 'manual')`, leakMarker+" colección")
	leakExec(t, ts.db, `INSERT INTO collection_items (collection_id, media_item_id) VALUES (?, ?)`,
		f.visibleCollection, f.visibleMovie)
	leakExec(t, ts.db, `INSERT INTO collection_items (collection_id, media_item_id) VALUES (?, ?)`,
		f.hiddenCollection, f.hiddenMovie)

	f.visibleArtist = leakExec(t, ts.db,
		`INSERT INTO music_artists (name, sort_name) VALUES (?, ?)`, "Artista visible", "Artista visible")
	f.hiddenArtist = leakExec(t, ts.db,
		`INSERT INTO music_artists (name, sort_name) VALUES (?, ?)`, leakMarker+" artista", leakMarker+" artista")
	f.visibleAlbum = leakExec(t, ts.db,
		`INSERT INTO music_albums (library_id, artist_id, title) VALUES (?, ?, ?)`,
		f.visibleMusic, f.visibleArtist, "Álbum visible")
	f.hiddenAlbum = leakExec(t, ts.db,
		`INSERT INTO music_albums (library_id, artist_id, title) VALUES (?, ?, ?)`,
		f.hiddenMusic, f.hiddenArtist, leakMarker+" álbum")
	visibleTrackPath := filepath.Join(visibleMusicRoot, "pista-visible.flac")
	writeFile(t, visibleTrackPath)
	visibleCoverPath := filepath.Join(visibleMusicRoot, "cover.jpg")
	if err := writeTestJPEG(visibleCoverPath, 220); err != nil {
		t.Fatalf("crear carátula visible: %v", err)
	}
	if _, err := ts.db.Exec(`UPDATE music_albums SET cover_path = ? WHERE id = ?`,
		visibleCoverPath, f.visibleAlbum); err != nil {
		t.Fatalf("guardar carátula visible: %v", err)
	}
	f.visibleTrack = leakExec(t, ts.db, `
		INSERT INTO music_tracks (library_id, album_id, artist_id, path, title, size_bytes, mtime_unix)
		VALUES (?, ?, ?, ?, ?, 100, 1)`,
		f.visibleMusic, f.visibleAlbum, f.visibleArtist,
		visibleTrackPath, "Pista visible")
	f.hiddenTrack = leakExec(t, ts.db, `
		INSERT INTO music_tracks (library_id, album_id, artist_id, path, title, size_bytes, mtime_unix)
		VALUES (?, ?, ?, ?, ?, 100, 1)`,
		f.hiddenMusic, f.hiddenAlbum, f.hiddenArtist,
		filepath.Join(hiddenMusicRoot, leakMarker+"-pista.flac"), leakMarker+" pista")

	hash, err := auth.HashPassword("Clave-user-1")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	userID := leakExec(t, ts.db, `
		INSERT INTO users (username, password_hash, role, all_libraries)
		VALUES ('user', ?, 'user', 0)`, hash)
	f.userID = userID
	for _, libraryID := range []int64{f.visibleVideo, f.visibleClips, f.visibleMusic} {
		leakExec(t, ts.db, `INSERT INTO user_libraries (user_id, library_id) VALUES (?, ?)`,
			userID, libraryID)
	}
	for _, itemID := range []int64{f.visibleMovie, f.hiddenMovie, f.visibleEpisode, f.hiddenEpisode} {
		leakExec(t, ts.db, `INSERT INTO item_lists (user_id, media_item_id, list) VALUES (?, ?, 'favorite')`,
			userID, itemID)
		leakExec(t, ts.db, `INSERT INTO playback_progress
			(user_id, media_item_id, position_seconds, duration_seconds) VALUES (?, ?, 1, 2)`,
			userID, itemID)
	}
	for _, itemID := range []int64{f.visibleEpisode, f.hiddenEpisode} {
		leakExec(t, ts.db, `INSERT INTO watch_state (user_id, media_item_id) VALUES (?, ?)`,
			userID, itemID)
	}
	leakExec(t, ts.db, `INSERT INTO playback_progress
		(user_id, media_item_id, position_seconds, duration_seconds) VALUES (?, ?, 12, 100)`,
		userID, f.hiddenEpisode2)
	leakExec(t, ts.db, `INSERT INTO watch_state (user_id, media_item_id) VALUES (?, ?)`,
		userID, f.hiddenEpisode2)
	leakExec(t, ts.db, `INSERT INTO item_lists (user_id, media_item_id, list) VALUES (?, ?, 'favorite')`,
		userID, f.hiddenEpisode2)
	leakExec(t, ts.db, `INSERT INTO item_preferences (user_id, media_item_id, quality) VALUES (?, ?, '720')`,
		userID, f.hiddenEpisode2)
	for _, seriesID := range []int64{f.visibleSeries, f.hiddenSeries} {
		leakExec(t, ts.db, `INSERT INTO series_lists (user_id, series_id, list) VALUES (?, ?, 'favorite')`,
			userID, seriesID)
	}
	for _, collectionID := range []int64{f.visibleCollection, f.hiddenCollection} {
		leakExec(t, ts.db, `INSERT INTO collection_lists (user_id, collection_id, list) VALUES (?, ?, 'favorite')`,
			userID, collectionID)
	}
	token, _, err := ts.auth.Login(context.Background(), "user", "Clave-user-1")
	if err != nil {
		t.Fatalf("Login user: %v", err)
	}
	return f, token
}

func leakAllLibrariesToken(t *testing.T, ts *libraryTestServer) string {
	t.Helper()
	hash, err := auth.HashPassword("Clave-all-1")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	leakExec(t, ts.db, `
		INSERT INTO users (username, password_hash, role, all_libraries)
		VALUES ('alluser', ?, 'user', 1)`, hash)
	token, _, err := ts.auth.Login(context.Background(), "alluser", "Clave-all-1")
	if err != nil {
		t.Fatalf("Login alluser: %v", err)
	}
	return token
}

func leakNoLibrariesToken(t *testing.T, ts *libraryTestServer) string {
	t.Helper()
	hash, err := auth.HashPassword("Clave-nadie-1")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	leakExec(t, ts.db, `
		INSERT INTO users (username, password_hash, role, all_libraries)
		VALUES ('nobody', ?, 'user', 0)`, hash)
	token, _, err := ts.auth.Login(context.Background(), "nobody", "Clave-nadie-1")
	if err != nil {
		t.Fatalf("Login nobody: %v", err)
	}
	return token
}

func newLeakHTTPServer(t *testing.T) *libraryTestServer {
	t.Helper()
	tmdbServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/person/777" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": leakPersonID, "name": "Persona",
			"combined_credits": map[string]any{
				"cast": []map[string]any{
					{"id": 1001, "media_type": "movie", "title": "Obra externa 1", "release_date": "2000-01-01", "character": "Rol"},
					{"id": 1002, "media_type": "movie", "title": "Obra externa 2", "release_date": "2001-01-01", "character": "Rol"},
					{"id": 2001, "media_type": "tv", "name": "Serie externa 1", "first_air_date": "2002-01-01", "character": "Rol"},
					{"id": 2002, "media_type": "tv", "name": "Serie externa 2", "first_air_date": "2003-01-01", "character": "Rol"},
				},
				"crew": []map[string]any{},
			},
		})
	}))
	t.Cleanup(tmdbServer.Close)
	tmdb, err := metadata.New(metadata.Options{
		APIKey:       "clave",
		BaseURL:      tmdbServer.URL,
		ImageBaseURL: tmdbServer.URL + "/img",
		CacheDir:     t.TempDir(),
		HTTPClient:   tmdbServer.Client(),
	})
	if err != nil {
		t.Fatalf("metadata.New: %v", err)
	}
	base := newLibraryHTTPTestServer(t, tmdb)
	playSvc, err := playback.New(playback.Options{WorkDir: t.TempDir()})
	if err != nil {
		t.Fatalf("playback.New: %v", err)
	}
	t.Cleanup(func() {
		if err := playSvc.Close(); err != nil {
			t.Errorf("playback.Close: %v", err)
		}
	})
	api := NewAPI(base.db, base.auth, playSvc, base.job, base.api.tmdb).
		WithSettings(base.api.settingsStore).
		WithFileInspector(&fakeFileInspector{info: fileInfoFixture()}).
		WithThumbnails(filepath.Join(t.TempDir(), "thumbs"))
	runner := &thumbnailRunnerRecord{}
	api.WithThumbnailRunner(runner.run)
	mux := http.NewServeMux()
	api.Register(mux)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	base.Server = server
	base.api = api
	return base
}

func leakRoutes() []leakCase {
	// Los paths se rellenan en el test, cuando ya existen los ids del fixture.
	return []leakCase{
		{name: "items", pattern: "/api/v1/items", path: "/api/v1/items", adminMarca: true},
		{name: "items búsqueda oculta", pattern: "/api/v1/items", path: "/api/v1/items?q=" + leakMarker},
		{name: "items biblioteca oculta", pattern: "/api/v1/items", path: "/api/v1/items?library_id="},
		{name: "items similares de oculto", pattern: "/api/v1/items/{id}/similar", notFound: true},
		{name: "items similares de visible", pattern: "/api/v1/items/{id}/similar"},
		{name: "items oculto", pattern: "/api/v1/items/{id}", notFound: true},
		{name: "items visible", pattern: "/api/v1/items/{id}"},
		{name: "items miniatura oculta", pattern: "/api/v1/items/{id}/thumbnail", notFound: true},
		{name: "items miniatura visible", pattern: "/api/v1/items/{id}/thumbnail"},
		{name: "items file-info oculto", pattern: "/api/v1/items/{id}/file-info", notFound: true},
		{name: "items file-info visible", pattern: "/api/v1/items/{id}/file-info"},
		{name: "items preferencias ocultas", pattern: "/api/v1/items/{id}/preferences", notFound: true},
		{name: "items preferencias visibles", pattern: "/api/v1/items/{id}/preferences"},
		{name: "items reproducción oculta", pattern: "/api/v1/items/{id}/play", notFound: true},
		{name: "items reproducción visible", pattern: "/api/v1/items/{id}/play", wantURL: true},
		{name: "progreso oculto PUT", pattern: "/api/v1/items/{id}/progress", path: "/api/v1/items/", method: http.MethodPut, body: map[string]any{"position_seconds": 20, "duration_seconds": 100}, notFound: true},
		{name: "progreso oculto DELETE", pattern: "/api/v1/items/{id}/progress", path: "/api/v1/items/", method: http.MethodDelete, notFound: true},
		{name: "visto oculto PUT", pattern: "/api/v1/items/{id}/watched", path: "/api/v1/items/", method: http.MethodPut, notFound: true},
		{name: "visto oculto DELETE", pattern: "/api/v1/items/{id}/watched", path: "/api/v1/items/", method: http.MethodDelete, notFound: true},
		{name: "lista oculta PUT", pattern: "/api/v1/items/{id}/lists/{list}", path: "/api/v1/items/", method: http.MethodPut, notFound: true},
		{name: "lista oculta DELETE", pattern: "/api/v1/items/{id}/lists/{list}", path: "/api/v1/items/", method: http.MethodDelete, notFound: true},
		{name: "preferencias ocultas PUT", pattern: "/api/v1/items/{id}/preferences", path: "/api/v1/items/", method: http.MethodPut, body: map[string]any{"quality": "1080"}, notFound: true},
		{name: "preferencias ocultas DELETE", pattern: "/api/v1/items/{id}/preferences", path: "/api/v1/items/", method: http.MethodDelete, notFound: true},
		{name: "géneros", pattern: "/api/v1/genres", path: "/api/v1/genres", adminMarca: true},
		{name: "portal", pattern: "/api/v1/home", path: "/api/v1/home", adminMarca: true},
		{name: "favoritos", pattern: "/api/v1/lists/{list}", path: "/api/v1/lists/favorite?kind=items"},
		{name: "próximas", pattern: "/api/v1/lists/{list}", path: "/api/v1/lists/watchlist?kind=items"},
		{name: "series favoritas", pattern: "/api/v1/lists/{list}", path: "/api/v1/lists/favorite?kind=series"},
		{name: "colecciones favoritas", pattern: "/api/v1/lists/{list}", path: "/api/v1/lists/favorite?kind=collections"},
		{name: "visto de serie oculta PUT", pattern: "/api/v1/series/{id}/watched", path: "/api/v1/series/", method: http.MethodPut, notFound: true},
		{name: "visto de serie oculta DELETE", pattern: "/api/v1/series/{id}/watched", path: "/api/v1/series/", method: http.MethodDelete, notFound: true},
		{name: "temporada vista de serie oculta PUT", pattern: "/api/v1/series/{id}/seasons/{n}/watched", path: "/api/v1/series/", method: http.MethodPut, notFound: true},
		{name: "temporada vista de serie oculta DELETE", pattern: "/api/v1/series/{id}/seasons/{n}/watched", path: "/api/v1/series/", method: http.MethodDelete, notFound: true},
		{name: "lista de serie oculta PUT", pattern: "/api/v1/series/{id}/lists/{list}", path: "/api/v1/series/", method: http.MethodPut, notFound: true},
		{name: "lista de serie oculta DELETE", pattern: "/api/v1/series/{id}/lists/{list}", path: "/api/v1/series/", method: http.MethodDelete, notFound: true},
		{name: "lista de colección oculta PUT", pattern: "/api/v1/collections/{id}/lists/{list}", path: "/api/v1/collections/", method: http.MethodPut, notFound: true},
		{name: "lista de colección oculta DELETE", pattern: "/api/v1/collections/{id}/lists/{list}", path: "/api/v1/collections/", method: http.MethodDelete, notFound: true},
		{name: "colecciones", pattern: "/api/v1/collections", path: "/api/v1/collections", adminMarca: true},
		{name: "colección oculta", pattern: "/api/v1/collections/{id}", notFound: true},
		{name: "colección visible", pattern: "/api/v1/collections/{id}"},
		{name: "persona", pattern: "/api/v1/people/{id}", path: "/api/v1/people/777", adminMarca: true},
		{name: "series", pattern: "/api/v1/series", path: "/api/v1/series", adminMarca: true},
		{name: "géneros de series", pattern: "/api/v1/series/genres", path: "/api/v1/series/genres", adminMarca: true},
		{name: "serie oculta", pattern: "/api/v1/series/{id}", notFound: true},
		{name: "serie visible", pattern: "/api/v1/series/{id}"},
		{name: "similares de serie oculta", pattern: "/api/v1/series/{id}/similar", notFound: true},
		{name: "similares de serie visible", pattern: "/api/v1/series/{id}/similar"},
		{name: "temporada de serie oculta", pattern: "/api/v1/series/{id}/seasons/{n}", notFound: true},
		{name: "temporada de serie visible", pattern: "/api/v1/series/{id}/seasons/{n}"},
		{name: "artistas", pattern: "/api/v1/music/artists", path: "/api/v1/music/artists", adminMarca: true},
		{name: "artista oculto", pattern: "/api/v1/music/artists/{id}", notFound: true},
		{name: "artista visible", pattern: "/api/v1/music/artists/{id}"},
		{name: "álbumes", pattern: "/api/v1/music/albums", path: "/api/v1/music/albums", adminMarca: true},
		{name: "álbum oculto", pattern: "/api/v1/music/albums/{id}", notFound: true},
		{name: "álbum visible", pattern: "/api/v1/music/albums/{id}"},
		{name: "carátula oculta", pattern: "/api/v1/music/albums/{id}/cover", notFound: true},
		{name: "carátula visible", pattern: "/api/v1/music/albums/{id}/cover"},
		{name: "pistas", pattern: "/api/v1/music/tracks", path: "/api/v1/music/tracks", adminMarca: true},
		{name: "reproducción musical oculta", pattern: "/api/v1/music/tracks/{id}/play", notFound: true},
		{name: "reproducción musical visible", pattern: "/api/v1/music/tracks/{id}/play", wantURL: true},
		{name: "bibliotecas", pattern: "/api/v1/libraries", path: "/api/v1/libraries", adminMarca: true},
		{name: "navegar clips visibles", pattern: "/api/v1/libraries/{id}/browse"},
		{name: "navegar clips ocultos", pattern: "/api/v1/libraries/{id}/browse", notFound: true},
		{name: "ajustes públicos", pattern: "/api/v1/settings/public", path: "/api/v1/settings/public"},
		{name: "stream de música sin firma", pattern: "/api/v1/stream/music/{id}", path: "/api/v1/stream/music/", wantStatus: http.StatusForbidden},
		{name: "stream directo sin firma", pattern: "/api/v1/stream/{id}", path: "/api/v1/stream/", wantStatus: http.StatusForbidden},
		{name: "archivo de stream sin firma", pattern: "/api/v1/stream/{id}/{file}", path: "/api/v1/stream/", wantStatus: http.StatusForbidden},
		{name: "subtítulos de stream sin firma", pattern: "/api/v1/stream/{id}/subtitles/{file}", path: "/api/v1/stream/", wantStatus: http.StatusForbidden},
	}
}

func TestPruebaDeFugas(t *testing.T) {
	ts := newLeakHTTPServer(t)
	fixture, userToken := newLeakFixture(t, ts)
	adminToken := ts.token

	paths := map[string]func() string{
		"items búsqueda oculta": func() string { return "/api/v1/items?q=" + leakMarker },
		"items biblioteca oculta": func() string {
			return "/api/v1/items?library_id=" + strconv.FormatInt(fixture.hiddenVideo, 10)
		},
		"items similares de oculto": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.hiddenMovie, 10) + "/similar"
		},
		"items similares de visible": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.visibleMovie, 10) + "/similar"
		},
		"items oculto": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.hiddenMovie, 10)
		},
		"items visible": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.visibleMovie, 10)
		},
		"items miniatura oculta": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.hiddenClip, 10) + "/thumbnail"
		},
		"items miniatura visible": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.visibleClip, 10) + "/thumbnail"
		},
		"items file-info oculto": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.hiddenMovie, 10) + "/file-info"
		},
		"items file-info visible": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.visibleMovie, 10) + "/file-info"
		},
		"items preferencias ocultas": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.hiddenMovie, 10) + "/preferences"
		},
		"items preferencias visibles": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.visibleMovie, 10) + "/preferences"
		},
		"items reproducción oculta": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.hiddenMovie, 10) + "/play"
		},
		"items reproducción visible": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.visibleMovie, 10) + "/play"
		},
		"progreso oculto PUT": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.hiddenEpisode2, 10) + "/progress"
		},
		"progreso oculto DELETE": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.hiddenEpisode2, 10) + "/progress"
		},
		"visto oculto PUT": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.hiddenEpisode2, 10) + "/watched"
		},
		"visto oculto DELETE": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.hiddenEpisode2, 10) + "/watched"
		},
		"lista oculta PUT": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.hiddenEpisode2, 10) + "/lists/favorite"
		},
		"lista oculta DELETE": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.hiddenEpisode2, 10) + "/lists/favorite"
		},
		"preferencias ocultas PUT": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.hiddenEpisode2, 10) + "/preferences"
		},
		"preferencias ocultas DELETE": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.hiddenEpisode2, 10) + "/preferences"
		},
		"colección oculta": func() string {
			return "/api/v1/collections/" + strconv.FormatInt(fixture.hiddenCollection, 10)
		},
		"colección visible": func() string {
			return "/api/v1/collections/" + strconv.FormatInt(fixture.visibleCollection, 10)
		},
		"visto de serie oculta PUT": func() string {
			return "/api/v1/series/" + strconv.FormatInt(fixture.hiddenSeries, 10) + "/watched"
		},
		"visto de serie oculta DELETE": func() string {
			return "/api/v1/series/" + strconv.FormatInt(fixture.hiddenSeries, 10) + "/watched"
		},
		"temporada vista de serie oculta PUT": func() string {
			return "/api/v1/series/" + strconv.FormatInt(fixture.hiddenSeries, 10) + "/seasons/1/watched"
		},
		"temporada vista de serie oculta DELETE": func() string {
			return "/api/v1/series/" + strconv.FormatInt(fixture.hiddenSeries, 10) + "/seasons/1/watched"
		},
		"lista de serie oculta PUT": func() string {
			return "/api/v1/series/" + strconv.FormatInt(fixture.hiddenSeries, 10) + "/lists/favorite"
		},
		"lista de serie oculta DELETE": func() string {
			return "/api/v1/series/" + strconv.FormatInt(fixture.hiddenSeries, 10) + "/lists/favorite"
		},
		"lista de colección oculta PUT": func() string {
			return "/api/v1/collections/" + strconv.FormatInt(fixture.hiddenCollection, 10) + "/lists/favorite"
		},
		"lista de colección oculta DELETE": func() string {
			return "/api/v1/collections/" + strconv.FormatInt(fixture.hiddenCollection, 10) + "/lists/favorite"
		},
		"serie oculta": func() string {
			return "/api/v1/series/" + strconv.FormatInt(fixture.hiddenSeries, 10)
		},
		"serie visible": func() string {
			return "/api/v1/series/" + strconv.FormatInt(fixture.visibleSeries, 10)
		},
		"similares de serie oculta": func() string {
			return "/api/v1/series/" + strconv.FormatInt(fixture.hiddenSeries, 10) + "/similar"
		},
		"similares de serie visible": func() string {
			return "/api/v1/series/" + strconv.FormatInt(fixture.visibleSeries, 10) + "/similar"
		},
		"temporada de serie oculta": func() string {
			return "/api/v1/series/" + strconv.FormatInt(fixture.hiddenSeries, 10) + "/seasons/1"
		},
		"temporada de serie visible": func() string {
			return "/api/v1/series/" + strconv.FormatInt(fixture.visibleSeries, 10) + "/seasons/1"
		},
		"artista oculto": func() string {
			return "/api/v1/music/artists/" + strconv.FormatInt(fixture.hiddenArtist, 10)
		},
		"artista visible": func() string {
			return "/api/v1/music/artists/" + strconv.FormatInt(fixture.visibleArtist, 10)
		},
		"álbum oculto": func() string {
			return "/api/v1/music/albums/" + strconv.FormatInt(fixture.hiddenAlbum, 10)
		},
		"álbum visible": func() string {
			return "/api/v1/music/albums/" + strconv.FormatInt(fixture.visibleAlbum, 10)
		},
		"carátula oculta": func() string {
			return "/api/v1/music/albums/" + strconv.FormatInt(fixture.hiddenAlbum, 10) + "/cover"
		},
		"carátula visible": func() string {
			return "/api/v1/music/albums/" + strconv.FormatInt(fixture.visibleAlbum, 10) + "/cover"
		},
		"reproducción musical oculta": func() string {
			return "/api/v1/music/tracks/" + strconv.FormatInt(fixture.hiddenTrack, 10) + "/play"
		},
		"reproducción musical visible": func() string {
			return "/api/v1/music/tracks/" + strconv.FormatInt(fixture.visibleTrack, 10) + "/play"
		},
		"navegar clips visibles": func() string {
			return "/api/v1/libraries/" + strconv.FormatInt(fixture.visibleClips, 10) + "/browse"
		},
		"navegar clips ocultos": func() string {
			return "/api/v1/libraries/" + strconv.FormatInt(fixture.hiddenClips, 10) + "/browse"
		},
		"stream de música sin firma": func() string {
			return "/api/v1/stream/music/" + strconv.FormatInt(fixture.hiddenTrack, 10)
		},
		"stream directo sin firma": func() string {
			return "/api/v1/stream/" + strconv.FormatInt(fixture.hiddenMovie, 10)
		},
		"archivo de stream sin firma": func() string {
			return "/api/v1/stream/" + strconv.FormatInt(fixture.hiddenMovie, 10) + "/playlist.m3u8"
		},
		"subtítulos de stream sin firma": func() string {
			return "/api/v1/stream/" + strconv.FormatInt(fixture.hiddenMovie, 10) + "/subtitles/es0.vtt"
		},
	}

	cases := leakRoutes()
	for i := range cases {
		if build, ok := paths[cases[i].name]; ok {
			cases[i].path = build()
		}
	}

	covered := make(map[string]bool)
	for _, item := range cases {
		covered[item.pattern] = true
		t.Run(item.name, func(t *testing.T) {
			method := item.method
			if method == "" {
				method = http.MethodGet
			}
			var status int
			var body []byte
			if method == http.MethodGet {
				status, body = libraryHTTPGet(t, ts.URL+item.path, userToken)
			} else {
				status, body = libraryHTTPDo(t, method, ts.URL+item.path, userToken, item.body)
			}
			want := item.wantStatus
			if want == 0 {
				if item.notFound {
					want = http.StatusNotFound
				} else {
					want = http.StatusOK
				}
			}
			if status != want {
				t.Fatalf("usuario: código = %d, se esperaba %d; cuerpo = %s", status, want, body)
			}
			if bytes.Contains(body, []byte(leakMarker)) {
				t.Fatalf("fuga en %s: %s", item.path, body)
			}
			if item.wantURL && !bytes.Contains(body, []byte("/api/v1/stream/")) {
				t.Fatalf("se esperaba una URL firmada en %s: %s", item.path, body)
			}
			if item.adminMarca && method == http.MethodGet {
				adminStatus, adminBody := libraryHTTPGet(t, ts.URL+item.path, adminToken)
				if adminStatus != http.StatusOK {
					t.Fatalf("admin: código = %d; cuerpo = %s", adminStatus, adminBody)
				}
				if !bytes.Contains(adminBody, []byte(leakMarker)) {
					t.Fatalf("el admin debería ver la marca en %s: %s", item.path, adminBody)
				}
			}
		})
	}

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/people/777", userToken)
	if status != http.StatusOK {
		t.Fatalf("persona parcial: %d %s", status, body)
	}
	var personResponse personMediaResponse
	if err := json.Unmarshal(body, &personResponse); err != nil {
		t.Fatalf("persona parcial: %v", err)
	}
	for _, item := range personResponse.InLibrary {
		if item.ID == fixture.hiddenMovie || item.ID == fixture.hiddenSeries {
			t.Fatalf("la persona no debe incluir contenido oculto: %+v", item)
		}
	}
	notInLibrary := make(map[int64]bool)
	for _, item := range personResponse.NotInLibrary {
		notInLibrary[item.TMDBID] = true
	}
	if !notInLibrary[1002] || !notInLibrary[2002] {
		t.Fatalf("las obras solo ocultas deben seguir en not_in_library: %+v", personResponse.NotInLibrary)
	}

	allToken := leakAllLibrariesToken(t, ts)
	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items", allToken)
	if status != http.StatusOK || !bytes.Contains(body, []byte(leakMarker)) {
		t.Fatalf("un usuario con all_libraries debe ver la marca: código = %d, cuerpo = %s", status, body)
	}

	nobodyToken := leakNoLibrariesToken(t, ts)
	fixtureTitles := []string{
		"Película visible", "Vídeo visible", "Serie visible", "Episodio visible",
		"Colección visible", "Artista visible", "Álbum visible", "Pista visible",
	}
	for _, item := range cases {
		method := item.method
		if method == "" {
			method = http.MethodGet
		}
		if method != http.MethodGet {
			continue
		}
		status, body := libraryHTTPGet(t, ts.URL+item.path, nobodyToken)
		if status != http.StatusOK && status != http.StatusNotFound && status != http.StatusForbidden {
			t.Errorf("sin bibliotecas, %s: código = %d; cuerpo = %s", item.name, status, body)
		}
		if bytes.Contains(body, []byte(leakMarker)) {
			t.Errorf("sin bibliotecas, fuga en %s: %s", item.name, body)
		}
		for _, title := range fixtureTitles {
			if bytes.Contains(body, []byte(title)) {
				t.Errorf("sin bibliotecas, %s lista %q: %s", item.name, title, body)
			}
		}
	}

	var count int
	for table, query := range map[string]string{
		"playback_progress": `SELECT count(*) FROM playback_progress WHERE user_id = ? AND media_item_id = ?`,
		"watch_state":       `SELECT count(*) FROM watch_state WHERE user_id = ? AND media_item_id = ?`,
		"item_lists":        `SELECT count(*) FROM item_lists WHERE user_id = ? AND media_item_id = ?`,
		"item_preferences":  `SELECT count(*) FROM item_preferences WHERE user_id = ? AND media_item_id = ?`,
	} {
		if err := ts.db.QueryRow(query, fixture.userID, fixture.hiddenEpisode2).Scan(&count); err != nil {
			t.Fatalf("contar %s del ítem oculto: %v", table, err)
		}
		if count != 1 {
			t.Errorf("los PUT/DELETE ocultos modificaron %s: quedan %d filas", table, count)
		}
	}
	var position float64
	if err := ts.db.QueryRow(`
		SELECT position_seconds FROM playback_progress
		 WHERE user_id = ? AND media_item_id = ?`, fixture.userID, fixture.hiddenEpisode2).
		Scan(&position); err != nil {
		t.Fatalf("leer progreso oculto: %v", err)
	}
	if position != 12 {
		t.Errorf("el progreso oculto cambió a %v", position)
	}
	var quality sql.NullString
	if err := ts.db.QueryRow(`
		SELECT quality FROM item_preferences
		 WHERE user_id = ? AND media_item_id = ?`, fixture.userID, fixture.hiddenEpisode2).
		Scan(&quality); err != nil {
		t.Fatalf("leer preferencias ocultas: %v", err)
	}
	if !quality.Valid || quality.String != "720" {
		t.Errorf("las preferencias ocultas cambiaron: %+v", quality)
	}

	checkLeakRouteCoverage(t, covered)
}

func checkLeakRouteCoverage(t *testing.T, covered map[string]bool) {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no se pudo localizar leak_test.go")
	}
	dir := filepath.Dir(current)
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("listar archivos de library: %v", err)
	}
	pattern := regexp.MustCompile(`mux\.Handle\("GET ([^"]+)"`)
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("leer %s: %v", filepath.Base(file), err)
		}
		for _, line := range strings.Split(string(source), "\n") {
			if !strings.Contains(line, "requireSession") || strings.Contains(line, "requireAdmin") {
				continue
			}
			match := pattern.FindStringSubmatch(line)
			if len(match) != 2 {
				continue
			}
			if !covered[match[1]] {
				t.Errorf("la ruta GET %s de %s no está cubierta por la prueba de fugas",
					match[1], filepath.Base(file))
			}
		}
	}
}
