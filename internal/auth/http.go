package auth

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
)

const maxRequestBody = 8 << 10 // 8 KiB

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// LoginHandler atiende POST /api/v1/auth/login.
func (s *Service) LoginHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "petición no válida")
			return
		}
		if strings.TrimSpace(req.Username) == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "faltan usuario o contraseña")
			return
		}

		token, expiresAt, err := s.Login(r.Context(), req.Username, req.Password)
		switch {
		case errors.Is(err, ErrAccountLocked):
			writeError(w, http.StatusTooManyRequests, "demasiados intentos fallidos; inténtalo más tarde")
		case errors.Is(err, ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "usuario o contraseña incorrectos")
		case err != nil:
			log.Printf("auth: error interno en el login: %v", err)
			writeError(w, http.StatusInternalServerError, "error interno")
		default:
			writeJSON(w, http.StatusOK, loginResponse{Token: token, ExpiresAt: expiresAt})
		}
	})
}

// LogoutHandler atiende POST /api/v1/auth/logout. Espera el token de sesión en
// la cabecera Authorization: Bearer <token>.
func (s *Service) LogoutHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "falta el token de sesión")
			return
		}

		if err := s.Logout(r.Context(), token); err != nil {
			log.Printf("auth: error interno al cerrar sesión: %v", err)
			writeError(w, http.StatusInternalServerError, "error interno")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func bearerToken(r *http.Request) string {
	const prefix = "bearer "
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("auth: escribir respuesta: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
