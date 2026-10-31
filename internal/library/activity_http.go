package library

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const (
	defaultActivityLimit = 50
	maxActivityLimit     = 200
)

type activityEntry struct {
	UserID          int64   `json:"user_id"`
	Username        string  `json:"username"`
	ItemID          int64   `json:"item_id"`
	Title           string  `json:"title"`
	Kind            string  `json:"kind"`
	SeriesTitle     *string `json:"series_title"`
	Season          *int    `json:"season"`
	Episode         *int    `json:"episode"`
	PositionSeconds float64 `json:"position_seconds"`
	DurationSeconds float64 `json:"duration_seconds"`
	Watched         bool    `json:"watched"`
	UpdatedAt       string  `json:"updated_at"`
}

type activityResponse struct {
	Entries []activityEntry `json:"entries"`
}

func (a *API) handleActivity(w http.ResponseWriter, r *http.Request) {
	limit, err := activityLimit(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID, err := activityUserFilter(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	entries, err := a.queryActivity(r, limit, userID)
	if err != nil {
		log.Printf("library: listar actividad: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if entries == nil {
		entries = []activityEntry{}
	}
	writeJSON(w, http.StatusOK, activityResponse{Entries: entries})
}

func activityLimit(r *http.Request) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return defaultActivityLimit, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, errors.New("limit debe ser un entero positivo")
	}
	if value > maxActivityLimit {
		return maxActivityLimit, nil
	}
	return value, nil
}

func activityUserFilter(r *http.Request) (*int64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return nil, errors.New("user_id debe ser un entero positivo")
	}
	return &value, nil
}

func (a *API) queryActivity(r *http.Request, limit int, userID *int64) ([]activityEntry, error) {
	where := ""
	args := make([]any, 0, 2)
	if userID != nil {
		where = " WHERE combined.user_id = ?"
		args = append(args, *userID)
	}
	args = append(args, limit)

	rows, err := a.db.QueryContext(r.Context(), `
		WITH activity AS (
			SELECT user_id, media_item_id, position_seconds, duration_seconds,
			       updated_at AS activity_at, 0 AS watched
			  FROM playback_progress
			UNION ALL
			SELECT user_id, media_item_id, 0.0, 0.0, watched_at, 1
			  FROM watch_state
		), combined AS (
			SELECT user_id, media_item_id,
			       MAX(activity_at) AS activity_at,
			       MAX(watched) AS watched,
			       MAX(position_seconds) AS position_seconds,
			       MAX(duration_seconds) AS duration_seconds
			  FROM activity
			 GROUP BY user_id, media_item_id
		)
		SELECT combined.user_id, users.username, combined.media_item_id,
		       COALESCE(NULLIF(media_items.episode_title, ''),
		                NULLIF(media_items.display_title, ''),
		                media_items.title),
		       media_items.kind,
		       COALESCE(NULLIF(series.display_title, ''), series.title),
		       media_items.season, media_items.episode,
		       combined.position_seconds, combined.duration_seconds,
		       combined.watched, combined.activity_at
		  FROM combined
		  JOIN users ON users.id = combined.user_id
		  JOIN media_items ON media_items.id = combined.media_item_id
		  LEFT JOIN series ON series.id = media_items.series_id`+where+`
		 ORDER BY combined.activity_at DESC, combined.user_id, combined.media_item_id
		 LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]activityEntry, 0)
	for rows.Next() {
		var entry activityEntry
		var seriesTitle sql.NullString
		var season, episode sql.NullInt64
		var watched int
		if err := rows.Scan(
			&entry.UserID, &entry.Username, &entry.ItemID, &entry.Title,
			&entry.Kind, &seriesTitle, &season, &episode,
			&entry.PositionSeconds, &entry.DurationSeconds, &watched, &entry.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if seriesTitle.Valid {
			value := seriesTitle.String
			entry.SeriesTitle = &value
		}
		if season.Valid {
			value := int(season.Int64)
			entry.Season = &value
		}
		if episode.Valid {
			value := int(episode.Int64)
			entry.Episode = &value
		}
		entry.Watched = watched != 0
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}
