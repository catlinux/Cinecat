package library

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
)

type seriesAutoplayResponse struct {
	Enabled bool `json:"enabled"`
}

type seriesAutoplayRequest struct {
	Enabled *bool `json:"enabled"`
}

// seriesAutoplayEnabled devuelve true si no hay fila guardada: la
// reproducción automática está activada por defecto.
func seriesAutoplayEnabled(ctx context.Context, db *sql.DB, userID, seriesID int64) (bool, error) {
	var enabled int
	err := db.QueryRowContext(ctx, `
		SELECT enabled FROM series_autoplay
		 WHERE user_id = ? AND series_id = ?`, userID, seriesID).Scan(&enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return enabled != 0, nil
}

func setSeriesAutoplay(ctx context.Context, db *sql.DB, userID, seriesID int64, enabled bool) error {
	value := 0
	if enabled {
		value = 1
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO series_autoplay (user_id, series_id, enabled)
		VALUES (?, ?, ?)
		ON CONFLICT (user_id, series_id) DO UPDATE SET enabled = excluded.enabled`,
		userID, seriesID, value)
	return err
}

func (a *API) autoplayNextForPlay(ctx context.Context, userID, itemID int64) (bool, error) {
	var kind string
	var seriesID sql.NullInt64
	if err := a.db.QueryRowContext(ctx, `
		SELECT kind, series_id FROM media_items WHERE id = ?`, itemID).
		Scan(&kind, &seriesID); err != nil {
		return true, err
	}
	if kind != KindEpisode || !seriesID.Valid {
		return true, nil
	}
	return seriesAutoplayEnabled(ctx, a.db, userID, seriesID.Int64)
}

func (a *API) handleSeriesAutoplayGet(w http.ResponseWriter, r *http.Request) {
	userID := sessionUserID(r)
	if userID == 0 {
		writeJSONError(w, http.StatusUnauthorized, "falta el token de sesión")
		return
	}
	record, ok := a.loadSeriesRecord(w, r)
	if !ok {
		return
	}
	enabled, err := seriesAutoplayEnabled(r.Context(), a.db, userID, record.ID)
	if err != nil {
		log.Printf("library: leer reproducción automática de la serie %d: %v", record.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, seriesAutoplayResponse{Enabled: enabled})
}

func (a *API) handleSeriesAutoplayPut(w http.ResponseWriter, r *http.Request) {
	userID := sessionUserID(r)
	if userID == 0 {
		writeJSONError(w, http.StatusUnauthorized, "falta el token de sesión")
		return
	}
	record, ok := a.loadSeriesRecord(w, r)
	if !ok {
		return
	}
	var request seriesAutoplayRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSONBodyError(w, err)
		return
	}
	if request.Enabled == nil {
		writeJSONError(w, http.StatusBadRequest, "enabled no válido")
		return
	}
	if err := setSeriesAutoplay(r.Context(), a.db, userID, record.ID, *request.Enabled); err != nil {
		log.Printf("library: guardar reproducción automática de la serie %d: %v", record.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, seriesAutoplayResponse{Enabled: *request.Enabled})
}
