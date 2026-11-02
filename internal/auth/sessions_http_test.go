package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func sessionProfileHTTPRequest(
	t *testing.T,
	serverURL, method, path, token string,
	body any,
) (int, []byte) {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
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
	status, response := doAuthRequest(t, req)
	return status, []byte(response)
}

func decodeSessionProfile(t *testing.T, body []byte) sessionProfileDTO {
	t.Helper()
	var profile sessionProfileDTO
	if err := json.Unmarshal(body, &profile); err != nil {
		t.Fatalf("perfil de sesión: %v", err)
	}
	return profile
}

func TestPerfilDeSesionHTTPActualYListado(t *testing.T) {
	server, _, db := newAuthHTTPServer(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")
	login := loginHTTPDevice(t, server.URL, "ana", "Clave-segura-1", "Portátil", "web", "")
	if login.status != http.StatusOK {
		t.Fatalf("login: %d %s", login.status, login.body)
	}

	status, body := sessionHTTPRequest(t, server.URL, http.MethodGet,
		"/api/v1/auth/sessions/current/profile", login.response.Token)
	if status != http.StatusOK {
		t.Fatalf("GET perfil vacío: %d %s", status, body)
	}
	if profile := decodeSessionProfile(t, body); profile != (sessionProfileDTO{}) {
		t.Fatalf("perfil inicial = %+v", profile)
	}

	want := sessionProfileDTO{
		MaxQuality:       "720",
		AudioLanguage:    "spa",
		SubtitleLanguage: "off",
	}
	status, body = sessionProfileHTTPRequest(t, server.URL, http.MethodPut,
		"/api/v1/auth/sessions/current/profile", login.response.Token, want)
	if status != http.StatusOK {
		t.Fatalf("PUT perfil: %d %s", status, body)
	}
	if profile := decodeSessionProfile(t, body); profile != want {
		t.Fatalf("perfil guardado = %+v, se esperaba %+v", profile, want)
	}

	status, body = sessionHTTPRequest(t, server.URL, http.MethodGet,
		"/api/v1/auth/sessions/current/profile", login.response.Token)
	if profile := decodeSessionProfile(t, body); status != http.StatusOK || profile != want {
		t.Fatalf("GET perfil guardado: %d %+v", status, profile)
	}

	status, body = sessionHTTPRequest(t, server.URL, http.MethodGet,
		"/api/v1/auth/sessions", login.response.Token)
	if status != http.StatusOK {
		t.Fatalf("GET sesiones: %d %s", status, body)
	}
	sessions := decodeSessions(t, body)
	if len(sessions) != 1 || sessions[0].Profile != want {
		t.Fatalf("perfil en el listado = %+v", sessions)
	}

	status, body = sessionHTTPRequest(t, server.URL, http.MethodGet,
		"/api/v1/auth/sessions/"+jsonNumber(login.response.SessionID)+"/profile", login.response.Token)
	if profile := decodeSessionProfile(t, body); status != http.StatusOK || profile != want {
		t.Fatalf("GET perfil por id: %d %+v", status, profile)
	}

	status, body = sessionProfileHTTPRequest(t, server.URL, http.MethodPut,
		"/api/v1/auth/sessions/current/profile", login.response.Token,
		sessionProfileDTO{})
	if status != http.StatusOK {
		t.Fatalf("PUT perfil vacío: %d %s", status, body)
	}
	if profile := decodeSessionProfile(t, body); profile != (sessionProfileDTO{}) {
		t.Fatalf("perfil vacío = %+v", profile)
	}
}

func jsonNumber(value int64) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func TestPerfilDeSesionAjenaOinexistenteDevuelve404(t *testing.T) {
	server, _, db := newAuthHTTPServer(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")
	createUser(t, db, "bea", "Clave-segura-2", "user")
	ana := loginHTTPDevice(t, server.URL, "ana", "Clave-segura-1", "Ana", "web", "")
	bea := loginHTTPDevice(t, server.URL, "bea", "Clave-segura-2", "Bea", "web", "")
	if ana.status != http.StatusOK || bea.status != http.StatusOK {
		t.Fatalf("logins: ana=%d bea=%d", ana.status, bea.status)
	}
	want := sessionProfileDTO{MaxQuality: "480", AudioLanguage: "eng", SubtitleLanguage: "eng"}
	if status, body := sessionProfileHTTPRequest(t, server.URL, http.MethodPut,
		"/api/v1/auth/sessions/current/profile", bea.response.Token, want); status != http.StatusOK {
		t.Fatalf("PUT perfil de bea: %d %s", status, body)
	}

	foreign := "/api/v1/auth/sessions/" + jsonNumber(bea.response.SessionID) + "/profile"
	status, foreignBody := sessionHTTPRequest(t, server.URL, http.MethodGet, foreign, ana.response.Token)
	if status != http.StatusNotFound {
		t.Fatalf("GET perfil ajeno: %d %s", status, foreignBody)
	}
	missing := "/api/v1/auth/sessions/999999/profile"
	status, missingBody := sessionHTTPRequest(t, server.URL, http.MethodGet, missing, ana.response.Token)
	if status != http.StatusNotFound || string(missingBody) != string(foreignBody) {
		t.Fatalf("GET perfil inexistente: %d %s (ajena %s)", status, missingBody, foreignBody)
	}

	status, body := sessionProfileHTTPRequest(t, server.URL, http.MethodPut,
		foreign, ana.response.Token, sessionProfileDTO{MaxQuality: "1080"})
	if status != http.StatusNotFound {
		t.Fatalf("PUT perfil ajeno: %d %s", status, body)
	}
	status, body = sessionHTTPRequest(t, server.URL, http.MethodGet,
		"/api/v1/auth/sessions/current/profile", bea.response.Token)
	if profile := decodeSessionProfile(t, body); status != http.StatusOK || profile != want {
		t.Fatalf("el perfil ajeno cambió: %d %+v", status, profile)
	}

	status, body = sessionHTTPRequest(t, server.URL, http.MethodGet,
		"/api/v1/auth/sessions/current/profile", "")
	if status != http.StatusUnauthorized {
		t.Fatalf("GET perfil sin sesión: %d %s", status, body)
	}
}

func TestPerfilDeSesionValidaSusCampos(t *testing.T) {
	server, _, db := newAuthHTTPServer(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")
	login := loginHTTPDevice(t, server.URL, "ana", "Clave-segura-1", "Ana", "web", "")
	if login.status != http.StatusOK {
		t.Fatalf("login: %d %s", login.status, login.body)
	}
	cases := []struct {
		name string
		body map[string]string
	}{
		{"calidad auto", map[string]string{"max_quality": "auto"}},
		{"calidad con p", map[string]string{"max_quality": "1080p"}},
		{"audio mayúsculas", map[string]string{"audio_language": "ES"}},
		{"audio corto", map[string]string{"audio_language": "e"}},
		{"audio con guion", map[string]string{"audio_language": "es-ES"}},
		{"subtítulo largo", map[string]string{"subtitle_language": "subs"}},
		{"subtítulo off mayúsculas", map[string]string{"subtitle_language": "OFF"}},
		{"campo desconocido", map[string]string{"extra": "x"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			status, body := sessionProfileHTTPRequest(t, server.URL, http.MethodPut,
				"/api/v1/auth/sessions/current/profile", login.response.Token, test.body)
			if status != http.StatusBadRequest {
				t.Fatalf("PUT inválido: %d %s", status, body)
			}
		})
	}
}
