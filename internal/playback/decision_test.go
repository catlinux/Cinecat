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
	playlist := "#EXTM3U\n#EXT-X-VERSION:7\n#EXT-X-MAP:URI=\"init.mp4\"\n#EXTINF:6.0,\nsegment_00000.m4s\n#EXT-X-ENDLIST\n"
	query := signedQuery("123", "firma")

	got := string(signPlaylist([]byte(playlist), query))
	if !strings.Contains(got, `#EXT-X-MAP:URI="init.mp4`+query+`"`) {
		t.Fatalf("el init no lleva la firma: %q", got)
	}
	if !strings.Contains(got, "segment_00000.m4s"+query) {
		t.Fatalf("el segmento no lleva la firma: %q", got)
	}
	if !strings.HasPrefix(got, "#EXTM3U\n") {
		t.Fatalf("playlist inesperada: %q", got)
	}
}

func TestHLSArgsEnLista(t *testing.T) {
	input := "/media/Biblioteca/Película; con espacios & comillas \"raras\".mkv"
	args := hlsArgs(input)

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-c:v copy") || !strings.Contains(joined, "-c:a copy") {
		t.Fatalf("faltan los argumentos de copia de códecs: %v", args)
	}

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

	if !strings.Contains(strings.Join(args, "\n"), "-hls_segment_type\nfmp4") {
		t.Fatalf("faltan segmentos fMP4: %v", args)
	}
}

func TestSinShellEnElCodigo(t *testing.T) {
	files := []string{"decision.go", "hls.go", "service.go"}
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
