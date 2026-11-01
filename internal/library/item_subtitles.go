package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/catlinux/cinecat/internal/playback"
)

const (
	maxSubtitleUploadSize = 2 << 20
	maxSubtitleUploadBody = maxSubtitleUploadSize + (1 << 20)
)

var subtitleLanguagePattern = regexp.MustCompile(`^[A-Za-z]{2,3}$`)

type uploadedSubtitleResponse struct {
	ID       string `json:"id"`
	Language string `json:"language"`
	Label    string `json:"label"`
	Codec    string `json:"codec"`
}

// WithSubtitles fija el directorio donde se guardan los subtítulos subidos,
// fuera de las carpetas de vídeo.
func (a *API) WithSubtitles(dir string) *API {
	a.subtitlesDir = strings.TrimSpace(dir)
	return a
}

// subtitlesDirForDatabase deja los subtítulos junto al archivo SQLite, que es
// el volumen que ya existe en las instalaciones.
func subtitlesDirForDatabase(db *sql.DB) string {
	if db == nil {
		return ""
	}
	rows, err := db.Query(`PRAGMA database_list`)
	if err != nil {
		return ""
	}
	defer rows.Close()
	for rows.Next() {
		var sequence int
		var name, file string
		if err := rows.Scan(&sequence, &name, &file); err != nil {
			return ""
		}
		file = strings.TrimSpace(file)
		if name == "main" && file != "" && file != ":memory:" {
			return filepath.Join(filepath.Dir(file), "subtitles")
		}
	}
	return ""
}

func (a *API) uploadedSubtitleTracks(ctx context.Context, itemID int64) ([]playback.SubtitleTrack, error) {
	if a.subtitlesDir == "" {
		return nil, nil
	}
	rows, err := a.db.QueryContext(ctx, `
		SELECT id, language, label, filename
		  FROM item_subtitles
		 WHERE media_item_id = ?
		 ORDER BY id`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tracks := make([]playback.SubtitleTrack, 0)
	dir := filepath.Join(a.subtitlesDir, strconv.FormatInt(itemID, 10))
	for rows.Next() {
		var id int64
		var language, label, filename string
		if err := rows.Scan(&id, &language, &label, &filename); err != nil {
			return nil, err
		}
		tracks = append(tracks, playback.SubtitleTrack{
			ID:       fmt.Sprintf("u%d", id),
			Source:   "uploaded",
			Language: language,
			Title:    label,
			Codec:    playback.SubtitleCodec(filepath.Ext(filename)),
			Path:     filepath.Join(dir, filepath.Base(filename)),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tracks, nil
}

func (a *API) uploadedSubtitleTrack(ctx context.Context, itemID int64, subtitleID string) (*playback.SubtitleTrack, error) {
	tracks, err := a.uploadedSubtitleTracks(ctx, itemID)
	if err != nil {
		return nil, err
	}
	for _, track := range tracks {
		if track.ID == subtitleID {
			copy := track
			return &copy, nil
		}
	}
	return nil, nil
}

func (a *API) handleItemSubtitleUpload(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	if _, ok := a.requireVisibleItem(w, r, id); !ok {
		return
	}
	if a.subtitlesDir == "" {
		writeJSONError(w, http.StatusServiceUnavailable, "la subida de subtítulos no está configurada")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxSubtitleUploadBody)
	if err := r.ParseMultipartForm(maxSubtitleUploadSize); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeJSONError(w, http.StatusRequestEntityTooLarge, "el subtítulo supera los 2 MB")
		} else {
			writeJSONError(w, http.StatusBadRequest, "petición multipart no válida")
		}
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "falta el archivo del subtítulo")
		return
	}
	defer file.Close()

	extension := strings.ToLower(filepath.Ext(header.Filename))
	if extension != ".srt" && extension != ".vtt" && extension != ".ass" {
		writeJSONError(w, http.StatusBadRequest, "solo se admiten archivos .srt, .vtt y .ass")
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, maxSubtitleUploadSize+1))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "no se pudo leer el archivo del subtítulo")
		return
	}
	if len(data) > maxSubtitleUploadSize {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "el subtítulo supera los 2 MB")
		return
	}
	normalized, err := playback.NormalizeSubtitleText(data)
	if errors.Is(err, playback.ErrSubtitleEmpty) {
		writeJSONError(w, http.StatusBadRequest, "el archivo del subtítulo está vacío")
		return
	}
	if errors.Is(err, playback.ErrSubtitleBinary) {
		writeJSONError(w, http.StatusBadRequest, "el archivo no parece un subtítulo de texto")
		return
	}
	if err != nil {
		log.Printf("library: validar subtítulo del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	language := strings.TrimSpace(r.FormValue("language"))
	if language != "" && !subtitleLanguagePattern.MatchString(language) {
		writeJSONError(w, http.StatusBadRequest, "el idioma debe ser un código de 2 o 3 letras")
		return
	}
	label := strings.TrimSpace(r.FormValue("label"))

	rowID, err := a.storeUploadedSubtitle(r.Context(), id, extension, normalized, language, label)
	if err != nil {
		log.Printf("library: guardar subtítulo del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusCreated, uploadedSubtitleResponse{
		ID:       fmt.Sprintf("u%d", rowID),
		Language: language,
		Label:    label,
		Codec:    playback.SubtitleCodec(extension),
	})
}

func (a *API) storeUploadedSubtitle(
	ctx context.Context,
	itemID int64,
	extension string,
	data []byte,
	language, label string,
) (int64, error) {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO item_subtitles (media_item_id, language, label, filename)
		VALUES (?, ?, ?, '')`, itemID, language, label)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	filename := strconv.FormatInt(id, 10) + extension
	if _, err := tx.ExecContext(ctx,
		`UPDATE item_subtitles SET filename = ? WHERE id = ?`, filename, id); err != nil {
		return 0, err
	}

	dir := filepath.Join(a.subtitlesDir, strconv.FormatInt(itemID, 10))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return 0, err
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		_ = os.Remove(path)
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		_ = os.Remove(path)
		return 0, err
	}
	return id, nil
}

func (a *API) handleItemSubtitleDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	if _, ok := a.requireVisibleItem(w, r, id); !ok {
		return
	}
	subtitleRowID, ok := subtitleRowIDFromPath(r.PathValue("subId"))
	if !ok {
		writeJSONError(w, http.StatusNotFound, "subtítulo no encontrado")
		return
	}

	var filename string
	err := a.db.QueryRowContext(r.Context(), `
		SELECT filename
		  FROM item_subtitles
		 WHERE id = ? AND media_item_id = ?`, subtitleRowID, id).Scan(&filename)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "subtítulo no encontrado")
		return
	case err != nil:
		log.Printf("library: leer subtítulo %d del ítem %d: %v", subtitleRowID, id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	if _, err := a.db.ExecContext(r.Context(),
		`DELETE FROM item_subtitles WHERE id = ? AND media_item_id = ?`, subtitleRowID, id); err != nil {
		log.Printf("library: borrar subtítulo %d del ítem %d: %v", subtitleRowID, id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if a.subtitlesDir != "" {
		dir := filepath.Join(a.subtitlesDir, strconv.FormatInt(id, 10))
		if err := os.Remove(filepath.Join(dir, filepath.Base(filename))); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("library: borrar archivo del subtítulo %d del ítem %d: %v", subtitleRowID, id, err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func subtitleRowIDFromPath(value string) (int64, bool) {
	value = strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(value), "u"), "U")
	id, err := strconv.ParseInt(value, 10, 64)
	return id, err == nil && id > 0
}
