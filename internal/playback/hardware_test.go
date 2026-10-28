package playback

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestHardwareArgsExactos(t *testing.T) {
	input := `C:\media\Pelicula.mkv`
	settings := TranscodeSettings{MaxHeight: 720, CRF: 20, Preset: "veryfast", Threads: 4, Audio: 1}
	device := "/dev/dri/renderD128"

	cpu := []string{
		"-hide_banner", "-loglevel", "error", "-nostdin", "-y",
		"-vaapi_device", device,
		"-ss", "12",
		"-i", input,
		"-map", "0:V:0", "-map", "0:a:1?",
		"-vf", `scale=-2:trunc(min(720\,ih)/2)*2,format=nv12,hwupload`,
		"-c:v", "h264_vaapi", "-qp", "22", "-g", "250",
		"-force_key_frames", "expr:gte(t,n_forced*6)",
		"-c:a", "aac", "-ac", "2", "-b:a", "160k",
		"-output_ts_offset", "22",
		"-f", "hls", "-hls_time", "6", "-hls_playlist_type", "event",
		"-hls_flags", "temp_file", "-hls_segment_type", "mpegts",
		"-start_number", "2",
		"-hls_segment_filename", "segment_%05d.ts",
		ffmpegPlaylistFile,
	}
	if got := hlsHardwareArgs(input, 2, settings, false, device); !reflect.DeepEqual(got, cpu) {
		t.Fatalf("argumentos CPU+GPU:\n%q\nse esperaba:\n%q", got, cpu)
	}

	full := []string{
		"-hide_banner", "-loglevel", "error", "-nostdin", "-y",
		"-hwaccel", "vaapi", "-hwaccel_device", device, "-hwaccel_output_format", "vaapi",
		"-ss", "12",
		"-i", input,
		"-map", "0:V:0", "-map", "0:a:1?",
		"-c:v", "h264_vaapi", "-qp", "22", "-g", "250",
		"-force_key_frames", "expr:gte(t,n_forced*6)",
		"-c:a", "aac", "-ac", "2", "-b:a", "160k",
		"-output_ts_offset", "22",
		"-f", "hls", "-hls_time", "6", "-hls_playlist_type", "event",
		"-hls_flags", "temp_file", "-hls_segment_type", "mpegts",
		"-start_number", "2",
		"-hls_segment_filename", "segment_%05d.ts",
		ffmpegPlaylistFile,
	}
	if got := hlsHardwareArgs(input, 2, settings, true, device); !reflect.DeepEqual(got, full) {
		t.Fatalf("argumentos GPU completa:\n%q\nse esperaba:\n%q", got, full)
	}
}

func TestSoftwareArgsNoCambian(t *testing.T) {
	input := `C:\media\Pelicula.mkv`
	settings := TranscodeSettings{MaxHeight: 720, CRF: 20, Preset: "veryfast", Threads: 4, Audio: 1}
	want := []string{
		"-hide_banner", "-loglevel", "error", "-nostdin", "-y",
		"-ss", "12",
		"-i", input,
		"-map", "0:V:0", "-map", "0:a:1?",
		"-vf", `scale=-2:trunc(min(720\,ih)/2)*2`,
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "20", "-threads", "4", "-pix_fmt", "yuv420p",
		"-force_key_frames", "expr:gte(t,n_forced*6)",
		"-c:a", "aac", "-ac", "2", "-b:a", "160k",
		"-output_ts_offset", "22",
		"-f", "hls", "-hls_time", "6", "-hls_playlist_type", "event",
		"-hls_flags", "temp_file", "-hls_segment_type", "mpegts",
		"-start_number", "2",
		"-hls_segment_filename", "segment_%05d.ts",
		ffmpegPlaylistFile,
	}
	if got := hlsTranscodeArgs(input, 2, settings); !reflect.DeepEqual(got, want) {
		t.Fatalf("argumentos software:\n%q\nse esperaba:\n%q", got, want)
	}
}

func TestHardwareFullRoute(t *testing.T) {
	h264 := VideoStream{Codec: "h264", Profile: "High", PixelFormat: "yuv420p", BitDepth: 8, Height: 1080, FieldOrder: "progressive"}
	h264Interlaced := VideoStream{Codec: "h264", Profile: "High", PixelFormat: "yuv420p", BitDepth: 8, Height: 1080, FieldOrder: "tt"}
	h26410 := VideoStream{Codec: "h264", Profile: "High 10", PixelFormat: "yuv420p10le", BitDepth: 10, Height: 1080, FieldOrder: "progressive"}
	hevc8 := VideoStream{Codec: "hevc", Profile: "Main", PixelFormat: "yuv420p", BitDepth: 8, Height: 1080, FieldOrder: "progressive"}
	hevc10 := VideoStream{Codec: "hevc", Profile: "Main 10", PixelFormat: "yuv420p10le", BitDepth: 10, Height: 1080, FieldOrder: "progressive"}

	tests := []struct {
		name      string
		info      *FileInfo
		maxHeight int
		want      bool
	}{
		{name: "H264 1080 sin escalar", info: &FileInfo{Video: []VideoStream{h264}}, maxHeight: 1080, want: true},
		{name: "H264 1080 a 720", info: &FileInfo{Video: []VideoStream{h264}}, maxHeight: 720, want: false},
		{name: "H264 entrelazado", info: &FileInfo{Video: []VideoStream{h264Interlaced}}, maxHeight: 1080, want: false},
		{name: "H264 10 bits", info: &FileInfo{Video: []VideoStream{h26410}}, maxHeight: 1080, want: false},
		{name: "HEVC 1080 sin escalar", info: &FileInfo{Video: []VideoStream{hevc8}}, maxHeight: 1080, want: true},
		{name: "HEVC 10 bits", info: &FileInfo{Video: []VideoStream{hevc10}}, maxHeight: 1080, want: false},
		{name: "sin ficha", info: nil, maxHeight: 1080, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := hardwareFullRoute(test.info, test.maxHeight); got != test.want {
				t.Fatalf("hardwareFullRoute = %v, se esperaba %v", got, test.want)
			}
		})
	}
}

func TestHardwareAvailableConComandoFalso(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("VA-API solo se prueba en Linux")
	}
	device := filepath.Join(t.TempDir(), "renderD128")
	if err := os.WriteFile(device, []byte("dispositivo de prueba"), 0o644); err != nil {
		t.Fatalf("crear dispositivo: %v", err)
	}
	t.Setenv("CINECAT_VAAPI_DEVICE", device)

	var calls int32
	success := func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		atomic.AddInt32(&calls, 1)
		return hardwareHelperCommand(ctx, 0)
	}
	svc := newHardwareService(t, true, success)
	if !svc.HardwareAvailable(context.Background()) {
		t.Fatal("se esperaba hardware disponible")
	}
	if !svc.HardwareAvailable(context.Background()) {
		t.Fatal("la segunda consulta debe recordar el resultado")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("comprobaciones = %d, se esperaba 1", got)
	}

	failureCalls := int32(0)
	failure := func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		atomic.AddInt32(&failureCalls, 1)
		return hardwareHelperCommand(ctx, 2)
	}
	failed := newHardwareService(t, true, failure)
	if failed.HardwareAvailable(context.Background()) {
		t.Fatal("un fallo de comprobación no debe habilitar el hardware")
	}
	if got := atomic.LoadInt32(&failureCalls); got != 1 {
		t.Fatalf("comprobaciones fallidas = %d, se esperaba 1", got)
	}

	offCalls := int32(0)
	off := newHardwareService(t, false, func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		atomic.AddInt32(&offCalls, 1)
		return hardwareHelperCommand(ctx, 0)
	})
	if off.HardwareAvailable(context.Background()) {
		t.Fatal("playback.hardware=off no debe usar hardware")
	}
	if atomic.LoadInt32(&offCalls) != 0 {
		t.Fatal("con el ajuste off no se debe ejecutar la comprobación")
	}
}

func TestHardwareFallaYRelanzaSoftware(t *testing.T) {
	svc := newHardwareService(t, true, hardwareFailureCommand)
	entry := newSupervisionEntry(t, svc, 71, time.Now())
	entry.hardware = true
	entry.hardwareDevice = "/dev/dri/renderD128"
	startSupervisionEntry(t, svc, entry, 0)

	if err := svc.waitForSegment(context.Background(), entry, 0); err != nil {
		t.Fatalf("waitForSegment: %v", err)
	}
	svc.mu.Lock()
	hardware := entry.hardware
	runHardware := entry.run != nil && entry.run.hardware
	disabled := svc.hardwareDisabledLocked(entry.inputPath)
	svc.mu.Unlock()
	if hardware || runHardware || !disabled {
		t.Fatalf("fallback inesperado: hardware=%v run=%v marcado=%v", hardware, runHardware, disabled)
	}
}

func TestHardwareFallaAlArrancarYRelanzaSoftware(t *testing.T) {
	svc := newHardwareService(t, true, hardwareStartFailureCommand)
	entry := newSupervisionEntry(t, svc, 72, time.Now())
	entry.hardware = true
	entry.hardwareDevice = "/dev/dri/renderD128"
	startSupervisionEntry(t, svc, entry, 0)
	waitForGeneratedSegment(t, entry, 0)
	svc.mu.Lock()
	hardware := entry.hardware
	disabled := svc.hardwareDisabledLocked(entry.inputPath)
	svc.mu.Unlock()
	if hardware || !disabled {
		t.Fatalf("fallback al arrancar inesperado: hardware=%v marcado=%v", hardware, disabled)
	}
}

func newHardwareService(t *testing.T, enabled bool, command func(context.Context, string, ...string) *exec.Cmd) *Service {
	t.Helper()
	svc, err := New(Options{
		WorkDir:         t.TempDir(),
		Command:         command,
		RunIdleTimeout:  time.Hour,
		MonitorInterval: time.Hour,
		Settings: func() TranscodeSettings {
			return TranscodeSettings{MaxHeight: 1080, CRF: 20, Preset: "veryfast", Threads: 2, Hardware: enabled}
		},
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

func hardwareHelperCommand(ctx context.Context, code int) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHardwareHelperProcess")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HARDWARE_HELPER=1",
		"GO_WANT_HARDWARE_EXIT="+strconv.Itoa(code),
	)
	return cmd
}

func TestHardwareHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HARDWARE_HELPER") != "1" {
		return
	}
	code, _ := strconv.Atoi(os.Getenv("GO_WANT_HARDWARE_EXIT"))
	os.Exit(code)
}

func hardwareFailureCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	for _, arg := range args {
		if arg == "-vaapi_device" || arg == "-hwaccel" {
			return hardwareHelperCommand(ctx, 2)
		}
	}
	return fakePlaybackCommand(0)(ctx, name, args...)
}

func hardwareStartFailureCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	for _, arg := range args {
		if arg == "-vaapi_device" || arg == "-hwaccel" {
			return nil
		}
	}
	return fakePlaybackCommand(0)(ctx, name, args...)
}

func TestHardwareDevicePathDefault(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("VA-API solo se prueba en Linux")
	}
	t.Setenv("CINECAT_VAAPI_DEVICE", "")
	if got := hardwareDevicePath(); got != defaultVAAPIDevice {
		t.Fatalf("dispositivo por defecto = %q", got)
	}
	t.Setenv("CINECAT_VAAPI_DEVICE", "/dev/dri/renderD129")
	if got := hardwareDevicePath(); got != "/dev/dri/renderD129" {
		t.Fatalf("dispositivo configurado = %q", got)
	}
}
