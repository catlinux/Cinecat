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
	"sort"
	"strconv"
	"strings"
	"sync"
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

	// minRequestInterval mantiene el límite de TMDB de ~20 peticiones/s.
	minRequestInterval = 50 * time.Millisecond
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

// Candidate es un resultado de búsqueda que el usuario puede elegir.
type Candidate struct {
	TMDBID        int     `json:"tmdb_id"`
	Title         string  `json:"title"`
	OriginalTitle string  `json:"original_title"`
	Year          int     `json:"year"`
	Overview      string  `json:"overview"`
	PosterURL     string  `json:"poster_url"`
	Popularity    float64 `json:"popularity"`
	VoteCount     int     `json:"vote_count"`
}

// CollectionRef identifica la saga a la que pertenece una película.
type CollectionRef struct {
	TMDBID      int    `json:"tmdb_id"`
	Name        string `json:"name"`
	PosterURL   string `json:"poster_url"`
	BackdropURL string `json:"backdrop_url"`
}

// Details es la ficha completa de una película o serie.
type Details struct {
	TMDBID           int            `json:"tmdb_id"`
	IMDbID           string         `json:"imdb_id"`
	Title            string         `json:"title"`
	OriginalTitle    string         `json:"original_title"`
	Year             int            `json:"year"`
	Tagline          string         `json:"tagline"`
	Overview         string         `json:"overview"`
	OverviewLanguage string         `json:"overview_language"`
	Genres           []string       `json:"genres"`
	RuntimeMinutes   int            `json:"runtime_minutes"`
	Rating           float64        `json:"rating"`
	Directors        []string       `json:"directors"`
	Cast             []string       `json:"cast"`
	PosterURL        string         `json:"poster_url"`
	BackdropURL      string         `json:"backdrop_url"`
	Collection       *CollectionRef `json:"collection,omitempty"`
}

// Episode es la ficha de un episodio concreto.
type Episode struct {
	Name             string `json:"name"`
	Overview         string `json:"overview"`
	OverviewLanguage string `json:"overview_language"`
	AirYear          int    `json:"air_year"`
	StillURL         string `json:"still_url"`
}

// Season agrupa los datos de una temporada y todos sus episodios.
type Season struct {
	Name             string          `json:"name"`
	Overview         string          `json:"overview"`
	OverviewLanguage string          `json:"overview_language"`
	PosterURL        string          `json:"poster_url"`
	AirYear          int             `json:"air_year"`
	Episodes         []SeasonEpisode `json:"episodes"`
}

// SeasonEpisode es un episodio dentro de la ficha de su temporada.
type SeasonEpisode struct {
	Number   int    `json:"number"`
	Name     string `json:"name"`
	Overview string `json:"overview"`
	StillURL string `json:"still_url"`
	AirYear  int    `json:"air_year"`
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
	// RetryDelay es la espera antes de reintentar un 429; 2 s por defecto.
	RetryDelay time.Duration
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
	retryDelay time.Duration

	rateMu      sync.Mutex
	lastRequest time.Time
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

	retryDelay := opts.RetryDelay
	if retryDelay <= 0 {
		retryDelay = 2 * time.Second
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
		retryDelay: retryDelay,
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
	Width       int     `json:"width"`
	Height      int     `json:"height"`
}

type imagesResponse struct {
	Posters   []imageInfo `json:"posters"`
	Backdrops []imageInfo `json:"backdrops"`
}

// ImageOption es una imagen disponible para elegir como cartel o fondo.
type ImageOption struct {
	URL         string  `json:"url"`
	FullURL     string  `json:"full_url"`
	Language    string  `json:"language"`
	VoteAverage float64 `json:"vote_average"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
}

// ImageOptions agrupa los carteles y fondos disponibles de una película.
type ImageOptions struct {
	Posters   []ImageOption `json:"posters"`
	Backdrops []ImageOption `json:"backdrops"`
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

type movieSearchResponse struct {
	Results []struct {
		ID            int     `json:"id"`
		Title         string  `json:"title"`
		OriginalTitle string  `json:"original_title"`
		ReleaseDate   string  `json:"release_date"`
		Overview      string  `json:"overview"`
		PosterPath    string  `json:"poster_path"`
		Popularity    float64 `json:"popularity"`
		VoteCount     int     `json:"vote_count"`
	} `json:"results"`
}

type tvSearchResponse struct {
	Results []struct {
		ID           int     `json:"id"`
		Name         string  `json:"name"`
		OriginalName string  `json:"original_name"`
		FirstAirDate string  `json:"first_air_date"`
		Overview     string  `json:"overview"`
		PosterPath   string  `json:"poster_path"`
		Popularity   float64 `json:"popularity"`
		VoteCount    int     `json:"vote_count"`
	} `json:"results"`
}

type collectionRef struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	PosterPath   string `json:"poster_path"`
	BackdropPath string `json:"backdrop_path"`
}

type collectionRefResponse struct {
	BelongsToCollection *collectionRef `json:"belongs_to_collection"`
}

type collectionResponse struct {
	Parts []struct {
		ID            int     `json:"id"`
		Title         string  `json:"title"`
		OriginalTitle string  `json:"original_title"`
		ReleaseDate   string  `json:"release_date"`
		Overview      string  `json:"overview"`
		PosterPath    string  `json:"poster_path"`
		Popularity    float64 `json:"popularity"`
		VoteCount     int     `json:"vote_count"`
	} `json:"parts"`
}

type castMember struct {
	Name  string `json:"name"`
	Order int    `json:"order"`
}

type creditsResponse struct {
	Cast []castMember `json:"cast"`
	Crew []struct {
		Name string `json:"name"`
		Job  string `json:"job"`
	} `json:"crew"`
}

type genre struct {
	Name string `json:"name"`
}

type movieDetailsResponse struct {
	ID                  int             `json:"id"`
	IMDbID              string          `json:"imdb_id"`
	Title               string          `json:"title"`
	OriginalTitle       string          `json:"original_title"`
	ReleaseDate         string          `json:"release_date"`
	Tagline             string          `json:"tagline"`
	Overview            string          `json:"overview"`
	Runtime             int             `json:"runtime"`
	VoteAverage         float64         `json:"vote_average"`
	Genres              []genre         `json:"genres"`
	Credits             creditsResponse `json:"credits"`
	Images              imagesResponse  `json:"images"`
	PosterPath          string          `json:"poster_path"`
	BackdropPath        string          `json:"backdrop_path"`
	BelongsToCollection *collectionRef  `json:"belongs_to_collection"`
}

type tvDetailsResponse struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	OriginalName string  `json:"original_name"`
	FirstAirDate string  `json:"first_air_date"`
	Tagline      string  `json:"tagline"`
	Overview     string  `json:"overview"`
	VoteAverage  float64 `json:"vote_average"`
	Genres       []genre `json:"genres"`
	CreatedBy    []struct {
		Name string `json:"name"`
	} `json:"created_by"`
	Credits      creditsResponse `json:"credits"`
	Images       imagesResponse  `json:"images"`
	PosterPath   string          `json:"poster_path"`
	BackdropPath string          `json:"backdrop_path"`
}

type episodeDetailsResponse struct {
	Name      string `json:"name"`
	Overview  string `json:"overview"`
	AirDate   string `json:"air_date"`
	StillPath string `json:"still_path"`
}

type seasonDetailsResponse struct {
	Name       string `json:"name"`
	Overview   string `json:"overview"`
	AirDate    string `json:"air_date"`
	PosterPath string `json:"poster_path"`
	Episodes   []struct {
		EpisodeNumber int    `json:"episode_number"`
		Name          string `json:"name"`
		Overview      string `json:"overview"`
		AirDate       string `json:"air_date"`
		StillPath     string `json:"still_path"`
	} `json:"episodes"`
}

type translationItem struct {
	ISO6391  string `json:"iso_639_1"`
	ISO31661 string `json:"iso_3166_1"`
	Data     struct {
		Overview string `json:"overview"`
	} `json:"data"`
}

type translationsResponse struct {
	Translations []translationItem `json:"translations"`
}

// SearchMovies busca películas por título y año opcional, en es-ES.
func (c *Client) SearchMovies(ctx context.Context, query string, year int) ([]Candidate, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("metadata: el título no puede estar vacío")
	}

	params := url.Values{}
	params.Set("language", defaultLanguage)
	params.Set("query", query)
	params.Set("include_adult", "false")
	if year > 0 {
		params.Set("year", strconv.Itoa(year))
	}

	var response movieSearchResponse
	if err := c.getCachedJSON(ctx, "/search/movie", params, &response); err != nil {
		return nil, err
	}
	if len(response.Results) == 0 {
		return nil, ErrNotFound
	}

	candidates := make([]Candidate, 0, len(response.Results))
	for _, result := range response.Results {
		if len(candidates) == 10 {
			break
		}
		candidates = append(candidates, Candidate{
			TMDBID:        result.ID,
			Title:         result.Title,
			OriginalTitle: result.OriginalTitle,
			Year:          releaseYear(result.ReleaseDate),
			Overview:      result.Overview,
			PosterURL:     c.imageURLSize(result.PosterPath, "w185"),
			Popularity:    result.Popularity,
			VoteCount:     result.VoteCount,
		})
	}
	return candidates, nil
}

// MovieCollection devuelve las películas de la colección (saga) a la que
// pertenece la película, ordenadas por fecha de estreno; las que no tienen
// fecha se descartan. Si no pertenece a ninguna, devuelve una lista vacía.
func (c *Client) MovieCollection(ctx context.Context, movieID int) ([]Candidate, error) {
	params := url.Values{}
	params.Set("language", defaultLanguage)

	// Misma petición que MovieDetails pero sin append_to_response: la ficha
	// básica basta y queda en caché aparte.
	var ref collectionRefResponse
	if err := c.getCachedJSON(ctx, "/movie/"+strconv.Itoa(movieID), params, &ref); err != nil {
		return nil, err
	}
	if ref.BelongsToCollection == nil || ref.BelongsToCollection.ID == 0 {
		return nil, nil
	}

	var collection collectionResponse
	if err := c.getCachedJSON(ctx, "/collection/"+strconv.Itoa(ref.BelongsToCollection.ID), params, &collection); err != nil {
		return nil, err
	}
	parts := collection.Parts[:0:0]
	for _, part := range collection.Parts {
		if part.ReleaseDate != "" {
			parts = append(parts, part)
		}
	}
	sort.SliceStable(parts, func(i, j int) bool { return parts[i].ReleaseDate < parts[j].ReleaseDate })

	candidates := make([]Candidate, 0, len(parts))
	for _, part := range parts {
		candidates = append(candidates, Candidate{
			TMDBID:        part.ID,
			Title:         part.Title,
			OriginalTitle: part.OriginalTitle,
			Year:          releaseYear(part.ReleaseDate),
			Overview:      part.Overview,
			PosterURL:     c.imageURLSize(part.PosterPath, "w185"),
			Popularity:    part.Popularity,
			VoteCount:     part.VoteCount,
		})
	}
	return candidates, nil
}

// MovieImages devuelve los carteles y fondos disponibles de una película.
func (c *Client) MovieImages(ctx context.Context, tmdbID int) (*ImageOptions, error) {
	return c.images(ctx, "/movie/"+strconv.Itoa(tmdbID)+"/images")
}

// TVImages devuelve los carteles y fondos disponibles de una serie.
func (c *Client) TVImages(ctx context.Context, tvID int) (*ImageOptions, error) {
	return c.images(ctx, "/tv/"+strconv.Itoa(tvID)+"/images")
}

func (c *Client) images(ctx context.Context, path string) (*ImageOptions, error) {
	params := url.Values{}
	params.Set("include_image_language", imageLanguages)

	var images imagesResponse
	if err := c.getCachedJSON(ctx, path, params, &images); err != nil {
		return nil, err
	}

	return &ImageOptions{
		Posters:   c.imageOptions(images.Posters, "w342", "w780"),
		Backdrops: c.imageOptions(images.Backdrops, "w780", "w1280"),
	}, nil
}

func (c *Client) imageOptions(images []imageInfo, listSize, fullSize string) []ImageOption {
	sorted := make([]imageInfo, 0, len(images))
	for _, image := range images {
		if strings.TrimSpace(image.FilePath) != "" {
			sorted = append(sorted, image)
		}
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		leftPriority := imageLanguagePriority(posterLanguage(sorted[i]))
		rightPriority := imageLanguagePriority(posterLanguage(sorted[j]))
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		return sorted[i].VoteAverage > sorted[j].VoteAverage
	})

	if len(sorted) > 40 {
		sorted = sorted[:40]
	}
	options := make([]ImageOption, 0, len(sorted))
	for _, image := range sorted {
		options = append(options, ImageOption{
			URL:         c.imageURLSize(image.FilePath, listSize),
			FullURL:     c.imageURLSize(image.FilePath, fullSize),
			Language:    imageLanguage(image),
			VoteAverage: image.VoteAverage,
			Width:       image.Width,
			Height:      image.Height,
		})
	}
	return options
}

func imageLanguagePriority(language string) int {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "es-es":
		return 0
	case "es":
		return 1
	case "en":
		return 2
	case "":
		return 3
	default:
		return 4
	}
}

// SearchTV busca series por nombre y año opcional, en es-ES.
func (c *Client) SearchTV(ctx context.Context, query string, year int) ([]Candidate, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, errors.New("metadata: el título no puede estar vacío")
	}

	params := url.Values{}
	params.Set("language", defaultLanguage)
	params.Set("query", query)
	params.Set("include_adult", "false")
	if year > 0 {
		params.Set("first_air_date_year", strconv.Itoa(year))
	}

	var response tvSearchResponse
	if err := c.getCachedJSON(ctx, "/search/tv", params, &response); err != nil {
		return nil, err
	}
	if len(response.Results) == 0 {
		return nil, ErrNotFound
	}

	candidates := make([]Candidate, 0, len(response.Results))
	for _, result := range response.Results {
		if len(candidates) == 10 {
			break
		}
		candidates = append(candidates, Candidate{
			TMDBID:        result.ID,
			Title:         result.Name,
			OriginalTitle: result.OriginalName,
			Year:          releaseYear(result.FirstAirDate),
			Overview:      result.Overview,
			PosterURL:     c.imageURLSize(result.PosterPath, "w185"),
			Popularity:    result.Popularity,
			VoteCount:     result.VoteCount,
		})
	}
	return candidates, nil
}

// MovieDetails obtiene la ficha completa de una película en es-ES.
func (c *Client) MovieDetails(ctx context.Context, id int) (*Details, error) {
	params := url.Values{}
	params.Set("language", defaultLanguage)
	params.Set("append_to_response", "credits,images")
	params.Set("include_image_language", "es,null,en")

	var response movieDetailsResponse
	path := "/movie/" + strconv.Itoa(id)
	if err := c.getCachedJSON(ctx, path, params, &response); err != nil {
		return nil, err
	}

	details := &Details{
		TMDBID:         response.ID,
		IMDbID:         response.IMDbID,
		Title:          response.Title,
		OriginalTitle:  response.OriginalTitle,
		Year:           releaseYear(response.ReleaseDate),
		Tagline:        response.Tagline,
		Overview:       response.Overview,
		RuntimeMinutes: response.Runtime,
		Rating:         response.VoteAverage,
		Genres:         genreNames(response.Genres),
		Directors:      directorsFromCrew(response.Credits.Crew),
		Cast:           castNames(response.Credits.Cast),
		PosterURL:      c.posterURL(response.Images.Posters, response.PosterPath),
		BackdropURL:    c.imageURLSize(response.BackdropPath, "w1280"),
	}
	if collection := response.BelongsToCollection; collection != nil && collection.ID > 0 {
		details.Collection = &CollectionRef{
			TMDBID:      collection.ID,
			Name:        collection.Name,
			PosterURL:   c.imageURLSize(collection.PosterPath, "w342"),
			BackdropURL: c.imageURLSize(collection.BackdropPath, "w1280"),
		}
	}
	if details.Overview != "" {
		details.OverviewLanguage = defaultLanguage
	} else if overview, language := c.translatedOverview(ctx, path+"/translations"); overview != "" {
		details.Overview = overview
		details.OverviewLanguage = language
	}
	return details, nil
}

// TVDetails obtiene la ficha completa de una serie en es-ES.
func (c *Client) TVDetails(ctx context.Context, id int) (*Details, error) {
	params := url.Values{}
	params.Set("language", defaultLanguage)
	params.Set("append_to_response", "credits,images")
	params.Set("include_image_language", "es,null,en")

	var response tvDetailsResponse
	path := "/tv/" + strconv.Itoa(id)
	if err := c.getCachedJSON(ctx, path, params, &response); err != nil {
		return nil, err
	}

	directors := make([]string, 0, len(response.CreatedBy))
	for _, creator := range response.CreatedBy {
		if strings.TrimSpace(creator.Name) != "" {
			directors = append(directors, creator.Name)
		}
	}

	details := &Details{
		TMDBID:         response.ID,
		Title:          response.Name,
		OriginalTitle:  response.OriginalName,
		Year:           releaseYear(response.FirstAirDate),
		Tagline:        response.Tagline,
		Overview:       response.Overview,
		RuntimeMinutes: 0,
		Rating:         response.VoteAverage,
		Genres:         genreNames(response.Genres),
		Directors:      directors,
		Cast:           castNames(response.Credits.Cast),
		PosterURL:      c.posterURL(response.Images.Posters, response.PosterPath),
		BackdropURL:    c.imageURLSize(response.BackdropPath, "w1280"),
	}
	if details.Overview != "" {
		details.OverviewLanguage = defaultLanguage
	} else if overview, language := c.translatedOverview(ctx, path+"/translations"); overview != "" {
		details.Overview = overview
		details.OverviewLanguage = language
	}
	return details, nil
}

// SeasonDetails obtiene una temporada y todos sus episodios en es-ES con una
// sola petición.
func (c *Client) SeasonDetails(ctx context.Context, tvID, season int) (*Season, error) {
	params := url.Values{}
	params.Set("language", defaultLanguage)

	path := fmt.Sprintf("/tv/%d/season/%d", tvID, season)
	var response seasonDetailsResponse
	if err := c.getCachedJSON(ctx, path, params, &response); err != nil {
		return nil, err
	}

	result := &Season{
		Name:      response.Name,
		Overview:  response.Overview,
		PosterURL: c.imageURLSize(response.PosterPath, "w342"),
		AirYear:   releaseYear(response.AirDate),
	}
	if result.Overview != "" {
		result.OverviewLanguage = defaultLanguage
	}
	for _, episode := range response.Episodes {
		result.Episodes = append(result.Episodes, SeasonEpisode{
			Number:   episode.EpisodeNumber,
			Name:     episode.Name,
			Overview: episode.Overview,
			StillURL: c.imageURLSize(episode.StillPath, "w300"),
			AirYear:  releaseYear(episode.AirDate),
		})
	}
	return result, nil
}

// EpisodeDetails obtiene la ficha de un episodio en es-ES.
func (c *Client) EpisodeDetails(ctx context.Context, tvID, season, episode int) (*Episode, error) {
	params := url.Values{}
	params.Set("language", defaultLanguage)

	path := fmt.Sprintf("/tv/%d/season/%d/episode/%d", tvID, season, episode)
	var response episodeDetailsResponse
	if err := c.getCachedJSON(ctx, path, params, &response); err != nil {
		return nil, err
	}

	result := &Episode{
		Name:     response.Name,
		Overview: response.Overview,
		AirYear:  releaseYear(response.AirDate),
		StillURL: c.imageURLSize(response.StillPath, "w300"),
	}
	if result.Overview != "" {
		result.OverviewLanguage = defaultLanguage
	} else if overview, language := c.translatedOverview(ctx, path+"/translations"); overview != "" {
		result.Overview = overview
		result.OverviewLanguage = language
	}
	return result, nil
}

func genreNames(genres []genre) []string {
	names := make([]string, 0, len(genres))
	for _, item := range genres {
		if strings.TrimSpace(item.Name) != "" {
			names = append(names, item.Name)
		}
	}
	return names
}

func directorsFromCrew(crew []struct {
	Name string `json:"name"`
	Job  string `json:"job"`
}) []string {
	var directors []string
	for _, member := range crew {
		if member.Job == "Director" && strings.TrimSpace(member.Name) != "" {
			directors = append(directors, member.Name)
		}
	}
	return directors
}

func castNames(cast []castMember) []string {
	sorted := append([]castMember(nil), cast...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Order < sorted[j].Order })

	names := make([]string, 0, 8)
	for _, member := range sorted {
		if len(names) == 8 {
			break
		}
		if strings.TrimSpace(member.Name) != "" {
			names = append(names, member.Name)
		}
	}
	return names
}

func (c *Client) posterURL(posters []imageInfo, fallbackPath string) string {
	posterPath, _ := pickPoster(posters)
	if posterPath == "" {
		posterPath = fallbackPath
	}
	return c.imageURL(posterPath)
}

// translatedOverview pide las traducciones y aplica es-MX -> cualquier es ->
// en-US. Si la petición falla, se devuelve vacío sin romper la ficha.
func (c *Client) translatedOverview(ctx context.Context, path string) (string, string) {
	var response translationsResponse
	if err := c.getCachedJSON(ctx, path, nil, &response); err != nil {
		return "", ""
	}

	for _, translation := range response.Translations {
		if translation.ISO6391 == "es" && translation.ISO31661 == "MX" &&
			strings.TrimSpace(translation.Data.Overview) != "" {
			return translation.Data.Overview, "es-MX"
		}
	}
	for _, translation := range response.Translations {
		if translation.ISO6391 == "es" && strings.TrimSpace(translation.Data.Overview) != "" {
			return translation.Data.Overview, "es"
		}
	}
	for _, translation := range response.Translations {
		if translation.ISO6391 == "en" && translation.ISO31661 == "US" &&
			strings.TrimSpace(translation.Data.Overview) != "" {
			return translation.Data.Overview, "en-US"
		}
	}
	return "", ""
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

func imageLanguage(image imageInfo) string {
	if image.Language == nil {
		return ""
	}
	return strings.TrimSpace(*image.Language)
}

func (c *Client) imageURL(filePath string) string {
	return c.imageURLSize(filePath, c.posterSize)
}

func (c *Client) imageURLSize(filePath, size string) string {
	if filePath == "" {
		return ""
	}
	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		return filePath
	}
	return strings.TrimRight(c.imageBase.String(), "/") + "/" + size + "/" + strings.TrimLeft(filePath, "/")
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

// getJSON hace una petición GET a TMDB con el limitador del cliente y un
// reintento tras 429. Nunca incluye la URL completa en los errores, porque
// lleva la clave de API.
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

	for attempt := 0; attempt < 2; attempt++ {
		if err := c.waitForRateLimit(ctx); err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return errors.New("metadata: no se pudo preparar la petición a TMDB")
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("metadata: error de red al consultar %s", path)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			if attempt == 1 {
				return ErrRateLimited
			}
			if err := sleepContext(ctx, c.retryDelay); err != nil {
				return err
			}
			continue
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return ErrInvalidAPIKey
		case http.StatusNotFound:
			return ErrNotFound
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return fmt.Errorf("metadata: TMDB respondió %d en %s", resp.StatusCode, path)
		}

		if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(dst); err != nil {
			return fmt.Errorf("metadata: respuesta no válida de %s", path)
		}
		return nil
	}
	return ErrRateLimited
}

func (c *Client) waitForRateLimit(ctx context.Context) error {
	c.rateMu.Lock()
	defer c.rateMu.Unlock()

	now := time.Now()
	if !c.lastRequest.IsZero() {
		if wait := minRequestInterval - now.Sub(c.lastRequest); wait > 0 {
			if err := sleepContext(ctx, wait); err != nil {
				return err
			}
		}
	}
	c.lastRequest = time.Now()
	return nil
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type cachedAPIResponse struct {
	Found bool            `json:"found"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// getCachedJSON cachea en disco cada petición nueva. La clave excluye api_key
// para que cambiarla no invalide la caché. Los 404 también se recuerdan.
func (c *Client) getCachedJSON(ctx context.Context, path string, params url.Values, dst any) error {
	key := requestCacheKey(path, params)
	var cached cachedAPIResponse
	if c.readCache("api", key, &cached) {
		if !cached.Found {
			return ErrNotFound
		}
		if len(cached.Data) > 0 && json.Unmarshal(cached.Data, dst) == nil {
			return nil
		}
	}

	if err := c.getJSON(ctx, path, params, dst); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.writeCache("api", key, cachedAPIResponse{Found: false})
		}
		return err
	}

	data, err := json.Marshal(dst)
	if err != nil {
		return nil
	}
	c.writeCache("api", key, cachedAPIResponse{Found: true, Data: data})
	return nil
}

func requestCacheKey(path string, params url.Values) string {
	filtered := url.Values{}
	for key, values := range params {
		if key == "api_key" {
			continue
		}
		for _, value := range values {
			filtered.Add(key, value)
		}
	}
	sum := sha256.Sum256([]byte(path + "?" + filtered.Encode()))
	return hex.EncodeToString(sum[:])
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
