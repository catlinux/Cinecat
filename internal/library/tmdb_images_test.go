package library

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func tmdbImageTestGet(t *testing.T, url, token string) (int, http.Header, []byte) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("crear petición: %v", err)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("leer respuesta: %v", err)
	}
	return response.StatusCode, response.Header, body
}

func TestTMDBImageEndpointCacheaYReutiliza(t *testing.T) {
	image := []byte("imagen-de-prueba")
	var calls atomic.Int64
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/w342/abc.jpg" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(image)
	}))
	defer fake.Close()

	ts := newLibraryHTTPTestServer(t)
	cacheDir := filepath.Join(t.TempDir(), "cache", "images")
	ts.api.WithTMDBImages(cacheDir).WithTMDBImageBaseURL(fake.URL)
	url := ts.URL + "/api/v1/images/tmdb/w342/abc.jpg"

	status, headers, body := tmdbImageTestGet(t, url, ts.token)
	if status != http.StatusOK || string(body) != string(image) {
		t.Fatalf("primera imagen: código = %d, cuerpo = %q", status, body)
	}
	if headers.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("Content-Type = %q", headers.Get("Content-Type"))
	}
	if headers.Get("Cache-Control") != "private, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q", headers.Get("Cache-Control"))
	}
	cachePath := filepath.Join(cacheDir, "w342", "abc.jpg")
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("la imagen no se guardó: %v", err)
	}

	status, _, body = tmdbImageTestGet(t, url, ts.token)
	if status != http.StatusOK || string(body) != string(image) {
		t.Fatalf("segunda imagen: código = %d, cuerpo = %q", status, body)
	}
	if calls.Load() != 1 {
		t.Fatalf("TMDB recibió %d peticiones, se esperaba una", calls.Load())
	}
}

func TestTMDBImageEndpointSinCacheDescargaSiempre(t *testing.T) {
	image := []byte("sin-cache")
	var calls atomic.Int64
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(image)
	}))
	defer fake.Close()

	ts := newLibraryHTTPTestServer(t)
	ts.api.WithTMDBImages("").WithTMDBImageBaseURL(fake.URL)
	url := ts.URL + "/api/v1/images/tmdb/w185/abc.png"

	for i := 0; i < 2; i++ {
		status, _, body := tmdbImageTestGet(t, url, ts.token)
		if status != http.StatusOK || string(body) != string(image) {
			t.Fatalf("petición %d: código = %d, cuerpo = %q", i+1, status, body)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("TMDB recibió %d peticiones, se esperaban dos", calls.Load())
	}
}

func TestTMDBImageValidation(t *testing.T) {
	for _, size := range []string{"w92", "w154", "w185", "w300", "w342", "w500", "w780", "w1280", "h632", "original"} {
		if !validTMDBImage(size, "abc.jpg") {
			t.Fatalf("tamaño válido rechazado: %s", size)
		}
	}
	for _, path := range []string{
		"../abc.jpg",
		"a/b.jpg",
		"abc.txt",
		"abc.JPG",
		"abc.jpg.exe",
		strings.Repeat("a", 101) + ".jpg",
	} {
		if validTMDBImage("w342", path) {
			t.Fatalf("archivo inválido aceptado: %s", path)
		}
	}
}

func TestTMDBImageEndpointRechazaRutasInvalidas(t *testing.T) {
	var calls atomic.Int64
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("no-deberia-llegar"))
	}))
	defer fake.Close()

	ts := newLibraryHTTPTestServer(t)
	ts.api.WithTMDBImages(filepath.Join(t.TempDir(), "images")).WithTMDBImageBaseURL(fake.URL)
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	paths := []string{
		"/w999/abc.jpg",
		"/w342/abc.txt",
		"/w342/" + strings.Repeat("a", 101) + ".jpg",
		"/w342/..%2Fsecreto.jpg",
		"/w342/%2e%2e/secreto.jpg",
		"/w342/a%2Fb.jpg",
	}
	for _, path := range paths {
		request, err := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/images/tmdb"+path, nil)
		if err != nil {
			t.Fatalf("crear petición %q: %v", path, err)
		}
		request.Header.Set("Authorization", "Bearer "+ts.token)
		response, err := client.Do(request)
		if err != nil {
			t.Fatalf("GET %q: %v", path, err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusNotFound {
			t.Fatalf("ruta %q: código = %d, se esperaba 404", path, response.StatusCode)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("TMDB recibió %d peticiones con rutas inválidas", calls.Load())
	}
}

func TestTMDBImageEndpointErroresDeTMDB(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		contentType string
		body        []byte
	}{
		{name: "404", status: http.StatusNotFound, contentType: "text/html", body: []byte("no")},
		{name: "500", status: http.StatusInternalServerError, contentType: "image/jpeg", body: []byte("error")},
		{name: "tipo no imagen", status: http.StatusOK, contentType: "text/html", body: []byte("html")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(tt.status)
				_, _ = w.Write(tt.body)
			}))
			defer fake.Close()

			ts := newLibraryHTTPTestServer(t)
			cacheDir := filepath.Join(t.TempDir(), "images")
			ts.api.WithTMDBImages(cacheDir).WithTMDBImageBaseURL(fake.URL)
			url := ts.URL + "/api/v1/images/tmdb/w342/error.jpg"
			status, _, _ := tmdbImageTestGet(t, url, ts.token)
			expected := http.StatusBadGateway
			if tt.status == http.StatusNotFound {
				expected = http.StatusNotFound
			}
			if status != expected {
				t.Fatalf("código = %d, se esperaba %d", status, expected)
			}
			if _, err := os.Stat(filepath.Join(cacheDir, "w342", "error.jpg")); !os.IsNotExist(err) {
				t.Fatalf("no debía guardarse la imagen: %v", err)
			}
		})
	}
}

func TestTMDBImageEndpointLimitaTamano(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(make([]byte, maxTMDBImageBytes+1))
	}))
	defer fake.Close()

	ts := newLibraryHTTPTestServer(t)
	cacheDir := filepath.Join(t.TempDir(), "images")
	ts.api.WithTMDBImages(cacheDir).WithTMDBImageBaseURL(fake.URL)
	url := ts.URL + "/api/v1/images/tmdb/w342/grande.jpg"
	status, _, _ := tmdbImageTestGet(t, url, ts.token)
	if status != http.StatusBadGateway {
		t.Fatalf("código = %d, se esperaba 502", status)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "w342", "grande.jpg")); !os.IsNotExist(err) {
		t.Fatalf("no debía guardarse una imagen demasiado grande: %v", err)
	}
}

func TestTMDBImageEndpointExigeSesion(t *testing.T) {
	var calls atomic.Int64
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("imagen"))
	}))
	defer fake.Close()

	ts := newLibraryHTTPTestServer(t)
	ts.api.WithTMDBImages(filepath.Join(t.TempDir(), "images")).WithTMDBImageBaseURL(fake.URL)
	status, _, _ := tmdbImageTestGet(t, ts.URL+"/api/v1/images/tmdb/w342/abc.jpg", "")
	if status != http.StatusUnauthorized {
		t.Fatalf("código = %d, se esperaba 401", status)
	}
	if calls.Load() != 0 {
		t.Fatalf("TMDB no debía recibir peticiones: %d", calls.Load())
	}
}

func TestTMDBImageEndpointAceptaLaCookieDeLaWeb(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("imagen"))
	}))
	defer fake.Close()

	ts := newLibraryHTTPTestServer(t)
	ts.api.WithTMDBImages(filepath.Join(t.TempDir(), "images")).WithTMDBImageBaseURL(fake.URL)
	url := ts.URL + "/api/v1/images/tmdb/w342/abc.jpg"

	for _, tc := range []struct {
		value string
		want  int
	}{{ts.token, http.StatusOK}, {"no-valido", http.StatusUnauthorized}} {
		request, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Fatalf("crear petición: %v", err)
		}
		request.AddCookie(&http.Cookie{Name: "cinecat_session", Value: tc.value})
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatalf("GET %s: %v", url, err)
		}
		response.Body.Close()
		if response.StatusCode != tc.want {
			t.Fatalf("cookie %q: código = %d, se esperaba %d", tc.value, response.StatusCode, tc.want)
		}
	}
}

func TestTMDBImageEndpointDescargaUnaVezEnConcurrente(t *testing.T) {
	image := []byte("concurrente")
	var calls atomic.Int64
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(image)
	}))
	defer fake.Close()

	ts := newLibraryHTTPTestServer(t)
	ts.api.WithTMDBImages(filepath.Join(t.TempDir(), "images")).WithTMDBImageBaseURL(fake.URL)
	url := ts.URL + "/api/v1/images/tmdb/w342/concurrente.jpg"

	const requests = 8
	statuses := make(chan int, requests)
	var wait sync.WaitGroup
	wait.Add(requests)
	for i := 0; i < requests; i++ {
		go func() {
			defer wait.Done()
			request, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				statuses <- 0
				return
			}
			request.Header.Set("Authorization", "Bearer "+ts.token)
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				statuses <- 0
				return
			}
			_, _ = io.Copy(io.Discard, response.Body)
			response.Body.Close()
			statuses <- response.StatusCode
		}()
	}

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("TMDB no recibió ninguna petición")
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wait.Wait()
	close(statuses)
	for status := range statuses {
		if status != http.StatusOK {
			t.Fatalf("código concurrente = %d, se esperaba 200", status)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("TMDB recibió %d peticiones, se esperaba una", calls.Load())
	}
}

func TestWriteJSONReescribeImagenesTMDB(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeJSON(recorder, http.StatusOK, map[string]string{
		"poster": "https://image.tmdb.org/t/p/w342/abc.jpg",
		"otro":   "https://images.example/fondo.jpg",
	})
	body := recorder.Body.String()
	if !strings.Contains(body, `"poster":"/api/v1/images/tmdb/w342/abc.jpg"`) {
		t.Fatalf("no se reescribió la URL de TMDB: %s", body)
	}
	if strings.Contains(body, "https://image.tmdb.org/t/p/") {
		t.Fatalf("queda la URL original de TMDB: %s", body)
	}
	if !strings.Contains(body, `"otro":"https://images.example/fondo.jpg"`) {
		t.Fatalf("se tocó una URL de otro origen: %s", body)
	}
}

func TestRespuestaDeBibliotecaReescribeImagenesTMDB(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	id := insertItemFixture(t, ts.db, itemFixture{
		libraryID:   libraryID,
		path:        filepath.Join(t.TempDir(), "Pelicula.mkv"),
		title:       "Pelicula",
		posterURL:   "https://image.tmdb.org/t/p/w342/poster.jpg",
		backdropURL: "https://images.example/fondo.jpg",
	})

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(id, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("código = %d, cuerpo = %s", status, body)
	}
	text := string(body)
	if !strings.Contains(text, `"poster_url":"/api/v1/images/tmdb/w342/poster.jpg"`) {
		t.Fatalf("poster_url no reescrito: %s", text)
	}
	if strings.Contains(text, "https://image.tmdb.org/t/p/") {
		t.Fatalf("queda una URL original de TMDB: %s", text)
	}
	if !strings.Contains(text, `"backdrop_url":"https://images.example/fondo.jpg"`) {
		t.Fatalf("se tocó el fondo de otro origen: %s", text)
	}
}
