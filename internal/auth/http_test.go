package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newAuthHTTPServer(t *testing.T) (*httptest.Server, *Service, *sql.DB) {
	t.Helper()
	svc, db, _ := newTestService(t)

	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/auth/login", svc.LoginHandler())
	mux.Handle("POST /api/v1/auth/logout", svc.LogoutHandler())
	mux.Handle("POST /api/v1/auth/password", svc.ChangePasswordHandler())
	mux.Handle("GET /api/v1/auth/me", svc.CurrentUserHandler())
	svc.RegisterUserRoutes(mux)
	svc.RegisterSessionRoutes(mux)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, svc, db
}

func httpLogin(t *testing.T, serverURL, username, password string) (int, string) {
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

	var payload loginResponse
	if resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("respuesta de login: %v", err)
		}
	}
	return resp.StatusCode, payload.Token
}

func httpChangePassword(t *testing.T, serverURL, token, currentPassword, newPassword string) (int, string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{
		"current_password": currentPassword,
		"new_password":     newPassword,
	})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, serverURL+"/api/v1/auth/password", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return doAuthRequest(t, req)
}

func doAuthRequest(t *testing.T, req *http.Request) (int, string) {
	t.Helper()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leer respuesta: %v", err)
	}
	return resp.StatusCode, string(data)
}

func TestChangePasswordHTTPFlujoCompleto(t *testing.T) {
	server, svc, db := newAuthHTTPServer(t)
	createUser(t, db, "ana", "Clave-vieja-1", "user")

	status, token := httpLogin(t, server.URL, "ana", "Clave-vieja-1")
	if status != http.StatusOK || token == "" {
		t.Fatalf("login inicial: código = %d, token vacío = %v", status, token == "")
	}

	status, body := httpChangePassword(t, server.URL, token, "Clave-vieja-1", "Clave-nueva-1")
	if status != http.StatusNoContent {
		t.Fatalf("cambio de contraseña: código = %d, cuerpo = %s", status, body)
	}

	// La sesión actual no se invalida al cambiar la contraseña.
	if _, err := svc.Authenticate(context.Background(), token); err != nil {
		t.Fatalf("la sesión actual debe seguir valiendo: %v", err)
	}

	if status, _ := httpLogin(t, server.URL, "ana", "Clave-vieja-1"); status != http.StatusUnauthorized {
		t.Fatalf("login con la contraseña antigua: código = %d, se esperaba 401", status)
	}
	if status, newToken := httpLogin(t, server.URL, "ana", "Clave-nueva-1"); status != http.StatusOK || newToken == "" {
		t.Fatalf("login con la contraseña nueva: código = %d", status)
	}
}

func TestChangePasswordCierraOtrasSesionesHTTP(t *testing.T) {
	server, svc, db := newAuthHTTPServer(t)
	createUser(t, db, "ana", "Clave-vieja-1", "user")
	createUser(t, db, "bea", "Clave-vieja-1", "user")

	_, current := httpLogin(t, server.URL, "ana", "Clave-vieja-1")
	_, other := httpLogin(t, server.URL, "ana", "Clave-vieja-1")
	_, bea := httpLogin(t, server.URL, "bea", "Clave-vieja-1")

	status, body := httpChangePassword(t, server.URL, current, "Clave-vieja-1", "Clave-nueva-1")
	if status != http.StatusNoContent {
		t.Fatalf("cambio de contraseña: %d %s", status, body)
	}
	if _, err := svc.Authenticate(context.Background(), current); err != nil {
		t.Fatalf("la sesión actual debe seguir válida: %v", err)
	}
	if _, err := svc.Authenticate(context.Background(), other); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("la otra sesión debe quedar invalidada: %v", err)
	}
	if _, err := svc.Authenticate(context.Background(), bea); err != nil {
		t.Fatalf("la sesión de otro usuario no debe tocarse: %v", err)
	}
}

func TestLoginRechazaCuerpoDemasiadoGrande(t *testing.T) {
	server, _, _ := newAuthHTTPServer(t)
	for _, body := range []string{
		strings.Repeat("x", 2<<20),
		`{"username":"` + strings.Repeat("x", 2<<20) + `","password":"x"}`,
	} {
		req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/auth/login", strings.NewReader(body))
		if err != nil {
			t.Fatalf("nueva petición: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		status, response := doAuthRequest(t, req)
		if status != http.StatusRequestEntityTooLarge {
			t.Fatalf("código = %d, se esperaba 413; cuerpo = %s", status, response)
		}
	}
}

func TestChangePasswordHTTPActualIncorrecta(t *testing.T) {
	server, _, db := newAuthHTTPServer(t)
	createUser(t, db, "ana", "Clave-vieja-1", "user")
	_, token := httpLogin(t, server.URL, "ana", "Clave-vieja-1")

	status, body := httpChangePassword(t, server.URL, token, "incorrecta", "Clave-nueva-1")
	if status != http.StatusUnauthorized {
		t.Fatalf("código = %d, se esperaba 401; cuerpo = %s", status, body)
	}
	if status, _ := httpLogin(t, server.URL, "ana", "Clave-vieja-1"); status != http.StatusOK {
		t.Fatalf("la contraseña antigua debe seguir valiendo: código = %d", status)
	}
	if status, _ := httpLogin(t, server.URL, "ana", "Clave-nueva-1"); status != http.StatusUnauthorized {
		t.Fatalf("la contraseña nueva no debe haberse guardado: código = %d", status)
	}
}

func TestChangePasswordHTTPDemasiadoCorta(t *testing.T) {
	server, _, db := newAuthHTTPServer(t)
	createUser(t, db, "ana", "Clave-vieja-1", "user")
	_, token := httpLogin(t, server.URL, "ana", "Clave-vieja-1")

	status, body := httpChangePassword(t, server.URL, token, "Clave-vieja-1", "corta")
	if status != http.StatusBadRequest {
		t.Fatalf("código = %d, se esperaba 400; cuerpo = %s", status, body)
	}
	if status, _ := httpLogin(t, server.URL, "ana", "Clave-vieja-1"); status != http.StatusOK {
		t.Fatalf("la contraseña antigua debe seguir valiendo: código = %d", status)
	}
}

func TestChangePasswordHTTPSinToken(t *testing.T) {
	server, _, _ := newAuthHTTPServer(t)

	status, body := httpChangePassword(t, server.URL, "", "Clave-vieja-1", "Clave-nueva-1")
	if status != http.StatusUnauthorized {
		t.Fatalf("código = %d, se esperaba 401; cuerpo = %s", status, body)
	}
}

func TestChangePasswordHTTPTokenInvalido(t *testing.T) {
	server, _, _ := newAuthHTTPServer(t)

	status, body := httpChangePassword(t, server.URL, "token-inventado", "Clave-vieja-1", "Clave-nueva-1")
	if status != http.StatusUnauthorized {
		t.Fatalf("código = %d, se esperaba 401; cuerpo = %s", status, body)
	}
}

func TestChangePasswordHTTPCuerpoInvalido(t *testing.T) {
	server, _, db := newAuthHTTPServer(t)
	createUser(t, db, "ana", "Clave-vieja-1", "user")
	_, token := httpLogin(t, server.URL, "ana", "Clave-vieja-1")

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/auth/password", bytes.NewBufferString("{no es json"))
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	status, body := doAuthRequest(t, req)
	if status != http.StatusBadRequest {
		t.Fatalf("código = %d, se esperaba 400; cuerpo = %s", status, body)
	}
}

func httpCurrentUser(t *testing.T, serverURL, token string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, serverURL+"/api/v1/auth/me", nil)
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return doAuthRequest(t, req)
}

func TestCurrentUserHTTP(t *testing.T) {
	server, _, db := newAuthHTTPServer(t)
	createUser(t, db, "ana", "Clave-segura-1", "admin")

	status, token := httpLogin(t, server.URL, "ana", "Clave-segura-1")
	if status != http.StatusOK || token == "" {
		t.Fatalf("login: código = %d", status)
	}

	status, body := httpCurrentUser(t, server.URL, token)
	if status != http.StatusOK {
		t.Fatalf("me: código = %d, cuerpo = %s", status, body)
	}
	var user currentUserResponse
	if err := json.Unmarshal([]byte(body), &user); err != nil {
		t.Fatalf("respuesta de me: %v", err)
	}
	if user.Username != "ana" || user.Role != "admin" {
		t.Fatalf("usuario inesperado: %+v", user)
	}
}

func TestCurrentUserHTTPSinToken(t *testing.T) {
	server, _, _ := newAuthHTTPServer(t)

	status, body := httpCurrentUser(t, server.URL, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("código = %d, se esperaba 401; cuerpo = %s", status, body)
	}
}

func TestCurrentUserHTTPTokenInvalido(t *testing.T) {
	server, _, _ := newAuthHTTPServer(t)

	status, body := httpCurrentUser(t, server.URL, "token-inventado")
	if status != http.StatusUnauthorized {
		t.Fatalf("código = %d, se esperaba 401; cuerpo = %s", status, body)
	}
}
