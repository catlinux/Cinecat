package metadata

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestOMDbClient(t *testing.T, baseURL string, mutate func(*OMDbOptions)) *OMDbClient {
	t.Helper()
	opts := OMDbOptions{
		APIKey:   "clave-de-prueba",
		BaseURL:  baseURL,
		CacheDir: t.TempDir(),
	}
	if mutate != nil {
		mutate(&opts)
	}
	client, err := NewOMDbClient(opts)
	if err != nil {
		t.Fatalf("NewOMDbClient: %v", err)
	}
	return client
}

type omdbFake struct {
	*httptest.Server

	mu        sync.Mutex
	calls     int
	lastQuery url.Values
	body      string
	status    int
	delay     time.Duration
}

func newOMDbFake(t *testing.T, body string) *omdbFake {
	t.Helper()
	fake := &omdbFake{body: body, status: http.StatusOK}
	fake.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.mu.Lock()
		fake.calls++
		fake.lastQuery = r.URL.Query()
		status, delay, body := fake.status, fake.delay, fake.body
		fake.mu.Unlock()

		if delay > 0 {
			time.Sleep(delay)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = fmt.Fprint(w, body)
	}))
	t.Cleanup(fake.Close)
	return fake
}

func (f *omdbFake) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func TestOMDbSearchConResultado(t *testing.T) {
	fake := newOMDbFake(t, `{
		"Title": "The Matrix",
		"Year": "1999",
		"Plot": "Un hacker descubre la verdad.",
		"Poster": "https://images.example/matrix.jpg",
		"imdbID": "tt0133093",
		"Type": "movie",
		"Response": "True"
	}`)
	client := newTestOMDbClient(t, fake.URL, nil)

	result, err := client.Search(context.Background(), "Matrix", 1999)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if result.Title != "The Matrix" || result.Year != 1999 ||
		result.Plot != "Un hacker descubre la verdad." ||
		result.PosterURL != "https://images.example/matrix.jpg" ||
		result.IMDbID != "tt0133093" || result.Type != "movie" {
		t.Fatalf("resultado inesperado: %+v", result)
	}

	query := fake.lastQuery
	if query.Get("t") != "Matrix" || query.Get("y") != "1999" ||
		query.Get("type") != "movie" || query.Get("apikey") != "clave-de-prueba" {
		t.Fatalf("parámetros inesperados: %v", query)
	}

	// La segunda consulta sale de la caché y no gasta otra petición.
	if _, err := client.Search(context.Background(), "Matrix", 1999); err != nil {
		t.Fatalf("segunda búsqueda: %v", err)
	}
	if fake.callCount() != 1 {
		t.Fatalf("peticiones = %d, se esperaba 1", fake.callCount())
	}
}

func TestOMDbSearchPosterNA(t *testing.T) {
	fake := newOMDbFake(t, `{
		"Title": "Sin cartel",
		"Year": "2020",
		"Plot": "N/A",
		"Poster": "N/A",
		"imdbID": "tt0000001",
		"Type": "movie",
		"Response": "True"
	}`)
	client := newTestOMDbClient(t, fake.URL, nil)

	result, err := client.Search(context.Background(), "Sin cartel", 2020)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if result.PosterURL != "" || result.Plot != "" {
		t.Fatalf("N/A debe quedar como cadena vacía: %+v", result)
	}
}

func TestOMDbSearchNoEncontrado(t *testing.T) {
	fake := newOMDbFake(t, `{"Response":"False","Error":"Movie not found!"}`)
	client := newTestOMDbClient(t, fake.URL, nil)

	_, err := client.Search(context.Background(), "No existe", 2020)
	if !errors.Is(err, ErrOMDbNotFound) {
		t.Fatalf("error = %v, se esperaba ErrOMDbNotFound", err)
	}
	if !strings.Contains(err.Error(), "Movie not found!") {
		t.Fatalf("el motivo de OMDb debería conservarse: %v", err)
	}

	// La caché negativa evita repetir la petición.
	if _, err := client.Search(context.Background(), "No existe", 2020); !errors.Is(err, ErrOMDbNotFound) {
		t.Fatalf("segunda búsqueda: %v", err)
	}
	if fake.callCount() != 1 {
		t.Fatalf("peticiones = %d, se esperaba 1", fake.callCount())
	}
}

func TestOMDbSearchErrorDeRed(t *testing.T) {
	fake := newOMDbFake(t, `{"Response":"True"}`)
	baseURL := fake.URL
	fake.Close()

	client := newTestOMDbClient(t, baseURL, nil)
	if _, err := client.Search(context.Background(), "Matrix", 1999); err == nil {
		t.Fatal("se esperaba error de red")
	}
}

func TestOMDbSearchTimeout(t *testing.T) {
	fake := newOMDbFake(t, `{"Response":"True"}`)
	fake.mu.Lock()
	fake.delay = 200 * time.Millisecond
	fake.mu.Unlock()

	client := newTestOMDbClient(t, fake.URL, func(opts *OMDbOptions) {
		opts.HTTPClient = &http.Client{Timeout: 20 * time.Millisecond}
	})
	if _, err := client.Search(context.Background(), "Matrix", 1999); err == nil {
		t.Fatal("se esperaba timeout")
	}
}

func TestOMDbSinClave(t *testing.T) {
	if _, err := NewOMDbClient(OMDbOptions{CacheDir: t.TempDir()}); !errors.Is(err, ErrMissingOMDbAPIKey) {
		t.Fatalf("error = %v, se esperaba ErrMissingOMDbAPIKey", err)
	}
}
