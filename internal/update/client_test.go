package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestClientUpdateRequiereSesionYRespetaComprobacionDesactivada(t *testing.T) {
	enabled := false
	ts := newUpdateTestServer(t, nil, &enabled)
	mux := http.NewServeMux()
	ts.api.Register(mux)
	path := "/api/v1/system/client-update?platform=android&version_code=1000"

	if response := updateRequest(t, mux, http.MethodGet, path, "", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("sin sesión = %d", response.Code)
	}
	if response := updateRequest(t, mux, http.MethodGet, path, "user", ""); response.Code != http.StatusNoContent {
		t.Fatalf("con comprobación desactivada = %d %s", response.Code, response.Body.String())
	}
	if ts.requests != 0 {
		t.Fatalf("la comprobación desactivada consultó GitHub: %d", ts.requests)
	}
}

func TestClientUpdate204SinActualizacion(t *testing.T) {
	enabled := true
	ts := newUpdateTestServer(t, nil, &enabled)
	var shaRequests int32
	sha := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&shaRequests, 1)
		_, _ = w.Write([]byte(strings.Repeat("a", 64) + "  cinecat.apk\n"))
	}))
	defer sha.Close()
	ts.api.WithClientUpdateBaseURL(sha.URL)
	mux := http.NewServeMux()
	ts.api.Register(mux)

	for _, code := range []string{"1301", "9999"} {
		path := "/api/v1/system/client-update?platform=android&version_code=" + code
		if response := updateRequest(t, mux, http.MethodGet, path, "user", ""); response.Code != http.StatusNoContent {
			t.Fatalf("version_code=%s: código = %d, cuerpo = %s", code, response.Code, response.Body.String())
		}
	}
	ts.checker.current = "dev"
	if response := updateRequest(t, mux, http.MethodGet,
		"/api/v1/system/client-update?platform=android&version_code=1", "user", ""); response.Code != http.StatusNoContent {
		t.Fatalf("versión inválida: código = %d, cuerpo = %s", response.Code, response.Body.String())
	}
	if shaRequests != 0 {
		t.Fatalf("se pidió el SHA sin actualización: %d", shaRequests)
	}
}

func TestClientUpdate400ConVersionCodeInvalido(t *testing.T) {
	enabled := true
	ts := newUpdateTestServer(t, nil, &enabled)
	mux := http.NewServeMux()
	ts.api.Register(mux)

	for _, query := range []string{
		"platform=android",
		"platform=android&version_code=",
		"platform=android&version_code=abc",
		"platform=android&version_code=-1",
		"platform=android&version_code=1.5",
	} {
		response := updateRequest(t, mux, http.MethodGet,
			"/api/v1/system/client-update?"+query, "user", "")
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s: código = %d, cuerpo = %s", query, response.Code, response.Body.String())
		}
	}
	if response := updateRequest(t, mux, http.MethodGet,
		"/api/v1/system/client-update?platform=ios&version_code=1", "user", ""); response.Code != http.StatusNoContent {
		t.Fatalf("plataforma distinta: código = %d", response.Code)
	}
}

func TestClientUpdate200ConSHAYCaches(t *testing.T) {
	enabled := true
	ts := newUpdateTestServer(t, nil, &enabled)
	var shaRequests int32
	sha := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&shaRequests, 1)
		if r.URL.Path != "/v0.13.1/cinecat-0.13.1.apk.sha256" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(strings.ToUpper(strings.Repeat("a", 64)) + "  cinecat-0.13.1.apk\n"))
	}))
	defer sha.Close()
	ts.api.WithClientUpdateBaseURL(sha.URL)
	mux := http.NewServeMux()
	ts.api.Register(mux)
	path := "/api/v1/system/client-update?platform=android&version_code=1000"

	response := updateRequest(t, mux, http.MethodGet, path, "user", "")
	if response.Code != http.StatusOK {
		t.Fatalf("código = %d, cuerpo = %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", response.Header().Get("Cache-Control"))
	}
	var payload struct {
		Version     string `json:"version"`
		VersionCode int    `json:"version_code"`
		URL         string `json:"url"`
		SHA256      string `json:"sha256"`
		Notes       string `json:"notes"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("respuesta: %v", err)
	}
	if payload.Version != "0.13.1" || payload.VersionCode != 1301 {
		t.Fatalf("versión inesperada: %+v", payload)
	}
	if payload.URL != sha.URL+"/v0.13.1/cinecat-0.13.1.apk" {
		t.Fatalf("URL = %q", payload.URL)
	}
	if payload.SHA256 != strings.Repeat("a", 64) {
		t.Fatalf("SHA = %q", payload.SHA256)
	}
	if payload.Notes != "" {
		t.Fatalf("notas inesperadas: %q", payload.Notes)
	}
	if shaRequests != 1 {
		t.Fatalf("peticiones SHA = %d", shaRequests)
	}

	response = updateRequest(t, mux, http.MethodGet, path, "user", "")
	if response.Code != http.StatusOK || shaRequests != 1 {
		t.Fatalf("la caché del SHA no se reutilizó: código %d, peticiones %d", response.Code, shaRequests)
	}
}

func TestClientUpdate204SiSHAFallaYCacheaElFallo(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
	}{
		{name: "404", status: http.StatusNotFound, body: "no"},
		{name: "formato", status: http.StatusOK, body: "no-es-un-sha"},
	} {
		t.Run(test.name, func(t *testing.T) {
			enabled := true
			ts := newUpdateTestServer(t, nil, &enabled)
			var shaRequests int32
			sha := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&shaRequests, 1)
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer sha.Close()
			ts.api.WithClientUpdateBaseURL(sha.URL)
			mux := http.NewServeMux()
			ts.api.Register(mux)
			path := "/api/v1/system/client-update?platform=android&version_code=1000"

			for i := 0; i < 2; i++ {
				response := updateRequest(t, mux, http.MethodGet, path, "user", "")
				if response.Code != http.StatusNoContent {
					t.Fatalf("llamada %d: código = %d, cuerpo = %s", i+1, response.Code, response.Body.String())
				}
			}
			if shaRequests != 1 {
				t.Fatalf("el fallo no se cacheó: %d peticiones", shaRequests)
			}
		})
	}
}

func TestClientUpdate200IncluyeNotasCacheadas(t *testing.T) {
	enabled := true
	ts := newUpdateTestServer(t, nil, &enabled)
	ts.checker.current = "0.14.0"
	if _, err := ts.checker.Check(context.Background(), true); err != nil {
		t.Fatalf("Check: %v", err)
	}
	var shaRequests int32
	sha := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&shaRequests, 1)
		_, _ = w.Write([]byte(strings.Repeat("b", 64) + "  cinecat-0.14.0.apk\n"))
	}))
	defer sha.Close()
	ts.api.WithClientUpdateBaseURL(sha.URL)
	mux := http.NewServeMux()
	ts.api.Register(mux)

	response := updateRequest(t, mux, http.MethodGet,
		"/api/v1/system/client-update?platform=android&version_code=1000", "user", "")
	if response.Code != http.StatusOK {
		t.Fatalf("código = %d, cuerpo = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Version string `json:"version"`
		Notes   string `json:"notes"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("respuesta: %v", err)
	}
	if payload.Version != "0.14.0" || !strings.Contains(payload.Notes, "Novedad") {
		t.Fatalf("respuesta inesperada: %+v", payload)
	}
}
