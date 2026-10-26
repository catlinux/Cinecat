package metadata

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
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
	backdrops      []imageInfo
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
		_ = json.NewEncoder(w).Encode(map[string]any{
			"posters":   f.posters,
			"backdrops": f.backdrops,
		})
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

type testAPI struct {
	*httptest.Server

	mu        sync.Mutex
	calls     int
	pathCalls map[string]int
	queries   map[string]url.Values
	times     []time.Time
	routes    map[string]http.HandlerFunc
}

func newTestAPI(t *testing.T) *testAPI {
	t.Helper()
	api := &testAPI{
		pathCalls: make(map[string]int),
		queries:   make(map[string]url.Values),
		routes:    make(map[string]http.HandlerFunc),
	}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.mu.Lock()
		api.calls++
		api.pathCalls[r.URL.Path]++
		api.queries[r.URL.Path] = r.URL.Query()
		api.times = append(api.times, time.Now())
		route := api.routes[r.URL.Path]
		api.mu.Unlock()

		if route == nil {
			http.NotFound(w, r)
			return
		}
		route(w, r)
	}))
	t.Cleanup(api.Close)
	return api
}

func (a *testAPI) route(path string, handler http.HandlerFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.routes[path] = handler
}

func (a *testAPI) count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls
}

func (a *testAPI) pathCount(path string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.pathCalls[path]
}

func (a *testAPI) query(path string) url.Values {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.queries[path]
}

func (a *testAPI) requestTimes() []time.Time {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]time.Time(nil), a.times...)
}

func newAPIClient(t *testing.T, api *testAPI, cacheDir string, clock *fakeClock, apiKey string) *Client {
	t.Helper()
	if clock == nil {
		clock = newFakeClock()
	}
	client, err := New(Options{
		APIKey:       apiKey,
		BaseURL:      api.URL,
		ImageBaseURL: api.URL + "/img",
		CacheDir:     cacheDir,
		HTTPClient:   api.Client(),
		Now:          clock.Now,
		RetryDelay:   10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func newLanguageTestClient(t *testing.T, api *testAPI, cacheDir, language string) *Client {
	t.Helper()
	client, err := New(Options{
		APIKey:       "clave-de-prueba",
		BaseURL:      api.URL,
		ImageBaseURL: api.URL + "/img",
		CacheDir:     cacheDir,
		HTTPClient:   api.Client(),
		Now:          newFakeClock().Now,
		RetryDelay:   10 * time.Millisecond,
		Language:     func() string { return language },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func jsonRoute(body any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}
}

func TestMovieCollectionOrdenaPorEstreno(t *testing.T) {
	api := newTestAPI(t)
	api.route("/movie/562", jsonRoute(map[string]any{
		"id": 562, "belongs_to_collection": map[string]any{"id": 1570},
	}))
	api.route("/collection/1570", jsonRoute(map[string]any{"parts": []map[string]any{
		{"id": 1573, "title": "Jungla de cristal: la venganza", "release_date": "1995-05-19", "vote_count": 6000},
		{"id": 562, "title": "Jungla de cristal", "release_date": "1988-07-15", "vote_count": 11000},
		{"id": 999, "title": "Sin estrenar", "release_date": ""},
		{"id": 1572, "title": "La jungla 2: Alerta roja", "release_date": "1990-07-02", "vote_count": 5000},
	}}))
	api.route("/movie/7", jsonRoute(map[string]any{"id": 7}))
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")

	parts, err := client.MovieCollection(context.Background(), 562)
	if err != nil {
		t.Fatalf("MovieCollection: %v", err)
	}
	var ids []int
	for _, part := range parts {
		ids = append(ids, part.TMDBID)
	}
	if fmt.Sprint(ids) != "[562 1572 1573]" {
		t.Fatalf("partes = %v, se esperaban por fecha y sin la que no tiene fecha", ids)
	}
	if parts[0].Year != 1988 || parts[0].VoteCount != 11000 {
		t.Fatalf("primera parte = %+v", parts[0])
	}
	if got := api.query("/collection/1570").Get("language"); got != "es-ES" {
		t.Fatalf("language = %q", got)
	}

	none, err := client.MovieCollection(context.Background(), 7)
	if err != nil || len(none) != 0 {
		t.Fatalf("sin colección: %v, %v", none, err)
	}
}

func TestSearchMovies(t *testing.T) {
	api := newTestAPI(t)
	results := make([]map[string]any, 0, 12)
	for i := 0; i < 12; i++ {
		results = append(results, map[string]any{
			"id":             100 + i,
			"title":          "Pelicula",
			"original_title": "Movie",
			"release_date":   "1999-03-31",
			"overview":       "Sinopsis",
			"poster_path":    "/poster.jpg",
			"popularity":     10.5,
		})
	}
	api.route("/search/movie", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") == "Nada" {
			_ = json.NewEncoder(w).Encode(map[string]any{"results": []any{}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
	})
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")

	candidates, err := client.SearchMovies(context.Background(), "Matrix", 1999)
	if err != nil {
		t.Fatalf("SearchMovies: %v", err)
	}
	if len(candidates) != 10 {
		t.Fatalf("candidatos = %d, se esperaban 10", len(candidates))
	}
	first := candidates[0]
	if first.TMDBID != 100 || first.Title != "Pelicula" || first.OriginalTitle != "Movie" ||
		first.Year != 1999 || first.Overview != "Sinopsis" || first.Popularity != 10.5 {
		t.Fatalf("candidato inesperado: %+v", first)
	}
	if want := api.URL + "/img/w185/poster.jpg"; first.PosterURL != want {
		t.Fatalf("PosterURL = %q, se esperaba %q", first.PosterURL, want)
	}

	query := api.query("/search/movie")
	if query.Get("language") != "es-ES" || query.Get("year") != "1999" ||
		query.Get("include_adult") != "false" {
		t.Fatalf("parámetros inesperados: %v", query)
	}

	if _, err := client.SearchMovies(context.Background(), "Nada", 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("sin resultados: %v, se esperaba ErrNotFound", err)
	}
}

func TestMovieDetails(t *testing.T) {
	api := newTestAPI(t)
	api.route("/movie/603", jsonRoute(map[string]any{
		"id":             603,
		"imdb_id":        "tt0133093",
		"title":          "Matrix",
		"original_title": "The Matrix",
		"release_date":   "1999-03-31",
		"tagline":        "La realidad es una ilusión",
		"overview":       "Sinopsis en castellano",
		"runtime":        136,
		"vote_average":   8.7,
		"genres":         []map[string]any{{"name": "Acción"}, {"name": "Ciencia ficción"}},
		"credits": map[string]any{
			"cast": []map[string]any{
				{"name": "Actor 5", "order": 5},
				{"name": "Actor 0", "order": 0},
				{"name": "Actor 2", "order": 2},
				{"name": "Actor 1", "order": 1},
				{"name": "Actor 3", "order": 3},
				{"name": "Actor 4", "order": 4},
				{"name": "Actor 6", "order": 6},
				{"name": "Actor 7", "order": 7},
				{"name": "Actor 8", "order": 8},
				{"name": "Actor 9", "order": 9},
			},
			"crew": []map[string]any{
				{"name": "Lana Wachowski", "job": "Director"},
				{"name": "Productor", "job": "Producer"},
				{"name": "Lilly Wachowski", "job": "Director"},
			},
		},
		"images": map[string]any{
			"posters": []map[string]any{
				{"file_path": "/en.jpg", "iso_639_1": "en", "vote_average": 9},
				{"file_path": "/es-ES.jpg", "iso_639_1": "es-ES", "vote_average": 5},
			},
		},
		"poster_path":   "/fallback.jpg",
		"backdrop_path": "/backdrop.jpg",
		"belongs_to_collection": map[string]any{
			"id":            10,
			"name":          "Saga Matrix",
			"poster_path":   "/collection.jpg",
			"backdrop_path": "/collection-backdrop.jpg",
		},
	}))
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")

	details, err := client.MovieDetails(context.Background(), 603)
	if err != nil {
		t.Fatalf("MovieDetails: %v", err)
	}
	if details.TMDBID != 603 || details.IMDbID != "tt0133093" || details.Title != "Matrix" ||
		details.OriginalTitle != "The Matrix" || details.Year != 1999 ||
		details.Tagline != "La realidad es una ilusión" || details.Overview != "Sinopsis en castellano" ||
		details.OverviewLanguage != "es-ES" || details.RuntimeMinutes != 136 || details.Rating != 8.7 {
		t.Fatalf("detalles inesperados: %+v", details)
	}
	if len(details.Genres) != 2 || details.Genres[0] != "Acción" {
		t.Fatalf("géneros inesperados: %v", details.Genres)
	}
	if len(details.Directors) != 2 || details.Directors[0] != "Lana Wachowski" {
		t.Fatalf("dirección inesperada: %v", details.Directors)
	}
	if len(details.Cast) != 8 || details.Cast[0] != "Actor 0" || details.Cast[7] != "Actor 7" {
		t.Fatalf("reparto inesperado: %v", details.Cast)
	}
	if len(details.Credits.Cast) != 10 || details.Credits.Cast[0].Name != "Actor 0" ||
		details.Credits.Cast[0].Character != "" || details.Credits.Cast[0].PhotoURL != "" {
		t.Fatalf("créditos sin foto/personaje inesperados: %+v", details.Credits.Cast)
	}
	if len(details.Credits.Crew) != 3 || details.Credits.Crew[0].Name != "Lana Wachowski" ||
		details.Credits.Crew[1].Name != "Lilly Wachowski" || details.Credits.Crew[2].Name != "Productor" ||
		details.Credits.Crew[0].PhotoURL != "" {
		t.Fatalf("equipo sin foto inesperado: %+v", details.Credits.Crew)
	}
	if want := api.URL + "/img/w500/es-ES.jpg"; details.PosterURL != want {
		t.Fatalf("PosterURL = %q, se esperaba %q", details.PosterURL, want)
	}
	if want := api.URL + "/img/w1280/backdrop.jpg"; details.BackdropURL != want {
		t.Fatalf("BackdropURL = %q, se esperaba %q", details.BackdropURL, want)
	}
	if details.Collection == nil || details.Collection.TMDBID != 10 ||
		details.Collection.Name != "Saga Matrix" ||
		details.Collection.PosterURL != api.URL+"/img/w342/collection.jpg" ||
		details.Collection.BackdropURL != api.URL+"/img/w1280/collection-backdrop.jpg" {
		t.Fatalf("colección inesperada: %+v", details.Collection)
	}

	query := api.query("/movie/603")
	if query.Get("language") != "es-ES" ||
		query.Get("append_to_response") != "credits,images" ||
		query.Get("include_image_language") != "es,null,en" {
		t.Fatalf("parámetros inesperados: %v", query)
	}
}

func TestMovieDetailsCreditsConFotosYLimites(t *testing.T) {
	api := newTestAPI(t)
	cast := make([]map[string]any, 0, 32)
	for i := 0; i < 32; i++ {
		entry := map[string]any{
			"name":      fmt.Sprintf("Actor %02d", i),
			"character": fmt.Sprintf("Personaje %d", i),
			"order":     i,
		}
		if i == 0 {
			entry["profile_path"] = "/actor.jpg"
		}
		cast = append(cast, entry)
	}
	api.route("/movie/700", jsonRoute(map[string]any{
		"id":    700,
		"title": "Reparto",
		"credits": map[string]any{
			"cast": cast,
			"crew": []map[string]any{
				{"name": "Directora", "job": "Director", "department": "Directing", "profile_path": "/directora.jpg"},
				{"name": "Guionista", "job": "Writer", "department": "Writing"},
				{"name": "Guionista", "job": "Writer", "department": "Writing"},
				{"name": "Catering", "job": "Catering", "department": "Crew"},
				{"name": "Productora", "job": "Producer", "department": "Production"},
				{"name": "Compositor", "job": "Original Music Composer", "department": "Sound"},
				{"name": "Fotógrafo", "job": "Director of Photography", "department": "Camera"},
				{"name": "Editor", "job": "Editor", "department": "Editing"},
			},
		},
	}))
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")
	details, err := client.MovieDetails(context.Background(), 700)
	if err != nil {
		t.Fatalf("MovieDetails: %v", err)
	}
	if len(details.Credits.Cast) != 30 {
		t.Fatalf("reparto = %d, se esperaban 30", len(details.Credits.Cast))
	}
	first := details.Credits.Cast[0]
	if first.Name != "Actor 00" || first.Character != "Personaje 0" ||
		first.PhotoURL != api.URL+"/img/w185/actor.jpg" {
		t.Fatalf("primera ficha de reparto inesperada: %+v", first)
	}
	if details.Credits.Cast[29].Name != "Actor 29" {
		t.Fatalf("última ficha de reparto inesperada: %+v", details.Credits.Cast[29])
	}
	if len(details.Cast) != 8 {
		t.Fatalf("el campo antiguo cast cambió: %v", details.Cast)
	}

	crew := details.Credits.Crew
	if len(crew) != 6 {
		t.Fatalf("equipo = %d, se esperaban 6: %+v", len(crew), crew)
	}
	wantJobs := []string{"Director", "Writer", "Producer", "Original Music Composer", "Director of Photography", "Editor"}
	wantNames := []string{"Directora", "Guionista", "Productora", "Compositor", "Fotógrafo", "Editor"}
	for i := range wantJobs {
		if crew[i].Job != wantJobs[i] || crew[i].Name != wantNames[i] {
			t.Fatalf("equipo %d = %+v, se esperaba %s/%s", i, crew[i], wantNames[i], wantJobs[i])
		}
	}
	if crew[0].Department != "Directing" || crew[0].PhotoURL != api.URL+"/img/w185/directora.jpg" {
		t.Fatalf("equipo con foto inesperado: %+v", crew[0])
	}
}

func TestMovieDetailsLimitaElEquipoADoce(t *testing.T) {
	api := newTestAPI(t)
	crew := make([]map[string]any, 0, 13)
	for i := 0; i < 13; i++ {
		crew = append(crew, map[string]any{
			"name": fmt.Sprintf("Director %02d", i), "job": "Director",
		})
	}
	api.route("/movie/702", jsonRoute(map[string]any{
		"id": 702, "title": "Equipo largo",
		"credits": map[string]any{"crew": crew},
	}))
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")
	details, err := client.MovieDetails(context.Background(), 702)
	if err != nil {
		t.Fatalf("MovieDetails: %v", err)
	}
	if len(details.Credits.Crew) != 12 || details.Credits.Crew[11].Name != "Director 11" {
		t.Fatalf("equipo limitado inesperado: %+v", details.Credits.Crew)
	}
}

func TestTVDetailsCreadoresConFoto(t *testing.T) {
	api := newTestAPI(t)
	api.route("/tv/701", jsonRoute(map[string]any{
		"id": 701, "name": "Serie", "overview": "Sinopsis",
		"created_by": []map[string]any{
			{"name": "Creadora", "profile_path": "/creadora.jpg"},
			{"name": "Creador"},
		},
		"credits": map[string]any{
			"cast": []map[string]any{
				{"name": "Actriz", "character": "Protagonista", "order": 0, "profile_path": "/actriz.jpg"},
			},
			"crew": []map[string]any{
				{"name": "Director TV", "job": "Director", "department": "Directing"},
				{"name": "Catering", "job": "Catering"},
			},
		},
	}))
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")
	details, err := client.TVDetails(context.Background(), 701)
	if err != nil {
		t.Fatalf("TVDetails: %v", err)
	}
	if len(details.Credits.Crew) != 3 {
		t.Fatalf("equipo = %+v", details.Credits.Crew)
	}
	if details.Credits.Crew[0].Name != "Creadora" || details.Credits.Crew[0].Job != "Creator" ||
		details.Credits.Crew[0].PhotoURL != api.URL+"/img/w185/creadora.jpg" {
		t.Fatalf("creadora inesperada: %+v", details.Credits.Crew[0])
	}
	if details.Credits.Crew[1].Name != "Creador" || details.Credits.Crew[1].Job != "Creator" {
		t.Fatalf("creador inesperado: %+v", details.Credits.Crew[1])
	}
	if details.Credits.Crew[2].Job != "Director" {
		t.Fatalf("equipo inesperado: %+v", details.Credits.Crew[2])
	}
	if details.Credits.Cast[0].Character != "Protagonista" ||
		details.Credits.Cast[0].PhotoURL != api.URL+"/img/w185/actriz.jpg" {
		t.Fatalf("reparto inesperado: %+v", details.Credits.Cast)
	}
}

func TestMovieDetailsSinColeccion(t *testing.T) {
	api := newTestAPI(t)
	api.route("/movie/604", jsonRoute(map[string]any{
		"id":    604,
		"title": "Película suelta",
	}))
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")

	details, err := client.MovieDetails(context.Background(), 604)
	if err != nil {
		t.Fatalf("MovieDetails: %v", err)
	}
	if details.Collection != nil {
		t.Fatalf("no debería tener colección: %+v", details.Collection)
	}
}

func TestMovieImagesOrdenTamanoLimiteYCaché(t *testing.T) {
	server := newFakeServer(t)
	server.posters = []imageInfo{
		{FilePath: "/en.jpg", Language: strPtr("en"), VoteAverage: 10, Width: 342, Height: 513},
		{FilePath: "/es.jpg", Language: strPtr("es"), VoteAverage: 9, Width: 342, Height: 513},
		{FilePath: "/es-ES.jpg", Language: strPtr("es-ES"), VoteAverage: 5, Width: 342, Height: 513},
		{FilePath: "/neutral.jpg", Language: nil, VoteAverage: 7, Width: 342, Height: 513},
		{FilePath: "/fr.jpg", Language: strPtr("fr"), VoteAverage: 20, Width: 342, Height: 513},
	}
	for i := 0; i < 36; i++ {
		server.posters = append(server.posters, imageInfo{
			FilePath: fmt.Sprintf("/fr-%d.jpg", i), Language: strPtr("fr"), Width: 342, Height: 513,
		})
	}
	server.backdrops = []imageInfo{
		{FilePath: "/en-backdrop.jpg", Language: strPtr("en"), VoteAverage: 4, Width: 1280, Height: 720},
		{FilePath: "/es-backdrop.jpg", Language: strPtr("es-ES"), VoteAverage: 8, Width: 1280, Height: 720},
	}

	client := server.client(t, newFakeClock(), nil)
	options, err := client.MovieImages(context.Background(), 603)
	if err != nil {
		t.Fatalf("MovieImages: %v", err)
	}
	if len(options.Posters) != 40 {
		t.Fatalf("posters = %d, se esperaban 40", len(options.Posters))
	}
	wantLanguages := []string{"es-ES", "es", "en", "", "fr"}
	for i, want := range wantLanguages {
		if options.Posters[i].Language != want {
			t.Fatalf("poster %d: idioma %q, se esperaba %q", i, options.Posters[i].Language, want)
		}
	}
	if options.Posters[4].VoteAverage != 20 {
		t.Fatalf("dentro del mismo idioma debe ir primero la mejor nota: %+v", options.Posters[4])
	}
	if options.Posters[0].URL != server.URL+"/img/w342/es-ES.jpg" ||
		options.Posters[0].FullURL != server.URL+"/img/w780/es-ES.jpg" ||
		options.Posters[0].Width != 342 || options.Posters[0].Height != 513 {
		t.Fatalf("cartel inesperado: %+v", options.Posters[0])
	}
	if len(options.Backdrops) != 2 || options.Backdrops[0].Language != "es-ES" ||
		options.Backdrops[0].URL != server.URL+"/img/w780/es-backdrop.jpg" ||
		options.Backdrops[0].FullURL != server.URL+"/img/w1280/es-backdrop.jpg" {
		t.Fatalf("fondos inesperados: %+v", options.Backdrops)
	}

	_, imagesQuery := server.queries()
	if got := imagesQuery.Get("include_image_language"); got != imageLanguages {
		t.Fatalf("include_image_language = %q, se esperaba %q", got, imageLanguages)
	}

	if _, err := client.MovieImages(context.Background(), 603); err != nil {
		t.Fatalf("MovieImages desde caché: %v", err)
	}
	if _, calls, _ := server.counts(); calls != 1 {
		t.Fatalf("llamadas a imágenes = %d, se esperaba una sola", calls)
	}
}

func TestMovieDetailsRespaldoDeSinopsis(t *testing.T) {
	tests := []struct {
		name         string
		translations []map[string]any
		wantText     string
		wantLang     string
	}{
		{
			name: "es-MX",
			translations: []map[string]any{
				{"iso_639_1": "en", "iso_3166_1": "US", "data": map[string]any{"overview": "English"}},
				{"iso_639_1": "es", "iso_3166_1": "MX", "data": map[string]any{"overview": "Sinopsis mexicana"}},
			},
			wantText: "Sinopsis mexicana",
			wantLang: "es-MX",
		},
		{
			name: "cualquier es",
			translations: []map[string]any{
				{"iso_639_1": "es", "iso_3166_1": "ES", "data": map[string]any{"overview": "Sinopsis española"}},
				{"iso_639_1": "en", "iso_3166_1": "US", "data": map[string]any{"overview": "English"}},
			},
			wantText: "Sinopsis española",
			wantLang: "es",
		},
		{
			name: "en-US",
			translations: []map[string]any{
				{"iso_639_1": "fr", "iso_3166_1": "FR", "data": map[string]any{"overview": "Français"}},
				{"iso_639_1": "en", "iso_3166_1": "US", "data": map[string]any{"overview": "English"}},
			},
			wantText: "English",
			wantLang: "en-US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := newTestAPI(t)
			api.route("/movie/7", jsonRoute(map[string]any{
				"id": 7, "title": "Sinopsis", "overview": "",
			}))
			api.route("/movie/7/translations", jsonRoute(map[string]any{
				"translations": tt.translations,
			}))
			client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")

			details, err := client.MovieDetails(context.Background(), 7)
			if err != nil {
				t.Fatalf("MovieDetails: %v", err)
			}
			if details.Overview != tt.wantText || details.OverviewLanguage != tt.wantLang {
				t.Fatalf("respaldo = %q/%q, se esperaba %q/%q",
					details.Overview, details.OverviewLanguage, tt.wantText, tt.wantLang)
			}
		})
	}
}

func TestClienteUsaIdiomaConfigurado(t *testing.T) {
	api := newTestAPI(t)
	api.route("/search/movie", jsonRoute(map[string]any{
		"results": []map[string]any{{
			"id": 7, "title": "Movie", "original_title": "Movie",
			"release_date": "2000-01-01", "overview": "English overview",
		}},
	}))
	api.route("/movie/7", jsonRoute(map[string]any{
		"id": 7, "title": "Movie", "overview": "English overview",
	}))

	client := newLanguageTestClient(t, api, t.TempDir(), "en-US")
	candidates, err := client.SearchMovies(context.Background(), "Movie", 0)
	if err != nil || len(candidates) != 1 {
		t.Fatalf("SearchMovies: %v, %+v", err, candidates)
	}
	if got := api.query("/search/movie").Get("language"); got != "en-US" {
		t.Fatalf("language de búsqueda = %q", got)
	}

	details, err := client.MovieDetails(context.Background(), 7)
	if err != nil {
		t.Fatalf("MovieDetails: %v", err)
	}
	if got := api.query("/movie/7").Get("language"); got != "en-US" {
		t.Fatalf("language de ficha = %q", got)
	}
	if details.OverviewLanguage != "en-US" {
		t.Fatalf("OverviewLanguage = %q", details.OverviewLanguage)
	}
}

func TestRespaldoSinopsisConIdiomaConfigurado(t *testing.T) {
	translations := []map[string]any{
		{"iso_639_1": "es", "iso_3166_1": "ES", "data": map[string]any{"overview": "Española"}},
		{"iso_639_1": "es", "iso_3166_1": "MX", "data": map[string]any{"overview": "Mexicana"}},
		{"iso_639_1": "en", "iso_3166_1": "US", "data": map[string]any{"overview": "English"}},
	}
	for _, test := range []struct {
		language string
		wantText string
		wantLang string
	}{
		{language: "es-MX", wantText: "Mexicana", wantLang: "es-MX"},
		{language: "es-ES", wantText: "Española", wantLang: "es"},
		{language: "en-US", wantText: "English", wantLang: "en-US"},
	} {
		t.Run(test.language, func(t *testing.T) {
			api := newTestAPI(t)
			api.route("/movie/7", jsonRoute(map[string]any{"id": 7, "title": "Sinopsis", "overview": ""}))
			api.route("/movie/7/translations", jsonRoute(map[string]any{"translations": translations}))
			client := newLanguageTestClient(t, api, t.TempDir(), test.language)
			details, err := client.MovieDetails(context.Background(), 7)
			if err != nil {
				t.Fatalf("MovieDetails: %v", err)
			}
			if details.Overview != test.wantText || details.OverviewLanguage != test.wantLang {
				t.Fatalf("respaldo = %q/%q, se esperaba %q/%q",
					details.Overview, details.OverviewLanguage, test.wantText, test.wantLang)
			}
		})
	}
}

func TestOrdenDeRespaldosDeIdioma(t *testing.T) {
	for language, want := range map[string][]string{
		"es-ES": {"es-ES", "es-MX", "es", "en-US"},
		"es-MX": {"es-MX", "es-ES", "es", "en-US"},
		"en-US": {"en-US"},
	} {
		if got := translationFallbacks(language); !reflect.DeepEqual(got, want) {
			t.Errorf("translationFallbacks(%q) = %v, se esperaba %v", language, got, want)
		}
	}
}

func TestSearchTV(t *testing.T) {
	api := newTestAPI(t)
	api.route("/search/tv", jsonRoute(map[string]any{
		"results": []map[string]any{
			{
				"id": 1399, "name": "Juego de tronos", "original_name": "Game of Thrones",
				"first_air_date": "2011-04-17", "overview": "Sinopsis",
				"poster_path": "/got.jpg", "popularity": 42.5,
			},
		},
	}))
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")

	candidates, err := client.SearchTV(context.Background(), "Juego de tronos", 2011)
	if err != nil {
		t.Fatalf("SearchTV: %v", err)
	}
	if len(candidates) != 1 || candidates[0].TMDBID != 1399 ||
		candidates[0].Title != "Juego de tronos" || candidates[0].OriginalTitle != "Game of Thrones" ||
		candidates[0].Year != 2011 || candidates[0].PosterURL != api.URL+"/img/w185/got.jpg" {
		t.Fatalf("candidato de TV inesperado: %+v", candidates)
	}
	query := api.query("/search/tv")
	if query.Get("language") != "es-ES" || query.Get("first_air_date_year") != "2011" {
		t.Fatalf("parámetros inesperados: %v", query)
	}
}

func TestTVDetails(t *testing.T) {
	api := newTestAPI(t)
	api.route("/tv/1399", jsonRoute(map[string]any{
		"id": 1399, "name": "Juego de tronos", "original_name": "Game of Thrones",
		"first_air_date": "2011-04-17", "overview": "Sinopsis de la serie", "vote_average": 8.4,
		"genres":     []map[string]any{{"name": "Drama"}},
		"created_by": []map[string]any{{"name": "David Benioff"}, {"name": "D. B. Weiss"}},
		"credits": map[string]any{
			"cast": []map[string]any{
				{"name": "Emilia Clarke", "order": 0},
				{"name": "Peter Dinklage", "order": 1},
			},
		},
		"images":        map[string]any{"posters": []map[string]any{}},
		"poster_path":   "/got.jpg",
		"backdrop_path": "/got-backdrop.jpg",
	}))
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")

	details, err := client.TVDetails(context.Background(), 1399)
	if err != nil {
		t.Fatalf("TVDetails: %v", err)
	}
	if details.TMDBID != 1399 || details.Title != "Juego de tronos" ||
		details.OriginalTitle != "Game of Thrones" || details.Year != 2011 ||
		details.Overview != "Sinopsis de la serie" || details.OverviewLanguage != "es-ES" ||
		details.RuntimeMinutes != 0 || details.Rating != 8.4 {
		t.Fatalf("detalles de TV inesperados: %+v", details)
	}
	if len(details.Directors) != 2 || details.Directors[1] != "D. B. Weiss" {
		t.Fatalf("creadores inesperados: %v", details.Directors)
	}
	if len(details.Cast) != 2 || details.Cast[0] != "Emilia Clarke" {
		t.Fatalf("reparto inesperado: %v", details.Cast)
	}
	if details.PosterURL != api.URL+"/img/w500/got.jpg" {
		t.Fatalf("PosterURL = %q", details.PosterURL)
	}
	if details.BackdropURL != api.URL+"/img/w1280/got-backdrop.jpg" {
		t.Fatalf("BackdropURL = %q", details.BackdropURL)
	}
}

func TestSeasonDetails(t *testing.T) {
	api := newTestAPI(t)
	api.route("/tv/1399/season/1", jsonRoute(map[string]any{
		"name": "Temporada 1", "overview": "Sinopsis de la temporada",
		"air_date": "2011-04-17", "poster_path": "/season.jpg",
		"episodes": []map[string]any{
			{
				"episode_number": 1, "name": "Winter Is Coming",
				"overview": "Sinopsis del episodio", "air_date": "2011-04-17",
				"still_path": "/still.jpg",
			},
			{
				"episode_number": 2, "name": "The Kingsroad",
				"overview": "", "air_date": "2011-04-24",
				"still_path": "/still2.jpg",
			},
		},
	}))
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")

	season, err := client.SeasonDetails(context.Background(), 1399, 1)
	if err != nil {
		t.Fatalf("SeasonDetails: %v", err)
	}
	if season.Name != "Temporada 1" || season.Overview != "Sinopsis de la temporada" ||
		season.OverviewLanguage != "es-ES" || season.AirYear != 2011 ||
		season.PosterURL != api.URL+"/img/w342/season.jpg" {
		t.Fatalf("temporada inesperada: %+v", season)
	}
	if len(season.Episodes) != 2 {
		t.Fatalf("episodios = %d", len(season.Episodes))
	}
	first := season.Episodes[0]
	if first.Number != 1 || first.Name != "Winter Is Coming" ||
		first.Overview != "Sinopsis del episodio" || first.AirYear != 2011 ||
		first.StillURL != api.URL+"/img/w300/still.jpg" {
		t.Fatalf("primer episodio inesperado: %+v", first)
	}
	if query := api.query("/tv/1399/season/1"); query.Get("language") != "es-ES" {
		t.Fatalf("parámetros inesperados: %v", query)
	}
	if _, err := client.SeasonDetails(context.Background(), 1399, 1); err != nil {
		t.Fatalf("SeasonDetails desde caché: %v", err)
	}
	if calls := api.pathCount("/tv/1399/season/1"); calls != 1 {
		t.Fatalf("peticiones de temporada = %d, se esperaba 1 con caché", calls)
	}
}

func TestTVImages(t *testing.T) {
	api := newTestAPI(t)
	api.route("/tv/1399/images", jsonRoute(map[string]any{
		"posters": []map[string]any{
			{"file_path": "/poster.jpg", "iso_639_1": "es", "vote_average": 8},
		},
		"backdrops": []map[string]any{
			{"file_path": "/backdrop.jpg", "iso_639_1": "es-ES", "vote_average": 9},
		},
	}))
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")

	options, err := client.TVImages(context.Background(), 1399)
	if err != nil {
		t.Fatalf("TVImages: %v", err)
	}
	if len(options.Posters) != 1 ||
		options.Posters[0].URL != api.URL+"/img/w342/poster.jpg" ||
		options.Posters[0].FullURL != api.URL+"/img/w780/poster.jpg" {
		t.Fatalf("carteles inesperados: %+v", options.Posters)
	}
	if len(options.Backdrops) != 1 ||
		options.Backdrops[0].URL != api.URL+"/img/w780/backdrop.jpg" {
		t.Fatalf("fondos inesperados: %+v", options.Backdrops)
	}
	if _, err := client.TVImages(context.Background(), 1399); err != nil {
		t.Fatalf("TVImages desde caché: %v", err)
	}
	if calls := api.pathCount("/tv/1399/images"); calls != 1 {
		t.Fatalf("peticiones de imágenes = %d, se esperaba 1", calls)
	}
}

func TestEpisodeDetails(t *testing.T) {
	api := newTestAPI(t)
	api.route("/tv/1399/season/1/episode/1", jsonRoute(map[string]any{
		"name": "Winter Is Coming", "overview": "Sinopsis del episodio",
		"air_date": "2011-04-17", "still_path": "/still.jpg",
	}))
	api.route("/tv/1399/season/2/episode/1", jsonRoute(map[string]any{
		"name": "The North Remembers", "overview": "",
		"air_date": "2012-04-01", "still_path": "/still2.jpg",
	}))
	api.route("/tv/1399/season/2/episode/1/translations", jsonRoute(map[string]any{
		"translations": []map[string]any{
			{"iso_639_1": "es", "iso_3166_1": "MX", "data": map[string]any{"overview": "Sinopsis mexicana"}},
		},
	}))
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")

	episode, err := client.EpisodeDetails(context.Background(), 1399, 1, 1)
	if err != nil {
		t.Fatalf("EpisodeDetails: %v", err)
	}
	if episode.Name != "Winter Is Coming" || episode.Overview != "Sinopsis del episodio" ||
		episode.OverviewLanguage != "es-ES" || episode.AirYear != 2011 ||
		episode.StillURL != api.URL+"/img/w300/still.jpg" {
		t.Fatalf("episodio inesperado: %+v", episode)
	}

	fallback, err := client.EpisodeDetails(context.Background(), 1399, 2, 1)
	if err != nil {
		t.Fatalf("EpisodeDetails con respaldo: %v", err)
	}
	if fallback.Overview != "Sinopsis mexicana" || fallback.OverviewLanguage != "es-MX" {
		t.Fatalf("respaldo de episodio inesperado: %+v", fallback)
	}
}

func TestRateLimitRetry(t *testing.T) {
	api := newTestAPI(t)
	attempts := 0
	api.route("/search/movie", func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{
			{"id": 1, "title": "Matrix", "release_date": "1999-01-01"},
		}})
	})
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")

	if _, err := client.SearchMovies(context.Background(), "Matrix", 0); err != nil {
		t.Fatalf("el reintento tras 429 debe funcionar: %v", err)
	}
	if api.pathCount("/search/movie") != 2 {
		t.Fatalf("peticiones = %d, se esperaban 2", api.pathCount("/search/movie"))
	}
}

func TestRateLimitPersistente(t *testing.T) {
	api := newTestAPI(t)
	api.route("/search/movie", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")

	if _, err := client.SearchMovies(context.Background(), "Matrix", 0); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("error = %v, se esperaba ErrRateLimited", err)
	}
	if api.pathCount("/search/movie") != 2 {
		t.Fatalf("peticiones = %d, se esperaban 2 antes de rendirse", api.pathCount("/search/movie"))
	}
}

func TestCacheEvitaPeticionesNuevas(t *testing.T) {
	api := newTestAPI(t)
	api.route("/search/movie", jsonRoute(map[string]any{"results": []map[string]any{
		{"id": 1, "title": "Matrix", "release_date": "1999-01-01"},
	}}))
	cacheDir := t.TempDir()
	client := newAPIClient(t, api, cacheDir, nil, "clave-de-prueba")

	if _, err := client.SearchMovies(context.Background(), "Matrix", 1999); err != nil {
		t.Fatalf("primera búsqueda: %v", err)
	}
	if _, err := client.SearchMovies(context.Background(), "Matrix", 1999); err != nil {
		t.Fatalf("segunda búsqueda: %v", err)
	}
	if api.pathCount("/search/movie") != 1 {
		t.Fatalf("la segunda búsqueda hizo una petición: %d", api.pathCount("/search/movie"))
	}
}

func TestCacheKeyNoIncluyeAPIKey(t *testing.T) {
	api := newTestAPI(t)
	api.route("/search/movie", jsonRoute(map[string]any{"results": []map[string]any{
		{"id": 1, "title": "Matrix", "release_date": "1999-01-01"},
	}}))
	cacheDir := t.TempDir()

	first := newAPIClient(t, api, cacheDir, nil, "clave-uno")
	second := newAPIClient(t, api, cacheDir, nil, "clave-dos")

	if _, err := first.SearchMovies(context.Background(), "Matrix", 1999); err != nil {
		t.Fatalf("primera búsqueda: %v", err)
	}
	if _, err := second.SearchMovies(context.Background(), "Matrix", 1999); err != nil {
		t.Fatalf("segunda búsqueda: %v", err)
	}
	if api.pathCount("/search/movie") != 1 {
		t.Fatalf("cambiar la api_key invalidó la caché: %d peticiones", api.pathCount("/search/movie"))
	}
}

func TestCacheGuardaLaRespuestaCrudaCompleta(t *testing.T) {
	api := newTestAPI(t)
	api.route("/cache/completa", jsonRoute(map[string]any{
		"id":          7,
		"campo_nuevo": "visible",
	}))
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")
	ctx := context.Background()

	var first struct {
		ID int `json:"id"`
	}
	if err := client.getCachedJSON(ctx, "/cache/completa", nil, &first); err != nil {
		t.Fatalf("primera lectura: %v", err)
	}
	var second struct {
		ID         int    `json:"id"`
		CampoNuevo string `json:"campo_nuevo"`
	}
	if err := client.getCachedJSON(ctx, "/cache/completa", nil, &second); err != nil {
		t.Fatalf("lectura ampliada: %v", err)
	}
	if second.ID != 7 || second.CampoNuevo != "visible" {
		t.Fatalf("la caché perdió campos de la respuesta: %+v", second)
	}
	if api.pathCount("/cache/completa") != 1 {
		t.Fatalf("la lectura ampliada volvió a pedir TMDB: %d peticiones", api.pathCount("/cache/completa"))
	}
}

func TestCacheAPIRoundTripY404(t *testing.T) {
	api := newTestAPI(t)
	api.route("/cache/existe", jsonRoute(map[string]any{"id": 3}))
	api.route("/cache/falta", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")
	ctx := context.Background()

	var ok struct {
		ID int `json:"id"`
	}
	for i := 0; i < 2; i++ {
		if err := client.getCachedJSON(ctx, "/cache/existe", nil, &ok); err != nil {
			t.Fatalf("lectura %d: %v", i+1, err)
		}
	}
	if ok.ID != 3 || api.pathCount("/cache/existe") != 1 {
		t.Fatalf("round-trip inesperado: %+v, %d peticiones", ok, api.pathCount("/cache/existe"))
	}

	for i := 0; i < 2; i++ {
		if err := client.getCachedJSON(ctx, "/cache/falta", nil, &ok); !errors.Is(err, ErrNotFound) {
			t.Fatalf("404 %d: %v", i+1, err)
		}
	}
	if api.pathCount("/cache/falta") != 1 {
		t.Fatalf("el 404 no se cacheó: %d peticiones", api.pathCount("/cache/falta"))
	}
}

func TestCacheIgnoraLasClavesDelFormatoAntiguo(t *testing.T) {
	api := newTestAPI(t)
	api.route("/cache/version", jsonRoute(map[string]any{"id": 2, "title": "nuevo"}))
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")
	ctx := context.Background()

	// Clave del formato anterior: path + "?" + parámetros, sin versión.
	sum := sha256.Sum256([]byte("/cache/version?"))
	oldKey := hex.EncodeToString(sum[:])
	client.writeCache("api", oldKey, cachedAPIResponse{
		Found: true,
		Data:  json.RawMessage(`{"id":1,"title":"viejo"}`),
	})

	var got struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}
	if err := client.getCachedJSON(ctx, "/cache/version", nil, &got); err != nil {
		t.Fatalf("lectura: %v", err)
	}
	if got.ID != 2 || got.Title != "nuevo" {
		t.Fatalf("se usó una entrada antigua: %+v", got)
	}
	if api.pathCount("/cache/version") != 1 {
		t.Fatalf("la clave antigua evitó la petición nueva: %d", api.pathCount("/cache/version"))
	}
}

func TestCache404Negativo(t *testing.T) {
	api := newTestAPI(t)
	api.route("/search/movie", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")

	if _, err := client.SearchMovies(context.Background(), "No existe", 2020); !errors.Is(err, ErrNotFound) {
		t.Fatalf("primera llamada: %v", err)
	}
	if _, err := client.SearchMovies(context.Background(), "No existe", 2020); !errors.Is(err, ErrNotFound) {
		t.Fatalf("segunda llamada: %v", err)
	}
	if api.pathCount("/search/movie") != 1 {
		t.Fatalf("el 404 no se cacheó: %d peticiones", api.pathCount("/search/movie"))
	}
}

func TestLimitadorDePeticiones(t *testing.T) {
	api := newTestAPI(t)
	api.route("/search/movie", jsonRoute(map[string]any{"results": []map[string]any{
		{"id": 1, "title": "Uno"}, {"id": 2, "title": "Dos"},
	}}))
	client := newAPIClient(t, api, t.TempDir(), nil, "clave-de-prueba")

	if _, err := client.SearchMovies(context.Background(), "Uno", 0); err != nil {
		t.Fatalf("primera: %v", err)
	}
	if _, err := client.SearchMovies(context.Background(), "Dos", 0); err != nil {
		t.Fatalf("segunda: %v", err)
	}
	times := api.requestTimes()
	if len(times) != 2 {
		t.Fatalf("peticiones = %d", len(times))
	}
	if gap := times[1].Sub(times[0]); gap < 40*time.Millisecond {
		t.Fatalf("las peticiones se hicieron con %s de diferencia; el limitador exige ~50 ms", gap)
	}
}
