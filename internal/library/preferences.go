package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type itemPreferencesDTO struct {
	AudioIndex *int    `json:"audio_index"`
	SubtitleID *string `json:"subtitle_id"`
	Quality    *string `json:"quality"`
}

var subtitlePreferencePattern = regexp.MustCompile(`^[ex][0-9]+$`)

func validQuality(value string) bool {
	switch value {
	case "auto", "original", "480", "720", "1080":
		return true
	default:
		return false
	}
}

func loadItemPreferences(ctx context.Context, db *sql.DB, userID, itemID int64) (*itemPreferencesDTO, error) {
	var preferences itemPreferencesDTO
	var audio sql.NullInt64
	var subtitle, quality sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT audio_index, subtitle_id, quality
		  FROM item_preferences
		 WHERE user_id = ? AND media_item_id = ?`, userID, itemID).
		Scan(&audio, &subtitle, &quality)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if audio.Valid {
		value := int(audio.Int64)
		preferences.AudioIndex = &value
	}
	if subtitle.Valid {
		value := subtitle.String
		preferences.SubtitleID = &value
	}
	if quality.Valid {
		value := quality.String
		preferences.Quality = &value
	}
	return &preferences, nil
}

func (a *API) itemExists(ctx context.Context, itemID int64) (bool, error) {
	var exists int
	err := a.db.QueryRowContext(ctx,
		`SELECT 1 FROM media_items WHERE id = ?`, itemID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (a *API) handleItemPreferencesGet(w http.ResponseWriter, r *http.Request) {
	user, ok := sessionUserFromRequest(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "falta el token de sesión")
		return
	}
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	exists, err := a.itemExists(r.Context(), id)
	if err != nil {
		log.Printf("library: comprobar el ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if !exists {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	preferences, err := loadItemPreferences(r.Context(), a.db, user.ID, id)
	if err != nil {
		log.Printf("library: leer preferencias del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if preferences == nil {
		preferences = &itemPreferencesDTO{}
	}
	writeJSON(w, http.StatusOK, preferences)
}

func (a *API) handleItemPreferencesPut(w http.ResponseWriter, r *http.Request) {
	user, ok := sessionUserFromRequest(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "falta el token de sesión")
		return
	}
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	exists, err := a.itemExists(r.Context(), id)
	if err != nil {
		log.Printf("library: comprobar el ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if !exists {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}

	var patch map[string]json.RawMessage
	if err := decodeJSONBody(w, r, &patch); err != nil {
		writeJSONBodyError(w, err)
		return
	}
	if patch == nil {
		writeJSONError(w, http.StatusBadRequest, "cuerpo no válido")
		return
	}
	current, err := loadItemPreferences(r.Context(), a.db, user.ID, id)
	if err != nil {
		log.Printf("library: leer preferencias del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if current == nil {
		current = &itemPreferencesDTO{}
	}
	for key, raw := range patch {
		switch key {
		case "audio_index":
			if string(raw) == "null" {
				current.AudioIndex = nil
				continue
			}
			var value int
			if err := json.Unmarshal(raw, &value); err != nil || value < 0 {
				writeJSONError(w, http.StatusBadRequest, "audio_index no válido")
				return
			}
			current.AudioIndex = &value
		case "subtitle_id":
			if string(raw) == "null" {
				current.SubtitleID = nil
				continue
			}
			var value string
			if err := json.Unmarshal(raw, &value); err != nil ||
				(value != "" && !subtitlePreferencePattern.MatchString(value)) {
				writeJSONError(w, http.StatusBadRequest, "subtitle_id no válido")
				return
			}
			current.SubtitleID = &value
		case "quality":
			if string(raw) == "null" {
				current.Quality = nil
				continue
			}
			var value string
			if err := json.Unmarshal(raw, &value); err != nil || !validQuality(value) {
				writeJSONError(w, http.StatusBadRequest, "quality no válida")
				return
			}
			current.Quality = &value
		default:
			writeJSONError(w, http.StatusBadRequest, "campo desconocido")
			return
		}
	}

	_, err = a.db.ExecContext(r.Context(), `
		INSERT INTO item_preferences (user_id, media_item_id, audio_index, subtitle_id, quality)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (user_id, media_item_id) DO UPDATE SET
			audio_index = excluded.audio_index,
			subtitle_id = excluded.subtitle_id,
			quality = excluded.quality,
			updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')`,
		user.ID, id,
		nullableIntPtr(current.AudioIndex),
		nullableStringPtr(current.SubtitleID),
		nullableStringPtr(current.Quality))
	if err != nil {
		log.Printf("library: guardar preferencias del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleItemPreferencesDelete(w http.ResponseWriter, r *http.Request) {
	user, ok := sessionUserFromRequest(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "falta el token de sesión")
		return
	}
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	exists, err := a.itemExists(r.Context(), id)
	if err != nil {
		log.Printf("library: comprobar el ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if !exists {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	if _, err := a.db.ExecContext(r.Context(),
		`DELETE FROM item_preferences WHERE user_id = ? AND media_item_id = ?`,
		user.ID, id); err != nil {
		log.Printf("library: borrar preferencias del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func nullableIntPtr(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableStringPtr(value *string) any {
	if value == nil {
		return nil
	}
	return strings.TrimSpace(*value)
}
