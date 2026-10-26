package library

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/catlinux/cinecat/internal/playback"
)

type playTestServer struct {
	*libraryTestServer
	playback *playback.Service
}

func TestLibraryPlaybackHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_LIBRARY_SUBTITLE_HELPER") != "1" {
		return
	}
	args := os.Args
	if len(args) == 0 {
		os.Exit(2)
	}
	if err := os.WriteFile(args[len(args)-1], []byte("WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nHola\n"), 0o644); err != nil {
		os.Exit(2)
	}
}

func newPlayTestServer(t *testing.T, info *playback.FileInfo, inspectErr error) *playTestServer {
	t.Helper()
	base := newLibraryHTTPTestServer(t)
	playSvc, err := playback.New(playback.Options{
		WorkDir: t.TempDir(),
		Command: func(ctx context.Context, _ string, args ...string) *exec.Cmd {
			cmd := exec.CommandContext(ctx, os.Args[0],
				append([]string{"-test.run=TestLibraryPlaybackHelperProcess", "--"}, args...)...)
			cmd.Env = append(os.Environ(), "GO_WANT_LIBRARY_SUBTITLE_HELPER=1")
			return cmd
		},
	})
	if err != nil {
		t.Fatalf("playback.New: %v", err)
	}
	t.Cleanup(func() {
		if err := playSvc.Close(); err != nil {
			t.Errorf("playback.Close: %v", err)
		}
	})
	api := NewAPI(base.db, base.auth, playSvc, base.job, nil).
		WithSettings(base.api.settingsStore).
		WithFileInspector(&fakeFileInspector{info: info, err: inspectErr})
	mux := http.NewServeMux()
	api.Register(mux)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	base.Server = server
	base.api = api
	return &playTestServer{libraryTestServer: base, playback: playSvc}
}

func playURL(ts *playTestServer, id int64, query string) string {
	url := ts.URL + "/api/v1/items/" + strconv.FormatInt(id, 10) + "/play"
	if query != "" {
		url += "?" + query
	}
	return url
}

func TestPlayConOpcionesYPreferencias(t *testing.T) {
	info := fileInfoFixture()
	info.Audio = []playback.AudioStream{
		{Index: 1, Codec: "ac3", Channels: 6, ChannelLayout: "5.1", Language: "spa"},
		{Index: 2, Codec: "aac", Channels: 2, ChannelLayout: "stereo", Language: "eng", Default: true},
	}
	info.Subtitles = []playback.SubtitleStream{
		{Index: 3, Codec: "subrip", Language: "spa", Text: true},
		{Index: 4, Codec: "hdmv_pgs_subtitle", Language: "eng", Text: false},
	}
	ts := newPlayTestServer(t, info, nil)
	id, _ := insertFileInfoItem(t, ts.libraryTestServer, "Pelicula.mkv", true)

	status, body := libraryHTTPGet(t, playURL(ts, id, ""), ts.token)
	if status != http.StatusOK {
		t.Fatalf("play: %d %s", status, body)
	}
	var response playResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("play: %v", err)
	}
	if response.Mode != playback.ModeRemux || len(response.AudioTracks) != 2 ||
		response.SelectedAudio != 1 || response.SelectedQuality != "auto" ||
		len(response.Qualities) == 0 || len(response.Subtitles) != 1 ||
		response.SelectedSubtitle != nil {
		t.Fatalf("respuesta de play inesperada: %+v", response)
	}
	if response.AudioTracks[1].Index != 1 || response.AudioTracks[1].LanguageName != "Inglés" ||
		!response.AudioTracks[1].Default {
		t.Fatalf("pista de audio inesperada: %+v", response.AudioTracks)
	}
	if !strings.Contains(response.URL, "audio=1") || !strings.Contains(response.URL, "q=auto") {
		t.Fatalf("URL sin selección: %s", response.URL)
	}
	if response.Subtitles[0].ID != "e0" || !strings.Contains(response.Subtitles[0].URL, "sig=") {
		t.Fatalf("subtítulo inesperado: %+v", response.Subtitles[0])
	}

	status, body = libraryHTTPGet(t, playURL(ts, id, "audio=0&quality=720"), ts.token)
	if err := json.Unmarshal(body, &response); err != nil || status != http.StatusOK {
		t.Fatalf("play con parámetros: %d %s (%v)", status, body, err)
	}
	if response.SelectedAudio != 0 || response.SelectedQuality != "720" ||
		!strings.Contains(response.URL, "audio=0") || !strings.Contains(response.URL, "q=720") {
		t.Fatalf("parámetros de play no aplicados: %+v", response)
	}

	if status, _ := libraryHTTPGet(t, playURL(ts, id, "audio=5"), ts.token); status != http.StatusBadRequest {
		t.Fatalf("audio fuera de rango: %d", status)
	}
	if status, _ := libraryHTTPGet(t, playURL(ts, id, "quality=4k"), ts.token); status != http.StatusBadRequest {
		t.Fatalf("calidad no válida: %d", status)
	}
	if status, _ := libraryHTTPGet(t, playURL(ts, id, "subtitle=x9"), ts.token); status != http.StatusBadRequest {
		t.Fatalf("subtítulo no válido: %d", status)
	}

	status, body = libraryHTTPDo(t, http.MethodPut, itemPreferencesURL(ts.libraryTestServer, id), ts.token,
		map[string]any{"audio_index": 0, "quality": "720", "subtitle_id": "e0"})
	if status != http.StatusNoContent {
		t.Fatalf("preferencias: %d %s", status, body)
	}
	status, body = libraryHTTPGet(t, playURL(ts, id, ""), ts.token)
	if err := json.Unmarshal(body, &response); err != nil || status != http.StatusOK {
		t.Fatalf("play con preferencias: %d %s (%v)", status, body, err)
	}
	if response.SelectedAudio != 0 || response.SelectedQuality != "720" ||
		response.SelectedSubtitle == nil || *response.SelectedSubtitle != "e0" {
		t.Fatalf("preferencias no aplicadas: %+v", response)
	}
	if !strings.Contains(response.URL, "audio=0") || !strings.Contains(response.URL, "q=720") {
		t.Fatalf("URL con preferencias inesperada: %s", response.URL)
	}

	status, body = libraryHTTPDo(t, http.MethodPut, itemPreferencesURL(ts.libraryTestServer, id), ts.token,
		map[string]any{"audio_index": 5, "subtitle_id": "x9"})
	if status != http.StatusNoContent {
		t.Fatalf("preferencias obsoletas: %d %s", status, body)
	}
	status, body = libraryHTTPGet(t, playURL(ts, id, ""), ts.token)
	if err := json.Unmarshal(body, &response); err != nil || status != http.StatusOK {
		t.Fatalf("play con preferencias obsoletas: %d %s (%v)", status, body, err)
	}
	if response.SelectedAudio != 1 || response.SelectedSubtitle != nil {
		t.Fatalf("las preferencias obsoletas no se ignoraron: %+v", response)
	}
}

func TestPlayDirectoEInspectorSinDatos(t *testing.T) {
	info := fileInfoFixture()
	info.Video[0].Height = 720
	ts := newPlayTestServer(t, info, nil)
	id, _ := insertFileInfoItem(t, ts.libraryTestServer, "Pelicula.mp4", true)

	status, body := libraryHTTPGet(t, playURL(ts, id, ""), ts.token)
	var response playResponse
	if err := json.Unmarshal(body, &response); err != nil || status != http.StatusOK {
		t.Fatalf("play directo: %d %s (%v)", status, body, err)
	}
	if response.Mode != playback.ModeDirect || len(response.Qualities) != 1 ||
		response.Qualities[0].Value != "original" || response.SelectedQuality != "original" {
		t.Fatalf("calidades directas inesperadas: %+v", response)
	}
	if status, _ := libraryHTTPGet(t, ts.URL+response.URL+"&audio=3", ""); status != http.StatusBadRequest {
		t.Fatalf("audio inválido en stream: %d", status)
	}
	if status, _ := libraryHTTPGet(t, ts.URL+response.URL+"&q=4k", ""); status != http.StatusBadRequest {
		t.Fatalf("calidad inválida en stream: %d", status)
	}

	ts = newPlayTestServer(t, nil, playback.ErrFFprobeNotFound)
	id, _ = insertFileInfoItem(t, ts.libraryTestServer, "Pelicula.mkv", true)
	status, body = libraryHTTPGet(t, playURL(ts, id, ""), ts.token)
	if err := json.Unmarshal(body, &response); err != nil || status != http.StatusOK {
		t.Fatalf("play sin ffprobe: %d %s (%v)", status, body, err)
	}
	if len(response.AudioTracks) != 0 || len(response.Subtitles) != 0 {
		t.Fatalf("listas sin ffprobe inesperadas: %+v", response)
	}
}

func TestPlayCalidadesNoSubenLaResolucion(t *testing.T) {
	info := fileInfoFixture()
	info.Video[0].Height = 720
	ts := newPlayTestServer(t, info, nil)
	id, _ := insertFileInfoItem(t, ts.libraryTestServer, "Pelicula.mkv", true)
	status, body := libraryHTTPGet(t, playURL(ts, id, "quality=1080"), ts.token)
	var response playResponse
	if err := json.Unmarshal(body, &response); err != nil || status != http.StatusOK {
		t.Fatalf("play: %d %s (%v)", status, body, err)
	}
	for _, quality := range response.Qualities {
		if quality.Value == "1080" {
			t.Fatalf("se ofreció 1080p para un original de 720p: %+v", response.Qualities)
		}
	}
	if response.SelectedQuality != "auto" {
		t.Fatalf("calidad seleccionada = %q, se esperaba auto", response.SelectedQuality)
	}
}

func TestPlayRespetaElTopeDeAltura(t *testing.T) {
	info := fileInfoFixture()
	info.Video[0].Height = 1080
	ts := newPlayTestServer(t, info, nil)
	if failures := ts.api.settingsStore.Set(context.Background(), map[string]json.RawMessage{
		"playback.max_height": json.RawMessage("480"),
	}); len(failures) != 0 {
		t.Fatalf("ajustar max_height: %v", failures)
	}
	id, _ := insertFileInfoItem(t, ts.libraryTestServer, "Pelicula.mkv", true)
	status, body := libraryHTTPGet(t, playURL(ts, id, ""), ts.token)
	var response playResponse
	if err := json.Unmarshal(body, &response); err != nil || status != http.StatusOK {
		t.Fatalf("play: %d %s (%v)", status, body, err)
	}
	for _, quality := range response.Qualities {
		if quality.Value == "720" || quality.Value == "1080" {
			t.Fatalf("calidad por encima del tope: %+v", response.Qualities)
		}
	}
	if len(response.Qualities) == 0 || response.Qualities[0].Value != "auto" ||
		!strings.Contains(response.Qualities[0].Label, "480") {
		t.Fatalf("calidades con tope inesperadas: %+v", response.Qualities)
	}
}

func TestPlaySubtitulosYTitulosFirmados(t *testing.T) {
	info := fileInfoFixture()
	ts := newPlayTestServer(t, info, nil)
	id, _ := insertFileInfoItem(t, ts.libraryTestServer, "Pelicula.mkv", true)
	status, body := libraryHTTPGet(t, playURL(ts, id, ""), ts.token)
	if status != http.StatusOK {
		t.Fatalf("play: %d %s", status, body)
	}
	var response playResponse
	if err := json.Unmarshal(body, &response); err != nil || len(response.Subtitles) == 0 {
		t.Fatalf("play sin subtítulos: %v %+v", err, response)
	}
	subtitleURL := response.Subtitles[0].URL
	status, data := libraryHTTPGet(t, ts.URL+subtitleURL, "")
	if status != http.StatusOK || !strings.Contains(string(data), "WEBVTT") {
		t.Fatalf("subtítulo: %d %s", status, data)
	}
	if status, _ := libraryHTTPGet(t, ts.URL+subtitleURL+"x", ""); status != http.StatusForbidden {
		t.Fatalf("firma inválida: %d", status)
	}
	bad := strings.Replace(subtitleURL, "e0.vtt", "e9.vtt", 1)
	if status, _ := libraryHTTPGet(t, ts.URL+bad, ""); status != http.StatusNotFound {
		t.Fatalf("sid inexistente: %d", status)
	}
}

func TestPlaySubtituloExterno(t *testing.T) {
	dir := t.TempDir()
	video := filepath.Join(dir, "Pelicula.mkv")
	if err := os.WriteFile(video, []byte("video"), 0o644); err != nil {
		t.Fatalf("crear vídeo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Pelicula.es.srt"), []byte("texto"), 0o644); err != nil {
		t.Fatalf("crear subtítulo: %v", err)
	}
	ts := newPlayTestServer(t, nil, playback.ErrFFprobeNotFound)
	libraryID := createLibrary(t, ts.db, dir)
	id := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: video, title: "Pelicula",
	})
	status, body := libraryHTTPGet(t, playURL(ts, id, ""), ts.token)
	var response playResponse
	if err := json.Unmarshal(body, &response); err != nil || status != http.StatusOK {
		t.Fatalf("play: %d %s (%v)", status, body, err)
	}
	if len(response.Subtitles) != 1 || response.Subtitles[0].Source != "external" ||
		response.Subtitles[0].Language != "es" {
		t.Fatalf("subtítulo externo inesperado: %+v", response.Subtitles)
	}
	status, data := libraryHTTPGet(t, ts.URL+response.Subtitles[0].URL, "")
	if status != http.StatusOK || !strings.Contains(string(data), "WEBVTT") {
		t.Fatalf("subtítulo externo: %d %s", status, data)
	}
}
