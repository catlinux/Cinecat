package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func createPhotoTestLibrary(t *testing.T, db *sql.DB, root string) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO libraries (path, name, kind) VALUES (?, ?, 'photos')`, root, "Fotos")
	if err != nil {
		t.Fatalf("crear biblioteca de fotos: %v", err)
	}
	id, _ := res.LastInsertId()
	addScannerLibraryPath(t, db, id, root)
	return id
}

func insertPhotoTestRow(t *testing.T, db *sql.DB, libraryID int64, path, folder, name string, takenAt *int64) int64 {
	t.Helper()
	info, err := os.Stat(path)
	size := int64(0)
	mtime := time.Now().Unix()
	if err == nil {
		size = info.Size()
		mtime = info.ModTime().Unix()
	}
	var taken any
	if takenAt != nil {
		taken = *takenAt
	}
	res, err := db.Exec(`
		INSERT INTO photos (library_id, path, folder, name, size, mtime, width, height, taken_at)
		VALUES (?, ?, ?, ?, ?, ?, 100, 50, ?)`,
		libraryID, path, folder, name, size, mtime, taken)
	if err != nil {
		t.Fatalf("insertar foto: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func TestPhotosAPIListaCarpetasOrdenYAcceso(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	root := t.TempDir()
	otherRoot := t.TempDir()
	libraryID := createPhotoTestLibrary(t, ts.db, root)
	hiddenLibraryID := createPhotoTestLibrary(t, ts.db, otherRoot)

	taken200 := time.Date(2021, 1, 1, 12, 0, 0, 0, time.Local).Unix()
	taken100 := time.Date(2020, 1, 1, 12, 0, 0, 0, time.Local).Unix()
	insertPhotoTestRow(t, ts.db, libraryID, filepath.Join(root, "raiz.jpg"), "", "raiz.jpg", nil)
	insertPhotoTestRow(t, ts.db, libraryID, filepath.Join(root, "2024", "b.jpg"), "2024", "b.jpg", &taken200)
	insertPhotoTestRow(t, ts.db, libraryID, filepath.Join(root, "2024", "a.jpg"), "2024", "a.jpg", &taken100)
	insertPhotoTestRow(t, ts.db, libraryID, filepath.Join(root, "2024", "sub", "c.jpg"), "2024/sub", "c.jpg", nil)
	hiddenID := insertPhotoTestRow(t, ts.db, hiddenLibraryID, filepath.Join(otherRoot, "oculta.jpg"), "", "oculta.jpg", nil)

	status, body, _ := musicTestGET(t, ts.URL+"/api/v1/photos?library_id="+strconv.FormatInt(libraryID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("lista raíz: %d %s", status, body)
	}
	var rootList photosResponse
	if err := json.Unmarshal(body, &rootList); err != nil {
		t.Fatalf("lista raíz JSON: %v", err)
	}
	if rootList.Total != 1 || len(rootList.Items) != 1 || rootList.Items[0].Name != "raiz.jpg" {
		t.Fatalf("lista raíz inesperada: %+v", rootList)
	}
	if len(rootList.Folders) != 1 || rootList.Folders[0] != "2024" {
		t.Fatalf("carpetas de la raíz: %+v", rootList.Folders)
	}

	status, body, _ = musicTestGET(t, ts.URL+"/api/v1/photos?library_id="+strconv.FormatInt(libraryID, 10)+"&folder=2024&sort=date", ts.token)
	if status != http.StatusOK {
		t.Fatalf("lista fecha: %d %s", status, body)
	}
	var dated photosResponse
	if err := json.Unmarshal(body, &dated); err != nil {
		t.Fatalf("lista fecha JSON: %v", err)
	}
	if dated.Total != 2 || len(dated.Items) != 2 || dated.Items[0].Name != "b.jpg" || dated.Items[1].Name != "a.jpg" {
		t.Fatalf("orden por fecha inesperado: %+v", dated.Items)
	}
	if len(dated.Folders) != 1 || dated.Folders[0] != "sub" {
		t.Fatalf("subcarpetas de 2024: %+v", dated.Folders)
	}

	status, body, _ = musicTestGET(t, ts.URL+"/api/v1/photos?library_id="+strconv.FormatInt(libraryID, 10)+"&folder=2024&sort=name", ts.token)
	if status != http.StatusOK {
		t.Fatalf("lista nombre: %d %s", status, body)
	}
	var named photosResponse
	if err := json.Unmarshal(body, &named); err != nil {
		t.Fatalf("lista nombre JSON: %v", err)
	}
	if len(named.Items) != 2 || named.Items[0].Name != "a.jpg" || named.Items[1].Name != "b.jpg" {
		t.Fatalf("orden por nombre inesperado: %+v", named.Items)
	}

	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/photos?library_id="+strconv.FormatInt(libraryID, 10)+"&folder=../", ts.token); status != http.StatusBadRequest {
		t.Fatalf("path traversal: %d", status)
	}
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/photos?library_id="+strconv.FormatInt(libraryID, 10)+"&sort=nope", ts.token); status != http.StatusBadRequest {
		t.Fatalf("orden inválido: %d", status)
	}
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/photos", ts.token); status != http.StatusBadRequest {
		t.Fatalf("sin library_id: %d", status)
	}
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/photos?library_id=999", ts.token); status != http.StatusNotFound {
		t.Fatalf("biblioteca inexistente: %d", status)
	}
	videoRoot := t.TempDir()
	videoRes, _ := ts.db.Exec(`INSERT INTO libraries (path, name, kind) VALUES (?, ?, 'video')`, videoRoot, "Vídeos")
	videoID, _ := videoRes.LastInsertId()
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/photos?library_id="+strconv.FormatInt(videoID, 10), ts.token); status != http.StatusNotFound {
		t.Fatalf("biblioteca que no es de fotos: %d", status)
	}

	partial, _ := partialAccessToken(t, ts, libraryID)
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/photos?library_id="+strconv.FormatInt(libraryID, 10), partial); status != http.StatusOK {
		t.Fatalf("usuario con acceso: %d", status)
	}
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/photos?library_id="+strconv.FormatInt(hiddenLibraryID, 10), partial); status != http.StatusNotFound {
		t.Fatalf("usuario sin acceso: %d", status)
	}
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/photos/"+strconv.FormatInt(hiddenID, 10)+"/original", partial); status != http.StatusNotFound {
		t.Fatalf("foto sin acceso: %d", status)
	}
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/photos/"+strconv.FormatInt(hiddenID, 10)+"/thumb", partial); status != http.StatusNotFound {
		t.Fatalf("miniatura sin acceso: %d", status)
	}
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/photos?library_id="+strconv.FormatInt(libraryID, 10), ""); status != http.StatusUnauthorized {
		t.Fatalf("sin sesión: %d", status)
	}
}

func TestPhotosThumbYOriginalConRunnerYRangos(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	root := t.TempDir()
	libraryID := createPhotoTestLibrary(t, ts.db, root)
	photoPath := filepath.Join(root, "2024", "foto.jpg")
	if err := os.MkdirAll(filepath.Dir(photoPath), 0o755); err != nil {
		t.Fatalf("crear carpeta: %v", err)
	}
	content := []byte("0123456789abcdef")
	if err := os.WriteFile(photoPath, content, 0o644); err != nil {
		t.Fatalf("crear foto: %v", err)
	}
	photoID := insertPhotoTestRow(t, ts.db, libraryID, photoPath, "2024", "foto.jpg", nil)
	cacheDir := t.TempDir()
	calls := 0
	ts.api.WithPhotoThumbnails(cacheDir).WithPhotoThumbnailRunner(func(ctx context.Context, args []string) error {
		calls++
		return os.WriteFile(args[len(args)-1], []byte{0xff, 0xd8, 0xff, 0xd9}, 0o644)
	})

	status, body, headers := musicTestGET(t, ts.URL+"/api/v1/photos/"+strconv.FormatInt(photoID, 10)+"/thumb", ts.token)
	if status != http.StatusOK || len(body) != 4 || headers.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("miniatura: %d %s %+v", status, body, headers)
	}
	if headers.Get("Cache-Control") != "private, max-age=86400" || headers.Get("ETag") == "" {
		t.Fatalf("cabeceras de miniatura: %+v", headers)
	}
	if calls != 1 {
		t.Fatalf("llamadas a FFmpeg = %d, se esperaba 1", calls)
	}
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/photos/"+strconv.FormatInt(photoID, 10)+"/thumb", ts.token); status != http.StatusOK || calls != 1 {
		t.Fatalf("segunda miniatura: %d, llamadas=%d", status, calls)
	}
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/photos/"+strconv.FormatInt(photoID, 10)+"/thumb", nil)
	req.Header.Set("Authorization", "Bearer "+ts.token)
	req.Header.Set("If-None-Match", headers.Get("ETag"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET con ETag: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("ETag de miniatura: %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/v1/photos/"+strconv.FormatInt(photoID, 10)+"/original", nil)
	req.Header.Set("Authorization", "Bearer "+ts.token)
	req.Header.Set("Range", "bytes=0-3")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET original: %v", err)
	}
	defer resp.Body.Close()
	part, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusPartialContent || string(part) != "0123" {
		t.Fatalf("rango del original: %d %q", resp.StatusCode, part)
	}
	if resp.Header.Get("Content-Type") != "image/jpeg" || resp.Header.Get("Content-Range") == "" {
		t.Fatalf("cabeceras del original: %+v", resp.Header)
	}

	outside := filepath.Join(t.TempDir(), "fuera.jpg")
	if err := os.WriteFile(outside, content, 0o644); err != nil {
		t.Fatalf("crear foto externa: %v", err)
	}
	outsideID := insertPhotoTestRow(t, ts.db, libraryID, outside, "", "fuera.jpg", nil)
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/photos/"+strconv.FormatInt(outsideID, 10)+"/original", ts.token); status != http.StatusNotFound {
		t.Fatalf("foto fuera de la biblioteca: %d", status)
	}
	if status, _, _ := musicTestGET(t, ts.URL+"/api/v1/photos/999/original", ts.token); status != http.StatusNotFound {
		t.Fatalf("foto inexistente: %d", status)
	}
}
