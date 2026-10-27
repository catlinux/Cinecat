package playback

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestPlaybackHelperProcess es un FFmpeg falso: crea unos segmentos y se queda
// vivo hasta que el contexto lo mata. Solo actúa cuando lo arranca la prueba.
func TestPlaybackHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_PLAYBACK_HELPER") != "1" {
		return
	}
	start := 0
	for index, arg := range os.Args {
		if (arg == "-start_number" || arg == "-segment_start_number") && index+1 < len(os.Args) {
			start, _ = strconv.Atoi(os.Args[index+1])
		}
	}
	ahead := 0
	if value := os.Getenv("PLAYBACK_FAKE_AHEAD"); value != "" {
		ahead, _ = strconv.Atoi(value)
	}

	var playlist strings.Builder
	playlist.WriteString("#EXTM3U\n")
	for index := 0; index <= ahead; index++ {
		name := segmentName(start + index)
		if err := os.WriteFile(name, []byte("segmento falso"), 0o644); err != nil {
			os.Exit(2)
		}
		fmt.Fprintf(&playlist, "#EXTINF:6.000000,\n%s\n", name)
	}
	if err := os.WriteFile(ffmpegPlaylistFile, []byte(playlist.String()), 0o644); err != nil {
		os.Exit(2)
	}
	time.Sleep(24 * time.Hour)
}

func fakePlaybackCommand(ahead int) func(context.Context, string, ...string) *exec.Cmd {
	return func(ctx context.Context, _ string, args ...string) *exec.Cmd {
		helperArgs := append([]string{"-test.run=TestPlaybackHelperProcess", "--"}, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], helperArgs...)
		cmd.Env = append(os.Environ(),
			"GO_WANT_PLAYBACK_HELPER=1",
			fmt.Sprintf("PLAYBACK_FAKE_AHEAD=%d", ahead),
		)
		return cmd
	}
}

func newSupervisionEntry(t *testing.T, svc *Service, itemID int64, lastRequest time.Time) *hlsEntry {
	t.Helper()
	dir := filepath.Join(svc.workDir, fmt.Sprintf("item-%d-test", itemID))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("crear directorio de la entrada: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "playlist.m3u8"),
		[]byte("#EXTM3U\n#EXT-X-PLAYLIST-TYPE:VOD\n"), 0o644); err != nil {
		t.Fatalf("crear playlist: %v", err)
	}
	entry := &hlsEntry{
		itemID:           itemID,
		dir:              dir,
		inputPath:        "entrada.mkv",
		lastUsed:         lastRequest,
		lastRequest:      lastRequest,
		playlistFile:     "playlist.m3u8",
		transcode:        defaultTranscodeSettings(),
		highestRequested: 0,
		segments:         1000,
	}
	svc.mu.Lock()
	svc.entries[hlsEntryKey{itemID: itemID, transcode: fmt.Sprintf("test-%d", itemID)}] = entry
	svc.mu.Unlock()
	return entry
}

func startSupervisionEntry(t *testing.T, svc *Service, entry *hlsEntry, start int) {
	t.Helper()
	svc.mu.Lock()
	err := svc.startRunLocked(entry, start)
	svc.mu.Unlock()
	if err != nil {
		t.Fatalf("startRunLocked: %v", err)
	}
}

func waitForGeneratedSegment(t *testing.T, entry *hlsEntry, index int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if fileNotEmpty(filepath.Join(entry.dir, segmentName(index))) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("el segmento %d no apareció", index)
}

func waitForFinishedRun(t *testing.T, svc *Service, entry *hlsEntry) *hlsRun {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		svc.mu.Lock()
		run := entry.run
		finished := run != nil && run.finished()
		svc.mu.Unlock()
		if finished {
			return run
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("el proceso falso no terminó")
	return nil
}

func TestSupervisionParaPorInactividadYRelanza(t *testing.T) {
	svc, err := New(Options{
		WorkDir:         t.TempDir(),
		Command:         fakePlaybackCommand(1),
		RunIdleTimeout:  30 * time.Millisecond,
		MonitorInterval: 5 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	entry := newSupervisionEntry(t, svc, 1, time.Now().Add(time.Minute))
	startSupervisionEntry(t, svc, entry, 0)
	waitForGeneratedSegment(t, entry, 0)

	svc.mu.Lock()
	entry.lastRequest = svc.now().Add(-time.Second)
	svc.mu.Unlock()

	run := waitForFinishedRun(t, svc, entry)
	svc.mu.Lock()
	interrupted := run.interrupted
	live := svc.liveTranscodes
	svc.mu.Unlock()
	if !interrupted || live != 0 {
		t.Fatalf("parada por inactividad inesperada: interrumpido=%v vivos=%d", interrupted, live)
	}
	if !fileNotEmpty(filepath.Join(entry.dir, segmentName(0))) {
		t.Fatal("la parada no debe borrar los segmentos")
	}

	// El segmento 2 no existe; la petición debe relanzar FFmpeg en ese punto.
	if err := svc.waitForSegment(context.Background(), entry, 2); err != nil {
		t.Fatalf("waitForSegment tras inactividad: %v", err)
	}
	svc.mu.Lock()
	start := -1
	if entry.run != nil {
		start = entry.run.start
	}
	svc.mu.Unlock()
	if start != 2 {
		t.Fatalf("el proceso relanzado empieza en %d, se esperaba 2", start)
	}
}

func TestSupervisionParaPorAdelantoYRelanza(t *testing.T) {
	svc, err := New(Options{
		WorkDir:          t.TempDir(),
		Command:          fakePlaybackCommand(10),
		MaxAheadSegments: 2,
		RunIdleTimeout:   time.Hour,
		MonitorInterval:  5 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	entry := newSupervisionEntry(t, svc, 2, time.Now())
	startSupervisionEntry(t, svc, entry, 0)
	run := waitForFinishedRun(t, svc, entry)
	svc.mu.Lock()
	interrupted := run.interrupted
	svc.mu.Unlock()
	if !interrupted {
		t.Fatal("el proceso debería pararse por adelanto")
	}
	if !fileNotEmpty(filepath.Join(entry.dir, segmentName(0))) {
		t.Fatal("los segmentos generados deben conservarse")
	}

	if err := svc.waitForSegment(context.Background(), entry, 30); err != nil {
		t.Fatalf("waitForSegment tras adelanto: %v", err)
	}
	svc.mu.Lock()
	start := -1
	if entry.run != nil {
		start = entry.run.start
	}
	svc.mu.Unlock()
	if start != 30 {
		t.Fatalf("el proceso relanzado empieza en %d, se esperaba 30", start)
	}
}

func TestSupervisionLimitaLaConcurrencia(t *testing.T) {
	svc, err := New(Options{
		WorkDir:         t.TempDir(),
		Command:         fakePlaybackCommand(0),
		MaxTranscodes:   2,
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

	now := time.Now()
	first := newSupervisionEntry(t, svc, 10, now.Add(-2*time.Minute))
	second := newSupervisionEntry(t, svc, 11, now.Add(-time.Minute))
	third := newSupervisionEntry(t, svc, 12, now)
	startSupervisionEntry(t, svc, first, 0)
	startSupervisionEntry(t, svc, second, 0)
	waitForGeneratedSegment(t, first, 0)
	waitForGeneratedSegment(t, second, 0)
	startSupervisionEntry(t, svc, third, 0)
	waitForGeneratedSegment(t, third, 0)

	svc.mu.Lock()
	firstFinished := first.run != nil && first.run.finished() && first.run.interrupted
	secondLive := second.run != nil && !second.run.finished()
	thirdLive := third.run != nil && !third.run.finished()
	live := svc.liveTranscodes
	svc.mu.Unlock()
	if !firstFinished {
		t.Fatal("debe detenerse la entrada con la petición más antigua")
	}
	if !secondLive || !thirdLive || live != 2 {
		t.Fatalf("concurrencia inesperada: segunda=%v tercera=%v vivos=%d", secondLive, thirdLive, live)
	}
}

func TestSupervisionCloseParaTodosLosProcesos(t *testing.T) {
	svc, err := New(Options{
		WorkDir:         t.TempDir(),
		Command:         fakePlaybackCommand(0),
		MaxTranscodes:   2,
		RunIdleTimeout:  time.Hour,
		MonitorInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		_ = svc.Close()
	})

	first := newSupervisionEntry(t, svc, 20, time.Now())
	second := newSupervisionEntry(t, svc, 21, time.Now())
	startSupervisionEntry(t, svc, first, 0)
	startSupervisionEntry(t, svc, second, 0)
	waitForGeneratedSegment(t, first, 0)
	waitForGeneratedSegment(t, second, 0)

	if err := svc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if svc.liveTranscodes != 0 {
		t.Fatalf("vivos tras Close = %d", svc.liveTranscodes)
	}
	if !first.run.finished() || !second.run.finished() {
		t.Fatal("Close debe esperar a que terminen los procesos")
	}
}
