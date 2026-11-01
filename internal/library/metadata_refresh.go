package library

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

const (
	metadataRefreshAfter       = 90 * 24 * time.Hour
	metadataRefreshMovieLimit  = 40
	metadataRefreshSeriesLimit = 20
	metadataRefreshTimeLayout  = "2006-01-02T15:04:05.000Z"
)

type staleMetadataItem struct {
	id     int64
	tmdbID int64
	score  float64
}

// refreshStaleTMDBMetadata vuelve a descargar las fichas TMDB más antiguas.
// Las correcciones manuales se dejan fuera para no pisarlas. Un error de TMDB
// o del contexto detiene el lote para no insistir durante una caída.
func (a *API) refreshStaleTMDBMetadata(ctx context.Context) (int, error) {
	if a == nil || a.db == nil || a.tmdb == nil {
		return 0, nil
	}
	cutoff := a.clockNow().Add(-metadataRefreshAfter).Format(metadataRefreshTimeLayout)

	movies, err := staleMetadataItems(ctx, a.db, `
		SELECT id, tmdb_id, COALESCE(match_score, 0)
		  FROM media_items
		 WHERE kind = 'movie'
		   AND metadata_source = 'tmdb'
		   AND (match_status IS NULL OR match_status = 'auto')
		   AND tmdb_id IS NOT NULL
		   AND (metadata_refreshed_at IS NULL OR metadata_refreshed_at < ?)
		 ORDER BY metadata_refreshed_at IS NOT NULL, metadata_refreshed_at, id
		 LIMIT ?`, cutoff, metadataRefreshMovieLimit)
	if err != nil {
		return 0, err
	}
	series, err := staleMetadataItems(ctx, a.db, `
		SELECT id, tmdb_id, COALESCE(match_score, 0)
		  FROM series
		 WHERE metadata_source = 'tmdb'
		   AND (match_status IS NULL OR match_status = 'auto')
		   AND tmdb_id IS NOT NULL
		   AND (metadata_refreshed_at IS NULL OR metadata_refreshed_at < ?)
		 ORDER BY metadata_refreshed_at IS NOT NULL, metadata_refreshed_at, id
		 LIMIT ?`, cutoff, metadataRefreshSeriesLimit)
	if err != nil {
		return 0, err
	}

	refreshed := 0
	fail := func(err error) (int, error) {
		if refreshed > 0 {
			log.Printf("library: metadatos TMDB: refrescados %d elemento(s)", refreshed)
		}
		return refreshed, err
	}
	for _, item := range movies {
		if refreshed > 0 {
			if err := waitMetadataRefresh(ctx, a.metadataRefreshDelay); err != nil {
				return fail(err)
			}
		}
		if err := applyMovieDetailsFromTMDB(
			ctx, a.db, a.tmdb, item.id, item.tmdbID, "auto", item.score,
		); err != nil {
			return fail(fmt.Errorf("library: refrescar película %d: %w", item.id, err))
		}
		refreshed++
	}
	for _, item := range series {
		if refreshed > 0 {
			if err := waitMetadataRefresh(ctx, a.metadataRefreshDelay); err != nil {
				return fail(err)
			}
		}
		if err := applyTVDetailsFromTMDB(
			ctx, a.db, a.tmdb, item.id, item.tmdbID, "auto", item.score,
		); err != nil {
			return fail(fmt.Errorf("library: refrescar serie %d: %w", item.id, err))
		}
		refreshed++
	}
	if refreshed > 0 {
		log.Printf("library: metadatos TMDB: refrescados %d elemento(s)", refreshed)
	}
	return refreshed, nil
}

func staleMetadataItems(
	ctx context.Context,
	db *sql.DB,
	query string,
	args ...any,
) ([]staleMetadataItem, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("library: listar metadatos TMDB pendientes: %w", err)
	}
	defer rows.Close()

	items := make([]staleMetadataItem, 0)
	for rows.Next() {
		var item staleMetadataItem
		if err := rows.Scan(&item.id, &item.tmdbID, &item.score); err != nil {
			return nil, fmt.Errorf("library: leer metadatos TMDB pendientes: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("library: recorrer metadatos TMDB pendientes: %w", err)
	}
	return items, nil
}

func applyMovieDetailsFromTMDB(
	ctx context.Context,
	db *sql.DB,
	tmdb TMDBClient,
	itemID, tmdbID int64,
	status string,
	score float64,
) error {
	details, err := tmdb.MovieDetails(ctx, int(tmdbID))
	if err != nil {
		return err
	}
	if details == nil {
		return fmt.Errorf("library: TMDB no devolvió la ficha de la película %d", tmdbID)
	}
	return ApplyMovieDetails(ctx, db, itemID, details, status, score)
}

func applyTVDetailsFromTMDB(
	ctx context.Context,
	db *sql.DB,
	tmdb TMDBClient,
	seriesID, tmdbID int64,
	status string,
	score float64,
) error {
	details, err := tmdb.TVDetails(ctx, int(tmdbID))
	if err != nil {
		return err
	}
	if details == nil {
		return fmt.Errorf("library: TMDB no devolvió la ficha de la serie %d", tmdbID)
	}
	return ApplyTVDetails(ctx, db, seriesID, details, status, score)
}

func waitMetadataRefresh(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
