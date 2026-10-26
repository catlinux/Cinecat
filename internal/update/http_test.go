package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/catlinux/cinecat/internal/auth"
)

type updateTestServer struct {
	api      *API
	checker  *Checker
	requests int32
	enabled  bool
}

func newUpdateTestServer(t *testing.T, requester *Requester, enabled *bool) *updateTestServer {
	t.Helper()
	ts := &updateTestServer{}
	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tags":
			atomic.AddInt32(&ts.requests, 1)
			_, _ = w.Write([]byte(`[{"name":"v0.13.1"},{"name":"v0.14.0"}]`))
		case "/raw/v0.14.0/CHANGELOG.md":
			_, _ = w.Write([]byte("## [0.14.0] - 2026-10-26\n- Novedad\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(github.Close)
	ts.checker = NewChecker(CheckerOptions{
		Current: "0.13.1", TagsURL: github.URL + "/tags", RawBaseURL: github.URL + "/raw",
	})
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
	ts.api = NewAPI(authenticate, ts.checker, requester, func() bool { return *enabled })
	return ts
}

func updateRequest(t *testing.T, handler http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestEndpointsUpdateAutenticacionYEstado(t *testing.T) {
	enabled := false
	requester := NewRequester(t.TempDir())
	ts := newUpdateTestServer(t, requester, &enabled)
	mux := http.NewServeMux()
	ts.api.Register(mux)

	if response := updateRequest(t, mux, http.MethodGet, "/api/v1/system/update", "", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("sin sesión = %d", response.Code)
	}
	if response := updateRequest(t, mux, http.MethodGet, "/api/v1/system/update", "user", ""); response.Code != http.StatusForbidden {
		t.Fatalf("sin rol = %d", response.Code)
	}

	response := updateRequest(t, mux, http.MethodGet, "/api/v1/system/update", "admin", "")
	if response.Code != http.StatusOK {
		t.Fatalf("GET desactivado = %d %s", response.Code, response.Body.String())
	}
	if ts.requests != 0 {
		t.Fatalf("el GET con el ajuste apagado consultó GitHub: %d", ts.requests)
	}
	var payload updateResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("respuesta: %v", err)
	}
	if payload.Current != "0.13.1" || payload.CheckEnabled || payload.Latest != "" {
		t.Fatalf("estado inicial inesperado: %+v", payload)
	}

	enabled = true
	response = updateRequest(t, mux, http.MethodGet, "/api/v1/system/update", "admin", "")
	if response.Code != http.StatusOK || ts.requests != 1 {
		t.Fatalf("GET activo = %d, peticiones %d", response.Code, ts.requests)
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("respuesta activa: %v", err)
	}
	if !payload.Available || payload.Latest != "v0.14.0" || !payload.CheckEnabled ||
		!strings.Contains(payload.Notes, "Novedad") {
		t.Fatalf("estado activo inesperado: %+v", payload)
	}

	enabled = false
	response = updateRequest(t, mux, http.MethodPost, "/api/v1/system/update/check", "admin", "")
	if response.Code != http.StatusOK || ts.requests != 2 {
		t.Fatalf("POST check = %d, peticiones %d", response.Code, ts.requests)
	}
}

func TestEndpointsUpdateApplyYErrores(t *testing.T) {
	enabled := false
	dir := t.TempDir()
	requester := NewRequester(dir)
	ts := newUpdateTestServer(t, requester, &enabled)

	apply := func(token, body string) *httptest.ResponseRecorder {
		mux := http.NewServeMux()
		ts.api.Register(mux)
		return updateRequest(t, mux, http.MethodPost, "/api/v1/system/update/apply", token, body)
	}
	if response := apply("admin", `{"tag":"v0.14.0"}`); response.Code != http.StatusBadRequest {
		t.Fatalf("apply sin comprobar = %d", response.Code)
	}
	if _, err := ts.checker.Check(context.Background(), true); err != nil {
		t.Fatalf("Check: %v", err)
	}
	if response := apply("admin", `{"tag":"v0.15.0"}`); response.Code != http.StatusBadRequest {
		t.Fatalf("tag distinto = %d", response.Code)
	}
	response := apply("admin", `{"tag":"v0.14.0"}`)
	if response.Code != http.StatusAccepted {
		t.Fatalf("apply válido = %d %s", response.Code, response.Body.String())
	}
	if _, err := os.Stat(filepath.Join(dir, requestFileName)); err != nil {
		t.Fatalf("no se escribió la petición: %v", err)
	}
	if response := apply("admin", `{"tag":"v0.14.0"}`); response.Code != http.StatusConflict {
		t.Fatalf("segunda apply = %d", response.Code)
	}

	unsupported := newUpdateTestServer(t, NewRequester(filepath.Join(t.TempDir(), "no-existe")), &enabled)
	if _, err := unsupported.checker.Check(context.Background(), true); err != nil {
		t.Fatalf("Check: %v", err)
	}
	mux := http.NewServeMux()
	unsupported.api.Register(mux)
	if response := updateRequest(t, mux, http.MethodPost, "/api/v1/system/update/apply", "admin",
		`{"tag":"v0.14.0"}`); response.Code != http.StatusServiceUnavailable ||
		!strings.Contains(response.Body.String(), "La actualización automática no está disponible") {
		t.Fatalf("apply no soportado = %d %s", response.Code, response.Body.String())
	}

	broken := NewChecker(CheckerOptions{Current: "0.13.1", TagsURL: "http://127.0.0.1:1/tags"})
	brokenAPI := NewAPI(func(ctx context.Context, token string) (auth.User, error) {
		return auth.User{Username: "ana", Role: "admin"}, nil
	}, broken, requester, func() bool { return true })
	mux = http.NewServeMux()
	brokenAPI.Register(mux)
	if response := updateRequest(t, mux, http.MethodPost, "/api/v1/system/update/check", "admin", ""); response.Code != http.StatusBadGateway {
		t.Fatalf("check roto = %d", response.Code)
	}
}
