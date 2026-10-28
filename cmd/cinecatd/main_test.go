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
	"sync/atomic"
	"testing"
	"time"

	"github.com/catlinux/cinecat/internal/auth"
	"github.com/catlinux/cinecat/internal/library"
	"github.com/catlinux/cinecat/internal/playback"
	"github.com/catlinux/cinecat/internal/settings"
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

func TestSecurityHeadersEnAPISinSesion(t *testing.T) {
	server := newTestServer(t)
	for _, path := range []string{"/healthz", "/api/v1/libraries"} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
			t.Fatalf("%s: X-Content-Type-Options = %q", path, got)
		}
		if got := resp.Header.Get("Referrer-Policy"); got != "no-referrer" {
			t.Fatalf("%s: Referrer-Policy = %q", path, got)
		}
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

func TestVersionEndpointSinSesion(t *testing.T) {
	oldVersion, oldCommit, oldBuiltAt := version, commit, builtAt
	t.Cleanup(func() {
		version, commit, builtAt = oldVersion, oldCommit, oldBuiltAt
	})
	version = "1.2.3"
	commit = "abc1234"
	builtAt = "2026-10-26T09:00:00Z"

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/version", nil)
	rec := httptest.NewRecorder()
	newHandler(nil, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("código = %d, se esperaba %d", rec.Code, http.StatusOK)
	}
	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", contentType)
	}

	var body versionResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("respuesta de versión: %v", err)
	}
	if body.Server != "1.2.3" || body.Commit != "abc1234" || body.BuiltAt != "2026-10-26T09:00:00Z" {
		t.Fatalf("versión inesperada: %+v", body)
	}
}

type testServer struct {
	*httptest.Server
	db       *sql.DB
	clock    *fakeClock
	scanJob  *library.ScanJob
	settings *settings.Store
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
	settingsStore := settings.NewStore(st.DB(), nil)
	if err := settingsStore.Load(ctx); err != nil {
		t.Fatalf("settings.Load: %v", err)
	}

	authSvc := auth.New(st.DB())
	clock := newFakeClock()
	playSvc, err := playback.New(playback.Options{
		WorkDir: filepath.Join(t.TempDir(), "hls"),
		Now:     clock.Now,
		Settings: func() playback.TranscodeSettings {
			return playback.TranscodeSettings{
				MaxHeight: settingsStore.Int("playback.max_height"),
				CRF:       settingsStore.Int("playback.crf"),
				Preset:    settingsStore.String("playback.preset"),
				Hardware:  settingsStore.String(settings.KeyPlaybackHardware) != "off",
			}
		},
		MaxTranscodesFunc: func() int {
			return settingsStore.Int("playback.max_transcodes")
		},
	})
	if err != nil {
		t.Fatalf("playback.New: %v", err)
	}
	t.Cleanup(func() {
		if err := playSvc.Close(); err != nil {
			t.Errorf("playback.Close: %v", err)
		}
	})

	scanJob := library.NewScanJob(library.New(st.DB()))
	t.Cleanup(scanJob.Close)
	api := library.NewAPI(st.DB(), authSvc, playSvc, scanJob, nil).WithSettings(settingsStore)
	// Las pruebas crean sus carpetas en el directorio temporal del sistema.
	api = api.WithMediaRoots([]string{os.TempDir(), "/media"})
	server := httptest.NewServer(newHandler(authSvc, api))
	t.Cleanup(server.Close)

	return &testServer{
		Server: server, db: st.DB(), clock: clock, scanJob: scanJob, settings: settingsStore,
	}
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
	moviesLibraryID int64
	seriesLibraryID int64
	movieID         int64
	episodeID       int64
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

	return seededData{
		moviesLibraryID: movies,
		seriesLibraryID: series,
		movieID:         movie,
		episodeID:       episode,
	}
}

func createTestUser(t *testing.T, db *sql.DB, username, password string) {
	t.Helper()
	createTestUserWithRole(t, db, username, password, "admin")
}

func createTestUserWithRole(t *testing.T, db *sql.DB, username, password, role string) {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	execSQL(t, db, `INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`,
		username, hash, role)
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

func doJSON(t *testing.T, method, rawURL, token string, body any) (int, []byte) {
	t.Helper()

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, rawURL, reader)
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, rawURL, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leer respuesta de %s: %v", rawURL, err)
	}
	return resp.StatusCode, data
}

func waitLibraryScan(t *testing.T, serverURL, token string) library.ScanStatus {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var status library.ScanStatus
		if code := getJSON(t, serverURL+"/api/v1/libraries/scan", token, &status); code != http.StatusOK {
			t.Fatalf("estado del escaneo: código = %d", code)
		}
		if !status.Running {
			return status
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("el escaneo no terminó a tiempo")
	return library.ScanStatus{}
}

func TestCrearBibliotecaAdmin(t *testing.T) {
	server := newTestServer(t)
	createTestUser(t, server.db, "ana", "Clave-segura-1")
	token := loginToken(t, server.URL, "ana", "Clave-segura-1")

	libDir := t.TempDir()
	status, body := doJSON(t, http.MethodPost, server.URL+"/api/v1/libraries", token,
		map[string]string{"name": "Peliculas", "path": libDir})
	if status != http.StatusCreated {
		t.Fatalf("código = %d, se esperaba 201; cuerpo = %s", status, body)
	}

	var created struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("respuesta de creación: %v", err)
	}
	if created.ID <= 0 || created.Name != "Peliculas" || created.Path != filepath.Clean(libDir) {
		t.Fatalf("biblioteca creada inesperada: %+v", created)
	}

	var list struct {
		Libraries []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
			Path string `json:"path"`
		} `json:"libraries"`
	}
	if status := getJSON(t, server.URL+"/api/v1/libraries", token, &list); status != http.StatusOK {
		t.Fatalf("listar: código = %d", status)
	}
	if len(list.Libraries) != 1 || list.Libraries[0].ID != created.ID ||
		list.Libraries[0].Name != "Peliculas" {
		t.Fatalf("la biblioteca creada no aparece en el listado: %+v", list.Libraries)
	}
}

func TestCrearBibliotecaNonAdmin(t *testing.T) {
	server := newTestServer(t)
	createTestUserWithRole(t, server.db, "ana", "Clave-segura-1", "user")
	token := loginToken(t, server.URL, "ana", "Clave-segura-1")

	status, body := doJSON(t, http.MethodPost, server.URL+"/api/v1/libraries", token,
		map[string]string{"name": "Peliculas", "path": t.TempDir()})
	if status != http.StatusForbidden {
		t.Fatalf("código = %d, se esperaba 403; cuerpo = %s", status, body)
	}

	var libraries int
	if err := server.db.QueryRow(`SELECT count(*) FROM libraries`).Scan(&libraries); err != nil {
		t.Fatalf("contar bibliotecas: %v", err)
	}
	if libraries != 0 {
		t.Fatalf("no debe crearse ninguna biblioteca, hay %d", libraries)
	}
}

func TestCrearBibliotecaPathInvalido(t *testing.T) {
	server := newTestServer(t)
	createTestUser(t, server.db, "ana", "Clave-segura-1")
	token := loginToken(t, server.URL, "ana", "Clave-segura-1")

	missing := filepath.Join(t.TempDir(), "no-existe")
	status, body := doJSON(t, http.MethodPost, server.URL+"/api/v1/libraries", token,
		map[string]string{"name": "Peliculas", "path": missing})
	if status != http.StatusBadRequest {
		t.Fatalf("ruta inexistente: código = %d, se esperaba 400; cuerpo = %s", status, body)
	}
	var errBody struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &errBody); err != nil || errBody.Error == "" {
		t.Fatalf("la respuesta 400 debe traer un mensaje claro: %s", body)
	}

	file := filepath.Join(t.TempDir(), "archivo.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("crear archivo: %v", err)
	}
	status, body = doJSON(t, http.MethodPost, server.URL+"/api/v1/libraries", token,
		map[string]string{"name": "Peliculas", "path": file})
	if status != http.StatusBadRequest {
		t.Fatalf("ruta a un archivo: código = %d, se esperaba 400; cuerpo = %s", status, body)
	}

	var libraries int
	if err := server.db.QueryRow(`SELECT count(*) FROM libraries`).Scan(&libraries); err != nil {
		t.Fatalf("contar bibliotecas: %v", err)
	}
	if libraries != 0 {
		t.Fatalf("no debe crearse ninguna biblioteca, hay %d", libraries)
	}
}

func TestBorrarBibliotecaAdmin(t *testing.T) {
	server := newTestServer(t)
	seed := seedLibraryData(t, server.db)
	createTestUser(t, server.db, "ana", "Clave-segura-1")
	token := loginToken(t, server.URL, "ana", "Clave-segura-1")

	url := fmt.Sprintf("%s/api/v1/libraries/%d", server.URL, seed.moviesLibraryID)
	status, body := doJSON(t, http.MethodDelete, url, token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("código = %d, se esperaba 204; cuerpo = %s", status, body)
	}

	var items int
	if err := server.db.QueryRow(
		`SELECT count(*) FROM media_items WHERE library_id = ?`, seed.moviesLibraryID,
	).Scan(&items); err != nil {
		t.Fatalf("contar ítems: %v", err)
	}
	if items != 0 {
		t.Fatalf("quedan %d media_items de la biblioteca borrada", items)
	}

	var total int
	if err := server.db.QueryRow(`SELECT count(*) FROM media_items`).Scan(&total); err != nil {
		t.Fatalf("contar ítems totales: %v", err)
	}
	if total != 1 {
		t.Fatalf("debería quedar solo el episodio de la otra biblioteca, hay %d", total)
	}

	var list struct {
		Libraries []struct {
			ID int64 `json:"id"`
		} `json:"libraries"`
	}
	if status := getJSON(t, server.URL+"/api/v1/libraries", token, &list); status != http.StatusOK {
		t.Fatalf("listar: código = %d", status)
	}
	if len(list.Libraries) != 1 || list.Libraries[0].ID != seed.seriesLibraryID {
		t.Fatalf("listado tras borrar inesperado: %+v", list.Libraries)
	}

	if status, _ := doJSON(t, http.MethodDelete, url, token, nil); status != http.StatusNotFound {
		t.Fatalf("segundo borrado: código = %d, se esperaba 404", status)
	}
}

func TestBorrarBibliotecaNonAdmin(t *testing.T) {
	server := newTestServer(t)
	seed := seedLibraryData(t, server.db)
	createTestUserWithRole(t, server.db, "ana", "Clave-segura-1", "user")
	token := loginToken(t, server.URL, "ana", "Clave-segura-1")

	url := fmt.Sprintf("%s/api/v1/libraries/%d", server.URL, seed.moviesLibraryID)
	status, body := doJSON(t, http.MethodDelete, url, token, nil)
	if status != http.StatusForbidden {
		t.Fatalf("código = %d, se esperaba 403; cuerpo = %s", status, body)
	}

	var items int
	if err := server.db.QueryRow(`SELECT count(*) FROM media_items`).Scan(&items); err != nil {
		t.Fatalf("contar ítems: %v", err)
	}
	if items != 3 {
		t.Fatalf("no debe borrarse nada, quedan %d ítems", items)
	}
}

func TestBorrarBibliotecaInexistente(t *testing.T) {
	server := newTestServer(t)
	createTestUser(t, server.db, "ana", "Clave-segura-1")
	token := loginToken(t, server.URL, "ana", "Clave-segura-1")

	status, body := doJSON(t, http.MethodDelete, server.URL+"/api/v1/libraries/999999", token, nil)
	if status != http.StatusNotFound {
		t.Fatalf("código = %d, se esperaba 404; cuerpo = %s", status, body)
	}
}

func TestScanBibliotecasAdmin(t *testing.T) {
	server := newTestServer(t)
	createTestUser(t, server.db, "ana", "Clave-segura-1")
	token := loginToken(t, server.URL, "ana", "Clave-segura-1")

	libDir := t.TempDir()
	for _, name := range []string{"Pelicula (1999).mkv", "Otra.mp4", "notas.txt"} {
		if err := os.WriteFile(filepath.Join(libDir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("crear archivo: %v", err)
		}
	}

	status, body := doJSON(t, http.MethodPost, server.URL+"/api/v1/libraries", token,
		map[string]string{"name": "Peliculas", "path": libDir})
	if status != http.StatusCreated {
		t.Fatalf("crear biblioteca: código = %d, cuerpo = %s", status, body)
	}

	status, body = doJSON(t, http.MethodPost, server.URL+"/api/v1/libraries/scan", token, nil)
	if status != http.StatusAccepted {
		t.Fatalf("scan: código = %d, cuerpo = %s", status, body)
	}
	var started struct {
		Started bool `json:"started"`
	}
	if err := json.Unmarshal(body, &started); err != nil || !started.Started {
		t.Fatalf("respuesta de inicio de scan inesperada: %s", body)
	}
	result := waitLibraryScan(t, server.URL, token)
	if result.Libraries != 1 || result.Files != 2 || result.Upserted != 2 {
		t.Fatalf("resultado de scan inesperado: %+v", result)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("avisos inesperados: %v", result.Warnings)
	}
	if result.Error != "" || result.FinishedAt == nil {
		t.Fatalf("estado final de scan inesperado: %+v", result)
	}

	var items int
	if err := server.db.QueryRow(`SELECT count(*) FROM media_items`).Scan(&items); err != nil {
		t.Fatalf("contar ítems: %v", err)
	}
	if items != 2 {
		t.Fatalf("media_items = %d, se esperaban 2", items)
	}

	// Un segundo escaneo no duplica entradas.
	if status, body := doJSON(t, http.MethodPost, server.URL+"/api/v1/libraries/scan", token, nil); status != http.StatusAccepted {
		t.Fatalf("segundo scan: código = %d, cuerpo = %s", status, body)
	}
	waitLibraryScan(t, server.URL, token)
	if err := server.db.QueryRow(`SELECT count(*) FROM media_items`).Scan(&items); err != nil {
		t.Fatalf("contar ítems tras el segundo scan: %v", err)
	}
	if items != 2 {
		t.Fatalf("tras el segundo scan hay %d ítems, se esperaban 2", items)
	}
}

func TestScanBibliotecasNonAdmin(t *testing.T) {
	server := newTestServer(t)
	createTestUserWithRole(t, server.db, "ana", "Clave-segura-1", "user")
	token := loginToken(t, server.URL, "ana", "Clave-segura-1")

	status, body := doJSON(t, http.MethodPost, server.URL+"/api/v1/libraries/scan", token, nil)
	if status != http.StatusForbidden {
		t.Fatalf("POST: código = %d, se esperaba 403; cuerpo = %s", status, body)
	}
	status, body = doJSON(t, http.MethodGet, server.URL+"/api/v1/libraries/scan", token, nil)
	if status != http.StatusForbidden {
		t.Fatalf("GET: código = %d, se esperaba 403; cuerpo = %s", status, body)
	}
}

func TestScanBibliotecasSinSesion(t *testing.T) {
	server := newTestServer(t)

	status, body := doJSON(t, http.MethodPost, server.URL+"/api/v1/libraries/scan", "", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("POST: código = %d, se esperaba 401; cuerpo = %s", status, body)
	}
	status, body = doJSON(t, http.MethodGet, server.URL+"/api/v1/libraries/scan", "", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("GET: código = %d, se esperaba 401; cuerpo = %s", status, body)
	}
}

type fakeScanStarter struct {
	calls atomic.Int32
}

func (f *fakeScanStarter) Start(ctx context.Context) bool {
	f.calls.Add(1)
	return true
}

type hotScanSettings struct {
	mu        sync.Mutex
	interval  time.Duration
	onStartup bool
	changes   chan struct{}
}

func newHotScanSettings(interval time.Duration, onStartup bool) *hotScanSettings {
	return &hotScanSettings{
		interval:  interval,
		onStartup: onStartup,
		changes:   make(chan struct{}, 1),
	}
}

func (f *hotScanSettings) Interval() time.Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.interval
}

func (f *hotScanSettings) OnStartup() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.onStartup
}

func (f *hotScanSettings) Changes() <-chan struct{} {
	return f.changes
}

func (f *hotScanSettings) set(interval time.Duration, onStartup bool) {
	f.mu.Lock()
	f.interval = interval
	f.onStartup = onStartup
	f.mu.Unlock()
	select {
	case f.changes <- struct{}{}:
	default:
	}
}

func waitScheduledScanCalls(t *testing.T, job *fakeScanStarter, want int32) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for job.calls.Load() < want && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if job.calls.Load() < want {
		t.Fatalf("el planificador solo lanzó %d escaneos, se esperaban %d", job.calls.Load(), want)
	}
}

func TestScheduleScansArranqueYPeriodico(t *testing.T) {
	var job fakeScanStarter
	settings := newHotScanSettings(5*time.Millisecond, true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		scheduleScans(ctx, &job, 2*time.Millisecond, settings)
		close(done)
	}()

	waitScheduledScanCalls(t, &job, 2)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("el planificador no terminó al cancelar el contexto")
	}
	before := job.calls.Load()
	time.Sleep(20 * time.Millisecond)
	if job.calls.Load() != before {
		t.Fatal("el planificador siguió lanzando escaneos tras cancelar")
	}
}

func TestScheduleScansSinArranque(t *testing.T) {
	var job fakeScanStarter
	settings := newHotScanSettings(500*time.Millisecond, false)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go scheduleScans(ctx, &job, 5*time.Millisecond, settings)
	time.Sleep(20 * time.Millisecond)
	if job.calls.Load() != 0 {
		t.Fatalf("sin arranque no debe escanearse antes del intervalo: %d", job.calls.Load())
	}
	waitScheduledScanCalls(t, &job, 1)
}

func TestScheduleScansCambioDeIntervaloYReactivacion(t *testing.T) {
	var job fakeScanStarter
	settings := newHotScanSettings(time.Second, false)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go scheduleScans(ctx, &job, time.Minute, settings)
	time.Sleep(20 * time.Millisecond)
	if job.calls.Load() != 0 {
		t.Fatalf("no debe esperar al intervalo largo: %d", job.calls.Load())
	}

	settings.set(5*time.Millisecond, false)
	waitScheduledScanCalls(t, &job, 1)

	settings.set(0, false)
	time.Sleep(20 * time.Millisecond)
	before := job.calls.Load()
	time.Sleep(30 * time.Millisecond)
	if job.calls.Load() != before {
		t.Fatalf("con intervalo 0 no deben lanzarse escaneos: %d -> %d", before, job.calls.Load())
	}

	settings.set(5*time.Millisecond, false)
	waitScheduledScanCalls(t, &job, before+1)
}

func TestStoreScanSettingsAdaptaYDespierta(t *testing.T) {
	server := newTestServer(t)
	ctx := context.Background()
	adapter := newStoreScanSettings(server.settings)
	if adapter.Interval() != 6*time.Hour || !adapter.OnStartup() {
		t.Fatalf("defectos del adaptador: %v, %v", adapter.Interval(), adapter.OnStartup())
	}

	if failures := server.settings.Set(ctx, map[string]json.RawMessage{
		settings.KeyScanInterval: json.RawMessage(`"7m"`),
		"scan.on_startup":        json.RawMessage("false"),
	}); len(failures) != 0 {
		t.Fatalf("Set: %v", failures)
	}
	select {
	case <-adapter.Changes():
	case <-time.After(time.Second):
		t.Fatal("el adaptador no avisó del cambio de escaneo")
	}
	if adapter.Interval() != 7*time.Minute || adapter.OnStartup() {
		t.Fatalf("adaptador sin actualizar: %v, %v", adapter.Interval(), adapter.OnStartup())
	}

	if failures := server.settings.Set(ctx, map[string]json.RawMessage{
		"ui.grid_columns": json.RawMessage("4"),
	}); len(failures) != 0 {
		t.Fatalf("Set ajeno: %v", failures)
	}
	select {
	case <-adapter.Changes():
		t.Fatal("un ajuste ajeno al escaneo no debe despertar el planificador")
	case <-time.After(20 * time.Millisecond):
	}
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

	codec := []string{"-c:v", "mpeg4", "-q:v", "5", "-g", "5"}
	if out, err := exec.Command(ffmpeg, "-hide_banner", "-loglevel", "error", "-encoders").Output(); err == nil &&
		strings.Contains(string(out), "libx264") {
		codec = []string{"-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p", "-g", "5", "-keyint_min", "5", "-sc_threshold", "0"}
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
	if !strings.Contains(body, ".ts?") {
		t.Fatalf("la playlist no trae segmentos MPEG-TS firmados: %q", body)
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

	server.clock.Advance(playback.DefaultURLTTL + time.Minute)
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
