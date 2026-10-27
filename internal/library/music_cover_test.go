package library

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestMusicCovererCacheEInvalidacion(t *testing.T) {
	cacheDir := t.TempDir()
	source := filepath.Join(t.TempDir(), "tema.mp3")
	if err := os.WriteFile(source, []byte("audio"), 0o644); err != nil {
		t.Fatalf("crear audio: %v", err)
	}
	var mu sync.Mutex
	calls := 0
	coverer := NewMusicCoverer(cacheDir).WithRunner(func(ctx context.Context, args []string) error {
		mu.Lock()
		calls++
		mu.Unlock()
		return os.WriteFile(args[len(args)-1], []byte{0xff, 0xd8, 0xff, 0xd9}, 0o644)
	})

	first, err := coverer.Get(context.Background(), 7, source, 100)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if filepath.Base(first) != "7-100.jpg" {
		t.Fatalf("caché = %q", first)
	}
	second, err := coverer.Get(context.Background(), 7, source, 100)
	if err != nil || second != first || calls != 1 {
		t.Fatalf("segunda llamada: %q %v calls=%d", second, err, calls)
	}
	third, err := coverer.Get(context.Background(), 7, source, 200)
	if err != nil || third == first || calls != 2 {
		t.Fatalf("nueva mtime: %q %v calls=%d", third, err, calls)
	}
}

func TestMusicCovererTimeout(t *testing.T) {
	cacheDir := t.TempDir()
	source := filepath.Join(t.TempDir(), "tema.mp3")
	if err := os.WriteFile(source, []byte("audio"), 0o644); err != nil {
		t.Fatalf("crear audio: %v", err)
	}
	coverer := NewMusicCoverer(cacheDir).WithLimits(time.Second, 20*time.Millisecond)
	coverer.WithRunner(func(ctx context.Context, args []string) error {
		<-ctx.Done()
		return ctx.Err()
	})
	if _, err := coverer.Get(context.Background(), 1, source, 1); !errors.Is(err, ErrThumbnailUnavailable) {
		t.Fatalf("error = %v, se esperaba ErrThumbnailUnavailable", err)
	}
}
