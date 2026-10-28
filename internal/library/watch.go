package library

import (
	"context"
	"database/sql"
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

	if _, ok := a.requireVisibleItem(w, r, id); !ok {
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("library: iniciar cambio de visionado del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if watched {
		_, err = tx.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO watch_state (user_id, media_item_id) VALUES (?, ?)`,
			user.ID, id)
		if err == nil {
			_, err = tx.ExecContext(r.Context(),
				`DELETE FROM item_lists WHERE user_id = ? AND media_item_id = ? AND list = 'watchlist'`,
				user.ID, id)
		}
	} else {
		_, err = tx.ExecContext(r.Context(),
			`DELETE FROM watch_state WHERE user_id = ? AND media_item_id = ?`,
			user.ID, id)
	}
	if err != nil {
		log.Printf("library: cambiar visionado del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if _, err := tx.ExecContext(r.Context(),
		`DELETE FROM playback_progress WHERE user_id = ? AND media_item_id = ?`,
		user.ID, id); err != nil {
		log.Printf("library: borrar progreso al cambiar visionado del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if watched {
		var seriesID sql.NullInt64
		if err := tx.QueryRowContext(r.Context(),
			`SELECT series_id FROM media_items WHERE id = ?`, id).Scan(&seriesID); err != nil {
			log.Printf("library: leer serie del ítem %d: %v", id, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		if seriesID.Valid {
			if err := clearSeriesWatchlistIfWatched(r.Context(), tx, user.ID, seriesID.Int64); err != nil {
				log.Printf("library: limpiar próxima a ver de la serie %d: %v", seriesID.Int64, err)
				writeJSONError(w, http.StatusInternalServerError, "error interno")
				return
			}
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("library: confirmar visionado del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
