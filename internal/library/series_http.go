package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/catlinux/cinecat/internal/metadata"
)

const (
	defaultSeriesLimit = 48
	maxSeriesLimit     = 200
)

type seriesListDTO struct {
	ID           int64    `json:"id"`
	Title        string   `json:"title"`
	FileTitle    string   `json:"file_title"`
	Year         *int     `json:"year"`
	PosterURL    *string  `json:"poster_url"`
	Genres       []string `json:"genres"`
	SeasonCount  int64    `json:"season_count"`
	EpisodeCount int64    `json:"episode_count"`
	WatchedCount int64    `json:"watched_count"`
	MatchStatus  *string  `json:"match_status"`
}

type seriesResponse struct {
	Series []seriesListDTO `json:"series"`
	Total  int64           `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type seriesSeasonDTO struct {
	Season       int     `json:"season"`
	Name         string  `json:"name"`
	Overview     *string `json:"overview"`
	PosterURL    *string `json:"poster_url"`
	EpisodeCount int64   `json:"episode_count"`
	WatchedCount int64   `json:"watched_count"`
}

type seriesDetailDTO struct {
	ID               int64             `json:"id"`
	Title            string            `json:"title"`
	FileTitle        string            `json:"file_title"`
	Folder           string            `json:"folder"`
	OriginalTitle    *string           `json:"original_title"`
	Year             *int              `json:"year"`
	Tagline          *string           `json:"tagline"`
	Overview         *string           `json:"overview"`
	OverviewLanguage *string           `json:"overview_language"`
	PosterURL        *string           `json:"poster_url"`
	BackdropURL      *string           `json:"backdrop_url"`
	Genres           []string          `json:"genres"`
	Rating           *float64          `json:"rating"`
	Directors        []string          `json:"directors"`
	Cast             []string          `json:"cast"`
	TMDBID           *int64            `json:"tmdb_id"`
	MatchStatus      *string           `json:"match_status"`
	MatchScore       *float64          `json:"match_score"`
	PosterCustom     bool              `json:"poster_custom"`
	BackdropCustom   bool              `json:"backdrop_custom"`
	SeasonCount      int64             `json:"season_count"`
	EpisodeCount     int64             `json:"episode_count"`
	WatchedCount     int64             `json:"watched_count"`
	Seasons          []seriesSeasonDTO `json:"seasons"`
	NextEpisode      *mediaItemDTO     `json:"next_episode"`
}

type seriesSeasonDetailDTO struct {
	Season    int            `json:"season"`
	Name      string         `json:"name"`
	Overview  *string        `json:"overview"`
	PosterURL *string        `json:"poster_url"`
	Items     []mediaItemDTO `json:"items"`
}

type seriesRecord struct {
	ID          int64
	Title       string
	FileTitle   string
	Folder      string
	Year        *int
	TMDBID      *int64
	MatchStatus *string
}

func seriesPagination(r *http.Request) (limit, offset int, err error) {
	limit = defaultSeriesLimit
	query := r.URL.Query()
	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		value, convErr := strconv.Atoi(raw)
		if convErr != nil || value < 1 || value > maxSeriesLimit {
			return 0, 0, fmt.Errorf("limit debe estar entre 1 y %d", maxSeriesLimit)
		}
		limit = value
	}
	if raw := strings.TrimSpace(query.Get("offset")); raw != "" {
		value, convErr := strconv.Atoi(raw)
		if convErr != nil || value < 0 {
			return 0, 0, errors.New("offset debe ser un número mayor o igual que 0")
		}
		offset = value
	}
	return limit, offset, nil
}

func (a *API) handleSeries(w http.ResponseWriter, r *http.Request) {
	if err := BackfillSeriesSortKeys(r.Context(), a.db); err != nil {
		log.Printf("library: normalizar títulos de series: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	limit, offset, err := seriesPagination(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	query := r.URL.Query()
	q := strings.TrimSpace(query.Get("q"))
	letter := strings.TrimSpace(query.Get("letter"))
	if letter != "" {
		parsed, parseErr := parseLetter(letter)
		if parseErr != nil {
			writeJSONError(w, http.StatusBadRequest, parseErr.Error())
			return
		}
		letter = parsed
	}
	genre := strings.TrimSpace(query.Get("genre"))
	unmatched := strings.TrimSpace(query.Get("unmatched"))
	switch unmatched {
	case "", "true", "false":
	default:
		writeJSONError(w, http.StatusBadRequest, "unmatched debe ser true o false")
		return
	}
	sortOrder := strings.TrimSpace(query.Get("sort"))
	if sortOrder == "" {
		sortOrder = "title"
	}

	where := "1 = 1"
	args := make([]any, 0)
	if q != "" {
		pattern := likePattern(q)
		where += ` AND (LOWER(s.display_title) LIKE LOWER(?) ESCAPE '\'
			OR LOWER(s.title) LIKE LOWER(?) ESCAPE '\'
			OR LOWER(s.original_title) LIKE LOWER(?) ESCAPE '\'
			OR LOWER(s.cast_names) LIKE LOWER(?) ESCAPE '\'
			OR LOWER(s.directors) LIKE LOWER(?) ESCAPE '\')`
		args = append(args, pattern, pattern, pattern, pattern, pattern)
	}
	if letter != "" {
		where += " AND s.sort_letter = ?"
		args = append(args, letter)
	}
	if genre != "" {
		where += ` AND EXISTS (
			SELECT 1 FROM json_each(CASE WHEN s.genres IS NOT NULL AND json_valid(s.genres)
				THEN s.genres ELSE '[]' END) je
			WHERE LOWER(je.value) = LOWER(?))`
		args = append(args, genre)
	}
	if unmatched == "true" {
		where += " AND (s.match_status IS NULL OR s.match_status = 'none')"
	}

	var total int64
	if err := a.db.QueryRowContext(r.Context(),
		`SELECT count(*) FROM series s WHERE `+where, args...).Scan(&total); err != nil {
		log.Printf("library: contar series: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	order := " ORDER BY s.sort_title COLLATE NOCASE, s.id"
	switch sortOrder {
	case "title":
	case "year":
		order = " ORDER BY (s.year IS NULL) ASC, s.year DESC, s.sort_title COLLATE NOCASE, s.id"
	case "added":
		order = " ORDER BY s.id DESC"
	default:
		writeJSONError(w, http.StatusBadRequest, "sort debe ser title, year o added")
		return
	}

	userID := sessionUserID(r)
	queryArgs := append([]any{userID}, args...)
	queryArgs = append(queryArgs, limit, offset)
	rows, err := a.db.QueryContext(r.Context(), `
		SELECT s.id, COALESCE(NULLIF(s.display_title, ''), s.title), s.title,
		       s.year, s.poster_url, s.genres,
		       (SELECT count(DISTINCT mi.season) FROM media_items mi
		         WHERE mi.series_id = s.id AND mi.season IS NOT NULL),
		       (SELECT count(*) FROM media_items mi WHERE mi.series_id = s.id),
		       (SELECT count(*) FROM media_items mi
		          JOIN watch_state ws ON ws.media_item_id = mi.id AND ws.user_id = ?
		         WHERE mi.series_id = s.id),
		       s.match_status
		  FROM series s
		 WHERE `+where+order+` LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		log.Printf("library: listar series: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer rows.Close()

	series := make([]seriesListDTO, 0)
	for rows.Next() {
		var item seriesListDTO
		var year sql.NullInt64
		var poster, genres, matchStatus sql.NullString
		if err := rows.Scan(&item.ID, &item.Title, &item.FileTitle, &year, &poster,
			&genres, &item.SeasonCount, &item.EpisodeCount, &item.WatchedCount,
			&matchStatus); err != nil {
			log.Printf("library: leer serie: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		item.Year = intPointer(year)
		item.PosterURL = stringPointer(poster)
		item.Genres = decodeStringArray(genres)
		item.MatchStatus = stringPointer(matchStatus)
		series = append(series, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("library: recorrer series: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	writeJSON(w, http.StatusOK, seriesResponse{
		Series: series, Total: total, Limit: limit, Offset: offset,
	})
}

func (a *API) handleSeriesGenres(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.QueryContext(r.Context(), `
		SELECT je.value, COUNT(DISTINCT s.id)
		  FROM series s
		  JOIN json_each(CASE WHEN s.genres IS NOT NULL AND json_valid(s.genres)
			THEN s.genres ELSE '[]' END) je
		 GROUP BY je.value
		 ORDER BY je.value COLLATE NOCASE, je.value`)
	if err != nil {
		log.Printf("library: listar géneros de series: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer rows.Close()

	genres := make([]genreCount, 0)
	for rows.Next() {
		var name sql.NullString
		var count int64
		if err := rows.Scan(&name, &count); err != nil {
			log.Printf("library: leer género de serie: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		if !name.Valid || strings.TrimSpace(name.String) == "" {
			continue
		}
		genres = append(genres, genreCount{Name: name.String, Count: count})
	}
	if err := rows.Err(); err != nil {
		log.Printf("library: recorrer géneros de series: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, genresResponse{Genres: genres})
}

func (a *API) loadSeriesRecord(w http.ResponseWriter, r *http.Request) (seriesRecord, bool) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "serie no encontrada")
		return seriesRecord{}, false
	}
	var record seriesRecord
	var year, tmdbID sql.NullInt64
	var matchStatus sql.NullString
	err := a.db.QueryRowContext(r.Context(), `
		SELECT id, COALESCE(NULLIF(display_title, ''), title), title, folder,
		       year, tmdb_id, match_status
		  FROM series WHERE id = ?`, id).
		Scan(&record.ID, &record.Title, &record.FileTitle, &record.Folder,
			&year, &tmdbID, &matchStatus)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "serie no encontrada")
		return seriesRecord{}, false
	case err != nil:
		log.Printf("library: leer serie %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return seriesRecord{}, false
	}
	record.Year = intPointer(year)
	record.TMDBID = int64Pointer(tmdbID)
	record.MatchStatus = stringPointer(matchStatus)
	return record, true
}

func querySeriesDetail(ctx context.Context, db *sql.DB, seriesID, userID int64) (seriesDetailDTO, error) {
	var detail seriesDetailDTO
	var originalTitle, tagline, overview, overviewLanguage, poster, backdrop sql.NullString
	var genres, directors, castNames, matchStatus sql.NullString
	var year, tmdbID sql.NullInt64
	var rating, matchScore sql.NullFloat64
	var posterLocked, backdropLocked int
	err := db.QueryRowContext(ctx, `
		SELECT s.id, COALESCE(NULLIF(s.display_title, ''), s.title), s.title, s.folder,
		       s.original_title, s.year, s.tagline, s.overview, s.overview_language,
		       s.poster_url, s.backdrop_url, s.genres, s.rating, s.directors,
		       s.cast_names, s.tmdb_id, s.match_status, s.match_score,
		       s.poster_locked, s.backdrop_locked,
		       (SELECT count(*) FROM media_items mi WHERE mi.series_id = s.id),
		       (SELECT count(*) FROM media_items mi
		          JOIN watch_state ws ON ws.media_item_id = mi.id AND ws.user_id = ?
		         WHERE mi.series_id = s.id)
		  FROM series s WHERE s.id = ?`, userID, seriesID).
		Scan(&detail.ID, &detail.Title, &detail.FileTitle, &detail.Folder,
			&originalTitle, &year, &tagline, &overview, &overviewLanguage,
			&poster, &backdrop, &genres, &rating, &directors, &castNames,
			&tmdbID, &matchStatus, &matchScore, &posterLocked, &backdropLocked,
			&detail.EpisodeCount, &detail.WatchedCount)
	if err != nil {
		return seriesDetailDTO{}, err
	}
	detail.OriginalTitle = stringPointer(originalTitle)
	detail.Year = intPointer(year)
	detail.Tagline = stringPointer(tagline)
	detail.Overview = stringPointer(overview)
	detail.OverviewLanguage = stringPointer(overviewLanguage)
	detail.PosterURL = stringPointer(poster)
	detail.BackdropURL = stringPointer(backdrop)
	detail.Genres = decodeStringArray(genres)
	detail.Rating = floatPointer(rating)
	detail.Directors = decodeStringArray(directors)
	detail.Cast = decodeStringArray(castNames)
	detail.TMDBID = int64Pointer(tmdbID)
	detail.MatchStatus = stringPointer(matchStatus)
	detail.MatchScore = floatPointer(matchScore)
	detail.PosterCustom = posterLocked != 0
	detail.BackdropCustom = backdropLocked != 0

	seasons, err := querySeriesSeasons(ctx, db, seriesID, userID)
	if err != nil {
		return seriesDetailDTO{}, err
	}
	detail.Seasons = seasons
	detail.SeasonCount = int64(len(seasons))

	next, err := queryNextSeriesEpisode(ctx, db, seriesID, userID)
	if err != nil {
		return seriesDetailDTO{}, err
	}
	if next != nil {
		item, err := queryMediaItem(ctx, db, *next, userID)
		if err != nil {
			return seriesDetailDTO{}, err
		}
		detail.NextEpisode = &item
	}
	return detail, nil
}

func querySeriesSeasons(ctx context.Context, db *sql.DB, seriesID, userID int64) ([]seriesSeasonDTO, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT mi.season,
		       COALESCE(NULLIF(ss.name, ''), 'Temporada ' || mi.season),
		       ss.overview, ss.poster_url,
		       count(*),
		       sum(CASE WHEN ws.user_id IS NULL THEN 0 ELSE 1 END)
		  FROM media_items mi
		  LEFT JOIN series_seasons ss
		    ON ss.series_id = mi.series_id AND ss.season = mi.season
		  LEFT JOIN watch_state ws
		    ON ws.media_item_id = mi.id AND ws.user_id = ?
		 WHERE mi.series_id = ? AND mi.season IS NOT NULL
		 GROUP BY mi.season, ss.name, ss.overview, ss.poster_url
		 ORDER BY mi.season`, userID, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seasons := make([]seriesSeasonDTO, 0)
	for rows.Next() {
		var season seriesSeasonDTO
		var overview, poster sql.NullString
		if err := rows.Scan(&season.Season, &season.Name, &overview, &poster,
			&season.EpisodeCount, &season.WatchedCount); err != nil {
			return nil, err
		}
		season.Overview = stringPointer(overview)
		season.PosterURL = stringPointer(poster)
		seasons = append(seasons, season)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return seasons, nil
}

func queryNextSeriesEpisode(ctx context.Context, db *sql.DB, seriesID, userID int64) (*int64, error) {
	var id int64
	err := db.QueryRowContext(ctx, `
		SELECT mi.id
		  FROM media_items mi
		  JOIN playback_progress pp
		    ON pp.media_item_id = mi.id AND pp.user_id = ?
		 WHERE mi.series_id = ?
		 ORDER BY pp.updated_at DESC, mi.id DESC
		 LIMIT 1`, userID, seriesID).Scan(&id)
	if err == nil {
		return &id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	err = db.QueryRowContext(ctx, `
		SELECT mi.id
		  FROM media_items mi
		  LEFT JOIN watch_state ws
		    ON ws.media_item_id = mi.id AND ws.user_id = ?
		 WHERE mi.series_id = ? AND ws.user_id IS NULL
		 ORDER BY mi.season IS NULL, mi.season, mi.episode IS NULL, mi.episode, mi.id
		 LIMIT 1`, userID, seriesID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (a *API) handleSeriesDetail(w http.ResponseWriter, r *http.Request) {
	record, ok := a.loadSeriesRecord(w, r)
	if !ok {
		return
	}
	detail, err := querySeriesDetail(r.Context(), a.db, record.ID, sessionUserID(r))
	if err != nil {
		log.Printf("library: leer ficha de la serie %d: %v", record.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (a *API) handleSeriesSeason(w http.ResponseWriter, r *http.Request) {
	record, ok := a.loadSeriesRecord(w, r)
	if !ok {
		return
	}
	seasonNumber, err := strconv.Atoi(strings.TrimSpace(r.PathValue("n")))
	if err != nil || seasonNumber <= 0 {
		writeJSONError(w, http.StatusNotFound, "temporada no encontrada")
		return
	}
	userID := sessionUserID(r)
	items, err := a.querySeriesItems(r.Context(), record.ID, &seasonNumber, userID)
	if err != nil {
		log.Printf("library: leer episodios de la serie %d: %v", record.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if len(items) == 0 {
		writeJSONError(w, http.StatusNotFound, "temporada no encontrada")
		return
	}

	var name string
	var overview, poster sql.NullString
	err = a.db.QueryRowContext(r.Context(), `
		SELECT COALESCE(NULLIF(name, ''), 'Temporada ' || ?), overview, poster_url
		  FROM series_seasons WHERE series_id = ? AND season = ?`,
		seasonNumber, record.ID, seasonNumber).Scan(&name, &overview, &poster)
	if errors.Is(err, sql.ErrNoRows) {
		name = fmt.Sprintf("Temporada %d", seasonNumber)
		overview = sql.NullString{}
		poster = sql.NullString{}
	} else if err != nil {
		log.Printf("library: leer temporada %d de la serie %d: %v", seasonNumber, record.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, seriesSeasonDetailDTO{
		Season: seasonNumber, Name: name, Overview: stringPointer(overview),
		PosterURL: stringPointer(poster), Items: items,
	})
}

func (a *API) querySeriesItems(ctx context.Context, seriesID int64, season *int, userID int64) ([]mediaItemDTO, error) {
	query := `SELECT ` + mediaItemColumns + watchedColumn + progressColumn + `
		  FROM media_items
		  LEFT JOIN watch_state ws
		    ON ws.media_item_id = media_items.id AND ws.user_id = ?
		  LEFT JOIN playback_progress pp
		    ON pp.media_item_id = media_items.id AND pp.user_id = ?
		 WHERE media_items.series_id = ?`
	args := []any{userID, userID, seriesID}
	if season != nil {
		query += " AND media_items.season = ?"
		args = append(args, *season)
	}
	query += " ORDER BY media_items.season IS NULL, media_items.season, media_items.episode IS NULL, media_items.episode, media_items.id"

	rows, err := a.db.QueryContext(ctx, query, args...)
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
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	collections, err := loadItemCollections(ctx, a.db, ids)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if found, ok := collections[items[i].ID]; ok {
			items[i].Collections = found
		}
	}
	return items, nil
}

func candidateQuery(r *http.Request) (string, int, error) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	year := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("year")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			return "", 0, errors.New("year debe ser un entero mayor o igual que 0")
		}
		year = value
	}
	return query, year, nil
}

func (a *API) handleSeriesCandidates(w http.ResponseWriter, r *http.Request) {
	record, ok := a.loadSeriesRecord(w, r)
	if !ok {
		return
	}
	if a.tmdb == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "TMDB no está configurado")
		return
	}
	query, year, err := candidateQuery(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if query == "" {
		query = record.FileTitle
		if strings.TrimSpace(query) == "" {
			query = record.Folder
		}
	}
	candidates, err := a.tmdb.SearchTV(r.Context(), query, year)
	if errors.Is(err, metadata.ErrNotFound) {
		candidates = nil
	} else if err != nil {
		log.Printf("library: buscar candidatos de series: %v", err)
		writeJSONError(w, http.StatusBadGateway, "no se pudo consultar TMDB")
		return
	}
	if candidates == nil {
		candidates = []metadata.Candidate{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"candidates": candidates})
}

func (a *API) handleSeriesMatch(w http.ResponseWriter, r *http.Request) {
	record, ok := a.loadSeriesRecord(w, r)
	if !ok {
		return
	}
	if a.tmdb == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "TMDB no está configurado")
		return
	}

	var req matchRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "petición no válida")
		return
	}
	if len(req.TMDBID) == 0 {
		writeJSONError(w, http.StatusBadRequest, "falta tmdb_id")
		return
	}
	if strings.TrimSpace(string(req.TMDBID)) == "null" {
		if err := ClearTVMetadata(r.Context(), a.db, record.ID, "none"); err != nil {
			log.Printf("library: quitar ficha de la serie %d: %v", record.ID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		a.writeSeriesByID(w, r, record.ID)
		return
	}

	var tmdbID int
	if err := decodeJSONValue(req.TMDBID, &tmdbID); err != nil || tmdbID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "tmdb_id debe ser un entero positivo o null")
		return
	}
	details, err := a.tmdb.TVDetails(r.Context(), tmdbID)
	if err != nil {
		log.Printf("library: ficha de serie TMDB %d: %v", tmdbID, err)
		writeJSONError(w, http.StatusBadGateway, "no se pudo consultar TMDB")
		return
	}
	if record.TMDBID != nil && *record.TMDBID != int64(tmdbID) {
		if err := ClearTVMetadata(r.Context(), a.db, record.ID, nil); err != nil {
			log.Printf("library: limpiar ficha anterior de la serie %d: %v", record.ID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
	}
	if err := ApplyTVDetails(r.Context(), a.db, record.ID, details, "manual", 1); err != nil {
		log.Printf("library: guardar ficha de la serie %d: %v", record.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if _, err := FillSeriesSeasons(r.Context(), a.db, a.tmdb, record.ID); err != nil {
		log.Printf("library: completar temporadas de la serie %d: %v", record.ID, err)
		writeJSONError(w, http.StatusBadGateway, "no se pudo consultar TMDB")
		return
	}
	a.writeSeriesByID(w, r, record.ID)
}

func (a *API) handleSeriesMatchDelete(w http.ResponseWriter, r *http.Request) {
	record, ok := a.loadSeriesRecord(w, r)
	if !ok {
		return
	}
	if a.tmdb == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "TMDB no está configurado")
		return
	}

	analyzed := ParseFilename(record.Folder)
	title := analyzed.Title
	if strings.TrimSpace(title) == "" {
		title = record.FileTitle
	}
	if err := ClearTVMetadata(r.Context(), a.db, record.ID, nil); err != nil {
		log.Printf("library: limpiar ficha de la serie %d: %v", record.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	candidate, score, found, err := IdentifySeries(r.Context(), a.tmdb, title, analyzed.Year)
	if err != nil {
		log.Printf("library: reidentificar serie %d: %v", record.ID, err)
		writeJSONError(w, http.StatusBadGateway, "no se pudo consultar TMDB")
		return
	}
	if found {
		details, err := a.tmdb.TVDetails(r.Context(), candidate.TMDBID)
		if err != nil {
			log.Printf("library: ficha de serie TMDB %d: %v", candidate.TMDBID, err)
			writeJSONError(w, http.StatusBadGateway, "no se pudo consultar TMDB")
			return
		}
		if err := ApplyTVDetails(r.Context(), a.db, record.ID, details, "auto", score); err != nil {
			log.Printf("library: guardar ficha de la serie %d: %v", record.ID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		if _, err := FillSeriesSeasons(r.Context(), a.db, a.tmdb, record.ID); err != nil {
			log.Printf("library: completar temporadas de la serie %d: %v", record.ID, err)
			writeJSONError(w, http.StatusBadGateway, "no se pudo consultar TMDB")
			return
		}
	}
	a.writeSeriesByID(w, r, record.ID)
}

func (a *API) writeSeriesByID(w http.ResponseWriter, r *http.Request, id int64) {
	detail, err := querySeriesDetail(r.Context(), a.db, id, sessionUserID(r))
	if err != nil {
		log.Printf("library: leer ficha de la serie %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (a *API) handleSeriesImages(w http.ResponseWriter, r *http.Request) {
	record, ok := a.loadSeriesRecord(w, r)
	if !ok {
		return
	}
	if record.TMDBID == nil {
		writeJSONError(w, http.StatusConflict, "la serie no está identificada")
		return
	}
	if a.tmdb == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "TMDB no está configurado")
		return
	}
	options, err := a.tmdb.TVImages(r.Context(), int(*record.TMDBID))
	if err != nil {
		log.Printf("library: imágenes de la serie TMDB %d: %v", *record.TMDBID, err)
		writeJSONError(w, http.StatusBadGateway, "no se pudo consultar TMDB")
		return
	}
	if options == nil {
		options = &metadata.ImageOptions{}
	}
	if options.Posters == nil {
		options.Posters = []metadata.ImageOption{}
	}
	if options.Backdrops == nil {
		options.Backdrops = []metadata.ImageOption{}
	}
	writeJSON(w, http.StatusOK, options)
}

func (a *API) handleSeriesImagePut(w http.ResponseWriter, r *http.Request) {
	record, ok := a.loadSeriesRecord(w, r)
	if !ok {
		return
	}
	column, lockColumn, ok := imageColumns(r.PathValue("kind"))
	if !ok {
		writeJSONError(w, http.StatusNotFound, "imagen no encontrada")
		return
	}
	var req imageURLRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "petición no válida")
		return
	}
	if !validImageURL(req.URL) {
		writeJSONError(w, http.StatusBadRequest, "la URL debe ser http o https, con host, sin espacios y de 2048 caracteres como máximo")
		return
	}
	if _, err := a.db.ExecContext(r.Context(),
		`UPDATE series SET `+column+` = ?, `+lockColumn+` = 1 WHERE id = ?`,
		req.URL, record.ID); err != nil {
		log.Printf("library: fijar imagen de la serie %d: %v", record.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	a.writeSeriesByID(w, r, record.ID)
}

func (a *API) handleSeriesImageDelete(w http.ResponseWriter, r *http.Request) {
	record, ok := a.loadSeriesRecord(w, r)
	if !ok {
		return
	}
	column, lockColumn, ok := imageColumns(r.PathValue("kind"))
	if !ok {
		writeJSONError(w, http.StatusNotFound, "imagen no encontrada")
		return
	}
	imageURL := ""
	if record.TMDBID != nil {
		if a.tmdb == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "TMDB no está configurado")
			return
		}
		details, err := a.tmdb.TVDetails(r.Context(), int(*record.TMDBID))
		if err != nil || details == nil {
			log.Printf("library: ficha de TMDB %d para restaurar la imagen: %v", *record.TMDBID, err)
			writeJSONError(w, http.StatusBadGateway, "no se pudo consultar TMDB")
			return
		}
		if column == "poster_url" {
			imageURL = details.PosterURL
		} else {
			imageURL = details.BackdropURL
		}
	}
	if _, err := a.db.ExecContext(r.Context(),
		`UPDATE series SET `+column+` = ?, `+lockColumn+` = 0 WHERE id = ?`,
		nullableText(imageURL), record.ID); err != nil {
		log.Printf("library: restaurar imagen de la serie %d: %v", record.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	a.writeSeriesByID(w, r, record.ID)
}

func (a *API) handleSeriesWatchedPut(w http.ResponseWriter, r *http.Request) {
	a.handleSeriesWatched(w, r, true)
}

func (a *API) handleSeriesWatchedDelete(w http.ResponseWriter, r *http.Request) {
	a.handleSeriesWatched(w, r, false)
}

func (a *API) handleSeriesSeasonWatchedPut(w http.ResponseWriter, r *http.Request) {
	a.handleSeriesSeasonWatched(w, r, true)
}

func (a *API) handleSeriesSeasonWatchedDelete(w http.ResponseWriter, r *http.Request) {
	a.handleSeriesSeasonWatched(w, r, false)
}

func (a *API) handleSeriesWatched(w http.ResponseWriter, r *http.Request, watched bool) {
	record, ok := a.loadSeriesRecord(w, r)
	if !ok {
		return
	}
	var exists int
	if err := a.db.QueryRowContext(r.Context(),
		`SELECT 1 FROM media_items WHERE series_id = ? LIMIT 1`, record.ID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "serie no encontrada")
		} else {
			log.Printf("library: comprobar episodios de la serie %d: %v", record.ID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
		}
		return
	}
	a.setSeriesWatched(w, r, seriesEpisodeWhere(record.ID, nil), watched)
}

func (a *API) handleSeriesSeasonWatched(w http.ResponseWriter, r *http.Request, watched bool) {
	record, ok := a.loadSeriesRecord(w, r)
	if !ok {
		return
	}
	seasonNumber, err := strconv.Atoi(strings.TrimSpace(r.PathValue("n")))
	if err != nil || seasonNumber <= 0 {
		writeJSONError(w, http.StatusNotFound, "temporada no encontrada")
		return
	}
	var exists int
	if err := a.db.QueryRowContext(r.Context(),
		`SELECT 1 FROM media_items WHERE series_id = ? AND season = ? LIMIT 1`,
		record.ID, seasonNumber).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "temporada no encontrada")
		} else {
			log.Printf("library: comprobar temporada %d de la serie %d: %v",
				seasonNumber, record.ID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
		}
		return
	}
	a.setSeriesWatched(w, r, seriesEpisodeWhere(record.ID, &seasonNumber), watched)
}

type seriesEpisodeFilter struct {
	query string
	args  []any
}

func seriesEpisodeWhere(seriesID int64, season *int) seriesEpisodeFilter {
	filter := seriesEpisodeFilter{
		query: "series_id = ?",
		args:  []any{seriesID},
	}
	if season != nil {
		filter.query += " AND season = ?"
		filter.args = append(filter.args, *season)
	}
	return filter
}

func (a *API) setSeriesWatched(w http.ResponseWriter, r *http.Request, filter seriesEpisodeFilter, watched bool) {
	userID := sessionUserID(r)
	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("library: iniciar visionado en bloque: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if watched {
		query := `INSERT OR IGNORE INTO watch_state (user_id, media_item_id)
			SELECT ?, id FROM media_items WHERE ` + filter.query
		if _, err := tx.ExecContext(r.Context(), query, append([]any{userID}, filter.args...)...); err != nil {
			log.Printf("library: marcar episodios vistos: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
	} else {
		query := `DELETE FROM watch_state
			WHERE user_id = ? AND media_item_id IN (
				SELECT id FROM media_items WHERE ` + filter.query + `)`
		if _, err := tx.ExecContext(r.Context(), query, append([]any{userID}, filter.args...)...); err != nil {
			log.Printf("library: desmarcar episodios vistos: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
	}
	query := `DELETE FROM playback_progress
		WHERE user_id = ? AND media_item_id IN (
			SELECT id FROM media_items WHERE ` + filter.query + `)`
	if _, err := tx.ExecContext(r.Context(), query, append([]any{userID}, filter.args...)...); err != nil {
		log.Printf("library: borrar progreso de episodios: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("library: confirmar visionado en bloque: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeJSONValue(raw json.RawMessage, dst any) error {
	return json.Unmarshal(raw, dst)
}
