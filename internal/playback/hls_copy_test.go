package playback

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestHLSCopyArgsReceta(t *testing.T) {
	bounds := []float64{0, 6.1, 12.2, 18.3}
	settings := TranscodeSettings{Audio: 1}
	input := `C:\media\Pelicula con espacios.mkv`

	startZero := []string{
		"-hide_banner", "-loglevel", "error", "-nostdin", "-y", "-copyts",
		"-i", input,
		"-map", "0:V:0", "-map", "0:a:1?",
		"-c:v", "copy", "-c:a", "aac", "-ac", "2", "-b:a", "160k",
		"-output_ts_offset", "10",
		"-f", "segment", "-segment_format", "mpegts",
		"-segment_times", "16.100000,22.200000,28.300000",
		"-segment_time_delta", "0.05",
		"-segment_start_number", "0",
		"-segment_list", ffmpegPlaylistFile,
		"-segment_list_type", "m3u8",
		"segment_%05d.ts",
	}
	if got := hlsCopyArgs(input, 0, bounds, settings); !reflect.DeepEqual(got, startZero) {
		t.Fatalf("argumentos de copia desde 0:\n%q\nse esperaba:\n%q", got, startZero)
	}

	startTwo := []string{
		"-hide_banner", "-loglevel", "error", "-nostdin", "-y", "-copyts",
		"-ss", "12.400000",
		"-i", input,
		"-map", "0:V:0", "-map", "0:a:1?",
		"-c:v", "copy", "-c:a", "aac", "-ac", "2", "-b:a", "160k",
		"-output_ts_offset", "10",
		"-f", "segment", "-segment_format", "mpegts",
		"-segment_times", "28.300000",
		"-segment_time_delta", "0.05",
		"-segment_start_number", "2",
		"-segment_list", ffmpegPlaylistFile,
		"-segment_list_type", "m3u8",
		"segment_%05d.ts",
	}
	if got := hlsCopyArgs(input, 2, bounds, settings); !reflect.DeepEqual(got, startTwo) {
		t.Fatalf("argumentos de copia desde 2:\n%q\nse esperaba:\n%q", got, startTwo)
	}

	last := hlsCopyArgs(input, 3, bounds, settings)
	for _, arg := range last {
		if arg == "-segment_times" {
			t.Fatalf("el último segmento no debe llevar -segment_times: %q", last)
		}
	}
}

func TestPrepareCopiaConIndiceListo(t *testing.T) {
	source := writeCopyTestFile(t, []byte("archivo de prueba"))
	svc := newCopyTestService(t, fakePlaybackCommand(0))
	injectCopyKeyframes(t, svc, source, []float64{0, 2, 4, 6, 8, 10, 12})

	entry, err := svc.prepare(context.Background(), 31, source,
		StreamSelection{Copy: true, Duration: 12 * time.Second, Quality: "auto"})
	if err != nil {
		t.Fatalf("prepare de copia: %v", err)
	}
	if entry.mode != hlsModeCopy || !strings.Contains(entry.dir, "copy") {
		t.Fatalf("la entrada no es de copia: modo=%q dir=%q", entry.mode, entry.dir)
	}
	if entry.playlistFile != "playlist.m3u8" || entry.segments != 2 {
		t.Fatalf("playlist de copia inesperada: archivo=%q segmentos=%d", entry.playlistFile, entry.segments)
	}
	if len(entry.copyBounds) != 2 || entry.copyBounds[0] != 0 || entry.copyBounds[1] != 6 {
		t.Fatalf("límites de copia = %v", entry.copyBounds)
	}
	data, err := os.ReadFile(filepath.Join(entry.dir, entry.playlistFile))
	if err != nil {
		t.Fatalf("leer playlist: %v", err)
	}
	if strings.Count(string(data), "#EXTINF:6.000000,") != 2 {
		t.Fatalf("duraciones reales inesperadas:\n%s", data)
	}

	again, err := svc.prepare(context.Background(), 31, source,
		StreamSelection{Copy: true, Duration: 12 * time.Second, Quality: "auto"})
	if err != nil {
		t.Fatalf("segundo prepare: %v", err)
	}
	if again != entry {
		t.Fatal("la segunda petición debe reutilizar la entrada de copia")
	}
}

func TestPrepareCopiaPendienteRecodifica(t *testing.T) {
	source := writeCopyTestFile(t, []byte("no es EBML"))
	svc := newCopyTestService(t, fakePlaybackCommand(0))

	entry, err := svc.prepare(context.Background(), 32, source,
		StreamSelection{Copy: true, Duration: 12 * time.Second, Quality: "auto"})
	if err != nil {
		t.Fatalf("prepare con índice pendiente: %v", err)
	}
	if entry.mode == hlsModeCopy || strings.Contains(entry.dir, "-copy") {
		t.Fatalf("un índice pendiente debe recodificar: modo=%q dir=%q", entry.mode, entry.dir)
	}
	if entry.playlistFile != ffmpegPlaylistFile {
		t.Fatalf("playlist = %q, se esperaba la de FFmpeg", entry.playlistFile)
	}
}

func TestServeHLSFileCopiaPoneMEnLosSegmentos(t *testing.T) {
	source := writeCopyTestFile(t, []byte("archivo de prueba"))
	svc := newCopyTestService(t, fakePlaybackCommand(0))
	injectCopyKeyframes(t, svc, source, []float64{0, 2, 4, 6, 8, 10, 12})
	selection := StreamSelection{Copy: true, Duration: 12 * time.Second, Quality: "auto"}

	playlistRecorder := httptest.NewRecorder()
	playlistRequest := httptest.NewRequest("GET", "/playlist.m3u8", nil)
	if err := svc.ServeHLSFile(playlistRecorder, playlistRequest, 33, source,
		"playlist.m3u8", "123", "firma", selection); err != nil {
		t.Fatalf("servir playlist: %v", err)
	}
	playlist := playlistRecorder.Body.String()
	if !strings.Contains(playlist, "m=copy") || !strings.Contains(playlist, "segment_00000.ts") {
		t.Fatalf("la playlist no marca los segmentos como copia:\n%s", playlist)
	}

	segmentRecorder := httptest.NewRecorder()
	segmentRequest := httptest.NewRequest("GET", "/segment_00000.ts", nil)
	if err := svc.ServeHLSFile(segmentRecorder, segmentRequest, 33, source,
		"segment_00000.ts", "123", "firma", selection); err != nil {
		t.Fatalf("servir segmento de copia: %v", err)
	}
	if segmentRecorder.Code != 200 || segmentRecorder.Body.String() != "segmento falso" {
		t.Fatalf("segmento = %d %q", segmentRecorder.Code, segmentRecorder.Body.String())
	}
}

func TestCopiaNoCuentaParaMaxTranscodes(t *testing.T) {
	svc := newCopyTestService(t, fakePlaybackCommand(0))
	svc.maxTranscodes = 1
	now := time.Now()

	transcode := newSupervisionEntry(t, svc, 41, now.Add(-2*time.Minute))
	copyEntry := newSupervisionEntry(t, svc, 42, now.Add(-time.Minute))
	copyEntry.mode = hlsModeCopy
	copyEntry.copyBounds = []float64{0, 6, 12}
	startSupervisionEntry(t, svc, transcode, 0)
	startSupervisionEntry(t, svc, copyEntry, 0)
	waitForGeneratedSegment(t, transcode, 0)
	waitForGeneratedSegment(t, copyEntry, 0)

	svc.mu.Lock()
	liveTranscodes := svc.liveTranscodes
	liveCopies := svc.liveCopies
	copyInterrupted := copyEntry.run != nil && copyEntry.run.finished() && copyEntry.run.interrupted
	transcodeLive := transcode.run != nil && !transcode.run.finished()
	svc.mu.Unlock()
	if liveTranscodes != 1 || liveCopies != 1 || copyInterrupted || !transcodeLive {
		t.Fatalf("contadores inesperados: transcodes=%d copias=%d copiaInterrumpida=%v transcodeVivo=%v",
			liveTranscodes, liveCopies, copyInterrupted, transcodeLive)
	}

	third := newSupervisionEntry(t, svc, 43, now)
	startSupervisionEntry(t, svc, third, 0)
	waitForGeneratedSegment(t, third, 0)
	svc.mu.Lock()
	firstStopped := transcode.run != nil && transcode.run.finished() && transcode.run.interrupted
	copyLive := copyEntry.run != nil && !copyEntry.run.finished()
	svc.mu.Unlock()
	if !firstStopped || !copyLive {
		t.Fatalf("la copia no debe ser víctima del límite de transcodificación: parado=%v copiaViva=%v",
			firstStopped, copyLive)
	}
}

func TestFalloCopiaMarcaYSiguientePrepareRecodifica(t *testing.T) {
	source := writeCopyTestFile(t, []byte("archivo de prueba"))
	svc := newCopyTestService(t, copyFailureCommand)
	injectCopyKeyframes(t, svc, source, []float64{0, 2, 4, 6, 8, 10, 12})

	entry, err := svc.prepare(context.Background(), 51, source,
		StreamSelection{Copy: true, Duration: 12 * time.Second, Quality: "auto"})
	if err != nil {
		t.Fatalf("prepare de copia: %v", err)
	}
	if err := svc.waitForSegment(context.Background(), entry, 0); !errors.Is(err, ErrFFmpegFailed) {
		t.Fatalf("error de copia = %v, se esperaba ErrFFmpegFailed", err)
	}

	svc.mu.Lock()
	disabled := svc.copyDisabledLocked(source)
	svc.mu.Unlock()
	if !disabled {
		t.Fatal("el archivo debe quedar marcado como no apto para copia")
	}

	next, err := svc.prepare(context.Background(), 51, source,
		StreamSelection{Copy: true, Duration: 12 * time.Second, Quality: "auto"})
	if err != nil {
		t.Fatalf("prepare tras el fallo: %v", err)
	}
	if next.mode == hlsModeCopy {
		t.Fatalf("la siguiente reproducción debe recodificar: %q", next.mode)
	}
}

func newCopyTestService(t *testing.T, command func(context.Context, string, ...string) *exec.Cmd) *Service {
	t.Helper()
	svc, err := New(Options{
		WorkDir:         t.TempDir(),
		Command:         command,
		RunIdleTimeout:  time.Hour,
		MonitorInterval: time.Hour,
	})
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

func writeCopyTestFile(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "copia.mkv")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("escribir archivo: %v", err)
	}
	return path
}

func injectCopyKeyframes(t *testing.T, svc *Service, path string, keyframes []float64) {
	t.Helper()
	absolute, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("ruta absoluta: %v", err)
	}
	stat, err := os.Stat(absolute)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	entry := &KeyframeEntry{
		Version:   keyframeCacheVersion,
		Path:      absolute,
		Size:      stat.Size(),
		MTime:     stat.ModTime().UnixNano(),
		Source:    "cues",
		Keyframes: append([]float64(nil), keyframes...),
	}
	key := keyframeCacheKey(keyframeFileInfo{
		path:  absolute,
		size:  stat.Size(),
		mtime: stat.ModTime().UnixNano(),
	})
	svc.keyframes.mu.Lock()
	svc.keyframes.entries[key] = entry
	svc.keyframes.mu.Unlock()
}

func copyFailureCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	if hasCopyVideoArg(args) {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestCopyFailureHelper")
		cmd.Env = append(os.Environ(), "GO_WANT_COPY_FAILURE_HELPER=1")
		return cmd
	}
	return fakePlaybackCommand(0)(ctx, name, args...)
}

func hasCopyVideoArg(args []string) bool {
	for index, arg := range args {
		if arg == "-c:v" && index+1 < len(args) && args[index+1] == "copy" {
			return true
		}
	}
	return false
}

func TestCopyFailureHelper(t *testing.T) {
	if os.Getenv("GO_WANT_COPY_FAILURE_HELPER") != "1" {
		return
	}
	code, _ := strconv.Atoi(os.Getenv("GO_WANT_COPY_FAILURE_CODE"))
	if code == 0 {
		code = 2
	}
	os.Exit(code)
}
