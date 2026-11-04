package playback

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

func newDownloadTestService(t *testing.T, maxTranscodes int) *Service {
	t.Helper()
	svc, err := New(Options{
		WorkDir:         t.TempDir(),
		Command:         fakePlaybackCommand(0),
		MaxTranscodes:   maxTranscodes,
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

func startAs(t *testing.T, svc *Service, entry *hlsEntry, download bool) error {
	t.Helper()
	svc.mu.Lock()
	defer svc.mu.Unlock()
	entry.requestDownload = download
	return svc.startRunLocked(entry, 0)
}

func TestSignDownloadUsaLaVidaLargaConfigurada(t *testing.T) {
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	svc, err := New(Options{
		WorkDir:         t.TempDir(),
		Now:             func() time.Time { return now },
		DownloadTTLFunc: func() time.Duration { return 5 * time.Hour },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })

	signature, expires := svc.SignDownload(7)
	if want := now.Add(5 * time.Hour); !expires.Equal(want) {
		t.Fatalf("caduca %v, se esperaba %v", expires, want)
	}
	exp := strconv.FormatInt(expires.Unix(), 10)
	if err := svc.Verify(7, exp, signature); err != nil {
		t.Fatalf("la firma de descarga debe validar como la normal: %v", err)
	}
	// La reproducción normal conserva su vida corta.
	if _, normal := svc.Sign(7); !normal.Equal(now.Add(DefaultURLTTL)) {
		t.Fatalf("Sign caduca %v", normal)
	}
}

func TestSignDownloadPorDefectoSon72Horas(t *testing.T) {
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	svc, err := New(Options{WorkDir: t.TempDir(), Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = svc.Close() })
	if _, expires := svc.SignDownload(1); !expires.Equal(now.Add(72 * time.Hour)) {
		t.Fatalf("caduca %v", expires)
	}
}

func TestDescargaNoExpulsaAUnaReproduccion(t *testing.T) {
	svc := newDownloadTestService(t, 1)
	now := time.Now()
	playing := newSupervisionEntry(t, svc, 30, now.Add(-time.Minute))
	downloading := newSupervisionEntry(t, svc, 31, now)

	if err := startAs(t, svc, playing, false); err != nil {
		t.Fatalf("arrancar la reproducción: %v", err)
	}
	waitForGeneratedSegment(t, playing, 0)

	err := startAs(t, svc, downloading, true)
	if !errors.Is(err, ErrDownloadBusy) {
		t.Fatalf("error = %v, se esperaba ErrDownloadBusy", err)
	}
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if playing.run == nil || playing.run.finished() {
		t.Fatal("la descarga no debe detener la reproducción")
	}
	if downloading.run != nil {
		t.Fatal("la descarga sin cupo no debe arrancar proceso")
	}
}

func TestReproduccionExpulsaPrimeroALaDescarga(t *testing.T) {
	svc := newDownloadTestService(t, 2)
	now := time.Now()
	// La reproducción es la más antigua: aun así, la víctima es la descarga.
	playing := newSupervisionEntry(t, svc, 40, now.Add(-time.Hour))
	downloading := newSupervisionEntry(t, svc, 41, now)
	newPlayer := newSupervisionEntry(t, svc, 42, now)

	if err := startAs(t, svc, playing, false); err != nil {
		t.Fatalf("arrancar la reproducción: %v", err)
	}
	if err := startAs(t, svc, downloading, true); err != nil {
		t.Fatalf("arrancar la descarga: %v", err)
	}
	waitForGeneratedSegment(t, playing, 0)
	waitForGeneratedSegment(t, downloading, 0)

	if err := startAs(t, svc, newPlayer, false); err != nil {
		t.Fatalf("arrancar la segunda reproducción: %v", err)
	}
	waitForGeneratedSegment(t, newPlayer, 0)

	svc.mu.Lock()
	defer svc.mu.Unlock()
	if !downloading.run.finished() || !downloading.run.interrupted {
		t.Fatal("debe expulsarse el proceso de descarga")
	}
	if playing.run.finished() {
		t.Fatal("la reproducción más antigua no debe expulsarse mientras haya una descarga")
	}
}

func TestSoloUnaRecodificacionDeDescargaALaVez(t *testing.T) {
	svc := newDownloadTestService(t, 4)
	now := time.Now()
	first := newSupervisionEntry(t, svc, 50, now)
	second := newSupervisionEntry(t, svc, 51, now)

	if err := startAs(t, svc, first, true); err != nil {
		t.Fatalf("primera descarga: %v", err)
	}
	waitForGeneratedSegment(t, first, 0)
	if err := startAs(t, svc, second, true); !errors.Is(err, ErrDownloadBusy) {
		t.Fatalf("segunda descarga: %v, se esperaba ErrDownloadBusy", err)
	}

	// Una reproducción sí tiene cupo aunque haya una descarga viva.
	player := newSupervisionEntry(t, svc, 52, now)
	if err := startAs(t, svc, player, false); err != nil {
		t.Fatalf("reproducción con una descarga viva: %v", err)
	}
}

func TestLasCopiasDeDescargaNoCuentan(t *testing.T) {
	svc := newDownloadTestService(t, 1)
	now := time.Now()
	transcode := newSupervisionEntry(t, svc, 60, now)
	copyEntry := newSupervisionEntry(t, svc, 61, now)
	copyEntry.mode = hlsModeCopy

	if err := startAs(t, svc, transcode, true); err != nil {
		t.Fatalf("descarga recodificada: %v", err)
	}
	if err := startAs(t, svc, copyEntry, true); err != nil {
		t.Fatalf("una copia no debe competir por el cupo de descarga: %v", err)
	}
}

func TestWaitForSegmentDevuelveDescargaOcupada(t *testing.T) {
	svc := newDownloadTestService(t, 1)
	now := time.Now()
	playing := newSupervisionEntry(t, svc, 70, now)
	downloading := newSupervisionEntry(t, svc, 71, now)
	if err := startAs(t, svc, playing, false); err != nil {
		t.Fatalf("arrancar la reproducción: %v", err)
	}
	waitForGeneratedSegment(t, playing, 0)

	err := svc.waitForSegmentAs(context.Background(), downloading, 5, true)
	if !errors.Is(err, ErrDownloadBusy) {
		t.Fatalf("error = %v, se esperaba ErrDownloadBusy", err)
	}
}

func TestReproduccionQuitaLaPrioridadBajaAlProcesoCompartido(t *testing.T) {
	svc := newDownloadTestService(t, 2)
	entry := newSupervisionEntry(t, svc, 80, time.Now())
	if err := startAs(t, svc, entry, true); err != nil {
		t.Fatalf("descarga: %v", err)
	}
	waitForGeneratedSegment(t, entry, 0)

	if err := svc.waitForSegmentAs(context.Background(), entry, 0, false); err != nil {
		t.Fatalf("segmento: %v", err)
	}
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if entry.run.download {
		t.Fatal("el proceso ya lo ve alguien: no debe seguir marcado como descarga")
	}
}

func TestPlaylistDeDescargaPropagaDlALosSegmentos(t *testing.T) {
	playlist := []byte("#EXTM3U\n#EXTINF:6.0,\nsegment_00000.ts\n#EXT-X-ENDLIST\n")
	query := signedQuery("123", "firma", "audio", "0", "dl", "1")
	signed := string(signPlaylist(playlist, query))
	if !strings.Contains(signed, "segment_00000.ts?") || !strings.Contains(signed, "dl=1") {
		t.Fatalf("la playlist firmada no lleva dl=1:\n%s", signed)
	}
}
