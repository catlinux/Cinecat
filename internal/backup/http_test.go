package backup

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/catlinux/cinecat/internal/auth"
)

func newBackupTestAPI(t *testing.T) (*Manager, *API, *http.ServeMux) {
	t.Helper()
	db, path := openBackupDB(t)
	now := time.Date(2026, time.March, 4, 8, 9, 10, 0, time.UTC)
	manager := New(db, path).WithClock(func() time.Time { return now })
	authenticate := func(ctx context.Context, token string) (auth.User, error) {
		switch token {
		case "admin":
			return auth.User{ID: 1, Username: "ana", Role: "admin"}, nil
		case "user":
			return auth.User{ID: 2, Username: "bea", Role: "user"}, nil
		default:
			return auth.User{}, auth.ErrInvalidSession
		}
	}
	api := NewAPI(authenticate, manager, func() bool { return true }, func() int { return 9 })
	mux := http.NewServeMux()
	api.Register(mux)
	return manager, api, mux
}

func backupRequest(t *testing.T, handler http.Handler, method, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(""))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestAPICopiasRequiereAdministrador(t *testing.T) {
	_, _, mux := newBackupTestAPI(t)

	if response := backupRequest(t, mux, http.MethodGet, "/api/v1/system/backups", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("sin sesión = %d", response.Code)
	}
	if response := backupRequest(t, mux, http.MethodGet, "/api/v1/system/backups", "user"); response.Code != http.StatusForbidden {
		t.Fatalf("sin rol = %d", response.Code)
	}
	if response := backupRequest(t, mux, http.MethodPost, "/api/v1/system/backups", "user"); response.Code != http.StatusForbidden {
		t.Fatalf("POST sin rol = %d", response.Code)
	}
}

func TestAPICopiasListaYCrear(t *testing.T) {
	manager, _, mux := newBackupTestAPI(t)

	response := backupRequest(t, mux, http.MethodGet, "/api/v1/system/backups", "admin")
	if response.Code != http.StatusOK {
		t.Fatalf("GET = %d %s", response.Code, response.Body.String())
	}
	var list listResponse
	if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
		t.Fatalf("respuesta GET: %v", err)
	}
	if !list.Enabled || list.RetentionDays != 9 || len(list.Backups) != 0 || list.LastError != "" {
		t.Fatalf("respuesta inicial = %+v", list)
	}

	response = backupRequest(t, mux, http.MethodPost, "/api/v1/system/backups", "admin")
	if response.Code != http.StatusCreated {
		t.Fatalf("POST = %d %s", response.Code, response.Body.String())
	}
	var created Copy
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatalf("respuesta POST: %v", err)
	}
	if created.Name != "cinecat-20260304-080910.db" || created.SizeBytes <= 0 ||
		created.CreatedAt != "2026-03-04T08:09:10Z" {
		t.Fatalf("copia creada = %+v", created)
	}

	response = backupRequest(t, mux, http.MethodGet, "/api/v1/system/backups", "admin")
	if response.Code != http.StatusOK {
		t.Fatalf("GET tras crear = %d", response.Code)
	}
	if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
		t.Fatalf("respuesta GET: %v", err)
	}
	if len(list.Backups) != 1 || list.Backups[0].Name != created.Name {
		t.Fatalf("lista = %+v", list)
	}
	if strings.Contains(response.Body.String(), manager.Dir()) {
		t.Fatal("la respuesta expone la ruta absoluta")
	}
}

func TestAPICopiasConflicto(t *testing.T) {
	manager, _, mux := newBackupTestAPI(t)

	manager.runMu.Lock()
	response := backupRequest(t, mux, http.MethodPost, "/api/v1/system/backups", "admin")
	manager.runMu.Unlock()
	if response.Code != http.StatusConflict {
		t.Fatalf("POST con copia en curso = %d %s", response.Code, response.Body.String())
	}
}
