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
	defaultOMDbBaseURL = "https://www.omdbapi.com/"
	defaultOMDbTimeout = 10 * time.Second

	// DefaultOMDbCacheTTL conserva las búsquedas en disco para no gastar la
	// cuota de la clave gratuita (1000 peticiones/día).
	DefaultOMDbCacheTTL = 30 * 24 * time.Hour

	maxOMDbResponseBytes = 1 << 20 // 1 MiB
)

// Errores del cliente de OMDb.
var (
	ErrMissingOMDbAPIKey = errors.New("metadata: falta la clave de API de OMDb")
	ErrOMDbNotFound      = errors.New("metadata: película no encontrada en OMDb")
)

// OMDbResult es la información que se guarda de una película.
type OMDbResult struct {
	Title     string `json:"title"`
	Year      int    `json:"year"`
	Plot      string `json:"plot"`
	PosterURL string `json:"poster_url"`
	IMDbID    string `json:"imdb_id"`
	Type      string `json:"type"`
}

// OMDbOptions configura el cliente.
type OMDbOptions struct {
	APIKey     string
	BaseURL    string
	CacheDir   string
	CacheTTL   time.Duration
	HTTPClient *http.Client
	// Now permite sustituir el reloj en las pruebas.
	Now func() time.Time
}

// OMDbClient consulta OMDb y cachea las búsquedas en disco.
type OMDbClient struct {
	apiKey     string
	baseURL    *url.URL
	cacheDir   string
	cacheTTL   time.Duration
	httpClient *http.Client
	now        func() time.Time
}

// NewOMDbClient crea el cliente de OMDb.
func NewOMDbClient(opts OMDbOptions) (*OMDbClient, error) {
	apiKey := strings.TrimSpace(opts.APIKey)
	if apiKey == "" {
		return nil, ErrMissingOMDbAPIKey
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = defaultOMDbBaseURL
	}
	base, err := url.Parse(baseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, errors.New("metadata: URL base de OMDb no válida")
	}

	cacheDir := opts.CacheDir
	if cacheDir == "" {
		dir, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("metadata: no se puede determinar el directorio de caché: %w", err)
		}
		cacheDir = filepath.Join(dir, "cinecat", "omdb")
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultOMDbTimeout}
	}

	ttl := opts.CacheTTL
	if ttl <= 0 {
		ttl = DefaultOMDbCacheTTL
	}

	now := opts.Now
	if now == nil {
		now = time.Now
	}

	return &OMDbClient{
		apiKey:     apiKey,
		baseURL:    base,
		cacheDir:   cacheDir,
		cacheTTL:   ttl,
		httpClient: httpClient,
		now:        now,
	}, nil
}

type omdbResponse struct {
	Title    string `json:"Title"`
	Year     string `json:"Year"`
	Plot     string `json:"Plot"`
	Poster   string `json:"Poster"`
	IMDbID   string `json:"imdbID"`
	Type     string `json:"Type"`
	Response string `json:"Response"`
	Error    string `json:"Error"`
}

type omdbCacheEntry struct {
	CachedAt time.Time       `json:"cached_at"`
	Data     json.RawMessage `json:"data"`
}

type omdbCached struct {
	Found  bool        `json:"found"`
	Result *OMDbResult `json:"result,omitempty"`
}

// Search busca una película por título y año. Devuelve ErrOMDbNotFound si la
// API responde que no la ha encontrado. Las búsquedas se cachean en disco para
// respetar la cuota de la clave gratuita.
func (c *OMDbClient) Search(ctx context.Context, title string, year int) (*OMDbResult, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("metadata: el título no puede estar vacío")
	}

	key := omdbCacheKey(title, year)
	var cached omdbCached
	if c.readCache(key, &cached) {
		if !cached.Found {
			return nil, ErrOMDbNotFound
		}
		if cached.Result != nil {
			return cached.Result, nil
		}
	}

	params := url.Values{}
	params.Set("apikey", c.apiKey)
	params.Set("t", title)
	params.Set("type", "movie")
	if year > 0 {
		params.Set("y", strconv.Itoa(year))
	}

	u := *c.baseURL
	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.New("metadata: no se pudo preparar la petición a OMDb")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.New("metadata: error de red al consultar OMDb")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata: OMDb respondió %d", resp.StatusCode)
	}

	var payload omdbResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxOMDbResponseBytes)).Decode(&payload); err != nil {
		return nil, errors.New("metadata: respuesta no válida de OMDb")
	}

	if strings.EqualFold(strings.TrimSpace(payload.Response), "False") {
		reason := cleanOMDbValue(payload.Error)
		c.writeCache(key, omdbCached{Found: false})
		if reason == "" {
			return nil, ErrOMDbNotFound
		}
		return nil, fmt.Errorf("%w: %s", ErrOMDbNotFound, reason)
	}

	result := &OMDbResult{
		Title:     cleanOMDbValue(payload.Title),
		Year:      parseOMDbYear(payload.Year),
		Plot:      cleanOMDbValue(payload.Plot),
		PosterURL: cleanOMDbValue(payload.Poster),
		IMDbID:    cleanOMDbValue(payload.IMDbID),
		Type:      cleanOMDbValue(payload.Type),
	}
	c.writeCache(key, omdbCached{Found: true, Result: result})
	return result, nil
}

// cleanOMDbValue convierte el literal "N/A" de OMDb en cadena vacía.
func cleanOMDbValue(value string) string {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "N/A") {
		return ""
	}
	return value
}

func parseOMDbYear(value string) int {
	value = strings.TrimSpace(value)
	if len(value) < 4 {
		return 0
	}
	year, err := strconv.Atoi(value[:4])
	if err != nil {
		return 0
	}
	return year
}

func omdbCacheKey(title string, year int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("omdb|%s|%d", strings.ToLower(strings.TrimSpace(title)), year)))
	return hex.EncodeToString(sum[:])
}

func (c *OMDbClient) readCache(key string, dst any) bool {
	path := filepath.Join(c.cacheDir, "search", key+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var entry omdbCacheEntry
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

func (c *OMDbClient) writeCache(key string, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	entry, err := json.Marshal(omdbCacheEntry{CachedAt: c.now(), Data: data})
	if err != nil {
		return
	}
	path := filepath.Join(c.cacheDir, "search", key+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(entry); err != nil {
		tmp.Close()
		return
	}
	if err := tmp.Close(); err != nil {
		return
	}
	_ = os.Rename(name, path)
}
