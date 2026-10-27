package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/catlinux/cinecat/internal/auth"
	"github.com/catlinux/cinecat/internal/settings"
)

func newTrashTestServer(t *testing.T) *libraryTestServer {
	t.Helper()
	ts := newLibraryHTTPTestServer(t)
	ts.api.scanJob = nil
	return ts
}

func setAllowFileChanges(t *testing.T, ts *libraryTestServer, allowed bool) {
	t.Helper()
	value := "false"
	if allowed {
		value = "true"
	}
	failures := ts.api.settingsStore.Set(context.Background(), map[string]json.RawMessage{
		settings.KeyAllowFileChanges: json.RawMessage(value),
	})
	if len(failures) != 0 {
		t.Fatalf("activar cambios de archivos: %v", failures)
	}
}

func addTrashLibrary(t *testing.T, ts *libraryTestServer, root string) int64 {
	t.Helper()
	result, err := ts.db.Exec(`INSERT INTO libraries (path, name) VALUES (?, ?)`, root, "Vídeo")
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

func addTrashItem(t *testing.T, db *sql.DB, libraryID int64, path, title string) int64 {
	t.Helper()
	result, err := db.Exec(`
		INSERT INTO media_items (library_id, path, title, year, kind)
		VALUES (?, ?, ?, 1999, 'movie')`, libraryID, path, title)
	if err != nil {
		t.Fatalf("crear ítem: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de ítem: %v", err)
	}
	return id
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o775); err != nil {
		t.Fatalf("crear carpeta de %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o664); err != nil {
		t.Fatalf("escribir %s: %v", path, err)
	}
}

func trashRequest(t *testing.T, method, url, token string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leer respuesta: %v", err)
	}
	return resp.StatusCode, body
}

func decodeTrash(t *testing.T, body []byte) trashItemDTO {
	t.Helper()
	var item trashItemDTO
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("respuesta de papelera: %v (%s)", err, body)
	}
	return item
}

func requireGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s debería haber desaparecido (err=%v)", path, err)
	}
}

func requireExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("%s debería existir: %v", path, err)
	}
}

func countTrashRows(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM trash_items`).Scan(&count); err != nil {
		t.Fatalf("contar papelera: %v", err)
	}
	return count
}

func trashEntries(t *testing.T, root string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, trashFolderName))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("leer papelera: %v", err)
	}
	return entries
}

func TestTrashMueveArchivoConSubtitulosYRestauraLaFicha(t *testing.T) {
	ts := newTrashTestServer(t)
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := addTrashLibrary(t, ts, root)
	original := filepath.Join(root, "Matrix (1999).mkv")
	subtitle := filepath.Join(root, "Matrix (1999).es.srt")
	otherSubtitle := filepath.Join(root, "Matrix (1999).vtt")
	writeTestFile(t, original, "película")
	writeTestFile(t, subtitle, "subtítulo")
	writeTestFile(t, otherSubtitle, "subtítulo vtt")
	itemID := addTrashItem(t, ts.db, libraryID, original, "Matrix")

	var userID int64
	if err := ts.db.QueryRow(`SELECT id FROM users WHERE username = 'ana'`).Scan(&userID); err != nil {
		t.Fatalf("id de usuario: %v", err)
	}
	collection, err := ts.db.Exec(`INSERT INTO collections (name, source) VALUES ('Saga', 'manual')`)
	if err != nil {
		t.Fatalf("crear colección: %v", err)
	}
	collectionID, _ := collection.LastInsertId()
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO watch_state (user_id, media_item_id, watched_at) VALUES (?, ?, ?)`,
			[]any{userID, itemID, "2026-01-02T03:04:05Z"}},
		{`INSERT INTO playback_state (user_id, media_item_id, position_seconds) VALUES (?, ?, ?)`,
			[]any{userID, itemID, 42}},
		{`INSERT INTO playback_progress (user_id, media_item_id, position_seconds, duration_seconds) VALUES (?, ?, ?, ?)`,
			[]any{userID, itemID, 12.5, 100.0}},
		{`INSERT INTO item_preferences (user_id, media_item_id, audio_index, subtitle_id, quality) VALUES (?, ?, ?, ?, ?)`,
			[]any{userID, itemID, 2, "es", "1080p"}},
		{`INSERT INTO item_lists (user_id, media_item_id, list, added_at) VALUES (?, ?, 'favorite', ?)`,
			[]any{userID, itemID, "2026-01-02T03:04:05Z"}},
		{`INSERT INTO collection_items (collection_id, media_item_id, position, excluded) VALUES (?, ?, ?, 0)`,
			[]any{collectionID, itemID, 1}},
	}
	for _, statement := range statements {
		if _, err := ts.db.Exec(statement.query, statement.args...); err != nil {
			t.Fatalf("preparar instantánea: %v", err)
		}
	}

	status, body := trashRequest(t, http.MethodDelete,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("mover a la papelera: código = %d, cuerpo = %s", status, body)
	}
	moved := decodeTrash(t, body)
	if moved.MovedKind != "file" {
		t.Fatalf("moved_kind = %q, se esperaba file", moved.MovedKind)
	}
	if moved.OriginalPath != original || moved.Title != "Matrix" || moved.Year == nil || *moved.Year != 1999 {
		t.Fatalf("elemento de papelera inesperado: %+v", moved)
	}
	if moved.DeletedAt == "" || moved.ExpiresAt == "" {
		t.Fatalf("faltan fechas en la papelera: %+v", moved)
	}
	requireGone(t, original)
	requireGone(t, subtitle)
	requireGone(t, otherSubtitle)
	if len(trashEntries(t, root)) != 3 {
		t.Fatalf("la papelera debería tener 3 archivos, tiene %d", len(trashEntries(t, root)))
	}
	if countTrashRows(t, ts.db) != 1 {
		t.Fatal("la papelera debería tener una fila")
	}
	var extraJSON string
	if err := ts.db.QueryRow(`SELECT extra_paths FROM trash_items WHERE id = ?`, moved.ID).Scan(&extraJSON); err != nil {
		t.Fatalf("leer extra_paths: %v", err)
	}
	var extra []trashExtraPath
	if err := json.Unmarshal([]byte(extraJSON), &extra); err != nil || len(extra) != 2 {
		t.Fatalf("extra_paths = %q (%v), se esperaban 2 subtítulos", extraJSON, err)
	}
	var mediaCount, dependentCount int
	if err := ts.db.QueryRow(`SELECT count(*) FROM media_items WHERE id = ?`, itemID).Scan(&mediaCount); err != nil {
		t.Fatalf("contar ítem: %v", err)
	}
	if mediaCount != 0 {
		t.Fatal("el ítem debería haberse borrado de media_items")
	}
	for _, table := range []string{"watch_state", "playback_state", "playback_progress", "item_preferences", "item_lists", "collection_items"} {
		if err := ts.db.QueryRow(`SELECT count(*) FROM `+table+` WHERE media_item_id = ?`, itemID).Scan(&dependentCount); err != nil {
			t.Fatalf("contar %s: %v", table, err)
		}
		if dependentCount != 0 {
			t.Fatalf("%s debería haberse borrado en cascada", table)
		}
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/trash", ts.token)
	if status != http.StatusOK {
		t.Fatalf("listar papelera: código = %d, cuerpo = %s", status, body)
	}
	var listed trashListResponse
	if err := json.Unmarshal(body, &listed); err != nil {
		t.Fatalf("respuesta de lista: %v", err)
	}
	if len(listed.Items) != 1 || listed.Items[0].ID != moved.ID {
		t.Fatalf("lista de papelera inesperada: %+v", listed)
	}

	status, body = libraryHTTPPost(t,
		ts.URL+"/api/v1/trash/"+strconv.FormatInt(moved.ID, 10)+"/restore", ts.token)
	if status != http.StatusOK {
		t.Fatalf("restaurar: código = %d, cuerpo = %s", status, body)
	}
	var restored struct {
		ItemID int64 `json:"item_id"`
	}
	if err := json.Unmarshal(body, &restored); err != nil {
		t.Fatalf("respuesta de restaurar: %v", err)
	}
	if restored.ItemID != itemID {
		t.Fatalf("item_id = %d, se esperaba el mismo %d", restored.ItemID, itemID)
	}
	requireExists(t, original)
	requireExists(t, subtitle)
	requireExists(t, otherSubtitle)
	if content, _ := os.ReadFile(original); string(content) != "película" {
		t.Fatalf("contenido restaurado = %q", content)
	}
	if len(trashEntries(t, root)) != 0 {
		t.Fatalf("la papelera debería estar vacía: %v", trashEntries(t, root))
	}
	if countTrashRows(t, ts.db) != 0 {
		t.Fatal("la papelera debería haberse vaciado")
	}

	var position int
	if err := ts.db.QueryRow(`SELECT position_seconds FROM playback_state WHERE user_id = ? AND media_item_id = ?`,
		userID, restored.ItemID).Scan(&position); err != nil || position != 42 {
		t.Fatalf("playback_state restaurado: %d, %v", position, err)
	}
	var progressPosition, duration float64
	if err := ts.db.QueryRow(`SELECT position_seconds, duration_seconds FROM playback_progress WHERE user_id = ? AND media_item_id = ?`,
		userID, restored.ItemID).Scan(&progressPosition, &duration); err != nil ||
		progressPosition != 12.5 || duration != 100 {
		t.Fatalf("playback_progress restaurado: %v, %v, %v", progressPosition, duration, err)
	}
	var watchedAt string
	if err := ts.db.QueryRow(`SELECT watched_at FROM watch_state WHERE user_id = ? AND media_item_id = ?`,
		userID, restored.ItemID).Scan(&watchedAt); err != nil || watchedAt != "2026-01-02T03:04:05Z" {
		t.Fatalf("watch_state restaurado: %q, %v", watchedAt, err)
	}
	var audio int
	var subtitleID, quality string
	if err := ts.db.QueryRow(`SELECT audio_index, subtitle_id, quality FROM item_preferences WHERE user_id = ? AND media_item_id = ?`,
		userID, restored.ItemID).Scan(&audio, &subtitleID, &quality); err != nil ||
		audio != 2 || subtitleID != "es" || quality != "1080p" {
		t.Fatalf("item_preferences restaurado: %d, %q, %q, %v", audio, subtitleID, quality, err)
	}
	var list string
	if err := ts.db.QueryRow(`SELECT list FROM item_lists WHERE user_id = ? AND media_item_id = ?`,
		userID, restored.ItemID).Scan(&list); err != nil || list != "favorite" {
		t.Fatalf("item_lists restaurado: %q, %v", list, err)
	}
	var collectionPosition int
	if err := ts.db.QueryRow(`SELECT position FROM collection_items WHERE collection_id = ? AND media_item_id = ?`,
		collectionID, restored.ItemID).Scan(&collectionPosition); err != nil || collectionPosition != 1 {
		t.Fatalf("collection_items restaurado: %d, %v", collectionPosition, err)
	}
}

func TestTrashMueveCarpetaDePelicula(t *testing.T) {
	ts := newTrashTestServer(t)
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := addTrashLibrary(t, ts, root)
	folder := filepath.Join(root, "Matrix")
	original := filepath.Join(folder, "Matrix.mkv")
	writeTestFile(t, original, "película")
	writeTestFile(t, filepath.Join(folder, "Matrix.es.srt"), "subtítulo")
	itemID := addTrashItem(t, ts.db, libraryID, original, "Matrix")

	status, body := trashRequest(t, http.MethodDelete,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("mover carpeta: código = %d, cuerpo = %s", status, body)
	}
	moved := decodeTrash(t, body)
	if moved.MovedKind != "folder" || moved.OriginalPath != folder {
		t.Fatalf("se esperaba la carpeta en la papelera: %+v", moved)
	}
	var extraJSON string
	if err := ts.db.QueryRow(`SELECT extra_paths FROM trash_items WHERE id = ?`, moved.ID).Scan(&extraJSON); err != nil || extraJSON != "[]" {
		t.Fatalf("extra_paths de carpeta = %q, %v", extraJSON, err)
	}
	requireGone(t, folder)
	entries := trashEntries(t, root)
	if len(entries) != 1 || !entries[0].IsDir() {
		t.Fatalf("la papelera debería contener una carpeta: %v", entries)
	}

	status, body = libraryHTTPPost(t,
		ts.URL+"/api/v1/trash/"+strconv.FormatInt(moved.ID, 10)+"/restore", ts.token)
	if status != http.StatusOK {
		t.Fatalf("restaurar carpeta: código = %d, cuerpo = %s", status, body)
	}
	requireExists(t, original)
	requireExists(t, filepath.Join(folder, "Matrix.es.srt"))
	if len(trashEntries(t, root)) != 0 {
		t.Fatalf("la papelera debería estar vacía: %v", trashEntries(t, root))
	}
}

func TestTrashNoMueveCarpetaConOtroVideo(t *testing.T) {
	ts := newTrashTestServer(t)
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := addTrashLibrary(t, ts, root)
	folder := filepath.Join(root, "Saga")
	original := filepath.Join(folder, "Uno.mkv")
	other := filepath.Join(folder, "Dos.mkv")
	writeTestFile(t, original, "uno")
	writeTestFile(t, other, "dos")
	itemID := addTrashItem(t, ts.db, libraryID, original, "Uno")

	status, body := trashRequest(t, http.MethodDelete,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("mover archivo: código = %d, cuerpo = %s", status, body)
	}
	moved := decodeTrash(t, body)
	if moved.MovedKind != "file" || moved.OriginalPath != original {
		t.Fatalf("se esperaba mover solo el archivo: %+v", moved)
	}
	requireGone(t, original)
	requireExists(t, other)
	requireExists(t, folder)
}

func TestTrashRestaurarConRutaOcupada(t *testing.T) {
	ts := newTrashTestServer(t)
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := addTrashLibrary(t, ts, root)
	original := filepath.Join(root, "Peli.mkv")
	writeTestFile(t, original, "original")
	itemID := addTrashItem(t, ts.db, libraryID, original, "Peli")
	status, body := trashRequest(t, http.MethodDelete,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("mover: código = %d, cuerpo = %s", status, body)
	}
	moved := decodeTrash(t, body)
	writeTestFile(t, original, "nueva")

	status, body = libraryHTTPPost(t,
		ts.URL+"/api/v1/trash/"+strconv.FormatInt(moved.ID, 10)+"/restore", ts.token)
	if status != http.StatusConflict {
		t.Fatalf("restaurar con ruta ocupada: código = %d, cuerpo = %s", status, body)
	}
	if content, _ := os.ReadFile(original); string(content) != "nueva" {
		t.Fatalf("la ruta ocupada cambió: %q", content)
	}
	if countTrashRows(t, ts.db) != 1 {
		t.Fatal("la papelera debe conservar el elemento")
	}
	if len(trashEntries(t, root)) != 1 {
		t.Fatalf("la papelera debería conservar el archivo: %v", trashEntries(t, root))
	}
}

func TestTrashRestaurarConIDOcupado(t *testing.T) {
	ts := newTrashTestServer(t)
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := addTrashLibrary(t, ts, root)
	original := filepath.Join(root, "Peli.mkv")
	writeTestFile(t, original, "original")
	itemID := addTrashItem(t, ts.db, libraryID, original, "Peli")
	var userID int64
	if err := ts.db.QueryRow(`SELECT id FROM users WHERE username = 'ana'`).Scan(&userID); err != nil {
		t.Fatalf("id de usuario: %v", err)
	}
	if _, err := ts.db.Exec(
		`INSERT INTO watch_state (user_id, media_item_id) VALUES (?, ?)`, userID, itemID,
	); err != nil {
		t.Fatalf("preparar visto: %v", err)
	}
	status, body := trashRequest(t, http.MethodDelete,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("mover: código = %d, cuerpo = %s", status, body)
	}
	moved := decodeTrash(t, body)
	other := filepath.Join(root, "Otro.mkv")
	writeTestFile(t, other, "otro")
	if _, err := ts.db.Exec(`
		INSERT INTO media_items (id, library_id, path, title, kind)
		VALUES (?, ?, ?, 'Otro', 'movie')`, itemID, libraryID, other); err != nil {
		t.Fatalf("ocupar el id: %v", err)
	}

	status, body = libraryHTTPPost(t,
		ts.URL+"/api/v1/trash/"+strconv.FormatInt(moved.ID, 10)+"/restore", ts.token)
	if status != http.StatusOK {
		t.Fatalf("restaurar con id ocupado: código = %d, cuerpo = %s", status, body)
	}
	var restored struct {
		ItemID int64 `json:"item_id"`
	}
	if err := json.Unmarshal(body, &restored); err != nil {
		t.Fatalf("respuesta: %v", err)
	}
	if restored.ItemID == itemID {
		t.Fatalf("se esperaba un id nuevo, se reutilizó %d", itemID)
	}
	var watched int
	if err := ts.db.QueryRow(`SELECT count(*) FROM watch_state WHERE media_item_id = ?`, restored.ItemID).Scan(&watched); err != nil || watched != 1 {
		t.Fatalf("visto no restaurado con el id nuevo: %d, %v", watched, err)
	}
	requireExists(t, original)
}

func TestTrashRestauraAunqueFaltenUsuarioYColeccion(t *testing.T) {
	ts := newTrashTestServer(t)
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := addTrashLibrary(t, ts, root)
	original := filepath.Join(root, "Peli.mkv")
	writeTestFile(t, original, "original")
	itemID := addTrashItem(t, ts.db, libraryID, original, "Peli")
	hash, err := auth.HashPassword("Clave-segura-1")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	user, err := ts.db.Exec(
		`INSERT INTO users (username, password_hash, role) VALUES ('pepe', ?, 'user')`, hash)
	if err != nil {
		t.Fatalf("crear usuario: %v", err)
	}
	userID, _ := user.LastInsertId()
	collection, err := ts.db.Exec(`INSERT INTO collections (name, source) VALUES ('Saga', 'manual')`)
	if err != nil {
		t.Fatalf("crear colección: %v", err)
	}
	collectionID, _ := collection.LastInsertId()
	if _, err := ts.db.Exec(
		`INSERT INTO item_lists (user_id, media_item_id, list) VALUES (?, ?, 'favorite')`, userID, itemID,
	); err != nil {
		t.Fatalf("crear lista: %v", err)
	}
	if _, err := ts.db.Exec(
		`INSERT INTO collection_items (collection_id, media_item_id) VALUES (?, ?)`, collectionID, itemID,
	); err != nil {
		t.Fatalf("crear colección e ítem: %v", err)
	}
	status, body := trashRequest(t, http.MethodDelete,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("mover: código = %d, cuerpo = %s", status, body)
	}
	moved := decodeTrash(t, body)
	if _, err := ts.db.Exec(`DELETE FROM users WHERE id = ?`, userID); err != nil {
		t.Fatalf("borrar usuario: %v", err)
	}
	if _, err := ts.db.Exec(`DELETE FROM collections WHERE id = ?`, collectionID); err != nil {
		t.Fatalf("borrar colección: %v", err)
	}
	status, body = libraryHTTPPost(t,
		ts.URL+"/api/v1/trash/"+strconv.FormatInt(moved.ID, 10)+"/restore", ts.token)
	if status != http.StatusOK {
		t.Fatalf("restaurar con dependencias borradas: código = %d, cuerpo = %s", status, body)
	}
	var restored struct {
		ItemID int64 `json:"item_id"`
	}
	if err := json.Unmarshal(body, &restored); err != nil {
		t.Fatalf("respuesta: %v", err)
	}
	var lists int
	if err := ts.db.QueryRow(`SELECT count(*) FROM item_lists WHERE media_item_id = ?`, restored.ItemID).Scan(&lists); err != nil || lists != 0 {
		t.Fatalf("item_lists inesperado: %d, %v", lists, err)
	}
	requireExists(t, original)
}

func TestTrashEliminaDefinitivamenteYPurga(t *testing.T) {
	ts := newTrashTestServer(t)
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := addTrashLibrary(t, ts, root)
	original := filepath.Join(root, "Peli.mkv")
	writeTestFile(t, original, "original")
	itemID := addTrashItem(t, ts.db, libraryID, original, "Peli")

	fixed := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	ts.api.now = func() time.Time { return fixed }
	status, body := trashRequest(t, http.MethodDelete,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("mover: código = %d, cuerpo = %s", status, body)
	}
	moved := decodeTrash(t, body)
	if len(trashEntries(t, root)) != 1 {
		t.Fatalf("la papelera debería tener el archivo: %v", trashEntries(t, root))
	}
	status, _ = trashRequest(t, http.MethodDelete,
		ts.URL+"/api/v1/trash/"+strconv.FormatInt(moved.ID, 10), ts.token)
	if status != http.StatusNoContent {
		t.Fatalf("eliminar definitivamente: código = %d", status)
	}
	requireGone(t, original)
	if len(trashEntries(t, root)) != 0 {
		t.Fatalf("la papelera debería estar vacía: %v", trashEntries(t, root))
	}
	if countTrashRows(t, ts.db) != 0 {
		t.Fatal("la fila debería haberse borrado")
	}

	otherOriginal := filepath.Join(root, "Vieja.mkv")
	writeTestFile(t, otherOriginal, "vieja")
	otherID := addTrashItem(t, ts.db, libraryID, otherOriginal, "Vieja")
	status, body = trashRequest(t, http.MethodDelete,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(otherID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("mover la segunda: código = %d, cuerpo = %s", status, body)
	}
	_ = decodeTrash(t, body)
	ts.api.now = func() time.Time { return fixed.Add(15 * 24 * time.Hour) }
	status, body = libraryHTTPPost(t, ts.URL+"/api/v1/trash/purge", ts.token)
	if status != http.StatusOK {
		t.Fatalf("purgar: código = %d, cuerpo = %s", status, body)
	}
	var purged struct {
		Purged int `json:"purged"`
	}
	if err := json.Unmarshal(body, &purged); err != nil {
		t.Fatalf("respuesta de purga: %v", err)
	}
	if purged.Purged != 1 {
		t.Fatalf("purgados = %d, se esperaba 1", purged.Purged)
	}
	requireGone(t, otherOriginal)
	if len(trashEntries(t, root)) != 0 {
		t.Fatalf("la papelera debería estar vacía: %v", trashEntries(t, root))
	}
	if countTrashRows(t, ts.db) != 0 {
		t.Fatal("la fila caducada debería haberse purgado")
	}
}

func TestTrashAjusteApagadoYOperacionesSiempreDisponibles(t *testing.T) {
	ts := newTrashTestServer(t)
	root := t.TempDir()
	libraryID := addTrashLibrary(t, ts, root)
	original := filepath.Join(root, "Peli.mkv")
	writeTestFile(t, original, "original")
	itemID := addTrashItem(t, ts.db, libraryID, original, "Peli")

	status, _ := trashRequest(t, http.MethodDelete,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10), ts.token)
	if status != http.StatusForbidden {
		t.Fatalf("con el ajuste apagado: código = %d, se esperaba 403", status)
	}
	requireExists(t, original)

	setAllowFileChanges(t, ts, true)
	status, body := trashRequest(t, http.MethodDelete,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("mover con el ajuste encendido: código = %d, cuerpo = %s", status, body)
	}
	moved := decodeTrash(t, body)
	setAllowFileChanges(t, ts, false)
	status, _ = libraryHTTPPost(t,
		ts.URL+"/api/v1/trash/"+strconv.FormatInt(moved.ID, 10)+"/restore", ts.token)
	if status != http.StatusForbidden {
		t.Fatalf("restaurar con el ajuste apagado: código = %d, se esperaba 403", status)
	}
	status, _ = libraryHTTPGet(t, ts.URL+"/api/v1/trash", ts.token)
	if status != http.StatusOK {
		t.Fatalf("listar con el ajuste apagado: código = %d", status)
	}
	status, _ = trashRequest(t, http.MethodDelete,
		ts.URL+"/api/v1/trash/"+strconv.FormatInt(moved.ID, 10), ts.token)
	if status != http.StatusNoContent {
		t.Fatalf("eliminar con el ajuste apagado: código = %d", status)
	}
}

func TestTrashRequiereAdministrador(t *testing.T) {
	ts := newTrashTestServer(t)
	hash, err := auth.HashPassword("Clave-segura-1")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if _, err := ts.db.Exec(
		`INSERT INTO users (username, password_hash, role) VALUES ('pepe', ?, 'user')`, hash,
	); err != nil {
		t.Fatalf("crear usuario normal: %v", err)
	}
	token, _, err := ts.auth.Login(context.Background(), "pepe", "Clave-segura-1")
	if err != nil {
		t.Fatalf("login normal: %v", err)
	}
	status, _ := libraryHTTPGet(t, ts.URL+"/api/v1/trash", token)
	if status != http.StatusForbidden {
		t.Fatalf("listar como usuario normal: código = %d, se esperaba 403", status)
	}
}

func TestTrashRechazaEnlacesSimbolicos(t *testing.T) {
	ts := newTrashTestServer(t)
	setAllowFileChanges(t, ts, true)
	root := t.TempDir()
	libraryID := addTrashLibrary(t, ts, root)
	target := filepath.Join(root, "real.mkv")
	link := filepath.Join(root, "enlace.mkv")
	writeTestFile(t, target, "real")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("no se pueden crear enlaces simbólicos: %v", err)
	}
	itemID := addTrashItem(t, ts.db, libraryID, link, "Enlace")
	status, body := trashRequest(t, http.MethodDelete,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10), ts.token)
	if status != http.StatusBadRequest {
		t.Fatalf("enlace simbólico: código = %d, cuerpo = %s", status, body)
	}
	requireExists(t, link)
	requireExists(t, target)
}
