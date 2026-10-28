package backup

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/catlinux/cinecat/internal/auth"
)

// Authenticator valida el token de una petición.
type Authenticator func(ctx context.Context, token string) (auth.User, error)

// API expone la gestión de copias de seguridad a la administración.
type API struct {
	authenticate  Authenticator
	manager       *Manager
	enabled       func() bool
	retentionDays func() int
}

// NewAPI crea los handlers de copias. enabled y retentionDays permiten leer
// los ajustes en cada petición sin guardar una copia de sus valores.
func NewAPI(
	authenticate Authenticator,
	manager *Manager,
	enabled func() bool,
	retentionDays func() int,
) *API {
	return &API{
		authenticate:  authenticate,
		manager:       manager,
		enabled:       enabled,
		retentionDays: retentionDays,
	}
}

// Register monta los endpoints de copias en el mux indicado.
func (a *API) Register(mux *http.ServeMux) {
	if a == nil || a.manager == nil {
		return
	}
	mux.Handle("GET /api/v1/system/backups", a.requireAdmin(a.handleList))
	mux.Handle("POST /api/v1/system/backups", a.requireAdmin(a.handleCreate))
}

type listResponse struct {
	Enabled       bool   `json:"enabled"`
	RetentionDays int    `json:"retention_days"`
	Backups       []Copy `json:"backups"`
	LastError     string `json:"last_error"`
}

func (a *API) handleList(w http.ResponseWriter, r *http.Request) {
	backups, err := a.manager.List()
	if err != nil {
		log.Printf("backup: listar copias: %v", err)
		writeError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, listResponse{
		Enabled:       a.isEnabled(),
		RetentionDays: a.retention(),
		Backups:       backups,
		LastError:     a.manager.LastError(),
	})
}

func (a *API) handleCreate(w http.ResponseWriter, r *http.Request) {
	copy, err := a.manager.Create(r.Context())
	if err != nil {
		if errors.Is(err, ErrBusy) {
			writeError(w, http.StatusConflict, "ya hay una copia de seguridad en curso")
			return
		}
		writeError(w, http.StatusInternalServerError, "no se pudo crear la copia de seguridad")
		return
	}
	writeJSON(w, http.StatusCreated, copy)
}

func (a *API) isEnabled() bool {
	if a.enabled != nil {
		return a.enabled()
	}
	return a.manager.Enabled()
}

func (a *API) retention() int {
	if a.retentionDays != nil {
		days := a.retentionDays()
		if days > 0 {
			return days
		}
	}
	return a.manager.RetentionDays()
}

func (a *API) requireAdmin(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "falta el token de sesión")
			return
		}
		if a.authenticate == nil {
			writeError(w, http.StatusUnauthorized, "sesión no válida")
			return
		}
		user, err := a.authenticate(r.Context(), token)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidSession) || errors.Is(err, auth.ErrSessionExpired) {
				writeError(w, http.StatusUnauthorized, "sesión no válida")
				return
			}
			log.Printf("backup: validar sesión: %v", err)
			writeError(w, http.StatusInternalServerError, "error interno")
			return
		}
		if user.Role != "admin" {
			writeError(w, http.StatusForbidden, "se requiere rol de administrador")
			return
		}
		next(w, r)
	})
}

func bearerToken(r *http.Request) string {
	const prefix = "bearer "
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(value) > len(prefix) && strings.EqualFold(value[:len(prefix)], prefix) {
		return strings.TrimSpace(value[len(prefix):])
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("backup: escribir respuesta: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
