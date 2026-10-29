package library

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	defaultTMDBImageBaseURL = "https://image.tmdb.org/t/p"
	maxTMDBImageBytes       = 15 << 20
	tmdbImageCacheControl   = "private, max-age=31536000, immutable"
)

var (
	ErrTMDBImageInvalid     = errors.New("library: tamaño o archivo de imagen de TMDB no válido")
	ErrTMDBImageNotFound    = errors.New("library: imagen de TMDB no encontrada")
	ErrTMDBImageUnavailable = errors.New("library: no se pudo obtener la imagen de TMDB")
)

var tmdbImageSizes = map[string]struct{}{
	"w92": {}, "w154": {}, "w185": {}, "w300": {}, "w342": {},
	"w500": {}, "w780": {}, "w1280": {}, "h632": {}, "original": {},
}

var tmdbImageFilePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}\.(jpg|jpeg|png|webp|svg)$`)

// TMDBImage es una imagen lista para servir, desde la caché o desde memoria.
type TMDBImage struct {
	Path        string
	Data        []byte
	ModTime     time.Time
	ContentType string
}

type tmdbImageCall struct {
	done  chan struct{}
	image *TMDBImage
	err   error
}

// TMDBImageCache descarga imágenes de TMDB y las conserva en disco.
type TMDBImageCache struct {
	cacheDir string
	baseURL  string
	client   *http.Client

	mu       sync.Mutex
	inflight map[string]*tmdbImageCall
}

func NewTMDBImageCache(cacheDir string) *TMDBImageCache {
	return &TMDBImageCache{
		cacheDir: strings.TrimSpace(cacheDir),
		baseURL:  defaultTMDBImageBaseURL,
		client:   &http.Client{Timeout: 20 * time.Second},
		inflight: make(map[string]*tmdbImageCall),
	}
}

func (c *TMDBImageCache) WithBaseURL(baseURL string) *TMDBImageCache {
	if strings.TrimSpace(baseURL) != "" {
		c.baseURL = baseURL
	}
	return c
}

func (c *TMDBImageCache) WithHTTPClient(client *http.Client) *TMDBImageCache {
	if client != nil {
		c.client = client
	}
	return c
}

// Get devuelve una imagen cacheada o la descarga. Las peticiones simultáneas
// de la misma imagen comparten una única descarga.
func (c *TMDBImageCache) Get(ctx context.Context, size, file string) (*TMDBImage, error) {
	if c == nil || !validTMDBImage(size, file) {
		return nil, ErrTMDBImageInvalid
	}
	if image, ok := c.cached(size, file); ok {
		return image, nil
	}

	key := size + "/" + file
	c.mu.Lock()
	if c.inflight == nil {
		c.inflight = make(map[string]*tmdbImageCall)
	}
	if call, ok := c.inflight[key]; ok {
		c.mu.Unlock()
		select {
		case <-call.done:
			return call.image, call.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if image, ok := c.cached(size, file); ok {
		c.mu.Unlock()
		return image, nil
	}
	call := &tmdbImageCall{done: make(chan struct{})}
	c.inflight[key] = call
	c.mu.Unlock()

	image, err := c.fetch(ctx, size, file)
	call.image = image
	call.err = err
	c.mu.Lock()
	delete(c.inflight, key)
	c.mu.Unlock()
	close(call.done)
	return image, err
}

func (c *TMDBImageCache) cached(size, file string) (*TMDBImage, bool) {
	if c.cacheDir == "" {
		return nil, false
	}
	path := filepath.Join(c.cacheDir, size, file)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() == 0 {
		return nil, false
	}
	return &TMDBImage{
		Path:        path,
		ModTime:     info.ModTime(),
		ContentType: tmdbImageContentType(file),
	}, true
}

func (c *TMDBImageCache) fetch(ctx context.Context, size, file string) (*TMDBImage, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(c.baseURL), "/")
	if baseURL == "" {
		baseURL = defaultTMDBImageBaseURL
	}
	client := c.client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/"+size+"/"+file, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTMDBImageUnavailable, err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTMDBImageUnavailable, err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, ErrTMDBImageNotFound
	default:
		return nil, fmt.Errorf("%w: TMDB respondió %d", ErrTMDBImageUnavailable, response.StatusCode)
	}
	contentType := strings.TrimSpace(strings.SplitN(response.Header.Get("Content-Type"), ";", 2)[0])
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return nil, fmt.Errorf("%w: TMDB devolvió %q", ErrTMDBImageUnavailable, contentType)
	}

	data, err := io.ReadAll(io.LimitReader(response.Body, maxTMDBImageBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTMDBImageUnavailable, err)
	}
	if len(data) > maxTMDBImageBytes {
		return nil, fmt.Errorf("%w: la imagen supera %d bytes", ErrTMDBImageUnavailable, maxTMDBImageBytes)
	}

	image := &TMDBImage{Data: data, ContentType: tmdbImageContentType(file)}
	if c.cacheDir == "" {
		return image, nil
	}

	target := filepath.Join(c.cacheDir, size, file)
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTMDBImageUnavailable, err)
	}
	temp, err := os.CreateTemp(dir, ".tmdb-image-*")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTMDBImageUnavailable, err)
	}
	tempName := temp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempName)
		}
	}()
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return nil, fmt.Errorf("%w: %v", ErrTMDBImageUnavailable, err)
	}
	if err := temp.Close(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTMDBImageUnavailable, err)
	}
	if err := os.Rename(tempName, target); err != nil {
		if cached, ok := c.cached(size, file); ok {
			removeTemp = false
			return cached, nil
		}
		return nil, fmt.Errorf("%w: %v", ErrTMDBImageUnavailable, err)
	}
	removeTemp = false
	info, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTMDBImageUnavailable, err)
	}
	image.Path = target
	image.ModTime = info.ModTime()
	image.Data = nil
	return image, nil
}

func tmdbImageContentType(file string) string {
	switch strings.ToLower(filepath.Ext(file)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	default:
		return "image/jpeg"
	}
}

func validTMDBImage(size, file string) bool {
	if _, ok := tmdbImageSizes[size]; !ok {
		return false
	}
	return tmdbImageFilePattern.MatchString(file)
}

func (a *API) WithTMDBImages(cacheDir string) *API {
	a.tmdbImages = NewTMDBImageCache(cacheDir)
	return a
}

// WithTMDBImageBaseURL sustituye TMDB en las pruebas.
func (a *API) WithTMDBImageBaseURL(baseURL string) *API {
	if a.tmdbImages != nil {
		a.tmdbImages.WithBaseURL(baseURL)
	}
	return a
}

func (a *API) WithTMDBImageHTTPClient(client *http.Client) *API {
	if a.tmdbImages != nil {
		a.tmdbImages.WithHTTPClient(client)
	}
	return a
}

func (a *API) handleTMDBImage(w http.ResponseWriter, r *http.Request) {
	size := r.PathValue("size")
	file := r.PathValue("file")
	if a.tmdbImages == nil || !validTMDBImage(size, file) {
		writeJSONError(w, http.StatusNotFound, "imagen no encontrada")
		return
	}
	image, err := a.tmdbImages.Get(r.Context(), size, file)
	switch {
	case errors.Is(err, ErrTMDBImageInvalid), errors.Is(err, ErrTMDBImageNotFound):
		writeJSONError(w, http.StatusNotFound, "imagen no encontrada")
		return
	case err != nil:
		log.Printf("library: imagen de TMDB %s/%s: %v", size, file, err)
		writeJSONError(w, http.StatusBadGateway, "no se pudo obtener la imagen")
		return
	}
	if image == nil {
		writeJSONError(w, http.StatusBadGateway, "no se pudo obtener la imagen")
		return
	}

	w.Header().Set("Content-Type", image.ContentType)
	w.Header().Set("Cache-Control", tmdbImageCacheControl)
	// Se sirve desde nuestro origen: un SVG abierto directamente no debe
	// poder ejecutar scripts con la sesión del usuario.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	if image.Path == "" {
		http.ServeContent(w, r, file, time.Time{}, bytes.NewReader(image.Data))
		return
	}
	opened, err := os.Open(image.Path)
	if err != nil {
		log.Printf("library: abrir imagen de TMDB %s/%s: %v", size, file, err)
		writeJSONError(w, http.StatusNotFound, "imagen no encontrada")
		return
	}
	defer opened.Close()
	info, err := opened.Stat()
	if err != nil || info.IsDir() {
		writeJSONError(w, http.StatusNotFound, "imagen no encontrada")
		return
	}
	http.ServeContent(w, r, file, info.ModTime(), opened)
}
