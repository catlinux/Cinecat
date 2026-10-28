package library

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/catlinux/cinecat/internal/auth"
)

const leakMarker = "OCULTO-7f3a"

type leakFixture struct {
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
	pending    string
	exempt     string
	notFound   bool
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
	f.visibleVideo, _ = leakLibrary(t, ts.db, "/media/visible-video", "Vídeo visible", "video")
	f.hiddenVideo, _ = leakLibrary(t, ts.db, "/media/OCULTO-7f3a-video", leakMarker+" vídeo", "video")
	f.visibleClips, _ = leakLibrary(t, ts.db, "/media/visible-clips", "Clips visibles", "clips")
	f.hiddenClips, _ = leakLibrary(t, ts.db, "/media/OCULTO-7f3a-clips", leakMarker+" clips", "clips")
	f.visibleMusic, _ = leakLibrary(t, ts.db, "/media/visible-music", "Música visible", "music")
	f.hiddenMusic, _ = leakLibrary(t, ts.db, "/media/OCULTO-7f3a-music", leakMarker+" música", "music")

	f.visibleMovie = leakItem(t, ts.db, f.visibleVideo,
		"/media/visible-video/pelicula-visible.mkv", "Película visible", "movie", `["Drama"]`, nil, nil, nil)
	f.hiddenMovie = leakItem(t, ts.db, f.hiddenVideo,
		"/media/OCULTO-7f3a-video/OCULTO-7f3a-pelicula.mkv", leakMarker+" película", "movie",
		`["Drama","OCULTO-7f3a"]`, nil, nil, nil)
	f.visibleClip = leakItem(t, ts.db, f.visibleClips,
		"/media/visible-clips/video-visible.mkv", "Vídeo visible", "movie", `[]`, nil, nil, nil)
	f.hiddenClip = leakItem(t, ts.db, f.hiddenClips,
		"/media/OCULTO-7f3a-clips/OCULTO-7f3a-video.mkv", leakMarker+" vídeo", "movie", `[]`, nil, nil, nil)

	f.visibleSeries = leakExec(t, ts.db, `
		INSERT INTO series (library_id, folder, title, display_title, genres, poster_url, match_status)
		VALUES (?, ?, ?, ?, '["Drama"]', 'https://img/visible.jpg', 'auto')`,
		f.visibleVideo, "visible-serie", "Serie visible", "Serie visible")
	f.hiddenSeries = leakExec(t, ts.db, `
		INSERT INTO series (library_id, folder, title, display_title, genres, poster_url, match_status)
		VALUES (?, ?, ?, ?, '["OCULTO-7f3a"]', 'https://img/OCULTO-7f3a.jpg', 'auto')`,
		f.hiddenVideo, leakMarker+"-serie", leakMarker+" serie", leakMarker+" serie")
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
	f.visibleTrack = leakExec(t, ts.db, `
		INSERT INTO music_tracks (library_id, album_id, artist_id, path, title, size_bytes, mtime_unix)
		VALUES (?, ?, ?, ?, ?, 100, 1)`,
		f.visibleMusic, f.visibleAlbum, f.visibleArtist,
		"/media/visible-music/pista-visible.flac", "Pista visible")
	f.hiddenTrack = leakExec(t, ts.db, `
		INSERT INTO music_tracks (library_id, album_id, artist_id, path, title, size_bytes, mtime_unix)
		VALUES (?, ?, ?, ?, ?, 100, 1)`,
		f.hiddenMusic, f.hiddenAlbum, f.hiddenArtist,
		"/media/OCULTO-7f3a-music/OCULTO-7f3a-pista.flac", leakMarker+" pista")

	hash, err := auth.HashPassword("Clave-user-1")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	userID := leakExec(t, ts.db, `
		INSERT INTO users (username, password_hash, role, all_libraries)
		VALUES ('user', ?, 'user', 0)`, hash)
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

func leakRoutes() []leakCase {
	// Los paths se rellenan en el test, cuando ya existen los ids del fixture.
	return []leakCase{
		{name: "items", pattern: "/api/v1/items", path: "/api/v1/items", adminMarca: true},
		{name: "items búsqueda oculta", pattern: "/api/v1/items", path: "/api/v1/items?q=" + leakMarker},
		{name: "items biblioteca oculta", pattern: "/api/v1/items", path: "/api/v1/items?library_id="},
		{name: "items similares de oculto", pattern: "/api/v1/items/{id}/similar", notFound: true},
		{name: "items similares de visible", pattern: "/api/v1/items/{id}/similar"},
		{name: "items oculto", pattern: "/api/v1/items/{id}", pending: "U2c"},
		{name: "items visible", pattern: "/api/v1/items/{id}", pending: "U2c"},
		{name: "items miniatura oculta", pattern: "/api/v1/items/{id}/thumbnail", pending: "U2c"},
		{name: "items miniatura visible", pattern: "/api/v1/items/{id}/thumbnail", pending: "U2c"},
		{name: "items file-info oculto", pattern: "/api/v1/items/{id}/file-info", pending: "U2c"},
		{name: "items file-info visible", pattern: "/api/v1/items/{id}/file-info", pending: "U2c"},
		{name: "items preferencias ocultas", pattern: "/api/v1/items/{id}/preferences", pending: "U2c"},
		{name: "items preferencias visibles", pattern: "/api/v1/items/{id}/preferences", pending: "U2c"},
		{name: "items reproducción oculta", pattern: "/api/v1/items/{id}/play", pending: "U2c"},
		{name: "items reproducción visible", pattern: "/api/v1/items/{id}/play", pending: "U2c"},
		{name: "géneros", pattern: "/api/v1/genres", path: "/api/v1/genres", adminMarca: true},
		{name: "portal", pattern: "/api/v1/home", path: "/api/v1/home", adminMarca: true},
		{name: "favoritos", pattern: "/api/v1/lists/{list}", path: "/api/v1/lists/favorite?kind=items"},
		{name: "próximas", pattern: "/api/v1/lists/{list}", path: "/api/v1/lists/watchlist?kind=items"},
		{name: "colecciones", pattern: "/api/v1/collections", pending: "U2b"},
		{name: "colección oculta", pattern: "/api/v1/collections/{id}", pending: "U2b"},
		{name: "colección visible", pattern: "/api/v1/collections/{id}", pending: "U2b"},
		{name: "persona oculta", pattern: "/api/v1/people/{id}", path: "/api/v1/people/1", pending: "U2b"},
		{name: "persona visible", pattern: "/api/v1/people/{id}", path: "/api/v1/people/2", pending: "U2b"},
		{name: "series", pattern: "/api/v1/series", pending: "U2b"},
		{name: "géneros de series", pattern: "/api/v1/series/genres", pending: "U2b"},
		{name: "serie oculta", pattern: "/api/v1/series/{id}", pending: "U2b"},
		{name: "serie visible", pattern: "/api/v1/series/{id}", pending: "U2b"},
		{name: "similares de serie oculta", pattern: "/api/v1/series/{id}/similar", pending: "U2b"},
		{name: "similares de serie visible", pattern: "/api/v1/series/{id}/similar", pending: "U2b"},
		{name: "temporada de serie oculta", pattern: "/api/v1/series/{id}/seasons/{n}", pending: "U2b"},
		{name: "temporada de serie visible", pattern: "/api/v1/series/{id}/seasons/{n}", pending: "U2b"},
		{name: "artistas", pattern: "/api/v1/music/artists", pending: "U2c"},
		{name: "artista oculto", pattern: "/api/v1/music/artists/{id}", pending: "U2c"},
		{name: "artista visible", pattern: "/api/v1/music/artists/{id}", pending: "U2c"},
		{name: "álbumes", pattern: "/api/v1/music/albums", pending: "U2c"},
		{name: "álbum oculto", pattern: "/api/v1/music/albums/{id}", pending: "U2c"},
		{name: "álbum visible", pattern: "/api/v1/music/albums/{id}", pending: "U2c"},
		{name: "carátula oculta", pattern: "/api/v1/music/albums/{id}/cover", pending: "U2c"},
		{name: "carátula visible", pattern: "/api/v1/music/albums/{id}/cover", pending: "U2c"},
		{name: "pistas", pattern: "/api/v1/music/tracks", pending: "U2c"},
		{name: "reproducción musical oculta", pattern: "/api/v1/music/tracks/{id}/play", pending: "U2c"},
		{name: "reproducción musical visible", pattern: "/api/v1/music/tracks/{id}/play", pending: "U2c"},
		{name: "bibliotecas", pattern: "/api/v1/libraries", path: "/api/v1/libraries", adminMarca: true},
		{name: "navegar clips visibles", pattern: "/api/v1/libraries/{id}/browse"},
		{name: "navegar clips ocultos", pattern: "/api/v1/libraries/{id}/browse", notFound: true},
		{name: "ajustes públicos", pattern: "/api/v1/settings/public", path: "/api/v1/settings/public", exempt: "sin contenido de bibliotecas"},
		{name: "stream de música", pattern: "/api/v1/stream/music/{id}", exempt: "stream firmado sin sesión"},
		{name: "stream directo", pattern: "/api/v1/stream/{id}", exempt: "stream firmado sin sesión"},
		{name: "archivo de stream", pattern: "/api/v1/stream/{id}/{file}", exempt: "stream firmado sin sesión"},
		{name: "subtítulos de stream", pattern: "/api/v1/stream/{id}/subtitles/{file}", exempt: "stream firmado sin sesión"},
	}
}

func TestPruebaDeFugas(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
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
			return "/api/v1/items/" + strconv.FormatInt(fixture.hiddenMovie, 10) + "/thumbnail"
		},
		"items miniatura visible": func() string {
			return "/api/v1/items/" + strconv.FormatInt(fixture.visibleMovie, 10) + "/thumbnail"
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
		"colección oculta": func() string {
			return "/api/v1/collections/" + strconv.FormatInt(fixture.hiddenCollection, 10)
		},
		"colección visible": func() string {
			return "/api/v1/collections/" + strconv.FormatInt(fixture.visibleCollection, 10)
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
	}

	cases := leakRoutes()
	for i := range cases {
		if cases[i].path == "" {
			if build, ok := paths[cases[i].name]; ok {
				cases[i].path = build()
			}
		}
	}

	covered := make(map[string]bool)
	for _, item := range cases {
		covered[item.pattern] = true
		if item.pending != "" {
			t.Run(item.name, func(t *testing.T) {
				t.Skip("pendiente " + item.pending)
			})
			continue
		}
		if item.exempt != "" {
			t.Run(item.name, func(t *testing.T) {
				t.Skip("exento: " + item.exempt)
			})
			continue
		}
		t.Run(item.name, func(t *testing.T) {
			status, body := libraryHTTPGet(t, ts.URL+item.path, userToken)
			if item.notFound {
				if status != http.StatusNotFound {
					t.Fatalf("usuario: código = %d, se esperaba 404; cuerpo = %s", status, body)
				}
			} else if status != http.StatusOK {
				t.Fatalf("usuario: código = %d; cuerpo = %s", status, body)
			}
			if bytes.Contains(body, []byte(leakMarker)) {
				t.Fatalf("fuga en %s: %s", item.path, body)
			}
			if item.adminMarca {
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

	allToken := leakAllLibrariesToken(t, ts)
	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items", allToken)
	if status != http.StatusOK || !bytes.Contains(body, []byte(leakMarker)) {
		t.Fatalf("un usuario con all_libraries debe ver la marca: código = %d, cuerpo = %s", status, body)
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
