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
	if len(definitions) != 17 {
		t.Fatalf("ajustes = %d, se esperaban 17", len(definitions))
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
	if public != 8 {
		t.Fatalf("ajustes públicos = %d, se esperaban 8", public)
	}

	sectionKeys := make([]string, 0, len(Sections()))
	for _, section := range Sections() {
		sectionKeys = append(sectionKeys, section.Key)
	}
	if !reflect.DeepEqual(sectionKeys, []string{"scan", "metadata", "ui", "playback", "system"}) {
		t.Fatalf("secciones inesperadas: %v", sectionKeys)
	}
	if lookup, ok := Lookup(KeyScanInterval); !ok || lookup.Default != "6h" {
		t.Fatalf("scan.interval inesperado: %+v, %v", lookup, ok)
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
		{"duración cero", KeyScanInterval, `"0"`, "0"},
		{"duración minutos", KeyScanInterval, `"30m"`, "30m"},
		{"duración horas", KeyScanInterval, `"6h"`, "6h"},
		{"duración compuesta", KeyScanInterval, `"1h30m"`, "1h30m"},
		{"enum texto", "metadata.language", `"es-MX"`, "es-MX"},
		{"enum numérico", "ui.page_size", "24", 24},
		{"enum numérico alto", "playback.max_height", "1080", 1080},
		{"enum numérico mínimo", "playback.max_height", "480", 480},
		{"transcodificaciones mínimo", "playback.max_transcodes", "1", 1},
		{"transcodificaciones máximo", "playback.max_transcodes", "8", 8},
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
		{"duración corta", KeyScanInterval, `"30s"`},
		{"duración negativa", KeyScanInterval, `"-1m"`},
		{"duración inválida", KeyScanInterval, `"mañana"`},
		{"duración número", KeyScanInterval, "3600"},
		{"enum desconocido", "metadata.language", `"fr-FR"`},
		{"enum numérico texto", "ui.page_size", `"24"`},
		{"enum numérico fuera", "ui.page_size", "50"},
		{"transcodificaciones fuera por abajo", "playback.max_transcodes", "0"},
		{"transcodificaciones fuera por arriba", "playback.max_transcodes", "9"},
		{"transcodificaciones decimal", "playback.max_transcodes", "2.5"},
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
	if store.Int("ui.page_size") != 48 || store.Bool("ui.show_series") != true {
		t.Fatalf("defectos inesperados: %+v", store.All())
	}
	if store.IsModified("ui.page_size") {
		t.Fatal("un ajuste sin fila no debe estar modificado")
	}

	changes := map[string]json.RawMessage{
		"ui.page_size":        raw("24"),
		"scan.interval":       raw(`"45m"`),
		"ui.show_collections": raw("false"),
	}
	if failures := store.Set(ctx, changes); len(failures) != 0 {
		t.Fatalf("Set: %v", failures)
	}
	if store.Int("ui.page_size") != 24 || store.Duration(KeyScanInterval) != 45*time.Minute {
		t.Fatalf("la caché no cambió: %+v", store.All())
	}
	if !store.IsModified("ui.page_size") || !store.IsModified(KeyScanInterval) {
		t.Fatal("los ajustes guardados deben estar modificados")
	}

	other := NewStore(store.db, map[string]any{KeyScanInterval: 30 * time.Minute})
	if err := other.Load(ctx); err != nil {
		t.Fatalf("Load del segundo almacén: %v", err)
	}
	if other.Int("ui.page_size") != 24 || other.Duration(KeyScanInterval) != 45*time.Minute {
		t.Fatalf("no se conservaron los valores: %+v", other.All())
	}
	if !other.Bool("ui.show_series") {
		t.Fatal("un ajuste no guardado debe conservar su defecto")
	}
}

func TestSetInvalidoNoGuardaNada(t *testing.T) {
	store, ctx := newTestStore(t, nil)

	failures := store.Set(ctx, map[string]json.RawMessage{
		"ui.page_size":      raw("24"),
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
	if store.Int("ui.page_size") != 48 {
		t.Fatalf("el ajuste válido no debe guardarse: %d", store.Int("ui.page_size"))
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
		_ = store.Get("ui.page_size")
	})

	if failures := store.Set(ctx, map[string]json.RawMessage{
		"ui.show_series": raw("false"),
		"ui.page_size":   raw("24"),
	}); len(failures) != 0 {
		t.Fatalf("Set: %v", failures)
	}
	if len(calls) != 1 || !reflect.DeepEqual(calls[0], []string{"ui.page_size", "ui.show_series"}) {
		t.Fatalf("llamadas a OnChange = %v", calls)
	}

	if err := store.Reset(ctx, "ui.page_size"); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if store.Int("ui.page_size") != 48 || store.IsModified("ui.page_size") {
		t.Fatal("Reset no volvió al defecto")
	}
	if len(calls) != 2 || !reflect.DeepEqual(calls[1], []string{"ui.page_size"}) {
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
			('ui.page_size', '25'),
			('ui.show_series', 'no-es-json'),
			('playback.crf', '22')`); err != nil {
		t.Fatalf("insertar ajustes de prueba: %v", err)
	}

	if err := store.Load(ctx); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if store.Int("ui.page_size") != 48 || !store.Bool("ui.show_series") {
		t.Fatalf("los valores inválidos no deben aplicarse: %+v", store.All())
	}
	if store.Int("playback.crf") != 22 || !store.IsModified("playback.crf") {
		t.Fatalf("el valor válido no se aplicó: %+v", store.All())
	}
	if store.IsModified("ui.page_size") || store.IsModified("ui.show_series") {
		t.Fatal("los valores inválidos no deben quedar modificados")
	}
	if store.Get("clave.desconocida") != nil {
		t.Fatal("la clave desconocida no debe estar en la caché")
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
				value := "24"
				if (index+iteration)%2 == 0 {
					value = "96"
				}
				_ = store.Set(ctx, map[string]json.RawMessage{
					"ui.page_size": raw(value),
				})
				_ = store.Get("ui.page_size")
				_ = store.All()
			}
		}(index)
	}
	workers.Wait()

	if value := store.Int("ui.page_size"); value != 24 && value != 96 {
		t.Fatalf("valor concurrente inesperado: %d", value)
	}
}
