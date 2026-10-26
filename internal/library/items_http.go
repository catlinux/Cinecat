package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/catlinux/cinecat/internal/metadata"
)

type mediaItemDTO struct {
	ID               int64             `json:"id"`
	LibraryID        int64             `json:"library_id"`
	Path             string            `json:"path"`
	Title            string            `json:"title"`
	FileTitle        string            `json:"file_title"`
	Year             *int              `json:"year"`
	Kind             string            `json:"kind"`
	Season           *int              `json:"season"`
	Episode          *int              `json:"episode"`
	TMDBID           *int64            `json:"tmdb_id"`
	Overview         *string           `json:"overview"`
	PosterURL        *string           `json:"poster_url"`
	PosterCustom     bool              `json:"poster_custom"`
	OriginalTitle    *string           `json:"original_title"`
	IMDbID           *string           `json:"imdb_id"`
	Tagline          *string           `json:"tagline"`
	Genres           []string          `json:"genres"`
	RuntimeMinutes   *int              `json:"runtime_minutes"`
	Rating           *float64          `json:"rating"`
	Directors        []string          `json:"directors"`
	Cast             []string          `json:"cast"`
	BackdropURL      *string           `json:"backdrop_url"`
	BackdropCustom   bool              `json:"backdrop_custom"`
	OverviewLanguage *string           `json:"overview_language"`
	MatchStatus      *string           `json:"match_status"`
	MatchScore       *float64          `json:"match_score"`
	Watched          bool              `json:"watched"`
	Progress         *playbackProgress `json:"progress"`
	Collections      []itemCollection  `json:"collections"`
	SeriesID         *int64            `json:"series_id"`
	SeriesTitle      *string           `json:"series_title"`
	EpisodeTitle     *string           `json:"episode_title"`
	StillURL         *string           `json:"still_url"`
}

type itemCollection struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type itemsResponse struct {
	Items  []mediaItemDTO `json:"items"`
	Total  int64          `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

type genreCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type genresResponse struct {
	Genres []genreCount `json:"genres"`
}

// mediaItemColumns la comparten la lista y la ficha para no divergir.
const mediaItemColumns = `
	id, library_id, path, title, COALESCE(episode_title, display_title, title), year,
	kind, season, episode, tmdb_id, overview, poster_url, poster_locked,
	original_title, imdb_id, tagline, genres, runtime_minutes, rating,
	directors, cast_names, backdrop_url, backdrop_locked, overview_language,
	match_status, match_score, series_id,
	(SELECT COALESCE(NULLIF(s.display_title, ''), s.title)
	   FROM series s WHERE s.id = media_items.series_id),
	episode_title, still_url`

const watchedColumn = `, CASE WHEN ws.user_id IS NULL THEN 0 ELSE 1 END`

const progressColumn = `, pp.position_seconds, pp.duration_seconds, pp.updated_at`

func queryMediaItem(
	ctx context.Context,
	db *sql.DB,
	id, userID int64,
	minCollectionItems ...int,
) (mediaItemDTO, error) {
	minimum := collectionMinimum(minCollectionItems...)
	row := db.QueryRowContext(ctx, `
		SELECT `+mediaItemColumns+watchedColumn+progressColumn+`
		  FROM media_items
		  LEFT JOIN watch_state ws
		    ON ws.media_item_id = media_items.id AND ws.user_id = ?
		  LEFT JOIN playback_progress pp
		    ON pp.media_item_id = media_items.id AND pp.user_id = ?
		 WHERE media_items.id = ?`, userID, userID, id)
	item, err := scanItem(row.Scan)
	if err != nil {
		return mediaItemDTO{}, err
	}
	byItem, err := loadItemCollections(ctx, db, []int64{item.ID}, minimum)
	if err != nil {
		return mediaItemDTO{}, err
	}
	if collections, ok := byItem[item.ID]; ok {
		item.Collections = collections
	}
	return item, nil
}

// loadItemCollections resuelve en una sola consulta las colecciones visibles
// de todos los ítems pedidos. Las automáticas necesitan el mínimo configurado
// de miembros.
func loadItemCollections(
	ctx context.Context,
	db *sql.DB,
	itemIDs []int64,
	minItems ...int,
) (map[int64][]itemCollection, error) {
	minimum := collectionMinimum(minItems...)
	result := make(map[int64][]itemCollection, len(itemIDs))
	if len(itemIDs) == 0 {
		return result, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(itemIDs)), ",")
	args := make([]any, 0, len(itemIDs)+1)
	for _, id := range itemIDs {
		args = append(args, id)
	}
	args = append(args, minimum)
	rows, err := db.QueryContext(ctx, `
		SELECT ci.media_item_id, c.id, c.name
		  FROM collection_items ci
		  JOIN collections c ON c.id = ci.collection_id
		 WHERE ci.media_item_id IN (`+placeholders+`)
		   AND ci.excluded = 0
		   AND c.deleted = 0
		   AND (c.source = 'manual' OR (
		         SELECT count(*) FROM collection_items cc
		          WHERE cc.collection_id = c.id AND cc.excluded = 0
		       ) >= ?)
		 ORDER BY c.name COLLATE NOCASE, c.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var itemID int64
		var collection itemCollection
		if err := rows.Scan(&itemID, &collection.ID, &collection.Name); err != nil {
			return nil, err
		}
		result[itemID] = append(result[itemID], collection)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func scanItem(scan func(dest ...any) error) (mediaItemDTO, error) {
	item := mediaItemDTO{Collections: make([]itemCollection, 0)}
	var year, season, episode, tmdbID, runtime, seriesID sql.NullInt64
	var overview, posterURL, originalTitle, imdbID, tagline sql.NullString
	var genres, directors, castNames, backdropURL, overviewLanguage, matchStatus sql.NullString
	var seriesTitle, episodeTitle, stillURL sql.NullString
	var rating, matchScore, progressPosition, progressDuration sql.NullFloat64
	var progressUpdated sql.NullString
	var posterLocked, backdropLocked, watched int

	err := scan(
		&item.ID, &item.LibraryID, &item.Path, &item.FileTitle, &item.Title,
		&year, &item.Kind, &season, &episode, &tmdbID, &overview,
		&posterURL, &posterLocked, &originalTitle, &imdbID, &tagline,
		&genres, &runtime, &rating, &directors, &castNames, &backdropURL,
		&backdropLocked, &overviewLanguage, &matchStatus, &matchScore,
		&seriesID, &seriesTitle, &episodeTitle, &stillURL, &watched,
		&progressPosition, &progressDuration, &progressUpdated,
	)
	if err != nil {
		return mediaItemDTO{}, err
	}

	item.Year = intPointer(year)
	item.Season = intPointer(season)
	item.Episode = intPointer(episode)
	item.TMDBID = int64Pointer(tmdbID)
	item.Overview = stringPointer(overview)
	item.PosterURL = stringPointer(posterURL)
	item.PosterCustom = posterLocked != 0
	item.OriginalTitle = stringPointer(originalTitle)
	item.IMDbID = stringPointer(imdbID)
	item.Tagline = stringPointer(tagline)
	item.Genres = decodeStringArray(genres)
	item.RuntimeMinutes = intPointer(runtime)
	item.Rating = floatPointer(rating)
	item.Directors = decodeStringArray(directors)
	item.Cast = decodeStringArray(castNames)
	item.BackdropURL = stringPointer(backdropURL)
	item.BackdropCustom = backdropLocked != 0
	item.OverviewLanguage = stringPointer(overviewLanguage)
	item.MatchStatus = stringPointer(matchStatus)
	item.MatchScore = floatPointer(matchScore)
	item.SeriesID = int64Pointer(seriesID)
	item.SeriesTitle = stringPointer(seriesTitle)
	item.EpisodeTitle = stringPointer(episodeTitle)
	item.StillURL = stringPointer(stillURL)
	item.Watched = watched != 0
	if progressPosition.Valid && progressDuration.Valid && progressUpdated.Valid {
		item.Progress = &playbackProgress{
			PositionSeconds: progressPosition.Float64,
			DurationSeconds: progressDuration.Float64,
			UpdatedAt:       progressUpdated.String,
		}
	}
	return item, nil
}

func decodeStringArray(raw sql.NullString) []string {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return []string{}
	}
	var values []string
	if err := json.Unmarshal([]byte(raw.String), &values); err != nil || values == nil {
		return []string{}
	}
	return values
}

func stringPointer(v sql.NullString) *string {
	if !v.Valid || strings.TrimSpace(v.String) == "" {
		return nil
	}
	value := v.String
	return &value
}

func intPointer(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	value := int(v.Int64)
	return &value
}

func int64Pointer(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	value := v.Int64
	return &value
}

func floatPointer(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	value := v.Float64
	return &value
}

func itemFilters(r *http.Request, userID int64) (string, []any, error) {
	query := r.URL.Query()
	var clauses []string
	var args []any

	field := strings.TrimSpace(query.Get("field"))
	if field == "" {
		field = "all"
	}
	switch field {
	case "all", "title", "cast", "director":
	default:
		return "", nil, errors.New("field debe ser all, title, cast o director")
	}

	if raw := strings.TrimSpace(query.Get("library_id")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return "", nil, errors.New("library_id debe ser un entero positivo")
		}
		clauses = append(clauses, "library_id = ?")
		args = append(args, id)
	}

	if raw := strings.TrimSpace(query.Get("kind")); raw != "" {
		if raw != KindMovie && raw != KindEpisode {
			return "", nil, errors.New("kind debe ser movie o episode")
		}
		clauses = append(clauses, "kind = ?")
		args = append(args, raw)
	}

	if raw := strings.TrimSpace(query.Get("q")); raw != "" {
		pattern := likePattern(raw)
		var columns []string
		switch field {
		case "title":
			columns = []string{"display_title", "title", "original_title"}
		case "cast":
			columns = []string{"cast_names"}
		case "director":
			columns = []string{"directors"}
		case "all":
			columns = []string{"display_title", "title", "original_title", "cast_names", "directors", "tagline"}
		}
		parts := make([]string, 0, len(columns))
		for _, column := range columns {
			parts = append(parts, "LOWER("+column+`) LIKE LOWER(?) ESCAPE '\'`)
			args = append(args, pattern)
		}
		clauses = append(clauses, "("+strings.Join(parts, " OR ")+")")
	}

	if raw := strings.TrimSpace(query.Get("letter")); raw != "" {
		letter, err := parseLetter(raw)
		if err != nil {
			return "", nil, err
		}
		if letter != "" {
			clauses = append(clauses, "sort_letter = ?")
			args = append(args, letter)
		}
	}

	if raw := strings.TrimSpace(query.Get("genre")); raw != "" {
		clauses = append(clauses,
			`EXISTS (SELECT 1 FROM json_each(CASE WHEN genres IS NOT NULL AND json_valid(genres) THEN genres ELSE '[]' END) je WHERE LOWER(je.value) = LOWER(?))`)
		args = append(args, raw)
	}

	if raw := strings.TrimSpace(query.Get("watched")); raw != "" {
		switch raw {
		case "true":
			clauses = append(clauses,
				`EXISTS (SELECT 1 FROM watch_state wf WHERE wf.media_item_id = media_items.id AND wf.user_id = ?)`)
			args = append(args, userID)
		case "false":
			clauses = append(clauses,
				`NOT EXISTS (SELECT 1 FROM watch_state wf WHERE wf.media_item_id = media_items.id AND wf.user_id = ?)`)
			args = append(args, userID)
		default:
			return "", nil, errors.New("watched debe ser true o false")
		}
	}

	if raw := strings.TrimSpace(query.Get("series_id")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return "", nil, errors.New("series_id debe ser un entero positivo")
		}
		clauses = append(clauses, "series_id = ?")
		args = append(args, id)
	}

	if raw := strings.TrimSpace(query.Get("collection_id")); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return "", nil, errors.New("collection_id debe ser un entero positivo")
		}
		clauses = append(clauses,
			`EXISTS (SELECT 1 FROM collection_items cf WHERE cf.media_item_id = media_items.id AND cf.collection_id = ? AND cf.excluded = 0)`)
		args = append(args, id)
	}

	if raw := strings.TrimSpace(query.Get("in_progress")); raw != "" {
		switch raw {
		case "true":
			clauses = append(clauses,
				`EXISTS (SELECT 1 FROM playback_progress pf WHERE pf.media_item_id = media_items.id AND pf.user_id = ?)`)
			args = append(args, userID)
		case "false":
			clauses = append(clauses,
				`NOT EXISTS (SELECT 1 FROM playback_progress pf WHERE pf.media_item_id = media_items.id AND pf.user_id = ?)`)
			args = append(args, userID)
		default:
			return "", nil, errors.New("in_progress debe ser true o false")
		}
	}

	if raw := strings.TrimSpace(query.Get("unmatched")); raw != "" {
		switch raw {
		case "true":
			clauses = append(clauses, "(kind = 'movie' AND (match_status IS NULL OR match_status = 'none'))")
		case "false":
		default:
			return "", nil, errors.New("unmatched debe ser true o false")
		}
	}

	if len(clauses) == 0 {
		return "", args, nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args, nil
}

func itemOrder(r *http.Request) (string, error) {
	switch strings.TrimSpace(r.URL.Query().Get("sort")) {
	case "":
		return " ORDER BY id", nil
	case "title":
		return " ORDER BY sort_title COLLATE NOCASE, id", nil
	case "year":
		return " ORDER BY (year IS NULL) ASC, year DESC, sort_title COLLATE NOCASE, id", nil
	case "added":
		return " ORDER BY id DESC", nil
	default:
		return "", errors.New("sort debe ser title, year o added")
	}
}

func likePattern(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + replacer.Replace(value) + "%"
}

func (a *API) handleGenres(w http.ResponseWriter, r *http.Request) {
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	if kind == "" {
		kind = KindMovie
	}
	if kind != KindMovie && kind != KindEpisode {
		writeJSONError(w, http.StatusBadRequest, "kind debe ser movie o episode")
		return
	}

	rows, err := a.db.QueryContext(r.Context(), `
		SELECT je.value, COUNT(DISTINCT mi.id)
		  FROM media_items mi
		  JOIN json_each(CASE WHEN mi.genres IS NOT NULL AND json_valid(mi.genres) THEN mi.genres ELSE '[]' END) je
		 WHERE mi.kind = ?
		 GROUP BY je.value
		 ORDER BY je.value COLLATE NOCASE, je.value`, kind)
	if err != nil {
		log.Printf("library: listar géneros: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer rows.Close()

	genres := make([]genreCount, 0)
	for rows.Next() {
		var name sql.NullString
		var count int64
		if err := rows.Scan(&name, &count); err != nil {
			log.Printf("library: leer género: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		if !name.Valid || strings.TrimSpace(name.String) == "" {
			continue
		}
		genres = append(genres, genreCount{Name: name.String, Count: count})
	}
	if err := rows.Err(); err != nil {
		log.Printf("library: recorrer géneros: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, genresResponse{Genres: genres})
}

func (a *API) loadMovieItem(w http.ResponseWriter, r *http.Request) (mediaItemDTO, bool) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return mediaItemDTO{}, false
	}

	item, err := queryMediaItem(r.Context(), a.db, id, sessionUserID(r), a.collectionMin())
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return mediaItemDTO{}, false
	case err != nil:
		log.Printf("library: leer ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return mediaItemDTO{}, false
	case item.Kind != KindMovie:
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return mediaItemDTO{}, false
	}
	return item, true
}

func (a *API) handleItemCandidates(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadMovieItem(w, r)
	if !ok {
		return
	}
	if a.tmdb == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "TMDB no está configurado")
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		query = item.FileTitle
	}
	year := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("year")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			writeJSONError(w, http.StatusBadRequest, "year debe ser un entero mayor o igual que 0")
			return
		}
		year = value
	}

	candidates, err := a.tmdb.SearchMovies(r.Context(), query, year)
	if errors.Is(err, metadata.ErrNotFound) {
		candidates = nil
	} else if err != nil {
		log.Printf("library: buscar candidatos de TMDB: %v", err)
		writeJSONError(w, http.StatusBadGateway, "no se pudo consultar TMDB")
		return
	}
	if candidates == nil {
		candidates = []metadata.Candidate{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"candidates": candidates})
}

type matchRequest struct {
	TMDBID json.RawMessage `json:"tmdb_id"`
}

func (a *API) handleItemMatch(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadMovieItem(w, r)
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
		if err := ClearMovieMetadata(r.Context(), a.db, item.ID, "none"); err != nil {
			log.Printf("library: quitar ficha del ítem %d: %v", item.ID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		a.writeItemByID(w, r, item.ID)
		return
	}

	var tmdbID int
	if err := json.Unmarshal(req.TMDBID, &tmdbID); err != nil || tmdbID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "tmdb_id debe ser un entero positivo o null")
		return
	}

	details, err := a.tmdb.MovieDetails(r.Context(), tmdbID)
	if err != nil {
		log.Printf("library: ficha de TMDB %d: %v", tmdbID, err)
		writeJSONError(w, http.StatusBadGateway, "no se pudo consultar TMDB")
		return
	}
	if err := ApplyMovieDetails(r.Context(), a.db, item.ID, details, "manual", 1); err != nil {
		log.Printf("library: guardar ficha del ítem %d: %v", item.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	a.writeItemByID(w, r, item.ID)
}

func (a *API) handleItemMatchDelete(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadMovieItem(w, r)
	if !ok {
		return
	}
	if a.tmdb == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "TMDB no está configurado")
		return
	}

	media, err := a.mediaFile(r, item.ID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	case err != nil:
		log.Printf("library: leer archivo del ítem %d: %v", item.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	libraryRoot, rootErr := filepath.Abs(media.libraryPath)
	mediaPath, pathErr := filepath.Abs(media.path)
	relative, err := filepath.Rel(libraryRoot, mediaPath)
	if rootErr != nil || pathErr != nil || err != nil || relative == "." || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		relative = filepath.Base(media.path)
	}

	analyzed := ParsePath(relative)
	title := analyzed.Title
	if title == "" {
		title = item.FileTitle
	}

	if err := ClearMovieMetadata(r.Context(), a.db, item.ID, nil); err != nil {
		log.Printf("library: limpiar ficha del ítem %d: %v", item.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if err := updateMediaYear(r.Context(), a.db, item.ID, analyzed.Year); err != nil {
		log.Printf("library: restaurar año del ítem %d: %v", item.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	candidate, score, found, err := IdentifyMovie(r.Context(), a.tmdb, title, analyzed.Year)
	if err != nil {
		log.Printf("library: reidentificar ítem %d: %v", item.ID, err)
		writeJSONError(w, http.StatusBadGateway, "no se pudo consultar TMDB")
		return
	}
	if found {
		details, err := a.tmdb.MovieDetails(r.Context(), candidate.TMDBID)
		if err != nil {
			log.Printf("library: ficha de TMDB %d: %v", candidate.TMDBID, err)
			writeJSONError(w, http.StatusBadGateway, "no se pudo consultar TMDB")
			return
		}
		if err := ApplyMovieDetails(r.Context(), a.db, item.ID, details, "auto", score); err != nil {
			log.Printf("library: guardar ficha del ítem %d: %v", item.ID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
	}
	a.writeItemByID(w, r, item.ID)
}

func updateMediaYear(ctx context.Context, db *sql.DB, id int64, year int) error {
	var value any
	if year > 0 {
		value = year
	}
	_, err := db.ExecContext(ctx, `UPDATE media_items SET year = ? WHERE id = ?`, value, id)
	if err != nil {
		return fmt.Errorf("library: actualizar año del ítem %d: %w", id, err)
	}
	return nil
}

func (a *API) writeItemByID(w http.ResponseWriter, r *http.Request, id int64) {
	item, err := queryMediaItem(r.Context(), a.db, id, sessionUserID(r), a.collectionMin())
	if err != nil {
		log.Printf("library: leer ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, item)
}
