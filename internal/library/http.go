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
	"path/filepath"
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
	db            *sql.DB
	auth          *auth.Service
	playback      *playback.Service
	scanJob       *ScanJob
	tmdb          TMDBClient
	fileInspector FileInspector
	fileInfoCache *fileInfoCache
}

// NewAPI crea los handlers de la API de biblioteca.
func NewAPI(db *sql.DB, authSvc *auth.Service, playSvc *playback.Service, scanJob *ScanJob, tmdb TMDBClient) *API {
	var inspector FileInspector
	if playSvc != nil {
		inspector = playSvc
	}
	return &API{
		db:            db,
		auth:          authSvc,
		playback:      playSvc,
		scanJob:       scanJob,
		tmdb:          tmdb,
		fileInspector: inspector,
		fileInfoCache: newFileInfoCache(),
	}
}

// Register monta los endpoints versionados en el mux indicado.
func (a *API) Register(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/libraries", a.requireSession(a.handleLibraries))
	mux.Handle("POST /api/v1/libraries", a.requireAdmin(a.handleCreateLibrary))
	if a.scanJob != nil {
		mux.Handle("POST /api/v1/libraries/scan", a.requireAdmin(a.handleScanLibraries))
		mux.Handle("GET /api/v1/libraries/scan", a.requireAdmin(a.handleScanStatus))
	}
	mux.Handle("DELETE /api/v1/libraries/{id}", a.requireAdmin(a.handleDeleteLibrary))
	mux.Handle("GET /api/v1/genres", a.requireSession(a.handleGenres))
	mux.Handle("GET /api/v1/home", a.requireSession(a.handleHome))
	mux.Handle("GET /api/v1/collections", a.requireSession(a.handleCollections))
	mux.Handle("GET /api/v1/collections/{id}", a.requireSession(a.handleCollection))
	mux.Handle("POST /api/v1/collections", a.requireAdmin(a.handleCollectionCreate))
	mux.Handle("PATCH /api/v1/collections/{id}", a.requireAdmin(a.handleCollectionPatch))
	mux.Handle("DELETE /api/v1/collections/{id}", a.requireAdmin(a.handleCollectionDelete))
	mux.Handle("POST /api/v1/collections/{id}/items", a.requireAdmin(a.handleCollectionItemAdd))
	mux.Handle("DELETE /api/v1/collections/{id}/items/{itemId}", a.requireAdmin(a.handleCollectionItemRemove))
	mux.Handle("PUT /api/v1/collections/{id}/order", a.requireAdmin(a.handleCollectionOrder))
	mux.Handle("GET /api/v1/series", a.requireSession(a.handleSeries))
	mux.Handle("GET /api/v1/series/genres", a.requireSession(a.handleSeriesGenres))
	mux.Handle("GET /api/v1/series/{id}", a.requireSession(a.handleSeriesDetail))
	mux.Handle("GET /api/v1/series/{id}/seasons/{n}", a.requireSession(a.handleSeriesSeason))
	mux.Handle("PUT /api/v1/series/{id}/watched", a.requireSession(a.handleSeriesWatchedPut))
	mux.Handle("DELETE /api/v1/series/{id}/watched", a.requireSession(a.handleSeriesWatchedDelete))
	mux.Handle("PUT /api/v1/series/{id}/seasons/{n}/watched", a.requireSession(a.handleSeriesSeasonWatchedPut))
	mux.Handle("DELETE /api/v1/series/{id}/seasons/{n}/watched", a.requireSession(a.handleSeriesSeasonWatchedDelete))
	mux.Handle("GET /api/v1/series/{id}/candidates", a.requireAdmin(a.handleSeriesCandidates))
	mux.Handle("PUT /api/v1/series/{id}/match", a.requireAdmin(a.handleSeriesMatch))
	mux.Handle("DELETE /api/v1/series/{id}/match", a.requireAdmin(a.handleSeriesMatchDelete))
	mux.Handle("GET /api/v1/series/{id}/images", a.requireAdmin(a.handleSeriesImages))
	mux.Handle("PUT /api/v1/series/{id}/images/{kind}", a.requireAdmin(a.handleSeriesImagePut))
	mux.Handle("DELETE /api/v1/series/{id}/images/{kind}", a.requireAdmin(a.handleSeriesImageDelete))
	mux.Handle("GET /api/v1/items", a.requireSession(a.handleItems))
	mux.Handle("GET /api/v1/items/{id}", a.requireSession(a.handleItem))
	mux.Handle("GET /api/v1/items/{id}/file-info", a.requireSession(a.handleItemFileInfo))
	mux.Handle("PUT /api/v1/items/{id}/progress", a.requireSession(a.handleItemProgressPut))
	mux.Handle("DELETE /api/v1/items/{id}/progress", a.requireSession(a.handleItemProgressDelete))
	mux.Handle("PUT /api/v1/items/{id}/watched", a.requireSession(a.handleItemWatched))
	mux.Handle("DELETE /api/v1/items/{id}/watched", a.requireSession(a.handleItemUnwatched))
	mux.Handle("GET /api/v1/items/{id}/candidates", a.requireAdmin(a.handleItemCandidates))
	mux.Handle("PUT /api/v1/items/{id}/match", a.requireAdmin(a.handleItemMatch))
	mux.Handle("DELETE /api/v1/items/{id}/match", a.requireAdmin(a.handleItemMatchDelete))
	mux.Handle("GET /api/v1/items/{id}/images", a.requireAdmin(a.handleItemImages))
	mux.Handle("PUT /api/v1/items/{id}/images/{kind}", a.requireAdmin(a.handleItemImagePut))
	mux.Handle("DELETE /api/v1/items/{id}/images/{kind}", a.requireAdmin(a.handleItemImageDelete))
	if a.playback != nil {
		mux.Handle("GET /api/v1/items/{id}/play", a.requireSession(a.handlePlay))
		mux.Handle("GET /api/v1/stream/{id}", http.HandlerFunc(a.handleStream))
		mux.Handle("GET /api/v1/stream/{id}/{file}", http.HandlerFunc(a.handleStreamFile))
	}
}

type sessionHandler func(http.ResponseWriter, *http.Request)

func (a *API) requireSession(next sessionHandler) http.Handler {
	return a.requireRole("", next)
}

func (a *API) requireAdmin(next sessionHandler) http.Handler {
	return a.requireRole("admin", next)
}

func (a *API) requireRole(role string, next sessionHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeJSONError(w, http.StatusUnauthorized, "falta el token de sesión")
			return
		}
		user, err := a.auth.Authenticate(r.Context(), token)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidSession) || errors.Is(err, auth.ErrSessionExpired) {
				writeJSONError(w, http.StatusUnauthorized, "sesión no válida")
				return
			}
			log.Printf("library: error al validar la sesión: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		if role != "" && user.Role != role {
			writeJSONError(w, http.StatusForbidden, "se requiere rol de administrador")
			return
		}
		next(w, r.WithContext(withSessionUser(r.Context(), user)))
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

const maxLibraryBody = 8 << 10 // 8 KiB

type createLibraryRequest struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (a *API) handleCreateLibrary(w http.ResponseWriter, r *http.Request) {
	var req createLibraryRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "petición no válida")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "el nombre no puede estar vacío")
		return
	}
	path := strings.TrimSpace(req.Path)
	if path == "" {
		writeJSONError(w, http.StatusBadRequest, "la ruta no puede estar vacía")
		return
	}
	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "la ruta no existe o no es accesible")
		return
	}
	if !info.IsDir() {
		writeJSONError(w, http.StatusBadRequest, "la ruta no es un directorio")
		return
	}

	var exists int
	err = a.db.QueryRowContext(r.Context(),
		`SELECT 1 FROM libraries WHERE path = ?`, path).Scan(&exists)
	switch {
	case err == nil:
		writeJSONError(w, http.StatusConflict, "ya existe una biblioteca con esa ruta")
		return
	case !errors.Is(err, sql.ErrNoRows):
		log.Printf("library: comprobar ruta de biblioteca: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	res, err := a.db.ExecContext(r.Context(),
		`INSERT INTO libraries (name, path) VALUES (?, ?)`, name, path)
	if err != nil {
		log.Printf("library: crear biblioteca: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("library: id de la biblioteca creada: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	writeJSON(w, http.StatusCreated, libraryDTO{ID: id, Name: name, Path: path})
}

func (a *API) handleDeleteLibrary(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
		return
	}

	res, err := a.db.ExecContext(r.Context(), `DELETE FROM libraries WHERE id = ?`, id)
	if err != nil {
		log.Printf("library: borrar biblioteca %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	affected, err := res.RowsAffected()
	if err != nil {
		log.Printf("library: filas borradas de la biblioteca %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if affected == 0 {
		writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleScanLibraries(w http.ResponseWriter, r *http.Request) {
	// El escaneo debe sobrevivir a esta petición; no se le pasa r.Context().
	if !a.scanJob.StartService() {
		writeJSONError(w, http.StatusConflict, "ya hay un escaneo en curso")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"started": true})
}

func (a *API) handleScanStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.scanJob.Status())
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxLibraryBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func (a *API) handleItems(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := pagination(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := BackfillSortKeys(r.Context(), a.db); err != nil {
		log.Printf("library: normalizar títulos: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	userID := sessionUserID(r)
	where, args, err := itemFilters(r, userID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	order, err := itemOrder(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var total int64
	if err := a.db.QueryRowContext(r.Context(),
		`SELECT count(*) FROM media_items`+where, args...).Scan(&total); err != nil {
		log.Printf("library: contar ítems: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	queryArgs := append([]any{userID, userID}, args...)
	queryArgs = append(queryArgs, limit, offset)
	rows, err := a.db.QueryContext(r.Context(),
		`SELECT `+mediaItemColumns+watchedColumn+progressColumn+`
		   FROM media_items
		   LEFT JOIN watch_state ws
		     ON ws.media_item_id = media_items.id AND ws.user_id = ?
		   LEFT JOIN playback_progress pp
		     ON pp.media_item_id = media_items.id AND pp.user_id = ?`+where+order+` LIMIT ? OFFSET ?`, queryArgs...)
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

	itemIDs := make([]int64, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}
	collectionsByItem, err := loadItemCollections(r.Context(), a.db, itemIDs)
	if err != nil {
		log.Printf("library: leer colecciones de los ítems: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	for i := range items {
		if collections, ok := collectionsByItem[items[i].ID]; ok {
			items[i].Collections = collections
		}
	}

	writeJSON(w, http.StatusOK, itemsResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (a *API) handleItem(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}

	item, err := queryMediaItem(r.Context(), a.db, id, sessionUserID(r))
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
	id, ok := idFromPath(r)
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
	id, ok := idFromPath(r)
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

func idFromPath(r *http.Request) (int64, bool) {
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
