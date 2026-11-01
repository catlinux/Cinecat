package library

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/catlinux/cinecat/internal/settings"
)

type homePreferenceRow struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Hidden bool   `json:"hidden"`
}

type homePreferencesResponse struct {
	Rows []homePreferenceRow `json:"rows"`
}

type homePreferencesRequest struct {
	Hidden *[]string `json:"hidden"`
}

func (a *API) portalRowDefinitions() []settings.PortalRow {
	if a.settingsStore != nil {
		return settings.PortalRows(a.settingsStore.String("ui.portal_rows"))
	}
	return settings.PortalRowsDefault()
}

func (a *API) hiddenPortalRows(ctx context.Context, userID int64) (map[string]bool, error) {
	rows, err := a.db.QueryContext(ctx,
		`SELECT row_id FROM user_portal_rows WHERE user_id = ? AND hidden = 1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hidden := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		hidden[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hidden, nil
}

func (a *API) handleHomePreferences(w http.ResponseWriter, r *http.Request) {
	user, ok := sessionUserFromRequest(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "falta el token de sesión")
		return
	}
	hidden, err := a.hiddenPortalRows(r.Context(), user.ID)
	if err != nil {
		log.Printf("library: leer preferencias del portal: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	rows := make([]homePreferenceRow, 0)
	for _, definition := range a.portalRowDefinitions() {
		rows = append(rows, homePreferenceRow{
			ID:     definition.ID,
			Title:  definition.Title,
			Hidden: hidden[definition.ID],
		})
	}
	writeJSON(w, http.StatusOK, homePreferencesResponse{Rows: rows})
}

func (a *API) handleHomePreferencesPut(w http.ResponseWriter, r *http.Request) {
	user, ok := sessionUserFromRequest(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "falta el token de sesión")
		return
	}
	var request homePreferencesRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSONBodyError(w, err)
		return
	}
	if request.Hidden == nil {
		writeJSONError(w, http.StatusBadRequest, "falta la lista hidden")
		return
	}
	known := make(map[string]bool)
	for _, definition := range a.portalRowDefinitions() {
		known[definition.ID] = true
	}
	hidden := make([]string, 0, len(*request.Hidden))
	for _, id := range *request.Hidden {
		id = strings.TrimSpace(id)
		if !known[id] {
			writeJSONError(w, http.StatusBadRequest, "fila del portal desconocida: "+id)
			return
		}
		hidden = append(hidden, id)
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if _, err := tx.ExecContext(r.Context(),
		`DELETE FROM user_portal_rows WHERE user_id = ?`, user.ID); err != nil {
		log.Printf("library: borrar preferencias del portal: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	for _, id := range hidden {
		if _, err := tx.ExecContext(r.Context(), `
			INSERT OR IGNORE INTO user_portal_rows (user_id, row_id, hidden)
			VALUES (?, ?, 1)`, user.ID, id); err != nil {
			log.Printf("library: guardar preferencias del portal: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("library: confirmar preferencias del portal: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func filterHomeRows(home *homeResponse, hidden map[string]bool) {
	if hidden["recent_movies"] {
		home.RecentMovies = []mediaItemDTO{}
	}
	if hidden["watched_movies"] {
		home.RecentlyWatched = []mediaItemDTO{}
	}
	if hidden["recent_series"] {
		home.RecentSeries = []seriesListDTO{}
	}
	if hidden["next_up"] {
		home.NextUp = []mediaItemDTO{}
	}
	if hidden["continue_movies"] || hidden["continue_series"] {
		filtered := make([]mediaItemDTO, 0, len(home.Continue))
		for _, item := range home.Continue {
			switch item.Kind {
			case KindMovie:
				if !hidden["continue_movies"] {
					filtered = append(filtered, item)
				}
			case KindEpisode:
				if !hidden["continue_series"] {
					filtered = append(filtered, item)
				}
			default:
				filtered = append(filtered, item)
			}
		}
		home.Continue = filtered
	}
	if hidden["watchlist_movies"] || hidden["watchlist_series"] {
		filtered := make([]homeListEntry, 0, len(home.Watchlist))
		for _, entry := range home.Watchlist {
			switch entry.Kind {
			case "item":
				if !hidden["watchlist_movies"] {
					filtered = append(filtered, entry)
				}
			case "series":
				if !hidden["watchlist_series"] {
					filtered = append(filtered, entry)
				}
			default:
				filtered = append(filtered, entry)
			}
		}
		home.Watchlist = filtered
	}
	if hidden["favorites"] {
		home.Favorites = []homeListEntry{}
	}
}
