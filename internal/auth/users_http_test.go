package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"testing"
)

func usersHTTPRequest(t *testing.T, serverURL, method, path, token string, body any) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, serverURL+path, reader)
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	status, data := doAuthRequest(t, req)
	return status, []byte(data)
}

func usernamesInList(t *testing.T, body []byte) []string {
	t.Helper()
	var response usersResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("respuesta de usuarios: %v", err)
	}
	names := make([]string, 0, len(response.Users))
	for _, user := range response.Users {
		names = append(names, user.Username)
	}
	return names
}

func listedUser(t *testing.T, body []byte, username string) userDTO {
	t.Helper()
	var response usersResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("respuesta de usuarios: %v", err)
	}
	for _, user := range response.Users {
		if user.Username == username {
			return user
		}
	}
	t.Fatalf("no aparece el usuario %q", username)
	return userDTO{}
}

func TestUsersHTTPCRUD(t *testing.T) {
	server, svc, db := newAuthHTTPServer(t)
	firstLibrary := createAuthLibrary(t, db, "/media/primera", "Primera")
	secondLibrary := createAuthLibrary(t, db, "/media/segunda", "Segunda")
	createUser(t, db, "root", "Clave-root-1", RoleAdmin)
	rootToken := loginTokenForUsers(t, server.URL, "root", "Clave-root-1")

	status, body := usersHTTPRequest(t, server.URL, http.MethodGet, "/api/v1/users", rootToken, nil)
	if status != http.StatusOK {
		t.Fatalf("GET usuarios: %d %s", status, body)
	}
	if names := usernamesInList(t, body); len(names) != 1 || names[0] != "root" {
		t.Fatalf("usuarios iniciales = %v", names)
	}
	root := listedUser(t, body, "root")
	if !root.AllLibraries || root.Disabled || root.Role != RoleAdmin || root.ActiveSessions != 1 {
		t.Fatalf("admin inicial inesperado: %+v", root)
	}

	status, body = usersHTTPRequest(t, server.URL, http.MethodPost, "/api/v1/users", rootToken,
		map[string]any{
			"username":      "bea",
			"password":      "Clave-bea-1",
			"role":          "user",
			"all_libraries": false,
			"library_ids":   []int64{firstLibrary},
		})
	if status != http.StatusCreated {
		t.Fatalf("POST usuario: %d %s", status, body)
	}
	var bea userDTO
	if err := json.Unmarshal(body, &bea); err != nil {
		t.Fatalf("usuario creado: %v", err)
	}
	if bea.ID == 0 || bea.Username != "bea" || bea.Role != RoleUser || bea.Disabled ||
		bea.AllLibraries || bea.CreatedAt == "" || bea.ActiveSessions != 0 ||
		len(bea.LibraryIDs) != 1 || bea.LibraryIDs[0] != firstLibrary {
		t.Fatalf("usuario creado inesperado: %+v", bea)
	}

	if status, body := usersHTTPRequest(t, server.URL, http.MethodPost, "/api/v1/users", rootToken,
		map[string]any{
			"username": "BEA", "password": "Clave-bea-1", "role": "user",
		}); status != http.StatusConflict {
		t.Fatalf("duplicado: %d %s", status, body)
	}
	if status, body := usersHTTPRequest(t, server.URL, http.MethodPost, "/api/v1/users", rootToken,
		map[string]any{
			"username": "carla", "password": "corta", "role": "user",
		}); status != http.StatusBadRequest {
		t.Fatalf("contraseña corta: %d %s", status, body)
	}
	if status, body := usersHTTPRequest(t, server.URL, http.MethodPost, "/api/v1/users", rootToken,
		map[string]any{
			"username": "carla", "password": "Clave-carla-1", "role": "user",
			"library_ids": []int64{999999},
		}); status != http.StatusBadRequest {
		t.Fatalf("biblioteca inexistente: %d %s", status, body)
	}

	beaToken := loginTokenForUsers(t, server.URL, "bea", "Clave-bea-1")
	status, body = usersHTTPRequest(t, server.URL, http.MethodGet, "/api/v1/users", rootToken, nil)
	if status != http.StatusOK {
		t.Fatalf("GET usuarios: %d %s", status, body)
	}
	if listed := listedUser(t, body, "bea"); listed.ActiveSessions != 1 {
		t.Fatalf("sesiones activas = %d", listed.ActiveSessions)
	}

	status, body = usersHTTPRequest(t, server.URL, http.MethodPatch,
		"/api/v1/users/"+strconv.FormatInt(bea.ID, 10), rootToken,
		map[string]any{"all_libraries": true})
	if status != http.StatusOK {
		t.Fatalf("PATCH usuario: %d %s", status, body)
	}
	if err := json.Unmarshal(body, &bea); err != nil {
		t.Fatalf("usuario actualizado: %v", err)
	}
	if !bea.AllLibraries || bea.Role != RoleUser || bea.Disabled {
		t.Fatalf("usuario tras PATCH inesperado: %+v", bea)
	}

	status, body = usersHTTPRequest(t, server.URL, http.MethodPut,
		"/api/v1/users/"+strconv.FormatInt(bea.ID, 10)+"/libraries", rootToken,
		map[string]any{"library_ids": []int64{secondLibrary}})
	if status != http.StatusOK {
		t.Fatalf("PUT bibliotecas: %d %s", status, body)
	}
	if err := json.Unmarshal(body, &bea); err != nil {
		t.Fatalf("bibliotecas actualizadas: %v", err)
	}
	if len(bea.LibraryIDs) != 1 || bea.LibraryIDs[0] != secondLibrary {
		t.Fatalf("bibliotecas = %v", bea.LibraryIDs)
	}

	if status, body := usersHTTPRequest(t, server.URL, http.MethodPut,
		"/api/v1/users/"+strconv.FormatInt(bea.ID, 10)+"/libraries", rootToken,
		map[string]any{"library_ids": []int64{999999}}); status != http.StatusBadRequest {
		t.Fatalf("PUT biblioteca inexistente: %d %s", status, body)
	}

	status, body = usersHTTPRequest(t, server.URL, http.MethodDelete,
		"/api/v1/users/"+strconv.FormatInt(bea.ID, 10), rootToken, nil)
	if status != http.StatusNoContent {
		t.Fatalf("DELETE usuario: %d %s", status, body)
	}
	if _, err := svc.Authenticate(context.Background(), beaToken); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("borrar debe cerrar la sesión: %v", err)
	}
}

func TestUsersHTTPUsuarioNormalRecibe403(t *testing.T) {
	server, _, db := newAuthHTTPServer(t)
	rootID := createUser(t, db, "root", "Clave-root-1", RoleAdmin)
	createUser(t, db, "bea", "Clave-bea-1", RoleUser)
	token := loginTokenForUsers(t, server.URL, "bea", "Clave-bea-1")

	requests := []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodGet, "/api/v1/users", nil},
		{http.MethodPost, "/api/v1/users", map[string]any{"username": "carla"}},
		{http.MethodPatch, "/api/v1/users/" + strconv.FormatInt(rootID, 10), map[string]any{"disabled": true}},
		{http.MethodPut, "/api/v1/users/" + strconv.FormatInt(rootID, 10) + "/libraries", map[string]any{"library_ids": []int64{}}},
		{http.MethodPost, "/api/v1/users/" + strconv.FormatInt(rootID, 10) + "/password", map[string]any{"password": "Clave-nueva-1"}},
		{http.MethodDelete, "/api/v1/users/" + strconv.FormatInt(rootID, 10), nil},
	}
	for _, request := range requests {
		status, body := usersHTTPRequest(t, server.URL, request.method, request.path, token, request.body)
		if status != http.StatusForbidden {
			t.Fatalf("%s %s: código = %d, cuerpo = %s", request.method, request.path, status, body)
		}
	}
}

func TestUsersHTTPNoPuedeModificarSuPropiaCuenta(t *testing.T) {
	server, _, db := newAuthHTTPServer(t)
	rootID := createUser(t, db, "root", "Clave-root-1", RoleAdmin)
	token := loginTokenForUsers(t, server.URL, "root", "Clave-root-1")
	base := "/api/v1/users/" + strconv.FormatInt(rootID, 10)

	for _, request := range []struct {
		method string
		path   string
		body   any
	}{
		{http.MethodDelete, base, nil},
		{http.MethodPatch, base, map[string]any{"disabled": true}},
		{http.MethodPatch, base, map[string]any{"role": "user"}},
	} {
		status, body := usersHTTPRequest(t, server.URL, request.method, request.path, token, request.body)
		if status != http.StatusForbidden {
			t.Fatalf("%s %s: código = %d, cuerpo = %s", request.method, request.path, status, body)
		}
	}

	status, body := usersHTTPRequest(t, server.URL, http.MethodPatch, base, token,
		map[string]any{"all_libraries": false})
	if status != http.StatusOK {
		t.Fatalf("cambiar all_libraries propio: %d %s", status, body)
	}
	if status, body := usersHTTPRequest(t, server.URL, http.MethodPatch, base, token,
		map[string]any{"role": "otro"}); status != http.StatusBadRequest {
		t.Fatalf("rol inválido: %d %s", status, body)
	}
}

func TestUsersHTTPRestablecerPassword(t *testing.T) {
	server, svc, db := newAuthHTTPServer(t)
	createUser(t, db, "root", "Clave-root-1", RoleAdmin)
	beaID := createUser(t, db, "bea", "Clave-vieja-1", RoleUser)
	rootToken := loginTokenForUsers(t, server.URL, "root", "Clave-root-1")
	first := loginTokenForUsers(t, server.URL, "bea", "Clave-vieja-1")
	second := loginTokenForUsers(t, server.URL, "bea", "Clave-vieja-1")
	path := "/api/v1/users/" + strconv.FormatInt(beaID, 10) + "/password"

	if status, body := usersHTTPRequest(t, server.URL, http.MethodPost, path, rootToken,
		map[string]any{"password": "corta"}); status != http.StatusBadRequest {
		t.Fatalf("contraseña corta: %d %s", status, body)
	}
	if _, err := svc.Authenticate(context.Background(), first); err != nil {
		t.Fatalf("el error no debe cerrar sesiones: %v", err)
	}

	if status, body := usersHTTPRequest(t, server.URL, http.MethodPost, path, rootToken,
		map[string]any{"password": "Clave-nueva-1"}); status != http.StatusNoContent {
		t.Fatalf("restablecer: %d %s", status, body)
	}
	for _, token := range []string{first, second} {
		if _, err := svc.Authenticate(context.Background(), token); !errors.Is(err, ErrInvalidSession) {
			t.Fatalf("la sesión debe quedar cerrada: %v", err)
		}
	}
	if status := loginStatusForUsers(t, server.URL, "bea", "Clave-vieja-1"); status != http.StatusUnauthorized {
		t.Fatalf("contraseña antigua: %d", status)
	}
	if status := loginStatusForUsers(t, server.URL, "bea", "Clave-nueva-1"); status != http.StatusOK {
		t.Fatalf("contraseña nueva: %d", status)
	}

	if status, body := usersHTTPRequest(t, server.URL, http.MethodPost,
		"/api/v1/users/999999/password", rootToken,
		map[string]any{"password": "Clave-nueva-1"}); status != http.StatusNotFound {
		t.Fatalf("usuario inexistente: %d %s", status, body)
	}
}

func TestUsersHTTPDesactivarYBorrar(t *testing.T) {
	server, svc, db := newAuthHTTPServer(t)
	libraryID := createAuthLibrary(t, db, "/media/bea", "Bea")
	createUser(t, db, "root", "Clave-root-1", RoleAdmin)
	beaID := createUser(t, db, "bea", "Clave-bea-1", RoleUser)
	if _, err := svc.SetUserLibraries(context.Background(), beaID, []int64{libraryID}); err != nil {
		t.Fatalf("SetUserLibraries: %v", err)
	}
	rootToken := loginTokenForUsers(t, server.URL, "root", "Clave-root-1")
	beaToken := loginTokenForUsers(t, server.URL, "bea", "Clave-bea-1")
	path := "/api/v1/users/" + strconv.FormatInt(beaID, 10)

	status, body := usersHTTPRequest(t, server.URL, http.MethodPatch, path, rootToken,
		map[string]any{"disabled": true})
	if status != http.StatusOK {
		t.Fatalf("desactivar: %d %s", status, body)
	}
	if _, err := svc.Authenticate(context.Background(), beaToken); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("desactivar debe cerrar la sesión: %v", err)
	}
	if status := loginStatusForUsers(t, server.URL, "bea", "Clave-bea-1"); status != http.StatusUnauthorized {
		t.Fatalf("login desactivado: %d", status)
	}

	if status, body := usersHTTPRequest(t, server.URL, http.MethodPatch, path, rootToken,
		map[string]any{"disabled": false}); status != http.StatusOK {
		t.Fatalf("reactivar: %d %s", status, body)
	}
	beaToken = loginTokenForUsers(t, server.URL, "bea", "Clave-bea-1")

	if status, body := usersHTTPRequest(t, server.URL, http.MethodDelete, path, rootToken, nil); status != http.StatusNoContent {
		t.Fatalf("borrar: %d %s", status, body)
	}
	if _, err := svc.Authenticate(context.Background(), beaToken); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("borrar debe cerrar la sesión: %v", err)
	}
	var sessions, links int
	if err := db.QueryRow(`SELECT count(*) FROM sessions WHERE user_id = ?`, beaID).Scan(&sessions); err != nil {
		t.Fatalf("contar sesiones: %v", err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM user_libraries WHERE user_id = ?`, beaID).Scan(&links); err != nil {
		t.Fatalf("contar bibliotecas: %v", err)
	}
	if sessions != 0 || links != 0 {
		t.Fatalf("quedan sessions=%d user_libraries=%d", sessions, links)
	}
}

func TestCurrentUserIncluyeAccesoABibliotecas(t *testing.T) {
	server, svc, db := newAuthHTTPServer(t)
	libraryID := createAuthLibrary(t, db, "/media/ana", "Ana")
	user, err := svc.CreateUser(context.Background(), "ana", "Clave-ana-1", RoleUser, false, []int64{libraryID})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	token := loginTokenForUsers(t, server.URL, "ana", "Clave-ana-1")
	status, body := usersHTTPRequest(t, server.URL, http.MethodGet, "/api/v1/auth/me", token, nil)
	if status != http.StatusOK {
		t.Fatalf("me: %d %s", status, body)
	}
	var response currentUserResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("respuesta de me: %v", err)
	}
	if response.Username != user.Username || response.Role != RoleUser ||
		response.AllLibraries || len(response.LibraryIDs) != 1 || response.LibraryIDs[0] != libraryID {
		t.Fatalf("me inesperado: %+v", response)
	}
}

func loginTokenForUsers(t *testing.T, serverURL, username, password string) string {
	t.Helper()
	status, token := httpLogin(t, serverURL, username, password)
	if status != http.StatusOK || token == "" {
		t.Fatalf("login %q: código = %d", username, status)
	}
	return token
}

func loginStatusForUsers(t *testing.T, serverURL, username, password string) int {
	t.Helper()
	status, _ := httpLogin(t, serverURL, username, password)
	return status
}
