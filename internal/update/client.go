package update

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultReleaseBaseURL = "https://github.com/catlinux/Cinecat/releases/download"
	defaultClientTimeout  = 15 * time.Second
	maxChecksumBytes      = 1 << 10
	checksumFailureTTL    = 10 * time.Minute
)

type checksumCacheEntry struct {
	sha256  string
	expires time.Time
}

// ClientUpdate descarga y cachea el SHA-256 del APK publicado para una versión.
type ClientUpdate struct {
	baseURL string
	client  *http.Client
	now     func() time.Time

	mu    sync.Mutex
	cache map[string]checksumCacheEntry
}

func NewClientUpdate(baseURL string, clients ...*http.Client) *ClientUpdate {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = defaultReleaseBaseURL
	}
	client := &http.Client{Timeout: defaultClientTimeout}
	if len(clients) > 0 && clients[0] != nil {
		client = clients[0]
	}
	return &ClientUpdate{
		baseURL: base,
		client:  client,
		now:     time.Now,
		cache:   make(map[string]checksumCacheEntry),
	}
}

func (c *ClientUpdate) WithBaseURL(baseURL string) *ClientUpdate {
	if c == nil {
		return c
	}
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base != "" {
		c.baseURL = base
	}
	return c
}

func (c *ClientUpdate) WithHTTPClient(client *http.Client) *ClientUpdate {
	if c != nil && client != nil {
		c.client = client
	}
	return c
}

func (c *ClientUpdate) WithNow(now func() time.Time) *ClientUpdate {
	if c != nil && now != nil {
		c.now = now
	}
	return c
}

func (c *ClientUpdate) apkURL(version string) string {
	base := c.baseURL
	if base == "" {
		base = defaultReleaseBaseURL
	}
	return base + "/v" + version + "/cinecat-" + version + ".apk"
}

func (c *ClientUpdate) sha256(ctx context.Context, version string) (string, bool) {
	if c == nil || strings.TrimSpace(version) == "" {
		return "", false
	}
	now := time.Now()
	if c.now != nil {
		now = c.now()
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache == nil {
		c.cache = make(map[string]checksumCacheEntry)
	}
	if entry, ok := c.cache[version]; ok {
		if entry.expires.IsZero() || now.Before(entry.expires) {
			return entry.sha256, entry.sha256 != ""
		}
	}

	sha, err := c.fetchSHA256(ctx, version)
	if err != nil {
		c.cache[version] = checksumCacheEntry{
			expires: now.Add(checksumFailureTTL),
		}
		return "", false
	}
	c.cache[version] = checksumCacheEntry{sha256: sha}
	return sha, true
}

func (c *ClientUpdate) fetchSHA256(ctx context.Context, version string) (string, error) {
	url := c.apkURL(version) + ".sha256"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	client := c.client
	if client == nil {
		client = &http.Client{Timeout: defaultClientTimeout}
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub respondió %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxChecksumBytes+1))
	if err != nil {
		return "", err
	}
	if len(body) > maxChecksumBytes || len(body) < 64 {
		return "", fmt.Errorf("suma de verificación no válida")
	}
	if _, err := hex.DecodeString(string(body[:64])); err != nil {
		return "", fmt.Errorf("suma de verificación no válida")
	}
	return strings.ToLower(string(body[:64])), nil
}

func (a *API) WithClientUpdateBaseURL(baseURL string) *API {
	if a == nil {
		return a
	}
	if a.clientUpdate == nil {
		a.clientUpdate = NewClientUpdate(baseURL)
	} else {
		a.clientUpdate.WithBaseURL(baseURL)
	}
	return a
}

func (a *API) WithClientUpdate(client *ClientUpdate) *API {
	if a != nil && client != nil {
		a.clientUpdate = client
	}
	return a
}

func (a *API) handleClientUpdate(w http.ResponseWriter, r *http.Request) {
	if a == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.URL.Query().Get("platform") != "android" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	rawCode := r.URL.Query().Get("version_code")
	if rawCode == "" {
		writeError(w, http.StatusBadRequest, "version_code no válido")
		return
	}
	clientCode, err := strconv.Atoi(rawCode)
	if err != nil || clientCode < 0 {
		writeError(w, http.StatusBadRequest, "version_code no válido")
		return
	}
	if !a.enabled() || a.checker == nil || a.clientUpdate == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	version, serverCode, ok := normalizedVersion(a.checker.Current())
	if !ok || clientCode >= serverCode {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	sha, ok := a.clientUpdate.sha256(r.Context(), version)
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	notes := ""
	if cached, ok := a.checker.Cached(); ok {
		if cachedVersion, _, valid := normalizedVersion(cached.Latest); valid && cachedVersion == version {
			notes = cached.Notes
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, struct {
		Version     string `json:"version"`
		VersionCode int    `json:"version_code"`
		URL         string `json:"url"`
		SHA256      string `json:"sha256"`
		Notes       string `json:"notes"`
	}{
		Version:     version,
		VersionCode: serverCode,
		URL:         a.clientUpdate.apkURL(version),
		SHA256:      sha,
		Notes:       notes,
	})
}

func normalizedVersion(value string) (string, int, bool) {
	version, ok := ParseSemver(value)
	if !ok {
		return "", 0, false
	}
	normalized := fmt.Sprintf("%d.%d.%d", version.Major, version.Minor, version.Patch)
	code := version.Major*10000 + version.Minor*100 + version.Patch
	return normalized, code, true
}
