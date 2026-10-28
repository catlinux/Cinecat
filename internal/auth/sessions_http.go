package auth

import (
	"errors"
	"log"
	"net/http"
	"strconv"
)

type sessionDTO struct {
	ID         int64  `json:"id"`
	DeviceName string `json:"device_name"`
	Client     string `json:"client"`
	CreatedAt  string `json:"created_at"`
	LastSeenAt string `json:"last_seen_at"`
	ExpiresAt  string `json:"expires_at"`
	Current    bool   `json:"current"`
}

type sessionsResponse struct {
	Sessions []sessionDTO `json:"sessions"`
}

// RegisterSessionRoutes monta la consulta y el cierre de sesiones propias y,
// para administradores, las de cualquier usuario.
func (s *Service) RegisterSessionRoutes(mux *http.ServeMux) {
	if s == nil || mux == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/auth/sessions", s.handleOwnSessions)
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

func writeSessionError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrSessionNotFound) {
		writeError(w, http.StatusNotFound, "sesión no encontrada")
		return
	}
	log.Printf("auth: error al cerrar sesión: %v", err)
	writeError(w, http.StatusInternalServerError, "error interno")
}
