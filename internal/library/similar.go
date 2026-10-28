package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultSimilarLimit = 12
	maxSimilarLimit     = 24
)

type similarProfile struct {
	id          int64
	genres      []string
	directors   []string
	cast        []string
	collections map[int64]bool
	year        int
	rating      float64
	hasRating   bool
	kind        string
}

type scoredSimilar struct {
	profile similarProfile
	score   float64
}

type similarItemsResponse struct {
	Items []mediaItemDTO `json:"items"`
}

type similarSeriesResponse struct {
	Series []seriesListDTO `json:"series"`
}

func similarLimit(r *http.Request) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("limit"))
	if raw == "" {
		return defaultSimilarLimit, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 || value > maxSimilarLimit {
		return 0, fmt.Errorf("limit debe estar entre 1 y %d", maxSimilarLimit)
	}
	return value, nil
}

func normalizedNames(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			result[value] = true
		}
	}
	return result
}

func similarScore(current, candidate similarProfile) (float64, bool) {
	if len(current.genres) == 0 && len(current.directors) == 0 && len(current.cast) == 0 {
		return 0, false
	}

	currentGenres := normalizedNames(current.genres)
	candidateGenres := normalizedNames(candidate.genres)
	union := make(map[string]bool, len(currentGenres)+len(candidateGenres))
	for name := range currentGenres {
		union[name] = true
	}
	for name := range candidateGenres {
		union[name] = true
	}
	sharedGenres := 0
	for name := range currentGenres {
		if candidateGenres[name] {
			sharedGenres++
		}
	}
	jaccard := 0.0
	if len(union) > 0 {
		jaccard = float64(sharedGenres) / float64(len(union))
	}

	currentDirectors := normalizedNames(current.directors)
	sharedDirector := false
	for name := range normalizedNames(candidate.directors) {
		if currentDirectors[name] {
			sharedDirector = true
			break
		}
	}

	currentCast := normalizedNames(current.cast)
	sharedActors := 0
	for name := range normalizedNames(candidate.cast) {
		if currentCast[name] {
			sharedActors++
		}
	}

	sharedCollection := false
	for id := range candidate.collections {
		if current.collections[id] {
			sharedCollection = true
			break
		}
	}

	if sharedGenres == 0 && !sharedDirector && sharedActors == 0 {
		return 0, false
	}

	score := 4 * jaccard
	if sharedDirector {
		score += 3
	}
	if sharedCollection {
		score += 2
	}
	score += 0.4 * float64(minInt(sharedActors, 5))
	if current.year > 0 && candidate.year > 0 && absInt(current.year-candidate.year) <= 5 {
		score += 0.5
	}
	if current.hasRating && candidate.hasRating && math.Abs(current.rating-candidate.rating) <= 1 {
		score += 0.3
	}
	return score, true
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func parseCollectionIDs(raw sql.NullString) map[int64]bool {
	result := make(map[int64]bool)
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return result
	}
	for _, part := range strings.Split(raw.String, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err == nil && id > 0 {
			result[id] = true
		}
	}
	return result
}

func scanSimilarProfile(
	scan func(dest ...any) error,
	withKind bool,
) (similarProfile, error) {
	var profile similarProfile
	var year sql.NullInt64
	var genres, castNames, directors, credits, collections sql.NullString
	var ratingFloat sql.NullFloat64
	var kind sql.NullString
	destinations := []any{
		&profile.id, &genres, &castNames, &directors, &year, &ratingFloat, &credits, &collections,
	}
	if withKind {
		destinations = append(destinations, &kind)
	}
	if err := scan(destinations...); err != nil {
		return similarProfile{}, err
	}
	profile.genres = decodeStringArray(genres)
	profile.directors = decodeStringArray(directors)
	profile.cast = similarCastNames(credits, castNames)
	profile.collections = parseCollectionIDs(collections)
	if year.Valid {
		profile.year = int(year.Int64)
	}
	if ratingFloat.Valid {
		profile.rating = ratingFloat.Float64
		profile.hasRating = true
	}
	profile.kind = kind.String
	return profile, nil
}

func loadMovieSimilarProfile(ctx context.Context, db *sql.DB, id int64, access libraryAccess) (similarProfile, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, genres, cast_names, directors, year, rating, credits,
		       (SELECT group_concat(ci.collection_id)
		          FROM collection_items ci
		          JOIN collections c ON c.id = ci.collection_id
		         WHERE ci.media_item_id = media_items.id
		           AND ci.excluded = 0 AND c.deleted = 0), kind
		  FROM media_items
		 WHERE id = ?
		   AND `+access.Clause("media_items"), id)
	return scanSimilarProfile(row.Scan, true)
}

func loadSeriesSimilarProfile(ctx context.Context, db *sql.DB, id int64, access libraryAccess) (similarProfile, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, genres, cast_names, directors, year, rating, credits, NULL
		  FROM series WHERE id = ? AND `+access.Clause("series"), id)
	return scanSimilarProfile(row.Scan, false)
}

func loadSimilarCandidates(
	ctx context.Context,
	db *sql.DB,
	excludeID int64,
	series bool,
	access libraryAccess,
) ([]similarProfile, error) {
	query := `
		SELECT id, genres, cast_names, directors, year, rating, credits,
		       (SELECT group_concat(ci.collection_id)
		          FROM collection_items ci
		          JOIN collections c ON c.id = ci.collection_id
		         WHERE ci.media_item_id = media_items.id
		           AND ci.excluded = 0 AND c.deleted = 0)
		  FROM media_items
		 WHERE kind = 'movie' AND id <> ?
		   AND ` + access.VideoClause("media_items") + `
		   AND poster_url IS NOT NULL AND trim(poster_url) <> ''`
	if series {
		query = `
			SELECT id, genres, cast_names, directors, year, rating, credits, NULL
			  FROM series
			 WHERE id <> ?
			   AND ` + access.Clause("series") + `
			   AND poster_url IS NOT NULL AND trim(poster_url) <> ''`
	}
	rows, err := db.QueryContext(ctx, query, excludeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := make([]similarProfile, 0)
	for rows.Next() {
		profile, err := scanSimilarProfile(rows.Scan, false)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
}

func rankSimilar(current similarProfile, candidates []similarProfile) []similarProfile {
	ranked := make([]scoredSimilar, 0, len(candidates))
	for _, candidate := range candidates {
		score, ok := similarScore(current, candidate)
		if ok {
			ranked = append(ranked, scoredSimilar{profile: candidate, score: score})
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		left, right := ranked[i].profile, ranked[j].profile
		if left.hasRating != right.hasRating {
			return left.hasRating
		}
		if left.hasRating && right.hasRating && left.rating != right.rating {
			return left.rating > right.rating
		}
		return left.id < right.id
	})
	result := make([]similarProfile, 0, len(ranked))
	for _, candidate := range ranked {
		result = append(result, candidate.profile)
	}
	return result
}

func (a *API) handleSimilarItems(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	limit, err := similarLimit(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	access := accessFromRequest(r)
	current, err := loadMovieSimilarProfile(r.Context(), a.db, id, access)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	case err != nil:
		log.Printf("library: leer ítem %d para similares: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if current.kind != KindMovie {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	var libraryKind string
	if err := a.db.QueryRowContext(r.Context(), `
		SELECT l.kind FROM media_items mi
		  JOIN libraries l ON l.id = mi.library_id
		 WHERE mi.id = ?`, id).Scan(&libraryKind); err != nil {
		log.Printf("library: leer biblioteca del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if libraryKind == "clips" {
		writeJSON(w, http.StatusOK, similarItemsResponse{Items: make([]mediaItemDTO, 0)})
		return
	}

	response := similarItemsResponse{Items: make([]mediaItemDTO, 0)}
	if len(current.genres) == 0 && len(current.directors) == 0 && len(current.cast) == 0 {
		writeJSON(w, http.StatusOK, response)
		return
	}

	candidates, err := loadSimilarCandidates(r.Context(), a.db, id, false, access)
	if err != nil {
		log.Printf("library: leer similares del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	ranked := rankSimilar(current, candidates)
	ids := make([]int64, 0, minInt(limit, len(ranked)))
	for _, candidate := range ranked {
		if len(ids) == limit {
			break
		}
		ids = append(ids, candidate.id)
	}

	items, err := a.mediaItemsByIDs(r.Context(), ids, sessionUserID(r), access)
	if err != nil {
		log.Printf("library: leer fichas de similares del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	compactMediaItemListSlice(items, isAdminRequest(r))
	response.Items = items
	writeJSON(w, http.StatusOK, response)
}

func (a *API) handleSimilarSeries(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "serie no encontrada")
		return
	}
	limit, err := similarLimit(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	access := accessFromRequest(r)
	current, err := loadSeriesSimilarProfile(r.Context(), a.db, id, access)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "serie no encontrada")
		return
	case err != nil:
		log.Printf("library: leer serie %d para similares: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	response := similarSeriesResponse{Series: make([]seriesListDTO, 0)}
	if len(current.genres) == 0 && len(current.directors) == 0 && len(current.cast) == 0 {
		writeJSON(w, http.StatusOK, response)
		return
	}

	candidates, err := loadSimilarCandidates(r.Context(), a.db, id, true, access)
	if err != nil {
		log.Printf("library: leer similares de la serie %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	ranked := rankSimilar(current, candidates)
	ids := make([]int64, 0, minInt(limit, len(ranked)))
	for _, candidate := range ranked {
		if len(ids) == limit {
			break
		}
		ids = append(ids, candidate.id)
	}

	series, err := a.seriesListByIDs(r.Context(), ids, sessionUserID(r), access)
	if err != nil {
		log.Printf("library: leer resúmenes de similares de la serie %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	response.Series = series
	writeJSON(w, http.StatusOK, response)
}

func (a *API) mediaItemsByIDs(
	ctx context.Context,
	ids []int64,
	userID int64,
	access libraryAccess,
) ([]mediaItemDTO, error) {
	if len(ids) == 0 {
		return []mediaItemDTO{}, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids)+4)
	args = append(args, userID, userID, userID, userID)
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := a.db.QueryContext(ctx, `
		SELECT `+mediaItemColumns+watchedColumn+progressColumn+itemListColumns+`
		  FROM media_items
		  LEFT JOIN watch_state ws
		    ON ws.media_item_id = media_items.id AND ws.user_id = ?
		  LEFT JOIN playback_progress pp
		    ON pp.media_item_id = media_items.id AND pp.user_id = ?
		 WHERE media_items.id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := make(map[int64]mediaItemDTO, len(ids))
	for rows.Next() {
		item, err := scanItem(rows.Scan)
		if err != nil {
			return nil, err
		}
		byID[item.ID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	byItem, err := loadItemCollectionsWithAccess(ctx, a.db, ids, access, a.collectionMin())
	if err != nil {
		return nil, err
	}
	result := make([]mediaItemDTO, 0, len(ids))
	for _, id := range ids {
		item, ok := byID[id]
		if !ok {
			continue
		}
		if collections, found := byItem[id]; found {
			item.Collections = collections
		}
		result = append(result, item)
	}
	return result, nil
}

func (a *API) seriesListByIDs(
	ctx context.Context,
	ids []int64,
	userID int64,
	access libraryAccess,
) ([]seriesListDTO, error) {
	if len(ids) == 0 {
		return []seriesListDTO{}, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids)+3)
	args = append(args, userID, userID, userID)
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := a.db.QueryContext(ctx, `
		SELECT s.id, COALESCE(NULLIF(s.display_title, ''), s.title), s.title,
		       s.year, s.poster_url, s.genres,
		       (SELECT count(DISTINCT mi.season) FROM media_items mi
		         WHERE mi.series_id = s.id AND mi.season IS NOT NULL
		           AND `+access.Clause("mi")+`),
		       (SELECT count(*) FROM media_items mi
		         WHERE mi.series_id = s.id AND `+access.Clause("mi")+`),
		       (SELECT count(*) FROM media_items mi
		          JOIN watch_state ws ON ws.media_item_id = mi.id AND ws.user_id = ?
		         WHERE mi.series_id = s.id AND `+access.Clause("mi")+`),
		       s.rating, s.trailer_url, s.trailer_language,
		       s.match_status,
		       EXISTS (SELECT 1 FROM series_lists sl_favorite
		                WHERE sl_favorite.user_id = ?
		                  AND sl_favorite.series_id = s.id
		                  AND sl_favorite.list = 'favorite'),
		       EXISTS (SELECT 1 FROM series_lists sl_watchlist
		                WHERE sl_watchlist.user_id = ?
		                  AND sl_watchlist.series_id = s.id
		                  AND sl_watchlist.list = 'watchlist')
		  FROM series s
		 WHERE s.id IN (`+placeholders+`) AND `+access.Clause("s"), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := make(map[int64]seriesListDTO, len(ids))
	for rows.Next() {
		item, err := scanSeriesList(rows.Scan)
		if err != nil {
			return nil, err
		}
		byID[item.ID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]seriesListDTO, 0, len(ids))
	for _, id := range ids {
		if item, ok := byID[id]; ok {
			result = append(result, item)
		}
	}
	return result, nil
}
