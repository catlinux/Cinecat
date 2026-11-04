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

	"github.com/catlinux/cinecat/internal/auth"
	"github.com/catlinux/cinecat/internal/playback"
)

type playTestServer struct {
	*libraryTestServer
	playback    *playback.Service
	subtitleDir string
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
	subtitleDir := t.TempDir()
	api := NewAPI(base.db, base.auth, playSvc, base.job, nil).
		WithSettings(base.api.settingsStore).
		WithSubtitles(subtitleDir).
		WithFileInspector(&fakeFileInspector{info: info, err: inspectErr})
	mux := http.NewServeMux()
	api.Register(mux)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	base.Server = server
	base.api = api
	return &playTestServer{libraryTestServer: base, playback: playSvc, subtitleDir: subtitleDir}
}

func playURL(ts *playTestServer, id int64, query string) string {
	url := ts.URL + "/api/v1/items/" + strconv.FormatInt(id, 10) + "/play"
	if query != "" {
		url += "?" + query
	}
	return url
}

func setDeviceProfile(t *testing.T, ts *playTestServer, profile auth.SessionProfile) {
	t.Helper()
	user, err := ts.auth.Authenticate(context.Background(), ts.token)
	if err != nil {
		t.Fatalf("autenticar sesión de prueba: %v", err)
	}
	if err := ts.auth.PutSessionProfile(
		context.Background(), user.ID, user.SessionID, profile,
	); err != nil {
		t.Fatalf("guardar perfil de sesión: %v", err)
	}
}

func playJSON(t *testing.T, ts *playTestServer, id int64, query string) playResponse {
	t.Helper()
	status, body := libraryHTTPGet(t, playURL(ts, id, query), ts.token)
	var response playResponse
	if err := json.Unmarshal(body, &response); err != nil || status != http.StatusOK {
		t.Fatalf("play %q: %d %s (%v)", query, status, body, err)
	}
	return response
}

func TestPlayInformaSiElOriginalEsDescargable(t *testing.T) {
	tests := []struct {
		name    string
		caps    string
		audio   playback.AudioStream
		want    bool
		blocked string
	}{
		{
			name:  "mkv h264 aac",
			caps:  "mkv,h264,aac",
			audio: playback.AudioStream{Codec: "aac", Channels: 2, Default: true},
			want:  true,
		},
		{
			name:  "mkv h264 ac3 declarado",
			caps:  "mkv,h264,ac3",
			audio: playback.AudioStream{Codec: "ac3", Channels: 6, Default: true},
			want:  true,
		},
		{
			name:    "mkv h264 dts",
			caps:    "mkv,h264,dts",
			audio:   playback.AudioStream{Codec: "dts", Channels: 6, Default: true},
			blocked: "audio",
		},
		{
			name:    "web no nativa",
			caps:    "h264,aac",
			audio:   playback.AudioStream{Codec: "aac", Channels: 2, Default: true},
			blocked: "client",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info := fileInfoFixture()
			info.FileName = "Episodio.mkv"
			info.Container = "matroska"
			info.Audio = []playback.AudioStream{test.audio}
			ts := newPlayTestServer(t, info, nil)
			id, _ := insertFileInfoItem(t, ts.libraryTestServer, "Episodio.mkv", true)

			response := playJSON(t, ts, id, "caps="+test.caps)
			if response.DownloadDirect != test.want || response.DownloadBlocked != test.blocked {
				t.Fatalf("download_direct/download_blocked = %v/%q, se esperaba %v/%q",
					response.DownloadDirect, response.DownloadBlocked, test.want, test.blocked)
			}
		})
	}
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
		len(response.Qualities) == 0 || len(response.Subtitles) != 2 ||
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
	if response.Subtitles[0].ID != "e0" || response.Subtitles[0].BurnIn ||
		!strings.Contains(response.Subtitles[0].URL, "sig=") {
		t.Fatalf("subtítulo inesperado: %+v", response.Subtitles[0])
	}
	if response.Subtitles[1].ID != "e1" || !response.Subtitles[1].BurnIn ||
		response.Subtitles[1].URL != "" {
		t.Fatalf("subtítulo de imagen inesperado: %+v", response.Subtitles[1])
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

func TestPlayAplicaIdiomasDelPerfilSinPisarPrecedencias(t *testing.T) {
	info := fileInfoFixture()
	info.Audio = []playback.AudioStream{
		{Index: 1, Codec: "aac", Channels: 2, Language: "eng", Default: true},
		{Index: 2, Codec: "ac3", Channels: 6, Language: "spa"},
	}
	info.Subtitles = []playback.SubtitleStream{
		{Index: 3, Codec: "subrip", Language: "spa", Text: true},
		{Index: 4, Codec: "subrip", Language: "eng", Text: true, Forced: true},
		{Index: 5, Codec: "subrip", Language: "eng", Text: true},
		{Index: 6, Codec: "hdmv_pgs_subtitle", Language: "eng", Text: false},
	}
	ts := newPlayTestServer(t, info, nil)
	id, _ := insertFileInfoItem(t, ts.libraryTestServer, "Pelicula.mp4", true)

	setDeviceProfile(t, ts, auth.SessionProfile{
		AudioLanguage:    "es",
		SubtitleLanguage: "eng",
	})
	response := playJSON(t, ts, id, "")
	if response.SelectedAudio != 1 || response.SelectedSubtitle == nil ||
		*response.SelectedSubtitle != "e2" {
		t.Fatalf("perfil no aplicado: %+v", response)
	}

	response = playJSON(t, ts, id, "audio=0&subtitle=e0")
	if response.SelectedAudio != 0 || response.SelectedSubtitle == nil ||
		*response.SelectedSubtitle != "e0" {
		t.Fatalf("el parámetro explícito debe ganar: %+v", response)
	}
	response = playJSON(t, ts, id, "subtitle=")
	if response.SelectedAudio != 1 || response.SelectedSubtitle != nil {
		t.Fatalf("el subtítulo vacío debe desactivar el perfil: %+v", response)
	}

	status, body := libraryHTTPDo(t, http.MethodPut, itemPreferencesURL(ts.libraryTestServer, id),
		ts.token, map[string]any{"audio_index": 0, "subtitle_id": "e0"})
	if status != http.StatusNoContent {
		t.Fatalf("preferencias: %d %s", status, body)
	}
	response = playJSON(t, ts, id, "")
	if response.SelectedAudio != 0 || response.SelectedSubtitle == nil ||
		*response.SelectedSubtitle != "e0" {
		t.Fatalf("la preferencia del ítem debe ganar al perfil: %+v", response)
	}

	status, body = libraryHTTPDo(t, http.MethodPut, itemPreferencesURL(ts.libraryTestServer, id),
		ts.token, map[string]any{"audio_index": 0, "subtitle_id": ""})
	if status != http.StatusNoContent {
		t.Fatalf("preferencia de subtítulo vacía: %d %s", status, body)
	}
	response = playJSON(t, ts, id, "")
	if response.SelectedAudio != 0 || response.SelectedSubtitle != nil {
		t.Fatalf("la preferencia vacía debe desactivar el perfil: %+v", response)
	}

	status, body = libraryHTTPDo(t, http.MethodDelete, itemPreferencesURL(ts.libraryTestServer, id),
		ts.token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("borrar preferencias: %d %s", status, body)
	}
	setDeviceProfile(t, ts, auth.SessionProfile{
		AudioLanguage:    "eng",
		SubtitleLanguage: "off",
	})
	response = playJSON(t, ts, id, "")
	if response.SelectedAudio != 0 || response.SelectedSubtitle != nil {
		t.Fatalf("perfil off/audio inglés inesperado: %+v", response)
	}
}

func TestPlayNoEligeUnaPistaDeImagenPorPerfil(t *testing.T) {
	info := fileInfoFixture()
	info.Subtitles = []playback.SubtitleStream{
		{Index: 2, Codec: "hdmv_pgs_subtitle", Language: "eng", Text: false},
	}
	ts := newPlayTestServer(t, info, nil)
	id, _ := insertFileInfoItem(t, ts.libraryTestServer, "Pelicula.mp4", true)
	setDeviceProfile(t, ts, auth.SessionProfile{SubtitleLanguage: "eng"})

	response := playJSON(t, ts, id, "")
	if response.SelectedSubtitle != nil || response.Mode != playback.ModeDirect {
		t.Fatalf("no debe elegir una imagen por defecto: %+v", response)
	}
}

func TestPlayPerfilLimitaLaCalidadSoloEnHLS(t *testing.T) {
	info := fileInfoFixture()
	info.Container = "matroska,webm"
	info.FileName = "Pelicula.mkv"
	info.Video[0].Height = 1080
	ts := newPlayTestServer(t, info, nil)
	id, _ := insertFileInfoItem(t, ts.libraryTestServer, "Pelicula.mkv", true)
	setDeviceProfile(t, ts, auth.SessionProfile{MaxQuality: "480"})

	response := playJSON(t, ts, id, "")
	if response.Mode != playback.ModeRemux || response.SelectedQuality != "480" ||
		!strings.Contains(response.URL, "q=480") {
		t.Fatalf("tope de calidad no aplicado: %+v", response)
	}
	for _, quality := range response.Qualities {
		if quality.Value == "720" || quality.Value == "1080" || quality.Value == "original" {
			t.Fatalf("calidad por encima del tope: %+v", response.Qualities)
		}
	}

	for _, query := range []string{"quality=720", "quality=1080", "quality=original"} {
		response = playJSON(t, ts, id, query)
		if response.SelectedQuality != "480" || !strings.Contains(response.URL, "q=480") {
			t.Fatalf("%s debía bajar al tope: %+v", query, response)
		}
	}

	status, body := libraryHTTPDo(t, http.MethodPut, itemPreferencesURL(ts.libraryTestServer, id),
		ts.token, map[string]any{"quality": "720"})
	if status != http.StatusNoContent {
		t.Fatalf("preferencia de calidad: %d %s", status, body)
	}
	response = playJSON(t, ts, id, "")
	if response.SelectedQuality != "480" {
		t.Fatalf("la preferencia no debe superar el perfil: %+v", response)
	}

	directInfo := fileInfoFixture()
	direct := newPlayTestServer(t, directInfo, nil)
	directID, _ := insertFileInfoItem(t, direct.libraryTestServer, "Pelicula.mp4", true)
	setDeviceProfile(t, direct, auth.SessionProfile{MaxQuality: "480"})
	directResponse := playJSON(t, direct, directID, "")
	if directResponse.Mode != playback.ModeDirect ||
		directResponse.SelectedQuality != "original" ||
		len(directResponse.Qualities) != 1 ||
		directResponse.Qualities[0].Value != "original" {
		t.Fatalf("el tope no debe afectar al directo: %+v", directResponse)
	}
}

func TestPlayAppNativa(t *testing.T) {
	info := fileInfoFixture()
	info.Container = "matroska,webm"
	info.FileName = "Pelicula.mkv"
	info.Video[0].PixelFormat = "yuv420p"
	info.Video[0].BitDepth = 8
	info.Audio = []playback.AudioStream{
		{Index: 1, Codec: "ac3", Channels: 6, Language: "spa", Default: true},
		{Index: 2, Codec: "dts", Channels: 6, Language: "eng"},
	}
	ts := newPlayTestServer(t, info, nil)
	id, _ := insertFileInfoItem(t, ts.libraryTestServer, "Pelicula.mkv", true)
	const tv = "caps=mkv,mp4,h264,aac,ac3"

	play := func(query string) playResponse {
		t.Helper()
		status, body := libraryHTTPGet(t, playURL(ts, id, query), ts.token)
		var response playResponse
		if err := json.Unmarshal(body, &response); err != nil || status != http.StatusOK {
			t.Fatalf("play %q: %d %s (%v)", query, status, body, err)
		}
		return response
	}

	direct := play(tv)
	if direct.Mode != playback.ModeDirect || direct.Reason != "" ||
		!strings.Contains(direct.URL, "audio=0") || !strings.Contains(direct.URL, "mkv") ||
		strings.Contains(direct.URL, "playlist.m3u8") {
		t.Fatalf("MKV con AC-3 debería ir en directo para la app: %+v", direct)
	}
	// /stream vuelve a decidir con los parámetros de la URL: tiene que servir
	// el archivo, no responder como si fuera HLS.
	if status, body := libraryHTTPGet(t, ts.URL+direct.URL, ""); status != http.StatusOK {
		t.Fatalf("stream directo de la app: %d %s", status, body)
	}

	if dts := play(tv + "&audio=1"); dts.Mode != playback.ModeRemux || dts.Reason != playback.ReasonAudio ||
		!strings.Contains(dts.URL, "audio=1") {
		t.Fatalf("la pista DTS no declarada debe ir por HLS: %+v", dts)
	}

	status, body := libraryHTTPDo(t, http.MethodPut, itemPreferencesURL(ts.libraryTestServer, id), ts.token,
		map[string]any{"audio_index": 1})
	if status != http.StatusNoContent {
		t.Fatalf("preferencias: %d %s", status, body)
	}
	if preferred := play(tv); preferred.Mode != playback.ModeRemux || preferred.Reason != playback.ReasonAudio {
		t.Fatalf("la pista preferida (DTS) debe decidir el modo: %+v", preferred)
	}

	if web := play(""); web.Mode != playback.ModeRemux || web.Reason != playback.ReasonContainer {
		t.Fatalf("la web sigue recibiendo HLS para MKV: %+v", web)
	}
}

func TestPlaySubtituloImagenFuerzaHLS(t *testing.T) {
	info := fileInfoFixture()
	info.Subtitles = []playback.SubtitleStream{
		{Index: 2, Codec: "subrip", Language: "spa", Text: true},
		{Index: 3, Codec: "hdmv_pgs_subtitle", Language: "eng", Text: false},
	}
	ts := newPlayTestServer(t, info, nil)
	id, _ := insertFileInfoItem(t, ts.libraryTestServer, "Pelicula.mp4", true)

	plain, body := libraryHTTPGet(t, playURL(ts, id, ""), ts.token)
	var response playResponse
	if err := json.Unmarshal(body, &response); err != nil || plain != http.StatusOK {
		t.Fatalf("play directo: %d %s (%v)", plain, body, err)
	}
	if response.Mode != playback.ModeDirect {
		t.Fatalf("sin subtítulo debía seguir en directo: %+v", response)
	}

	status, body := libraryHTTPGet(t, playURL(ts, id, "subtitle=e1"), ts.token)
	if err := json.Unmarshal(body, &response); err != nil || status != http.StatusOK {
		t.Fatalf("play con imagen: %d %s (%v)", status, body, err)
	}
	if response.Mode != playback.ModeRemux || response.Reason != playback.ReasonSubtitle ||
		response.SelectedSubtitle == nil || *response.SelectedSubtitle != "e1" {
		t.Fatalf("el subtítulo de imagen debía forzar HLS: %+v", response)
	}
	if !strings.Contains(response.URL, "burnsub=1") {
		t.Fatalf("falta burnsub en la URL: %s", response.URL)
	}
	if len(response.Subtitles) != 2 || !response.Subtitles[1].BurnIn ||
		response.Subtitles[1].URL != "" || response.Subtitles[0].BurnIn {
		t.Fatalf("subtítulos inesperados: %+v", response.Subtitles)
	}

	status, body = libraryHTTPDo(t, http.MethodPut, itemPreferencesURL(ts.libraryTestServer, id), ts.token,
		map[string]any{"subtitle_id": "e1"})
	if status != http.StatusNoContent {
		t.Fatalf("preferencia de imagen: %d %s", status, body)
	}
	status, body = libraryHTTPGet(t, playURL(ts, id, ""), ts.token)
	if err := json.Unmarshal(body, &response); err != nil || status != http.StatusOK {
		t.Fatalf("play con preferencia de imagen: %d %s (%v)", status, body, err)
	}
	if response.Mode != playback.ModeRemux || response.Reason != playback.ReasonSubtitle {
		t.Fatalf("la preferencia de imagen debía forzar HLS: %+v", response)
	}

	selection, selectionErr := ts.api.streamSelection(playback.ModeDirect, info, map[string][]string{
		"burnsub": {"1"},
	})
	if selectionErr != nil || selection.BurnSubtitle != 2 ||
		!selection.ForceTranscode || selection.Copy || selection.ForceCopy {
		t.Fatalf("selección de stream con burnsub inesperada: %+v (%v)", selection, selectionErr)
	}
	for _, raw := range []string{"0", "2", "x"} {
		if _, err := ts.api.streamSelection(playback.ModeDirect, info,
			map[string][]string{"burnsub": {raw}}); err == nil || err.status != http.StatusBadRequest {
			t.Fatalf("burnsub %q debería dar 400: %+v", raw, err)
		}
	}

	signature, expiresAt := ts.playback.Sign(id)
	imageURL := ts.URL + "/api/v1/stream/" + strconv.FormatInt(id, 10) +
		"/subtitles/e1.vtt?exp=" + strconv.FormatInt(expiresAt.Unix(), 10) + "&sig=" + signature
	if status, body := libraryHTTPGet(t, imageURL, ""); status != http.StatusNotFound {
		t.Fatalf("el subtítulo de imagen no debe convertirse: %d %s", status, body)
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
