package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/catlinux/cinecat/internal/playback"
)

const fileInfoCacheMax = 256

// FileInspector obtiene la ficha técnica de un archivo. Lo implementan el
// Prober y el Service de playback; en las pruebas se sustituye por un falso.
type FileInspector interface {
	Inspect(ctx context.Context, path string) (*playback.FileInfo, error)
}

type PlaybackInfo struct {
	Mode string `json:"mode"`
	// Method distingue dentro de HLS si se copia el vídeo o se recodifica.
	Method string `json:"method"`
	// Hardware indica que la recodificación prevista usaría la gráfica.
	Hardware    bool   `json:"hardware"`
	Description string `json:"description"`
	Supported   bool   `json:"supported"`
}

type fileInfoResponse struct {
	*playback.FileInfo
	Path     string       `json:"path"`
	Playback PlaybackInfo `json:"playback"`
}

// WithFileInspector permite usar el analizador del servicio de playback.
func (a *API) WithFileInspector(inspector FileInspector) *API {
	a.fileInspector = inspector
	if a.thumbnails != nil {
		a.thumbnails.inspector = inspector
	}
	return a
}

type fileInfoKey struct {
	path       string
	size       int64
	modifiedAt int64
}

type fileInfoCache struct {
	mu      sync.Mutex
	entries map[fileInfoKey]playback.FileInfo
}

func newFileInfoCache() *fileInfoCache {
	return &fileInfoCache{entries: make(map[fileInfoKey]playback.FileInfo)}
}

func (c *fileInfoCache) get(key fileInfoKey) (playback.FileInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	info, ok := c.entries[key]
	return info, ok
}

func (c *fileInfoCache) put(key fileInfoKey, info playback.FileInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= fileInfoCacheMax {
		for existing := range c.entries {
			delete(c.entries, existing)
			break
		}
	}
	c.entries[key] = info
}

func (a *API) handleItemFileInfo(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}

	media, err := a.mediaFile(r, id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	case err != nil:
		log.Printf("library: leer archivo del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	safePath, err := playback.SafePath(media.libraryPath, media.path)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}
	stat, err := os.Stat(safePath)
	if err != nil || stat.IsDir() {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}

	libraryRoot, rootErr := filepath.Abs(media.libraryPath)
	relative, err := filepath.Rel(libraryRoot, safePath)
	if rootErr != nil || err != nil || relative == "." || relative == ".." ||
		strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		relative = filepath.Base(safePath)
	}
	relative = filepath.ToSlash(relative)

	info, err := a.cachedFileInfo(r.Context(), safePath)
	if err != nil {
		if errors.Is(err, playback.ErrFFprobeNotFound) {
			writeJSONError(w, http.StatusServiceUnavailable, "ffprobe no está disponible en el servidor")
			return
		}
		log.Printf("library: analizar archivo del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusBadGateway, "no se pudo analizar el archivo")
		return
	}

	if info.FileName == "" {
		info.FileName = filepath.Base(safePath)
	}
	caps := playback.ParseCapabilities(r.URL.Query().Get("caps"))
	writeJSON(w, http.StatusOK, fileInfoResponse{
		FileInfo: &info,
		Path:     relative,
		Playback: playbackInfo(safePath, &info, caps, a.maxPlaybackHeight(), func() bool {
			if a.playback == nil {
				return false
			}
			return a.playback.HardwareAvailableFor(r.Context(), safePath)
		}),
	})
}

// cachedFileInfo reutiliza el análisis de ffprobe mientras el archivo no
// cambie de tamaño ni de fecha.
func (a *API) cachedFileInfo(ctx context.Context, safePath string) (playback.FileInfo, error) {
	stat, err := os.Stat(safePath)
	if err != nil || stat.IsDir() {
		return playback.FileInfo{}, playback.ErrFileNotFound
	}
	key := fileInfoKey{
		path:       safePath,
		size:       stat.Size(),
		modifiedAt: stat.ModTime().UnixNano(),
	}
	cache := a.fileInfoCache
	if cache == nil {
		cache = newFileInfoCache()
		a.fileInfoCache = cache
	}
	if info, cached := cache.get(key); cached {
		return info, nil
	}
	if a.fileInspector == nil {
		return playback.FileInfo{}, playback.ErrFFprobeNotFound
	}
	inspected, err := a.fileInspector.Inspect(ctx, safePath)
	if err != nil {
		return playback.FileInfo{}, err
	}
	if inspected == nil {
		return playback.FileInfo{}, playback.ErrProbeFailed
	}
	info := normalizedFileInfo(inspected)
	cache.put(key, info)
	return info, nil
}

func normalizedFileInfo(info *playback.FileInfo) playback.FileInfo {
	result := *info
	if result.Video == nil {
		result.Video = []playback.VideoStream{}
	}
	if result.Audio == nil {
		result.Audio = []playback.AudioStream{}
	}
	if result.Subtitles == nil {
		result.Subtitles = []playback.SubtitleStream{}
	}
	return result
}

func playbackInfo(path string, info *playback.FileInfo, caps playback.Capabilities, maxHeight int, hardware ...func() bool) PlaybackInfo {
	mode, err := playback.DecideFor(path, info, caps)
	if errors.Is(err, playback.ErrUnsupportedContainer) {
		return PlaybackInfo{
			Description: "Contenedor no soportado para reproducción",
			Supported:   false,
		}
	}
	if err != nil {
		return PlaybackInfo{
			Description: "No se pudo determinar el modo de reproducción",
			Supported:   false,
		}
	}

	switch mode {
	case playback.ModeDirect:
		extension := strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), "."))
		description := "Reproducción directa"
		if extension != "" {
			description = fmt.Sprintf("Reproducción directa (%s)", extension)
		}
		return PlaybackInfo{
			Mode:        "direct",
			Method:      "direct",
			Description: description,
			Supported:   true,
		}
	case playback.ModeRemux:
		copyInfo := info
		if info != nil {
			// La extensión del archivo manda sobre un FileName de caché que no
			// coincida con la ruta real.
			clone := *info
			clone.FileName = filepath.Base(path)
			copyInfo = &clone
		}
		if playback.CanCopy(copyInfo, caps, maxHeight) {
			return PlaybackInfo{
				Mode:        "hls",
				Method:      "copy",
				Description: "Copia (sin recodificar)",
				Supported:   true,
			}
		}
		hardwareUsed := len(hardware) > 0 && hardware[0] != nil && hardware[0]()
		return PlaybackInfo{
			Mode:        "hls",
			Method:      "transcode",
			Hardware:    hardwareUsed,
			Description: "Se recodifica a H.264 + AAC estéreo por HLS",
			Supported:   true,
		}
	default:
		return PlaybackInfo{Description: "No se pudo determinar el modo de reproducción"}
	}
}
