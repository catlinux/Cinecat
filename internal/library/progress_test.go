package library

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
)

func progressUserID(t *testing.T, db *sql.DB, username string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(`SELECT id FROM users WHERE username = ?`, username).Scan(&id); err != nil {
		t.Fatalf("id del usuario %q: %v", username, err)
	}
	return id
}

func progressItem(t *testing.T, ts *libraryTestServer, libraryID int64, title, kind string) int64 {
	t.Helper()
	return insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      filepath.Join(t.TempDir(), title+".mkv"),
		title:     title,
		kind:      kind,
	})
}

func progressURL(ts *libraryTestServer, id int64, suffix string) string {
	return ts.URL + "/api/v1/items/" + strconv.FormatInt(id, 10) + "/progress" + suffix
}

func readProgress(t *testing.T, db *sql.DB, userID, itemID int64) (position, duration float64, updatedAt string, ok bool) {
	t.Helper()
	err := db.QueryRow(`
		SELECT position_seconds, duration_seconds, updated_at
		  FROM playback_progress
		 WHERE user_id = ? AND media_item_id = ?`, userID, itemID).
		Scan(&position, &duration, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, "", false
	}
	if err != nil {
		t.Fatalf("leer progreso: %v", err)
	}
	return position, duration, updatedAt, true
}

func hasWatchState(t *testing.T, db *sql.DB, userID, itemID int64) bool {
	t.Helper()
	var exists int
	err := db.QueryRow(`SELECT 1 FROM watch_state WHERE user_id = ? AND media_item_id = ?`,
		userID, itemID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}
	if err != nil {
		t.Fatalf("leer estado de visionado: %v", err)
	}
	return true
}

func setProgressRow(t *testing.T, db *sql.DB, userID, itemID int64, position, duration float64, updatedAt string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO playback_progress (
			user_id, media_item_id, position_seconds, duration_seconds, updated_at
		) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (user_id, media_item_id) DO UPDATE SET
			position_seconds = excluded.position_seconds,
			duration_seconds = excluded.duration_seconds,
			updated_at = excluded.updated_at`,
		userID, itemID, position, duration, updatedAt); err != nil {
		t.Fatalf("guardar progreso de prueba: %v", err)
	}
}

func setWatchState(t *testing.T, db *sql.DB, userID, itemID int64) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT OR IGNORE INTO watch_state (user_id, media_item_id) VALUES (?, ?)`,
		userID, itemID); err != nil {
		t.Fatalf("guardar visionado de prueba: %v", err)
	}
}

func TestProgressGuardaYActualiza(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	itemID := progressItem(t, ts, libraryID, "Pelicula de progreso", KindMovie)
	userID := progressUserID(t, ts.db, "ana")

	status, body := libraryHTTPDo(t, http.MethodPut, progressURL(ts, itemID, ""), ts.token,
		map[string]float64{"position_seconds": 120, "duration_seconds": 600})
	if status != http.StatusNoContent {
		t.Fatalf("guardar progreso: código = %d, cuerpo = %s", status, body)
	}
	position, duration, updatedAt, ok := readProgress(t, ts.db, userID, itemID)
	if !ok || position != 120 || duration != 600 || updatedAt == "" {
		t.Fatalf("progreso guardado inesperado: %v %v %q %v", position, duration, updatedAt, ok)
	}

	if _, err := ts.db.Exec(
		`UPDATE playback_progress SET updated_at = '2000-01-01T00:00:00Z'
		 WHERE user_id = ? AND media_item_id = ?`, userID, itemID); err != nil {
		t.Fatalf("retrasar marca de tiempo: %v", err)
	}
	status, body = libraryHTTPDo(t, http.MethodPut, progressURL(ts, itemID, ""), ts.token,
		map[string]float64{"position_seconds": 240, "duration_seconds": 600})
	if status != http.StatusNoContent {
		t.Fatalf("actualizar progreso: código = %d, cuerpo = %s", status, body)
	}
	position, duration, updatedAt, ok = readProgress(t, ts.db, userID, itemID)
	if !ok || position != 240 || duration != 600 ||
		updatedAt == "2000-01-01T00:00:00Z" || updatedAt == "" {
		t.Fatalf("progreso actualizado inesperado: %v %v %q %v", position, duration, updatedAt, ok)
	}
	if hasWatchState(t, ts.db, userID, itemID) {
		t.Fatal("guardar progreso normal no debe marcar la película como vista")
	}
}

func TestProgressMarcaVistoAlLlegarAlNoventaPorCiento(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	itemID := progressItem(t, ts, libraryID, "Final de pelicula", KindMovie)
	userID := progressUserID(t, ts.db, "ana")

	status, body := libraryHTTPDo(t, http.MethodPut, progressURL(ts, itemID, ""), ts.token,
		map[string]float64{"position_seconds": 100, "duration_seconds": 600})
	if status != http.StatusNoContent {
		t.Fatalf("progreso previo: código = %d, cuerpo = %s", status, body)
	}
	status, body = libraryHTTPDo(t, http.MethodPut, progressURL(ts, itemID, ""), ts.token,
		map[string]float64{"position_seconds": 540, "duration_seconds": 600})
	if status != http.StatusNoContent {
		t.Fatalf("progreso al 90%%: código = %d, cuerpo = %s", status, body)
	}
	if _, _, _, ok := readProgress(t, ts.db, userID, itemID); ok {
		t.Fatal("al marcar visto debe borrarse el progreso")
	}
	if !hasWatchState(t, ts.db, userID, itemID) {
		t.Fatal("al llegar al 90 % debe marcarse como visto")
	}
}

func TestProgressMenosDeDiezSegundosBorra(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	itemID := progressItem(t, ts, libraryID, "Inicio de pelicula", KindMovie)
	userID := progressUserID(t, ts.db, "ana")

	if status, body := libraryHTTPDo(t, http.MethodPut, progressURL(ts, itemID, ""), ts.token,
		map[string]float64{"position_seconds": 120, "duration_seconds": 600}); status != http.StatusNoContent {
		t.Fatalf("progreso previo: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodPut, progressURL(ts, itemID, ""), ts.token,
		map[string]float64{"position_seconds": 9.5, "duration_seconds": 600}); status != http.StatusNoContent {
		t.Fatalf("progreso inicial: código = %d, cuerpo = %s", status, body)
	}
	if _, _, _, ok := readProgress(t, ts.db, userID, itemID); ok {
		t.Fatal("un progreso de menos de 10 s no debe guardarse")
	}
	if hasWatchState(t, ts.db, userID, itemID) {
		t.Fatal("un progreso inicial no debe marcar como visto")
	}
}

func TestProgressValidaciones(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	itemID := progressItem(t, ts, libraryID, "Pelicula de validacion", KindMovie)

	for name, body := range map[string]any{
		"sin campos":           map[string]any{},
		"sin duración":         map[string]any{"position_seconds": 30},
		"posición negativa":    map[string]any{"position_seconds": -1, "duration_seconds": 10},
		"duración negativa":    map[string]any{"position_seconds": 1, "duration_seconds": -1},
		"posición no numérica": map[string]any{"position_seconds": "30", "duration_seconds": 10},
	} {
		t.Run(name, func(t *testing.T) {
			status, response := libraryHTTPDo(t, http.MethodPut, progressURL(ts, itemID, ""), ts.token, body)
			if status != http.StatusBadRequest {
				t.Fatalf("código = %d, cuerpo = %s", status, response)
			}
		})
	}

	status, body := libraryHTTPDo(t, http.MethodPut, progressURL(ts, 999999, ""), ts.token,
		map[string]float64{"position_seconds": 30, "duration_seconds": 100})
	if status != http.StatusNotFound {
		t.Fatalf("ítem inexistente: código = %d, cuerpo = %s", status, body)
	}

	status, body = libraryHTTPDo(t, http.MethodPut, progressURL(ts, itemID, ""), "",
		map[string]float64{"position_seconds": 30, "duration_seconds": 100})
	if status != http.StatusUnauthorized {
		t.Fatalf("sin sesión: código = %d, cuerpo = %s", status, body)
	}
}

func TestProgressDeleteEsIdempotente(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	itemID := progressItem(t, ts, libraryID, "Pelicula para borrar", KindMovie)
	userID := progressUserID(t, ts.db, "ana")
	setProgressRow(t, ts.db, userID, itemID, 60, 600, "2026-10-26T10:00:00Z")

	for i := 0; i < 2; i++ {
		status, body := libraryHTTPDo(t, http.MethodDelete, progressURL(ts, itemID, ""), ts.token, nil)
		if status != http.StatusNoContent {
			t.Fatalf("DELETE %d: código = %d, cuerpo = %s", i+1, status, body)
		}
	}
	if _, _, _, ok := readProgress(t, ts.db, userID, itemID); ok {
		t.Fatal("el progreso debía quedar borrado")
	}
}

func TestWatchedBorraProgreso(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	itemID := progressItem(t, ts, libraryID, "Pelicula vista", KindMovie)
	userID := progressUserID(t, ts.db, "ana")

	setProgressRow(t, ts.db, userID, itemID, 120, 600, "2026-10-26T10:00:00Z")
	status, body := libraryHTTPDo(t, http.MethodPut,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10)+"/watched", ts.token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("marcar visto: código = %d, cuerpo = %s", status, body)
	}
	if _, _, _, ok := readProgress(t, ts.db, userID, itemID); ok {
		t.Fatal("marcar como visto debe borrar el progreso")
	}
	if !hasWatchState(t, ts.db, userID, itemID) {
		t.Fatal("el ítem debía quedar visto")
	}

	setProgressRow(t, ts.db, userID, itemID, 180, 600, "2026-10-26T10:01:00Z")
	status, body = libraryHTTPDo(t, http.MethodDelete,
		ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10)+"/watched", ts.token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("desmarcar visto: código = %d, cuerpo = %s", status, body)
	}
	if _, _, _, ok := readProgress(t, ts.db, userID, itemID); ok {
		t.Fatal("desmarcar como visto debe borrar el progreso")
	}
	if hasWatchState(t, ts.db, userID, itemID) {
		t.Fatal("el ítem no debía quedar visto")
	}
}

func TestProgressDTOYAislamientoEntreUsuarios(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	itemID := progressItem(t, ts, libraryID, "Pelicula aislada", KindMovie)
	userID := progressUserID(t, ts.db, "ana")
	beaToken := nonAdminToken(t, ts)

	if status, body := libraryHTTPDo(t, http.MethodPut, progressURL(ts, itemID, ""), ts.token,
		map[string]float64{"position_seconds": 300, "duration_seconds": 1200}); status != http.StatusNoContent {
		t.Fatalf("guardar progreso: código = %d, cuerpo = %s", status, body)
	}

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("ficha de ana: código = %d, cuerpo = %s", status, body)
	}
	var anaItem mediaItemDTO
	if err := json.Unmarshal(body, &anaItem); err != nil {
		t.Fatalf("ficha de ana: %v", err)
	}
	if anaItem.Progress == nil || anaItem.Progress.PositionSeconds != 300 ||
		anaItem.Progress.DurationSeconds != 1200 || anaItem.Progress.UpdatedAt == "" {
		t.Fatalf("progreso en la ficha de ana: %+v", anaItem.Progress)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10), beaToken)
	if status != http.StatusOK {
		t.Fatalf("ficha de bea: código = %d, cuerpo = %s", status, body)
	}
	var beaItem mediaItemDTO
	if err := json.Unmarshal(body, &beaItem); err != nil {
		t.Fatalf("ficha de bea: %v", err)
	}
	if beaItem.Progress != nil {
		t.Fatalf("bea no debe ver el progreso de ana: %+v", beaItem.Progress)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items?kind=movie&limit=200", ts.token)
	if status != http.StatusOK {
		t.Fatalf("lista de ana: código = %d, cuerpo = %s", status, body)
	}
	var list itemsResponse
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("lista de ana: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].Progress == nil {
		t.Fatalf("la lista debe incluir el progreso: %+v", list.Items)
	}
	if read, _, _, ok := readProgress(t, ts.db, userID, itemID); !ok || read != 300 {
		t.Fatalf("progreso en base = %v, %v", read, ok)
	}
}

func TestProgressFiltroInProgress(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	withProgress := progressItem(t, ts, libraryID, "Con progreso", KindMovie)
	withoutProgress := progressItem(t, ts, libraryID, "Sin progreso", KindMovie)
	userID := progressUserID(t, ts.db, "ana")
	setProgressRow(t, ts.db, userID, withProgress, 60, 600, "2026-10-26T10:00:00Z")

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items?in_progress=true&limit=200", ts.token)
	if status != http.StatusOK {
		t.Fatalf("filtro activo: código = %d, cuerpo = %s", status, body)
	}
	var list itemsResponse
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("filtro activo: %v", err)
	}
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].ID != withProgress ||
		list.Items[0].Progress == nil {
		t.Fatalf("resultado con progreso inesperado: %+v", list)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items?in_progress=false&limit=200", ts.token)
	if status != http.StatusOK {
		t.Fatalf("filtro sin progreso: código = %d, cuerpo = %s", status, body)
	}
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("filtro sin progreso: %v", err)
	}
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].ID != withoutProgress {
		t.Fatalf("resultado sin progreso inesperado: %+v", list)
	}

	if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items?in_progress=1", ts.token); status != http.StatusBadRequest {
		t.Fatalf("filtro inválido: código = %d, cuerpo = %s", status, body)
	}
}
