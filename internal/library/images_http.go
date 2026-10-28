package library

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/catlinux/cinecat/internal/metadata"
)

type imageURLRequest struct {
	URL string `json:"url"`
}

func (a *API) handleItemImages(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadMovieItem(w, r)
	if !ok {
		return
	}
	if item.TMDBID == nil {
		writeJSONError(w, http.StatusConflict, "la película no está identificada")
		return
	}
	if a.tmdb == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "TMDB no está configurado")
		return
	}

	options, err := a.tmdb.MovieImages(r.Context(), int(*item.TMDBID))
	if err != nil {
		log.Printf("library: imágenes de TMDB %d: %v", *item.TMDBID, err)
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

func (a *API) handleItemImagePut(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadMovieItem(w, r)
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
		writeJSONBodyError(w, err)
		return
	}
	if !validImageURL(req.URL) {
		writeJSONError(w, http.StatusBadRequest, "la URL debe ser http o https, con host, sin espacios y de 2048 caracteres como máximo")
		return
	}

	_, err := a.db.ExecContext(r.Context(),
		`UPDATE media_items SET `+column+` = ?, `+lockColumn+` = 1 WHERE id = ?`,
		req.URL, item.ID)
	if err != nil {
		log.Printf("library: fijar imagen del ítem %d: %v", item.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	a.writeItemByID(w, r, item.ID)
}

func (a *API) handleItemImageDelete(w http.ResponseWriter, r *http.Request) {
	item, ok := a.loadMovieItem(w, r)
	if !ok {
		return
	}
	column, lockColumn, ok := imageColumns(r.PathValue("kind"))
	if !ok {
		writeJSONError(w, http.StatusNotFound, "imagen no encontrada")
		return
	}

	imageURL := ""
	if item.TMDBID != nil {
		if a.tmdb == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "TMDB no está configurado")
			return
		}
		details, err := a.tmdb.MovieDetails(r.Context(), int(*item.TMDBID))
		if err != nil || details == nil {
			log.Printf("library: ficha de TMDB %d para restaurar la imagen: %v", *item.TMDBID, err)
			writeJSONError(w, http.StatusBadGateway, "no se pudo consultar TMDB")
			return
		}
		if column == "poster_url" {
			imageURL = details.PosterURL
		} else {
			imageURL = details.BackdropURL
		}
	}

	_, err := a.db.ExecContext(r.Context(),
		`UPDATE media_items SET `+column+` = ?, `+lockColumn+` = 0 WHERE id = ?`,
		nullableText(imageURL), item.ID)
	if err != nil {
		log.Printf("library: restaurar imagen del ítem %d: %v", item.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	a.writeItemByID(w, r, item.ID)
}

func imageColumns(kind string) (column, lockColumn string, ok bool) {
	switch kind {
	case "poster":
		return "poster_url", "poster_locked", true
	case "backdrop":
		return "backdrop_url", "backdrop_locked", true
	default:
		return "", "", false
	}
}

func validImageURL(raw string) bool {
	if raw == "" || utf8.RuneCountInString(raw) > 2048 {
		return false
	}
	if strings.IndexFunc(raw, unicode.IsSpace) >= 0 {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return false
	}
	return strings.EqualFold(parsed.Scheme, "http") || strings.EqualFold(parsed.Scheme, "https")
}
