package library

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	listFavorite  = "favorite"
	listWatchlist = "watchlist"
)

type listsTotals struct {
	Items       int64 `json:"items"`
	Series      int64 `json:"series"`
	Collections int64 `json:"collections"`
}

type listsResponse struct {
	Items       []mediaItemDTO  `json:"items"`
	Series      []seriesListDTO `json:"series"`
	Collections []collectionDTO `json:"collections"`
	Totals      listsTotals     `json:"totals"`
	Total       *int64          `json:"total,omitempty"`
	Limit       int             `json:"limit"`
	Offset      int             `json:"offset"`
}

type listedItem struct {
	Item    mediaItemDTO
	AddedAt string
}

type listedSeries struct {
	Series  seriesListDTO
	AddedAt string
}

type listedCollection struct {
	Collection collectionDTO
	AddedAt    string
}

func validListName(value string) bool {
	return value == listFavorite || value == listWatchlist
}

func queryListTotals(ctx context.Context, db *sql.DB, userID int64, list string) (listsTotals, error) {
	var totals listsTotals
	err := db.QueryRowContext(ctx, `
		SELECT
		  (SELECT count(*) FROM item_lists WHERE user_id = ? AND list = ?),
		  (SELECT count(*) FROM series_lists WHERE user_id = ? AND list = ?),
		  (SELECT count(*) FROM collection_lists cl
		     JOIN collections c ON c.id = cl.collection_id AND c.deleted = 0
		    WHERE cl.user_id = ? AND cl.list = ?)`,
		userID, list, userID, list, userID, list).
		Scan(&totals.Items, &totals.Series, &totals.Collections)
	return totals, err
}

func queryListItems(ctx context.Context, db *sql.DB, userID int64, list string, limit, offset int) ([]listedItem, int64, error) {
	var total int64
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM item_lists WHERE user_id = ? AND list = ?`, userID, list).
		Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT `+mediaItemColumns+watchedColumn+progressColumn+itemListColumns+`, il.added_at
		  FROM item_lists il
		  JOIN media_items ON media_items.id = il.media_item_id
		  LEFT JOIN watch_state ws
		    ON ws.media_item_id = media_items.id AND ws.user_id = ?
		  LEFT JOIN playback_progress pp
		    ON pp.media_item_id = media_items.id AND pp.user_id = ?
		 WHERE il.user_id = ? AND il.list = ?
		 ORDER BY il.added_at DESC, il.media_item_id DESC
		 LIMIT ? OFFSET ?`,
		userID, userID, userID, userID, userID, list, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]listedItem, 0)
	for rows.Next() {
		var addedAt string
		scan := func(dest ...any) error {
			all := append(append([]any{}, dest...), &addedAt)
			return rows.Scan(all...)
		}
		item, err := scanItem(scan)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, listedItem{Item: item, AddedAt: addedAt})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(items) > 0 {
		ids := make([]int64, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.Item.ID)
		}
		collections, err := loadItemCollections(ctx, db, ids)
		if err != nil {
			return nil, 0, err
		}
		for index := range items {
			if found, ok := collections[items[index].Item.ID]; ok {
				items[index].Item.Collections = found
			}
		}
	}
	return items, total, nil
}

func queryListSeries(ctx context.Context, db *sql.DB, userID int64, list string, limit, offset int) ([]listedSeries, int64, error) {
	var total int64
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM series_lists WHERE user_id = ? AND list = ?`, userID, list).
		Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT s.id, COALESCE(NULLIF(s.display_title, ''), s.title), s.title,
		       s.year, s.poster_url, s.genres,
		       (SELECT count(DISTINCT mi.season) FROM media_items mi
		         WHERE mi.series_id = s.id AND mi.season IS NOT NULL),
		       (SELECT count(*) FROM media_items mi WHERE mi.series_id = s.id),
		       (SELECT count(*) FROM media_items mi
		          JOIN watch_state ws ON ws.media_item_id = mi.id AND ws.user_id = ?
		         WHERE mi.series_id = s.id),
		       s.rating, s.trailer_url, s.trailer_language,
		       s.match_status,
		       EXISTS (SELECT 1 FROM series_lists sl_favorite
		                WHERE sl_favorite.user_id = ?
		                  AND sl_favorite.series_id = s.id
		                  AND sl_favorite.list = 'favorite'),
		       EXISTS (SELECT 1 FROM series_lists sl_watchlist
		                WHERE sl_watchlist.user_id = ?
		                  AND sl_watchlist.series_id = s.id
		                  AND sl_watchlist.list = 'watchlist'),
		       sl.added_at
		  FROM series_lists sl
		  JOIN series s ON s.id = sl.series_id
		 WHERE sl.user_id = ? AND sl.list = ?
		 ORDER BY sl.added_at DESC, sl.series_id DESC
		 LIMIT ? OFFSET ?`,
		userID, userID, userID, userID, list, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	series := make([]listedSeries, 0)
	for rows.Next() {
		var addedAt string
		scan := func(dest ...any) error {
			all := append(append([]any{}, dest...), &addedAt)
			return rows.Scan(all...)
		}
		item, err := scanSeriesList(scan)
		if err != nil {
			return nil, 0, err
		}
		series = append(series, listedSeries{Series: item, AddedAt: addedAt})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return series, total, nil
}

func queryListCollections(ctx context.Context, db *sql.DB, userID int64, list string, limit, offset int) ([]listedCollection, int64, error) {
	var total int64
	if err := db.QueryRowContext(ctx, `
		SELECT count(*) FROM collection_lists cl
		  JOIN collections c ON c.id = cl.collection_id AND c.deleted = 0
		 WHERE cl.user_id = ? AND cl.list = ?`, userID, list).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT c.id, c.name, c.overview, `+collectionPosterExpr+`, c.backdrop_url,
		       `+collectionItemCountExpr+`, c.source, c.edited,
		       EXISTS (SELECT 1 FROM collection_lists cl_favorite
		                WHERE cl_favorite.user_id = ?
		                  AND cl_favorite.collection_id = c.id
		                  AND cl_favorite.list = 'favorite'),
		       EXISTS (SELECT 1 FROM collection_lists cl_watchlist
		                WHERE cl_watchlist.user_id = ?
		                  AND cl_watchlist.collection_id = c.id
		                  AND cl_watchlist.list = 'watchlist'),
		       cl.added_at
		  FROM collection_lists cl
		  JOIN collections c ON c.id = cl.collection_id AND c.deleted = 0
		 WHERE cl.user_id = ? AND cl.list = ?
		 ORDER BY cl.added_at DESC, cl.collection_id DESC
		 LIMIT ? OFFSET ?`,
		userID, userID, userID, list, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	collections := make([]listedCollection, 0)
	for rows.Next() {
		var addedAt string
		scan := func(dest ...any) error {
			all := append(append([]any{}, dest...), &addedAt)
			return rows.Scan(all...)
		}
		item, err := scanCollection(scan)
		if err != nil {
			return nil, 0, err
		}
		collections = append(collections, listedCollection{Collection: item, AddedAt: addedAt})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return collections, total, nil
}

func listKindName(kind string) (string, error) {
	switch kind {
	case "", "items", "series", "collections":
		return kind, nil
	default:
		return "", fmt.Errorf("kind debe ser items, series o collections")
	}
}

func clearSeriesWatchlistIfWatched(ctx context.Context, tx *sql.Tx, userID, seriesID int64) error {
	var total, watched int64
	if err := tx.QueryRowContext(ctx, `
		SELECT count(*), count(ws.media_item_id)
		  FROM media_items mi
		  LEFT JOIN watch_state ws
		    ON ws.media_item_id = mi.id AND ws.user_id = ?
		 WHERE mi.series_id = ?`, userID, seriesID).Scan(&total, &watched); err != nil {
		return err
	}
	if total == 0 || watched < total {
		return nil
	}
	_, err := tx.ExecContext(ctx,
		`DELETE FROM series_lists WHERE user_id = ? AND series_id = ? AND list = 'watchlist'`,
		userID, seriesID)
	return err
}
