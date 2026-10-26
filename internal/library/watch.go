package library

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/catlinux/cinecat/internal/auth"
)

type sessionUserKey struct{}

func withSessionUser(ctx context.Context, user auth.User) context.Context {
	return context.WithValue(ctx, sessionUserKey{}, user)
}

func sessionUserFromRequest(r *http.Request) (auth.User, bool) {
	user, ok := r.Context().Value(sessionUserKey{}).(auth.User)
	return user, ok
}

func sessionUserID(r *http.Request) int64 {
	user, ok := sessionUserFromRequest(r)
	if !ok {
		return 0
	}
	return user.ID
}

func (a *API) handleItemWatched(w http.ResponseWriter, r *http.Request) {
	a.setWatched(w, r, true)
}

func (a *API) handleItemUnwatched(w http.ResponseWriter, r *http.Request) {
	a.setWatched(w, r, false)
}

func (a *API) setWatched(w http.ResponseWriter, r *http.Request, watched bool) {
	user, ok := sessionUserFromRequest(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "falta el token de sesión")
		return
	}
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}

	var exists int
	err := a.db.QueryRowContext(r.Context(), `SELECT 1 FROM media_items WHERE id = ?`, id).Scan(&exists)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	case err != nil:
		log.Printf("library: comprobar ítem %d para visionado: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	if watched {
		_, err = a.db.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO watch_state (user_id, media_item_id) VALUES (?, ?)`,
			user.ID, id)
	} else {
		_, err = a.db.ExecContext(r.Context(),
			`DELETE FROM watch_state WHERE user_id = ? AND media_item_id = ?`,
			user.ID, id)
	}
	if err != nil {
		log.Printf("library: cambiar visionado del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
