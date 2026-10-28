package library

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/catlinux/cinecat/internal/settings"
)

type settingsOptionDTO struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type settingDTO struct {
	Key         string              `json:"key"`
	Label       string              `json:"label"`
	Description string              `json:"description"`
	Type        settings.Type       `json:"type"`
	Options     []settingsOptionDTO `json:"options"`
	Min         float64             `json:"min"`
	Max         float64             `json:"max"`
	Default     any                 `json:"default"`
	Value       any                 `json:"value"`
	Modified    bool                `json:"modified"`
}

type settingsSectionDTO struct {
	Key      string       `json:"key"`
	Label    string       `json:"label"`
	Settings []settingDTO `json:"settings"`
}

type settingsResponse struct {
	Sections []settingsSectionDTO `json:"sections"`
}

type settingsErrorResponse struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields"`
}

const defaultCollectionMinItems = 2

func collectionMinimum(values ...int) int {
	if len(values) > 0 && values[0] >= 1 {
		return values[0]
	}
	return defaultCollectionMinItems
}

// collectionMin es el mínimo de películas para mostrar una colección
// automática. Sin Store se conserva el comportamiento anterior.
func (a *API) collectionMin() int {
	if a.settingsStore == nil {
		return defaultCollectionMinItems
	}
	value := a.settingsStore.Int("ui.collection_min_items")
	if value < 1 {
		return defaultCollectionMinItems
	}
	return value
}

func (a *API) handleSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.buildSettingsResponse())
}

func (a *API) handleSettingsPatch(w http.ResponseWriter, r *http.Request) {
	var changes map[string]json.RawMessage
	if err := decodeJSONBody(w, r, &changes); err != nil {
		writeJSONBodyError(w, err)
		return
	}
	if changes == nil {
		writeJSONError(w, http.StatusBadRequest, "petición no válida")
		return
	}

	failures := a.settingsStore.Set(r.Context(), changes)
	if len(failures) > 0 {
		fields := make(map[string]string, len(failures))
		storageError := false
		for key, err := range failures {
			fields[key] = err.Error()
			if errors.Is(err, settings.ErrStorage) {
				storageError = true
			}
		}
		if storageError {
			log.Printf("library: guardar ajustes: %v", failures)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		writeJSON(w, http.StatusBadRequest, settingsErrorResponse{
			Error:  "hay ajustes no válidos",
			Fields: fields,
		})
		return
	}

	writeJSON(w, http.StatusOK, a.buildSettingsResponse())
}

func (a *API) handleSettingDelete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		writeJSONError(w, http.StatusNotFound, "ajuste no encontrado")
		return
	}
	definition, ok := settings.Lookup(key)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ajuste no encontrado")
		return
	}

	if err := a.settingsStore.Reset(r.Context(), key); err != nil {
		if errors.Is(err, settings.ErrUnknownKey) {
			writeJSONError(w, http.StatusNotFound, "ajuste no encontrado")
			return
		}
		log.Printf("library: restablecer ajuste %q: %v", key, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	writeJSON(w, http.StatusOK, a.settingDTO(definition))
}

func (a *API) handleSettingsPublic(w http.ResponseWriter, r *http.Request) {
	values := make(map[string]any)
	for _, definition := range settings.Definitions() {
		if definition.Public {
			values[definition.Key] = a.settingsStore.Get(definition.Key)
		}
	}
	writeJSON(w, http.StatusOK, values)
}

func (a *API) buildSettingsResponse() settingsResponse {
	response := settingsResponse{Sections: make([]settingsSectionDTO, 0, len(settings.Sections()))}
	sectionIndex := make(map[string]int, len(settings.Sections()))
	for _, section := range settings.Sections() {
		sectionIndex[section.Key] = len(response.Sections)
		response.Sections = append(response.Sections, settingsSectionDTO{
			Key:      section.Key,
			Label:    section.Label,
			Settings: make([]settingDTO, 0),
		})
	}

	for _, definition := range settings.Definitions() {
		index, ok := sectionIndex[definition.Section]
		if !ok {
			continue
		}
		response.Sections[index].Settings = append(
			response.Sections[index].Settings,
			a.settingDTO(definition),
		)
	}
	return response
}

func (a *API) settingDTO(definition settings.Definition) settingDTO {
	options := make([]settingsOptionDTO, 0, len(definition.Options))
	for _, option := range definition.Options {
		options = append(options, settingsOptionDTO{
			Value: option.Value,
			Label: option.Label,
		})
	}
	return settingDTO{
		Key:         definition.Key,
		Label:       definition.Label,
		Description: definition.Description,
		Type:        definition.Type,
		Options:     options,
		Min:         definition.Min,
		Max:         definition.Max,
		Default:     a.settingsStore.Default(definition.Key),
		Value:       a.settingsStore.Get(definition.Key),
		Modified:    a.settingsStore.IsModified(definition.Key),
	}
}
