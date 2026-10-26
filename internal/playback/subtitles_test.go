package playback

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSubtitleHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_SUBTITLE_HELPER") != "1" {
		return
	}
	args := os.Args
	if len(args) == 0 {
		os.Exit(2)
	}
	output := args[len(args)-1]
	if err := os.WriteFile(output, []byte("WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nHola\n"), 0o644); err != nil {
		os.Exit(2)
	}
}

func fakeSubtitleCommand(calls *int32) func(context.Context, string, ...string) *exec.Cmd {
	return func(ctx context.Context, _ string, args ...string) *exec.Cmd {
		atomic.AddInt32(calls, 1)
		helperArgs := append([]string{"-test.run=TestSubtitleHelperProcess", "--"}, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], helperArgs...)
		cmd.Env = append(os.Environ(), "GO_WANT_SUBTITLE_HELPER=1")
		return cmd
	}
}

func subtitleTestService(t *testing.T, calls *int32) *Service {
	t.Helper()
	svc, err := New(Options{
		WorkDir: t.TempDir(),
		Command: fakeSubtitleCommand(calls),
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

func TestDiscoverSubtitlesIncrustadosYExternos(t *testing.T) {
	dir := t.TempDir()
	video := filepath.Join(dir, "Pelicula.mkv")
	if err := os.WriteFile(video, []byte("video"), 0o644); err != nil {
		t.Fatalf("crear vídeo: %v", err)
	}
	for _, name := range []string{"Pelicula.en.forced.vtt", "Pelicula.es.srt", "Pelicula.spa.ass", "Otra.es.srt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("texto"), 0o644); err != nil {
			t.Fatalf("crear subtítulo: %v", err)
		}
	}
	info := &FileInfo{Subtitles: []SubtitleStream{
		{Codec: "subrip", Language: "es", Text: true},
		{Codec: "hdmv_pgs_subtitle", Language: "en", Text: false},
		{Codec: "ass", Language: "fr", Text: true},
	}}
	tracks := DiscoverSubtitles(video, info)
	if len(tracks) != 5 {
		t.Fatalf("pistas = %d, se esperaban 5: %+v", len(tracks), tracks)
	}
	if tracks[0].ID != "e0" || tracks[1].ID != "e2" || tracks[2].ID != "x0" ||
		tracks[3].ID != "x1" || tracks[4].ID != "x2" {
		t.Fatalf("ids inesperados: %+v", tracks)
	}
	if tracks[2].Source != "external" || !tracks[2].Forced || tracks[2].Language != "en" {
		t.Fatalf("subtítulo externo inesperado: %+v", tracks[2])
	}
	if tracks[3].Language != "es" || tracks[4].Language != "spa" {
		t.Fatalf("idiomas externos inesperados: %+v", tracks[3:])
	}
}

func TestSubtitleConversionYCache(t *testing.T) {
	var calls int32
	svc := subtitleTestService(t, &calls)
	dir := t.TempDir()
	video := filepath.Join(dir, "Pelicula.mkv")
	if err := os.WriteFile(video, []byte("video"), 0o644); err != nil {
		t.Fatalf("crear vídeo: %v", err)
	}
	info := &FileInfo{Subtitles: []SubtitleStream{{Codec: "subrip", Language: "es", Text: true}}}
	track := DiscoverSubtitles(video, info)[0]

	data, err := svc.SubtitleWebVTT(context.Background(), 1, video, track)
	if err != nil {
		t.Fatalf("convertir incrustado: %v", err)
	}
	if !strings.Contains(string(data), "WEBVTT") {
		t.Fatalf("salida inesperada: %s", data)
	}
	if calls != 1 {
		t.Fatalf("llamadas = %d, se esperaba 1", calls)
	}
	if _, err := svc.SubtitleWebVTT(context.Background(), 1, video, track); err != nil {
		t.Fatalf("segunda conversión: %v", err)
	}
	if calls != 1 {
		t.Fatalf("la caché no evitó la segunda conversión: %d", calls)
	}
}

func TestSubtitleExternoLatin1YVTT(t *testing.T) {
	var calls int32
	svc := subtitleTestService(t, &calls)
	dir := t.TempDir()
	video := filepath.Join(dir, "Pelicula.mkv")
	if err := os.WriteFile(video, []byte("video"), 0o644); err != nil {
		t.Fatalf("crear vídeo: %v", err)
	}
	srt := filepath.Join(dir, "Pelicula.es.srt")
	if err := os.WriteFile(srt, []byte("1\n00:00:01,000 --> 00:00:02,000\nCaf\xe9 y ni\xf1o\n"), 0o644); err != nil {
		t.Fatalf("crear srt: %v", err)
	}
	vtt := filepath.Join(dir, "Pelicula.en.vtt")
	if err := os.WriteFile(vtt, []byte("WEBVTT\n\n"), 0o644); err != nil {
		t.Fatalf("crear vtt: %v", err)
	}
	tracks := DiscoverSubtitles(video, nil)
	if len(tracks) != 2 {
		t.Fatalf("pistas externas = %d: %+v", len(tracks), tracks)
	}
	var srtTrack, vttTrack SubtitleTrack
	for _, track := range tracks {
		if strings.HasSuffix(track.Path, ".srt") {
			srtTrack = track
		} else {
			vttTrack = track
		}
	}
	if _, err := svc.SubtitleWebVTT(context.Background(), 2, video, srtTrack); err != nil {
		t.Fatalf("convertir externo: %v", err)
	}
	if calls != 1 {
		t.Fatalf("llamadas FFmpeg = %d", calls)
	}
	if _, err := svc.SubtitleWebVTT(context.Background(), 2, video, vttTrack); err != nil {
		t.Fatalf("servir vtt: %v", err)
	}
	if calls != 1 {
		t.Fatalf("el vtt no debe pasar por FFmpeg: %d", calls)
	}
	decoded := string(decodeSubtitleBytes([]byte("Caf\xe9")))
	if decoded != "Café" {
		t.Fatalf("Latin-1 mal convertido: %q", decoded)
	}
}

func TestSubtitleCacheInvalidaPorFecha(t *testing.T) {
	var calls int32
	svc := subtitleTestService(t, &calls)
	dir := t.TempDir()
	video := filepath.Join(dir, "Pelicula.mkv")
	if err := os.WriteFile(video, []byte("video"), 0o644); err != nil {
		t.Fatalf("crear vídeo: %v", err)
	}
	info := &FileInfo{Subtitles: []SubtitleStream{{Codec: "subrip", Text: true}}}
	track := DiscoverSubtitles(video, info)[0]
	ctx := context.Background()
	if _, err := svc.SubtitleWebVTT(ctx, 3, video, track); err != nil {
		t.Fatalf("primera conversión: %v", err)
	}
	modified := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(video, modified, modified); err != nil {
		t.Fatalf("cambiar fecha: %v", err)
	}
	if _, err := svc.SubtitleWebVTT(ctx, 3, video, track); err != nil {
		t.Fatalf("segunda conversión: %v", err)
	}
	if calls != 2 {
		t.Fatalf("la fecha no invalidó la caché: %d llamadas", calls)
	}
}
