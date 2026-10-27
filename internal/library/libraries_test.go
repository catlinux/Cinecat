package library

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func libraryJSONDo(t *testing.T, method, url, token string, body any) (int, []byte) {
	t.Helper()
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("serializar cuerpo: %v", err)
		}
		payload = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leer respuesta: %v", err)
	}
	return resp.StatusCode, data
}

func makeDirs(t *testing.T, root string, names ...string) {
	t.Helper()
	for _, name := range names {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatalf("crear carpeta %s: %v", name, err)
		}
	}
}

func createLibraryViaAPI(t *testing.T, ts *libraryTestServer, payload any) libraryDTO {
	t.Helper()
	status, body := libraryJSONDo(t, http.MethodPost, ts.URL+"/api/v1/libraries", ts.token, payload)
	if status != http.StatusCreated {
		t.Fatalf("crear biblioteca: código = %d, cuerpo = %s", status, body)
	}
	var library libraryDTO
	if err := json.Unmarshal(body, &library); err != nil {
		t.Fatalf("respuesta de creación: %v", err)
	}
	return library
}

func TestBibliotecasVariasCarpetasYCompatibilidad(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	root := t.TempDir()
	makeDirs(t, root, "pelis", "series", "antigua")
	ts.api.WithMediaRoots([]string{root})

	library := createLibraryViaAPI(t, ts, map[string]any{
		"name":  "Varias",
		"kind":  "clips",
		"paths": []string{filepath.Join(root, "pelis"), filepath.Join(root, "series")},
	})
	if library.Kind != "clips" || library.Path != filepath.Join(root, "pelis") ||
		len(library.Paths) != 2 || library.ItemCount != 0 {
		t.Fatalf("biblioteca creada inesperada: %+v", library)
	}
	if library.Paths[0].ID <= 0 || library.Paths[1].ID <= 0 {
		t.Fatalf("las carpetas deben tener id: %+v", library.Paths)
	}

	old := createLibraryViaAPI(t, ts, map[string]string{
		"name": "Antigua",
		"path": filepath.Join(root, "antigua"),
	})
	_ = old

	status, body := libraryJSONDo(t, http.MethodPost, ts.URL+"/api/v1/libraries", ts.token,
		map[string]any{"name": "Repetida", "paths": []string{
			filepath.Join(root, "pelis"), filepath.Join(root, "pelis"),
		}})
	if status != http.StatusBadRequest {
		t.Fatalf("carpetas repetidas: código = %d, cuerpo = %s", status, body)
	}
}

func TestBibliotecasValidacionDeRutasYTipos(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	root := t.TempDir()
	outside := t.TempDir()
	makeDirs(t, root, "a", "b", "a/sub")
	ts.api.WithMediaRoots([]string{root})
	file := filepath.Join(root, "archivo.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("crear archivo: %v", err)
	}

	cases := []struct {
		name    string
		payload any
		want    int
	}{
		{"sin carpetas", map[string]any{"name": "Sin", "paths": []string{}}, http.StatusBadRequest},
		{"tipo inválido", map[string]any{"name": "Tipo", "kind": "fotos", "paths": []string{filepath.Join(root, "a")}}, http.StatusBadRequest},
		{"nombre vacío", map[string]any{"name": " ", "paths": []string{filepath.Join(root, "a")}}, http.StatusBadRequest},
		{"nombre largo", map[string]any{"name": strings.Repeat("x", 81), "paths": []string{filepath.Join(root, "a")}}, http.StatusBadRequest},
		{"fuera de raíces", map[string]any{"name": "Fuera", "paths": []string{filepath.Join(outside, "a")}}, http.StatusBadRequest},
		{"inexistente", map[string]any{"name": "No", "paths": []string{filepath.Join(root, "no-existe")}}, http.StatusBadRequest},
		{"archivo", map[string]any{"name": "Archivo", "paths": []string{file}}, http.StatusBadRequest},
		{"solapada", map[string]any{"name": "Solapada", "paths": []string{filepath.Join(root, "a"), filepath.Join(root, "a", "sub")}}, http.StatusBadRequest},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			status, body := libraryJSONDo(t, http.MethodPost, ts.URL+"/api/v1/libraries",
				ts.token, test.payload)
			if status != test.want {
				t.Fatalf("código = %d, se esperaba %d; cuerpo = %s", status, test.want, body)
			}
		})
	}

	first := createLibraryViaAPI(t, ts, map[string]any{
		"name": "Primera", "paths": []string{filepath.Join(root, "a")},
	})
	status, body := libraryJSONDo(t, http.MethodPost, ts.URL+"/api/v1/libraries", ts.token,
		map[string]any{"name": "Misma ruta", "path": filepath.Join(root, "a")})
	if status != http.StatusConflict {
		t.Fatalf("ruta ya usada: código = %d, cuerpo = %s", status, body)
	}
	status, body = libraryJSONDo(t, http.MethodPost,
		ts.URL+"/api/v1/libraries/"+itoa(first.ID)+"/paths", ts.token,
		map[string]string{"path": filepath.Join(root, "b")})
	if status != http.StatusCreated {
		t.Fatalf("añadir carpeta: código = %d, cuerpo = %s", status, body)
	}
	var withPath libraryDTO
	if err := json.Unmarshal(body, &withPath); err != nil {
		t.Fatalf("respuesta al añadir carpeta: %v", err)
	}
	if len(withPath.Paths) != 2 {
		t.Fatalf("carpetas tras añadir = %d", len(withPath.Paths))
	}
	status, body = libraryJSONDo(t, http.MethodPost,
		ts.URL+"/api/v1/libraries/"+itoa(first.ID)+"/paths", ts.token,
		map[string]string{"path": filepath.Join(root, "a", "sub")})
	if status != http.StatusConflict {
		t.Fatalf("solapamiento con la misma biblioteca: código = %d, cuerpo = %s", status, body)
	}

	status, body = libraryJSONDo(t, http.MethodPatch,
		ts.URL+"/api/v1/libraries/"+itoa(first.ID), ts.token,
		map[string]string{"name": "Primera renombrada"})
	if status != http.StatusOK {
		t.Fatalf("renombrar: código = %d, cuerpo = %s", status, body)
	}
	var renamed libraryDTO
	if err := json.Unmarshal(body, &renamed); err != nil {
		t.Fatalf("respuesta de renombrar: %v", err)
	}
	if renamed.Name != "Primera renombrada" {
		t.Fatalf("nombre = %q", renamed.Name)
	}
}

func itoa(value int64) string {
	return strconv.FormatInt(value, 10)
}

func TestBibliotecaVariasCarpetasListadoYRetirada(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	root := t.TempDir()
	makeDirs(t, root, "a", "b", "b-otro")
	ts.api.WithMediaRoots([]string{root})
	library := createLibraryViaAPI(t, ts, map[string]any{
		"name": "Carpetas",
		"paths": []string{
			filepath.Join(root, "a"),
			filepath.Join(root, "b"),
			filepath.Join(root, "b-otro"),
		},
	})
	if len(library.Paths) != 3 {
		t.Fatalf("carpetas creadas = %d", len(library.Paths))
	}
	var pathB, pathBOther int64
	for _, path := range library.Paths {
		switch path.Path {
		case filepath.Join(root, "b"):
			pathB = path.ID
		case filepath.Join(root, "b-otro"):
			pathBOther = path.ID
		}
	}
	if pathB == 0 || pathBOther == 0 {
		t.Fatalf("no se encontraron las carpetas: %+v", library.Paths)
	}

	itemA := insertLibraryTestItem(t, ts, library.ID, filepath.Join(root, "a", "a.mkv"), "A", KindMovie)
	itemB := insertLibraryTestItem(t, ts, library.ID, filepath.Join(root, "b", "b.mkv"), "B", KindMovie)
	itemBOther := insertLibraryTestItem(t, ts, library.ID, filepath.Join(root, "b-otro", "otro.mkv"), "Otro", KindMovie)
	_ = itemA
	_ = itemBOther

	seriesID := insertLibraryTestSeries(t, ts, library.ID, "Serie")
	episodeID := insertLibraryTestItem(t, ts, library.ID,
		filepath.Join(root, "b", "serie", "1x01.mkv"), "Episodio", KindEpisode)
	if _, err := ts.db.Exec(`UPDATE media_items SET series_id = ?, season = 1, episode = 1 WHERE id = ?`,
		seriesID, episodeID); err != nil {
		t.Fatalf("asociar episodio: %v", err)
	}
	collectionID := insertLibraryTestCollection(t, ts, "Colección", itemB)
	anaID := progressUserID(t, ts.db, "ana")
	setWatchState(t, ts.db, anaID, itemB)
	setProgressRow(t, ts.db, anaID, itemB, 30, 100, "2026-10-26T10:00:00Z")

	status, body := libraryJSONDo(t, http.MethodGet, ts.URL+"/api/v1/libraries", ts.token, nil)
	if status != http.StatusOK {
		t.Fatalf("listar: código = %d, cuerpo = %s", status, body)
	}
	var list librariesResponse
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(list.Libraries) != 1 || list.Libraries[0].ItemCount != 4 ||
		len(list.Libraries[0].Paths) != 3 {
		t.Fatalf("listado inesperado: %+v", list.Libraries)
	}

	status, body = libraryJSONDo(t, http.MethodDelete,
		ts.URL+"/api/v1/libraries/"+itoa(library.ID)+"/paths/"+itoa(pathB), ts.token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("quitar carpeta: código = %d, cuerpo = %s", status, body)
	}
	assertLibraryItemCount(t, ts, "SELECT count(*) FROM media_items WHERE path = ?", filepath.Join(root, "b", "b.mkv"), 0)
	assertLibraryItemCount(t, ts, "SELECT count(*) FROM media_items WHERE path = ?", filepath.Join(root, "b", "serie", "1x01.mkv"), 0)
	assertLibraryItemCount(t, ts, "SELECT count(*) FROM media_items WHERE path = ?", filepath.Join(root, "b-otro", "otro.mkv"), 1)
	if count := countRows(t, ts, `SELECT count(*) FROM series WHERE id = ?`, seriesID); count != 0 {
		t.Fatalf("la serie huérfana debe borrarse: %d", count)
	}
	if count := countRows(t, ts, `SELECT count(*) FROM collections WHERE id = ?`, collectionID); count != 0 {
		t.Fatalf("la colección vacía debe borrarse: %d", count)
	}
	if count := countRows(t, ts, `SELECT count(*) FROM watch_state WHERE user_id = ? AND media_item_id = ?`, anaID, itemB); count != 0 {
		t.Fatalf("el visto del ítem borrado debe desaparecer: %d", count)
	}
	if count := countRows(t, ts, `SELECT count(*) FROM playback_progress WHERE user_id = ? AND media_item_id = ?`, anaID, itemB); count != 0 {
		t.Fatalf("el progreso del ítem borrado debe desaparecer: %d", count)
	}

	status, body = libraryJSONDo(t, http.MethodDelete,
		ts.URL+"/api/v1/libraries/"+itoa(library.ID)+"/paths/"+itoa(library.Paths[0].ID), ts.token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("quitar carpeta principal: código = %d, cuerpo = %s", status, body)
	}
	library2, err := getLibraryDTO(t, ts, library.ID)
	if err != nil {
		t.Fatalf("leer biblioteca tras retirar principal: %v", err)
	}
	if library2.Path != filepath.Join(root, "b-otro") {
		t.Fatalf("carpeta principal = %q", library2.Path)
	}
	status, body = libraryJSONDo(t, http.MethodDelete,
		ts.URL+"/api/v1/libraries/"+itoa(library.ID)+"/paths/"+itoa(pathBOther), ts.token, nil)
	if status != http.StatusBadRequest {
		t.Fatalf("la última carpeta debe rechazarse: código = %d, cuerpo = %s", status, body)
	}
}

func TestBibliotecasPermisos(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	root := t.TempDir()
	makeDirs(t, root, "a", "b")
	ts.api.WithMediaRoots([]string{root})
	library := createLibraryViaAPI(t, ts, map[string]any{
		"name": "Permisos", "paths": []string{filepath.Join(root, "a")},
	})
	userToken := nonAdminToken(t, ts)

	if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/libraries", userToken); status != http.StatusOK {
		t.Fatalf("listar como usuario: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/libraries", ""); status != http.StatusUnauthorized {
		t.Fatalf("listar sin sesión: código = %d, cuerpo = %s", status, body)
	}
	path := "/api/v1/libraries/" + itoa(library.ID)
	cases := []struct {
		method string
		url    string
		body   any
	}{
		{http.MethodPost, ts.URL + "/api/v1/libraries", map[string]any{"name": "No", "paths": []string{filepath.Join(root, "b")}}},
		{http.MethodPatch, ts.URL + path, map[string]string{"name": "No"}},
		{http.MethodPost, ts.URL + path + "/paths", map[string]string{"path": filepath.Join(root, "b")}},
		{http.MethodDelete, ts.URL + path + "/paths/1", nil},
	}
	for _, test := range cases {
		status, body := libraryJSONDo(t, test.method, test.url, userToken, test.body)
		if status != http.StatusForbidden {
			t.Fatalf("%s %s: código = %d, cuerpo = %s", test.method, test.url, status, body)
		}
	}
}

func insertLibraryTestItem(t *testing.T, ts *libraryTestServer, libraryID int64, path, title, kind string) int64 {
	t.Helper()
	result, err := ts.db.Exec(`INSERT INTO media_items (library_id, path, title, kind) VALUES (?, ?, ?, ?)`,
		libraryID, path, title, kind)
	if err != nil {
		t.Fatalf("insertar ítem %s: %v", path, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id del ítem %s: %v", path, err)
	}
	return id
}

func insertLibraryTestSeries(t *testing.T, ts *libraryTestServer, libraryID int64, folder string) int64 {
	t.Helper()
	result, err := ts.db.Exec(`INSERT INTO series (library_id, folder, title) VALUES (?, ?, ?)`,
		libraryID, folder, folder)
	if err != nil {
		t.Fatalf("insertar serie: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de serie: %v", err)
	}
	return id
}

func insertLibraryTestCollection(t *testing.T, ts *libraryTestServer, name string, itemID int64) int64 {
	t.Helper()
	result, err := ts.db.Exec(`
		INSERT INTO collections (name, source, tmdb_collection_id) VALUES (?, 'tmdb', 9001)`, name)
	if err != nil {
		t.Fatalf("insertar colección: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de colección: %v", err)
	}
	if _, err := ts.db.Exec(`INSERT INTO collection_items (collection_id, media_item_id) VALUES (?, ?)`,
		id, itemID); err != nil {
		t.Fatalf("insertar miembro de colección: %v", err)
	}
	return id
}

func assertLibraryItemCount(t *testing.T, ts *libraryTestServer, query string, arg any, want int) {
	t.Helper()
	if got := countRows(t, ts, query, arg); got != want {
		t.Fatalf("consulta %q con %v: %d, se esperaba %d", query, arg, got, want)
	}
}

func countRows(t *testing.T, ts *libraryTestServer, query string, args ...any) int {
	t.Helper()
	var count int
	if err := ts.db.QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatalf("contar filas: %v", err)
	}
	return count
}

func getLibraryDTO(t *testing.T, ts *libraryTestServer, id int64) (libraryDTO, error) {
	t.Helper()
	status, body := libraryJSONDo(t, http.MethodGet, ts.URL+"/api/v1/libraries", ts.token, nil)
	if status != http.StatusOK {
		t.Fatalf("listar: código = %d, cuerpo = %s", status, body)
	}
	var list librariesResponse
	if err := json.Unmarshal(body, &list); err != nil {
		return libraryDTO{}, err
	}
	for _, item := range list.Libraries {
		if item.ID == id {
			return item, nil
		}
	}
	return libraryDTO{}, os.ErrNotExist
}
