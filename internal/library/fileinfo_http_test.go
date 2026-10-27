package library

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/catlinux/cinecat/internal/playback"
)

type fakeFileInspector struct {
	calls    int
	lastPath string
	info     *playback.FileInfo
	err      error
}

func (f *fakeFileInspector) Inspect(ctx context.Context, path string) (*playback.FileInfo, error) {
	f.calls++
	f.lastPath = path
	if f.err != nil {
		return nil, f.err
	}
	return f.info, nil
}

func fileInfoFixture() *playback.FileInfo {
	return &playback.FileInfo{
		Container:         "mp4",
		ContainerLongName: "MPEG-4",
		SizeBytes:         1234,
		DurationSeconds:   95.5,
		BitRate:           2500000,
		ModifiedAt:        "2026-10-26T09:00:00Z",
		FileName:          "Pelicula (2019).mp4",
		Video: []playback.VideoStream{{
			Index:         0,
			Codec:         "h264",
			CodecLongName: "H.264 / AVC",
			Profile:       "High",
			Width:         1920,
			Height:        1080,
			FrameRate:     24,
			PixelFormat:   "yuv420p",
			BitDepth:      8,
			Language:      "spa",
			Default:       true,
		}},
		Audio: []playback.AudioStream{{
			Index:         1,
			Codec:         "aac",
			Channels:      2,
			ChannelLayout: "stereo",
			SampleRate:    48000,
			Language:      "spa",
			Default:       true,
		}},
		Subtitles: []playback.SubtitleStream{{
			Index:    2,
			Codec:    "subrip",
			Language: "spa",
			Text:     true,
		}},
		Chapters: 1,
		Quality:  "Full HD",
	}
}

func insertFileInfoItem(t *testing.T, ts *libraryTestServer, relative string, withFile bool) (int64, string) {
	t.Helper()
	root := t.TempDir()
	file := filepath.Join(root, filepath.FromSlash(relative))
	if withFile {
		writeFile(t, file)
	}
	libraryID := createLibrary(t, ts.db, root)
	id := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      file,
		title:     "Pelicula",
	})
	return id, file
}

func itemFileInfoURL(ts *libraryTestServer, id int64) string {
	return ts.URL + "/api/v1/items/" + strconv.FormatInt(id, 10) + "/file-info"
}

// Una biblioteca con varias carpetas debe servir los archivos de todas, no solo
// los de la principal; un archivo fuera de ellas sigue rechazado.
func TestFileInfoArchivoEnCarpetaSecundaria(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	ts.api.WithFileInspector(&fakeFileInspector{info: fileInfoFixture()})
	primary := t.TempDir()
	secondary := t.TempDir()
	outside := t.TempDir()
	libraryID := createLibrary(t, ts.db, primary)
	addScannerLibraryPath(t, ts.db, libraryID, primary)
	addScannerLibraryPath(t, ts.db, libraryID, secondary)

	inSecondary := filepath.Join(secondary, "Concierto (2019).mp4")
	writeFile(t, inSecondary)
	id := insertItemFixture(t, ts.db, itemFixture{libraryID: libraryID, path: inSecondary, title: "Concierto"})
	if status, body := libraryHTTPGet(t, itemFileInfoURL(ts, id), ts.token); status != http.StatusOK {
		t.Fatalf("archivo de la carpeta secundaria: código = %d, cuerpo = %s", status, body)
	}

	stray := filepath.Join(outside, "Ajena (2001).mp4")
	writeFile(t, stray)
	strayID := insertItemFixture(t, ts.db, itemFixture{libraryID: libraryID, path: stray, title: "Ajena"})
	if status, _ := libraryHTTPGet(t, itemFileInfoURL(ts, strayID), ts.token); status != http.StatusNotFound {
		t.Fatalf("archivo fuera de las carpetas: código = %d, se esperaba 404", status)
	}
}

func TestFileInfoDirectoYCache(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	id, file := insertFileInfoItem(t, ts, "Sub/Pelicula (2019).mp4", true)
	fake := &fakeFileInspector{info: fileInfoFixture()}
	ts.api.WithFileInspector(fake)

	status, body := libraryHTTPGet(t, itemFileInfoURL(ts, id), ts.token)
	if status != http.StatusOK {
		t.Fatalf("file-info: código = %d, cuerpo = %s", status, body)
	}
	var response struct {
		playback.FileInfo
		Path     string       `json:"path"`
		Playback PlaybackInfo `json:"playback"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("respuesta file-info: %v", err)
	}
	if response.Path != "Sub/Pelicula (2019).mp4" {
		t.Fatalf("path = %q", response.Path)
	}
	if response.Playback.Mode != "direct" || !response.Playback.Supported ||
		response.Playback.Description != "Reproducción directa (MP4)" {
		t.Fatalf("playback = %+v", response.Playback)
	}
	if response.Container != "mp4" || response.SizeBytes != 1234 ||
		response.DurationSeconds != 95.5 || response.Quality != "Full HD" ||
		len(response.Video) != 1 || len(response.Audio) != 1 || len(response.Subtitles) != 1 {
		t.Fatalf("ficha inesperada: %+v", response.FileInfo)
	}
	if fake.calls != 1 || filepath.Clean(fake.lastPath) != filepath.Clean(file) {
		t.Fatalf("inspector = %d llamadas, última ruta %q", fake.calls, fake.lastPath)
	}

	status, body = libraryHTTPGet(t, itemFileInfoURL(ts, id), ts.token)
	if status != http.StatusOK {
		t.Fatalf("segunda ficha: código = %d", status)
	}
	if fake.calls != 1 {
		t.Fatalf("la caché no evitó la segunda inspección: %d", fake.calls)
	}
}

func TestFileInfoHLSYContenedorNoSoportado(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	id, _ := insertFileInfoItem(t, ts, "Pelicula.mkv", true)
	fake := &fakeFileInspector{info: fileInfoFixture()}
	ts.api.WithFileInspector(fake)

	status, body := libraryHTTPGet(t, itemFileInfoURL(ts, id), ts.token)
	if status != http.StatusOK {
		t.Fatalf("MKV: código = %d, cuerpo = %s", status, body)
	}
	var response struct {
		Playback PlaybackInfo `json:"playback"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("MKV: %v", err)
	}
	if response.Playback.Mode != "hls" || !response.Playback.Supported ||
		response.Playback.Description != "Se recodifica a H.264 + AAC estéreo por HLS" {
		t.Fatalf("playback MKV = %+v", response.Playback)
	}

	id, _ = insertFileInfoItem(t, ts, "Pelicula.xyz", true)
	status, body = libraryHTTPGet(t, itemFileInfoURL(ts, id), ts.token)
	if status != http.StatusOK {
		t.Fatalf("XYZ: código = %d, cuerpo = %s", status, body)
	}
	response.Playback = PlaybackInfo{}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("XYZ: %v", err)
	}
	if response.Playback.Supported || response.Playback.Mode != "" ||
		response.Playback.Description != "Contenedor no soportado para reproducción" {
		t.Fatalf("playback XYZ = %+v", response.Playback)
	}
}

func TestFileInfoErrores(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	id, _ := insertFileInfoItem(t, ts, "Pelicula.mp4", true)

	if status, body := libraryHTTPGet(t, itemFileInfoURL(ts, 999999), ts.token); status != http.StatusNotFound {
		t.Fatalf("ítem inexistente: código = %d, cuerpo = %s", status, body)
	}

	missingID, _ := insertFileInfoItem(t, ts, "No existe.mp4", false)
	if status, body := libraryHTTPGet(t, itemFileInfoURL(ts, missingID), ts.token); status != http.StatusNotFound {
		t.Fatalf("archivo ausente: código = %d, cuerpo = %s", status, body)
	}

	if status, body := libraryHTTPGet(t, itemFileInfoURL(ts, id), ""); status != http.StatusUnauthorized {
		t.Fatalf("sin sesión: código = %d, cuerpo = %s", status, body)
	}

	// Sin inspector configurado, el endpoint avisa de que falta ffprobe.
	if status, body := libraryHTTPGet(t, itemFileInfoURL(ts, id), ts.token); status != http.StatusServiceUnavailable {
		t.Fatalf("sin inspector: código = %d, cuerpo = %s", status, body)
	}

	fake := &fakeFileInspector{err: playback.ErrFFprobeNotFound}
	ts.api.WithFileInspector(fake)
	if status, body := libraryHTTPGet(t, itemFileInfoURL(ts, id), ts.token); status != http.StatusServiceUnavailable {
		t.Fatalf("ffprobe ausente: código = %d, cuerpo = %s", status, body)
	}

	fake.err = errors.New("ffprobe falló")
	if status, body := libraryHTTPGet(t, itemFileInfoURL(ts, id), ts.token); status != http.StatusBadGateway {
		t.Fatalf("ffprobe roto: código = %d, cuerpo = %s", status, body)
	}
}

func TestFileInfoNormalizaArrays(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	id, _ := insertFileInfoItem(t, ts, "Pelicula.mp4", true)
	info := fileInfoFixture()
	info.Video = nil
	info.Audio = nil
	info.Subtitles = nil
	ts.api.WithFileInspector(&fakeFileInspector{info: info})

	status, body := libraryHTTPGet(t, itemFileInfoURL(ts, id), ts.token)
	if status != http.StatusOK {
		t.Fatalf("file-info: código = %d, cuerpo = %s", status, body)
	}
	var response map[string]json.RawMessage
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("respuesta: %v", err)
	}
	for _, field := range []string{"video", "audio", "subtitles"} {
		if string(response[field]) != "[]" {
			t.Fatalf("%s = %s, se esperaba []", field, response[field])
		}
	}
}
