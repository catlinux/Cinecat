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

type mediaItemDTO struct {
	ID               int64    `json:"id"`
	LibraryID        int64    `json:"library_id"`
	Path             string   `json:"path"`
	Title            string   `json:"title"`
	FileTitle        string   `json:"file_title"`
	Year             *int     `json:"year"`
	Kind             string   `json:"kind"`
	Season           *int     `json:"season"`
	Episode          *int     `json:"episode"`
	TMDBID           *int64   `json:"tmdb_id"`
	Overview         *string  `json:"overview"`
	PosterURL        *string  `json:"poster_url"`
	OriginalTitle    *string  `json:"original_title"`
	IMDbID           *string  `json:"imdb_id"`
	Tagline          *string  `json:"tagline"`
	Genres           []string `json:"genres"`
	RuntimeMinutes   *int     `json:"runtime_minutes"`
	Rating           *float64 `json:"rating"`
	Directors        []string `json:"directors"`
	Cast             []string `json:"cast"`
	BackdropURL      *string  `json:"backdrop_url"`
	OverviewLanguage *string  `json:"overview_language"`
	MatchStatus      *string  `json:"match_status"`
	MatchScore       *float64 `json:"match_score"`
}

type itemsResponse struct {
	Items  []mediaItemDTO `json:"items"`
	Total  int64          `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

// mediaItemColumns la comparten la lista y la ficha para no divergir.
const mediaItemColumns = `
	id, library_id, path, title, COALESCE(display_title, title), year,
	kind, season, episode, tmdb_id, overview, poster_url, original_title,
	imdb_id, tagline, genres, runtime_minutes, rating, directors, cast_names,
	backdrop_url, overview_language, match_status, match_score`

func queryMediaItem(ctx context.Context, db *sql.DB, id int64) (mediaItemDTO, error) {
	row := db.QueryRowContext(ctx, `SELECT `+mediaItemColumns+` FROM media_items WHERE id = ?`, id)
	return scanItem(row.Scan)
}

func scanItem(scan func(dest ...any) error) (mediaItemDTO, error) {
	var item mediaItemDTO
	var year, season, episode, tmdbID, runtime sql.NullInt64
	var overview, posterURL, originalTitle, imdbID, tagline sql.NullString
	var genres, directors, castNames, backdropURL, overviewLanguage, matchStatus sql.NullString
	var rating, matchScore sql.NullFloat64

	err := scan(
		&item.ID, &item.LibraryID, &item.Path, &item.FileTitle, &item.Title,
		&year, &item.Kind, &season, &episode, &tmdbID, &overview,
		&posterURL, &originalTitle, &imdbID, &tagline, &genres, &runtime,
		&rating, &directors, &castNames, &backdropURL, &overviewLanguage,
		&matchStatus, &matchScore,
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
	item.OriginalTitle = stringPointer(originalTitle)
	item.IMDbID = stringPointer(imdbID)
	item.Tagline = stringPointer(tagline)
	item.Genres = decodeStringArray(genres)
	item.RuntimeMinutes = intPointer(runtime)
	item.Rating = floatPointer(rating)
	item.Directors = decodeStringArray(directors)
	item.Cast = decodeStringArray(castNames)
	item.BackdropURL = stringPointer(backdropURL)
	item.OverviewLanguage = stringPointer(overviewLanguage)
	item.MatchStatus = stringPointer(matchStatus)
	item.MatchScore = floatPointer(matchScore)
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

func itemFilters(r *http.Request) (string, []any, error) {
	query := r.URL.Query()
	var clauses []string
	var args []any

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
		clauses = append(clauses,
			`(LOWER(display_title) LIKE LOWER(?) ESCAPE '\' OR LOWER(title) LIKE LOWER(?) ESCAPE '\' OR LOWER(original_title) LIKE LOWER(?) ESCAPE '\')`)
		args = append(args, pattern, pattern, pattern)
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
		return " ORDER BY COALESCE(display_title, title) COLLATE NOCASE, id", nil
	case "year":
		return " ORDER BY (year IS NULL) ASC, year DESC, COALESCE(display_title, title) COLLATE NOCASE, id", nil
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

func (a *API) loadMovieItem(w http.ResponseWriter, r *http.Request) (mediaItemDTO, bool) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return mediaItemDTO{}, false
	}

	item, err := queryMediaItem(r.Context(), a.db, id)
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

	analyzed := ParsePath(item.Path)
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
	item, err := queryMediaItem(r.Context(), a.db, id)
	if err != nil {
		log.Printf("library: leer ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, item)
}
