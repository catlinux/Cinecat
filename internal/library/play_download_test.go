package library

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/catlinux/cinecat/internal/playback"
)

func TestPlayDownloadForzaHLSConVidaLargaYDl(t *testing.T) {
	ts := newPlayTestServer(t, fileInfoFixture(), nil)
	id, _ := insertFileInfoItem(t, ts.libraryTestServer, "Pelicula.mp4", true)

	normal := playJSON(t, ts, id, "")
	if normal.Mode != playback.ModeDirect {
		t.Fatalf("sin download debía seguir en directo: %+v", normal)
	}

	response := playJSON(t, ts, id, "download=1")
	if response.Mode != playback.ModeRemux {
		t.Fatalf("download=1 debe forzar HLS: %+v", response)
	}
	parsed, err := url.Parse(response.URL)
	if err != nil {
		t.Fatalf("URL no válida %q: %v", response.URL, err)
	}
	if !strings.HasSuffix(parsed.Path, "/playlist.m3u8") {
		t.Fatalf("la URL de descarga debe ser la playlist: %s", response.URL)
	}
	query := parsed.Query()
	if query.Get("dl") != "1" || query.Get("exp") == "" || query.Get("sig") == "" ||
		query.Get("audio") == "" || query.Get("q") == "" {
		t.Fatalf("query de descarga incompleta: %s", response.URL)
	}

	if left := time.Until(response.ExpiresAt); left < 71*time.Hour || left > 73*time.Hour {
		t.Fatalf("la descarga caduca en %v, se esperaban unas 72 h", left)
	}
	if left := time.Until(normal.ExpiresAt); left > 9*time.Hour {
		t.Fatalf("la reproducción normal debe conservar su vida corta, caduca en %v", left)
	}
	if strings.Contains(normal.URL, "dl=1") {
		t.Fatalf("la reproducción normal no lleva dl: %s", normal.URL)
	}
}

func TestPlayDownloadUsaLaMismaCalidadYAudioQueLaReproduccion(t *testing.T) {
	ts := newPlayTestServer(t, fileInfoFixture(), nil)
	id, _ := insertFileInfoItem(t, ts.libraryTestServer, "Pelicula.mp4", true)

	response := playJSON(t, ts, id, "download=1")
	parsed, err := url.Parse(response.URL)
	if err != nil {
		t.Fatalf("URL no válida: %v", err)
	}
	if parsed.Query().Get("q") != response.SelectedQuality ||
		parsed.Query().Get("audio") != strconv.Itoa(response.SelectedAudio) {
		t.Fatalf("la URL no coincide con la selección %q/%d: %s",
			response.SelectedQuality, response.SelectedAudio, response.URL)
	}
}

func TestDescargaOcupadaResponde503ConRetryAfter(t *testing.T) {
	ts := newPlayTestServer(t, fileInfoFixture(), nil)
	recorder := httptest.NewRecorder()
	ts.api.writePlaybackError(recorder, playback.ErrDownloadBusy)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("estado = %d, se esperaba 503", recorder.Code)
	}
	if got := recorder.Header().Get("Retry-After"); got != "30" {
		t.Fatalf("Retry-After = %q, se esperaba 30", got)
	}
}
