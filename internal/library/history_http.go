package library

import (
	"log"
	"net/http"
	"strconv"
)

type historyEntry struct {
	ItemID          int64   `json:"item_id"`
	Title           string  `json:"title"`
	Kind            string  `json:"kind"`
	SeriesTitle     *string `json:"series_title"`
	Season          *int    `json:"season"`
	Episode         *int    `json:"episode"`
	PosterURL       *string `json:"poster_url"`
	PositionSeconds float64 `json:"position_seconds"`
	DurationSeconds float64 `json:"duration_seconds"`
	Watched         bool    `json:"watched"`
	UpdatedAt       string  `json:"updated_at"`
}

type historyResponse struct {
	Entries []historyEntry `json:"entries"`
}

func (a *API) handleHistory(w http.ResponseWriter, r *http.Request) {
	limit, err := activityLimit(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID := sessionUserID(r)
	entries, err := a.queryActivity(r, limit, &userID, accessFromRequest(r))
	if err != nil {
		log.Printf("library: listar historial del usuario %d: %v", userID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	response := historyResponse{Entries: make([]historyEntry, 0, len(entries))}
	for _, entry := range entries {
		response.Entries = append(response.Entries, historyEntry{
			ItemID:          entry.ItemID,
			Title:           entry.Title,
			Kind:            entry.Kind,
			SeriesTitle:     entry.SeriesTitle,
			Season:          entry.Season,
			Episode:         entry.Episode,
			PosterURL:       entry.PosterURL,
			PositionSeconds: entry.PositionSeconds,
			DurationSeconds: entry.DurationSeconds,
			Watched:         entry.Watched,
			UpdatedAt:       entry.UpdatedAt,
		})
	}
	writeJSON(w, http.StatusOK, response)
}

func (a *API) handleHistoryDelete(w http.ResponseWriter, r *http.Request) {
	user, ok := sessionUserFromRequest(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "falta el token de sesión")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("itemId"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	if _, ok := a.requireVisibleItem(w, r, id); !ok {
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()
	for _, query := range []string{
		`DELETE FROM playback_progress WHERE user_id = ? AND media_item_id = ?`,
		`DELETE FROM playback_state WHERE user_id = ? AND media_item_id = ?`,
		`DELETE FROM watch_state WHERE user_id = ? AND media_item_id = ?`,
	} {
		if _, err := tx.ExecContext(r.Context(), query, user.ID, id); err != nil {
			log.Printf("library: borrar historial del ítem %d: %v", id, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("library: confirmar borrado del historial del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
