package update

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/catlinux/cinecat/internal/auth"
)

type Authenticator func(ctx context.Context, token string) (auth.User, error)

type API struct {
	authenticate Authenticator
	checker      *Checker
	requester    *Requester
	checkEnabled func() bool
}

func NewAPI(
	authenticate Authenticator,
	checker *Checker,
	requester *Requester,
	checkEnabled func() bool,
) *API {
	return &API{
		authenticate: authenticate,
		checker:      checker,
		requester:    requester,
		checkEnabled: checkEnabled,
	}
}

func (a *API) Register(mux *http.ServeMux) {
	if a == nil {
		return
	}
	mux.Handle("GET /api/v1/system/update", a.requireAdmin(a.handleStatus))
	mux.Handle("POST /api/v1/system/update/check", a.requireAdmin(a.handleCheck))
	mux.Handle("POST /api/v1/system/update/apply", a.requireAdmin(a.handleApply))
}

type updateResponse struct {
	Current        string `json:"current"`
	Latest         string `json:"latest"`
	Available      bool   `json:"available"`
	CheckedAt      string `json:"checked_at"`
	Notes          string `json:"notes"`
	ApplySupported bool   `json:"apply_supported"`
	CheckEnabled   bool   `json:"check_enabled"`
	Status         Status `json:"status"`
}

func (a *API) response(result Result) updateResponse {
	checkedAt := ""
	if !result.CheckedAt.IsZero() {
		checkedAt = result.CheckedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	status := Status{}
	if a.requester != nil {
		status = a.requester.Status()
	}
	return updateResponse{
		Current:        result.Current,
		Latest:         result.Latest,
		Available:      result.Available,
		CheckedAt:      checkedAt,
		Notes:          result.Notes,
		ApplySupported: a.requester != nil && a.requester.Supported(),
		CheckEnabled:   a.enabled(),
		Status:         status,
	}
}

func (a *API) enabled() bool {
	return a.checkEnabled != nil && a.checkEnabled()
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
			log.Printf("update: validar sesión: %v", err)
			writeError(w, http.StatusInternalServerError, "error interno")
			return
		}
		if user.Role != "admin" {
			writeError(w, http.StatusForbidden, "se requiere rol de administrador")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey{}, user)))
	})
}

type userKey struct{}

func userFromRequest(r *http.Request) (auth.User, bool) {
	user, ok := r.Context().Value(userKey{}).(auth.User)
	return user, ok
}

func (a *API) handleStatus(w http.ResponseWriter, r *http.Request) {
	if a.checker == nil {
		writeError(w, http.StatusBadGateway, "no se pudo comprobar la actualización")
		return
	}
	result, _ := a.checker.Cached()
	if a.enabled() {
		checked, err := a.checker.Check(r.Context(), false)
		if err != nil {
			// Sin conexión se muestra lo último que se supo, no un error.
			log.Printf("update: comprobar: %v", err)
		} else {
			result = checked
		}
	}
	writeJSON(w, http.StatusOK, a.response(result))
}

func (a *API) handleCheck(w http.ResponseWriter, r *http.Request) {
	if a.checker == nil {
		writeError(w, http.StatusBadGateway, "no se pudo comprobar la actualización")
		return
	}
	result, err := a.checker.Check(r.Context(), true)
	if err != nil {
		log.Printf("update: comprobar: %v", err)
		writeError(w, http.StatusBadGateway, "no se pudo comprobar la actualización")
		return
	}
	writeJSON(w, http.StatusOK, a.response(result))
}

const maxRequestBody = 1 << 20 // 1 MiB

var errBodyTooLarge = errors.New("update: cuerpo de petición demasiado grande")

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	if r.ContentLength > maxRequestBody {
		return errBodyTooLarge
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(dst)
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return errBodyTooLarge
	}
	return err
}

func (a *API) handleApply(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Tag string `json:"tag"`
	}
	if err := decodeJSONBody(w, r, &request); err != nil {
		if errors.Is(err, errBodyTooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "el cuerpo de la petición supera el límite de 1 MiB")
			return
		}
		writeError(w, http.StatusBadRequest, "petición no válida")
		return
	}
	if _, ok := ParseTag(strings.TrimSpace(request.Tag)); !ok {
		writeError(w, http.StatusBadRequest, "etiqueta no válida")
		return
	}
	if a.checker == nil {
		writeError(w, http.StatusBadRequest, "no hay una versión nueva conocida")
		return
	}
	known, ok := a.checker.Cached()
	if !ok || known.Latest == "" || known.Latest != request.Tag || !known.Available {
		writeError(w, http.StatusBadRequest, "la etiqueta no es la última versión conocida")
		return
	}
	if a.requester == nil || !a.requester.Supported() {
		writeError(w, http.StatusServiceUnavailable, "La actualización automática no está disponible en esta instalación")
		return
	}
	user, _ := userFromRequest(r)
	if err := a.requester.Request(request.Tag, user.Username); err != nil {
		switch {
		case errors.Is(err, ErrConflict):
			writeError(w, http.StatusConflict, "ya hay una actualización en curso")
		case errors.Is(err, ErrUnsupported):
			writeError(w, http.StatusServiceUnavailable, "La actualización automática no está disponible en esta instalación")
		case errors.Is(err, ErrInvalidTag):
			writeError(w, http.StatusBadRequest, "etiqueta no válida")
		default:
			log.Printf("update: guardar petición: %v", err)
			writeError(w, http.StatusInternalServerError, "error interno")
		}
		return
	}
	writeJSON(w, http.StatusAccepted, a.response(known))
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
		log.Printf("update: escribir respuesta: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
