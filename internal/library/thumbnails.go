package library

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/catlinux/cinecat/internal/playback"
)

const (
	thumbnailWaitTimeout       = 30 * time.Second
	thumbnailGenerationTimeout = 20 * time.Second
	thumbnailConcurrency       = 2
	// Un fotograma por debajo de 24/255 se considera casi negro y se reintenta
	// en otro instante del vídeo.
	thumbnailDarkLuma = 24.0
)

var (
	ErrThumbnailBusy        = errors.New("library: demasiadas miniaturas generándose")
	ErrThumbnailUnavailable = errors.New("library: no se pudo generar la miniatura")
)

// ThumbnailRunner abstrae la ejecución de FFmpeg. Las pruebas lo sustituyen
// para no depender del binario real.
type ThumbnailRunner func(ctx context.Context, args []string) error

// Thumbnailer genera y cachea un fotograma JPEG por ítem. La clave incluye
// fecha y tamaño del archivo, así que reemplazarlo invalida la miniatura.
type Thumbnailer struct {
	cacheDir  string
	inspector FileInspector
	runner    ThumbnailRunner

	sem             chan struct{}
	waitTimeout     time.Duration
	generationLimit time.Duration
	locks           sync.Map
}

// NewThumbnailer crea el generador perezoso de miniaturas.
func NewThumbnailer(cacheDir string, inspector FileInspector) *Thumbnailer {
	return &Thumbnailer{
		cacheDir:        strings.TrimSpace(cacheDir),
		inspector:       inspector,
		runner:          runThumbnailFFmpeg,
		sem:             make(chan struct{}, thumbnailConcurrency),
		waitTimeout:     thumbnailWaitTimeout,
		generationLimit: thumbnailGenerationTimeout,
	}
}

// WithRunner sustituye FFmpeg en las pruebas.
func (t *Thumbnailer) WithRunner(runner ThumbnailRunner) *Thumbnailer {
	if runner != nil {
		t.runner = runner
	}
	return t
}

// WithLimits ajusta los tiempos de espera y generación; existe para pruebas.
func (t *Thumbnailer) WithLimits(wait, generation time.Duration) *Thumbnailer {
	if wait > 0 {
		t.waitTimeout = wait
	}
	if generation > 0 {
		t.generationLimit = generation
	}
	return t
}

// Get devuelve la ruta de la miniatura, generándola si no está en caché.
func (t *Thumbnailer) Get(ctx context.Context, id int64, path string) (string, error) {
	if t == nil || t.cacheDir == "" {
		return "", ErrThumbnailUnavailable
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", fmt.Errorf("%w: archivo no encontrado", ErrThumbnailUnavailable)
	}
	target := filepath.Join(t.cacheDir,
		fmt.Sprintf("%d-%d-%d.jpg", id, info.ModTime().UnixNano(), info.Size()))
	if usableThumbnail(target) {
		return target, nil
	}
	if err := os.MkdirAll(t.cacheDir, 0o750); err != nil {
		return "", fmt.Errorf("%w: %v", ErrThumbnailUnavailable, err)
	}

	if err := t.acquire(ctx); err != nil {
		return "", err
	}
	defer func() { <-t.sem }()

	keyLock, _ := t.locks.LoadOrStore(target, &sync.Mutex{})
	mutex := keyLock.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()
	if usableThumbnail(target) {
		return target, nil
	}

	genCtx, cancel := context.WithTimeout(ctx, t.generationLimit)
	defer cancel()

	duration := 0.0
	if t.inspector != nil {
		inspected, err := t.inspector.Inspect(genCtx, path)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrThumbnailUnavailable, err)
		}
		if inspected != nil {
			duration = inspected.DurationSeconds
		}
	}

	firstTime := thumbnailSeek(duration, 0.10)
	firstPath, firstLuma, err := t.render(genCtx, path, firstTime)
	if err != nil {
		return "", err
	}
	chosen := firstPath

	if firstLuma < thumbnailDarkLuma {
		secondTime := thumbnailSeek(duration, 0.30)
		secondPath, secondLuma, secondErr := t.render(genCtx, path, secondTime)
		if secondErr == nil {
			if secondLuma > firstLuma {
				chosen = secondPath
			} else {
				_ = os.Remove(secondPath)
			}
		}
	}

	if err := os.Rename(chosen, target); err != nil {
		_ = os.Remove(chosen)
		if usableThumbnail(target) {
			return target, nil
		}
		return "", fmt.Errorf("%w: %v", ErrThumbnailUnavailable, err)
	}
	if chosen != firstPath {
		_ = os.Remove(firstPath)
	}
	return target, nil
}

func (t *Thumbnailer) acquire(ctx context.Context) error {
	select {
	case t.sem <- struct{}{}:
		return nil
	default:
	}
	wait := t.waitTimeout
	if wait <= 0 {
		wait = thumbnailWaitTimeout
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case t.sem <- struct{}{}:
		return nil
	case <-timer.C:
		return ErrThumbnailBusy
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *Thumbnailer) render(ctx context.Context, source string, at float64) (string, float64, error) {
	file, err := os.CreateTemp(t.cacheDir, ".thumb-*.jpg")
	if err != nil {
		return "", 0, fmt.Errorf("%w: %v", ErrThumbnailUnavailable, err)
	}
	temp := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(temp)
		return "", 0, fmt.Errorf("%w: %v", ErrThumbnailUnavailable, err)
	}
	_ = os.Remove(temp)

	args := []string{
		"-hide_banner", "-loglevel", "error",
		"-ss", strconv.FormatFloat(at, 'f', 3, 64),
		"-i", source,
		"-frames:v", "1",
		"-vf", "scale=480:-2",
		"-q:v", "4",
		"-f", "image2",
		"-y", temp,
	}
	if err := t.runner(ctx, args); err != nil {
		_ = os.Remove(temp)
		return "", 0, fmt.Errorf("%w: %v", ErrThumbnailUnavailable, err)
	}
	data, err := os.ReadFile(temp)
	if err != nil || len(data) == 0 {
		_ = os.Remove(temp)
		return "", 0, fmt.Errorf("%w: archivo de miniatura vacío", ErrThumbnailUnavailable)
	}
	luma, err := meanJPEGLuma(data)
	if err != nil {
		_ = os.Remove(temp)
		return "", 0, fmt.Errorf("%w: %v", ErrThumbnailUnavailable, err)
	}
	return temp, luma, nil
}

func usableThumbnail(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}

// thumbnailSeek calcula el segundo del fotograma: 10 % de la duración (30 s
// si se desconoce), entre 1 y 600 s y sin superar duración − 1 s (en vídeos de
// menos de dos segundos manda el límite superior).
func thumbnailSeek(duration float64, fraction float64) float64 {
	if duration <= 0 {
		return 30
	}
	if duration <= 1 {
		return 0
	}
	value := duration * fraction
	if value < 1 {
		value = 1
	}
	if value > 600 {
		value = 600
	}
	if max := duration - 1; value > max {
		value = max
	}
	if value < 0 {
		return 0
	}
	return value
}

// meanJPEGLuma decodifica el JPEG y devuelve la media de luminosidad 0-255.
func meanJPEGLuma(data []byte) (float64, error) {
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	bounds := img.Bounds()
	var total float64
	var count int64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			total += 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
			count++
		}
	}
	if count == 0 {
		return 0, errors.New("jpeg sin píxeles")
	}
	return total / float64(count), nil
}

func runThumbnailFFmpeg(ctx context.Context, args []string) error {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("%w: ffmpeg no está disponible", ErrThumbnailUnavailable)
	}
	name := ffmpeg
	commandArgs := append([]string{}, args...)
	if runtime.GOOS != "windows" {
		if nice, err := exec.LookPath("nice"); err == nil {
			commandArgs = append([]string{"-n", "10", ffmpeg}, commandArgs...)
			name = nice
		}
	}
	cmd := exec.CommandContext(ctx, name, commandArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%w: %v", ErrThumbnailUnavailable, ctx.Err())
		}
		return fmt.Errorf("%w: %v: %s", ErrThumbnailUnavailable, err, truncateText(stderr.String(), 300))
	}
	return nil
}

func truncateText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[len(value)-limit:]
}

// WithThumbnails activa el endpoint de miniaturas con la caché indicada.
func (a *API) WithThumbnails(cacheDir string) *API {
	a.thumbnails = NewThumbnailer(cacheDir, a.fileInspector)
	return a
}

// WithThumbnailRunner sustituye FFmpeg; existe para las pruebas.
func (a *API) WithThumbnailRunner(runner ThumbnailRunner) *API {
	if a.thumbnails != nil {
		a.thumbnails.WithRunner(runner)
	}
	return a
}

// WithThumbnailLimits ajusta los límites de espera y generación en pruebas.
func (a *API) WithThumbnailLimits(wait, generation time.Duration) *API {
	if a.thumbnails != nil {
		a.thumbnails.WithLimits(wait, generation)
	}
	return a
}

func (a *API) handleItemThumbnail(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	if _, ok := a.requireVisibleItem(w, r, id); !ok {
		return
	}
	if a.thumbnails == nil {
		writeJSONError(w, http.StatusNotFound, "miniatura no disponible")
		return
	}

	var mediaPath, libraryPath, kind string
	var libraryID int64
	err := a.db.QueryRowContext(r.Context(), `
		SELECT mi.path, l.path, l.kind, l.id
		  FROM media_items mi
		  JOIN libraries l ON l.id = mi.library_id
		 WHERE mi.id = ?`, id).Scan(&mediaPath, &libraryPath, &kind, &libraryID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	case err != nil:
		log.Printf("library: leer ítem %d para miniatura: %v", id, err)
		writeJSONError(w, http.StatusNotFound, "miniatura no disponible")
		return
	case kind != "clips":
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}

	libraryPath = a.libraryRootFor(r.Context(), libraryID, mediaPath, libraryPath)
	safePath, err := playback.SafePath(libraryPath, mediaPath)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}
	stat, err := os.Stat(safePath)
	if err != nil || stat.IsDir() {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}

	thumbnailPath, err := a.thumbnails.Get(r.Context(), id, safePath)
	if err != nil {
		if errors.Is(err, ErrThumbnailBusy) {
			w.Header().Set("Retry-After", "5")
			writeJSONError(w, http.StatusServiceUnavailable, "miniatura ocupada, inténtalo de nuevo")
			return
		}
		log.Printf("library: miniatura del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusNotFound, "miniatura no disponible")
		return
	}

	cacheInfo, err := os.Stat(thumbnailPath)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "miniatura no disponible")
		return
	}
	etag := fmt.Sprintf(`"%x-%x"`, cacheInfo.ModTime().UnixNano(), cacheInfo.Size())
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("ETag", etag)
	w.Header().Set("Content-Type", "image/jpeg")
	if matchesETag(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	file, err := os.Open(thumbnailPath)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "miniatura no disponible")
		return
	}
	defer file.Close()
	http.ServeContent(w, r, "thumbnail.jpg", cacheInfo.ModTime(), file)
}

func matchesETag(header, etag string) bool {
	for _, value := range strings.Split(header, ",") {
		value = strings.TrimSpace(value)
		value = strings.TrimPrefix(value, "W/")
		if value == etag || value == "*" {
			return true
		}
	}
	return false
}
