package playback

import (
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// generateDivXVideo genera un AVI como los de la biblioteca real: MPEG-4 Part 2
// con dimensiones impares y audio AC-3 5.1.
func generateDivXVideo(t *testing.T, target string, seconds int) string {
	t.Helper()
	duration := strconv.Itoa(seconds)
	cmd := exec.Command("ffmpeg",
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=duration="+duration+":size=161x121:rate=25",
		"-f", "lavfi", "-i", "sine=frequency=440:duration="+duration,
		"-c:v", "mpeg4", "-c:a", "ac3", "-ac", "6",
		target,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generar vídeo: %v: %s", err, out)
	}
	return target
}

func newHLSTestService(t *testing.T) *Service {
	t.Helper()
	svc, err := New(Options{WorkDir: filepath.Join(t.TempDir(), "hls")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return svc
}

// probeStreams devuelve las líneas "tipo,códec,canales,inicio" de un segmento.
func probeStreams(t *testing.T, path string) []string {
	t.Helper()
	out, err := exec.Command("ffprobe", "-v", "error",
		"-show_entries", "stream=codec_type,codec_name,channels,start_time",
		"-of", "csv=p=0", path).Output()
	if err != nil {
		t.Fatalf("ffprobe %s: %v", path, err)
	}
	return strings.Fields(strings.TrimSpace(string(out)))
}

func videoStart(t *testing.T, path string) float64 {
	t.Helper()
	for _, line := range probeStreams(t, path) {
		fields := strings.Split(line, ",")
		if len(fields) >= 3 && fields[1] == "video" {
			value, err := strconv.ParseFloat(fields[len(fields)-1], 64)
			if err != nil {
				t.Fatalf("start_time no válido en %q", line)
			}
			return value
		}
	}
	t.Fatalf("%s no tiene vídeo", path)
	return 0
}

func currentRunStart(svc *Service, entry *hlsEntry) int {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if entry.run == nil {
		return -1
	}
	return entry.run.start
}

func TestHLSArgsSaltoYAudioEstereo(t *testing.T) {
	first := strings.Join(hlsTranscodeArgs("/tmp/peli.avi", 0, defaultTranscodeSettings()), "\n")
	if strings.Contains(first, "-ss\n") {
		t.Fatalf("el proceso inicial no debe buscar: %v", first)
	}
	for _, expected := range []string{
		"-c:v\nlibx264",
		"-c:a\naac",
		"-ac\n2",
		"-start_number\n0",
		"-output_ts_offset\n10",
		"-hls_segment_filename\nsegment_%05d.ts",
		"-threads\n" + strconv.Itoa(defaultTranscodeThreads()),
	} {
		if !strings.Contains(first, expected) {
			t.Fatalf("falta %q: %v", expected, first)
		}
	}

	seek := strings.Join(hlsTranscodeArgs("/tmp/peli.avi", 500, defaultTranscodeSettings()), "\n")
	for _, expected := range []string{"-ss\n3000\n-i", "-start_number\n500", "-output_ts_offset\n3010"} {
		if !strings.Contains(seek, expected) {
			t.Fatalf("falta %q en el relanzado: %v", expected, seek)
		}
	}
}

func TestUniformSegmentDurations(t *testing.T) {
	for _, tc := range []struct {
		duration time.Duration
		count    int
		last     time.Duration
	}{
		{300 * time.Second, 50, 6 * time.Second},
		{300*time.Second + 30*time.Millisecond, 50, 6*time.Second + 30*time.Millisecond},
		{9082480 * time.Millisecond, 1514, 4480 * time.Millisecond},
		{500 * time.Millisecond, 1, 500 * time.Millisecond},
	} {
		got := uniformSegmentDurations(tc.duration)
		if len(got) != tc.count || got[len(got)-1] != tc.last {
			t.Errorf("%s: %d segmentos, último %s; se esperaban %d y %s",
				tc.duration, len(got), got[len(got)-1], tc.count, tc.last)
		}
	}
}

func TestPrepareEscribeVODSinArrancarFFmpeg(t *testing.T) {
	requireVideoTools(t)
	source := generateDivXVideo(t, filepath.Join(t.TempDir(), "larga.avi"), 300)
	svc := newHLSTestService(t)

	start := time.Now()
	entry, err := svc.prepare(context.Background(), 1, source)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("prepare tardó %s", elapsed)
	}
	if entry.run != nil {
		t.Fatal("prepare no debe arrancar FFmpeg hasta que se pida un segmento")
	}
	// El AVI dura unos milisegundos más de 300 s por el audio; ese resto va
	// al último segmento.
	if entry.segments != 50 {
		t.Fatalf("segmentos = %d, se esperaban 50 para 300 s", entry.segments)
	}
	data, err := os.ReadFile(filepath.Join(entry.dir, "playlist.m3u8"))
	if err != nil {
		t.Fatalf("leer playlist: %v", err)
	}
	text := string(data)
	for _, expected := range []string{"#EXT-X-PLAYLIST-TYPE:VOD", "#EXT-X-ENDLIST", "segment_00049.ts"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("la playlist no contiene %q: %s", expected, text)
		}
	}
	if strings.Contains(text, "EXT-X-MAP") || strings.Contains(text, ".m4s") {
		t.Fatalf("la playlist no debe usar fMP4: %s", text)
	}
}

// TestSaltosRelanzanFFmpeg reproduce el fallo real: tras saltar a un punto
// lejano, el segmento debe llegar en segundos (no esperar a que FFmpeg
// recorra todo el vídeo), y los segmentos de procesos distintos deben encajar
// en la misma línea de tiempo.
func TestSaltosRelanzanFFmpeg(t *testing.T) {
	requireVideoTools(t)
	source := generateDivXVideo(t, filepath.Join(t.TempDir(), "saltos.avi"), 300)
	svc := newHLSTestService(t)

	entry, err := svc.prepare(context.Background(), 2, source)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	// Salto directo a mitad de vídeo: el primer proceso empieza ahí.
	start := time.Now()
	if err := svc.waitForSegment(context.Background(), entry, 40); err != nil {
		t.Fatalf("segmento 40: %v", err)
	}
	t.Logf("segmento 40 servible en %s", time.Since(start))
	if got := currentRunStart(svc, entry); got != 40 {
		t.Fatalf("el proceso empieza en %d, se esperaba 40", got)
	}
	offset40 := videoStart(t, filepath.Join(entry.dir, segmentName(40))) - 40*hlsSegmentSeconds

	// Salto atrás a un hueco: no se espera al proceso actual, se relanza
	// (pasado el margen mínimo de vida del proceso).
	time.Sleep(hlsRestartMinAge)
	start = time.Now()
	if err := svc.waitForSegment(context.Background(), entry, 2); err != nil {
		t.Fatalf("segmento 2: %v", err)
	}
	t.Logf("segmento 2 servible en %s", time.Since(start))
	if got := currentRunStart(svc, entry); got != 2 {
		t.Fatalf("el proceso empieza en %d, se esperaba 2 tras saltar atrás", got)
	}
	if err := svc.waitForSegment(context.Background(), entry, 3); err != nil {
		t.Fatalf("segmento 3: %v", err)
	}

	// Misma línea de tiempo: el segmento N empieza en N*6 + desplazamiento,
	// con el mismo desfase venga del proceso que venga.
	offset2 := videoStart(t, filepath.Join(entry.dir, segmentName(2))) - 2*hlsSegmentSeconds
	offset3 := videoStart(t, filepath.Join(entry.dir, segmentName(3))) - 3*hlsSegmentSeconds
	t.Logf("desfases: seg2 %.3f, seg3 %.3f, seg40 %.3f", offset2, offset3, offset40)
	if math.Abs(offset2-offset40) > 0.05 || math.Abs(offset3-offset40) > 0.05 {
		t.Fatalf("los segmentos de procesos distintos no encajan: %.3f, %.3f, %.3f", offset2, offset3, offset40)
	}
	if offset40 < hlsTimestampOffsetSeconds || offset40 > hlsTimestampOffsetSeconds+2 {
		t.Fatalf("desfase inesperado %.3f", offset40)
	}
}

func TestAudioSeBajaAEstereo(t *testing.T) {
	requireVideoTools(t)
	source := generateDivXVideo(t, filepath.Join(t.TempDir(), "surround.avi"), 20)
	svc := newHLSTestService(t)

	entry, err := svc.prepare(context.Background(), 3, source)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := svc.waitForSegment(context.Background(), entry, 0); err != nil {
		t.Fatalf("segmento 0: %v", err)
	}
	streams := strings.Join(probeStreams(t, filepath.Join(entry.dir, segmentName(0))), " ")
	if !strings.Contains(streams, "h264,video") || !strings.Contains(streams, "aac,audio,2") {
		t.Fatalf("se esperaba H.264 y AAC estéreo, hay: %s", streams)
	}
}

func TestSegmentoFueraDelVideo(t *testing.T) {
	requireVideoTools(t)
	source := generateDivXVideo(t, filepath.Join(t.TempDir(), "corta.avi"), 20)
	svc := newHLSTestService(t)

	entry, err := svc.prepare(context.Background(), 4, source)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := svc.waitForSegment(context.Background(), entry, entry.segments); !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("error = %v, se esperaba ErrFileNotFound", err)
	}
	// El último segmento listado sí debe poder servirse.
	if err := svc.waitForSegment(context.Background(), entry, entry.segments-1); err != nil {
		t.Fatalf("último segmento: %v", err)
	}
}

func TestPrepareConcurrenteNoDuplicaEntrada(t *testing.T) {
	requireVideoTools(t)
	source := generateDivXVideo(t, filepath.Join(t.TempDir(), "concurrente.avi"), 30)
	svc := newHLSTestService(t)

	var entries [2]*hlsEntry
	var errs [2]error
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			entries[index], errs[index] = svc.prepare(context.Background(), 9, source)
		}(i)
	}
	wg.Wait()

	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("errores concurrentes: %v, %v", errs[0], errs[1])
	}
	if entries[0] == nil || entries[0] != entries[1] {
		t.Fatalf("las dos peticiones deben reutilizar la misma entrada: %p, %p", entries[0], entries[1])
	}
}

func TestPrepareDevuelveErrorSiFFmpegFalla(t *testing.T) {
	requireVideoTools(t)
	bad := filepath.Join(t.TempDir(), "roto.avi")
	if err := os.WriteFile(bad, []byte("no es un vídeo"), 0o644); err != nil {
		t.Fatalf("crear archivo: %v", err)
	}
	svc := newHLSTestService(t)

	if _, err := svc.prepare(context.Background(), 5, bad); !errors.Is(err, ErrFFmpegFailed) {
		t.Fatalf("error = %v, se esperaba ErrFFmpegFailed", err)
	}
}

func TestServeHLSPorHTTP(t *testing.T) {
	requireVideoTools(t)
	source := generateDivXVideo(t, filepath.Join(t.TempDir(), "http.avi"), 60)
	svc := newHLSTestService(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if err := svc.ServeHLSFile(w, r, 55, source, name, "123", "firma"); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	t.Cleanup(server.Close)

	get := func(path string) (int, string, string) {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, resp.Header.Get("Content-Type"), string(data)
	}

	status, _, playlist := get("/playlist.m3u8")
	if status != http.StatusOK || strings.Count(playlist, "#EXTINF:") != 10 ||
		!strings.Contains(playlist, "segment_00009.ts"+signedQuery("123", "firma", "m", "transcode")) {
		t.Fatalf("playlist HTTP inesperada (%d): %s", status, playlist)
	}

	// Dos peticiones simultáneas a puntos lejanos: ambas deben acabar
	// recibiendo su segmento, sin relanzarse FFmpeg la una a la otra.
	paths := []string{"/segment_00000.ts", "/segment_00008.ts"}
	results := make([]string, len(paths))
	var wg sync.WaitGroup
	for i, path := range paths {
		wg.Add(1)
		go func(index int, path string) {
			defer wg.Done()
			status, contentType, body := get(path)
			if status != http.StatusOK || contentType != "video/mp2t" || len(body) == 0 {
				results[index] = path + ": " + strconv.Itoa(status) + " " + contentType
			}
		}(i, path)
	}
	wg.Wait()
	for _, result := range results {
		if result != "" {
			t.Fatal(result)
		}
	}

	if status, _, _ := get("/init.mp4"); status == http.StatusOK {
		t.Fatal("init.mp4 ya no existe en HLS MPEG-TS")
	}
}

func TestPrepareCaeAPlaylistEventSiFaltaDuracion(t *testing.T) {
	requireVideoTools(t)
	raw := generateRawH264(t, filepath.Join(t.TempDir(), "raw.h264"))
	data, err := os.ReadFile(raw)
	if err != nil {
		t.Fatalf("leer h264: %v", err)
	}
	// Un flujo H.264 crudo con extensión .avi no tiene duración de
	// contenedor: se sirve la playlist event de FFmpeg como fallback.
	avi := filepath.Join(t.TempDir(), "sin-duracion.avi")
	if err := os.WriteFile(avi, data, 0o644); err != nil {
		t.Fatalf("crear avi: %v", err)
	}
	svc := newHLSTestService(t)

	entry, err := svc.prepare(context.Background(), 6, avi)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if entry.playlistFile != ffmpegPlaylistFile || entry.segments != 0 {
		t.Fatalf("playlist = %q (%d segmentos), se esperaba el fallback", entry.playlistFile, entry.segments)
	}
	if err := svc.waitForSegment(context.Background(), entry, 0); err != nil {
		t.Fatalf("segmento 0 del fallback: %v", err)
	}
	select {
	case <-entry.run.done:
	case <-time.After(30 * time.Second):
		t.Fatal("FFmpeg no terminó en 30 s")
	}
	playlist, err := os.ReadFile(filepath.Join(entry.dir, entry.playlistFile))
	if err != nil {
		t.Fatalf("leer playlist fallback: %v", err)
	}
	if !strings.Contains(string(playlist), "#EXT-X-PLAYLIST-TYPE:EVENT") {
		t.Fatalf("la playlist fallback debería ser event: %s", playlist)
	}
}
