package library

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/catlinux/cinecat/internal/auth"
	"github.com/catlinux/cinecat/internal/settings"
)

func addUploadLibrary(t *testing.T, ts *libraryTestServer, root, kind string) int64 {
	t.Helper()
	result, err := ts.db.Exec(`INSERT INTO libraries (path, name, kind) VALUES (?, ?, ?)`, root, "Subidas", kind)
	if err != nil {
		t.Fatalf("crear biblioteca: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de biblioteca: %v", err)
	}
	if _, err := ts.db.Exec(
		`INSERT INTO library_paths (library_id, path) VALUES (?, ?)`, id, root,
	); err != nil {
		t.Fatalf("crear carpeta de biblioteca: %v", err)
	}
	return id
}

func uploadURL(ts *libraryTestServer, libraryID int64, dir, name, subdir string) string {
	values := url.Values{"dir": {dir}, "name": {name}}
	if subdir != "" {
		values.Set("subdir", subdir)
	}
	return fmt.Sprintf("%s/api/v1/libraries/%d/files?%s", ts.URL, libraryID, values.Encode())
}

func uploadRequest(t *testing.T, url, token string, body io.Reader, contentLength int64) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, url, body)
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.ContentLength = contentLength
	if contentLength < 0 {
		req.TransferEncoding = []string{"chunked"}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT %s: %v", url, err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leer respuesta: %v", err)
	}
	return resp.StatusCode, responseBody
}

func setUploadMaxGB(t *testing.T, ts *libraryTestServer, gigabytes int) {
	t.Helper()
	failures := ts.api.settingsStore.Set(context.Background(), map[string]json.RawMessage{
		settings.KeyUploadMaxGB: json.RawMessage(fmt.Sprint(gigabytes)),
	})
	if len(failures) != 0 {
		t.Fatalf("ajustar tamaño máximo: %v", failures)
	}
}

func hasPartFiles(t *testing.T, dir string) bool {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("leer %s: %v", dir, err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".part") {
			return true
		}
	}
	return false
}

func TestUploadArchivoCorrectoYSubcarpeta(t *testing.T) {
	ts := newTrashTestServer(t)
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := addUploadLibrary(t, ts, root, "video")
	name := "Peli (2026).mkv"
	content := []byte("contenido de vídeo")
	status, body := uploadRequest(t, uploadURL(ts, libraryID, root, name, ""),
		ts.token, bytes.NewReader(content), int64(len(content)))
	if status != http.StatusCreated {
		t.Fatalf("subida: código = %d, cuerpo = %s", status, body)
	}
	var result uploadResult
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("respuesta de subida: %v", err)
	}
	final := filepath.Join(root, name)
	if result.Path != final || result.Size != int64(len(content)) {
		t.Fatalf("resultado = %+v, se esperaba %s (%d)", result, final, len(content))
	}
	stored, err := os.ReadFile(final)
	if err != nil || !bytes.Equal(stored, content) {
		t.Fatalf("contenido final = %q, %v", stored, err)
	}
	if hasPartFiles(t, root) {
		t.Fatal("no debe quedar ningún .part")
	}

	subdir := "Temporada 1"
	status, body = uploadRequest(t, uploadURL(ts, libraryID, root, "Capitulo.mkv", subdir),
		ts.token, bytes.NewReader([]byte("capítulo")), int64(len("capítulo")))
	if status != http.StatusCreated {
		t.Fatalf("subida con subcarpeta: código = %d, cuerpo = %s", status, body)
	}
	info, err := os.Stat(filepath.Join(root, subdir))
	if err != nil || !info.IsDir() {
		t.Fatalf("subcarpeta no creada: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o775 {
		t.Fatalf("permisos de subcarpeta = %o, se esperaban 775", info.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(root, subdir, "Capitulo.mkv")); err != nil {
		t.Fatalf("archivo en subcarpeta: %v", err)
	}
}

func TestUploadValidaDirYNombre(t *testing.T) {
	ts := newTrashTestServer(t)
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := addUploadLibrary(t, ts, root, "video")
	other := t.TempDir()
	content := []byte("x")
	length := int64(len(content))

	outside := uploadURL(ts, libraryID, other, "Peli.mkv", "")
	status, body := uploadRequest(t, outside, ts.token, bytes.NewReader(content), length)
	if status != http.StatusBadRequest {
		t.Fatalf("dir fuera: código = %d, cuerpo = %s", status, body)
	}

	parent := root + string(os.PathSeparator) + ".." + string(os.PathSeparator) + filepath.Base(root)
	status, body = uploadRequest(t, uploadURL(ts, libraryID, parent, "Peli.mkv", ""),
		ts.token, bytes.NewReader(content), length)
	if status != http.StatusBadRequest {
		t.Fatalf("dir con ..: código = %d, cuerpo = %s", status, body)
	}

	trashDir := filepath.Join(root, trashFolderName)
	if err := os.Mkdir(trashDir, 0o775); err != nil {
		t.Fatalf("crear papelera: %v", err)
	}
	status, body = uploadRequest(t, uploadURL(ts, libraryID, trashDir, "Peli.mkv", ""),
		ts.token, bytes.NewReader(content), length)
	if status != http.StatusBadRequest {
		t.Fatalf("dir dentro de papelera: código = %d, cuerpo = %s", status, body)
	}

	for _, name := range []string{"sub/Peli.mkv", `sub\Peli.mkv`, ".Peli.mkv", "Peli.txt", "Peli"} {
		status, body = uploadRequest(t, uploadURL(ts, libraryID, root, name, ""),
			ts.token, bytes.NewReader(content), length)
		if status != http.StatusBadRequest {
			t.Fatalf("nombre %q: código = %d, cuerpo = %s", name, status, body)
		}
	}
	if hasPartFiles(t, root) {
		t.Fatal("no debe quedar ningún .part")
	}
}

func TestUploadConflictosYErroresDeCuerpo(t *testing.T) {
	ts := newTrashTestServer(t)
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := addUploadLibrary(t, ts, root, "video")
	content := []byte("x")

	final := filepath.Join(root, "Existe.mkv")
	writeTestFile(t, final, "anterior")
	status, body := uploadRequest(t, uploadURL(ts, libraryID, root, "Existe.mkv", ""),
		ts.token, bytes.NewReader(content), int64(len(content)))
	if status != http.StatusConflict {
		t.Fatalf("destino existente: código = %d, cuerpo = %s", status, body)
	}
	if stored, _ := os.ReadFile(final); string(stored) != "anterior" {
		t.Fatalf("se sobrescribió el destino: %q", stored)
	}

	status, body = uploadRequest(t, uploadURL(ts, libraryID, root, "SinLongitud.mkv", ""),
		ts.token, io.NopCloser(strings.NewReader("x")), -1)
	if status != http.StatusLengthRequired {
		t.Fatalf("sin Content-Length: código = %d, cuerpo = %s", status, body)
	}

	setUploadMaxGB(t, ts, 1)
	largeRequest := httptest.NewRequest(http.MethodPut,
		uploadURL(ts, libraryID, root, "Grande.mkv", ""), nil)
	largeRequest.SetPathValue("id", strconv.FormatInt(libraryID, 10))
	largeRequest.ContentLength = int64(1)<<30 + 1
	largeResponse := httptest.NewRecorder()
	ts.api.handleLibraryFileUpload(largeResponse, largeRequest)
	if largeResponse.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("demasiado grande: código = %d, cuerpo = %s", largeResponse.Code, largeResponse.Body.String())
	}

	shortRequest := httptest.NewRequest(http.MethodPut,
		uploadURL(ts, libraryID, root, "Corto.mkv", ""), strings.NewReader("x"))
	shortRequest.SetPathValue("id", strconv.FormatInt(libraryID, 10))
	shortRequest.ContentLength = 100
	shortResponse := httptest.NewRecorder()
	ts.api.handleLibraryFileUpload(shortResponse, shortRequest)
	if shortResponse.Code < 400 {
		t.Fatalf("cuerpo corto: código = %d, cuerpo = %s", shortResponse.Code, shortResponse.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "Corto.mkv")); !os.IsNotExist(err) {
		t.Fatal("no debe existir el archivo final de una subida incompleta")
	}
	if hasPartFiles(t, root) {
		t.Fatal("no debe quedar ningún .part tras una subida incompleta")
	}

	oldDiskFree := diskFree
	diskFree = func(string) (int64, bool) { return 0, true }
	t.Cleanup(func() { diskFree = oldDiskFree })
	status, body = uploadRequest(t, uploadURL(ts, libraryID, root, "SinEspacio.mkv", ""),
		ts.token, strings.NewReader("x"), int64(len(content)))
	if status != http.StatusInsufficientStorage {
		t.Fatalf("sin espacio: código = %d, cuerpo = %s", status, body)
	}
}

func TestUploadLanzaEscaneo(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	ts.api.scanJob = ts.job
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := addUploadLibrary(t, ts, root, "video")
	started := make(chan struct{})
	ts.job.scan = func(context.Context) (Result, error) {
		close(started)
		return Result{}, nil
	}
	content := []byte("x")
	status, body := uploadRequest(t, uploadURL(ts, libraryID, root, "Peli.mkv", ""),
		ts.token, bytes.NewReader(content), int64(len(content)))
	if status != http.StatusCreated {
		t.Fatalf("subida: código = %d, cuerpo = %s", status, body)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("la subida no lanzó el escaneo")
	}
}

func TestUploadAjusteApagadoNoAdminYMusica(t *testing.T) {
	ts := newTrashTestServer(t)
	root := t.TempDir()
	libraryID := addUploadLibrary(t, ts, root, "video")
	content := []byte("x")
	status, _ := uploadRequest(t, uploadURL(ts, libraryID, root, "Peli.mkv", ""),
		ts.token, bytes.NewReader(content), int64(len(content)))
	if status != http.StatusForbidden {
		t.Fatalf("ajuste apagado: código = %d, se esperaba 403", status)
	}

	setAllowFileChanges(t, ts, true)
	musicRoot := t.TempDir()
	musicID := addUploadLibrary(t, ts, musicRoot, "music")
	status, _ = uploadRequest(t, uploadURL(ts, musicID, musicRoot, "Peli.mkv", ""),
		ts.token, bytes.NewReader(content), int64(len(content)))
	if status != http.StatusBadRequest {
		t.Fatalf("biblioteca de música: código = %d, se esperaba 400", status)
	}

	hash, err := auth.HashPassword("Clave-segura-1")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if _, err := ts.db.Exec(
		`INSERT INTO users (username, password_hash, role) VALUES ('pepe', ?, 'user')`, hash,
	); err != nil {
		t.Fatalf("crear usuario normal: %v", err)
	}
	userToken, _, err := ts.auth.Login(context.Background(), "pepe", "Clave-segura-1")
	if err != nil {
		t.Fatalf("login normal: %v", err)
	}
	status, _ = uploadRequest(t, uploadURL(ts, libraryID, root, "Peli.mkv", ""),
		userToken, bytes.NewReader(content), int64(len(content)))
	if status != http.StatusForbidden {
		t.Fatalf("usuario normal: código = %d, se esperaba 403", status)
	}
}

func TestUploadPropietarioSinVariablesNoHaceNada(t *testing.T) {
	t.Setenv("CINECAT_FILE_UID", "")
	t.Setenv("CINECAT_FILE_GID", "")
	path := filepath.Join(t.TempDir(), "archivo")
	writeTestFile(t, path, "x")
	applyConfiguredOwner(path)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("el archivo desapareció: %v", err)
	}
}
