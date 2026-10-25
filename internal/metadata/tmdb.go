// Package metadata implementa el cliente de metadatos de TMDB en español de
// España (es-ES), con caché en disco para búsquedas e imágenes.
package metadata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL      = "https://api.themoviedb.org/3"
	defaultImageBaseURL = "https://image.tmdb.org/t/p"
	defaultPosterSize   = "w500"
	defaultLanguage     = "es-ES"
	imageLanguages      = "es,es-ES,en,null"

	maxResponseBytes = 4 << 20  // 4 MiB
	maxImageBytes    = 20 << 20 // 20 MiB
)

// DefaultCacheTTL es el tiempo que se conservan las búsquedas y las imágenes
// en disco. Es un valor de partida: antes de producción hay que revisarlo
// contra los términos vigentes de TMDB, que limitan la conservación de sus
// datos en caché (no se fija aquí un plazo oficial).
const DefaultCacheTTL = 7 * 24 * time.Hour

// Errores que el llamador puede distinguir.
var (
	ErrMissingAPIKey = errors.New("metadata: falta la clave de API de TMDB")
	ErrNotFound      = errors.New("metadata: sin resultados en TMDB")
	ErrInvalidAPIKey = errors.New("metadata: clave de API de TMDB no válida")
	ErrRateLimited   = errors.New("metadata: límite de peticiones de TMDB alcanzado")
)

// Movie es el resultado de una búsqueda.
type Movie struct {
	TMDBID         int    `json:"tmdb_id"`
	Title          string `json:"title"`
	OriginalTitle  string `json:"original_title"`
	Year           int    `json:"year"`
	Overview       string `json:"overview"`
	PosterURL      string `json:"poster_url"`
	PosterLanguage string `json:"poster_language"`
}

// Options configura el cliente. Todos los campos son opcionales salvo APIKey
// y CacheDir (si CacheDir está vacío se usa el directorio de caché del usuario).
type Options struct {
	APIKey       string
	BaseURL      string
	ImageBaseURL string
	CacheDir     string
	CacheTTL     time.Duration
	PosterSize   string
	HTTPClient   *http.Client
	// Now permite sustituir el reloj en las pruebas.
	Now func() time.Time
}

// Client consulta TMDB y cachea en disco las búsquedas y las imágenes.
type Client struct {
	apiKey     string
	baseURL    *url.URL
	imageBase  *url.URL
	cacheDir   string
	cacheTTL   time.Duration
	posterSize string
	httpClient *http.Client
	now        func() time.Time
}

// New crea el cliente de metadatos.
func New(opts Options) (*Client, error) {
	apiKey := strings.TrimSpace(opts.APIKey)
	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	base, err := url.Parse(baseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, errors.New("metadata: URL base de TMDB no válida")
	}

	imageBaseURL := opts.ImageBaseURL
	if imageBaseURL == "" {
		imageBaseURL = defaultImageBaseURL
	}
	imageBase, err := url.Parse(imageBaseURL)
	if err != nil || imageBase.Scheme == "" || imageBase.Host == "" {
		return nil, errors.New("metadata: URL base de imágenes de TMDB no válida")
	}

	cacheDir := opts.CacheDir
	if cacheDir == "" {
		dir, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("metadata: no se puede determinar el directorio de caché: %w", err)
		}
		cacheDir = filepath.Join(dir, "cinecat", "metadata")
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}

	ttl := opts.CacheTTL
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}

	posterSize := opts.PosterSize
	if posterSize == "" {
		posterSize = defaultPosterSize
	}

	now := opts.Now
	if now == nil {
		now = time.Now
	}

	return &Client{
		apiKey:     apiKey,
		baseURL:    base,
		imageBase:  imageBase,
		cacheDir:   cacheDir,
		cacheTTL:   ttl,
		posterSize: posterSize,
		httpClient: httpClient,
		now:        now,
	}, nil
}

type searchResponse struct {
	Results []struct {
		ID            int     `json:"id"`
		Title         string  `json:"title"`
		OriginalTitle string  `json:"original_title"`
		ReleaseDate   string  `json:"release_date"`
		Overview      string  `json:"overview"`
		PosterPath    string  `json:"poster_path"`
		Popularity    float64 `json:"popularity"`
	} `json:"results"`
}

type imageInfo struct {
	FilePath    string  `json:"file_path"`
	Language    *string `json:"iso_639_1"`
	VoteAverage float64 `json:"vote_average"`
}

type imagesResponse struct {
	Posters []imageInfo `json:"posters"`
}

type cachedSearch struct {
	Found bool   `json:"found"`
	Movie *Movie `json:"movie,omitempty"`
}

type cacheEntry struct {
	CachedAt time.Time       `json:"cached_at"`
	Data     json.RawMessage `json:"data"`
}

// Search busca una película por título y año en es-ES y elige el cartel según
// la prioridad es-ES -> español genérico -> inglés -> sin idioma. Devuelve
// ErrNotFound si no hay resultados. Una caché vigente evita repetir las
// peticiones HTTP.
func (c *Client) Search(ctx context.Context, title string, year int) (*Movie, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("metadata: el título no puede estar vacío")
	}

	key := cacheKey(title, year)
	var cached cachedSearch
	if c.readCache("search", key, &cached) {
		if !cached.Found {
			return nil, ErrNotFound
		}
		if cached.Movie != nil {
			return cached.Movie, nil
		}
	}

	params := url.Values{}
	params.Set("language", defaultLanguage)
	params.Set("query", title)
	params.Set("include_adult", "false")
	if year > 0 {
		params.Set("year", strconv.Itoa(year))
	}

	var search searchResponse
	if err := c.getJSON(ctx, "/search/movie", params, &search); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.writeCache("search", key, cachedSearch{Found: false})
			return nil, ErrNotFound
		}
		return nil, err
	}
	if len(search.Results) == 0 {
		c.writeCache("search", key, cachedSearch{Found: false})
		return nil, ErrNotFound
	}

	chosen := search.Results[0]
	if year > 0 {
		for _, result := range search.Results {
			if releaseYear(result.ReleaseDate) == year {
				chosen = result
				break
			}
		}
	}

	movie := &Movie{
		TMDBID:        chosen.ID,
		Title:         chosen.Title,
		OriginalTitle: chosen.OriginalTitle,
		Year:          releaseYear(chosen.ReleaseDate),
		Overview:      chosen.Overview,
	}

	posterPath, posterLanguage, err := c.posterFor(ctx, chosen.ID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if posterPath == "" && chosen.PosterPath != "" {
		// Sin listado de imágenes: sirve el cartel de la propia búsqueda, que
		// se pidió en es-ES.
		posterPath, posterLanguage = chosen.PosterPath, defaultLanguage
	}
	if posterPath != "" {
		movie.PosterURL = c.imageURL(posterPath)
		movie.PosterLanguage = posterLanguage
	}

	c.writeCache("search", key, cachedSearch{Found: true, Movie: movie})
	return movie, nil
}

// posterFor pide todas las imágenes del título en varios idiomas a la vez y
// devuelve la ruta y el idioma del cartel elegido.
func (c *Client) posterFor(ctx context.Context, tmdbID int) (string, string, error) {
	params := url.Values{}
	params.Set("include_image_language", imageLanguages)

	var images imagesResponse
	path := "/movie/" + strconv.Itoa(tmdbID) + "/images"
	if err := c.getJSON(ctx, path, params, &images); err != nil {
		return "", "", err
	}

	filePath, language := pickPoster(images.Posters)
	return filePath, language, nil
}

// pickPoster aplica la prioridad es-ES -> es -> en -> sin idioma. Dentro de un
// mismo idioma se queda con el cartel mejor votado.
func pickPoster(posters []imageInfo) (string, string) {
	priorities := []string{"es-ES", "es", "en", ""}
	for _, language := range priorities {
		best := -1
		for i, poster := range posters {
			if posterLanguage(poster) != strings.ToLower(language) {
				continue
			}
			if best == -1 || poster.VoteAverage > posters[best].VoteAverage {
				best = i
			}
		}
		if best >= 0 && posters[best].FilePath != "" {
			return posters[best].FilePath, language
		}
	}
	return "", ""
}

func posterLanguage(poster imageInfo) string {
	if poster.Language == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(*poster.Language))
}

func (c *Client) imageURL(filePath string) string {
	if filePath == "" {
		return ""
	}
	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		return filePath
	}
	return strings.TrimRight(c.imageBase.String(), "/") + "/" + c.posterSize + "/" + strings.TrimLeft(filePath, "/")
}

// DownloadImage descarga una imagen y la cachea en disco. La devuelve desde la
// caché si sigue vigente.
func (c *Client) DownloadImage(ctx context.Context, imageURL string) ([]byte, error) {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return nil, errors.New("metadata: URL de imagen vacía")
	}
	parsed, err := url.Parse(imageURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("metadata: URL de imagen no válida")
	}

	sum := sha256.Sum256([]byte(imageURL))
	base := filepath.Join(c.cacheDir, "images", hex.EncodeToString(sum[:]))
	dataPath := base + imageExtension(parsed.Path)
	metaPath := base + ".json"

	if c.imageCacheFresh(metaPath, dataPath) {
		if data, err := os.ReadFile(dataPath); err == nil {
			return data, nil
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, errors.New("metadata: no se pudo preparar la descarga de la imagen")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.New("metadata: error de red al descargar la imagen")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata: descarga de imagen con estado %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes))
	if err != nil {
		return nil, fmt.Errorf("metadata: no se pudo leer la imagen: %w", err)
	}

	// La caché es un extra: si no se puede escribir, se devuelve igualmente.
	_ = writeFileAtomic(dataPath, data)
	meta, err := json.Marshal(imageCacheMeta{CachedAt: c.now()})
	if err == nil {
		_ = writeFileAtomic(metaPath, meta)
	}
	return data, nil
}

type imageCacheMeta struct {
	CachedAt time.Time `json:"cached_at"`
}

func (c *Client) imageCacheFresh(metaPath, dataPath string) bool {
	if _, err := os.Stat(dataPath); err != nil {
		return false
	}
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		return false
	}
	var meta imageCacheMeta
	if err := json.Unmarshal(raw, &meta); err != nil || meta.CachedAt.IsZero() {
		return false
	}
	return c.now().Sub(meta.CachedAt) <= c.cacheTTL
}

func imageExtension(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".webp":
		return strings.ToLower(filepath.Ext(path))
	default:
		return ".img"
	}
}

// getJSON hace una petición GET a TMDB. Nunca incluye la URL completa en los
// errores, porque lleva la clave de API.
func (c *Client) getJSON(ctx context.Context, path string, params url.Values, dst any) error {
	u := *c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + path
	query := u.Query()
	for key, values := range params {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	query.Set("api_key", c.apiKey)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return errors.New("metadata: no se pudo preparar la petición a TMDB")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("metadata: error de red al consultar %s", path)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return ErrInvalidAPIKey
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimited
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("metadata: TMDB respondió %d en %s", resp.StatusCode, path)
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(dst); err != nil {
		return fmt.Errorf("metadata: respuesta no válida de %s", path)
	}
	return nil
}

func cacheKey(title string, year int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("movie|%s|%d", strings.ToLower(strings.TrimSpace(title)), year)))
	return hex.EncodeToString(sum[:])
}

func (c *Client) readCache(kind, key string, dst any) bool {
	path := filepath.Join(c.cacheDir, kind, key+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var entry cacheEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return false
	}
	if entry.CachedAt.IsZero() || c.now().Sub(entry.CachedAt) > c.cacheTTL {
		_ = os.Remove(path)
		return false
	}
	if err := json.Unmarshal(entry.Data, dst); err != nil {
		return false
	}
	return true
}

func (c *Client) writeCache(kind, key string, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	entry, err := json.Marshal(cacheEntry{CachedAt: c.now(), Data: data})
	if err != nil {
		return
	}
	_ = writeFileAtomic(filepath.Join(c.cacheDir, kind, key+".json"), entry)
}

func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func releaseYear(date string) int {
	if len(date) < 4 {
		return 0
	}
	year, err := strconv.Atoi(date[:4])
	if err != nil {
		return 0
	}
	return year
}
