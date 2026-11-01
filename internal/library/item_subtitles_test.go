package library

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/catlinux/cinecat/internal/playback"
)

func uploadSubtitleFile(
	t *testing.T,
	ts *playTestServer,
	token string,
	itemID int64,
	filename string,
	content []byte,
	fields map[string]string,
) (int, []byte) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("crear parte multipart: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("escribir archivo: %v", err)
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("escribir campo %s: %v", key, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("cerrar multipart: %v", err)
	}
	url := ts.URL + "/api/v1/items/" + strconv.FormatInt(itemID, 10) + "/subtitles"
	req, err := http.NewRequest(http.MethodPost, url, &body)
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subir subtítulo: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leer respuesta: %v", err)
	}
	return resp.StatusCode, data
}

func subtitleDeleteURL(ts *playTestServer, itemID int64, subtitle string) string {
	return ts.URL + "/api/v1/items/" + strconv.FormatInt(itemID, 10) + "/subtitles/" + subtitle
}

func TestSubtitlesDirPorDefectoJuntoAlaBase(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	api := NewAPI(ts.db, ts.auth, nil, ts.job, nil)
	if api.subtitlesDir == "" || filepath.Base(api.subtitlesDir) != "subtitles" {
		t.Fatalf("directorio de subtítulos inesperado: %q", api.subtitlesDir)
	}
}

func TestSubtitleUploadPlayServeDelete(t *testing.T) {
	dir := t.TempDir()
	video := filepath.Join(dir, "Pelicula.mkv")
	if err := os.WriteFile(video, []byte("video"), 0o644); err != nil {
		t.Fatalf("crear vídeo: %v", err)
	}
	ts := newPlayTestServer(t, nil, playback.ErrFFprobeNotFound)
	libraryID := createLibrary(t, ts.db, dir)
	id := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: video, title: "Pelicula",
	})

	status, body := uploadSubtitleFile(t, ts, ts.token, id, "Pelicula.es.srt",
		[]byte("1\n00:00:01,000 --> 00:00:02,000\nCaf\xe9\n"),
		map[string]string{"language": "es", "label": "Castellano"})
	if status != http.StatusCreated {
		t.Fatalf("subir subtítulo: código = %d, cuerpo = %s", status, body)
	}
	var uploaded uploadedSubtitleResponse
	if err := json.Unmarshal(body, &uploaded); err != nil {
		t.Fatalf("respuesta de subida: %v", err)
	}
	if uploaded.ID != "u1" || uploaded.Language != "es" || uploaded.Label != "Castellano" ||
		uploaded.Codec != "subrip" {
		t.Fatalf("respuesta de subida inesperada: %+v", uploaded)
	}

	var rowID int64
	var language, label, filename string
	if err := ts.db.QueryRow(`
		SELECT id, language, label, filename FROM item_subtitles
		 WHERE media_item_id = ?`, id).Scan(&rowID, &language, &label, &filename); err != nil {
		t.Fatalf("leer subtítulo guardado: %v", err)
	}
	if rowID != 1 || language != "es" || label != "Castellano" || filename != "1.srt" {
		t.Fatalf("fila inesperada: %d %q %q %q", rowID, language, label, filename)
	}
	path := filepath.Join(ts.subtitleDir, strconv.FormatInt(id, 10), "1.srt")
	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer archivo guardado: %v", err)
	}
	if !strings.Contains(string(stored), "Café") {
		t.Fatalf("el Latin-1 no se convirtió a UTF-8: %q", stored)
	}

	status, body = libraryHTTPGet(t, playURL(ts, id, ""), ts.token)
	if status != http.StatusOK {
		t.Fatalf("opciones: código = %d, cuerpo = %s", status, body)
	}
	var options playResponse
	if err := json.Unmarshal(body, &options); err != nil {
		t.Fatalf("opciones: %v", err)
	}
	if len(options.Subtitles) != 1 || options.Subtitles[0].ID != "u1" ||
		options.Subtitles[0].Source != "uploaded" || options.Subtitles[0].LanguageName != "Español" ||
		options.Subtitles[0].Title != "Castellano" {
		t.Fatalf("subtítulo subido ausente en las opciones: %+v", options.Subtitles)
	}

	status, body = libraryHTTPDo(t, http.MethodPut, itemPreferencesURL(ts.libraryTestServer, id),
		ts.token, map[string]any{"subtitle_id": "u1"})
	if status != http.StatusNoContent {
		t.Fatalf("preferencia u1: código = %d, cuerpo = %s", status, body)
	}
	status, body = libraryHTTPGet(t, playURL(ts, id, ""), ts.token)
	if err := json.Unmarshal(body, &options); err != nil || status != http.StatusOK {
		t.Fatalf("opciones con preferencia: %d %s (%v)", status, body, err)
	}
	if options.SelectedSubtitle == nil || *options.SelectedSubtitle != "u1" {
		t.Fatalf("preferencia u1 no aplicada: %+v", options.SelectedSubtitle)
	}

	status, body = libraryHTTPGet(t, ts.URL+options.Subtitles[0].URL, "")
	if status != http.StatusOK || !strings.Contains(string(body), "WEBVTT") {
		t.Fatalf("servir WebVTT: código = %d, cuerpo = %s", status, body)
	}

	otherVideo := filepath.Join(dir, "Otra.mkv")
	if err := os.WriteFile(otherVideo, []byte("video"), 0o644); err != nil {
		t.Fatalf("crear otro vídeo: %v", err)
	}
	otherID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: otherVideo, title: "Otra",
	})
	signature, expiresAt := ts.playback.Sign(otherID)
	foreignURL := ts.URL + "/api/v1/stream/" + strconv.FormatInt(otherID, 10) +
		"/subtitles/u1.vtt?exp=" + strconv.FormatInt(expiresAt.Unix(), 10) +
		"&sig=" + url.QueryEscape(signature)
	if status, body := libraryHTTPGet(t, foreignURL, ""); status != http.StatusNotFound {
		t.Fatalf("subtítulo de otro ítem: código = %d, cuerpo = %s", status, body)
	}

	status, body = libraryHTTPDo(t, http.MethodDelete, subtitleDeleteURL(ts, id, "u1"), ts.token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("borrar subtítulo: código = %d, cuerpo = %s", status, body)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("el archivo del subtítulo sigue en disco: %v", err)
	}
	var count int
	if err := ts.db.QueryRow(`SELECT count(*) FROM item_subtitles WHERE media_item_id = ?`, id).Scan(&count); err != nil {
		t.Fatalf("contar subtítulos: %v", err)
	}
	if count != 0 {
		t.Fatalf("quedan %d filas de subtítulos", count)
	}
	status, body = libraryHTTPGet(t, playURL(ts, id, ""), ts.token)
	if err := json.Unmarshal(body, &options); err != nil || status != http.StatusOK || len(options.Subtitles) != 0 {
		t.Fatalf("opciones tras borrar: %d %s (%v) %+v", status, body, err, options.Subtitles)
	}
	if status, body := libraryHTTPDo(t, http.MethodDelete,
		subtitleDeleteURL(ts, id, "u1"), ts.token, nil); status != http.StatusNotFound {
		t.Fatalf("borrar dos veces: código = %d, cuerpo = %s", status, body)
	}
}

func TestSubtitleUploadValidationYPermisos(t *testing.T) {
	dir := t.TempDir()
	video := filepath.Join(dir, "Pelicula.mkv")
	if err := os.WriteFile(video, []byte("video"), 0o644); err != nil {
		t.Fatalf("crear vídeo: %v", err)
	}
	ts := newPlayTestServer(t, nil, playback.ErrFFprobeNotFound)
	libraryID := createLibrary(t, ts.db, dir)
	id := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: video, title: "Pelicula",
	})
	userToken := nonAdminToken(t, ts.libraryTestServer)

	if status, body := uploadSubtitleFile(t, ts, userToken, id, "Pelicula.srt",
		[]byte("1\n00:00:01,000 --> 00:00:02,000\nHola\n"), nil); status != http.StatusForbidden {
		t.Fatalf("subida sin admin: código = %d, cuerpo = %s", status, body)
	}
	status, body := uploadSubtitleFile(t, ts, ts.token, id, "Pelicula.srt",
		[]byte("1\n00:00:01,000 --> 00:00:02,000\nHola\n"), nil)
	if status != http.StatusCreated {
		t.Fatalf("subida de referencia: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodDelete,
		subtitleDeleteURL(ts, id, "u1"), userToken, nil); status != http.StatusForbidden {
		t.Fatalf("borrado sin admin: código = %d, cuerpo = %s", status, body)
	}

	cases := []struct {
		name    string
		file    string
		content []byte
		fields  map[string]string
	}{
		{name: "extensión", file: "Pelicula.txt", content: []byte("Hola")},
		{name: "binario", file: "Pelicula.srt", content: []byte{'P', 'K', 0, 1, 2}},
		{name: "vacío", file: "Pelicula.srt", content: nil},
		{name: "idioma", file: "Pelicula.srt", content: []byte("Hola"), fields: map[string]string{"language": "español"}},
		{name: "demasiado grande", file: "Pelicula.srt",
			content: bytes.Repeat([]byte("a"), maxSubtitleUploadSize+1)},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			status, body := uploadSubtitleFile(t, ts, ts.token, id, test.file, test.content, test.fields)
			if status != http.StatusBadRequest && status != http.StatusRequestEntityTooLarge {
				t.Fatalf("código = %d, cuerpo = %s", status, body)
			}
		})
	}
}
