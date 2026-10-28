package library

import (
	"context"
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
	"github.com/catlinux/cinecat/internal/settings"
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
	settingsStore *settings.Store
	mediaRoots    []string
	thumbnails    *Thumbnailer
	musicCovers   *MusicCoverer
	now           func() time.Time
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
		mediaRoots:    defaultMediaRoots(),
		now:           time.Now,
	}
}

// WithSettings activa los endpoints de ajustes. Con nil no se registran.
func (a *API) WithSettings(store *settings.Store) *API {
	a.settingsStore = store
	if store != nil {
		SetMatchThreshold(store.Float("metadata.match_threshold"))
		store.OnChange(func(keys []string) {
			for _, key := range keys {
				if key == "metadata.match_threshold" {
					SetMatchThreshold(store.Float(key))
					return
				}
			}
		})
	}
	return a
}

// Register monta los endpoints versionados en el mux indicado.
func (a *API) Register(mux *http.ServeMux) {
	if a.settingsStore != nil {
		mux.Handle("GET /api/v1/settings", a.requireAdmin(a.handleSettings))
		mux.Handle("PATCH /api/v1/settings", a.requireAdmin(a.handleSettingsPatch))
		mux.Handle("DELETE /api/v1/settings/{key}", a.requireAdmin(a.handleSettingDelete))
		mux.Handle("GET /api/v1/settings/public", a.requireSession(a.handleSettingsPublic))
	}
	mux.Handle("GET /api/v1/libraries", a.requireSession(a.handleLibraries))
	mux.Handle("POST /api/v1/libraries", a.requireAdmin(a.handleCreateLibrary))
	mux.Handle("GET /api/v1/fs", a.requireAdmin(a.handleFS))
	if a.scanJob != nil {
		mux.Handle("POST /api/v1/libraries/scan", a.requireAdmin(a.handleScanLibraries))
		mux.Handle("GET /api/v1/libraries/scan", a.requireAdmin(a.handleScanStatus))
	}
	mux.Handle("PATCH /api/v1/libraries/{id}", a.requireAdmin(a.handleUpdateLibrary))
	mux.Handle("DELETE /api/v1/libraries/{id}", a.requireAdmin(a.handleDeleteLibrary))
	mux.Handle("POST /api/v1/libraries/{id}/paths", a.requireAdmin(a.handleAddLibraryPath))
	mux.Handle("DELETE /api/v1/libraries/{id}/paths/{pathId}", a.requireAdmin(a.handleDeleteLibraryPath))
	mux.Handle("PUT /api/v1/libraries/{id}/files", a.requireAdmin(a.handleLibraryFileUpload))
	mux.Handle("GET /api/v1/uploads/targets", a.requireAdmin(a.handleUploadTargets))
	mux.Handle("PUT /api/v1/series/{id}/files", a.requireAdmin(a.handleSeriesFileUpload))
	mux.Handle("GET /api/v1/libraries/{id}/browse", a.requireSession(a.handleClipBrowse))
	mux.Handle("GET /api/v1/music/artists", a.requireSession(a.handleMusicArtists))
	mux.Handle("GET /api/v1/music/artists/{id}", a.requireSession(a.handleMusicArtist))
	mux.Handle("GET /api/v1/music/albums", a.requireSession(a.handleMusicAlbums))
	mux.Handle("GET /api/v1/music/albums/{id}", a.requireSession(a.handleMusicAlbum))
	mux.Handle("GET /api/v1/music/albums/{id}/cover", a.requireSession(a.handleMusicAlbumCover))
	mux.Handle("GET /api/v1/music/tracks", a.requireSession(a.handleMusicTracks))
	mux.Handle("GET /api/v1/music/tracks/{id}/play", a.requireSession(a.handleMusicTrackPlay))
	mux.Handle("GET /api/v1/genres", a.requireSession(a.handleGenres))
	mux.Handle("GET /api/v1/home", a.requireSession(a.handleHome))
	mux.Handle("GET /api/v1/lists/{list}", a.requireSession(a.handleLists))
	mux.Handle("GET /api/v1/collections", a.requireSession(a.handleCollections))
	mux.Handle("GET /api/v1/collections/{id}", a.requireSession(a.handleCollection))
	mux.Handle("PUT /api/v1/collections/{id}/lists/{list}", a.requireSession(a.handleCollectionListPut))
	mux.Handle("DELETE /api/v1/collections/{id}/lists/{list}", a.requireSession(a.handleCollectionListDelete))
	mux.Handle("POST /api/v1/collections", a.requireAdmin(a.handleCollectionCreate))
	mux.Handle("PATCH /api/v1/collections/{id}", a.requireAdmin(a.handleCollectionPatch))
	mux.Handle("DELETE /api/v1/collections/{id}", a.requireAdmin(a.handleCollectionDelete))
	mux.Handle("POST /api/v1/collections/{id}/items", a.requireAdmin(a.handleCollectionItemAdd))
	mux.Handle("DELETE /api/v1/collections/{id}/items/{itemId}", a.requireAdmin(a.handleCollectionItemRemove))
	mux.Handle("PUT /api/v1/collections/{id}/order", a.requireAdmin(a.handleCollectionOrder))
	mux.Handle("GET /api/v1/people/{id}", a.requireSession(a.handlePerson))
	mux.Handle("GET /api/v1/series", a.requireSession(a.handleSeries))
	mux.Handle("GET /api/v1/series/genres", a.requireSession(a.handleSeriesGenres))
	mux.Handle("GET /api/v1/series/{id}", a.requireSession(a.handleSeriesDetail))
	mux.Handle("GET /api/v1/series/{id}/similar", a.requireSession(a.handleSimilarSeries))
	mux.Handle("PUT /api/v1/series/{id}/lists/{list}", a.requireSession(a.handleSeriesListPut))
	mux.Handle("DELETE /api/v1/series/{id}/lists/{list}", a.requireSession(a.handleSeriesListDelete))
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
	mux.Handle("DELETE /api/v1/items/{id}", a.requireAdmin(a.handleItemTrash))
	mux.Handle("GET /api/v1/trash", a.requireAdmin(a.handleTrashList))
	mux.Handle("POST /api/v1/trash/{id}/restore", a.requireAdmin(a.handleTrashRestore))
	mux.Handle("DELETE /api/v1/trash/{id}", a.requireAdmin(a.handleTrashDelete))
	mux.Handle("POST /api/v1/trash/purge", a.requireAdmin(a.handleTrashPurge))
	mux.Handle("GET /api/v1/items/{id}", a.requireSession(a.handleItem))
	mux.Handle("GET /api/v1/items/{id}/similar", a.requireSession(a.handleSimilarItems))
	mux.Handle("GET /api/v1/items/{id}/thumbnail", a.requireSession(a.handleItemThumbnail))
	mux.Handle("PUT /api/v1/items/{id}/lists/{list}", a.requireSession(a.handleItemListPut))
	mux.Handle("DELETE /api/v1/items/{id}/lists/{list}", a.requireSession(a.handleItemListDelete))
	mux.Handle("GET /api/v1/items/{id}/file-info", a.requireSession(a.handleItemFileInfo))
	mux.Handle("GET /api/v1/items/{id}/preferences", a.requireSession(a.handleItemPreferencesGet))
	mux.Handle("PUT /api/v1/items/{id}/preferences", a.requireSession(a.handleItemPreferencesPut))
	mux.Handle("DELETE /api/v1/items/{id}/preferences", a.requireSession(a.handleItemPreferencesDelete))
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
	mux.Handle("GET /api/v1/stream/music/{id}", http.HandlerFunc(a.handleMusicStream))
	if a.playback != nil {
		mux.Handle("GET /api/v1/items/{id}/play", a.requireSession(a.handlePlay))
		mux.Handle("GET /api/v1/stream/{id}", http.HandlerFunc(a.handleStream))
		mux.Handle("GET /api/v1/stream/{id}/{file}", http.HandlerFunc(a.handleStreamFile))
		mux.Handle("GET /api/v1/stream/{id}/subtitles/{file}", http.HandlerFunc(a.handleStreamSubtitle))
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

const maxLibraryBody = 8 << 10 // 8 KiB

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
	where, args, err := itemFilters(r, userID, a.collectionMin())
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

	queryArgs := append([]any{userID, userID, userID, userID}, args...)
	queryArgs = append(queryArgs, limit, offset)
	rows, err := a.db.QueryContext(r.Context(),
		`SELECT `+mediaItemColumns+watchedColumn+progressColumn+itemListColumns+`
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
	collectionsByItem, err := loadItemCollections(r.Context(), a.db, itemIDs, a.collectionMin())
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
	compactMediaItemListSlice(items, isAdminRequest(r))

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

	item, err := queryMediaItem(r.Context(), a.db, id, sessionUserID(r), a.collectionMin())
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
	case err != nil:
		log.Printf("library: leer ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
	default:
		compactMediaItemDetail(&item, isAdminRequest(r))
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
	URL              string          `json:"url"`
	Mode             playback.Mode   `json:"mode"`
	ExpiresAt        time.Time       `json:"expires_at"`
	AudioTracks      []audioTrackDTO `json:"audio_tracks"`
	SelectedAudio    int             `json:"selected_audio"`
	Qualities        []qualityDTO    `json:"qualities"`
	SelectedQuality  string          `json:"selected_quality"`
	Subtitles        []subtitleDTO   `json:"subtitles"`
	SelectedSubtitle *string         `json:"selected_subtitle"`
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

	caps := playback.ParseCapabilities(r.URL.Query().Get("caps"))
	mode, safePath, ok := a.resolveMedia(w, r, id)
	if !ok {
		return
	}
	if mode == playback.ModeRemux {
		if err := a.playback.CheckFFmpeg(); err != nil {
			writeJSONError(w, http.StatusServiceUnavailable, "ffmpeg no está disponible en el servidor")
			return
		}
	}

	preferences, err := loadItemPreferences(r.Context(), a.db, sessionUserID(r), id)
	if err != nil {
		log.Printf("library: leer preferencias del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	info, _ := a.cachedFileInfo(r.Context(), safePath)
	var infoPointer *playback.FileInfo
	if info.FileName != "" || len(info.Video) > 0 || len(info.Audio) > 0 || len(info.Subtitles) > 0 {
		infoPointer = &info
	}

	signature, expiresAt := a.playback.Sign(id)
	expires := strconv.FormatInt(expiresAt.Unix(), 10)
	options, optionsErr := a.playOptions(
		id, mode, safePath, infoPointer, preferences, r.URL.Query(), expires, signature,
	)
	if optionsErr != nil {
		writeJSONError(w, optionsErr.status, optionsErr.message)
		return
	}

	target := "/api/v1/stream/" + strconv.FormatInt(id, 10)
	if mode == playback.ModeRemux {
		target += "/playlist.m3u8"
	}
	query := url.Values{}
	query.Set("exp", expires)
	query.Set("sig", signature)
	if value := caps.QueryValue(); value != "" {
		query.Set("caps", value)
	}
	if mode == playback.ModeRemux {
		query.Set("audio", strconv.Itoa(options.Audio))
		query.Set("q", options.Quality)
	}

	writeJSON(w, http.StatusOK, playResponse{
		URL:              target + "?" + query.Encode(),
		Mode:             mode,
		ExpiresAt:        expiresAt,
		AudioTracks:      options.AudioTracks,
		SelectedAudio:    options.SelectedAudio,
		Qualities:        options.Qualities,
		SelectedQuality:  options.SelectedQuality,
		Subtitles:        options.Subtitles,
		SelectedSubtitle: options.SelectedSubtitle,
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

	info, _ := a.cachedFileInfo(r.Context(), safePath)
	var infoPointer *playback.FileInfo
	if info.FileName != "" || len(info.Video) > 0 || len(info.Audio) > 0 || len(info.Subtitles) > 0 {
		infoPointer = &info
	}
	selection, selectionErr := a.streamSelection(mode, infoPointer, r.URL.Query())
	if selectionErr != nil {
		writeJSONError(w, selectionErr.status, selectionErr.message)
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
		if err := a.playback.ServeHLSFile(w, r, id, safePath, name, exp, signature, selection); err != nil {
			a.writePlaybackError(w, err)
		}
	}
}

// handleStreamSubtitle sirve una pista de subtítulos convertida a WebVTT con
// la misma firma que el vídeo.
func (a *API) handleStreamSubtitle(w http.ResponseWriter, r *http.Request) {
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
	file := r.PathValue("file")
	if !strings.HasSuffix(file, ".vtt") {
		writeJSONError(w, http.StatusNotFound, "subtítulo no encontrado")
		return
	}
	subtitleID := strings.TrimSuffix(file, ".vtt")
	if !playback.ValidSubtitleID(subtitleID) {
		writeJSONError(w, http.StatusNotFound, "subtítulo no encontrado")
		return
	}
	_, safePath, ok := a.resolveMedia(w, r, id)
	if !ok {
		return
	}
	info, _ := a.cachedFileInfo(r.Context(), safePath)
	var infoPointer *playback.FileInfo
	if info.FileName != "" || len(info.Video) > 0 || len(info.Audio) > 0 || len(info.Subtitles) > 0 {
		infoPointer = &info
	}
	var found *playback.SubtitleTrack
	for _, track := range playback.DiscoverSubtitles(safePath, infoPointer) {
		if track.ID == subtitleID {
			copy := track
			found = &copy
			break
		}
	}
	if found == nil {
		writeJSONError(w, http.StatusNotFound, "subtítulo no encontrado")
		return
	}
	data, err := a.playback.SubtitleWebVTT(r.Context(), id, safePath, *found)
	if err != nil {
		switch {
		case errors.Is(err, playback.ErrSubtitleNotFound),
			errors.Is(err, playback.ErrFileNotFound):
			writeJSONError(w, http.StatusNotFound, "subtítulo no encontrado")
		case errors.Is(err, playback.ErrFFmpegNotFound),
			errors.Is(err, playback.ErrSubtitleConversion):
			writeJSONError(w, http.StatusBadGateway, "no se pudo convertir el subtítulo")
		default:
			log.Printf("library: convertir subtítulo %s del ítem %d: %v", subtitleID, id, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
		}
		return
	}
	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
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

	if _, err := playback.Decide(media.path); errors.Is(err, playback.ErrUnsupportedContainer) {
		writeJSONError(w, http.StatusUnsupportedMediaType, "contenedor no soportado para reproducción")
		return "", "", false
	} else if err != nil {
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

	stat, err := os.Stat(safePath)
	if err != nil || stat.IsDir() {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return "", "", false
	}
	fileInfo, _ := a.cachedFileInfo(r.Context(), safePath)
	var infoPointer *playback.FileInfo
	if fileInfo.FileName != "" || len(fileInfo.Video) > 0 ||
		len(fileInfo.Audio) > 0 || len(fileInfo.Subtitles) > 0 {
		infoPointer = &fileInfo
	}
	mode, err := playback.DecideFor(
		media.path, infoPointer, playback.ParseCapabilities(r.URL.Query().Get("caps")),
	)
	if err != nil {
		log.Printf("library: decidir reproducción del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return "", "", false
	}
	return mode, safePath, true
}

func (a *API) mediaFile(r *http.Request, id int64) (mediaFile, error) {
	var media mediaFile
	var libraryID int64
	err := a.db.QueryRowContext(r.Context(), `
		SELECT mi.path, l.path, l.id
		  FROM media_items mi
		  JOIN libraries l ON l.id = mi.library_id
		 WHERE mi.id = ?`, id,
	).Scan(&media.path, &media.libraryPath, &libraryID)
	if err != nil {
		return media, err
	}
	media.libraryPath = a.libraryRootFor(r.Context(), libraryID, media.path, media.libraryPath)
	return media, nil
}

// libraryRootFor devuelve la carpeta de la biblioteca que contiene el archivo:
// una biblioteca puede tener varias y `libraries.path` es solo la principal, así
// que comprobar contra ella dejaría fuera los archivos de las demás carpetas.
func (a *API) libraryRootFor(ctx context.Context, libraryID int64, mediaPath, fallback string) string {
	rows, err := a.db.QueryContext(ctx,
		`SELECT path FROM library_paths WHERE library_id = ?`, libraryID)
	if err != nil {
		return fallback
	}
	defer rows.Close()
	best := ""
	for rows.Next() {
		var root string
		if rows.Scan(&root) != nil {
			continue
		}
		if pathWithin(root, mediaPath) && len(root) > len(best) {
			best = root
		}
	}
	if best == "" {
		return fallback
	}
	return best
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
