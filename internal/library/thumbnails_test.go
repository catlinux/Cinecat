package library

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/catlinux/cinecat/internal/playback"
)

type thumbnailRunnerRecord struct {
	mu        sync.Mutex
	calls     int
	times     []float64
	sources   []string
	dest      []string
	active    int
	maxActive int
	release   chan struct{}
	started   chan struct{}
	runErr    error
	darkBelow float64
}

func (r *thumbnailRunnerRecord) run(ctx context.Context, args []string) error {
	r.mu.Lock()
	r.calls++
	r.active++
	if r.active > r.maxActive {
		r.maxActive = r.active
	}
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.active--
		r.mu.Unlock()
	}()

	ss := 0.0
	source := ""
	for index, arg := range args {
		if arg == "-ss" && index+1 < len(args) {
			ss, _ = strconv.ParseFloat(args[index+1], 64)
		}
		if arg == "-i" && index+1 < len(args) {
			source = args[index+1]
		}
	}
	r.mu.Lock()
	r.times = append(r.times, ss)
	r.sources = append(r.sources, source)
	r.dest = append(r.dest, args[len(args)-1])
	err := r.runErr
	release := r.release
	started := r.started
	darkBelow := r.darkBelow
	r.mu.Unlock()
	if started != nil {
		select {
		case started <- struct{}{}:
		default:
		}
	}
	if release != nil {
		select {
		case <-release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err != nil {
		return err
	}
	luma := uint8(220)
	if darkBelow > 0 && ss < darkBelow {
		luma = 0
	}
	return writeTestJPEG(args[len(args)-1], luma)
}

func writeTestJPEG(path string, value uint8) error {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.SetRGBA(x, y, color.RGBA{R: value, G: value, B: value, A: 255})
		}
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return jpeg.Encode(file, img, &jpeg.Options{Quality: 80})
}

func thumbnailTestSetup(t *testing.T) (*libraryTestServer, *fakeFileInspector, *thumbnailRunnerRecord, int64, string, string) {
	t.Helper()
	ts := newLibraryHTTPTestServer(t)
	root := t.TempDir()
	path := filepath.Join(root, "Concierto Ñandú 2020.mp4")
	writeFile(t, path)
	libraryID := createClipLibrary(t, ts.db, root)
	addScannerLibraryPath(t, ts.db, libraryID, root)
	itemID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: path, title: "Concierto", kind: KindMovie,
		matchStatus: "none",
	})
	cacheDir := filepath.Join(t.TempDir(), "cache", "thumbs")
	inspector := &fakeFileInspector{info: &playback.FileInfo{DurationSeconds: 100}}
	runner := &thumbnailRunnerRecord{}
	ts.api.WithFileInspector(inspector)
	ts.api.WithThumbnails(cacheDir)
	ts.api.WithThumbnailRunner(runner.run)
	return ts, inspector, runner, itemID, path, cacheDir
}

func thumbnailHTTPGet(t *testing.T, url, token string, headers map[string]string) (int, http.Header, []byte) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("crear petición: %v", err)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("leer miniatura: %v", err)
	}
	return response.StatusCode, response.Header, body
}

func TestThumbnailSeek(t *testing.T) {
	if value := thumbnailSeek(100, 0.10); value != 10 {
		t.Fatalf("10%% de 100 = %v", value)
	}
	if value := thumbnailSeek(100, 0.30); value != 30 {
		t.Fatalf("30%% de 100 = %v", value)
	}
	if value := thumbnailSeek(10000, 0.10); value != 600 {
		t.Fatalf("recorte superior = %v", value)
	}
	if value := thumbnailSeek(3, 0.10); value != 1 {
		t.Fatalf("recorte inferior = %v", value)
	}
	if value := thumbnailSeek(0, 0.10); value != 30 {
		t.Fatalf("duración desconocida = %v", value)
	}
}

func TestThumbnailEndpointCacheEtagsYArtificios(t *testing.T) {
	ts, _, runner, itemID, source, cacheDir := thumbnailTestSetup(t)
	url := ts.URL + "/api/v1/items/" + strconv.FormatInt(itemID, 10) + "/thumbnail"

	status, headers, body := thumbnailHTTPGet(t, url, ts.token, nil)
	if status != http.StatusOK {
		t.Fatalf("miniatura: código = %d, cuerpo = %s", status, body)
	}
	if headers.Get("Content-Type") != "image/jpeg" ||
		headers.Get("Cache-Control") != "private, max-age=86400" ||
		headers.Get("ETag") == "" {
		t.Fatalf("cabeceras inesperadas: %+v", headers)
	}
	if len(body) == 0 {
		t.Fatal("miniatura vacía")
	}
	stat, err := os.Stat(source)
	if err != nil {
		t.Fatalf("stat fuente: %v", err)
	}
	expected := filepath.Join(cacheDir, itemIDString(itemID)+"-"+
		strconv.FormatInt(stat.ModTime().UnixNano(), 10)+"-"+strconv.FormatInt(stat.Size(), 10)+".jpg")
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("no existe la caché esperada %s: %v", expected, err)
	}
	if runner.calls != 1 || len(runner.times) != 1 || runner.times[0] != 10 {
		t.Fatalf("generación inesperada: %+v", runner)
	}
	if runner.sources[0] != source {
		t.Fatalf("ruta pasada a FFmpeg = %q", runner.sources[0])
	}

	status, _, _ = thumbnailHTTPGet(t, url, ts.token, nil)
	if status != http.StatusOK || runner.calls != 1 {
		t.Fatalf("la segunda petición debe usar caché: status=%d calls=%d", status, runner.calls)
	}

	etag := headers.Get("ETag")
	status, _, _ = thumbnailHTTPGet(t, url, ts.token, map[string]string{"If-None-Match": etag})
	if status != http.StatusNotModified || runner.calls != 1 {
		t.Fatalf("ETag: status=%d calls=%d", status, runner.calls)
	}

	if err := os.WriteFile(source, []byte("contenido cambiado"), 0o644); err != nil {
		t.Fatalf("cambiar archivo: %v", err)
	}
	status, _, _ = thumbnailHTTPGet(t, url, ts.token, nil)
	if status != http.StatusOK || runner.calls != 2 {
		t.Fatalf("cambio de archivo: status=%d calls=%d", status, runner.calls)
	}
}

func itemIDString(id int64) string {
	return strconv.FormatInt(id, 10)
}

func TestThumbnailRetryDarkFrame(t *testing.T) {
	ts, _, runner, itemID, _, _ := thumbnailTestSetup(t)
	runner.darkBelow = 20
	url := ts.URL + "/api/v1/items/" + strconv.FormatInt(itemID, 10) + "/thumbnail"

	status, _, body := thumbnailHTTPGet(t, url, ts.token, nil)
	if status != http.StatusOK {
		t.Fatalf("miniatura oscura: código = %d, cuerpo = %s", status, body)
	}
	if runner.calls != 2 || len(runner.times) != 2 || runner.times[0] != 10 || runner.times[1] != 30 {
		t.Fatalf("no se reintentó al 30%%: %+v", runner)
	}
	luma, err := meanJPEGLuma(body)
	if err != nil || luma < thumbnailDarkLuma {
		t.Fatalf("debería conservarse el fotograma claro: luma=%v err=%v", luma, err)
	}
}

func TestThumbnailSoloClipsYErrores(t *testing.T) {
	ts, _, runner, clipID, source, _ := thumbnailTestSetup(t)
	clipURL := ts.URL + "/api/v1/items/" + strconv.FormatInt(clipID, 10) + "/thumbnail"

	if status, _, _ := thumbnailHTTPGet(t, clipURL, "", nil); status != http.StatusUnauthorized {
		t.Fatalf("sin sesión: código = %d", status)
	}

	videoLibrary := createLibrary(t, ts.db, t.TempDir())
	videoID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: videoLibrary, path: "/media/pelicula.mkv", title: "Pelicula",
	})
	if status, _, _ := thumbnailHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(videoID, 10)+"/thumbnail", ts.token, nil); status != http.StatusNotFound {
		t.Fatalf("película: código = %d", status)
	}
	if runner.calls != 0 {
		t.Fatal("no se debe generar la miniatura de una película")
	}

	if err := os.Remove(source); err != nil {
		t.Fatalf("retirar archivo: %v", err)
	}
	if status, _, _ := thumbnailHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(clipID, 10)+"/thumbnail", ts.token, nil); status != http.StatusNotFound {
		t.Fatalf("archivo ausente: código = %d", status)
	}
}

func TestThumbnailFalloDeGeneracionYFFprobe(t *testing.T) {
	ts, inspector, runner, itemID, _, _ := thumbnailTestSetup(t)
	runner.runErr = errors.New("ffmpeg de prueba")
	url := ts.URL + "/api/v1/items/" + strconv.FormatInt(itemID, 10) + "/thumbnail"
	if status, _, body := thumbnailHTTPGet(t, url, ts.token, nil); status != http.StatusNotFound || len(body) == 0 {
		t.Fatalf("error de FFmpeg: código = %d, cuerpo = %s", status, body)
	}

	runner.runErr = nil
	inspector.err = playback.ErrFFprobeNotFound
	if status, _, _ := thumbnailHTTPGet(t, url, ts.token, nil); status != http.StatusNotFound {
		t.Fatalf("error de ffprobe: código = %d", status)
	}
}

func TestThumbnailConcurrenciaYLimite(t *testing.T) {
	ts, _, runner, itemID, source, _ := thumbnailTestSetup(t)
	var libraryID int64
	if err := ts.db.QueryRow(`SELECT library_id FROM media_items WHERE id = ?`, itemID).Scan(&libraryID); err != nil {
		t.Fatalf("biblioteca del ítem: %v", err)
	}
	secondSource := source + ".segundo.mp4"
	writeFile(t, secondSource)
	secondID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: secondSource, title: "Segundo", kind: KindMovie,
		matchStatus: "none",
	})
	ts.api.WithThumbnailLimits(40*time.Millisecond, time.Second)
	runner.started = make(chan struct{}, 3)
	runner.release = make(chan struct{})
	urls := []string{
		ts.URL + "/api/v1/items/" + strconv.FormatInt(itemID, 10) + "/thumbnail",
		ts.URL + "/api/v1/items/" + strconv.FormatInt(secondID, 10) + "/thumbnail",
	}

	var wg sync.WaitGroup
	statuses := make(chan int, 2)
	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			status, _, _ := thumbnailHTTPGet(t, url, ts.token, nil)
			statuses <- status
		}(url)
	}
	for i := 0; i < 2; i++ {
		select {
		case <-runner.started:
		case <-time.After(2 * time.Second):
			t.Fatal("no arrancaron las dos generaciones")
		}
	}

	start := time.Now()
	status, headers, _ := thumbnailHTTPGet(t, urls[0], ts.token, nil)
	if status != http.StatusServiceUnavailable {
		t.Fatalf("saturación: código = %d", status)
	}
	if headers.Get("Retry-After") != "5" {
		t.Fatalf("Retry-After = %q", headers.Get("Retry-After"))
	}
	if time.Since(start) < 30*time.Millisecond {
		t.Fatalf("la espera fue demasiado corta")
	}

	close(runner.release)
	wg.Wait()
	close(statuses)
	for status := range statuses {
		if status != http.StatusOK {
			t.Fatalf("generación concurrente: código = %d", status)
		}
	}
	if runner.maxActive > 2 {
		t.Fatalf("demasiadas generaciones simultáneas: %d", runner.maxActive)
	}
}

func TestThumbnailTimeoutGeneracion(t *testing.T) {
	ts, _, runner, itemID, _, _ := thumbnailTestSetup(t)
	ts.api.WithThumbnailLimits(time.Second, 30*time.Millisecond)
	runner.release = make(chan struct{})
	url := ts.URL + "/api/v1/items/" + strconv.FormatInt(itemID, 10) + "/thumbnail"

	status, _, _ := thumbnailHTTPGet(t, url, ts.token, nil)
	if status != http.StatusNotFound {
		t.Fatalf("timeout de generación: código = %d", status)
	}
}
