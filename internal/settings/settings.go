// Package settings define, valida y guarda los ajustes configurables del
// servidor. Los valores efectivos se mantienen en memoria para que el resto
// del servidor no consulte la base de datos en cada uso.
package settings

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Type identifica el tipo de un ajuste.
type Type string

const (
	TypeBool     Type = "bool"
	TypeInt      Type = "int"
	TypeFloat    Type = "float"
	TypeString   Type = "string"
	TypeDuration Type = "duration"
	TypeEnum     Type = "enum"
)

func (t Type) String() string {
	return string(t)
}

// KeyScanInterval es la clave del intervalo de escaneo.
const KeyScanInterval = "scan.interval"

// ErrUnknownKey indica que la clave no forma parte del registro de ajustes.
var ErrUnknownKey = errors.New("ajuste desconocido")

// ErrStorage indica que no se pudieron persistir los ajustes.
var ErrStorage = errors.New("no se pudieron guardar los ajustes")

// Option es una de las opciones de un ajuste de tipo enum.
type Option struct {
	Value string
	Label string
}

// Section agrupa los ajustes que se muestran juntos en la interfaz.
type Section struct {
	Key   string
	Label string
}

// Definition describe un ajuste: su tipo, valor por defecto y límites.
type Definition struct {
	Key         string
	Section     string
	Label       string
	Description string
	Type        Type
	Default     any
	Options     []Option
	Min         float64
	Max         float64
	Public      bool
}

var sections = []Section{
	{Key: "scan", Label: "Escaneo"},
	{Key: "metadata", Label: "Metadatos"},
	{Key: "ui", Label: "Interfaz"},
	{Key: "playback", Label: "Reproducción"},
	{Key: "system", Label: "Sistema"},
}

var definitions = []Definition{
	{
		Key:         KeyScanInterval,
		Section:     "scan",
		Label:       "Intervalo de escaneo",
		Description: "Cada cuánto se escanea sola la biblioteca. Usa «0» para desactivarlo; el mínimo es 1 minuto.",
		Type:        TypeDuration,
		Default:     "6h",
		Min:         60,
	},
	{
		Key:         "scan.on_startup",
		Section:     "scan",
		Label:       "Escanear al arrancar",
		Description: "Lanza un escaneo un minuto después de iniciar el servidor.",
		Type:        TypeBool,
		Default:     true,
	},
	{
		Key:         "metadata.language",
		Section:     "metadata",
		Label:       "Idioma de los metadatos",
		Description: "Idioma preferido para las sinopsis y fichas nuevas de TMDB; no reescribe las fichas ya guardadas.",
		Type:        TypeEnum,
		Default:     "es-ES",
		Options: []Option{
			{Value: "es-ES", Label: "Español (España)"},
			{Value: "es-MX", Label: "Español (México)"},
			{Value: "en-US", Label: "Inglés (Estados Unidos)"},
		},
	},
	{
		Key:         "metadata.match_threshold",
		Section:     "metadata",
		Label:       "Umbral de identificación",
		Description: "Confianza mínima para aceptar automáticamente una coincidencia.",
		Type:        TypeFloat,
		Default:     0.75,
		Min:         0.6,
		Max:         0.95,
	},
	{
		Key:         "ui.server_name",
		Section:     "ui",
		Label:       "Nombre del servidor",
		Description: "Nombre que aparece en la interfaz.",
		Type:        TypeString,
		Default:     "Cinecat",
		Min:         1,
		Max:         40,
		Public:      true,
	},
	{
		Key:         "ui.tagline",
		Section:     "ui",
		Label:       "Eslogan",
		Description: "Frase corta junto al nombre en la cabecera. Vacía para ocultarla.",
		Type:        TypeString,
		Default:     "Sesión continua",
		Max:         60,
		Public:      true,
	},
	{
		Key:         "ui.portal_title",
		Section:     "ui",
		Label:       "Título del portal",
		Description: "Título principal de la página de inicio.",
		Type:        TypeString,
		Default:     "Tu cine en casa",
		Min:         1,
		Max:         60,
		Public:      true,
	},
	{
		Key:         "ui.portal_intro",
		Section:     "ui",
		Label:       "Texto del portal",
		Description: "Párrafo bajo el título de la página de inicio. Vacío para ocultarlo.",
		Type:        TypeString,
		Default:     "Una biblioteca personal con olor a palomitas, carteles pintados a mano y la sesión de siempre esperándote.",
		Max:         240,
		Public:      true,
	},
	{
		Key:         "ui.nav_layout",
		Section:     "ui",
		Label:       "Menú de navegación",
		Description: "Posición del menú principal: lateral en escritorio o superior.",
		Type:        TypeEnum,
		Default:     "side",
		Options: []Option{
			{Value: "side", Label: "Lateral (izquierda)"},
			{Value: "top", Label: "Superior"},
		},
		Public: true,
	},
	{
		Key:         "ui.grid_columns",
		Section:     "ui",
		Label:       "Columnas de los listados",
		Description: "Carátulas por fila en películas, series y colecciones.",
		Type:        TypeInt,
		Default:     6,
		Min:         4,
		Max:         8,
		Public:      true,
	},
	{
		Key:         "ui.grid_rows",
		Section:     "ui",
		Label:       "Filas por página",
		Description: "Filas de carátulas por página; el número de resultados es columnas × filas.",
		Type:        TypeInt,
		Default:     8,
		Min:         2,
		Max:         12,
		Public:      true,
	},
	{
		Key:         "ui.row_columns",
		Section:     "ui",
		Label:       "Carátulas por fila en el portal",
		Description: "Cuántas carátulas se ven a la vez en las filas desplazables.",
		Type:        TypeInt,
		Default:     6,
		Min:         3,
		Max:         8,
		Public:      true,
	},
	{
		Key:         "ui.default_sort",
		Section:     "ui",
		Label:       "Orden por defecto",
		Description: "Criterio de orden inicial de las películas.",
		Type:        TypeEnum,
		Default:     "title",
		Options: []Option{
			{Value: "title", Label: "Título"},
			{Value: "year", Label: "Año"},
			{Value: "added", Label: "Añadido"},
		},
		Public: true,
	},
	{
		Key:         "ui.show_collections",
		Section:     "ui",
		Label:       "Mostrar colecciones",
		Description: "Muestra la sección de colecciones en la interfaz.",
		Type:        TypeBool,
		Default:     true,
		Public:      true,
	},
	{
		Key:         "ui.show_series",
		Section:     "ui",
		Label:       "Mostrar series",
		Description: "Muestra la sección de series en la interfaz.",
		Type:        TypeBool,
		Default:     true,
		Public:      true,
	},
	{
		Key:         "ui.collection_min_items",
		Section:     "ui",
		Label:       "Mínimo para colecciones",
		Description: "Número mínimo de películas para mostrar una colección automática.",
		Type:        TypeInt,
		Default:     2,
		Min:         1,
		Max:         10,
		Public:      true,
	},
	{
		Key:         "ui.theme_default",
		Section:     "ui",
		Label:       "Tema por defecto",
		Description: "Tema inicial cuando no hay una preferencia guardada.",
		Type:        TypeEnum,
		Default:     "system",
		Options: []Option{
			{Value: "system", Label: "Sistema"},
			{Value: "light", Label: "Claro"},
			{Value: "dark", Label: "Oscuro"},
		},
		Public: true,
	},
	{
		Key:         "ui.accent",
		Section:     "ui",
		Label:       "Color de acento",
		Description: "Color de acento de la interfaz.",
		Type:        TypeEnum,
		Default:     "granate",
		Options: []Option{
			{Value: "granate", Label: "Granate"},
			{Value: "naranja", Label: "Naranja"},
			{Value: "dorado", Label: "Dorado"},
		},
		Public: true,
	},
	{
		Key:         "playback.max_height",
		Section:     "playback",
		Label:       "Altura máxima",
		Description: "Resolución máxima de los remux.",
		Type:        TypeEnum,
		Default:     1080,
		Options: []Option{
			{Value: "480", Label: "480p"},
			{Value: "720", Label: "720p"},
			{Value: "1080", Label: "1080p"},
		},
		Min: 480,
		Max: 1080,
	},
	{
		Key:         "playback.crf",
		Section:     "playback",
		Label:       "Calidad (CRF)",
		Description: "Calidad de compresión del remux; un valor menor da más calidad y tamaño.",
		Type:        TypeInt,
		Default:     20,
		Min:         18,
		Max:         28,
	},
	{
		Key:         "playback.preset",
		Section:     "playback",
		Label:       "Velocidad de codificación",
		Description: "Preset de x264 usado al remuxear.",
		Type:        TypeEnum,
		Default:     "veryfast",
		Options: []Option{
			{Value: "veryfast", Label: "Muy rápida"},
			{Value: "faster", Label: "Rápida"},
			{Value: "fast", Label: "Normal"},
		},
	},
	{
		Key:         "playback.max_transcodes",
		Section:     "playback",
		Label:       "Transcodificaciones simultáneas",
		Description: "Cuántas películas se pueden recodificar a la vez. Más de las que aguante el procesador provoca cortes al avanzar.",
		Type:        TypeInt,
		Default:     2,
		Min:         1,
		Max:         8,
	},
	{
		Key:         "system.check_updates",
		Section:     "system",
		Label:       "Buscar actualizaciones",
		Description: "Comprueba si hay versiones nuevas; implica una conexión saliente opcional que debe activar el administrador.",
		Type:        TypeBool,
		Default:     false,
	},
}

var definitionsByKey = func() map[string]Definition {
	byKey := make(map[string]Definition, len(definitions))
	for _, definition := range definitions {
		byKey[definition.Key] = definition
	}
	return byKey
}()

// Definitions devuelve el registro ordenado de ajustes. La copia evita que
// quien consulte el catálogo pueda alterar el registro compartido.
func Definitions() []Definition {
	result := make([]Definition, 0, len(definitions))
	for _, definition := range definitions {
		result = append(result, copyDefinition(definition))
	}
	return result
}

// Sections devuelve las secciones en el orden en que se muestran.
func Sections() []Section {
	result := make([]Section, len(sections))
	copy(result, sections)
	return result
}

// Lookup busca una definición por su clave.
func Lookup(key string) (Definition, bool) {
	definition, ok := definitionsByKey[key]
	if !ok {
		return Definition{}, false
	}
	return copyDefinition(definition), true
}

func copyDefinition(definition Definition) Definition {
	definition.Options = append([]Option(nil), definition.Options...)
	return definition
}

// Validate comprueba y normaliza un valor JSON para la clave indicada.
func Validate(key string, raw json.RawMessage) (any, error) {
	definition, ok := Lookup(key)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownKey, key)
	}

	value, err := decodeValue(raw)
	if err != nil {
		return nil, err
	}

	switch definition.Type {
	case TypeBool:
		boolean, ok := value.(bool)
		if !ok {
			return nil, errors.New("el valor debe ser booleano")
		}
		return boolean, nil

	case TypeInt:
		number, ok := value.(json.Number)
		if !ok {
			return nil, errors.New("el valor debe ser un número entero")
		}
		integer, err := strconv.ParseInt(number.String(), 10, 64)
		if err != nil {
			return nil, errors.New("el valor debe ser un número entero")
		}
		if err := validateRange(definition, float64(integer)); err != nil {
			return nil, err
		}
		return int(integer), nil

	case TypeFloat:
		number, ok := value.(json.Number)
		if !ok {
			return nil, errors.New("el valor debe ser un número")
		}
		decimal, err := number.Float64()
		if err != nil {
			return nil, errors.New("el valor debe ser un número")
		}
		if err := validateRange(definition, decimal); err != nil {
			return nil, err
		}
		return decimal, nil

	case TypeString:
		text, ok := value.(string)
		if !ok {
			return nil, errors.New("el valor debe ser un texto")
		}
		text = strings.TrimSpace(text)
		length := utf8.RuneCountInString(text)
		if definition.Min > 0 && length < int(definition.Min) {
			return nil, fmt.Errorf("el texto debe tener al menos %d caracteres", int(definition.Min))
		}
		if definition.Max > 0 && length > int(definition.Max) {
			return nil, fmt.Errorf("el texto no puede superar los %d caracteres", int(definition.Max))
		}
		return text, nil

	case TypeDuration:
		text, ok := value.(string)
		if !ok {
			return nil, errors.New("el valor debe ser una duración de texto")
		}
		duration, err := time.ParseDuration(strings.TrimSpace(text))
		if err != nil {
			return nil, errors.New("la duración no es válida")
		}
		if duration < 0 {
			return nil, errors.New("la duración no puede ser negativa")
		}
		if duration != 0 && duration < time.Minute {
			return nil, errors.New("la duración debe ser «0» o de al menos 1 minuto")
		}
		return formatDuration(duration), nil

	case TypeEnum:
		return validateEnum(definition, value)
	}

	return nil, errors.New("tipo de ajuste no válido")
}

func decodeValue(raw json.RawMessage) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, errors.New("el valor no es un JSON válido")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, errors.New("el valor no es un JSON válido")
	}
	return value, nil
}

func validateRange(definition Definition, value float64) error {
	if definition.Min != 0 && value < definition.Min {
		return fmt.Errorf("el valor debe estar entre %s y %s", formatNumber(definition.Min), formatNumber(definition.Max))
	}
	if definition.Max != 0 && value > definition.Max {
		return fmt.Errorf("el valor debe estar entre %s y %s", formatNumber(definition.Min), formatNumber(definition.Max))
	}
	return nil
}

func formatNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func validateEnum(definition Definition, value any) (any, error) {
	numeric := numericEnum(definition)
	if numeric {
		number, ok := value.(json.Number)
		if !ok {
			return nil, errors.New("el valor debe ser un número entero")
		}
		integer, err := strconv.ParseInt(number.String(), 10, 64)
		if err != nil {
			return nil, errors.New("el valor debe ser un número entero")
		}
		if !enumHasValue(definition, strconv.FormatInt(integer, 10)) {
			return nil, enumError(definition)
		}
		return int(integer), nil
	}

	text, ok := value.(string)
	if !ok {
		return nil, errors.New("el valor debe ser un texto")
	}
	text = strings.TrimSpace(text)
	if !enumHasValue(definition, text) {
		return nil, enumError(definition)
	}
	return text, nil
}

func numericEnum(definition Definition) bool {
	if len(definition.Options) == 0 {
		return false
	}
	for _, option := range definition.Options {
		if _, err := strconv.Atoi(option.Value); err != nil {
			return false
		}
	}
	return true
}

func enumHasValue(definition Definition, value string) bool {
	for _, option := range definition.Options {
		if option.Value == value {
			return true
		}
	}
	return false
}

func enumError(definition Definition) error {
	values := make([]string, 0, len(definition.Options))
	for _, option := range definition.Options {
		values = append(values, option.Value)
	}
	return fmt.Errorf("debe ser una de las opciones: %s", strings.Join(values, ", "))
}

func formatDuration(duration time.Duration) string {
	if duration == 0 {
		return "0"
	}
	sign := ""
	if duration < 0 {
		sign = "-"
		duration = -duration
	}

	hours := duration / time.Hour
	duration -= hours * time.Hour
	minutes := duration / time.Minute
	duration -= minutes * time.Minute

	var builder strings.Builder
	builder.WriteString(sign)
	if hours > 0 {
		fmt.Fprintf(&builder, "%dh", hours)
	}
	if minutes > 0 {
		fmt.Fprintf(&builder, "%dm", minutes)
	}
	if duration > 0 {
		seconds := float64(duration) / float64(time.Second)
		builder.WriteString(strconv.FormatFloat(seconds, 'f', -1, 64))
		builder.WriteString("s")
	}
	return builder.String()
}

// Store mantiene los ajustes efectivos en memoria y los persiste en SQLite.
type Store struct {
	db *sql.DB

	writeMu sync.Mutex
	mu      sync.RWMutex

	defaults map[string]any
	values   map[string]any
	modified map[string]bool

	callbacks []func(keys []string)
}

// NewStore crea el almacén con los valores por defecto del registro. El mapa
// defaults permite sustituirlos, por ejemplo con el intervalo de la config.
func NewStore(db *sql.DB, defaults map[string]any) *Store {
	store := &Store{
		db:       db,
		defaults: make(map[string]any, len(definitions)),
		values:   make(map[string]any, len(definitions)),
		modified: make(map[string]bool, len(definitions)),
	}
	for _, definition := range definitions {
		store.defaults[definition.Key] = definition.Default
	}
	for key, value := range defaults {
		definition, ok := Lookup(key)
		if !ok {
			log.Printf("settings: se ignora el defecto de la clave desconocida %q", key)
			continue
		}
		normalized, ok := normalizeDefault(definition, value)
		if !ok {
			log.Printf("settings: se ignora el defecto no válido de %q", key)
			continue
		}
		store.defaults[key] = normalized
	}
	for key, value := range store.defaults {
		store.values[key] = value
	}
	return store
}

func normalizeDefault(definition Definition, value any) (any, bool) {
	if definition.Type == TypeDuration {
		switch typed := value.(type) {
		case time.Duration:
			if typed < 0 {
				return nil, false
			}
			return formatDuration(typed), true
		case string:
			duration, err := time.ParseDuration(strings.TrimSpace(typed))
			if err != nil || duration < 0 {
				return nil, false
			}
			return formatDuration(duration), true
		default:
			return nil, false
		}
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	normalized, err := Validate(definition.Key, raw)
	if err != nil {
		return nil, false
	}
	return normalized, true
}

// Load lee los ajustes guardados y actualiza la caché. Las claves
// desconocidas y los valores inválidos se ignoran y conservan su defecto.
func (s *Store) Load(ctx context.Context) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return fmt.Errorf("settings: leer ajustes: %w", err)
	}
	defer rows.Close()

	values := copyMap(s.defaults)
	modified := make(map[string]bool, len(values))
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return fmt.Errorf("settings: leer ajuste: %w", err)
		}
		if _, ok := Lookup(key); !ok {
			log.Printf("settings: se ignora la clave desconocida %q", key)
			continue
		}
		normalized, err := Validate(key, json.RawMessage(value))
		if err != nil {
			log.Printf("settings: se ignora el valor no válido de %q: %v", key, err)
			continue
		}
		values[key] = normalized
		modified[key] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("settings: recorrer ajustes: %w", err)
	}

	s.mu.Lock()
	s.values = values
	s.modified = modified
	s.mu.Unlock()
	return nil
}

// Get devuelve el valor efectivo de una clave, o nil si no existe.
func (s *Store) Get(key string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.values[key]
}

// Default devuelve el valor por defecto efectivo de una clave.
func (s *Store) Default(key string) any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaults[key]
}

// IsModified indica si la clave tiene un valor guardado en la base de datos.
func (s *Store) IsModified(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.modified[key]
}

// All devuelve una copia de todos los valores efectivos.
func (s *Store) All() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return copyMap(s.values)
}

// String devuelve un ajuste de texto.
func (s *Store) String(key string) string {
	value, _ := s.Get(key).(string)
	return value
}

// Bool devuelve un ajuste booleano.
func (s *Store) Bool(key string) bool {
	value, _ := s.Get(key).(bool)
	return value
}

// Int devuelve un ajuste entero.
func (s *Store) Int(key string) int {
	switch value := s.Get(key).(type) {
	case int:
		return value
	case int64:
		return int(value)
	case json.Number:
		integer, err := strconv.Atoi(value.String())
		if err == nil {
			return integer
		}
	}
	return 0
}

// Float devuelve un ajuste decimal.
func (s *Store) Float(key string) float64 {
	switch value := s.Get(key).(type) {
	case float64:
		return value
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case json.Number:
		decimal, err := value.Float64()
		if err == nil {
			return decimal
		}
	}
	return 0
}

// Duration devuelve un ajuste de duración.
func (s *Store) Duration(key string) time.Duration {
	switch value := s.Get(key).(type) {
	case time.Duration:
		return value
	case string:
		duration, err := time.ParseDuration(value)
		if err == nil {
			return duration
		}
	}
	return 0
}

// Set valida todas las claves antes de guardar nada. Si alguna falla, devuelve
// los errores por clave y la caché y la base de datos quedan intactas.
func (s *Store) Set(ctx context.Context, changes map[string]json.RawMessage) map[string]error {
	normalized := make(map[string]any, len(changes))
	failures := make(map[string]error)
	for key, raw := range changes {
		value, err := Validate(key, raw)
		if err != nil {
			failures[key] = err
			continue
		}
		normalized[key] = value
	}
	if len(failures) > 0 {
		return failures
	}
	if len(normalized) == 0 {
		return nil
	}

	keys := keysInDefinitionOrder(normalized)
	s.writeMu.Lock()

	s.mu.RLock()
	changed := make([]string, 0, len(keys))
	for _, key := range keys {
		if !s.modified[key] || !reflect.DeepEqual(s.values[key], normalized[key]) {
			changed = append(changed, key)
		}
	}
	s.mu.RUnlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.writeMu.Unlock()
		return storageFailures(keys, err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, key := range keys {
		data, err := json.Marshal(normalized[key])
		if err != nil {
			s.writeMu.Unlock()
			return storageFailures(keys, err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO settings (key, value) VALUES (?, ?)
			ON CONFLICT (key) DO UPDATE SET value = excluded.value,
			                               updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
			key, string(data)); err != nil {
			s.writeMu.Unlock()
			return storageFailures(keys, err)
		}
	}
	if err := tx.Commit(); err != nil {
		s.writeMu.Unlock()
		return storageFailures(keys, err)
	}

	s.mu.Lock()
	for _, key := range keys {
		s.values[key] = normalized[key]
		s.modified[key] = true
	}
	callbacks := append([]func([]string){}, s.callbacks...)
	s.mu.Unlock()
	s.writeMu.Unlock()

	notify(callbacks, changed)
	return nil
}

// Reset borra el valor guardado de una clave y vuelve a su defecto.
func (s *Store) Reset(ctx context.Context, key string) error {
	if _, ok := Lookup(key); !ok {
		return fmt.Errorf("%w: %s", ErrUnknownKey, key)
	}

	s.writeMu.Lock()

	if _, err := s.db.ExecContext(ctx, `DELETE FROM settings WHERE key = ?`, key); err != nil {
		s.writeMu.Unlock()
		return fmt.Errorf("settings: restablecer ajuste: %w", err)
	}

	s.mu.Lock()
	changed := s.modified[key] || !reflect.DeepEqual(s.values[key], s.defaults[key])
	s.values[key] = s.defaults[key]
	s.modified[key] = false
	callbacks := append([]func([]string){}, s.callbacks...)
	s.mu.Unlock()
	s.writeMu.Unlock()

	if changed {
		notify(callbacks, []string{key})
	}
	return nil
}

// OnChange registra una función que recibe las claves que cambian. No se
// llama con la caché bloqueada para que pueda consultar el almacén.
func (s *Store) OnChange(callback func(keys []string)) {
	if callback == nil {
		return
	}
	s.mu.Lock()
	s.callbacks = append(s.callbacks, callback)
	s.mu.Unlock()
}

func notify(callbacks []func([]string), keys []string) {
	if len(keys) == 0 {
		return
	}
	for _, callback := range callbacks {
		callback(append([]string(nil), keys...))
	}
}

func storageFailures(keys []string, cause error) map[string]error {
	failures := make(map[string]error, len(keys))
	for _, key := range keys {
		failures[key] = fmt.Errorf("%w: %s: %v", ErrStorage, key, cause)
	}
	return failures
}

func keysInDefinitionOrder(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for _, definition := range definitions {
		if _, ok := values[definition.Key]; ok {
			keys = append(keys, definition.Key)
		}
	}
	return keys
}

func copyMap(source map[string]any) map[string]any {
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
