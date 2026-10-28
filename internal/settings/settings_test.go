package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	dbstore "github.com/catlinux/cinecat/internal/store"
)

func openSettingsDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	st, err := dbstore.Open(ctx, ":memory:")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close: %v", err)
		}
	})
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("store.Migrate: %v", err)
	}
	return st.DB()
}

func newTestStore(t *testing.T, defaults map[string]any) (*Store, context.Context) {
	t.Helper()
	ctx := context.Background()
	store := NewStore(openSettingsDB(t), defaults)
	if err := store.Load(ctx); err != nil {
		t.Fatalf("Load: %v", err)
	}
	return store, ctx
}

func raw(value string) json.RawMessage {
	return json.RawMessage(value)
}

func TestDefinitions(t *testing.T) {
	definitions := Definitions()
	if len(definitions) != 27 {
		t.Fatalf("ajustes = %d, se esperaban 27", len(definitions))
	}

	seen := make(map[string]bool)
	public := 0
	for _, definition := range definitions {
		if seen[definition.Key] {
			t.Fatalf("clave duplicada: %s", definition.Key)
		}
		seen[definition.Key] = true
		if definition.Section == "" || definition.Label == "" || definition.Description == "" {
			t.Fatalf("metadatos incompletos en %q: %+v", definition.Key, definition)
		}
		if definition.Public {
			public++
		}
		defaultJSON, err := json.Marshal(definition.Default)
		if err != nil {
			t.Fatalf("serializar el defecto de %q: %v", definition.Key, err)
		}
		normalized, err := Validate(definition.Key, defaultJSON)
		if err != nil {
			t.Fatalf("el defecto de %q no es válido: %v", definition.Key, err)
		}
		if !reflect.DeepEqual(normalized, definition.Default) {
			t.Fatalf("el defecto de %q se normaliza a %#v", definition.Key, normalized)
		}
	}
	if public != 14 {
		t.Fatalf("ajustes públicos = %d, se esperaban 14", public)
	}

	sectionKeys := make([]string, 0, len(Sections()))
	for _, section := range Sections() {
		sectionKeys = append(sectionKeys, section.Key)
	}
	if !reflect.DeepEqual(sectionKeys, []string{"scan", "metadata", "ui", "playback", "library", "system"}) {
		t.Fatalf("secciones inesperadas: %v", sectionKeys)
	}
	if lookup, ok := Lookup(KeyScanInterval); !ok || lookup.Default != "6h" {
		t.Fatalf("scan.interval inesperado: %+v, %v", lookup, ok)
	}
	if _, ok := Lookup("ui.page_size"); ok {
		t.Fatal("ui.page_size ya no debe existir")
	}

	uiDefaults := map[string]any{
		"ui.server_name":  "Cinecat",
		"ui.tagline":      "Sesión continua",
		"ui.portal_title": "Tu cine en casa",
		"ui.portal_intro": "Una biblioteca personal con olor a palomitas, carteles pintados a mano y la sesión de siempre esperándote.",
		"ui.nav_layout":   "side",
		"ui.grid_columns": 6,
		"ui.grid_rows":    8,
		"ui.row_columns":  6,
	}
	for key, want := range uiDefaults {
		definition, ok := Lookup(key)
		if !ok {
			t.Fatalf("falta el ajuste %q", key)
		}
		if definition.Default != want || !definition.Public || definition.Section != "ui" {
			t.Fatalf("%s inesperado: %+v", key, definition)
		}
	}
	if layout, ok := Lookup("ui.nav_layout"); !ok ||
		len(layout.Options) != 2 ||
		layout.Options[0].Value != "side" ||
		layout.Options[0].Label != "Lateral (izquierda)" ||
		layout.Options[1].Value != "top" ||
		layout.Options[1].Label != "Superior" {
		t.Fatalf("opciones de ui.nav_layout inesperadas: %+v, %v", layout, ok)
	}
	hardware, ok := Lookup(KeyPlaybackHardware)
	if !ok || hardware.Default != "auto" || len(hardware.Options) != 2 ||
		hardware.Options[0].Value != "auto" || hardware.Options[0].Label != "Automática" ||
		hardware.Options[1].Value != "off" || hardware.Options[1].Label != "Desactivada" {
		t.Fatalf("playback.hardware inesperado: %+v, %v", hardware, ok)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		key  string
		raw  string
		want any
	}{
		{"bool", "ui.show_collections", "false", false},
		{"int mínimo", "ui.collection_min_items", "1", 1},
		{"int máximo", "ui.collection_min_items", "10", 10},
		{"float mínimo", "metadata.match_threshold", "0.6", 0.6},
		{"float máximo", "metadata.match_threshold", "0.95", 0.95},
		{"texto recortado", "ui.server_name", `"  Mi cine  "`, "Mi cine"},
		{"eslogan vacío", "ui.tagline", `"   "`, ""},
		{"eslogan máximo", "ui.tagline", `"` + strings.Repeat("a", 60) + `"`, strings.Repeat("a", 60)},
		{"título del portal", "ui.portal_title", `"  Mi casa  "`, "Mi casa"},
		{"título del portal máximo", "ui.portal_title", `"` + strings.Repeat("a", 60) + `"`, strings.Repeat("a", 60)},
		{"intro vacía", "ui.portal_intro", `""`, ""},
		{"intro máxima", "ui.portal_intro", `"` + strings.Repeat("a", 240) + `"`, strings.Repeat("a", 240)},
		{"menú lateral", "ui.nav_layout", `"side"`, "side"},
		{"menú superior", "ui.nav_layout", `"top"`, "top"},
		{"columnas mínimo", "ui.grid_columns", "4", 4},
		{"columnas máximo", "ui.grid_columns", "8", 8},
		{"filas mínimo", "ui.grid_rows", "2", 2},
		{"filas máximo", "ui.grid_rows", "12", 12},
		{"carátulas por fila mínimo", "ui.row_columns", "3", 3},
		{"carátulas por fila máximo", "ui.row_columns", "8", 8},
		{"duración cero", KeyScanInterval, `"0"`, "0"},
		{"duración minutos", KeyScanInterval, `"30m"`, "30m"},
		{"duración horas", KeyScanInterval, `"6h"`, "6h"},
		{"duración compuesta", KeyScanInterval, `"1h30m"`, "1h30m"},
		{"enum texto", "metadata.language", `"es-MX"`, "es-MX"},
		{"entero de rejilla", "ui.grid_columns", "7", 7},
		{"enum numérico alto", "playback.max_height", "1080", 1080},
		{"enum numérico mínimo", "playback.max_height", "480", 480},
		{"transcodificaciones mínimo", "playback.max_transcodes", "1", 1},
		{"transcodificaciones máximo", "playback.max_transcodes", "8", 8},
		{"hardware automático", KeyPlaybackHardware, `"auto"`, "auto"},
		{"hardware desactivado", KeyPlaybackHardware, `"off"`, "off"},
		{"cambiar archivos", KeyAllowFileChanges, "true", true},
		{"subida mínima", KeyUploadMaxGB, "1", 1},
		{"subida máxima", KeyUploadMaxGB, "500", 500},
		{"retención mínima", KeyTrashRetention, "1", 1},
		{"retención máxima", KeyTrashRetention, "90", 90},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Validate(test.key, raw(test.raw))
			if err != nil {
				t.Fatalf("Validate: %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("valor = %#v, se esperaba %#v", got, test.want)
			}
		})
	}

	invalid := []struct {
		name string
		key  string
		raw  string
	}{
		{"bool texto", "ui.show_collections", `"true"`},
		{"int fuera por abajo", "ui.collection_min_items", "0"},
		{"int fuera por arriba", "ui.collection_min_items", "11"},
		{"int decimal", "ui.collection_min_items", "2.5"},
		{"float bajo", "metadata.match_threshold", "0.59"},
		{"float alto", "metadata.match_threshold", "0.96"},
		{"texto vacío", "ui.server_name", `"   "`},
		{"texto largo", "ui.server_name", `"` + strings.Repeat("a", 41) + `"`},
		{"eslogan largo", "ui.tagline", `"` + strings.Repeat("a", 61) + `"`},
		{"título del portal vacío", "ui.portal_title", `"   "`},
		{"título del portal largo", "ui.portal_title", `"` + strings.Repeat("a", 61) + `"`},
		{"intro larga", "ui.portal_intro", `"` + strings.Repeat("a", 241) + `"`},
		{"menú desconocido", "ui.nav_layout", `"left"`},
		{"columnas fuera por abajo", "ui.grid_columns", "3"},
		{"columnas fuera por arriba", "ui.grid_columns", "9"},
		{"filas fuera por abajo", "ui.grid_rows", "1"},
		{"filas fuera por arriba", "ui.grid_rows", "13"},
		{"carátulas fuera por abajo", "ui.row_columns", "2"},
		{"carátulas fuera por arriba", "ui.row_columns", "9"},
		{"duración corta", KeyScanInterval, `"30s"`},
		{"duración negativa", KeyScanInterval, `"-1m"`},
		{"duración inválida", KeyScanInterval, `"mañana"`},
		{"duración número", KeyScanInterval, "3600"},
		{"enum desconocido", "metadata.language", `"fr-FR"`},
		{"entero como texto", "ui.grid_columns", `"4"`},
		{"entero decimal", "ui.grid_columns", "4.5"},
		{"transcodificaciones fuera por abajo", "playback.max_transcodes", "0"},
		{"transcodificaciones fuera por arriba", "playback.max_transcodes", "9"},
		{"transcodificaciones decimal", "playback.max_transcodes", "2.5"},
		{"hardware desconocido", KeyPlaybackHardware, `"on"`},
		{"subida fuera por abajo", KeyUploadMaxGB, "0"},
		{"subida fuera por arriba", KeyUploadMaxGB, "501"},
		{"retención fuera por abajo", KeyTrashRetention, "0"},
		{"retención fuera por arriba", KeyTrashRetention, "91"},
		{"valor null", "ui.show_series", "null"},
		{"json inválido", "ui.show_series", "{"},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Validate(test.key, raw(test.raw)); err == nil {
				t.Fatal("se esperaba un error")
			}
		})
	}

	if _, err := Validate("no.existe", raw("true")); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("error de clave desconocida = %v", err)
	}
}

func TestStoredDefaultsAndPersistence(t *testing.T) {
	store, ctx := newTestStore(t, map[string]any{
		KeyScanInterval: 30 * time.Minute,
	})

	if store.String(KeyScanInterval) != "30m" {
		t.Fatalf("intervalo por defecto = %q", store.String(KeyScanInterval))
	}
	if store.Duration(KeyScanInterval) != 30*time.Minute {
		t.Fatalf("duración por defecto = %v", store.Duration(KeyScanInterval))
	}
	if store.Int("ui.grid_columns") != 6 || store.Bool("ui.show_series") != true {
		t.Fatalf("defectos inesperados: %+v", store.All())
	}
	if store.IsModified("ui.grid_columns") {
		t.Fatal("un ajuste sin fila no debe estar modificado")
	}

	changes := map[string]json.RawMessage{
		"ui.grid_columns":     raw("4"),
		"scan.interval":       raw(`"45m"`),
		"ui.show_collections": raw("false"),
	}
	if failures := store.Set(ctx, changes); len(failures) != 0 {
		t.Fatalf("Set: %v", failures)
	}
	if store.Int("ui.grid_columns") != 4 || store.Duration(KeyScanInterval) != 45*time.Minute {
		t.Fatalf("la caché no cambió: %+v", store.All())
	}
	if !store.IsModified("ui.grid_columns") || !store.IsModified(KeyScanInterval) {
		t.Fatal("los ajustes guardados deben estar modificados")
	}

	other := NewStore(store.db, map[string]any{KeyScanInterval: 30 * time.Minute})
	if err := other.Load(ctx); err != nil {
		t.Fatalf("Load del segundo almacén: %v", err)
	}
	if other.Int("ui.grid_columns") != 4 || other.Duration(KeyScanInterval) != 45*time.Minute {
		t.Fatalf("no se conservaron los valores: %+v", other.All())
	}
	if !other.Bool("ui.show_series") {
		t.Fatal("un ajuste no guardado debe conservar su defecto")
	}
}

func TestSetInvalidoNoGuardaNada(t *testing.T) {
	store, ctx := newTestStore(t, nil)

	failures := store.Set(ctx, map[string]json.RawMessage{
		"ui.grid_columns":   raw("4"),
		"playback.crf":      raw("99"),
		"clave.desconocida": raw("true"),
	})
	if len(failures) != 2 {
		t.Fatalf("errores = %d, se esperaban 2: %v", len(failures), failures)
	}
	if _, ok := failures["playback.crf"]; !ok {
		t.Fatalf("falta el error de playback.crf: %v", failures)
	}
	if _, ok := failures["clave.desconocida"]; !ok {
		t.Fatalf("falta el error de la clave desconocida: %v", failures)
	}
	if store.Int("ui.grid_columns") != 6 {
		t.Fatalf("el ajuste válido no debe guardarse: %d", store.Int("ui.grid_columns"))
	}
	var count int
	if err := store.db.QueryRow(`SELECT count(*) FROM settings`).Scan(&count); err != nil {
		t.Fatalf("contar ajustes: %v", err)
	}
	if count != 0 {
		t.Fatalf("filas guardadas = %d, se esperaban 0", count)
	}
}

func TestResetYOnChange(t *testing.T) {
	store, ctx := newTestStore(t, nil)
	var calls [][]string
	store.OnChange(func(keys []string) {
		calls = append(calls, append([]string(nil), keys...))
		_ = store.Get("ui.grid_columns")
	})

	if failures := store.Set(ctx, map[string]json.RawMessage{
		"ui.show_series":  raw("false"),
		"ui.grid_columns": raw("4"),
	}); len(failures) != 0 {
		t.Fatalf("Set: %v", failures)
	}
	if len(calls) != 1 || !reflect.DeepEqual(calls[0], []string{"ui.grid_columns", "ui.show_series"}) {
		t.Fatalf("llamadas a OnChange = %v", calls)
	}

	if err := store.Reset(ctx, "ui.grid_columns"); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if store.Int("ui.grid_columns") != 6 || store.IsModified("ui.grid_columns") {
		t.Fatal("Reset no volvió al defecto")
	}
	if len(calls) != 2 || !reflect.DeepEqual(calls[1], []string{"ui.grid_columns"}) {
		t.Fatalf("llamadas tras Reset = %v", calls)
	}
	if err := store.Reset(ctx, "clave.desconocida"); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("Reset de clave desconocida = %v", err)
	}
}

func TestLoadIgnoraClavesYValoresInvalidos(t *testing.T) {
	store, ctx := newTestStore(t, nil)
	if _, err := store.db.ExecContext(ctx, `
		INSERT INTO settings (key, value) VALUES
			('clave.desconocida', '{"x":1}'),
			('ui.page_size', '24'),
			('ui.grid_columns', '9'),
			('ui.show_series', 'no-es-json'),
			('playback.crf', '22')`); err != nil {
		t.Fatalf("insertar ajustes de prueba: %v", err)
	}

	if err := store.Load(ctx); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if store.Int("ui.grid_columns") != 6 || !store.Bool("ui.show_series") {
		t.Fatalf("los valores inválidos no deben aplicarse: %+v", store.All())
	}
	if store.Int("playback.crf") != 22 || !store.IsModified("playback.crf") {
		t.Fatalf("el valor válido no se aplicó: %+v", store.All())
	}
	if store.IsModified("ui.grid_columns") || store.IsModified("ui.show_series") {
		t.Fatal("los valores inválidos no deben quedar modificados")
	}
	if store.Get("clave.desconocida") != nil || store.Get("ui.page_size") != nil {
		t.Fatal("las claves desconocidas no deben estar en la caché")
	}
}

func TestStoreConcurrente(t *testing.T) {
	store, ctx := newTestStore(t, nil)
	var workers sync.WaitGroup
	for index := 0; index < 8; index++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			for iteration := 0; iteration < 10; iteration++ {
				value := "4"
				if (index+iteration)%2 == 0 {
					value = "8"
				}
				_ = store.Set(ctx, map[string]json.RawMessage{
					"ui.grid_columns": raw(value),
				})
				_ = store.Get("ui.grid_columns")
				_ = store.All()
			}
		}(index)
	}
	workers.Wait()

	if value := store.Int("ui.grid_columns"); value != 4 && value != 8 {
		t.Fatalf("valor concurrente inesperado: %d", value)
	}
}
