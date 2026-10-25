package library

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/catlinux/cinecat/internal/auth"
	"github.com/catlinux/cinecat/internal/playback"
)

const (
	defaultItemsLimit = 50
	maxItemsLimit     = 200
)

// API expone por HTTP las bibliotecas y los ítems de la base de datos.
type API struct {
	db       *sql.DB
	auth     *auth.Service
	playback *playback.Service
}

// NewAPI crea los handlers de la API de biblioteca.
func NewAPI(db *sql.DB, authSvc *auth.Service, playSvc *playback.Service) *API {
	return &API{db: db, auth: authSvc, playback: playSvc}
}

// Register monta los endpoints versionados en el mux indicado.
func (a *API) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/libraries", a.requireSession(a.handleLibraries))
	mux.Handle("GET /api/v1/items", a.requireSession(a.handleItems))
	mux.Handle("GET /api/v1/items/{id}", a.requireSession(a.handleItem))
	if a.playback != nil {
		mux.Handle("GET /api/v1/items/{id}/play", a.requireSession(a.handlePlay))
		mux.Handle("GET /api/v1/stream/{id}", http.HandlerFunc(a.handleStream))
		mux.Handle("GET /api/v1/stream/{id}/{file}", http.HandlerFunc(a.handleStreamFile))
	}
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

type playResponse struct {
	URL       string        `json:"url"`
	Mode      playback.Mode `json:"mode"`
	ExpiresAt time.Time     `json:"expires_at"`
}

type mediaFile struct {
	path        string
	libraryPath string
}

func (a *API) handlePlay(w http.ResponseWriter, r *http.Request) {
	id, ok := itemIDFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}

	mode, _, ok := a.resolveMedia(w, r, id)
	if !ok {
		return
	}
	if mode == playback.ModeRemux {
		if err := a.playback.CheckFFmpeg(); err != nil {
			writeJSONError(w, http.StatusServiceUnavailable, "ffmpeg no está disponible en el servidor")
			return
		}
	}

	signature, expiresAt := a.playback.Sign(id)
	target := "/api/v1/stream/" + strconv.FormatInt(id, 10)
	if mode == playback.ModeRemux {
		target += "/playlist.m3u8"
	}
	query := url.Values{}
	query.Set("exp", strconv.FormatInt(expiresAt.Unix(), 10))
	query.Set("sig", signature)

	writeJSON(w, http.StatusOK, playResponse{
		URL:       target + "?" + query.Encode(),
		Mode:      mode,
		ExpiresAt: expiresAt,
	})
}

func (a *API) handleStream(w http.ResponseWriter, r *http.Request) {
	a.serveSigned(w, r, "")
}

func (a *API) handleStreamFile(w http.ResponseWriter, r *http.Request) {
	a.serveSigned(w, r, r.PathValue("file"))
}

// serveSigned verifica la firma y sirve el archivo directo, la playlist o uno
// de los archivos del remux. name vacío significa la ruta base del ítem.
func (a *API) serveSigned(w http.ResponseWriter, r *http.Request, name string) {
	id, ok := itemIDFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}

	exp := r.URL.Query().Get("exp")
	signature := r.URL.Query().Get("sig")
	if err := a.playback.Verify(id, exp, signature); err != nil {
		if errors.Is(err, playback.ErrExpired) {
			writeJSONError(w, http.StatusForbidden, "URL de reproducción caducada")
		} else {
			writeJSONError(w, http.StatusForbidden, "firma de reproducción no válida")
		}
		return
	}

	mode, safePath, ok := a.resolveMedia(w, r, id)
	if !ok {
		return
	}

	switch mode {
	case playback.ModeDirect:
		if name != "" {
			writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
			return
		}
		if err := playback.ServeDirect(w, r, safePath); err != nil {
			a.writePlaybackError(w, err)
		}
	case playback.ModeRemux:
		if name == "" {
			name = "playlist.m3u8"
		}
		if err := a.playback.ServeHLSFile(w, r, id, safePath, name, exp, signature); err != nil {
			a.writePlaybackError(w, err)
		}
	}
}

// resolveMedia busca el ítem, decide el modo y comprueba que el archivo sigue
// dentro de su biblioteca y existe. Si algo falla, ya escribe la respuesta.
func (a *API) resolveMedia(w http.ResponseWriter, r *http.Request, id int64) (playback.Mode, string, bool) {
	media, err := a.mediaFile(r, id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return "", "", false
	case err != nil:
		log.Printf("library: leer archivo del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return "", "", false
	}

	mode, err := playback.Decide(media.path)
	if errors.Is(err, playback.ErrUnsupportedContainer) {
		writeJSONError(w, http.StatusUnsupportedMediaType, "contenedor no soportado para reproducción")
		return "", "", false
	}
	if err != nil {
		log.Printf("library: decidir reproducción del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return "", "", false
	}

	safePath, err := playback.SafePath(media.libraryPath, media.path)
	if err != nil {
		log.Printf("library: ítem %d fuera de la biblioteca: %v", id, err)
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return "", "", false
	}

	info, err := os.Stat(safePath)
	if err != nil || info.IsDir() {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return "", "", false
	}
	return mode, safePath, true
}

func (a *API) mediaFile(r *http.Request, id int64) (mediaFile, error) {
	var media mediaFile
	err := a.db.QueryRowContext(r.Context(), `
		SELECT mi.path, l.path
		  FROM media_items mi
		  JOIN libraries l ON l.id = mi.library_id
		 WHERE mi.id = ?`, id,
	).Scan(&media.path, &media.libraryPath)
	return media, err
}

func (a *API) writePlaybackError(w http.ResponseWriter, err error) {
	log.Printf("library: reproducción: %v", err)
	switch {
	case errors.Is(err, playback.ErrFFmpegNotFound):
		writeJSONError(w, http.StatusServiceUnavailable, "ffmpeg no está disponible en el servidor")
	case errors.Is(err, playback.ErrFileNotFound):
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
	default:
		writeJSONError(w, http.StatusInternalServerError, "error interno")
	}
}

func itemIDFromPath(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
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
