package playback

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestService(t *testing.T, ttl time.Duration) (*Service, *fakeClock) {
	t.Helper()
	clock := newFakeClock()
	svc, err := New(Options{
		TTL:     ttl,
		Now:     clock.Now,
		WorkDir: filepath.Join(t.TempDir(), "hls"),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return svc, clock
}

func TestDecide(t *testing.T) {
	tests := []struct {
		path string
		want Mode
	}{
		{path: "/media/Pelicula.mp4", want: ModeDirect},
		{path: "/media/Pelicula.M4V", want: ModeDirect},
		{path: "/media/Serie.mkv", want: ModeRemux},
		{path: "/media/Serie.avi", want: ModeRemux},
		{path: "/media/Serie.mov", want: ModeRemux},
		{path: "/media/Serie.ts", want: ModeRemux},
		{path: "/media/Serie.m2ts", want: ModeRemux},
	}

	for _, tt := range tests {
		got, err := Decide(tt.path)
		if err != nil {
			t.Fatalf("Decide(%q): %v", tt.path, err)
		}
		if got != tt.want {
			t.Fatalf("Decide(%q) = %q, se esperaba %q", tt.path, got, tt.want)
		}
	}

	if _, err := Decide("/media/notas.txt"); !errors.Is(err, ErrUnsupportedContainer) {
		t.Fatalf("txt: error = %v, se esperaba ErrUnsupportedContainer", err)
	}
}

func TestSafePath(t *testing.T) {
	root := filepath.Join("tmp", "biblioteca")

	if _, err := SafePath(root, filepath.Join(root, "Pelicula (1999).mkv")); err != nil {
		t.Fatalf("ruta válida: %v", err)
	}
	if _, err := SafePath(root, filepath.Join(root, "sub", "Serie", "S01E01.mkv")); err != nil {
		t.Fatalf("ruta válida anidada: %v", err)
	}

	for _, path := range []string{
		filepath.Join(root, "..", "fuera.mkv"),
		filepath.Join(root, "sub", "..", "..", "fuera.mkv"),
		root,
		filepath.Join(root, "..", "biblioteca-otra", "x.mkv"),
	} {
		if _, err := SafePath(root, path); !errors.Is(err, ErrOutsideLibrary) {
			t.Fatalf("SafePath(%q) = %v, se esperaba ErrOutsideLibrary", path, err)
		}
	}
}

func TestFirmaValida(t *testing.T) {
	svc, _ := newTestService(t, 15*time.Minute)

	signature, expires := svc.Sign(42)
	if signature == "" {
		t.Fatal("la firma no puede estar vacía")
	}
	if err := svc.Verify(42, strconv.FormatInt(expires.Unix(), 10), signature); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got := expires.Unix(); got == 0 {
		t.Fatal("la caducidad no puede ser cero")
	}
}

func TestFirmaCaducada(t *testing.T) {
	svc, clock := newTestService(t, 15*time.Minute)

	signature, expires := svc.Sign(42)
	expRaw := strconv.FormatInt(expires.Unix(), 10)
	if err := svc.Verify(42, expRaw, signature); err != nil {
		t.Fatalf("antes de caducar: %v", err)
	}

	clock.Advance(16 * time.Minute)
	if err := svc.Verify(42, expRaw, signature); !errors.Is(err, ErrExpired) {
		t.Fatalf("caducada: %v, se esperaba ErrExpired", err)
	}

	if err := svc.Verify(42, "no-es-un-numero", signature); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("caducidad no numérica: %v, se esperaba ErrInvalidSignature", err)
	}
}

func TestFirmaManipulada(t *testing.T) {
	svc, _ := newTestService(t, 15*time.Minute)

	signature, expires := svc.Sign(42)
	expRaw := strconv.FormatInt(expires.Unix(), 10)

	// Cambiar el ítem invalida la firma.
	if err := svc.Verify(43, expRaw, signature); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("otro ítem: %v, se esperaba ErrInvalidSignature", err)
	}

	// Manipular la firma invalida la firma.
	tampered := "A" + signature[1:]
	if signature[0] == 'A' {
		tampered = "B" + signature[1:]
	}
	if err := svc.Verify(42, expRaw, tampered); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("firma manipulada: %v, se esperaba ErrInvalidSignature", err)
	}

	if err := svc.Verify(42, expRaw, ""); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("firma vacía: %v, se esperaba ErrInvalidSignature", err)
	}
}

func TestPlaylistFirmada(t *testing.T) {
	playlist := "#EXTM3U\n#EXT-X-VERSION:3\n#EXTINF:6.0,\nsegment_00000.ts\n#EXT-X-ENDLIST\n"
	query := signedQuery("123", "firma")

	got := string(signPlaylist([]byte(playlist), query))
	if !strings.Contains(got, "segment_00000.ts"+query) {
		t.Fatalf("el segmento no lleva la firma: %q", got)
	}
	if !strings.HasPrefix(got, "#EXTM3U\n") {
		t.Fatalf("playlist inesperada: %q", got)
	}
}

func TestHLSArgsEnLista(t *testing.T) {
	input := "/media/Biblioteca/Película; con espacios & comillas \"raras\".mkv"
	args := hlsTranscodeArgs(input, 0, defaultTranscodeSettings())

	foundInput := false
	for _, arg := range args {
		if arg == input {
			foundInput = true
			break
		}
	}
	if !foundInput {
		t.Fatalf("la ruta de entrada debe ser un argumento propio: %v", args)
	}

	for _, forbidden := range []string{"sh", "bash", "cmd", "powershell", "/bin/sh", "-c"} {
		for _, arg := range args {
			if arg == forbidden {
				// "-c" es legítimo: es el flag de ffmpeg para copiar códecs.
				if forbidden == "-c" {
					continue
				}
				t.Fatalf("argumento de shell inesperado %q en %v", arg, args)
			}
		}
	}

	if !strings.Contains(strings.Join(args, "\n"), "-hls_segment_type\nmpegts") {
		t.Fatalf("faltan segmentos MPEG-TS: %v", args)
	}
}

func TestValidHLSName(t *testing.T) {
	for name, want := range map[string]bool{
		"playlist.m3u8":       true,
		"segment_00000.ts":    true,
		"segment_01514.ts":    true,
		"segment_00000.m4s":   false,
		"init.mp4":            false,
		"ffmpeg.m3u8":         false,
		"../segment_00000.ts": false,
		"segment_-1.ts":       false,
	} {
		if got := ValidHLSName(name); got != want {
			t.Errorf("ValidHLSName(%q) = %v, se esperaba %v", name, got, want)
		}
	}
}

func TestSinShellEnElCodigo(t *testing.T) {
	files := []string{"decision.go", "hls.go", "service.go", "probe.go"}
	var all string
	for _, name := range files {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("leer %s: %v", name, err)
		}
		all += string(data)
	}

	for _, forbidden := range []string{"exec.Command(", "sh -c", "cmd /c", "/bin/sh", "powershell"} {
		if strings.Contains(all, forbidden) {
			t.Fatalf("el código de reproducción no debe contener %q", forbidden)
		}
	}
	for _, required := range []string{"exec.CommandContext(", "[]string{"} {
		if !strings.Contains(all, required) {
			t.Fatalf("el código de reproducción debe contener %q", required)
		}
	}
}

func TestFirmaPorEspacioDeNombres(t *testing.T) {
	svc, _ := newTestService(t, 15*time.Minute)

	signature, expires := svc.SignResource("music", 7)
	expRaw := strconv.FormatInt(expires.Unix(), 10)
	if err := svc.VerifyResource("music", 7, expRaw, signature); err != nil {
		t.Fatalf("música: %v", err)
	}
	if err := svc.VerifyResource("video", 7, expRaw, signature); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("un id de música no puede validar como vídeo: %v", err)
	}
	if err := svc.VerifyResource("music", 8, expRaw, signature); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("otro id: %v", err)
	}
}

func TestParseCapabilities(t *testing.T) {
	if got := ParseCapabilities("hevc, HEVC10, av1"); !got.HEVC || !got.HEVC10 {
		t.Fatalf("capacidades = %+v", got)
	}
	if got := ParseCapabilities("hevc10"); !got.HEVC || !got.HEVC10 {
		t.Fatalf("hevc10 debe implicar hevc: %+v", got)
	}
	if got := ParseCapabilities("hevc10mse"); !got.HEVCMSE || !got.HEVC10MSE {
		t.Fatalf("hevc10mse debe implicar hevcmse: %+v", got)
	}
	if got := ParseCapabilities("hev1"); !got.HEV1 || got.HEVC {
		t.Fatalf("hev1 no debe implicar hevc: %+v", got)
	}
	if got := (Capabilities{HEV1: true}).QueryValue(); got != "hev1" {
		t.Fatalf("QueryValue de hev1 = %q", got)
	}
	if got := ParseCapabilities("otro"); got.HEVC || got.HEVC10 || got.HEVCMSE || got.HEVC10MSE {
		t.Fatalf("token desconocido: %+v", got)
	}
	if got := (Capabilities{HEVC: true, HEVC10: true, HEVCMSE: true, HEVC10MSE: true, HEV1: true}).QueryValue(); got != "hevc,hevc10,hevcmse,hevc10mse,hev1" {
		t.Fatalf("QueryValue = %q", got)
	}
}

func TestCanCopy(t *testing.T) {
	h264 := VideoStream{Codec: "h264", Profile: "High", PixelFormat: "yuv420p", BitDepth: 8, Height: 1080}
	h264High10 := VideoStream{Codec: "h264", Profile: "High 10", PixelFormat: "yuv420p10le", BitDepth: 10, Height: 1080}
	hevc8 := VideoStream{Codec: "hevc", Profile: "Main", PixelFormat: "yuv420p", BitDepth: 8, Height: 1080}
	hevc10 := VideoStream{Codec: "hevc", Profile: "Main 10", PixelFormat: "yuv420p10le", BitDepth: 10, Height: 1080}
	tests := []struct {
		name      string
		info      *FileInfo
		caps      Capabilities
		maxHeight int
		want      bool
	}{
		{name: "sin ficha", info: nil, want: false},
		{name: "h264 mkv", info: &FileInfo{Container: "matroska", FileName: "peli.mkv", Video: []VideoStream{h264}}, want: true},
		{name: "h264 mp4", info: &FileInfo{Container: "mp4", FileName: "peli.mp4", Video: []VideoStream{h264}}, want: true},
		{name: "avi por formato", info: &FileInfo{Container: "avi", FileName: "peli.avi", Video: []VideoStream{h264}}, want: false},
		{name: "avi por extensión", info: &FileInfo{Container: "matroska", FileName: "peli.avi", Video: []VideoStream{h264}}, want: false},
		{name: "ts por formato", info: &FileInfo{Container: "mpegts", FileName: "peli.mkv", Video: []VideoStream{h264}}, want: false},
		{name: "m2ts por extensión", info: &FileInfo{Container: "mpegts", FileName: "peli.m2ts", Video: []VideoStream{h264}}, want: false},
		{name: "h264 high10", info: &FileInfo{Container: "matroska", FileName: "peli.mkv", Video: []VideoStream{h264High10}}, want: false},
		{name: "sin pista de vídeo", info: &FileInfo{Container: "matroska", FileName: "peli.mkv"}, want: false},
		{name: "altura por encima del límite", info: &FileInfo{Container: "matroska", FileName: "peli.mkv", Video: []VideoStream{h264}}, maxHeight: 720, want: false},
		{name: "altura desconocida", info: &FileInfo{Container: "matroska", FileName: "peli.mkv", Video: []VideoStream{{Codec: "h264", PixelFormat: "yuv420p"}}}, maxHeight: 720, want: false},
		{name: "hevc 8 sin mse", info: &FileInfo{Container: "matroska", FileName: "peli.mkv", Video: []VideoStream{hevc8}}, caps: Capabilities{HEVC: true}, want: false},
		{name: "hevc 8 con mse", info: &FileInfo{Container: "matroska", FileName: "peli.mkv", Video: []VideoStream{hevc8}}, caps: Capabilities{HEVCMSE: true}, want: true},
		{name: "hevc 10 con mse 8", info: &FileInfo{Container: "matroska", FileName: "peli.mkv", Video: []VideoStream{hevc10}}, caps: Capabilities{HEVCMSE: true}, want: false},
		{name: "hevc 10 con mse 10", info: &FileInfo{Container: "matroska", FileName: "peli.mkv", Video: []VideoStream{hevc10}}, caps: Capabilities{HEVC10MSE: true}, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CanCopy(test.info, test.caps, test.maxHeight); got != test.want {
				t.Fatalf("CanCopy = %v, se esperaba %v", got, test.want)
			}
		})
	}
}

func TestHEVCTagDirecto(t *testing.T) {
	hevc8 := VideoStream{Codec: "hevc", Profile: "Main", PixelFormat: "yuv420p", BitDepth: 8}
	hevc10 := VideoStream{Codec: "hevc", Profile: "Main 10", PixelFormat: "yuv420p10le", BitDepth: 10}
	tests := []struct {
		name  string
		video VideoStream
		caps  Capabilities
		want  bool
	}{
		{name: "hvc1 directo", video: withCodecTag(hevc8, "hvc1"), caps: Capabilities{HEVC: true}, want: true},
		{name: "hev1 sin token", video: withCodecTag(hevc8, "hev1"), caps: Capabilities{HEVC: true}, want: false},
		{name: "hev1 con token", video: withCodecTag(hevc8, "hev1"), caps: Capabilities{HEVC: true, HEV1: true}, want: true},
		{name: "etiqueta vacía", video: hevc8, caps: Capabilities{HEVC: true}, want: false},
		{name: "otra etiqueta", video: withCodecTag(hevc8, "hev0"), caps: Capabilities{HEVC: true, HEV1: true}, want: false},
		{name: "hev1 10 bits con hevc10", video: withCodecTag(hevc10, "hev1"), caps: Capabilities{HEVC: true, HEVC10: true}, want: false},
		{name: "hev1 10 bits con hevc10 y token", video: withCodecTag(hevc10, "hev1"), caps: Capabilities{HEVC: true, HEVC10: true, HEV1: true}, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := directVideo(&FileInfo{Video: []VideoStream{test.video}}, test.caps); got != test.want {
				t.Fatalf("directVideo = %v, se esperaba %v", got, test.want)
			}
		})
	}
}

func withCodecTag(video VideoStream, tag string) VideoStream {
	video.CodecTag = tag
	return video
}

func TestDecideFor(t *testing.T) {
	h264 := VideoStream{Codec: "h264", Profile: "High", PixelFormat: "yuv420p", BitDepth: 8}
	hevc := VideoStream{Codec: "hevc", CodecTag: "hvc1", Profile: "Main", PixelFormat: "yuv420p", BitDepth: 8}
	hevc10 := VideoStream{Codec: "hevc", CodecTag: "hvc1", Profile: "Main 10", PixelFormat: "yuv420p10le", BitDepth: 10}
	aac := AudioStream{Codec: "aac", Default: true}
	ac3 := AudioStream{Codec: "ac3", Default: true}
	tests := []struct {
		name string
		path string
		info *FileInfo
		caps Capabilities
		want Mode
	}{
		{name: "h264 mp4 aac directo", path: "/media/peli.mp4", info: &FileInfo{Video: []VideoStream{h264}, Audio: []AudioStream{aac}}, want: ModeDirect},
		{name: "h264 mp4 sin audio directo", path: "/media/peli.m4v", info: &FileInfo{Video: []VideoStream{h264}}, want: ModeDirect},
		{name: "h264 mp4 ac3 recodifica", path: "/media/peli.mp4", info: &FileInfo{Video: []VideoStream{h264}, Audio: []AudioStream{ac3}}, want: ModeRemux},
		{name: "hevc sin capacidades recodifica", path: "/media/peli.mp4", info: &FileInfo{Video: []VideoStream{hevc}, Audio: []AudioStream{aac}}, want: ModeRemux},
		{name: "hevc con capacidad directo", path: "/media/peli.mp4", info: &FileInfo{Video: []VideoStream{hevc}, Audio: []AudioStream{aac}}, caps: Capabilities{HEVC: true}, want: ModeDirect},
		{name: "hevc 10 bits sin hevc10 recodifica", path: "/media/peli.mp4", info: &FileInfo{Video: []VideoStream{hevc10}, Audio: []AudioStream{aac}}, caps: Capabilities{HEVC: true}, want: ModeRemux},
		{name: "hevc 10 bits con hevc10 directo", path: "/media/peli.mp4", info: &FileInfo{Video: []VideoStream{hevc10}, Audio: []AudioStream{aac}}, caps: Capabilities{HEVC: true, HEVC10: true}, want: ModeDirect},
		{name: "h264 high10 recodifica", path: "/media/peli.mp4", info: &FileInfo{Video: []VideoStream{{Codec: "h264", Profile: "High 10", PixelFormat: "yuv420p10le", BitDepth: 10}}, Audio: []AudioStream{aac}}, caps: Capabilities{HEVC: true, HEVC10: true}, want: ModeRemux},
		{name: "mkv siempre recodifica", path: "/media/peli.mkv", info: &FileInfo{Video: []VideoStream{h264}, Audio: []AudioStream{aac}}, want: ModeRemux},
		{name: "av1 mp4 recodifica", path: "/media/peli.mp4", info: &FileInfo{Video: []VideoStream{{Codec: "av1", PixelFormat: "yuv420p", BitDepth: 8}}, Audio: []AudioStream{aac}}, want: ModeRemux},
		{name: "sin info mp4 directo", path: "/media/peli.mp4", info: nil, want: ModeDirect},
		{name: "sin info mkv recodifica", path: "/media/peli.mkv", info: nil, want: ModeRemux},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := DecideFor(test.path, test.info, test.caps)
			if err != nil {
				t.Fatalf("DecideFor: %v", err)
			}
			if got != test.want {
				t.Fatalf("modo = %q, se esperaba %q", got, test.want)
			}
		})
	}
	if _, err := DecideFor("/media/notas.txt", nil, Capabilities{}); !errors.Is(err, ErrUnsupportedContainer) {
		t.Fatalf("extensión desconocida: %v", err)
	}
}
