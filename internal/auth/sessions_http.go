package auth

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type sessionProfileDTO struct {
	MaxQuality       string `json:"max_quality"`
	AudioLanguage    string `json:"audio_language"`
	SubtitleLanguage string `json:"subtitle_language"`
}

type sessionDTO struct {
	ID         int64             `json:"id"`
	DeviceName string            `json:"device_name"`
	Client     string            `json:"client"`
	CreatedAt  string            `json:"created_at"`
	LastSeenAt string            `json:"last_seen_at"`
	ExpiresAt  string            `json:"expires_at"`
	Current    bool              `json:"current"`
	Profile    sessionProfileDTO `json:"profile"`
}

type sessionsResponse struct {
	Sessions []sessionDTO `json:"sessions"`
}

var profileLanguagePattern = regexp.MustCompile(`^[a-z]{2,3}$`)

// RegisterSessionRoutes monta la consulta y el cierre de sesiones propias y,
// para administradores, las de cualquier usuario.
func (s *Service) RegisterSessionRoutes(mux *http.ServeMux) {
	if s == nil || mux == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/auth/sessions", s.handleOwnSessions)
	mux.HandleFunc("GET /api/v1/auth/sessions/{id}/profile", s.handleOwnSessionProfileGet)
	mux.HandleFunc("PUT /api/v1/auth/sessions/{id}/profile", s.handleOwnSessionProfilePut)
	mux.HandleFunc("DELETE /api/v1/auth/sessions/{id}", s.handleOwnSessionDelete)
	mux.Handle("GET /api/v1/users/{id}/sessions",
		s.RequireAdmin(http.HandlerFunc(s.handleUserSessions)))
	mux.Handle("DELETE /api/v1/users/{id}/sessions/{sessionId}",
		s.RequireAdmin(http.HandlerFunc(s.handleUserSessionDelete)))
}

func (s *Service) handleOwnSessions(w http.ResponseWriter, r *http.Request) {
	user, ok := s.authenticateSessionRequest(w, r)
	if !ok {
		return
	}
	s.writeSessions(w, r, user.ID, user.SessionID)
}

func (s *Service) handleOwnSessionDelete(w http.ResponseWriter, r *http.Request) {
	user, ok := s.authenticateSessionRequest(w, r)
	if !ok {
		return
	}
	sessionID, ok := sessionIDFromPath(w, r, "id")
	if !ok {
		return
	}
	if err := s.DeleteSession(r.Context(), user.ID, sessionID); err != nil {
		writeSessionError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) handleOwnSessionProfileGet(w http.ResponseWriter, r *http.Request) {
	user, ok := s.authenticateSessionRequest(w, r)
	if !ok {
		return
	}
	sessionID, ok := profileSessionIDFromPath(w, r, user.SessionID)
	if !ok {
		return
	}
	profile, err := s.SessionProfileFor(r.Context(), user.ID, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "sesión no encontrada")
			return
		}
		log.Printf("auth: leer perfil de sesión %d: %v", sessionID, err)
		writeError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, sessionProfileToDTO(profile))
}

func (s *Service) handleOwnSessionProfilePut(w http.ResponseWriter, r *http.Request) {
	user, ok := s.authenticateSessionRequest(w, r)
	if !ok {
		return
	}
	sessionID, ok := profileSessionIDFromPath(w, r, user.SessionID)
	if !ok {
		return
	}
	var request *sessionProfileDTO
	if err := decodeJSON(w, r, &request); err != nil {
		if errors.Is(err, errBodyTooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "el cuerpo de la petición supera el límite de 1 MiB")
			return
		}
		writeError(w, http.StatusBadRequest, "petición no válida")
		return
	}
	if request == nil {
		writeError(w, http.StatusBadRequest, "petición no válida")
		return
	}
	profile, message := validateSessionProfile(*request)
	if message != "" {
		writeError(w, http.StatusBadRequest, message)
		return
	}
	if err := s.PutSessionProfile(r.Context(), user.ID, sessionID, profile); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "sesión no encontrada")
			return
		}
		log.Printf("auth: guardar perfil de sesión %d: %v", sessionID, err)
		writeError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, sessionProfileToDTO(profile))
}

func (s *Service) handleUserSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromPath(w, r)
	if !ok {
		return
	}
	exists, err := s.userExists(r.Context(), userID)
	if err != nil {
		log.Printf("auth: comprobar usuario para listar sesiones: %v", err)
		writeError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if !exists {
		writeError(w, http.StatusNotFound, "usuario no encontrado")
		return
	}
	s.writeSessions(w, r, userID, 0)
}

func (s *Service) handleUserSessionDelete(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromPath(w, r)
	if !ok {
		return
	}
	sessionID, ok := sessionIDFromPath(w, r, "sessionId")
	if !ok {
		return
	}
	if err := s.DeleteSession(r.Context(), userID, sessionID); err != nil {
		writeSessionError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) writeSessions(w http.ResponseWriter, r *http.Request, userID, currentID int64) {
	sessions, err := s.ListSessions(r.Context(), userID, currentID)
	if err != nil {
		log.Printf("auth: listar sesiones del usuario %d: %v", userID, err)
		writeError(w, http.StatusInternalServerError, "error interno")
		return
	}
	response := sessionsResponse{Sessions: make([]sessionDTO, 0, len(sessions))}
	for _, session := range sessions {
		response.Sessions = append(response.Sessions, sessionDTO{
			ID:         session.ID,
			DeviceName: session.DeviceName,
			Client:     session.Client,
			CreatedAt:  session.CreatedAt,
			LastSeenAt: session.LastSeenAt,
			ExpiresAt:  session.ExpiresAt,
			Current:    session.Current,
			Profile:    sessionProfileToDTO(session.Profile),
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Service) authenticateSessionRequest(w http.ResponseWriter, r *http.Request) (User, bool) {
	token := bearerToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "falta el token de sesión")
		return User{}, false
	}
	user, err := s.Authenticate(r.Context(), token)
	if err != nil {
		if errors.Is(err, ErrInvalidSession) || errors.Is(err, ErrSessionExpired) {
			writeError(w, http.StatusUnauthorized, "sesión no válida")
			return User{}, false
		}
		log.Printf("auth: validar sesión para gestionar sesiones: %v", err)
		writeError(w, http.StatusInternalServerError, "error interno")
		return User{}, false
	}
	return user, true
}

func sessionIDFromPath(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusNotFound, "sesión no encontrada")
		return 0, false
	}
	return id, true
}

func profileSessionIDFromPath(w http.ResponseWriter, r *http.Request, currentID int64) (int64, bool) {
	raw := strings.TrimSpace(r.PathValue("id"))
	if raw == "current" {
		return currentID, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusNotFound, "sesión no encontrada")
		return 0, false
	}
	return id, true
}

func validateSessionProfile(request sessionProfileDTO) (SessionProfile, string) {
	switch request.MaxQuality {
	case "", "480", "720", "1080":
	default:
		return SessionProfile{}, "max_quality no válida"
	}
	if request.AudioLanguage != "" && !profileLanguagePattern.MatchString(request.AudioLanguage) {
		return SessionProfile{}, "audio_language no válido"
	}
	if request.SubtitleLanguage != "" && request.SubtitleLanguage != "off" &&
		!profileLanguagePattern.MatchString(request.SubtitleLanguage) {
		return SessionProfile{}, "subtitle_language no válido"
	}
	return SessionProfile{
		MaxQuality:       request.MaxQuality,
		AudioLanguage:    request.AudioLanguage,
		SubtitleLanguage: request.SubtitleLanguage,
	}, ""
}

func sessionProfileToDTO(profile SessionProfile) sessionProfileDTO {
	return sessionProfileDTO{
		MaxQuality:       profile.MaxQuality,
		AudioLanguage:    profile.AudioLanguage,
		SubtitleLanguage: profile.SubtitleLanguage,
	}
}

func writeSessionError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrSessionNotFound) {
		writeError(w, http.StatusNotFound, "sesión no encontrada")
		return
	}
	log.Printf("auth: error al cerrar sesión: %v", err)
	writeError(w, http.StatusInternalServerError, "error interno")
}
