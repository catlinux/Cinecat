package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/catlinux/cinecat/internal/metadata"
)

// ensureSeries crea o actualiza la serie de una carpeta de primer nivel. El
// título analizado se conserva y los metadatos de TMDB no se pisan.
func ensureSeries(ctx context.Context, db *sql.DB, libraryID int64, folder, title string, year int) (int64, error) {
	sortTitle, sortLetter := sortKeys(title)
	_, err := db.ExecContext(ctx, `
		INSERT INTO series (library_id, folder, title, year, sort_title, sort_letter)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (library_id, folder) DO UPDATE SET
			title = excluded.title,
			year = CASE
				WHEN series.metadata_source = 'tmdb' THEN series.year
				WHEN series.year IS NULL THEN excluded.year
				ELSE series.year
			END,
			sort_title = CASE
				WHEN series.metadata_source IS NULL THEN excluded.sort_title
				ELSE series.sort_title
			END,
			sort_letter = CASE
				WHEN series.metadata_source IS NULL THEN excluded.sort_letter
				ELSE series.sort_letter
			END`,
		libraryID, folder, title, nullableInt(year), sortTitle, sortLetter,
	)
	if err != nil {
		return 0, fmt.Errorf("library: guardar la serie %q: %w", folder, err)
	}

	var id int64
	if err := db.QueryRowContext(ctx,
		`SELECT id FROM series WHERE library_id = ? AND folder = ?`,
		libraryID, folder,
	).Scan(&id); err != nil {
		return 0, fmt.Errorf("library: leer la serie %q: %w", folder, err)
	}
	return id, nil
}

type sqlExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// deleteOrphanSeries elimina las series que se quedaron sin episodios. Las
// corregidas a mano se conservan aunque no tengan archivos.
func deleteOrphanSeries(ctx context.Context, db sqlExecer) error {
	_, err := db.ExecContext(ctx, `
		DELETE FROM series
		 WHERE (match_status IS NULL OR match_status <> 'manual')
		   AND NOT EXISTS (
		       SELECT 1 FROM media_items WHERE media_items.series_id = series.id
		   )`)
	if err != nil {
		return fmt.Errorf("library: borrar series huérfanas: %w", err)
	}
	return nil
}

func deleteEmptyAutomaticCollections(ctx context.Context, db sqlExecer) error {
	_, err := db.ExecContext(ctx, `
		DELETE FROM collections
		 WHERE source = 'tmdb'
		   AND NOT EXISTS (
		       SELECT 1 FROM collection_items
		        WHERE collection_items.collection_id = collections.id
		   )`)
	if err != nil {
		return fmt.Errorf("library: borrar colecciones automáticas vacías: %w", err)
	}
	return nil
}

func cleanupOrphans(ctx context.Context, db sqlExecer) error {
	if err := deleteOrphanSeries(ctx, db); err != nil {
		return err
	}
	return deleteEmptyAutomaticCollections(ctx, db)
}

// BackfillSeriesSortKeys completa las claves de orden de las series insertadas
// antes de que existiera el escaneo o directamente en la base de datos.
func BackfillSeriesSortKeys(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `
		SELECT id, COALESCE(NULLIF(display_title, ''), title)
		  FROM series
		 WHERE sort_title IS NULL OR sort_letter IS NULL`)
	if err != nil {
		return fmt.Errorf("library: listar series sin normalizar: %w", err)
	}
	type pending struct {
		id    int64
		title string
	}
	var items []pending
	for rows.Next() {
		var item pending
		if err := rows.Scan(&item.id, &item.title); err != nil {
			rows.Close()
			return fmt.Errorf("library: leer serie sin normalizar: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("library: recorrer series sin normalizar: %w", err)
	}
	rows.Close()

	for _, item := range items {
		sortTitle, sortLetter := sortKeys(item.title)
		if _, err := db.ExecContext(ctx,
			`UPDATE series SET sort_title = ?, sort_letter = ? WHERE id = ?`,
			sortTitle, sortLetter, item.id,
		); err != nil {
			return fmt.Errorf("library: guardar título normalizado de la serie %d: %w", item.id, err)
		}
	}
	return nil
}

// ApplyTVDetails guarda la ficha completa de una serie. El título analizado de
// la carpeta no se toca; el de TMDB se guarda en display_title.
func ApplyTVDetails(ctx context.Context, db *sql.DB, seriesID int64, d *metadata.Details, status string, score float64) error {
	if d == nil {
		return errors.New("library: ficha de serie vacía")
	}
	sortSource := d.Title
	if strings.TrimSpace(sortSource) == "" {
		if err := db.QueryRowContext(ctx,
			`SELECT title FROM series WHERE id = ?`, seriesID,
		).Scan(&sortSource); err != nil {
			return fmt.Errorf("library: leer el título de la serie %d: %w", seriesID, err)
		}
	}
	sortTitle, sortLetter := sortKeys(sortSource)
	_, err := db.ExecContext(ctx, `
		UPDATE series SET
			tmdb_id = ?, original_title = ?, tagline = ?, overview = ?,
			overview_language = ?, year = ?, display_title = ?,
			sort_title = ?, sort_letter = ?, genres = ?, rating = ?,
			directors = ?, cast_names = ?, credits = ?,
			poster_url = CASE WHEN poster_locked = 1 AND tmdb_id = ? THEN poster_url ELSE ? END,
			backdrop_url = CASE WHEN backdrop_locked = 1 AND tmdb_id = ? THEN backdrop_url ELSE ? END,
			poster_locked = CASE WHEN tmdb_id = ? THEN poster_locked ELSE 0 END,
			backdrop_locked = CASE WHEN tmdb_id = ? THEN backdrop_locked ELSE 0 END,
			metadata_source = 'tmdb', match_status = ?, match_score = ?
		WHERE id = ?`,
		nullableInt(d.TMDBID), nullableText(d.OriginalTitle), nullableText(d.Tagline),
		nullableText(d.Overview), nullableText(d.OverviewLanguage), nullableInt(d.Year),
		nullableText(d.Title), sortTitle, sortLetter, jsonArray(d.Genres),
		nullableFloat(d.Rating), jsonArray(d.Directors), jsonArray(d.Cast),
		creditsJSON(d.Credits), d.TMDBID, nullableText(d.PosterURL),
		d.TMDBID, nullableText(d.BackdropURL),
		d.TMDBID, d.TMDBID,
		nullableText(status), score, seriesID,
	)
	if err != nil {
		return fmt.Errorf("library: guardar la ficha de la serie %d: %w", seriesID, err)
	}
	return nil
}

// ClearTVMetadata deja la serie sin ficha y limpia las temporadas y los datos
// de episodios que vinieron de TMDB.
func ClearTVMetadata(ctx context.Context, db *sql.DB, seriesID int64, status any) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("library: iniciar limpieza de la serie %d: %w", seriesID, err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var title string
	err = tx.QueryRowContext(ctx, `SELECT title FROM series WHERE id = ?`, seriesID).Scan(&title)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("library: leer el título de la serie %d: %w", seriesID, err)
	}
	sortTitle, sortLetter := sortKeys(title)

	if _, err := tx.ExecContext(ctx, `
		UPDATE media_items
		   SET episode_title = NULL, still_url = NULL, overview = NULL,
		       overview_language = NULL
		 WHERE series_id = ?`, seriesID); err != nil {
		return fmt.Errorf("library: limpiar los episodios de la serie %d: %w", seriesID, err)
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM series_seasons WHERE series_id = ?`, seriesID); err != nil {
		return fmt.Errorf("library: borrar las temporadas de la serie %d: %w", seriesID, err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE series SET
			tmdb_id = NULL, original_title = NULL, year = NULL, tagline = NULL,
			overview = NULL, overview_language = NULL, display_title = NULL,
			sort_title = ?, sort_letter = ?, genres = NULL, rating = NULL,
			directors = NULL, cast_names = NULL, credits = NULL, poster_url = NULL,
			backdrop_url = NULL, poster_locked = 0, backdrop_locked = 0,
			metadata_source = NULL, match_score = NULL, match_status = ?
		WHERE id = ?`,
		sortTitle, sortLetter, status, seriesID,
	); err != nil {
		return fmt.Errorf("library: borrar la ficha de la serie %d: %w", seriesID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("library: confirmar limpieza de la serie %d: %w", seriesID, err)
	}
	return nil
}

func (s *Scanner) seriesIDsWithEpisodes(ctx context.Context, libraryID int64) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id
		  FROM series s
		 WHERE s.library_id = ?
		   AND EXISTS (SELECT 1 FROM media_items mi WHERE mi.series_id = s.id)
		 ORDER BY s.id`, libraryID)
	if err != nil {
		return nil, fmt.Errorf("library: listar series de la biblioteca %d: %w", libraryID, err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("library: leer serie de la biblioteca %d: %w", libraryID, err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("library: recorrer series de la biblioteca %d: %w", libraryID, err)
	}
	return ids, nil
}

func (s *Scanner) enrichSeriesTMDB(ctx context.Context, seriesID int64, res *Result) {
	var title string
	var year sql.NullInt64
	var matchStatus, source, credits sql.NullString
	var tmdbID sql.NullInt64
	var matchScore sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `
		SELECT title, year, match_status, metadata_source, tmdb_id, match_score, credits
		  FROM series WHERE id = ?`, seriesID,
	).Scan(&title, &year, &matchStatus, &source, &tmdbID, &matchScore, &credits)
	if errors.Is(err, sql.ErrNoRows) {
		return
	}
	if err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("serie %d: no se pudo comprobar los metadatos: %v", seriesID, err))
		return
	}

	canUpdate := source.Valid && source.String == "tmdb" &&
		tmdbID.Valid && tmdbID.Int64 > 0
	identified := matchStatus.Valid && matchStatus.String != "none"
	autoMatched := matchStatus.Valid && matchStatus.String == "auto"

	missingPosters := []int{}
	if canUpdate && autoMatched {
		var posterErr error
		missingPosters, posterErr = seriesSeasonMissingPosters(ctx, s.db, seriesID)
		if posterErr != nil {
			res.Warnings = append(res.Warnings,
				fmt.Sprintf("%s: no se pudieron comprobar las carátulas de temporada: %v", title, posterErr))
			missingPosters = nil
		}
	}

	// Créditos y carátulas de temporada se rellenan con una sola petición a
	// TVDetails, aunque los episodios ya estén completos.
	if canUpdate && identified && (!credits.Valid || len(missingPosters) > 0) {
		score := 0.0
		if matchScore.Valid {
			score = matchScore.Float64
		}
		details, err := s.tmdb.TVDetails(ctx, int(tmdbID.Int64))
		if err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudo obtener la ficha: %v", title, err))
			return
		}
		if !credits.Valid {
			if err := ApplyTVDetails(ctx, s.db, seriesID, details, matchStatus.String, score); err != nil {
				res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudieron guardar los metadatos: %v", title, err))
				return
			}
			if !matchScore.Valid {
				if _, err := s.db.ExecContext(ctx,
					`UPDATE series SET match_score = NULL WHERE id = ?`, seriesID); err != nil {
					res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudieron guardar los metadatos: %v", title, err))
				}
			}
		}
		for _, number := range missingPosters {
			poster := details.SeasonPosters[number]
			// La cadena vacía marca que TMDB ya se consultó y no tiene cartel.
			if _, err := s.db.ExecContext(ctx, `
				UPDATE series_seasons SET poster_url = ?
				 WHERE series_id = ? AND season = ? AND poster_url IS NULL`,
				poster, seriesID, number); err != nil {
				res.Warnings = append(res.Warnings,
					fmt.Sprintf("%s: no se pudo guardar la carátula de la temporada %d: %v", title, number, err))
			}
		}
	}

	if matchStatus.Valid {
		switch matchStatus.String {
		case "none":
			return
		case "manual":
			return
		case "auto":
			if canUpdate {
				s.enrichSeriesSeasons(ctx, seriesID, tmdbID.Int64, title, res)
				return
			}
		}
	}

	candidate, score, ok, err := IdentifySeries(ctx, s.tmdb, title, int(year.Int64))
	if err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudo identificar: %v", title, err))
		return
	}
	if !ok {
		if err := ClearTVMetadata(ctx, s.db, seriesID, nil); err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudieron borrar los metadatos: %v", title, err))
		}
		res.Unidentified++
		res.SeriesUnidentified++
		return
	}

	details, err := s.tmdb.TVDetails(ctx, candidate.TMDBID)
	if err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudo obtener la ficha: %v", title, err))
		return
	}
	if err := ApplyTVDetails(ctx, s.db, seriesID, details, "auto", score); err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudieron guardar los metadatos: %v", title, err))
		return
	}
	res.Identified++
	res.SeriesIdentified++
	s.enrichSeriesSeasons(ctx, seriesID, int64(candidate.TMDBID), title, res)
}

// TVSeasonClient es lo necesario para completar las temporadas de una serie.
type TVSeasonClient interface {
	SeasonDetails(ctx context.Context, tvID, season int) (*metadata.Season, error)
}

// FillSeriesSeasons completa las temporadas y episodios que falten de una
// serie ya identificada. Los avisos son temporadas que TMDB no tiene.
func FillSeriesSeasons(ctx context.Context, db *sql.DB, tmdb TVSeasonClient, seriesID int64) ([]string, error) {
	if tmdb == nil {
		return nil, errors.New("library: falta el cliente de TMDB")
	}
	var title string
	var tmdbID sql.NullInt64
	if err := db.QueryRowContext(ctx,
		`SELECT title, tmdb_id FROM series WHERE id = ?`, seriesID,
	).Scan(&title, &tmdbID); err != nil {
		return nil, fmt.Errorf("library: leer serie %d para sus temporadas: %w", seriesID, err)
	}
	if !tmdbID.Valid || tmdbID.Int64 <= 0 {
		return nil, nil
	}
	return fillSeriesSeasons(ctx, db, tmdb, seriesID, tmdbID.Int64, title)
}

func fillSeriesSeasons(ctx context.Context, db *sql.DB, tmdb TVSeasonClient, seriesID, tmdbID int64, title string) ([]string, error) {
	needs, err := seriesSeasonNeeds(ctx, db, seriesID)
	if err != nil {
		return nil, fmt.Errorf("no se pudieron comprobar las temporadas: %w", err)
	}
	warnings := make([]string, 0)
	for _, need := range needs {
		if !need.missingRow && need.missingTitles == 0 {
			continue
		}
		season, err := tmdb.SeasonDetails(ctx, int(tmdbID), need.number)
		if errors.Is(err, metadata.ErrNotFound) {
			warnings = append(warnings,
				fmt.Sprintf("%s: TMDB no tiene la temporada %d", title, need.number))
			continue
		}
		if err != nil {
			return warnings,
				fmt.Errorf("no se pudo obtener la temporada %d: %w", need.number, err)
		}
		if err := applySeasonDetails(ctx, db, seriesID, need.number, season); err != nil {
			return warnings,
				fmt.Errorf("no se pudo guardar la temporada %d: %w", need.number, err)
		}
	}
	return warnings, nil
}

func (s *Scanner) enrichSeriesSeasons(ctx context.Context, seriesID, tmdbID int64, title string, res *Result) {
	warnings, err := fillSeriesSeasons(ctx, s.db, s.tmdb, seriesID, tmdbID, title)
	res.Warnings = append(res.Warnings, warnings...)
	if err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", title, err))
	}
}

func seriesSeasonMissingPosters(ctx context.Context, db *sql.DB, seriesID int64) ([]int, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT mi.season
		  FROM media_items mi
		  LEFT JOIN series_seasons ss
		    ON ss.series_id = mi.series_id AND ss.season = mi.season
		 WHERE mi.series_id = ? AND mi.season IS NOT NULL
		   AND ss.poster_url IS NULL
		 ORDER BY mi.season`, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var numbers []int
	for rows.Next() {
		var number int
		if err := rows.Scan(&number); err != nil {
			return nil, err
		}
		numbers = append(numbers, number)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return numbers, nil
}

type seasonNeed struct {
	number        int
	missingRow    bool
	missingTitles int
}

func seriesSeasonNeeds(ctx context.Context, db *sql.DB, seriesID int64) ([]seasonNeed, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT mi.season,
		       CASE WHEN ss.season IS NULL THEN 1 ELSE 0 END,
		       SUM(CASE WHEN mi.episode_title IS NULL THEN 1 ELSE 0 END)
		  FROM media_items mi
		  LEFT JOIN series_seasons ss
		    ON ss.series_id = mi.series_id AND ss.season = mi.season
		 WHERE mi.series_id = ? AND mi.season > 0 AND mi.episode IS NOT NULL
		 GROUP BY mi.season, ss.season
		 ORDER BY mi.season`, seriesID)
	if err != nil {
		return nil, fmt.Errorf("library: listar temporadas de la serie %d: %w", seriesID, err)
	}
	defer rows.Close()

	var needs []seasonNeed
	for rows.Next() {
		var need seasonNeed
		var missingRow, missingTitles int64
		if err := rows.Scan(&need.number, &missingRow, &missingTitles); err != nil {
			return nil, fmt.Errorf("library: leer temporada de la serie %d: %w", seriesID, err)
		}
		need.missingRow = missingRow != 0
		need.missingTitles = int(missingTitles)
		needs = append(needs, need)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("library: recorrer temporadas de la serie %d: %w", seriesID, err)
	}
	return needs, nil
}

func applySeasonDetails(ctx context.Context, db *sql.DB, seriesID int64, seasonNumber int, season *metadata.Season) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("library: iniciar temporada %d de la serie %d: %w", seasonNumber, seriesID, err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO series_seasons (series_id, season, name, overview, poster_url)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (series_id, season) DO UPDATE SET
			name = excluded.name,
			overview = excluded.overview,
			poster_url = excluded.poster_url`,
		seriesID, seasonNumber, nullableText(season.Name), nullableText(season.Overview),
		season.PosterURL,
	); err != nil {
		return fmt.Errorf("library: guardar la temporada %d de la serie %d: %w", seasonNumber, seriesID, err)
	}

	language := season.OverviewLanguage
	if language == "" {
		language = "es-ES"
	}
	for _, episode := range season.Episodes {
		if episode.Number <= 0 {
			continue
		}
		var overview, overviewLanguage any
		if strings.TrimSpace(episode.Overview) != "" {
			overview = episode.Overview
			overviewLanguage = language
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE media_items SET
				episode_title = ?,
				overview = CASE WHEN ? IS NOT NULL THEN ? ELSE overview END,
				overview_language = CASE WHEN ? IS NOT NULL THEN ? ELSE overview_language END,
				still_url = ?,
				year = CASE WHEN year IS NULL AND ? > 0 THEN ? ELSE year END
			WHERE series_id = ? AND season = ? AND episode = ?`,
			nullableText(episode.Name), overview, overview, overviewLanguage, overviewLanguage,
			nullableText(episode.StillURL), episode.AirYear, episode.AirYear,
			seriesID, seasonNumber, episode.Number,
		); err != nil {
			return fmt.Errorf("library: guardar el episodio %dx%d de la serie %d: %w",
				seasonNumber, episode.Number, seriesID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("library: confirmar temporada %d de la serie %d: %w", seasonNumber, seriesID, err)
	}
	return nil
}
