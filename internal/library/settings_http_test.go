package library

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/catlinux/cinecat/internal/settings"
)

type settingsTestServer struct {
	*libraryTestServer
	url   string
	store *settings.Store
}

func newSettingsHTTPTestServer(t *testing.T) *settingsTestServer {
	t.Helper()
	base := newLibraryHTTPTestServer(t)
	store := settings.NewStore(base.db, nil)
	if err := store.Load(context.Background()); err != nil {
		t.Fatalf("settings.Load: %v", err)
	}
	api := NewAPI(base.db, base.auth, nil, base.job, nil).WithSettings(store)
	mux := http.NewServeMux()
	api.Register(mux)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &settingsTestServer{libraryTestServer: base, url: server.URL, store: store}
}

func settingsJSON(t *testing.T, body []byte) settingsResponse {
	t.Helper()
	var response settingsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("respuesta de ajustes: %v", err)
	}
	return response
}

func settingsFind(t *testing.T, response settingsResponse, key string) settingDTO {
	t.Helper()
	for _, section := range response.Sections {
		for _, setting := range section.Settings {
			if setting.Key == key {
				return setting
			}
		}
	}
	t.Fatalf("no aparece el ajuste %q", key)
	return settingDTO{}
}

func TestSettingsGetEstructuraYOrden(t *testing.T) {
	ts := newSettingsHTTPTestServer(t)

	status, body := libraryHTTPGet(t, ts.url+"/api/v1/settings", ts.token)
	if status != http.StatusOK {
		t.Fatalf("GET ajustes: código = %d, cuerpo = %s", status, body)
	}
	response := settingsJSON(t, body)
	if len(response.Sections) != 6 {
		t.Fatalf("secciones = %d, se esperaban 6", len(response.Sections))
	}
	labels := make([]string, 0, len(response.Sections))
	for _, section := range response.Sections {
		labels = append(labels, section.Label)
	}
	if !reflect.DeepEqual(labels, []string{"Escaneo", "Metadatos", "Interfaz", "Reproducción", "Biblioteca", "Sistema"}) {
		t.Fatalf("etiquetas de sección inesperadas: %v", labels)
	}

	first := response.Sections[0].Settings[0]
	if first.Key != settings.KeyScanInterval || first.Type != settings.TypeDuration ||
		first.Default != "6h" || first.Value != "6h" || first.Modified ||
		first.Min != 60 || first.Max != 0 {
		t.Fatalf("scan.interval inesperado: %+v", first)
	}
	if first.Options == nil {
		t.Fatal("options debe ser un array, no null")
	}

	gridColumns := settingsFind(t, response, "ui.grid_columns")
	if gridColumns.Default.(float64) != 6 || gridColumns.Value.(float64) != 6 ||
		gridColumns.Modified || gridColumns.Min != 4 || gridColumns.Max != 8 {
		t.Fatalf("ui.grid_columns inesperado: %+v", gridColumns)
	}

	if status, body := libraryHTTPGet(t, ts.url+"/api/v1/settings", ""); status != http.StatusUnauthorized {
		t.Fatalf("sin sesión: código = %d, cuerpo = %s", status, body)
	}
	userToken := nonAdminToken(t, ts.libraryTestServer)
	if status, body := libraryHTTPGet(t, ts.url+"/api/v1/settings", userToken); status != http.StatusForbidden {
		t.Fatalf("sin admin: código = %d, cuerpo = %s", status, body)
	}
}

func TestSettingsPATCH(t *testing.T) {
	ts := newSettingsHTTPTestServer(t)

	status, body := libraryHTTPDo(t, http.MethodPatch, ts.url+"/api/v1/settings", ts.token,
		map[string]any{
			"ui.grid_columns":     4,
			"scan.interval":       "45m",
			"ui.show_collections": false,
		})
	if status != http.StatusOK {
		t.Fatalf("PATCH válido: código = %d, cuerpo = %s", status, body)
	}
	response := settingsJSON(t, body)
	gridColumns := settingsFind(t, response, "ui.grid_columns")
	if gridColumns.Value.(float64) != 4 || !gridColumns.Modified {
		t.Fatalf("ui.grid_columns no actualizado: %+v", gridColumns)
	}
	interval := settingsFind(t, response, settings.KeyScanInterval)
	if interval.Value != "45m" || !interval.Modified {
		t.Fatalf("scan.interval no actualizado: %+v", interval)
	}
	if got := ts.store.Int("ui.grid_columns"); got != 4 {
		t.Fatalf("caché de ui.grid_columns = %d", got)
	}
	if got := ts.store.Duration(settings.KeyScanInterval); got.String() != "45m0s" {
		t.Fatalf("caché de scan.interval = %v", got)
	}

	status, body = libraryHTTPDo(t, http.MethodPatch, ts.url+"/api/v1/settings", ts.token,
		map[string]any{
			"ui.grid_columns": 9,
			"playback.crf":    99,
			"ui.show_series":  "no",
		})
	if status != http.StatusBadRequest {
		t.Fatalf("PATCH inválido: código = %d, cuerpo = %s", status, body)
	}
	var failure settingsErrorResponse
	if err := json.Unmarshal(body, &failure); err != nil {
		t.Fatalf("respuesta de error: %v", err)
	}
	if len(failure.Fields) != 3 || failure.Fields["ui.grid_columns"] == "" ||
		failure.Fields["playback.crf"] == "" || failure.Fields["ui.show_series"] == "" {
		t.Fatalf("fields inesperados: %+v", failure.Fields)
	}
	if ts.store.Int("ui.grid_columns") != 4 {
		t.Fatal("un PATCH inválido no debe guardar ninguna clave")
	}

	status, body = libraryHTTPDo(t, http.MethodPatch, ts.url+"/api/v1/settings", ts.token,
		map[string]any{"clave.desconocida": true})
	if status != http.StatusBadRequest {
		t.Fatalf("PATCH desconocido: código = %d, cuerpo = %s", status, body)
	}
	if err := json.Unmarshal(body, &failure); err != nil {
		t.Fatalf("respuesta de clave desconocida: %v", err)
	}
	if failure.Fields["clave.desconocida"] == "" {
		t.Fatalf("falta el campo de la clave desconocida: %+v", failure)
	}

	status, body = libraryHTTPDo(t, http.MethodPatch, ts.url+"/api/v1/settings", ts.token, map[string]any{})
	if status != http.StatusOK {
		t.Fatalf("PATCH vacío: código = %d, cuerpo = %s", status, body)
	}

	status, body = libraryHTTPDo(t, http.MethodPatch, ts.url+"/api/v1/settings", ts.token, nil)
	if status != http.StatusBadRequest {
		t.Fatalf("PATCH sin cuerpo: código = %d, cuerpo = %s", status, body)
	}

	userToken := nonAdminToken(t, ts.libraryTestServer)
	if status, body := libraryHTTPDo(t, http.MethodPatch, ts.url+"/api/v1/settings", userToken,
		map[string]any{"ui.grid_columns": 4}); status != http.StatusForbidden {
		t.Fatalf("PATCH sin admin: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodPatch, ts.url+"/api/v1/settings", "",
		map[string]any{"ui.grid_columns": 4}); status != http.StatusUnauthorized {
		t.Fatalf("PATCH sin sesión: código = %d, cuerpo = %s", status, body)
	}
}

func TestSettingsDeleteYClaveConPuntos(t *testing.T) {
	ts := newSettingsHTTPTestServer(t)
	if failures := ts.store.Set(context.Background(), map[string]json.RawMessage{
		"ui.grid_columns": json.RawMessage("4"),
	}); len(failures) != 0 {
		t.Fatalf("preparar ajuste: %v", failures)
	}

	status, body := libraryHTTPDo(t, http.MethodDelete,
		ts.url+"/api/v1/settings/ui.grid_columns", ts.token, nil)
	if status != http.StatusOK {
		t.Fatalf("DELETE: código = %d, cuerpo = %s", status, body)
	}
	var setting settingDTO
	if err := json.Unmarshal(body, &setting); err != nil {
		t.Fatalf("respuesta de DELETE: %v", err)
	}
	if setting.Key != "ui.grid_columns" || setting.Value.(float64) != 6 || setting.Modified {
		t.Fatalf("ajuste restablecido inesperado: %+v", setting)
	}
	if ts.store.Int("ui.grid_columns") != 6 || ts.store.IsModified("ui.grid_columns") {
		t.Fatal("la caché no se restableció")
	}

	if status, body := libraryHTTPDo(t, http.MethodDelete,
		ts.url+"/api/v1/settings/no.existe", ts.token, nil); status != http.StatusNotFound {
		t.Fatalf("DELETE desconocido: código = %d, cuerpo = %s", status, body)
	}

	userToken := nonAdminToken(t, ts.libraryTestServer)
	if status, body := libraryHTTPDo(t, http.MethodDelete,
		ts.url+"/api/v1/settings/ui.grid_columns", userToken, nil); status != http.StatusForbidden {
		t.Fatalf("DELETE sin admin: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodDelete,
		ts.url+"/api/v1/settings/ui.grid_columns", "", nil); status != http.StatusUnauthorized {
		t.Fatalf("DELETE sin sesión: código = %d, cuerpo = %s", status, body)
	}
}

func TestSettingsPublic(t *testing.T) {
	ts := newSettingsHTTPTestServer(t)
	userToken := nonAdminToken(t, ts.libraryTestServer)

	status, body := libraryHTTPGet(t, ts.url+"/api/v1/settings/public", userToken)
	if status != http.StatusOK {
		t.Fatalf("GET público: código = %d, cuerpo = %s", status, body)
	}
	var values map[string]any
	if err := json.Unmarshal(body, &values); err != nil {
		t.Fatalf("respuesta pública: %v", err)
	}
	if len(values) != 14 {
		t.Fatalf("ajustes públicos = %d, se esperaban 14: %v", len(values), values)
	}
	defaults := map[string]any{
		"ui.server_name":      "Cinecat",
		"ui.tagline":          "Sesión continua",
		"ui.portal_title":     "Tu cine en casa",
		"ui.portal_intro":     "Una biblioteca personal con olor a palomitas, carteles pintados a mano y la sesión de siempre esperándote.",
		"ui.nav_layout":       "side",
		"ui.show_collections": true,
		"ui.show_series":      true,
		"ui.theme_default":    "system",
		"ui.accent":           "granate",
		"ui.grid_columns":     float64(6),
		"ui.grid_rows":        float64(8),
		"ui.row_columns":      float64(6),
	}
	for key, want := range defaults {
		if values[key] != want {
			t.Fatalf("%s = %v, se esperaba %v", key, values[key], want)
		}
	}
	if _, ok := values[settings.KeyScanInterval]; ok {
		t.Fatal("scan.interval no es público")
	}

	if status, body := libraryHTTPGet(t, ts.url+"/api/v1/settings/public", ""); status != http.StatusUnauthorized {
		t.Fatalf("público sin sesión: código = %d, cuerpo = %s", status, body)
	}
}
