package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	Repository         = "catlinux/Cinecat"
	defaultTagsURL     = "https://api.github.com/repos/" + Repository + "/tags?per_page=100"
	defaultRawBase     = "https://raw.githubusercontent.com/" + Repository
	defaultCacheTTL    = time.Hour
	defaultHTTPTimeout = 10 * time.Second
	maxTagsBytes       = 1 << 20
	maxChangelogBytes  = 512 << 10
)

// Result es el estado de la última comprobación de versiones.
type Result struct {
	Current   string
	Latest    string
	Available bool
	CheckedAt time.Time
	Notes     string
}

// CheckError indica que no se pudo consultar GitHub.
type CheckError struct {
	Err error
}

func (e *CheckError) Error() string {
	if e == nil || e.Err == nil {
		return "update: no se pudo comprobar la actualización"
	}
	return "update: no se pudo comprobar la actualización: " + e.Err.Error()
}

func (e *CheckError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type CheckerOptions struct {
	Current    string
	TagsURL    string
	RawBaseURL string
	Client     *http.Client
	Now        func() time.Time
	CacheTTL   time.Duration
}

type Checker struct {
	current    string
	tagsURL    string
	rawBaseURL string
	client     *http.Client
	now        func() time.Time
	cacheTTL   time.Duration

	mu      sync.Mutex
	cached  Result
	hasData bool
}

func NewChecker(options CheckerOptions) *Checker {
	tagsURL := strings.TrimSpace(options.TagsURL)
	if tagsURL == "" {
		tagsURL = defaultTagsURL
	}
	rawBase := strings.TrimRight(strings.TrimSpace(options.RawBaseURL), "/")
	if rawBase == "" {
		rawBase = defaultRawBase
	}
	client := options.Client
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	} else if client.Timeout == 0 {
		clientCopy := *client
		clientCopy.Timeout = defaultHTTPTimeout
		client = &clientCopy
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	ttl := options.CacheTTL
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	return &Checker{
		current:    strings.TrimSpace(options.Current),
		tagsURL:    tagsURL,
		rawBaseURL: rawBase,
		client:     client,
		now:        now,
		cacheTTL:   ttl,
	}
}

func (c *Checker) Current() string {
	if c == nil {
		return ""
	}
	return c.current
}

// Cached devuelve el último resultado sin consultar GitHub.
func (c *Checker) Cached() (Result, bool) {
	if c == nil {
		return Result{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.hasData {
		return Result{Current: c.current}, false
	}
	return c.cached, true
}

// Check consulta GitHub. force ignora la caché; en caso contrario se reutiliza
// el resultado durante una hora.
func (c *Checker) Check(ctx context.Context, force bool) (Result, error) {
	if c == nil {
		return Result{}, &CheckError{Err: errors.New("checker no configurado")}
	}
	c.mu.Lock()
	if !force && c.hasData && c.now().Sub(c.cached.CheckedAt) < c.cacheTTL {
		result := c.cached
		c.mu.Unlock()
		return result, nil
	}
	c.mu.Unlock()

	tagsBody, err := c.get(ctx, c.tagsURL, maxTagsBytes)
	if err != nil {
		return Result{}, &CheckError{Err: err}
	}
	var tags []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(tagsBody, &tags); err != nil {
		return Result{}, &CheckError{Err: fmt.Errorf("respuesta de etiquetas no válida")}
	}
	var latest Semver
	latestTag := ""
	for _, tag := range tags {
		version, ok := ParseTag(tag.Name)
		if !ok {
			continue
		}
		if latestTag == "" || Compare(version, latest) > 0 {
			latest = version
			latestTag = tag.Name
		}
	}

	result := Result{
		Current:   c.current,
		Latest:    latestTag,
		CheckedAt: c.now().UTC(),
	}
	if current, ok := ParseSemver(c.current); ok && latestTag != "" {
		result.Available = Compare(latest, current) > 0
	}
	if latestTag != "" {
		result.Notes = c.notes(ctx, latestTag)
	}

	c.mu.Lock()
	c.cached = result
	c.hasData = true
	c.mu.Unlock()
	return result, nil
}

func (c *Checker) notes(ctx context.Context, tag string) string {
	url := c.rawBaseURL + "/" + tag + "/CHANGELOG.md"
	body, err := c.get(ctx, url, maxChangelogBytes)
	if err != nil {
		return ""
	}
	return ExtractNotes(body, tag)
}

func (c *Checker) get(ctx context.Context, url string, limit int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	userAgentVersion := c.current
	if userAgentVersion == "" {
		userAgentVersion = "desconocida"
	}
	request.Header.Set("User-Agent", "cinecat/"+userAgentVersion)
	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, fmt.Errorf("GitHub respondió %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("respuesta demasiado grande")
	}
	return body, nil
}
