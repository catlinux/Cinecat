package library

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"math"
	"net/http"
)

type playbackProgress struct {
	PositionSeconds float64 `json:"position_seconds"`
	DurationSeconds float64 `json:"duration_seconds"`
	UpdatedAt       string  `json:"updated_at"`
}

type progressRequest struct {
	PositionSeconds *float64 `json:"position_seconds"`
	DurationSeconds *float64 `json:"duration_seconds"`
}

func (a *API) handleItemProgressPut(w http.ResponseWriter, r *http.Request) {
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

	var req progressRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "petición no válida")
		return
	}
	if req.PositionSeconds == nil || req.DurationSeconds == nil ||
		!validSeconds(*req.PositionSeconds) || !validSeconds(*req.DurationSeconds) {
		writeJSONError(w, http.StatusBadRequest, "position_seconds y duration_seconds deben ser números finitos mayores o iguales que 0")
		return
	}

	var exists int
	err := a.db.QueryRowContext(r.Context(),
		`SELECT 1 FROM media_items WHERE id = ?`, id).Scan(&exists)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	case err != nil:
		log.Printf("library: comprobar ítem %d para el progreso: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	position := *req.PositionSeconds
	duration := *req.DurationSeconds
	if duration > 0 && position >= 0.9*duration {
		if err := a.markWatchedAndClearProgress(r.Context(), user.ID, id); err != nil {
			log.Printf("library: marcar visto al guardar el progreso del ítem %d: %v", id, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if position < 10 {
		if _, err := a.db.ExecContext(r.Context(),
			`DELETE FROM playback_progress WHERE user_id = ? AND media_item_id = ?`,
			user.ID, id); err != nil {
			log.Printf("library: borrar el progreso inicial del ítem %d: %v", id, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if _, err := a.db.ExecContext(r.Context(), `
		INSERT INTO playback_progress (
			user_id, media_item_id, position_seconds, duration_seconds, updated_at
		) VALUES (?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		ON CONFLICT (user_id, media_item_id) DO UPDATE SET
			position_seconds = excluded.position_seconds,
			duration_seconds = excluded.duration_seconds,
			updated_at = excluded.updated_at`,
		user.ID, id, position, duration); err != nil {
		log.Printf("library: guardar el progreso del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleItemProgressDelete(w http.ResponseWriter, r *http.Request) {
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

	if _, err := a.db.ExecContext(r.Context(),
		`DELETE FROM playback_progress WHERE user_id = ? AND media_item_id = ?`,
		user.ID, id); err != nil {
		log.Printf("library: borrar el progreso del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) markWatchedAndClearProgress(ctx context.Context, userID, itemID int64) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO watch_state (user_id, media_item_id) VALUES (?, ?)`,
		userID, itemID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM playback_progress WHERE user_id = ? AND media_item_id = ?`,
		userID, itemID); err != nil {
		return err
	}
	return tx.Commit()
}

func validSeconds(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}
