package library

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"sort"
)

type homeCounts struct {
	Movies   int64 `json:"movies"`
	Episodes int64 `json:"episodes"`
	Series   int64 `json:"series"`
}

type homeListEntry struct {
	Kind   string         `json:"kind"`
	Item   *mediaItemDTO  `json:"item,omitempty"`
	Series *seriesListDTO `json:"series,omitempty"`
}

type homeResponse struct {
	Continue     []mediaItemDTO  `json:"continue"`
	NextUp       []mediaItemDTO  `json:"next_up"`
	RecentMovies []mediaItemDTO  `json:"recent_movies"`
	RecentSeries []seriesListDTO `json:"recent_series"`
	Watchlist    []homeListEntry `json:"watchlist"`
	Favorites    []homeListEntry `json:"favorites"`
	HasClips     bool            `json:"has_clips"`
	HasMusic     bool            `json:"has_music"`
	Counts       homeCounts      `json:"counts"`
}

func (a *API) handleHome(w http.ResponseWriter, r *http.Request) {
	user, ok := sessionUserFromRequest(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "falta el token de sesión")
		return
	}

	continueItems, err := queryHomeItems(r.Context(), a.db, `
		SELECT `+mediaItemColumns+watchedColumn+progressColumn+itemListColumns+`
		  FROM media_items
		  JOIN playback_progress pp
		    ON pp.media_item_id = media_items.id AND pp.user_id = ?
		  LEFT JOIN watch_state ws
		    ON ws.media_item_id = media_items.id AND ws.user_id = ?
		 WHERE media_items.kind IN ('movie', 'episode')
		 ORDER BY pp.updated_at DESC, media_items.id DESC
		 LIMIT 20`, user.ID, user.ID, user.ID, user.ID)
	if err != nil {
		log.Printf("library: listar continuar viendo: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	recentMovies, err := queryHomeItems(r.Context(), a.db, `
		SELECT `+mediaItemColumns+watchedColumn+progressColumn+itemListColumns+`
		  FROM media_items
		  LEFT JOIN watch_state ws
		    ON ws.media_item_id = media_items.id AND ws.user_id = ?
		  LEFT JOIN playback_progress pp
		    ON pp.media_item_id = media_items.id AND pp.user_id = ?
		 WHERE media_items.kind = 'movie'
		   AND `+videoLibraryClause("media_items")+`
		 ORDER BY media_items.id DESC
		 LIMIT 20`, user.ID, user.ID, user.ID, user.ID)
	if err != nil {
		log.Printf("library: listar películas recientes: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	recentSeries, err := queryRecentSeries(r.Context(), a.db, user.ID, 20)
	if err != nil {
		log.Printf("library: listar series recientes: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	nextUp, err := queryHomeNextUp(r.Context(), a.db, user.ID, a.collectionMin())
	if err != nil {
		log.Printf("library: listar siguientes episodios: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	watchlist, err := queryHomeList(r.Context(), a.db, user.ID, listWatchlist)
	if err != nil {
		log.Printf("library: listar próximas a ver: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	favorites, err := queryHomeList(r.Context(), a.db, user.ID, listFavorite)
	if err != nil {
		log.Printf("library: listar favoritos: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	var counts homeCounts
	if err := a.db.QueryRowContext(r.Context(), `
		SELECT COALESCE(SUM(CASE WHEN kind = 'movie' AND `+videoLibraryClause("media_items")+` THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN kind = 'episode' AND `+videoLibraryClause("media_items")+` THEN 1 ELSE 0 END), 0),
		       (SELECT count(*) FROM series s
		         JOIN libraries l ON l.id = s.library_id AND l.kind = 'video')
		  FROM media_items`).Scan(&counts.Movies, &counts.Episodes, &counts.Series); err != nil {
		log.Printf("library: contar ítems del portal: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	var hasClips int
	if err := a.db.QueryRowContext(r.Context(), `
		SELECT EXISTS (
			SELECT 1 FROM libraries l
			 WHERE l.kind = 'clips'
			   AND EXISTS (SELECT 1 FROM media_items mi WHERE mi.library_id = l.id)
		)`).Scan(&hasClips); err != nil {
		log.Printf("library: comprobar bibliotecas de vídeos: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	var hasMusic int
	if err := a.db.QueryRowContext(r.Context(), `
		SELECT EXISTS (
			SELECT 1 FROM music_tracks mt
			  JOIN libraries l ON l.id = mt.library_id
			 WHERE l.kind = 'music'
		)`).Scan(&hasMusic); err != nil {
		log.Printf("library: comprobar bibliotecas de música: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	admin := isAdminRequest(r)
	compactMediaItemListSlice(continueItems, admin)
	compactMediaItemListSlice(nextUp, admin)
	compactMediaItemListSlice(recentMovies, admin)
	for _, list := range [][]homeListEntry{watchlist, favorites} {
		for i := range list {
			if list[i].Item != nil {
				compactMediaItemList(list[i].Item, admin)
			}
		}
	}

	writeJSON(w, http.StatusOK, homeResponse{
		Continue:     continueItems,
		NextUp:       nextUp,
		RecentMovies: recentMovies,
		RecentSeries: recentSeries,
		Watchlist:    watchlist,
		Favorites:    favorites,
		HasClips:     hasClips != 0,
		HasMusic:     hasMusic != 0,
		Counts:       counts,
	})
}

func queryHomeNextUp(
	ctx context.Context,
	db *sql.DB,
	userID int64,
	minCollectionItems ...int,
) ([]mediaItemDTO, error) {
	minimum := collectionMinimum(minCollectionItems...)
	rows, err := db.QueryContext(ctx, `
		WITH watched AS (
			SELECT mi.series_id, MAX(ws.watched_at) AS activity
			  FROM media_items mi
			  JOIN watch_state ws
			    ON ws.media_item_id = mi.id AND ws.user_id = ?
			 WHERE mi.series_id IS NOT NULL
			   AND `+videoLibraryClause("mi")+`
			 GROUP BY mi.series_id
		),
		in_progress AS (
			SELECT DISTINCT mi.series_id
			  FROM media_items mi
			  JOIN playback_progress pp
			    ON pp.media_item_id = mi.id AND pp.user_id = ?
			 WHERE mi.series_id IS NOT NULL
			   AND `+videoLibraryClause("mi")+`
		),
		next_episode AS (
			SELECT watched.series_id, watched.activity,
			       (SELECT mi.id
			          FROM media_items mi
			         WHERE mi.series_id = watched.series_id
			           AND `+videoLibraryClause("mi")+`
			           AND NOT EXISTS (
			               SELECT 1 FROM watch_state w2
			                WHERE w2.media_item_id = mi.id AND w2.user_id = ?)
			         ORDER BY mi.season IS NULL, mi.season,
			                  mi.episode IS NULL, mi.episode, mi.id
			         LIMIT 1) AS episode_id
			  FROM watched
			 WHERE NOT EXISTS (
			     SELECT 1 FROM in_progress p WHERE p.series_id = watched.series_id)
		)
		SELECT series_id, episode_id, activity
		  FROM next_episode
		 WHERE episode_id IS NOT NULL
		 ORDER BY activity DESC, series_id
		 LIMIT 20`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type nextSeries struct {
		seriesID  int64
		episodeID int64
	}
	next := make([]nextSeries, 0)
	for rows.Next() {
		var item nextSeries
		var activity sql.NullString
		if err := rows.Scan(&item.seriesID, &item.episodeID, &activity); err != nil {
			return nil, err
		}
		next = append(next, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	items := make([]mediaItemDTO, 0, len(next))
	for _, item := range next {
		dto, err := queryMediaItem(ctx, db, item.episodeID, userID, minimum)
		if err != nil {
			return nil, err
		}
		items = append(items, dto)
	}
	return items, nil
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

func queryHomeList(ctx context.Context, db *sql.DB, userID int64, list string) ([]homeListEntry, error) {
	items, _, err := queryListItems(ctx, db, userID, list, 20, 0)
	if err != nil {
		return nil, err
	}
	series, _, err := queryListSeries(ctx, db, userID, list, 20, 0)
	if err != nil {
		return nil, err
	}
	type entry struct {
		addedAt string
		value   homeListEntry
	}
	entries := make([]entry, 0, len(items)+len(series))
	for i := range items {
		item := items[i].Item
		entries = append(entries, entry{
			addedAt: items[i].AddedAt,
			value:   homeListEntry{Kind: "item", Item: &item},
		})
	}
	for i := range series {
		item := series[i].Series
		entries = append(entries, entry{
			addedAt: series[i].AddedAt,
			value:   homeListEntry{Kind: "series", Series: &item},
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].addedAt > entries[j].addedAt
	})
	result := make([]homeListEntry, 0, len(entries))
	for _, item := range entries {
		result = append(result, item.value)
		if len(result) == 20 {
			break
		}
	}
	return result, nil
}
