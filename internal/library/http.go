package library

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/catlinux/cinecat/internal/auth"
)

const (
	defaultItemsLimit = 50
	maxItemsLimit     = 200
)

// API expone por HTTP las bibliotecas y los ítems de la base de datos.
type API struct {
	db   *sql.DB
	auth *auth.Service
}

// NewAPI crea los handlers de la API de biblioteca.
func NewAPI(db *sql.DB, authSvc *auth.Service) *API {
	return &API{db: db, auth: authSvc}
}

// Register monta los endpoints versionados en el mux indicado.
func (a *API) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/libraries", a.requireSession(a.handleLibraries))
	mux.Handle("GET /api/v1/items", a.requireSession(a.handleItems))
	mux.Handle("GET /api/v1/items/{id}", a.requireSession(a.handleItem))
}

type sessionHandler func(http.ResponseWriter, *http.Request)

func (a *API) requireSession(next sessionHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeJSONError(w, http.StatusUnauthorized, "falta el token de sesión")
			return
		}
		if _, err := a.auth.Authenticate(r.Context(), token); err != nil {
			if errors.Is(err, auth.ErrInvalidSession) || errors.Is(err, auth.ErrSessionExpired) {
				writeJSONError(w, http.StatusUnauthorized, "sesión no válida")
				return
			}
			log.Printf("library: error al validar la sesión: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		next(w, r)
	})
}

type libraryDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

type librariesResponse struct {
	Libraries []libraryDTO `json:"libraries"`
}

type mediaItemDTO struct {
	ID        int64  `json:"id"`
	LibraryID int64  `json:"library_id"`
	Path      string `json:"path"`
	Title     string `json:"title"`
	Year      *int   `json:"year"`
	Kind      string `json:"kind"`
	Season    *int   `json:"season"`
	Episode   *int   `json:"episode"`
	TMDBID    *int64 `json:"tmdb_id"`
}

type itemsResponse struct {
	Items  []mediaItemDTO `json:"items"`
	Total  int64          `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

func (a *API) handleLibraries(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.QueryContext(r.Context(),
		`SELECT id, name, path FROM libraries ORDER BY name COLLATE NOCASE, id`)
	if err != nil {
		log.Printf("library: listar bibliotecas: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer rows.Close()

	libraries := make([]libraryDTO, 0)
	for rows.Next() {
		var l libraryDTO
		if err := rows.Scan(&l.ID, &l.Name, &l.Path); err != nil {
			log.Printf("library: leer biblioteca: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		libraries = append(libraries, l)
	}
	if err := rows.Err(); err != nil {
		log.Printf("library: recorrer bibliotecas: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	writeJSON(w, http.StatusOK, librariesResponse{Libraries: libraries})
}

func (a *API) handleItems(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := pagination(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var total int64
	if err := a.db.QueryRowContext(r.Context(),
		`SELECT count(*) FROM media_items`).Scan(&total); err != nil {
		log.Printf("library: contar ítems: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	rows, err := a.db.QueryContext(r.Context(), `
		SELECT id, library_id, path, title, year, kind, season, episode, tmdb_id
		  FROM media_items
		 ORDER BY id
		 LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		log.Printf("library: listar ítems: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer rows.Close()

	items := make([]mediaItemDTO, 0)
	for rows.Next() {
		item, err := scanItem(rows.Scan)
		if err != nil {
			log.Printf("library: leer ítem: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("library: recorrer ítems: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	writeJSON(w, http.StatusOK, itemsResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (a *API) handleItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}

	row := a.db.QueryRowContext(r.Context(), `
		SELECT id, library_id, path, title, year, kind, season, episode, tmdb_id
		  FROM media_items
		 WHERE id = ?`, id)

	item, err := scanItem(row.Scan)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
	case err != nil:
		log.Printf("library: leer ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
	default:
		writeJSON(w, http.StatusOK, item)
	}
}

func scanItem(scan func(dest ...any) error) (mediaItemDTO, error) {
	var item mediaItemDTO
	var year, season, episode, tmdbID sql.NullInt64

	err := scan(
		&item.ID, &item.LibraryID, &item.Path, &item.Title, &year, &item.Kind,
		&season, &episode, &tmdbID,
	)
	if err != nil {
		return mediaItemDTO{}, err
	}

	item.Year = intPointer(year)
	item.Season = intPointer(season)
	item.Episode = intPointer(episode)
	if tmdbID.Valid {
		value := tmdbID.Int64
		item.TMDBID = &value
	}
	return item, nil
}

func intPointer(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	value := int(v.Int64)
	return &value
}

func pagination(r *http.Request) (limit, offset int, err error) {
	limit = defaultItemsLimit
	query := r.URL.Query()

	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		n, convErr := strconv.Atoi(raw)
		if convErr != nil || n < 1 || n > maxItemsLimit {
			return 0, 0, fmt.Errorf("limit debe estar entre 1 y %d", maxItemsLimit)
		}
		limit = n
	}

	if raw := strings.TrimSpace(query.Get("offset")); raw != "" {
		n, convErr := strconv.Atoi(raw)
		if convErr != nil || n < 0 {
			return 0, 0, errors.New("offset debe ser un número mayor o igual que 0")
		}
		offset = n
	}
	return limit, offset, nil
}

func bearerToken(r *http.Request) string {
	const prefix = "bearer "
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("library: escribir respuesta: %v", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
