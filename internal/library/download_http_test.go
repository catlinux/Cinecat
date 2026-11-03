package library

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func downloadTestURL(ts *libraryTestServer, id int64) string {
	return ts.URL + "/api/v1/items/" + strconv.FormatInt(id, 10) + "/download"
}

func downloadTestGet(t *testing.T, rawURL, token string, extra http.Header) (int, http.Header, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for key, values := range extra {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", rawURL, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leer respuesta: %v", err)
	}
	return resp.StatusCode, resp.Header.Clone(), body
}

func TestDescargaOriginalConRangos(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	root := t.TempDir()
	libraryID := createLibrary(t, ts.db, root)
	filename := "Película (2020).mkv"
	file := filepath.Join(root, filename)
	content := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	if err := os.WriteFile(file, content, 0o644); err != nil {
		t.Fatalf("crear archivo: %v", err)
	}
	itemID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      file,
		title:     "Película",
		kind:      KindMovie,
	})

	status, header, body := downloadTestGet(t, downloadTestURL(ts, itemID), ts.token, nil)
	if status != http.StatusOK {
		t.Fatalf("descarga completa: código = %d, cuerpo = %s", status, body)
	}
	if !bytes.Equal(body, content) {
		t.Fatalf("descarga completa: cuerpo = %q", body)
	}
	if got := header.Get("Content-Type"); got != "video/x-matroska" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got, want := header.Get("Content-Disposition"),
		"attachment; filename*=UTF-8''Pel%C3%ADcula%20%282020%29.mkv"; got != want {
		t.Fatalf("Content-Disposition = %q, se esperaba %q", got, want)
	}
	if got := header.Get("Accept-Ranges"); got != "bytes" {
		t.Fatalf("Accept-Ranges = %q", got)
	}
	if header.Get("Last-Modified") == "" {
		t.Fatal("falta Last-Modified")
	}

	rangeHeader := http.Header{"Range": []string{"bytes=10-19"}}
	status, header, body = downloadTestGet(t, downloadTestURL(ts, itemID), ts.token, rangeHeader)
	if status != http.StatusPartialContent {
		t.Fatalf("descarga por rango: código = %d, cuerpo = %s", status, body)
	}
	if !bytes.Equal(body, content[10:20]) {
		t.Fatalf("descarga por rango: cuerpo = %q", body)
	}
	if got, want := header.Get("Content-Range"), fmt.Sprintf("bytes 10-19/%d", len(content)); got != want {
		t.Fatalf("Content-Range = %q, se esperaba %q", got, want)
	}
	if got := header.Get("Content-Type"); got != "video/x-matroska" {
		t.Fatalf("Content-Type con rango = %q", got)
	}
}

func TestDescargaSinSesion(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	root := t.TempDir()
	libraryID := createLibrary(t, ts.db, root)
	file := filepath.Join(root, "peli.mkv")
	if err := os.WriteFile(file, []byte("contenido"), 0o644); err != nil {
		t.Fatalf("crear archivo: %v", err)
	}
	itemID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: file, title: "Peli", kind: KindMovie,
	})

	status, _, body := downloadTestGet(t, downloadTestURL(ts, itemID), "", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("sin sesión: código = %d, cuerpo = %s", status, body)
	}
}

func TestDescargaOcultaYSerie(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)

	visibleRoot := t.TempDir()
	hiddenRoot := t.TempDir()
	visibleLibrary := createLibrary(t, ts.db, visibleRoot)
	hiddenLibrary := createLibrary(t, ts.db, hiddenRoot)
	token, _ := partialAccessToken(t, ts, visibleLibrary)

	hiddenFile := filepath.Join(hiddenRoot, "secreto.mkv")
	if err := os.WriteFile(hiddenFile, []byte("secreto"), 0o644); err != nil {
		t.Fatalf("crear archivo oculto: %v", err)
	}
	hiddenItem := insertItemFixture(t, ts.db, itemFixture{
		libraryID: hiddenLibrary, path: hiddenFile, title: "Oculto", kind: KindMovie,
	})
	status, _, body := downloadTestGet(t, downloadTestURL(ts, hiddenItem), token, nil)
	if status != http.StatusNotFound {
		t.Fatalf("descarga oculta: código = %d, cuerpo = %s", status, body)
	}
	if bytes.Contains(body, []byte(hiddenRoot)) || bytes.Contains(body, []byte("secreto.mkv")) {
		t.Fatalf("la respuesta revela la ruta oculta: %s", body)
	}

	seriesFile := filepath.Join(visibleRoot, "serie.mkv")
	if err := os.WriteFile(seriesFile, []byte("serie"), 0o644); err != nil {
		t.Fatalf("crear archivo de serie: %v", err)
	}
	seriesItem := insertItemFixture(t, ts.db, itemFixture{
		libraryID: visibleLibrary, path: seriesFile, title: "Serie", kind: KindSeries,
	})
	status, _, body = downloadTestGet(t, downloadTestURL(ts, seriesItem), ts.token, nil)
	if status != http.StatusNotFound {
		t.Fatalf("descarga de serie: código = %d, cuerpo = %s", status, body)
	}
}

func TestDescargaTipoContenidoYNombre(t *testing.T) {
	for name, want := range map[string]string{
		"peli.mkv":     "video/x-matroska",
		"peli.MKV":     "video/x-matroska",
		"peli.mp4":     "video/mp4",
		"peli.m4v":     "video/mp4",
		"peli.avi":     "application/octet-stream",
		"sin-ext":      "application/octet-stream",
		"peli.MP4 ":    "application/octet-stream",
		"peli.mkv.bak": "application/octet-stream",
	} {
		if got := downloadContentType(name); got != want {
			t.Errorf("downloadContentType(%q) = %q, se esperaba %q", name, got, want)
		}
	}

	for name, want := range map[string]string{
		"Película (2020).mkv": "attachment; filename*=UTF-8''Pel%C3%ADcula%20%282020%29.mkv",
		"100%_seguro?.mkv":    "attachment; filename*=UTF-8''100%25_seguro%3F.mkv",
	} {
		if got := downloadDisposition(name); got != want {
			t.Errorf("downloadDisposition(%q) = %q, se esperaba %q", name, got, want)
		}
	}
}
