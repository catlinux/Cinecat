package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
)

type userDTO struct {
	ID             int64   `json:"id"`
	Username       string  `json:"username"`
	Role           string  `json:"role"`
	Disabled       bool    `json:"disabled"`
	AllLibraries   bool    `json:"all_libraries"`
	LibraryIDs     []int64 `json:"library_ids"`
	CreatedAt      string  `json:"created_at"`
	ActiveSessions int     `json:"active_sessions"`
}

type usersResponse struct {
	Users []userDTO `json:"users"`
}

type createUserRequest struct {
	Username     string  `json:"username"`
	Password     string  `json:"password"`
	Role         string  `json:"role"`
	AllLibraries bool    `json:"all_libraries"`
	LibraryIDs   []int64 `json:"library_ids"`
}

type updateUserRequest struct {
	Role         *string `json:"role"`
	Disabled     *bool   `json:"disabled"`
	AllLibraries *bool   `json:"all_libraries"`
}

type userLibrariesRequest struct {
	LibraryIDs []int64 `json:"library_ids"`
}

type resetUserPasswordRequest struct {
	Password string `json:"password"`
}

type sessionUserKey struct{}

func withSessionUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, sessionUserKey{}, user)
}

func sessionUser(r *http.Request) (User, bool) {
	user, ok := r.Context().Value(sessionUserKey{}).(User)
	return user, ok
}

// RegisterUserRoutes monta la gestión de usuarios, reservada a administradores.
func (s *Service) RegisterUserRoutes(mux *http.ServeMux) {
	if s == nil || mux == nil {
		return
	}
	mux.Handle("GET /api/v1/users", s.RequireAdmin(http.HandlerFunc(s.handleUsersList)))
	mux.Handle("POST /api/v1/users", s.RequireAdmin(http.HandlerFunc(s.handleUsersCreate)))
	mux.Handle("PATCH /api/v1/users/{id}", s.RequireAdmin(http.HandlerFunc(s.handleUsersUpdate)))
	mux.Handle("PUT /api/v1/users/{id}/libraries", s.RequireAdmin(http.HandlerFunc(s.handleUsersLibraries)))
	mux.Handle("POST /api/v1/users/{id}/password", s.RequireAdmin(http.HandlerFunc(s.handleUsersPassword)))
	mux.Handle("DELETE /api/v1/users/{id}", s.RequireAdmin(http.HandlerFunc(s.handleUsersDelete)))
}

// RequireAdmin valida la sesión Bearer y deja pasar solo a administradores.
func (s *Service) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "falta el token de sesión")
			return
		}
		user, err := s.Authenticate(r.Context(), token)
		if err != nil {
			if errors.Is(err, ErrInvalidSession) || errors.Is(err, ErrSessionExpired) {
				writeError(w, http.StatusUnauthorized, "sesión no válida")
				return
			}
			log.Printf("auth: error al validar la sesión: %v", err)
			writeError(w, http.StatusInternalServerError, "error interno")
			return
		}
		if user.Role != RoleAdmin {
			writeError(w, http.StatusForbidden, "se requiere rol de administrador")
			return
		}
		next.ServeHTTP(w, r.WithContext(withSessionUser(r.Context(), user)))
	})
}

func (s *Service) handleUsersList(w http.ResponseWriter, r *http.Request) {
	users, err := s.ListUsers(r.Context())
	if err != nil {
		writeUserServiceError(w, err)
		return
	}
	response := usersResponse{Users: make([]userDTO, 0, len(users))}
	for _, user := range users {
		response.Users = append(response.Users, userResponse(user))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Service) handleUsersCreate(w http.ResponseWriter, r *http.Request) {
	actor, _ := sessionUser(r)
	var req createUserRequest
	if !decodeUserRequest(w, r, &req) {
		return
	}
	user, err := s.CreateUser(r.Context(), req.Username, req.Password, req.Role,
		req.AllLibraries, req.LibraryIDs)
	if err != nil {
		writeUserServiceError(w, err)
		return
	}
	log.Printf("auth: el administrador %q (id %d) creó al usuario %q (id %d)",
		actor.Username, actor.ID, user.Username, user.ID)
	writeJSON(w, http.StatusCreated, userResponse(user))
}

func (s *Service) handleUsersUpdate(w http.ResponseWriter, r *http.Request) {
	actor, _ := sessionUser(r)
	id, ok := userIDFromPath(w, r)
	if !ok {
		return
	}
	var req updateUserRequest
	if !decodeUserRequest(w, r, &req) {
		return
	}
	if id == actor.ID {
		if req.Disabled != nil && *req.Disabled {
			writeError(w, http.StatusForbidden, "no puedes desactivar tu propia cuenta")
			return
		}
		if req.Role != nil && *req.Role == RoleUser {
			writeError(w, http.StatusForbidden, "no puedes quitarte el rol de administrador")
			return
		}
	}
	user, err := s.UpdateUser(r.Context(), id, UserChanges{
		Role:         req.Role,
		Disabled:     req.Disabled,
		AllLibraries: req.AllLibraries,
	})
	if err != nil {
		writeUserServiceError(w, err)
		return
	}
	log.Printf("auth: el administrador %q (id %d) actualizó al usuario %q (id %d)",
		actor.Username, actor.ID, user.Username, user.ID)
	writeJSON(w, http.StatusOK, userResponse(user))
}

func (s *Service) handleUsersLibraries(w http.ResponseWriter, r *http.Request) {
	actor, _ := sessionUser(r)
	id, ok := userIDFromPath(w, r)
	if !ok {
		return
	}
	var req userLibrariesRequest
	if !decodeUserRequest(w, r, &req) {
		return
	}
	user, err := s.SetUserLibraries(r.Context(), id, req.LibraryIDs)
	if err != nil {
		writeUserServiceError(w, err)
		return
	}
	log.Printf("auth: el administrador %q (id %d) cambió las bibliotecas del usuario %q (id %d)",
		actor.Username, actor.ID, user.Username, user.ID)
	writeJSON(w, http.StatusOK, userResponse(user))
}

func (s *Service) handleUsersPassword(w http.ResponseWriter, r *http.Request) {
	actor, _ := sessionUser(r)
	id, ok := userIDFromPath(w, r)
	if !ok {
		return
	}
	var req resetUserPasswordRequest
	if !decodeUserRequest(w, r, &req) {
		return
	}
	if err := s.ResetPassword(r.Context(), id, req.Password); err != nil {
		writeUserServiceError(w, err)
		return
	}
	log.Printf("auth: el administrador %q (id %d) restableció la contraseña del usuario con id %d",
		actor.Username, actor.ID, id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) handleUsersDelete(w http.ResponseWriter, r *http.Request) {
	actor, _ := sessionUser(r)
	id, ok := userIDFromPath(w, r)
	if !ok {
		return
	}
	if id == actor.ID {
		writeError(w, http.StatusForbidden, "no puedes borrar tu propia cuenta")
		return
	}
	if err := s.DeleteUser(r.Context(), id); err != nil {
		writeUserServiceError(w, err)
		return
	}
	log.Printf("auth: el administrador %q (id %d) borró el usuario con id %d",
		actor.Username, actor.ID, id)
	w.WriteHeader(http.StatusNoContent)
}

func decodeUserRequest(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := decodeJSON(w, r, dst); err != nil {
		if errors.Is(err, errBodyTooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "el cuerpo de la petición supera el límite de 1 MiB")
			return false
		}
		writeError(w, http.StatusBadRequest, "petición no válida")
		return false
	}
	return true
}

func userIDFromPath(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusNotFound, "usuario no encontrado")
		return 0, false
	}
	return id, true
}

func userResponse(user User) userDTO {
	libraryIDs := user.LibraryIDs
	if libraryIDs == nil {
		libraryIDs = make([]int64, 0)
	}
	return userDTO{
		ID:             user.ID,
		Username:       user.Username,
		Role:           user.Role,
		Disabled:       user.Disabled,
		AllLibraries:   user.AllLibraries,
		LibraryIDs:     libraryIDs,
		CreatedAt:      user.CreatedAt,
		ActiveSessions: user.ActiveSessions,
	}
}

func writeUserServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrPasswordTooShort):
		writeError(w, http.StatusBadRequest, "la contraseña debe tener al menos 8 caracteres")
	case errors.Is(err, ErrInvalidUsername), errors.Is(err, ErrInvalidRole),
		errors.Is(err, ErrLibraryNotFound):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrUsernameTaken), errors.Is(err, ErrLastAdmin):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrUserNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		log.Printf("auth: error en la gestión de usuarios: %v", err)
		writeError(w, http.StatusInternalServerError, "error interno")
	}
}
