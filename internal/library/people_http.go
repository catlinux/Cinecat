package library

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/catlinux/cinecat/internal/metadata"
)

type personDTO struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Biography    string `json:"biography"`
	Birthday     string `json:"birthday"`
	Deathday     string `json:"deathday"`
	PlaceOfBirth string `json:"place_of_birth"`
	PhotoURL     string `json:"photo_url"`
	KnownFor     string `json:"known_for"`
}

type personMediaDTO struct {
	Type      string   `json:"type"`
	ID        int64    `json:"id,omitempty"`
	TMDBID    int64    `json:"tmdb_id,omitempty"`
	Title     string   `json:"title"`
	Year      *int     `json:"year"`
	PosterURL *string  `json:"poster_url"`
	Rating    *float64 `json:"rating"`
	Roles     []string `json:"roles"`
}

type personMediaResponse struct {
	Person       personDTO        `json:"person"`
	InLibrary    []personMediaDTO `json:"in_library"`
	NotInLibrary []personMediaDTO `json:"not_in_library"`
}

type localPersonMedia struct {
	ID      int64
	Title   string
	Year    *int
	Poster  *string
	Rating  *float64
	TMDBID  *int64
	Credits creditsDTO
}

type personDetailsClient interface {
	PersonDetails(ctx context.Context, id int) (*metadata.Person, error)
}

func (a *API) handlePerson(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(strings.TrimSpace(r.PathValue("id")))
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusNotFound, "persona no encontrada")
		return
	}
	client, ok := a.tmdb.(personDetailsClient)
	if !ok || a.tmdb == nil {
		writeJSONError(w, http.StatusNotFound, "persona no encontrada")
		return
	}
	person, err := client.PersonDetails(r.Context(), id)
	if errors.Is(err, metadata.ErrNotFound) {
		writeJSONError(w, http.StatusNotFound, "persona no encontrada")
		return
	}
	if err != nil {
		log.Printf("library: ficha de persona TMDB %d: %v", id, err)
		writeJSONError(w, http.StatusBadGateway, "no se pudo consultar TMDB")
		return
	}
	if person == nil || person.ID <= 0 {
		writeJSONError(w, http.StatusNotFound, "persona no encontrada")
		return
	}

	access := accessFromRequest(r)
	movies, err := a.localPersonMovies(r.Context(), access)
	if err != nil {
		log.Printf("library: personas: películas locales: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	series, err := a.localPersonSeries(r.Context(), access)
	if err != nil {
		log.Printf("library: personas: series locales: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	combinedRoles := make(map[string][]string)
	for _, credit := range person.Credits {
		key := credit.MediaType + "\x00" + strconv.Itoa(credit.TMDBID)
		combinedRoles[key] = appendRole(combinedRoles[key], credit.Character, credit.Job)
	}

	inLibrary := make([]personMediaDTO, 0)
	localMovieTMDB := make(map[int64]bool)
	localSeriesTMDB := make(map[int64]bool)
	for _, item := range movies {
		if item.TMDBID != nil {
			localMovieTMDB[*item.TMDBID] = true
		}
		roles, matched := rolesForPerson(item.Credits, id)
		if item.TMDBID != nil {
			roles = appendRoles(roles, combinedRoles["movie\x00"+strconv.FormatInt(*item.TMDBID, 10)]...)
		}
		if matched || len(roles) > 0 {
			inLibrary = append(inLibrary, personMedia(item, "movie", roles))
		}
	}
	for _, item := range series {
		if item.TMDBID != nil {
			localSeriesTMDB[*item.TMDBID] = true
		}
		roles, matched := rolesForPerson(item.Credits, id)
		if item.TMDBID != nil {
			roles = appendRoles(roles, combinedRoles["tv\x00"+strconv.FormatInt(*item.TMDBID, 10)]...)
		}
		if matched || len(roles) > 0 {
			inLibrary = append(inLibrary, personMedia(item, "series", roles))
		}
	}
	sort.SliceStable(inLibrary, func(i, j int) bool {
		if inLibrary[i].Type != inLibrary[j].Type {
			return inLibrary[i].Type < inLibrary[j].Type
		}
		return strings.ToLower(inLibrary[i].Title) < strings.ToLower(inLibrary[j].Title)
	})

	notInLibrary := make([]personMediaDTO, 0)
	seen := make(map[string]int)
	for _, credit := range person.Credits {
		key := credit.MediaType + "\x00" + strconv.Itoa(credit.TMDBID)
		if credit.MediaType == "movie" && localMovieTMDB[int64(credit.TMDBID)] {
			continue
		}
		if credit.MediaType == "tv" && localSeriesTMDB[int64(credit.TMDBID)] {
			continue
		}
		kind := "movie"
		if credit.MediaType == "tv" {
			kind = "series"
		}
		roles := []string{}
		if value := credit.Character; strings.TrimSpace(value) != "" {
			roles = append(roles, strings.TrimSpace(value))
		}
		if value := credit.Job; strings.TrimSpace(value) != "" {
			roles = append(roles, strings.TrimSpace(value))
		}
		if index, ok := seen[key]; ok {
			notInLibrary[index].Roles = appendRoles(notInLibrary[index].Roles, roles...)
			continue
		}
		seen[key] = len(notInLibrary)
		notInLibrary = append(notInLibrary, personMediaDTO{
			Type:      kind,
			TMDBID:    int64(credit.TMDBID),
			Title:     credit.Title,
			Year:      intValuePointer(credit.Year),
			PosterURL: textPointer(credit.PosterURL),
			Roles:     roles,
		})
	}
	sort.SliceStable(notInLibrary, func(i, j int) bool {
		first, second := notInLibrary[i].Year, notInLibrary[j].Year
		if first == nil {
			return false
		}
		if second == nil {
			return true
		}
		if *first != *second {
			return *first > *second
		}
		return strings.ToLower(notInLibrary[i].Title) < strings.ToLower(notInLibrary[j].Title)
	})

	writeJSON(w, http.StatusOK, personMediaResponse{
		Person: personDTO{
			ID:           person.ID,
			Name:         person.Name,
			Biography:    person.Biography,
			Birthday:     person.Birthday,
			Deathday:     person.Deathday,
			PlaceOfBirth: person.PlaceOfBirth,
			PhotoURL:     person.PhotoURL,
			KnownFor:     person.KnownFor,
		},
		InLibrary:    inLibrary,
		NotInLibrary: notInLibrary,
	})
}

func (a *API) localPersonMovies(ctx context.Context, access libraryAccess) ([]localPersonMedia, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT mi.id, COALESCE(NULLIF(mi.display_title, ''), mi.title), mi.year,
		       mi.poster_url, mi.rating, mi.tmdb_id, mi.credits
		  FROM media_items mi
		  JOIN libraries l ON l.id = mi.library_id AND l.kind = 'video'
		 WHERE mi.kind = 'movie' AND `+access.Clause("mi")+`
		 ORDER BY mi.title COLLATE NOCASE, mi.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]localPersonMedia, 0)
	for rows.Next() {
		var item localPersonMedia
		var year, tmdbID sql.NullInt64
		var poster, credits sql.NullString
		var rating sql.NullFloat64
		if err := rows.Scan(&item.ID, &item.Title, &year, &poster, &rating, &tmdbID, &credits); err != nil {
			return nil, err
		}
		item.Year = intPointer(year)
		item.Poster = stringPointer(poster)
		item.Rating = floatPointer(rating)
		item.TMDBID = int64Pointer(tmdbID)
		item.Credits = decodeCredits(credits)
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (a *API) localPersonSeries(ctx context.Context, access libraryAccess) ([]localPersonMedia, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT s.id, COALESCE(NULLIF(s.display_title, ''), s.title), s.year,
		       s.poster_url, s.rating, s.tmdb_id, s.credits
		  FROM series s
		  JOIN libraries l ON l.id = s.library_id AND l.kind = 'video'
		 WHERE `+access.Clause("s")+`
		 ORDER BY s.title COLLATE NOCASE, s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]localPersonMedia, 0)
	for rows.Next() {
		var item localPersonMedia
		var year, tmdbID sql.NullInt64
		var poster, credits sql.NullString
		var rating sql.NullFloat64
		if err := rows.Scan(&item.ID, &item.Title, &year, &poster, &rating, &tmdbID, &credits); err != nil {
			return nil, err
		}
		item.Year = intPointer(year)
		item.Poster = stringPointer(poster)
		item.Rating = floatPointer(rating)
		item.TMDBID = int64Pointer(tmdbID)
		item.Credits = decodeCredits(credits)
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func rolesForPerson(credits creditsDTO, id int) ([]string, bool) {
	roles := []string{}
	matched := false
	for _, credit := range credits.Cast {
		if credit.ID != id {
			continue
		}
		matched = true
		if value := strings.TrimSpace(credit.Character); value != "" {
			roles = appendRole(roles, value, "")
		}
	}
	for _, credit := range credits.Crew {
		if credit.ID != id {
			continue
		}
		matched = true
		if value := strings.TrimSpace(credit.Job); value != "" {
			roles = appendRole(roles, value, "")
		}
	}
	return roles, matched
}

func appendRole(roles []string, character, job string) []string {
	if value := strings.TrimSpace(character); value != "" {
		roles = appendRoles(roles, value)
	}
	if value := strings.TrimSpace(job); value != "" {
		roles = appendRoles(roles, value)
	}
	return roles
}

func appendRoles(roles []string, values ...string) []string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		found := false
		for _, current := range roles {
			if current == value {
				found = true
				break
			}
		}
		if !found {
			roles = append(roles, value)
		}
	}
	return roles
}

func personMedia(item localPersonMedia, mediaType string, roles []string) personMediaDTO {
	var tmdbID int64
	if item.TMDBID != nil {
		tmdbID = *item.TMDBID
	}
	if roles == nil {
		roles = []string{}
	}
	return personMediaDTO{
		Type:      mediaType,
		ID:        item.ID,
		TMDBID:    tmdbID,
		Title:     item.Title,
		Year:      item.Year,
		PosterURL: item.Poster,
		Rating:    item.Rating,
		Roles:     roles,
	}
}

func intValuePointer(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}
