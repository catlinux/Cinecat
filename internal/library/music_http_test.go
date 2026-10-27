package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/catlinux/cinecat/internal/playback"
)

func newMusicAPITestServer(t *testing.T) *libraryTestServer {
	t.Helper()
	ts := newLibraryHTTPTestServer(t)
	svc, err := playback.New(playback.Options{TTL: time.Hour, Key: []byte("clave-de-prueba")})
	if err != nil {
		t.Fatalf("playback.New: %v", err)
	}
	ts.api.playback = svc
	t.Cleanup(func() { _ = svc.Close() })
	return ts
}

func musicTestGET(t *testing.T, url, token string) (int, []byte, http.Header) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leer respuesta: %v", err)
	}
	return resp.StatusCode, body, resp.Header
}

type insertSeedMusicRow struct {
	Artist   string
	Album    string
	Title    string
	Path     string
	Cover    string
	Codec    string
	Disc     int
	Track    int
	Channels int
	Embedded bool
	Year     int
}

func insertSeedMusic(t *testing.T, db *sql.DB, libraryID int64, track insertSeedMusicRow) (artistID, albumID, trackID int64) {
	t.Helper()
	if track.Codec == "" {
		track.Codec = "mp3"
	}
	if track.Disc <= 0 {
		track.Disc = 1
	}
	if track.Channels <= 0 {
		track.Channels = 2
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO music_artists (name, sort_name) VALUES (?, ?)`,
		track.Artist, musicSortName(track.Artist)); err != nil {
		t.Fatalf("crear artista: %v", err)
	}
	if err := db.QueryRow(`SELECT id FROM music_artists WHERE name = ? COLLATE NOCASE`, track.Artist).Scan(&artistID); err != nil {
		t.Fatalf("leer artista: %v", err)
	}
	var cover any
	if track.Cover != "" {
		cover = track.Cover
	}
	if _, err := db.Exec(`INSERT OR IGNORE INTO music_albums (library_id, artist_id, title, year, cover_path) VALUES (?, ?, ?, ?, ?)`,
		libraryID, artistID, track.Album, nullableInt(track.Year), cover); err != nil {
		t.Fatalf("crear álbum: %v", err)
	}
	if err := db.QueryRow(`SELECT id FROM music_albums WHERE library_id = ? AND artist_id = ? AND title = ? COLLATE NOCASE`,
		libraryID, artistID, track.Album).Scan(&albumID); err != nil {
		t.Fatalf("leer álbum: %v", err)
	}
	if track.Cover != "" {
		if _, err := db.Exec(`UPDATE music_albums SET cover_path = ? WHERE id = ?`, track.Cover, albumID); err != nil {
			t.Fatalf("guardar carátula: %v", err)
		}
	}
	info, err := os.Stat(track.Path)
	size := int64(0)
	mtime := time.Now().Unix()
	if err == nil {
		size = info.Size()
		mtime = info.ModTime().Unix()
	}
	res, err := db.Exec(`
		INSERT INTO music_tracks (
			library_id, album_id, artist_id, path, title, track_no, disc_no,
			year, duration_seconds, codec, bitrate, sample_rate, channels,
			size_bytes, mtime_unix, has_embedded_cover
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 180, ?, 320000, 44100, ?, ?, ?, ?)`,
		libraryID, albumID, artistID, track.Path, track.Title, nullableInt(track.Track),
		track.Disc, nullableInt(track.Year), track.Codec, track.Channels, size, mtime,
		boolInt(track.Embedded))
	if err != nil {
		t.Fatalf("crear pista: %v", err)
	}
	trackID, _ = res.LastInsertId()
	return artistID, albumID, trackID
}

func createDirectMusicLibrary(t *testing.T, db *sql.DB, root string) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO libraries (path, name, kind) VALUES (?, ?, 'music')`, root, "Música")
	if err != nil {
		t.Fatalf("crear biblioteca de música: %v", err)
	}
	id, _ := res.LastInsertId()
	addScannerLibraryPath(t, db, id, root)
	return id
}

func TestMusicAPIArtistasAlbumesPistasBusquedaYOrden(t *testing.T) {
	ts := newMusicAPITestServer(t)
	root := t.TempDir()
	libraryID := createDirectMusicLibrary(t, ts.db, root)
	beatles, abbey, _ := insertSeedMusic(t, ts.db, libraryID, insertSeedMusicRow{
		Artist: "The Beatles", Album: "Abbey Road", Title: "Come Together", Track: 1, Year: 1969,
		Path: filepath.Join(root, "The Beatles", "Abbey Road", "01.mp3"),
	})
	_, _, _ = insertSeedMusic(t, ts.db, libraryID, insertSeedMusicRow{
		Artist: "The Beatles", Album: "Abbey Road", Title: "Something", Track: 2, Year: 1969,
		Path: filepath.Join(root, "The Beatles", "Abbey Road", "02.mp3"),
	})
	oscar, _, _ := insertSeedMusic(t, ts.db, libraryID, insertSeedMusicRow{
		Artist: "Óscar", Album: "Canciones", Title: "Una canción", Track: 1, Year: 2020,
		Path: filepath.Join(root, "Óscar", "Canciones", "01.mp3"),
	})
	_, _, _ = insertSeedMusic(t, ts.db, libraryID, insertSeedMusicRow{
		Artist: "Varios artistas", Album: "Recopilatorio", Title: "Otra", Disc: 2, Track: 1,
		Path: filepath.Join(root, "Recopilatorio", "01.mp3"),
	})

	status, body, _ := musicTestGET(t, ts.URL+"/api/v1/music/artists", ts.token)
	if status != http.StatusOK {
		t.Fatalf("artistas: %d %s", status, body)
	}
	var artists musicArtistsResponse
	if err := json.Unmarshal(body, &artists); err != nil {
		t.Fatalf("artistas JSON: %v", err)
	}
	if artists.Total != 3 || len(artists.Artists) != 3 || artists.Artists[0].Name != "The Beatles" {
		t.Fatalf("artistas inesperados: %+v", artists)
	}
	if artists.Artists[0].AlbumCount != 1 || artists.Artists[0].TrackCount != 2 {
		t.Fatalf("contadores inesperados: %+v", artists.Artists[0])
	}

	status, body, _ = musicTestGET(t, ts.URL+"/api/v1/music/artists?letter=O", ts.token)
	if status != http.StatusOK {
		t.Fatalf("artistas por letra: %d %s", status, body)
	}
	if err := json.Unmarshal(body, &artists); err != nil || artists.Total != 1 || artists.Artists[0].ID != oscar {
		t.Fatalf("letra O: %+v %v", artists, err)
	}

	status, body, _ = musicTestGET(t, ts.URL+"/api/v1/music/artists?limit=1&offset=1", ts.token)
	if status != http.StatusOK {
		t.Fatalf("paginación de artistas: %d %s", status, body)
	}
	if err := json.Unmarshal(body, &artists); err != nil || artists.Total != 3 || len(artists.Artists) != 1 || artists.Limit != 1 || artists.Offset != 1 {
		t.Fatalf("paginación: %+v %v", artists, err)
	}

	status, body, _ = musicTestGET(t, ts.URL+"/api/v1/music/artists/"+strconv.FormatInt(beatles, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("detalle de artista: %d %s", status, body)
	}
	var artistDetail musicArtistDetailDTO
	if err := json.Unmarshal(body, &artistDetail); err != nil || artistDetail.Name != "The Beatles" || len(artistDetail.Albums) != 1 {
		t.Fatalf("detalle de artista: %+v %v", artistDetail, err)
	}

	status, body, _ = musicTestGET(t, ts.URL+"/api/v1/music/albums?sort=artist&artist="+strconv.FormatInt(beatles, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("álbumes: %d %s", status, body)
	}
	var albums musicAlbumsResponse
	if err := json.Unmarshal(body, &albums); err != nil || albums.Total != 1 || len(albums.Albums) != 1 {
		t.Fatalf("álbumes: %+v %v", albums, err)
	}
	if albums.Albums[0].Title != "Abbey Road" || albums.Albums[0].Artist != "The Beatles" ||
		albums.Albums[0].TrackCount != 2 || albums.Albums[0].DurationSeconds != 360 {
		t.Fatalf("álbum inesperado: %+v", albums.Albums[0])
	}
	if albums.Albums[0].Year == nil || *albums.Albums[0].Year != 1969 {
		t.Fatalf("año inesperado: %+v", albums.Albums[0])
	}

	status, body, _ = musicTestGET(t, ts.URL+"/api/v1/music/albums/"+strconv.FormatInt(abbey, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("detalle de álbum: %d %s", status, body)
	}
	var albumDetail musicAlbumDetailDTO
	if err := json.Unmarshal(body, &albumDetail); err != nil || len(albumDetail.Tracks) != 2 {
		t.Fatalf("detalle de álbum: %+v %v", albumDetail, err)
	}
	if albumDetail.Tracks[0].Title != "Come Together" || albumDetail.Tracks[0].TrackNo == nil ||
		*albumDetail.Tracks[0].TrackNo != 1 || !albumDetail.Tracks[0].Playable {
		t.Fatalf("pista inesperada: %+v", albumDetail.Tracks[0])
	}

	status, body, _ = musicTestGET(t, ts.URL+"/api/v1/music/tracks?q=Abbey", ts.token)
	if status != http.StatusOK {
		t.Fatalf("buscar pistas por álbum: %d %s", status, body)
	}
	var tracks musicTracksResponse
	if err := json.Unmarshal(body, &tracks); err != nil || tracks.Total != 2 {
		t.Fatalf("pistas por álbum: %+v %v", tracks, err)
	}
	status, body, _ = musicTestGET(t, ts.URL+"/api/v1/music/tracks?q=beatles", ts.token)
	if status != http.StatusOK || !strings.Contains(string(body), `"total":2`) {
		t.Fatalf("buscar pistas por artista: %d %s", status, body)
	}

	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/music/artists", ""); status != http.StatusUnauthorized {
		t.Fatalf("sin sesión: %d", status)
	}
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/music/albums?sort=raro", ts.token); status != http.StatusBadRequest {
		t.Fatalf("orden inválido: %d", status)
	}
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/music/albums/999", ts.token); status != http.StatusNotFound {
		t.Fatalf("álbum inexistente: %d", status)
	}
}

func TestMusicCoverCarpetaIncrustadaCacheY404(t *testing.T) {
	ts := newMusicAPITestServer(t)
	root := t.TempDir()
	libraryID := createDirectMusicLibrary(t, ts.db, root)
	coversDir := t.TempDir()
	runnerCalls := 0
	ts.api.WithMusicCovers(coversDir).WithMusicCoverRunner(func(ctx context.Context, args []string) error {
		runnerCalls++
		return os.WriteFile(args[len(args)-1], []byte{0xff, 0xd8, 0xff, 0xd9}, 0o644)
	})

	folderCover := filepath.Join(root, "Artista", "Álbum", "cover.jpg")
	if err := os.MkdirAll(filepath.Dir(folderCover), 0o755); err != nil {
		t.Fatalf("crear carpeta: %v", err)
	}
	if err := os.WriteFile(folderCover, []byte{0xff, 0xd8, 0xff, 0x00}, 0o644); err != nil {
		t.Fatalf("crear carátula: %v", err)
	}
	_, folderAlbumID, _ := insertSeedMusic(t, ts.db, libraryID, insertSeedMusicRow{
		Artist: "Artista", Album: "Álbum", Title: "Tema", Path: filepath.Join(root, "Artista", "Álbum", "01.mp3"),
		Cover: folderCover,
	})
	status, body, headers := musicTestGET(t, ts.URL+"/api/v1/music/albums/"+strconv.FormatInt(folderAlbumID, 10)+"/cover", ts.token)
	if status != http.StatusOK || len(body) == 0 || headers.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("carátula de carpeta: %d %s", status, body)
	}
	if headers.Get("Cache-Control") != "private, max-age=86400" || headers.Get("ETag") == "" {
		t.Fatalf("cabeceras de carátula: %+v", headers)
	}
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/music/albums/"+strconv.FormatInt(folderAlbumID, 10)+"/cover", nil)
	req.Header.Set("Authorization", "Bearer "+ts.token)
	req.Header.Set("If-None-Match", headers.Get("ETag"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET con ETag: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("ETag: %d", resp.StatusCode)
	}

	embeddedPath := filepath.Join(root, "Otro", "Álbum", "02.mp3")
	writeFile(t, embeddedPath)
	_, embeddedAlbumID, _ := insertSeedMusic(t, ts.db, libraryID, insertSeedMusicRow{
		Artist: "Otro", Album: "Álbum", Title: "Con portada", Path: embeddedPath, Embedded: true,
	})
	status, body, headers = musicTestGET(t, ts.URL+"/api/v1/music/albums/"+strconv.FormatInt(embeddedAlbumID, 10)+"/cover", ts.token)
	if status != http.StatusOK || len(body) != 4 || headers.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("carátula incrustada: %d %s", status, body)
	}
	if runnerCalls != 1 {
		t.Fatalf("llamadas a FFmpeg = %d, se esperaba 1", runnerCalls)
	}
	status, _, _ = musicTestGET(t, ts.URL+"/api/v1/music/albums/"+strconv.FormatInt(embeddedAlbumID, 10)+"/cover", ts.token)
	if status != http.StatusOK || runnerCalls != 1 {
		t.Fatalf("la segunda carátula debe venir de caché: %d, llamadas=%d", status, runnerCalls)
	}

	_, sinCoverID, _ := insertSeedMusic(t, ts.db, libraryID, insertSeedMusicRow{
		Artist: "Sin", Album: "Portada", Title: "Tema", Path: filepath.Join(root, "Sin", "Portada", "01.mp3"),
	})
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/music/albums/"+strconv.FormatInt(sinCoverID, 10)+"/cover", ts.token); status != http.StatusNotFound {
		t.Fatalf("sin carátula: %d", status)
	}

	outside := filepath.Join(t.TempDir(), "cover.jpg")
	if err := os.WriteFile(outside, []byte("fuera"), 0o644); err != nil {
		t.Fatalf("crear carátula externa: %v", err)
	}
	_, outsideID, _ := insertSeedMusic(t, ts.db, libraryID, insertSeedMusicRow{
		Artist: "Fuera", Album: "Fuera", Title: "Tema", Path: filepath.Join(root, "Fuera", "01.mp3"),
		Cover: outside,
	})
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/music/albums/"+strconv.FormatInt(outsideID, 10)+"/cover", ts.token); status != http.StatusNotFound {
		t.Fatalf("carátula fuera de la biblioteca: %d", status)
	}
}

func TestMusicPlayYStreamFirmadoConRangos(t *testing.T) {
	ts := newMusicAPITestServer(t)
	root := t.TempDir()
	libraryID := createDirectMusicLibrary(t, ts.db, root)
	audio := filepath.Join(root, "Artista", "Álbum", "tema.mp3")
	if err := os.MkdirAll(filepath.Dir(audio), 0o755); err != nil {
		t.Fatalf("crear carpeta: %v", err)
	}
	content := []byte("0123456789abcdef")
	if err := os.WriteFile(audio, content, 0o644); err != nil {
		t.Fatalf("crear audio: %v", err)
	}
	_, _, trackID := insertSeedMusic(t, ts.db, libraryID, insertSeedMusicRow{
		Artist: "Artista", Album: "Álbum", Title: "Tema", Path: audio,
	})

	status, body, _ := musicTestGET(t, ts.URL+"/api/v1/music/tracks/"+strconv.FormatInt(trackID, 10)+"/play", ts.token)
	if status != http.StatusOK {
		t.Fatalf("play: %d %s", status, body)
	}
	var play musicPlayResponse
	if err := json.Unmarshal(body, &play); err != nil {
		t.Fatalf("play JSON: %v", err)
	}
	if play.URL == "" || play.MIME != "audio/mpeg" || !play.Playable || play.ExpiresAt.IsZero() {
		t.Fatalf("respuesta de reproducción: %+v", play)
	}

	req, _ := http.NewRequest(http.MethodGet, ts.URL+play.URL, nil)
	req.Header.Set("Range", "bytes=0-3")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	defer resp.Body.Close()
	streamed, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusPartialContent || string(streamed) != "0123" {
		t.Fatalf("rango: %d %q", resp.StatusCode, streamed)
	}
	if resp.Header.Get("Content-Type") != "audio/mpeg" || resp.Header.Get("Content-Range") == "" {
		t.Fatalf("cabeceras de stream: %+v", resp.Header)
	}

	tampered := play.URL + "x"
	if status, _, _ := musicTestGET(t, ts.URL+tampered, ""); status != http.StatusForbidden {
		t.Fatalf("firma manipulada: %d", status)
	}
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/stream/music/"+strconv.FormatInt(trackID, 10), ""); status != http.StatusForbidden {
		t.Fatalf("sin firma: %d", status)
	}

	wav := filepath.Join(root, "Artista", "Álbum", "multicanal.wav")
	writeFile(t, wav)
	_, _, wavID := insertSeedMusic(t, ts.db, libraryID, insertSeedMusicRow{
		Artist: "Artista", Album: "Álbum", Title: "WAV", Path: wav, Codec: "pcm_s24le", Channels: 6,
	})
	status, body, _ = musicTestGET(t, ts.URL+"/api/v1/music/tracks/"+strconv.FormatInt(wavID, 10)+"/play", ts.token)
	if status != http.StatusOK {
		t.Fatalf("play WAV: %d %s", status, body)
	}
	if err := json.Unmarshal(body, &play); err != nil || play.Playable {
		t.Fatalf("WAV multicanal no debe ser reproducible: %+v %v", play, err)
	}

	outside := filepath.Join(t.TempDir(), "fuera.mp3")
	writeFile(t, outside)
	_, _, outsideID := insertSeedMusic(t, ts.db, libraryID, insertSeedMusicRow{
		Artist: "Fuera", Album: "Fuera", Title: "Fuera", Path: outside,
	})
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/music/tracks/"+strconv.FormatInt(outsideID, 10)+"/play", ts.token); status != http.StatusNotFound {
		t.Fatalf("pista fuera de la biblioteca: %d", status)
	}
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/music/tracks/999/play", ts.token); status != http.StatusNotFound {
		t.Fatalf("pista inexistente: %d", status)
	}
}

func TestMusicItemCountHomeYBorradoConLimpieza(t *testing.T) {
	ts := newMusicAPITestServer(t)
	root := t.TempDir()
	makeDirs(t, root, "a", "b")
	ts.api.WithMediaRoots([]string{root})
	library := createLibraryViaAPI(t, ts, map[string]any{
		"name": "Música", "kind": "music",
		"paths": []string{filepath.Join(root, "a"), filepath.Join(root, "b")},
	})
	pathA, pathB := library.Paths[0].ID, library.Paths[1].ID
	_, albumA, _ := insertSeedMusic(t, ts.db, library.ID, insertSeedMusicRow{
		Artist: "Artista A", Album: "Álbum A", Title: "A", Path: filepath.Join(root, "a", "01.mp3"),
	})
	_, _, _ = insertSeedMusic(t, ts.db, library.ID, insertSeedMusicRow{
		Artist: "Artista B", Album: "Álbum B", Title: "B", Path: filepath.Join(root, "b", "01.mp3"),
	})

	status, body, _ := musicTestGET(t, ts.URL+"/api/v1/libraries", ts.token)
	if status != http.StatusOK {
		t.Fatalf("bibliotecas: %d %s", status, body)
	}
	var libraries librariesResponse
	if err := json.Unmarshal(body, &libraries); err != nil {
		t.Fatalf("bibliotecas JSON: %v", err)
	}
	if len(libraries.Libraries) != 1 || libraries.Libraries[0].ItemCount != 2 {
		t.Fatalf("item_count de música: %+v", libraries.Libraries)
	}
	status, body, _ = musicTestGET(t, ts.URL+"/api/v1/home", ts.token)
	if status != http.StatusOK {
		t.Fatalf("home: %d %s", status, body)
	}
	var home homeResponse
	if err := json.Unmarshal(body, &home); err != nil || !home.HasMusic {
		t.Fatalf("has_music: %+v %v", home, err)
	}

	status, body = libraryJSONDo(t, http.MethodDelete,
		ts.URL+"/api/v1/libraries/"+strconv.FormatInt(library.ID, 10)+"/paths/"+strconv.FormatInt(pathA, 10),
		ts.token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("retirar carpeta: %d %s", status, body)
	}
	var count int
	if err := ts.db.QueryRow(`SELECT count(*) FROM music_tracks WHERE album_id = ?`, albumA).Scan(&count); err != nil || count != 0 {
		t.Fatalf("la pista de A debe borrarse: %d %v", count, err)
	}
	if err := ts.db.QueryRow(`SELECT count(*) FROM music_artists WHERE name = 'Artista A'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("el artista A debe borrarse: %d %v", count, err)
	}
	if err := ts.db.QueryRow(`SELECT count(*) FROM music_tracks`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("debe quedar la pista B: %d %v", count, err)
	}
	if pathB <= 0 {
		t.Fatal("la segunda carpeta debe tener id")
	}

	status, body = libraryJSONDo(t, http.MethodDelete,
		ts.URL+"/api/v1/libraries/"+strconv.FormatInt(library.ID, 10), ts.token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("borrar biblioteca: %d %s", status, body)
	}
	for _, table := range []string{"music_tracks", "music_albums", "music_artists"} {
		if err := ts.db.QueryRow(`SELECT count(*) FROM ` + table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s tras borrar biblioteca = %d, %v", table, count, err)
		}
	}
}
