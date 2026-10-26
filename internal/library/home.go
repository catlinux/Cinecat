package library

import (
	"context"
	"database/sql"
	"log"
	"net/http"
)

type homeCounts struct {
	Movies   int64 `json:"movies"`
	Episodes int64 `json:"episodes"`
}

type homeResponse struct {
	Continue     []mediaItemDTO `json:"continue"`
	NextUp       []mediaItemDTO `json:"next_up"`
	RecentMovies []mediaItemDTO `json:"recent_movies"`
	Counts       homeCounts     `json:"counts"`
}

func (a *API) handleHome(w http.ResponseWriter, r *http.Request) {
	user, ok := sessionUserFromRequest(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "falta el token de sesión")
		return
	}

	continueItems, err := queryHomeItems(r.Context(), a.db, `
		SELECT `+mediaItemColumns+watchedColumn+progressColumn+`
		  FROM media_items
		  JOIN playback_progress pp
		    ON pp.media_item_id = media_items.id AND pp.user_id = ?
		  LEFT JOIN watch_state ws
		    ON ws.media_item_id = media_items.id AND ws.user_id = ?
		 WHERE media_items.kind IN ('movie', 'episode')
		 ORDER BY pp.updated_at DESC, media_items.id DESC
		 LIMIT 20`, user.ID, user.ID)
	if err != nil {
		log.Printf("library: listar continuar viendo: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	recentMovies, err := queryHomeItems(r.Context(), a.db, `
		SELECT `+mediaItemColumns+watchedColumn+progressColumn+`
		  FROM media_items
		  LEFT JOIN watch_state ws
		    ON ws.media_item_id = media_items.id AND ws.user_id = ?
		  LEFT JOIN playback_progress pp
		    ON pp.media_item_id = media_items.id AND pp.user_id = ?
		 WHERE media_items.kind = 'movie'
		 ORDER BY media_items.id DESC
		 LIMIT 20`, user.ID, user.ID)
	if err != nil {
		log.Printf("library: listar películas recientes: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	var counts homeCounts
	if err := a.db.QueryRowContext(r.Context(), `
		SELECT COALESCE(SUM(CASE WHEN kind = 'movie' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN kind = 'episode' THEN 1 ELSE 0 END), 0)
		  FROM media_items`).Scan(&counts.Movies, &counts.Episodes); err != nil {
		log.Printf("library: contar ítems del portal: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	writeJSON(w, http.StatusOK, homeResponse{
		Continue: continueItems,
		// Hueco para series: next_up se rellenará cuando existan como entidad.
		NextUp:       make([]mediaItemDTO, 0),
		RecentMovies: recentMovies,
		Counts:       counts,
	})
}

func queryHomeItems(ctx context.Context, db *sql.DB, query string, args ...any) ([]mediaItemDTO, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]mediaItemDTO, 0)
	for rows.Next() {
		item, err := scanItem(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
