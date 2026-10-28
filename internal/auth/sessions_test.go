package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catlinux/cinecat/internal/store"
)

func TestMigracionSesionesConservaLasExistentes(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "cinecat.db")
	st, err := store.Open(ctx, path)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})

	if _, err := st.DB().ExecContext(ctx, `
		CREATE TABLE schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		) STRICT`); err != nil {
		t.Fatalf("crear schema_migrations: %v", err)
	}
	for _, version := range []string{
		"0001_init", "0002_media_item_season_episode", "0003_media_item_omdb",
		"0004_media_item_tmdb", "0005_media_item_display_title", "0006_watch_state",
		"0007_media_item_image_locks", "0008_playback_progress", "0009_collections",
		"0010_series", "0011_settings", "0012_credits", "0013_refill_tmdb_metadata",
		"0014_item_preferences", "0015_library_paths", "0016_music",
		"0017_user_lists", "0018_trash", "0019_people_trailers", "0020_user_access",
	} {
		if _, err := st.DB().ExecContext(ctx,
			`INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
			t.Fatalf("registrar migración %s: %v", version, err)
		}
	}
	if _, err := st.DB().ExecContext(ctx, `
		CREATE TABLE users (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			username      TEXT    NOT NULL UNIQUE,
			password_hash TEXT    NOT NULL,
			role          TEXT    NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user')),
			created_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
			disabled      INTEGER NOT NULL DEFAULT 0,
			all_libraries INTEGER NOT NULL DEFAULT 1
		) STRICT`); err != nil {
		t.Fatalf("crear users antiguo: %v", err)
	}
	if _, err := st.DB().ExecContext(ctx, `
		CREATE TABLE sessions (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
			token_hash TEXT    NOT NULL UNIQUE,
			expires_at TEXT    NOT NULL
		) STRICT`); err != nil {
		t.Fatalf("crear sessions antiguo: %v", err)
	}
	if _, err := st.DB().ExecContext(ctx,
		`CREATE INDEX sessions_user_id ON sessions (user_id)`); err != nil {
		t.Fatalf("crear índice de usuario: %v", err)
	}
	if _, err := st.DB().ExecContext(ctx,
		`CREATE INDEX sessions_expires_at ON sessions (expires_at)`); err != nil {
		t.Fatalf("crear índice de caducidad: %v", err)
	}
	if _, err := st.DB().ExecContext(ctx,
		`INSERT INTO users (id, username, password_hash) VALUES (7, 'ana', 'hash')`); err != nil {
		t.Fatalf("crear usuario previo: %v", err)
	}
	token := "token-anterior"
	if _, err := st.DB().ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, token_hash, expires_at) VALUES (42, 7, ?, ?)`,
		hashToken(token), "2099-01-01T00:00:00.000Z"); err != nil {
		t.Fatalf("crear sesión previa: %v", err)
	}

	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var (
		id                    int64
		userID                int64
		tokenHash, expires    string
		deviceName, client    string
		createdAt, lastSeenAt string
	)
	if err := st.DB().QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, expires_at, device_name, client, created_at, last_seen_at
		  FROM sessions WHERE id = 42`).Scan(
		&id, &userID, &tokenHash, &expires, &deviceName, &client, &createdAt, &lastSeenAt,
	); err != nil {
		t.Fatalf("leer sesión migrada: %v", err)
	}
	if id != 42 || userID != 7 || tokenHash != hashToken(token) ||
		expires != "2099-01-01T00:00:00.000Z" || deviceName != "" || client != "web" ||
		createdAt == "" || lastSeenAt == "" {
		t.Fatalf("sesión migrada inesperada: id=%d user=%d hash=%q expires=%q device=%q client=%q created=%q last=%q",
			id, userID, tokenHash, expires, deviceName, client, createdAt, lastSeenAt)
	}

	svc := New(st.DB())
	user, err := svc.Authenticate(ctx, token)
	if err != nil {
		t.Fatalf("la sesión anterior debe seguir autenticando: %v", err)
	}
	if user.ID != 7 || user.SessionID != 42 {
		t.Fatalf("usuario de la sesión migrada = %+v", user)
	}

	for _, index := range []string{"sessions_user_id", "sessions_expires_at"} {
		var found int
		if err := st.DB().QueryRowContext(ctx,
			`SELECT count(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, index).Scan(&found); err != nil {
			t.Fatalf("buscar índice %s: %v", index, err)
		}
		if found != 1 {
			t.Fatalf("falta el índice %s", index)
		}
	}

	inserted, err := st.DB().ExecContext(ctx,
		`INSERT INTO sessions (user_id, token_hash, expires_at) VALUES (7, 'hash-defecto', '2099-01-01T00:00:00.000Z')`)
	if err != nil {
		t.Fatalf("crear sesión con los defectos nuevos: %v", err)
	}
	newID, err := inserted.LastInsertId()
	if err != nil {
		t.Fatalf("id de la sesión nueva: %v", err)
	}
	if newID <= 42 {
		t.Fatalf("el id nuevo (%d) debe continuar la secuencia existente", newID)
	}
	var defaultDevice, defaultClient, defaultCreated, defaultLastSeen string
	if err := st.DB().QueryRowContext(ctx, `
		SELECT device_name, client, created_at, last_seen_at
		  FROM sessions WHERE token_hash = 'hash-defecto'`).Scan(
		&defaultDevice, &defaultClient, &defaultCreated, &defaultLastSeen,
	); err != nil {
		t.Fatalf("leer defectos nuevos: %v", err)
	}
	if defaultDevice != "" || defaultClient != "web" || defaultCreated == "" || defaultLastSeen == "" {
		t.Fatalf("defectos de la sesión = %q %q %q %q",
			defaultDevice, defaultClient, defaultCreated, defaultLastSeen)
	}
}

func TestLoginConDispositivoYTTLPorCliente(t *testing.T) {
	svc, db, clock := newTestService(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")
	ctx := context.Background()

	cases := []struct {
		client string
		ttl    time.Duration
		name   string
	}{
		{ClientWeb, DefaultSessionTTL, "Escritorio"},
		{ClientAndroid, NativeSessionTTL, "Móvil"},
		{ClientAndroidTV, NativeSessionTTL, "Televisión"},
	}
	for _, test := range cases {
		result, err := svc.LoginWithDevice(ctx, "ana", "Clave-segura-1", "", test.name, test.client, "")
		if err != nil {
			t.Fatalf("login %s: %v", test.client, err)
		}
		if result.SessionID == 0 || result.Token == "" {
			t.Fatalf("resultado incompleto para %s: %+v", test.client, result)
		}
		if got := result.ExpiresAt.Sub(clock.Now()); got != test.ttl {
			t.Fatalf("%s: caducidad %v, se esperaba %v", test.client, got, test.ttl)
		}
		var deviceName, client string
		if err := db.QueryRow(`SELECT device_name, client FROM sessions WHERE id = ?`,
			result.SessionID).Scan(&deviceName, &client); err != nil {
			t.Fatalf("leer sesión %s: %v", test.client, err)
		}
		if deviceName != test.name || client != test.client {
			t.Fatalf("%s: dispositivo=%q cliente=%q", test.client, deviceName, client)
		}
	}

	if _, err := svc.LoginWithDevice(ctx, "ana", "Clave-segura-1", "", "", "ios", ""); !errors.Is(err, ErrInvalidClient) {
		t.Fatalf("cliente no válido: %v", err)
	}
}

func TestEtiquetaDeDispositivo(t *testing.T) {
	cases := []struct {
		userAgent string
		want      string
	}{
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/130.0 Safari/537.36", "Chrome · Windows"},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 Version/18.0 Mobile/15E148 Safari/604.1", "Safari · iOS"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/130.0 Safari/537.36 Edg/130.0", "Edge · Windows"},
		{"Mozilla/5.0 (X11; Linux x86_64; rv:131.0) Gecko/20100101 Firefox/131.0", "Firefox · Linux"},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/130.0 Safari/537.36 OPR/115.0", "Opera · macOS"},
		{"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 Chrome/130.0 Mobile Safari/537.36", "Chrome · Android"},
		{"Mozilla/5.0 (iPad; CPU OS 18_0 like Mac OS X) AppleWebKit/605.1.15 Version/18.0 Mobile/15E148 Safari/604.1", "Safari · iOS"},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Version/18.0 Mobile/15E148 Safari/604.1", "Safari · iOS"},
		{"curl/8.7.1", "Navegador"},
		{"", "Navegador"},
	}
	for _, test := range cases {
		if got := deviceLabel(test.userAgent); got != test.want {
			t.Errorf("deviceLabel(%q) = %q, se esperaba %q", test.userAgent, got, test.want)
		}
	}
}

func TestNombreDeDispositivoLimpiadoYRecortado(t *testing.T) {
	name := cleanDeviceName("  Portátil\x00 de\x1f\n casa  ")
	if name != "Portátil de casa" {
		t.Fatalf("nombre limpio = %q", name)
	}
	long := strings.Repeat("á", maxDeviceNameLength+20)
	if got := cleanDeviceName(long); got != strings.Repeat("á", maxDeviceNameLength) {
		t.Fatalf("longitud recortada = %d", len([]rune(got)))
	}
}

func TestListarSesionesYMarcarLaActual(t *testing.T) {
	svc, db, clock := newTestService(t)
	anaID := createUser(t, db, "ana", "Clave-segura-1", "user")
	createUser(t, db, "bea", "Clave-segura-2", "user")
	ctx := context.Background()

	desktop, err := svc.LoginWithDevice(ctx, "ana", "Clave-segura-1", "", "Escritorio", ClientWeb, "")
	if err != nil {
		t.Fatalf("login escritorio: %v", err)
	}
	clock.Advance(time.Minute)
	phone, err := svc.LoginWithDevice(ctx, "ana", "Clave-segura-1", "", "Móvil", ClientAndroid, "")
	if err != nil {
		t.Fatalf("login móvil: %v", err)
	}
	bea, err := svc.LoginWithDevice(ctx, "bea", "Clave-segura-2", "", "Bea", ClientWeb, "")
	if err != nil {
		t.Fatalf("login bea: %v", err)
	}

	list, err := svc.ListSessions(ctx, anaID, desktop.SessionID)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("sesiones = %d, se esperaban 2", len(list))
	}
	if list[0].ID != phone.SessionID || list[1].ID != desktop.SessionID {
		t.Fatalf("orden inesperado: %+v", list)
	}
	current := 0
	for _, session := range list {
		if session.Current {
			current++
			if session.ID != desktop.SessionID {
				t.Fatalf("sesión actual inesperada: %+v", session)
			}
		}
	}
	if current != 1 {
		t.Fatalf("sesiones marcadas como actuales = %d", current)
	}

	if err := svc.DeleteSession(ctx, anaID, bea.SessionID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("cerrar sesión ajena = %v", err)
	}
	if _, err := svc.Authenticate(ctx, bea.Token); err != nil {
		t.Fatalf("la sesión ajena no debe cerrarse: %v", err)
	}
	if err := svc.DeleteSession(ctx, anaID, desktop.SessionID); err != nil {
		t.Fatalf("cerrar sesión propia: %v", err)
	}
	if _, err := svc.Authenticate(ctx, desktop.Token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("la sesión cerrada debe quedar inválida: %v", err)
	}
}

func TestLastSeenSeActualizaComoMuchoCadaCincoMinutos(t *testing.T) {
	svc, db, clock := newTestService(t)
	userID := createUser(t, db, "ana", "Clave-segura-1", "user")
	ctx := context.Background()

	result, err := svc.LoginWithDevice(ctx, "ana", "Clave-segura-1", "", "Escritorio", ClientWeb, "")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	var initial string
	if err := db.QueryRow(`SELECT last_seen_at FROM sessions WHERE id = ?`, result.SessionID).
		Scan(&initial); err != nil {
		t.Fatalf("leer last_seen_at: %v", err)
	}

	clock.Advance(4 * time.Minute)
	if _, err := svc.Authenticate(ctx, result.Token); err != nil {
		t.Fatalf("Authenticate a los 4 minutos: %v", err)
	}
	var afterFour string
	if err := db.QueryRow(`SELECT last_seen_at FROM sessions WHERE id = ?`, result.SessionID).
		Scan(&afterFour); err != nil {
		t.Fatalf("leer last_seen_at: %v", err)
	}
	if afterFour != initial {
		t.Fatalf("last_seen_at se actualizó antes de 5 minutos: %q -> %q", initial, afterFour)
	}

	clock.Advance(time.Minute)
	if _, err := svc.Authenticate(ctx, result.Token); err != nil {
		t.Fatalf("Authenticate a los 5 minutos: %v", err)
	}
	var afterFive string
	if err := db.QueryRow(`SELECT last_seen_at FROM sessions WHERE id = ?`, result.SessionID).
		Scan(&afterFive); err != nil {
		t.Fatalf("leer last_seen_at: %v", err)
	}
	if afterFive != formatTime(clock.Now()) {
		t.Fatalf("last_seen_at = %q, se esperaba %q", afterFive, formatTime(clock.Now()))
	}
	var userIDFromDB int64
	if err := db.QueryRow(`SELECT user_id FROM sessions WHERE id = ?`, result.SessionID).
		Scan(&userIDFromDB); err != nil || userIDFromDB != userID {
		t.Fatalf("usuario de la sesión = %d, %v", userIDFromDB, err)
	}
}

func TestRenovacionNativaUsaTTLDeNoventaDias(t *testing.T) {
	svc, db, clock := newTestService(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")
	ctx := context.Background()

	result, err := svc.LoginWithDevice(ctx, "ana", "Clave-segura-1", "", "Móvil", ClientAndroid, "")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	clock.Advance(NativeSessionTTL - time.Hour)
	if _, err := svc.Authenticate(ctx, result.Token); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	var raw string
	if err := db.QueryRow(`SELECT expires_at FROM sessions WHERE id = ?`, result.SessionID).Scan(&raw); err != nil {
		t.Fatalf("leer expires_at: %v", err)
	}
	expiresAt, err := parseTime(raw)
	if err != nil {
		t.Fatalf("parseTime: %v", err)
	}
	want := clock.Now().Add(NativeSessionTTL).Truncate(time.Millisecond)
	if !expiresAt.Equal(want) {
		t.Fatalf("caducidad nativa = %v, se esperaba %v", expiresAt, want)
	}
}

type sessionHTTPResult struct {
	status   int
	body     []byte
	response loginResponse
}

func loginHTTPDevice(
	t *testing.T,
	serverURL, username, password, deviceName, client, userAgent string,
) sessionHTTPResult {
	t.Helper()
	body, err := json.Marshal(map[string]string{
		"username": username, "password": password, "device_name": deviceName, "client": client,
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, serverURL+"/api/v1/auth/login", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	status, data := doAuthRequest(t, req)
	result := sessionHTTPResult{status: status, body: []byte(data)}
	if status == http.StatusOK {
		if err := json.Unmarshal([]byte(data), &result.response); err != nil {
			t.Fatalf("respuesta de login: %v", err)
		}
	}
	return result
}

func sessionHTTPRequest(t *testing.T, serverURL, method, path, token string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, serverURL+path, nil)
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	status, body := doAuthRequest(t, req)
	return status, []byte(body)
}

func decodeSessions(t *testing.T, body []byte) []sessionDTO {
	t.Helper()
	var response sessionsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("respuesta de sesiones: %v", err)
	}
	return response.Sessions
}

func TestSesionesHTTPPropiasYAjenas(t *testing.T) {
	server, svc, db := newAuthHTTPServer(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")
	createUser(t, db, "bea", "Clave-segura-2", "user")

	ana := loginHTTPDevice(t, server.URL, "ana", "Clave-segura-1", "Escritorio", "web", "")
	if ana.status != http.StatusOK || ana.response.SessionID == 0 {
		t.Fatalf("login ana: %d %s", ana.status, ana.body)
	}
	bea := loginHTTPDevice(t, server.URL, "bea", "Clave-segura-2", "Pixel", "android", "")
	if bea.status != http.StatusOK || bea.response.SessionID == 0 {
		t.Fatalf("login bea: %d %s", bea.status, bea.body)
	}

	status, body := sessionHTTPRequest(t, server.URL, http.MethodGet, "/api/v1/auth/sessions", bea.response.Token)
	if status != http.StatusOK {
		t.Fatalf("GET sesiones propias: %d %s", status, body)
	}
	sessions := decodeSessions(t, body)
	if len(sessions) != 1 || sessions[0].ID != bea.response.SessionID ||
		sessions[0].DeviceName != "Pixel" || sessions[0].Client != ClientAndroid || !sessions[0].Current {
		t.Fatalf("sesiones propias inesperadas: %+v", sessions)
	}
	if strings.Contains(string(body), bea.response.Token) || strings.Contains(string(body), hashToken(bea.response.Token)) {
		t.Fatal("la respuesta no debe contener el token ni su hash")
	}

	status, body = sessionHTTPRequest(t, server.URL, http.MethodDelete,
		fmt.Sprintf("/api/v1/auth/sessions/%d", ana.response.SessionID), bea.response.Token)
	if status != http.StatusNotFound {
		t.Fatalf("DELETE sesión ajena: %d %s", status, body)
	}
	foreignBody := string(body)
	status, body = sessionHTTPRequest(t, server.URL, http.MethodDelete,
		"/api/v1/auth/sessions/999999", bea.response.Token)
	if status != http.StatusNotFound || string(body) != foreignBody {
		t.Fatalf("sesión inexistente = %d %s, se esperaba el mismo 404 que la ajena (%s)",
			status, body, foreignBody)
	}
	if _, err := svc.Authenticate(context.Background(), ana.response.Token); err != nil {
		t.Fatalf("la sesión ajena no debe cerrarse: %v", err)
	}

	status, body = sessionHTTPRequest(t, server.URL, http.MethodDelete,
		fmt.Sprintf("/api/v1/auth/sessions/%d", bea.response.SessionID), bea.response.Token)
	if status != http.StatusNoContent {
		t.Fatalf("DELETE sesión actual: %d %s", status, body)
	}
	if _, err := svc.Authenticate(context.Background(), bea.response.Token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("la sesión actual debe quedar cerrada: %v", err)
	}
}

func TestSesionesHTTPAdministrador(t *testing.T) {
	server, svc, db := newAuthHTTPServer(t)
	createUser(t, db, "root", "Clave-root-1", RoleAdmin)
	beaID := createUser(t, db, "bea", "Clave-segura-2", "user")
	createUser(t, db, "ana", "Clave-segura-1", "user")

	root := loginHTTPDevice(t, server.URL, "root", "Clave-root-1", "Panel", "web", "")
	beaWeb := loginHTTPDevice(t, server.URL, "bea", "Clave-segura-2", "Portátil", "web", "")
	beaPhone := loginHTTPDevice(t, server.URL, "bea", "Clave-segura-2", "Pixel", "android", "")
	ana := loginHTTPDevice(t, server.URL, "ana", "Clave-segura-1", "Ana", "web", "")
	if root.status != http.StatusOK || beaWeb.status != http.StatusOK ||
		beaPhone.status != http.StatusOK || ana.status != http.StatusOK {
		t.Fatalf("logins: root=%d beaWeb=%d beaPhone=%d ana=%d",
			root.status, beaWeb.status, beaPhone.status, ana.status)
	}

	status, body := sessionHTTPRequest(t, server.URL, http.MethodGet,
		fmt.Sprintf("/api/v1/users/%d/sessions", beaID), root.response.Token)
	if status != http.StatusOK {
		t.Fatalf("GET sesiones de bea: %d %s", status, body)
	}
	sessions := decodeSessions(t, body)
	if len(sessions) != 2 {
		t.Fatalf("sesiones de bea = %d", len(sessions))
	}
	for _, session := range sessions {
		if session.Current {
			t.Fatalf("una sesión ajena no puede marcarse como actual: %+v", session)
		}
	}

	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		path := fmt.Sprintf("/api/v1/users/%d/sessions", beaID)
		if method == http.MethodDelete {
			path += fmt.Sprintf("/%d", beaWeb.response.SessionID)
		}
		status, body = sessionHTTPRequest(t, server.URL, method, path, beaWeb.response.Token)
		if status != http.StatusForbidden {
			t.Fatalf("%s admin con usuario normal = %d %s", method, status, body)
		}
	}

	status, body = sessionHTTPRequest(t, server.URL, http.MethodDelete,
		fmt.Sprintf("/api/v1/users/%d/sessions/%d", beaID, beaWeb.response.SessionID), root.response.Token)
	if status != http.StatusNoContent {
		t.Fatalf("DELETE sesión de bea: %d %s", status, body)
	}
	if _, err := svc.Authenticate(context.Background(), beaWeb.response.Token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("la sesión cerrada por el admin sigue válida: %v", err)
	}
	if _, err := svc.Authenticate(context.Background(), beaPhone.response.Token); err != nil {
		t.Fatalf("las otras sesiones de bea deben seguir válidas: %v", err)
	}

	status, foreignBody := sessionHTTPRequest(t, server.URL, http.MethodDelete,
		fmt.Sprintf("/api/v1/users/%d/sessions/%d", beaID, ana.response.SessionID), root.response.Token)
	if status != http.StatusNotFound {
		t.Fatalf("DELETE par ajeno: %d %s", status, foreignBody)
	}
	status, missingBody := sessionHTTPRequest(t, server.URL, http.MethodDelete,
		fmt.Sprintf("/api/v1/users/%d/sessions/999999", beaID), root.response.Token)
	if status != http.StatusNotFound || string(missingBody) != string(foreignBody) {
		t.Fatalf("sesión inexistente = %d %s, se esperaba el mismo 404 (%s)",
			status, missingBody, foreignBody)
	}
	if _, err := svc.Authenticate(context.Background(), ana.response.Token); err != nil {
		t.Fatalf("la sesión de ana no debe cerrarse: %v", err)
	}
}

func TestLoginHTTPDerivaEtiquetaYRechazaCliente(t *testing.T) {
	server, _, db := newAuthHTTPServer(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")

	chrome := loginHTTPDevice(t, server.URL, "ana", "Clave-segura-1", "",
		"web", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/130.0 Safari/537.36")
	if chrome.status != http.StatusOK {
		t.Fatalf("login chrome: %d %s", chrome.status, chrome.body)
	}
	status, body := sessionHTTPRequest(t, server.URL, http.MethodGet,
		"/api/v1/auth/sessions", chrome.response.Token)
	if status != http.StatusOK {
		t.Fatalf("GET sesiones: %d %s", status, body)
	}
	sessions := decodeSessions(t, body)
	if len(sessions) != 1 || sessions[0].DeviceName != "Chrome · Windows" {
		t.Fatalf("etiqueta derivada inesperada: %+v", sessions)
	}

	invalid := loginHTTPDevice(t, server.URL, "ana", "Clave-segura-1", "Tele", "ios", "")
	if invalid.status != http.StatusBadRequest {
		t.Fatalf("cliente inválido: %d %s", invalid.status, invalid.body)
	}

	status, body = sessionHTTPRequest(t, server.URL, http.MethodGet, "/api/v1/auth/sessions", "")
	if status != http.StatusUnauthorized {
		t.Fatalf("GET sin token: %d %s", status, body)
	}
}
