package playback

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func playlistSegmentDurations(t *testing.T, playlistPath string) []float64 {
	t.Helper()
	file, err := os.Open(playlistPath)
	if err != nil {
		t.Fatalf("abrir playlist: %v", err)
	}
	defer file.Close()

	var durations []float64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "#EXTINF:") {
			continue
		}
		value := strings.TrimSuffix(strings.TrimPrefix(line, "#EXTINF:"), ",")
		duration, err := strconv.ParseFloat(value, 64)
		if err != nil {
			t.Fatalf("EXTINF no válido %q: %v", line, err)
		}
		durations = append(durations, duration)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("leer playlist: %v", err)
	}
	return durations
}

func TestHLSArgsCopiaVsTranscodificacion(t *testing.T) {
	copyArgs := strings.Join(hlsArgs("/tmp/peli.mkv"), "\n")
	if !strings.Contains(copyArgs, "-c:v\ncopy") || !strings.Contains(copyArgs, "-c:a\ncopy") {
		t.Fatalf("el remux normal debe copiar vídeo y audio: %v", copyArgs)
	}
	if strings.Contains(copyArgs, "libx264") {
		t.Fatalf("el remux normal no debe recodificar: %v", copyArgs)
	}

	transcodeArgs := strings.Join(hlsTranscodeArgs("/tmp/peli.mkv"), "\n")
	for _, expected := range []string{
		"-c:v\nlibx264",
		"-preset\nveryfast",
		"-crf\n20",
		"-force_key_frames\nexpr:gte(t,n_forced*6)",
		"-c:a\naac",
	} {
		if !strings.Contains(transcodeArgs, expected) {
			t.Fatalf("faltan argumentos de transcodificación %q: %v", expected, transcodeArgs)
		}
	}
}

func TestPrepareTranscodificaKeyframesDispersos(t *testing.T) {
	requireVideoTools(t)
	sparse := generateKeyframeVideo(t, filepath.Join(t.TempDir(), "disperso.mkv"), "40",
		"-x264-params", "keyint=9999:min-keyint=9999:scenecut=0",
		"-force_key_frames", "expr:gte(t,20)")

	svc, err := New(Options{WorkDir: filepath.Join(t.TempDir(), "hls")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	entry, err := svc.prepare(context.Background(), 1, sparse)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if entry.mode != ModeTranscode {
		t.Fatalf("modo = %q, se esperaba %q", entry.mode, ModeTranscode)
	}
	if _, err := os.Stat(filepath.Join(entry.dir, "playlist.m3u8")); err != nil {
		t.Fatalf("prepare debe devolver la playlist inicial: %v", err)
	}
	select {
	case <-entry.done:
	case <-time.After(60 * time.Second):
		t.Fatal("FFmpeg no terminó en 60 s")
	}
	if entry.err != nil {
		t.Fatalf("FFmpeg terminó con error: %v", entry.err)
	}

	durations := playlistSegmentDurations(t, filepath.Join(entry.dir, "playlist.m3u8"))
	t.Logf("segmentos finales: %v", durations)
	if len(durations) < 2 {
		t.Fatalf("se esperaban varios segmentos, hay %d: %v", len(durations), durations)
	}
	for _, duration := range durations {
		if duration > 8 {
			t.Fatalf("segmento demasiado largo para hls_time 6: %.2f s", duration)
		}
	}
}

func TestPrepareMantieneRemuxConKeyframesNormales(t *testing.T) {
	requireVideoTools(t)
	normal := generateKeyframeVideo(t, filepath.Join(t.TempDir(), "normal.mkv"), "2",
		"-g", "5", "-keyint_min", "5", "-sc_threshold", "0")

	svc, err := New(Options{WorkDir: filepath.Join(t.TempDir(), "hls")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	entry, err := svc.prepare(context.Background(), 2, normal)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if entry.mode != ModeRemux {
		t.Fatalf("modo = %q, se esperaba %q", entry.mode, ModeRemux)
	}
	if _, err := os.Stat(filepath.Join(entry.dir, "playlist.m3u8")); err != nil {
		t.Fatalf("no se generó la playlist: %v", err)
	}
}

func TestNeedsTranscodeFuerzaAVISiempre(t *testing.T) {
	svc, err := New(Options{WorkDir: filepath.Join(t.TempDir(), "hls")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	// Caso real de producción: un .avi puede tener keyframes bien repartidos
	// (el prober lo consideraría viable) y aun así no segmentar bien con
	// -c:v copy porque los timestamps de paquete no son fiables en ese
	// contenedor. Por eso .avi siempre transcodifica, sin consultar al prober.
	if !svc.needsTranscode(context.Background(), "inexistente.avi") {
		t.Fatal(".avi debe forzar transcodificación siempre, sin depender del prober")
	}
	if !svc.needsTranscode(context.Background(), "/ruta/con/mayusculas/Pelicula.AVI") {
		t.Fatal("la detección de .avi debe ser insensible a mayúsculas")
	}
}

func TestNeedsTranscodeSiElProberFalla(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	svc, err := New(Options{WorkDir: filepath.Join(t.TempDir(), "hls")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	if !svc.needsTranscode(context.Background(), "inexistente.mkv") {
		t.Fatal("si no se puede comprobar la viabilidad, debe usarse la transcodificación como fallback")
	}
}

func TestPrepareDevuelveAntesDeTerminarYEsperaSegmentos(t *testing.T) {
	requireVideoTools(t)
	source := generateKeyframeVideo(t, filepath.Join(t.TempDir(), "larga.mkv"), "300",
		"-x264-params", "keyint=9999:min-keyint=9999:scenecut=0",
		"-force_key_frames", "expr:gte(t,120)")

	svc, err := New(Options{WorkDir: filepath.Join(t.TempDir(), "hls")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	start := time.Now()
	entry, err := svc.prepare(context.Background(), 1, source)
	prepareElapsed := time.Since(start)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	t.Logf("prepare() devolvió en %s y FFmpeg sigue en segundo plano", prepareElapsed)
	if prepareElapsed > 15*time.Second {
		t.Fatalf("prepare tardó %s; debería devolver tras la playlist inicial", prepareElapsed)
	}
	playlistPath := filepath.Join(entry.dir, "playlist.m3u8")
	playlist, err := os.ReadFile(playlistPath)
	if err != nil {
		t.Fatalf("no hay playlist inicial: %v", err)
	}
	text := string(playlist)
	if !strings.Contains(text, "#EXT-X-PLAYLIST-TYPE:VOD") || !strings.Contains(text, "#EXT-X-ENDLIST") {
		t.Fatalf("la playlist inicial no es VOD completa: %s", text)
	}
	if !strings.Contains(text, "segment_00049.m4s") {
		t.Fatalf("la playlist no lista el último segmento esperado de 300 s: %s", text)
	}
	if durations := playlistSegmentDurations(t, playlistPath); len(durations) != 50 {
		t.Fatalf("segmentos en la playlist = %d, se esperaban 50", len(durations))
	}
	if _, err := os.Stat(filepath.Join(entry.dir, "segment_00049.m4s")); err == nil {
		t.Fatal("el último segmento no debería existir aún al volver prepare()")
	}

	select {
	case <-entry.done:
		t.Fatal("FFmpeg terminó antes de poder comprobar que prepare() devuelve en segundo plano")
	default:
	}

	start = time.Now()
	if err := svc.waitForHLSFile(context.Background(), entry, "segment_00000.m4s"); err != nil {
		t.Fatalf("esperar primer segmento: %v", err)
	}
	firstElapsed := time.Since(start)
	t.Logf("primer segmento servible en %s", firstElapsed)
	if firstElapsed > 15*time.Second {
		t.Fatalf("el primer segmento tardó %s", firstElapsed)
	}

	// Un segmento avanzado que FFmpeg aún no ha generado también debe esperarse.
	if _, err := os.Stat(filepath.Join(entry.dir, "segment_00040.m4s")); err == nil {
		t.Fatal("el segmento avanzado no debería existir aún al volver prepare()")
	}
	if err := svc.waitForHLSFile(context.Background(), entry, "segment_00040.m4s"); err != nil {
		t.Fatalf("esperar segmento avanzado: %v", err)
	}
	if _, err := os.Stat(filepath.Join(entry.dir, "segment_00040.m4s")); err != nil {
		t.Fatalf("el segmento avanzado no apareció: %v", err)
	}
}

func TestPlaylistVODRemuxCoincideConKeyframes(t *testing.T) {
	requireVideoTools(t)
	source := generateKeyframeVideo(t, filepath.Join(t.TempDir(), "keyframes8.mkv"), "40",
		"-g", "80", "-keyint_min", "80", "-sc_threshold", "0")

	svc, err := New(Options{WorkDir: filepath.Join(t.TempDir(), "hls")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	entry, err := svc.prepare(context.Background(), 4, source)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if entry.mode != ModeRemux {
		t.Fatalf("modo = %q, se esperaba remux", entry.mode)
	}
	durations := playlistSegmentDurations(t, filepath.Join(entry.dir, "playlist.m3u8"))
	if len(durations) != 5 {
		t.Fatalf("segmentos VOD = %v, se esperaban 5 según los keyframes cada 8 s", durations)
	}
	select {
	case <-entry.done:
	case <-time.After(30 * time.Second):
		t.Fatal("FFmpeg no terminó en 30 s")
	}
	if entry.err != nil {
		t.Fatalf("FFmpeg terminó con error: %v", entry.err)
	}
	for i := 0; i < 5; i++ {
		name := filepath.Join(entry.dir, fmt.Sprintf("segment_%05d.m4s", i))
		if _, err := os.Stat(name); err != nil {
			t.Fatalf("falta el segmento %d: %v", i, err)
		}
	}
}

func TestPrepareConcurrenteNoDuplicaProceso(t *testing.T) {
	requireVideoTools(t)
	source := generateKeyframeVideo(t, filepath.Join(t.TempDir(), "concurrente.mkv"), "60",
		"-x264-params", "keyint=9999:min-keyint=9999:scenecut=0",
		"-force_key_frames", "expr:gte(t,30)")

	svc, err := New(Options{WorkDir: filepath.Join(t.TempDir(), "hls")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

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

	svc, err := New(Options{WorkDir: filepath.Join(t.TempDir(), "hls")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	if _, err := svc.prepare(context.Background(), 3, bad); !errors.Is(err, ErrFFmpegFailed) {
		t.Fatalf("error = %v, se esperaba ErrFFmpegFailed", err)
	}
}

func TestServeHLSSimultaneoReutilizaProceso(t *testing.T) {
	requireVideoTools(t)
	source := generateKeyframeVideo(t, filepath.Join(t.TempDir(), "http.mkv"), "60",
		"-x264-params", "keyint=9999:min-keyint=9999:scenecut=0",
		"-force_key_frames", "expr:gte(t,30)")

	svc, err := New(Options{WorkDir: filepath.Join(t.TempDir(), "hls")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if err := svc.ServeHLSFile(w, r, 55, source, name, "123", "firma"); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	t.Cleanup(server.Close)

	paths := []string{"/playlist.m3u8", "/segment_00000.m4s", "/segment_00008.m4s"}
	statuses := make([]int, len(paths))
	bodies := make([]string, len(paths))
	var wg sync.WaitGroup
	for i, path := range paths {
		wg.Add(1)
		go func(index int, path string) {
			defer wg.Done()
			resp, err := http.Get(server.URL + path)
			if err != nil {
				statuses[index] = 0
				return
			}
			defer resp.Body.Close()
			statuses[index] = resp.StatusCode
			data, _ := io.ReadAll(resp.Body)
			bodies[index] = string(data)
		}(i, path)
	}
	wg.Wait()

	for i, status := range statuses {
		if status != http.StatusOK {
			t.Fatalf("%s devolvió %d", paths[i], status)
		}
	}
	if !strings.Contains(bodies[0], "#EXT-X-PLAYLIST-TYPE:VOD") ||
		!strings.Contains(bodies[0], "#EXT-X-ENDLIST") ||
		!strings.Contains(bodies[0], "segment_00009.m4s") {
		t.Fatalf("la playlist HTTP no es un VOD completo: %s", bodies[0])
	}
	if strings.Count(bodies[0], "#EXTINF:") != 10 {
		t.Fatalf("la playlist HTTP no lista los 10 segmentos de 60 s: %s", bodies[0])
	}
	if len(bodies[1]) == 0 || len(bodies[2]) == 0 {
		t.Fatal("los segmentos servidos por HTTP llegaron vacíos")
	}

	svc.mu.Lock()
	entry := svc.entries[55]
	svc.mu.Unlock()
	if entry == nil {
		t.Fatal("no se creó la entrada del ítem")
	}
	// Solo debe existir un directorio de trabajo para el ítem: no se lanzó
	// un segundo FFmpeg para la petición simultánea.
	matches, err := filepath.Glob(filepath.Join(svc.workDir, "item-55"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("directorios del ítem: %v, error %v", matches, err)
	}
}

func TestPrepareCaeAPlaylistEventSiFaltaDuracion(t *testing.T) {
	requireVideoTools(t)
	raw := generateRawH264(t, filepath.Join(t.TempDir(), "raw.h264"))
	data, err := os.ReadFile(raw)
	if err != nil {
		t.Fatalf("leer h264: %v", err)
	}
	// Un flujo H.264 crudo con extensión .avi: fuerza transcodificación y no
	// tiene duración de contenedor, así que debe usarse la playlist event de
	// FFmpeg como fallback.
	avi := filepath.Join(t.TempDir(), "sin-duracion.avi")
	if err := os.WriteFile(avi, data, 0o644); err != nil {
		t.Fatalf("crear avi: %v", err)
	}

	svc, err := New(Options{WorkDir: filepath.Join(t.TempDir(), "hls")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	entry, err := svc.prepare(context.Background(), 6, avi)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if entry.playlistFile != ffmpegPlaylistFile {
		t.Fatalf("playlist = %q, se esperaba el fallback %q", entry.playlistFile, ffmpegPlaylistFile)
	}
	select {
	case <-entry.done:
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
