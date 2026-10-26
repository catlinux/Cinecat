package playback

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
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

	durations := playlistSegmentDurations(t, filepath.Join(entry.dir, "playlist.m3u8"))
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
