package playback

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func requireVideoTools(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg no está disponible en PATH; se omite la prueba de keyframes")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe no está disponible en PATH; se omite la prueba de keyframes")
	}
	out, err := exec.Command("ffmpeg", "-hide_banner", "-encoders").Output()
	if err != nil || !strings.Contains(string(out), "libx264") {
		t.Skip("ffmpeg no incluye libx264; se omite la prueba de keyframes")
	}
}

func generateKeyframeVideo(t *testing.T, target, duration string, extra ...string) string {
	t.Helper()
	args := []string{
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=duration=" + duration + ":size=160x120:rate=10",
		"-c:v", "libx264", "-preset", "ultrafast", "-pix_fmt", "yuv420p",
	}
	args = append(args, extra...)
	args = append(args, "-an", target)

	cmd := exec.Command("ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generar vídeo: %v: %s", err, out)
	}
	return target
}

func TestMaxKeyframeIntervalNormal(t *testing.T) {
	requireVideoTools(t)
	file := generateKeyframeVideo(t, filepath.Join(t.TempDir(), "normal.mkv"), "2",
		"-g", "5", "-keyint_min", "5", "-sc_threshold", "0")

	interval, err := NewProber().MaxKeyframeInterval(context.Background(), file)
	if err != nil {
		t.Fatalf("MaxKeyframeInterval: %v", err)
	}
	if interval > MaxViableKeyframeInterval {
		t.Fatalf("intervalo = %s, debería ser viable (<= %s)", interval, MaxViableKeyframeInterval)
	}
	if err := NewProber().CheckViable(context.Background(), file); err != nil {
		t.Fatalf("CheckViable: %v", err)
	}
}

func TestMaxKeyframeIntervalDisperso(t *testing.T) {
	requireVideoTools(t)
	file := generateKeyframeVideo(t, filepath.Join(t.TempDir(), "disperso.mkv"), "40",
		"-x264-params", "keyint=9999:min-keyint=9999:scenecut=0",
		"-force_key_frames", "expr:gte(t,20)")

	interval, err := NewProber().MaxKeyframeInterval(context.Background(), file)
	if err != nil {
		t.Fatalf("MaxKeyframeInterval: %v", err)
	}
	if interval <= MaxViableKeyframeInterval {
		t.Fatalf("intervalo = %s, debería superar %s", interval, MaxViableKeyframeInterval)
	}

	err = NewProber().CheckViable(context.Background(), file)
	if !errors.Is(err, ErrKeyframeIntervalTooLarge) {
		t.Fatalf("CheckViable = %v, se esperaba ErrKeyframeIntervalTooLarge", err)
	}
}

func TestMaxKeyframeIntervalUnSoloKeyframe(t *testing.T) {
	requireVideoTools(t)
	file := generateKeyframeVideo(t, filepath.Join(t.TempDir(), "unico.mkv"), "3",
		"-x264-params", "keyint=9999:min-keyint=9999:scenecut=0")

	_, err := NewProber().MaxKeyframeInterval(context.Background(), file)
	if !errors.Is(err, ErrInsufficientKeyframes) {
		t.Fatalf("error = %v, se esperaba ErrInsufficientKeyframes", err)
	}
	if err := NewProber().CheckViable(context.Background(), file); !errors.Is(err, ErrKeyframeIntervalTooLarge) {
		t.Fatalf("CheckViable = %v, se esperaba ErrKeyframeIntervalTooLarge", err)
	}
}

func TestProberSinFFprobe(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	if _, err := NewProber().MaxKeyframeInterval(context.Background(), "x.mkv"); !errors.Is(err, ErrFFprobeNotFound) {
		t.Fatalf("error = %v, se esperaba ErrFFprobeNotFound", err)
	}
}

func TestParseKeyframeTimes(t *testing.T) {
	output := "0.000000,0.000000,K__\n0.200000,0.200000,___\n1.500000,1.500000,K__\n"
	times := parseKeyframeTimes(output)
	if len(times) != 2 || times[0] != 0 || times[1] != 1.5 {
		t.Fatalf("tiempos inesperados: %v", times)
	}
}

func generateRawH264(t *testing.T, target string) string {
	t.Helper()
	args := []string{
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=duration=1:size=160x120:rate=10",
		"-c:v", "libx264", "-preset", "ultrafast", "-f", "h264",
		target,
	}
	if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("generar h264: %v: %s", err, out)
	}
	return target
}

func TestVideoDuration(t *testing.T) {
	requireVideoTools(t)
	file := generateKeyframeVideo(t, filepath.Join(t.TempDir(), "duracion.mkv"), "2",
		"-g", "5", "-keyint_min", "5", "-sc_threshold", "0")

	duration, err := NewProber().VideoDuration(context.Background(), file)
	if err != nil {
		t.Fatalf("VideoDuration: %v", err)
	}
	if duration < 1900*time.Millisecond || duration > 2100*time.Millisecond {
		t.Fatalf("duración = %s, se esperaban ~2 s", duration)
	}
}

func TestVideoDurationDesconocida(t *testing.T) {
	requireVideoTools(t)
	raw := generateRawH264(t, filepath.Join(t.TempDir(), "raw.h264"))

	if _, err := NewProber().VideoDuration(context.Background(), raw); !errors.Is(err, ErrDurationUnknown) {
		t.Fatalf("error = %v, se esperaba ErrDurationUnknown", err)
	}
}
