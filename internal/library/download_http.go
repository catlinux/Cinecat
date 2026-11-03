package library

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/catlinux/cinecat/internal/playback"
)

func (a *API) handleItemDownload(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	if _, ok := a.requireVisibleItem(w, r, id); !ok {
		return
	}

	media, err := a.mediaFile(r, id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	case err != nil:
		log.Printf("library: leer archivo del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if media.kind != KindMovie && media.kind != KindEpisode {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}

	safePath, err := playback.SafePath(media.libraryPath, media.path)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}
	file, err := os.Open(safePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
			return
		}
		log.Printf("library: abrir archivo del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		log.Printf("library: consultar archivo del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if !stat.Mode().IsRegular() {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}

	filename := filepath.Base(safePath)
	w.Header().Set("Content-Type", downloadContentType(filename))
	w.Header().Set("Content-Disposition", downloadDisposition(filename))
	http.ServeContent(w, r, filename, stat.ModTime(), file)
}

// downloadDisposition codifica el nombre según RFC 5987 para que los caracteres
// no ASCII y los separadores no rompan la cabecera.
func downloadDisposition(filename string) string {
	const allowed = "!#$&+-.^_`|~"
	var disposition strings.Builder
	disposition.WriteString("attachment; filename*=UTF-8''")
	for _, r := range filename {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
			strings.ContainsRune(allowed, r) {
			disposition.WriteRune(r)
			continue
		}
		for _, b := range []byte(string(r)) {
			fmt.Fprintf(&disposition, "%%%02X", b)
		}
	}
	return disposition.String()
}

func downloadContentType(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".mkv":
		return "video/x-matroska"
	case ".mp4", ".m4v":
		return "video/mp4"
	default:
		return "application/octet-stream"
	}
}
