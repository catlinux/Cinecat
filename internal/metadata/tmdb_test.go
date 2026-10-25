package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type fakeServer struct {
	*httptest.Server

	mu             sync.Mutex
	searchCalls    int
	imagesCalls    int
	imageDownloads int
	status         int
	searchResults  []map[string]any
	posters        []imageInfo
	imageBody      []byte
	lastSearch     url.Values
	lastImages     url.Values
}

func newFakeServer(t *testing.T) *fakeServer {
	t.Helper()
	f := &fakeServer{
		searchResults: []map[string]any{},
		imageBody:     []byte("imagen de prueba"),
	}
	f.Server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeServer) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.URL.Path == "/search/movie":
		f.searchCalls++
		f.lastSearch = r.URL.Query()
		if f.status != 0 {
			w.WriteHeader(f.status)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"results": f.searchResults})
	case strings.HasSuffix(r.URL.Path, "/images"):
		f.imagesCalls++
		f.lastImages = r.URL.Query()
		_ = json.NewEncoder(w).Encode(map[string]any{"posters": f.posters})
	case strings.HasPrefix(r.URL.Path, "/img/"):
		f.imageDownloads++
		_, _ = w.Write(f.imageBody)
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeServer) client(t *testing.T, clock *fakeClock, mutate func(*Options)) *Client {
	t.Helper()
	opts := Options{
		APIKey:       "clave-de-prueba",
		BaseURL:      f.URL,
		ImageBaseURL: f.URL + "/img",
		CacheDir:     t.TempDir(),
		HTTPClient:   f.Client(),
		Now:          clock.Now,
	}
	if mutate != nil {
		mutate(&opts)
	}
	c, err := New(opts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func strPtr(s string) *string { return &s }

// counts devuelve los contadores del servidor simulado.
func (f *fakeServer) counts() (search, images, downloads int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.searchCalls, f.imagesCalls, f.imageDownloads
}

// queries devuelve los últimos parámetros recibidos.
func (f *fakeServer) queries() (search, images url.Values) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastSearch, f.lastImages
}

func TestSearchConResultado(t *testing.T) {
	server := newFakeServer(t)
	server.searchResults = []map[string]any{{
		"id":             603,
		"title":          "Matrix",
		"original_title": "The Matrix",
		"release_date":   "1999-03-31",
		"overview":       "Un hacker descubre la verdad.",
		"poster_path":    "/matrix.jpg",
	}}
	server.posters = []imageInfo{
		{FilePath: "/es-ES.jpg", Language: strPtr("es-ES"), VoteAverage: 5},
		{FilePath: "/es.jpg", Language: strPtr("es"), VoteAverage: 9},
		{FilePath: "/en.jpg", Language: strPtr("en"), VoteAverage: 10},
		{FilePath: "/neutral.jpg", Language: nil, VoteAverage: 7},
	}

	client := server.client(t, newFakeClock(), nil)
	movie, err := client.Search(context.Background(), "Matrix", 1999)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if movie.TMDBID != 603 || movie.Title != "Matrix" || movie.OriginalTitle != "The Matrix" ||
		movie.Year != 1999 || movie.Overview == "" {
		t.Fatalf("resultado inesperado: %+v", movie)
	}
	if want := server.URL + "/img/w500/es-ES.jpg"; movie.PosterURL != want {
		t.Fatalf("PosterURL = %q, se esperaba %q", movie.PosterURL, want)
	}
	if movie.PosterLanguage != "es-ES" {
		t.Fatalf("PosterLanguage = %q, se esperaba es-ES", movie.PosterLanguage)
	}

	searchQuery, imagesQuery := server.queries()
	if got := searchQuery.Get("language"); got != "es-ES" {
		t.Fatalf("language = %q, se esperaba es-ES", got)
	}
	if got := searchQuery.Get("query"); got != "Matrix" {
		t.Fatalf("query = %q, se esperaba Matrix", got)
	}
	if got := searchQuery.Get("year"); got != "1999" {
		t.Fatalf("year = %q, se esperaba 1999", got)
	}
	if got := imagesQuery.Get("include_image_language"); got != imageLanguages {
		t.Fatalf("include_image_language = %q, se esperaba %q", got, imageLanguages)
	}
}

func TestSearchSinResultados(t *testing.T) {
	server := newFakeServer(t)
	client := server.client(t, newFakeClock(), nil)

	ctx := context.Background()
	if movie, err := client.Search(ctx, "No existe", 2020); !errors.Is(err, ErrNotFound) || movie != nil {
		t.Fatalf("se esperaba ErrNotFound y nil, hay %v, %v", movie, err)
	}
	if _, err := client.Search(ctx, "No existe", 2020); !errors.Is(err, ErrNotFound) {
		t.Fatalf("segunda búsqueda: %v", err)
	}

	search, images, _ := server.counts()
	if search != 1 {
		t.Fatalf("peticiones de búsqueda = %d, se esperaba 1 (caché de no encontrado)", search)
	}
	if images != 0 {
		t.Fatalf("peticiones de imágenes = %d, se esperaba 0", images)
	}
}

func TestPrioridadCarteles(t *testing.T) {
	tests := []struct {
		name     string
		posters  []imageInfo
		wantPath string
		wantLang string
	}{
		{
			name: "es-ES gana al resto",
			posters: []imageInfo{
				{FilePath: "/es.jpg", Language: strPtr("es")},
				{FilePath: "/es-ES.jpg", Language: strPtr("es-ES")},
				{FilePath: "/en.jpg", Language: strPtr("en")},
				{FilePath: "/neutral.jpg", Language: nil},
			},
			wantPath: "/es-ES.jpg",
			wantLang: "es-ES",
		},
		{
			name: "sin es-ES se usa español genérico",
			posters: []imageInfo{
				{FilePath: "/en.jpg", Language: strPtr("en")},
				{FilePath: "/es.jpg", Language: strPtr("es")},
				{FilePath: "/neutral.jpg", Language: nil},
			},
			wantPath: "/es.jpg",
			wantLang: "es",
		},
		{
			name: "sin español se usa inglés",
			posters: []imageInfo{
				{FilePath: "/neutral.jpg", Language: nil},
				{FilePath: "/en.jpg", Language: strPtr("en")},
			},
			wantPath: "/en.jpg",
			wantLang: "en",
		},
		{
			name: "solo sin idioma",
			posters: []imageInfo{
				{FilePath: "/neutral.jpg", Language: nil},
			},
			wantPath: "/neutral.jpg",
			wantLang: "",
		},
		{
			name: "varios del mismo idioma: gana el mejor votado",
			posters: []imageInfo{
				{FilePath: "/es-ES-malo.jpg", Language: strPtr("es-ES"), VoteAverage: 2},
				{FilePath: "/es-ES-bueno.jpg", Language: strPtr("es-ES"), VoteAverage: 9},
			},
			wantPath: "/es-ES-bueno.jpg",
			wantLang: "es-ES",
		},
		{
			name:     "sin carteles",
			posters:  nil,
			wantPath: "",
			wantLang: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, language := pickPoster(tt.posters)
			if path != tt.wantPath || language != tt.wantLang {
				t.Fatalf("pickPoster = (%q, %q), se esperaba (%q, %q)", path, language, tt.wantPath, tt.wantLang)
			}
		})
	}
}

func TestCacheEvitaSegundasPeticiones(t *testing.T) {
	server := newFakeServer(t)
	server.searchResults = []map[string]any{{
		"id":           603,
		"title":        "Matrix",
		"release_date": "1999-03-31",
	}}
	server.posters = []imageInfo{{FilePath: "/es.jpg", Language: strPtr("es")}}

	clock := newFakeClock()
	client := server.client(t, clock, nil)
	ctx := context.Background()

	first, err := client.Search(ctx, "Matrix", 1999)
	if err != nil {
		t.Fatalf("primera búsqueda: %v", err)
	}
	search, images, _ := server.counts()
	if search != 1 || images != 1 {
		t.Fatalf("peticiones tras la primera búsqueda: %d búsquedas, %d imágenes", search, images)
	}

	second, err := client.Search(ctx, "Matrix", 1999)
	if err != nil {
		t.Fatalf("segunda búsqueda: %v", err)
	}
	if *first != *second {
		t.Fatalf("la caché devolvió otro resultado: %+v frente a %+v", second, first)
	}
	search, images, _ = server.counts()
	if search != 1 || images != 1 {
		t.Fatalf("la segunda búsqueda hizo peticiones: %d búsquedas, %d imágenes", search, images)
	}

	clock.Advance(DefaultCacheTTL + time.Hour)
	if _, err := client.Search(ctx, "Matrix", 1999); err != nil {
		t.Fatalf("búsqueda tras el TTL: %v", err)
	}
	search, images, _ = server.counts()
	if search != 2 || images != 2 {
		t.Fatalf("tras el TTL se esperaban 2 peticiones de cada tipo: %d búsquedas, %d imágenes", search, images)
	}
}

func TestErroresDeLaAPI(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   error
	}{
		{name: "clave inválida", status: http.StatusUnauthorized, want: ErrInvalidAPIKey},
		{name: "límite de tasa", status: http.StatusTooManyRequests, want: ErrRateLimited},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newFakeServer(t)
			server.status = tt.status
			client := server.client(t, newFakeClock(), nil)

			_, err := client.Search(context.Background(), "Matrix", 1999)
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, se esperaba %v", err, tt.want)
			}
			if strings.Contains(err.Error(), "clave-de-prueba") {
				t.Fatal("el error no debe incluir la clave de API")
			}
		})
	}
}

func TestErrorDelServidor(t *testing.T) {
	server := newFakeServer(t)
	server.status = http.StatusInternalServerError
	client := server.client(t, newFakeClock(), nil)

	_, err := client.Search(context.Background(), "Matrix", 1999)
	if err == nil {
		t.Fatal("se esperaba un error con un 500")
	}
	if errors.Is(err, ErrInvalidAPIKey) || errors.Is(err, ErrRateLimited) || errors.Is(err, ErrNotFound) {
		t.Fatalf("el 500 no debe confundirse con otro error: %v", err)
	}
	if strings.Contains(err.Error(), "clave-de-prueba") {
		t.Fatal("el error no debe incluir la clave de API")
	}
}

func TestDescargaImagenConCache(t *testing.T) {
	server := newFakeServer(t)
	clock := newFakeClock()
	client := server.client(t, clock, nil)
	ctx := context.Background()
	imageURL := server.URL + "/img/w500/es-ES.jpg"

	first, err := client.DownloadImage(ctx, imageURL)
	if err != nil {
		t.Fatalf("DownloadImage: %v", err)
	}
	if string(first) != string(server.imageBody) {
		t.Fatalf("imagen inesperada: %q", first)
	}

	second, err := client.DownloadImage(ctx, imageURL)
	if err != nil {
		t.Fatalf("segunda DownloadImage: %v", err)
	}
	if string(second) != string(first) {
		t.Fatal("la caché devolvió otra imagen")
	}
	if _, _, downloads := server.counts(); downloads != 1 {
		t.Fatalf("descargas = %d, se esperaba 1 con la caché vigente", downloads)
	}

	clock.Advance(DefaultCacheTTL + time.Hour)
	if _, err := client.DownloadImage(ctx, imageURL); err != nil {
		t.Fatalf("DownloadImage tras el TTL: %v", err)
	}
	if _, _, downloads := server.counts(); downloads != 2 {
		t.Fatalf("descargas tras el TTL = %d, se esperaban 2", downloads)
	}
}

func TestNewSinClave(t *testing.T) {
	if _, err := New(Options{CacheDir: t.TempDir()}); !errors.Is(err, ErrMissingAPIKey) {
		t.Fatalf("error = %v, se esperaba ErrMissingAPIKey", err)
	}
}
