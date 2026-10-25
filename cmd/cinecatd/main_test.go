package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/catlinux/cinecat/internal/auth"
	"github.com/catlinux/cinecat/internal/library"
	"github.com/catlinux/cinecat/internal/playback"
	"github.com/catlinux/cinecat/internal/store"
)

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	newHandler(nil, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /healthz devolvió %d, se esperaba %d", rec.Code, http.StatusOK)
	}
}

func TestHealthzRechazaOtrosMetodos(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rec := httptest.NewRecorder()

	newHandler(nil, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /healthz devolvió %d, se esperaba %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

type testServer struct {
	*httptest.Server
	db    *sql.DB
	clock *fakeClock
}

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	ctx := context.Background()

	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "cinecat.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close: %v", err)
		}
	})
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	authSvc := auth.New(st.DB())
	clock := newFakeClock()
	playSvc, err := playback.New(playback.Options{
		WorkDir: filepath.Join(t.TempDir(), "hls"),
		Now:     clock.Now,
	})
	if err != nil {
		t.Fatalf("playback.New: %v", err)
	}
	t.Cleanup(func() {
		if err := playSvc.Close(); err != nil {
			t.Errorf("playback.Close: %v", err)
		}
	})

	server := httptest.NewServer(newHandler(authSvc, library.NewAPI(st.DB(), authSvc, playSvc)))
	t.Cleanup(server.Close)

	return &testServer{Server: server, db: st.DB(), clock: clock}
}

func execSQL(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	res, err := db.Exec(query, args...)
	if err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId: %v", err)
	}
	return id
}

type seededData struct {
	movieID   int64
	episodeID int64
}

func seedLibraryData(t *testing.T, db *sql.DB) seededData {
	t.Helper()

	movies := execSQL(t, db, `INSERT INTO libraries (path, name) VALUES (?, ?)`, "/media/peliculas", "Peliculas")
	series := execSQL(t, db, `INSERT INTO libraries (path, name) VALUES (?, ?)`, "/media/series", "Series")

	movie := execSQL(t, db, `
		INSERT INTO media_items (library_id, path, title, year, kind, tmdb_id)
		VALUES (?, ?, ?, ?, ?, ?)`,
		movies, "/media/peliculas/Matrix (1999).mkv", "Matrix", 1999, "movie", 603)
	episode := execSQL(t, db, `
		INSERT INTO media_items (library_id, path, title, year, kind, season, episode)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		series, "/media/series/Serie/Season 1/S01E01.mkv", "Serie", nil, "episode", 1, 1)
	execSQL(t, db, `
		INSERT INTO media_items (library_id, path, title, year, kind)
		VALUES (?, ?, ?, ?, ?)`,
		movies, "/media/peliculas/Otra (2010).mkv", "Otra", 2010, "movie")

	return seededData{movieID: movie, episodeID: episode}
}

func createTestUser(t *testing.T, db *sql.DB, username, password string) {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	execSQL(t, db, `INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`,
		username, hash, "admin")
}

func loginToken(t *testing.T, serverURL, username, password string) string {
	t.Helper()
	body, err := json.Marshal(map[string]string{"username": username, "password": password})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	resp, err := http.Post(serverURL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login devolvió %d", resp.StatusCode)
	}

	var payload struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("respuesta de login: %v", err)
	}
	if payload.Token == "" {
		t.Fatal("el login no devolvió token")
	}
	return payload.Token
}

func getJSON(t *testing.T, url, token string, dst any) int {
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

	if dst != nil {
		if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
			t.Fatalf("respuesta de %s: %v", url, err)
		}
	}
	return resp.StatusCode
}

func TestLibrariesSinToken(t *testing.T) {
	server := newTestServer(t)

	var body struct {
		Error string `json:"error"`
	}
	status := getJSON(t, server.URL+"/api/v1/libraries", "", &body)
	if status != http.StatusUnauthorized {
		t.Fatalf("código = %d, se esperaba 401", status)
	}
	if body.Error == "" {
		t.Fatal("la respuesta 401 debe incluir un mensaje de error")
	}
}

func TestLibrariesConToken(t *testing.T) {
	server := newTestServer(t)
	seedLibraryData(t, server.db)
	createTestUser(t, server.db, "ana", "Clave-segura-1")

	token := loginToken(t, server.URL, "ana", "Clave-segura-1")

	var body struct {
		Libraries []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
			Path string `json:"path"`
		} `json:"libraries"`
	}
	status := getJSON(t, server.URL+"/api/v1/libraries", token, &body)
	if status != http.StatusOK {
		t.Fatalf("código = %d, se esperaba 200", status)
	}
	if len(body.Libraries) != 2 {
		t.Fatalf("bibliotecas = %d, se esperaban 2", len(body.Libraries))
	}
	if body.Libraries[0].Name != "Peliculas" || body.Libraries[0].Path != "/media/peliculas" {
		t.Fatalf("primera biblioteca inesperada: %+v", body.Libraries[0])
	}
	if body.Libraries[1].Name != "Series" || body.Libraries[1].Path != "/media/series" {
		t.Fatalf("segunda biblioteca inesperada: %+v", body.Libraries[1])
	}
}

func TestItemExistenteYNoEncontrado(t *testing.T) {
	server := newTestServer(t)
	seed := seedLibraryData(t, server.db)
	createTestUser(t, server.db, "ana", "Clave-segura-1")
	token := loginToken(t, server.URL, "ana", "Clave-segura-1")

	var movie struct {
		ID        int64  `json:"id"`
		LibraryID int64  `json:"library_id"`
		Path      string `json:"path"`
		Title     string `json:"title"`
		Year      *int   `json:"year"`
		Kind      string `json:"kind"`
		Season    *int   `json:"season"`
		Episode   *int   `json:"episode"`
		TMDBID    *int64 `json:"tmdb_id"`
	}
	status := getJSON(t, fmt.Sprintf("%s/api/v1/items/%d", server.URL, seed.movieID), token, &movie)
	if status != http.StatusOK {
		t.Fatalf("código = %d, se esperaba 200", status)
	}
	if movie.ID != seed.movieID || movie.Title != "Matrix" || movie.Kind != "movie" ||
		movie.Year == nil || *movie.Year != 1999 ||
		movie.TMDBID == nil || *movie.TMDBID != 603 ||
		movie.Season != nil || movie.Episode != nil {
		t.Fatalf("ítem inesperado: %+v", movie)
	}

	var episode struct {
		Season  *int `json:"season"`
		Episode *int `json:"episode"`
	}
	status = getJSON(t, fmt.Sprintf("%s/api/v1/items/%d", server.URL, seed.episodeID), token, &episode)
	if status != http.StatusOK {
		t.Fatalf("episodio: código = %d", status)
	}
	if episode.Season == nil || *episode.Season != 1 || episode.Episode == nil || *episode.Episode != 1 {
		t.Fatalf("episodio inesperado: %+v", episode)
	}

	var notFound struct {
		Error string `json:"error"`
	}
	status = getJSON(t, server.URL+"/api/v1/items/999999", token, &notFound)
	if status != http.StatusNotFound {
		t.Fatalf("inexistente: código = %d, se esperaba 404", status)
	}
	if notFound.Error == "" {
		t.Fatal("la respuesta 404 debe incluir un mensaje de error")
	}

	if status := getJSON(t, server.URL+"/api/v1/items/abc", token, nil); status != http.StatusNotFound {
		t.Fatalf("id no numérico: código = %d, se esperaba 404", status)
	}
}

func TestItemsPaginacion(t *testing.T) {
	server := newTestServer(t)
	seed := seedLibraryData(t, server.db)
	createTestUser(t, server.db, "ana", "Clave-segura-1")
	token := loginToken(t, server.URL, "ana", "Clave-segura-1")

	type itemsPage struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
		Total  int64 `json:"total"`
		Limit  int   `json:"limit"`
		Offset int   `json:"offset"`
	}

	var first itemsPage
	if status := getJSON(t, server.URL+"/api/v1/items?limit=1", token, &first); status != http.StatusOK {
		t.Fatalf("primera página: código = %d", status)
	}
	if first.Total != 3 || first.Limit != 1 || first.Offset != 0 || len(first.Items) != 1 {
		t.Fatalf("primera página inesperada: %+v", first)
	}
	if first.Items[0].ID != seed.movieID {
		t.Fatalf("primer ítem = %d, se esperaba %d", first.Items[0].ID, seed.movieID)
	}

	var second itemsPage
	if status := getJSON(t, server.URL+"/api/v1/items?limit=1&offset=1", token, &second); status != http.StatusOK {
		t.Fatalf("segunda página: código = %d", status)
	}
	if len(second.Items) != 1 || second.Items[0].ID == first.Items[0].ID {
		t.Fatalf("segunda página inesperada: %+v", second)
	}

	for _, query := range []string{"limit=0", "limit=201", "limit=abc", "offset=-1"} {
		var body struct {
			Error string `json:"error"`
		}
		if status := getJSON(t, server.URL+"/api/v1/items?"+query, token, &body); status != http.StatusBadRequest {
			t.Fatalf("%s: código = %d, se esperaba 400", query, status)
		}
	}
}

func TestItemsConTokenInvalido(t *testing.T) {
	server := newTestServer(t)
	seed := seedLibraryData(t, server.db)

	var body struct {
		Error string `json:"error"`
	}
	status := getJSON(t, fmt.Sprintf("%s/api/v1/items/%d", server.URL, seed.movieID), "token-inventado", &body)
	if status != http.StatusUnauthorized {
		t.Fatalf("código = %d, se esperaba 401", status)
	}
	if body.Error == "" {
		t.Fatal("la respuesta 401 debe incluir un mensaje de error")
	}
}

func TestLogoutInvalidaLaSesionEnLaAPI(t *testing.T) {
	server := newTestServer(t)
	createTestUser(t, server.db, "ana", "Clave-segura-1")
	token := loginToken(t, server.URL, "ana", "Clave-segura-1")

	if status := getJSON(t, server.URL+"/api/v1/libraries", token, nil); status != http.StatusOK {
		t.Fatalf("con sesión válida: código = %d, se esperaba 200", status)
	}

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/auth/logout", nil)
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("logout: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout: código = %d, se esperaba 204", resp.StatusCode)
	}

	if status := getJSON(t, server.URL+"/api/v1/libraries", token, nil); status != http.StatusUnauthorized {
		t.Fatalf("tras logout: código = %d, se esperaba 401", status)
	}
}

// --- Reproducción ---

func requireFFmpeg(t *testing.T) string {
	t.Helper()
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg no está disponible en PATH; se omite esta prueba de reproducción")
	}
	return ffmpeg
}

func generateLibraryVideo(t *testing.T, ext string) (libDir, file string) {
	t.Helper()
	ffmpeg := requireFFmpeg(t)

	codec := []string{"-c:v", "mpeg4", "-q:v", "5"}
	if out, err := exec.Command(ffmpeg, "-hide_banner", "-loglevel", "error", "-encoders").Output(); err == nil &&
		strings.Contains(string(out), "libx264") {
		codec = []string{"-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p"}
	}

	libDir = t.TempDir()
	file = filepath.Join(libDir, "Prueba"+ext)
	args := []string{
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=duration=1:size=160x120:rate=10",
	}
	args = append(args, codec...)
	args = append(args, "-an", file)

	cmd := exec.Command(ffmpeg, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v: %s", err, out)
	}
	return libDir, file
}

func insertMediaFile(t *testing.T, db *sql.DB, libDir, file, title string) int64 {
	t.Helper()
	libraryID := execSQL(t, db, `INSERT INTO libraries (path, name) VALUES (?, ?)`, libDir, "Pruebas")
	return execSQL(t, db, `
		INSERT INTO media_items (library_id, path, title, kind) VALUES (?, ?, ?, ?)`,
		libraryID, file, title, "movie")
}

type playInfo struct {
	URL       string     `json:"url"`
	Mode      string     `json:"mode"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func requestPlay(t *testing.T, serverURL string, id int64, token string) playInfo {
	t.Helper()
	var info playInfo
	status := getJSON(t, fmt.Sprintf("%s/api/v1/items/%d/play", serverURL, id), token, &info)
	if status != http.StatusOK {
		t.Fatalf("play: código = %d, se esperaba 200", status)
	}
	if info.URL == "" || info.Mode == "" || info.ExpiresAt == nil {
		t.Fatalf("respuesta de play incompleta: %+v", info)
	}
	return info
}

func getRaw(t *testing.T, rawURL string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", rawURL, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leer respuesta de %s: %v", rawURL, err)
	}
	return resp, data
}

func TestReproduccionDirectaMP4(t *testing.T) {
	libDir, file := generateLibraryVideo(t, ".mp4")
	server := newTestServer(t)
	createTestUser(t, server.db, "ana", "Clave-segura-1")
	token := loginToken(t, server.URL, "ana", "Clave-segura-1")
	id := insertMediaFile(t, server.db, libDir, file, "Prueba")

	info := requestPlay(t, server.URL, id, token)
	if info.Mode != "direct" {
		t.Fatalf("modo = %q, se esperaba direct", info.Mode)
	}
	if strings.Contains(info.URL, "playlist") {
		t.Fatalf("la URL directa no debe apuntar a una playlist: %q", info.URL)
	}

	resp, data := getRaw(t, server.URL+info.URL, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stream directo: código = %d", resp.StatusCode)
	}
	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("leer el archivo de prueba: %v", err)
	}
	if !bytes.Equal(data, want) {
		t.Fatalf("el cuerpo servido (%d bytes) no coincide con el archivo (%d bytes)", len(data), len(want))
	}

	resp, part := getRaw(t, server.URL+info.URL, map[string]string{"Range": "bytes=0-9"})
	if resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("petición con Range: código = %d, se esperaba 206", resp.StatusCode)
	}
	if len(part) != 10 || resp.Header.Get("Content-Range") == "" {
		t.Fatalf("rango inesperado: %d bytes, Content-Range = %q", len(part), resp.Header.Get("Content-Range"))
	}
}

func TestRemuxMKVGeneraHLS(t *testing.T) {
	libDir, file := generateLibraryVideo(t, ".mkv")
	server := newTestServer(t)
	createTestUser(t, server.db, "ana", "Clave-segura-1")
	token := loginToken(t, server.URL, "ana", "Clave-segura-1")
	id := insertMediaFile(t, server.db, libDir, file, "Prueba")

	info := requestPlay(t, server.URL, id, token)
	if info.Mode != "remux" {
		t.Fatalf("modo = %q, se esperaba remux", info.Mode)
	}
	if !strings.Contains(info.URL, "/playlist.m3u8") {
		t.Fatalf("la URL de remux debe apuntar a la playlist: %q", info.URL)
	}

	resp, playlist := getRaw(t, server.URL+info.URL, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("playlist: código = %d", resp.StatusCode)
	}
	body := string(playlist)
	if !strings.HasPrefix(body, "#EXTM3U") {
		t.Fatalf("playlist inesperada: %q", body)
	}
	if !strings.Contains(body, `#EXT-X-MAP:URI="init.mp4?`) {
		t.Fatalf("la playlist no trae init de fMP4 firmado: %q", body)
	}
	if !strings.Contains(body, ".m4s?") {
		t.Fatalf("la playlist no trae segmentos fMP4 firmados: %q", body)
	}

	var segment string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			segment = line
			break
		}
	}
	if segment == "" {
		t.Fatal("la playlist no lista ningún segmento")
	}

	segmentURL := server.URL + segment
	if !strings.HasPrefix(segment, "/") {
		segmentURL = fmt.Sprintf("%s/api/v1/stream/%d/%s", server.URL, id, segment)
	}
	resp, data := getRaw(t, segmentURL, nil)
	if resp.StatusCode != http.StatusOK || len(data) == 0 {
		t.Fatalf("segmento: código = %d, %d bytes", resp.StatusCode, len(data))
	}
}

func TestFirmaManipuladaYCaducada(t *testing.T) {
	libDir := t.TempDir()
	file := filepath.Join(libDir, "Pelicula.mp4")
	if err := os.WriteFile(file, []byte("datos de prueba"), 0o644); err != nil {
		t.Fatalf("crear archivo: %v", err)
	}

	server := newTestServer(t)
	createTestUser(t, server.db, "ana", "Clave-segura-1")
	token := loginToken(t, server.URL, "ana", "Clave-segura-1")
	id := insertMediaFile(t, server.db, libDir, file, "Pelicula")

	info := requestPlay(t, server.URL, id, token)
	validURL := server.URL + info.URL
	if resp, _ := getRaw(t, validURL, nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("URL válida: código = %d", resp.StatusCode)
	}

	parsed, err := url.Parse(validURL)
	if err != nil {
		t.Fatalf("parsear URL: %v", err)
	}
	query := parsed.Query()
	signature := query.Get("sig")
	tampered := "A" + signature[1:]
	if signature[0] == 'A' {
		tampered = "B" + signature[1:]
	}
	query.Set("sig", tampered)
	parsed.RawQuery = query.Encode()
	if resp, _ := getRaw(t, parsed.String(), nil); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("firma manipulada: código = %d, se esperaba 403", resp.StatusCode)
	}

	server.clock.Advance(16 * time.Minute)
	if resp, _ := getRaw(t, validURL, nil); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("URL caducada: código = %d, se esperaba 403", resp.StatusCode)
	}
}

func TestPlaySinToken(t *testing.T) {
	server := newTestServer(t)

	var body struct {
		Error string `json:"error"`
	}
	if status := getJSON(t, server.URL+"/api/v1/items/1/play", "", &body); status != http.StatusUnauthorized {
		t.Fatalf("código = %d, se esperaba 401", status)
	}
}
