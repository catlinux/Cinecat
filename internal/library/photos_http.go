package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/catlinux/cinecat/internal/playback"
)

type photoDTO struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	TakenAt *int64 `json:"taken_at"`
	Size    int64  `json:"size"`
}

type photosResponse struct {
	Folders []string   `json:"folders"`
	Items   []photoDTO `json:"items"`
	Total   int64      `json:"total"`
}

type photoFileRef struct {
	path        string
	libraryID   int64
	libraryRoot string
}

func (a *API) handlePhotos(w http.ResponseWriter, r *http.Request) {
	rawLibraryID := strings.TrimSpace(r.URL.Query().Get("library_id"))
	libraryID, err := strconv.ParseInt(rawLibraryID, 10, 64)
	if err != nil || libraryID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "library_id debe ser un entero positivo")
		return
	}
	access := accessFromRequest(r)
	var kind string
	err = a.db.QueryRowContext(r.Context(), `
		SELECT l.kind FROM libraries l
		 WHERE l.id = ? AND `+access.LibraryClause("l"), libraryID).Scan(&kind)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
		return
	case err != nil:
		log.Printf("library: leer biblioteca de fotos %d: %v", libraryID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	case kind != "photos":
		writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
		return
	}

	parts, err := clipRelativeParts(r.URL.Query().Get("folder"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	folder := strings.Join(parts, "/")
	order := " ORDER BY COALESCE(taken_at, mtime) DESC, name COLLATE NOCASE, id"
	switch strings.TrimSpace(r.URL.Query().Get("sort")) {
	case "", "date":
	case "name":
		order = " ORDER BY name COLLATE NOCASE, id"
	default:
		writeJSONError(w, http.StatusBadRequest, "sort debe ser date o name")
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	var total int64
	if err := a.db.QueryRowContext(r.Context(),
		`SELECT count(*) FROM photos WHERE library_id = ? AND folder = ?`,
		libraryID, folder).Scan(&total); err != nil {
		log.Printf("library: contar fotos de la biblioteca %d: %v", libraryID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	rows, err := a.db.QueryContext(r.Context(), `
		SELECT id, name, width, height, taken_at, size
		  FROM photos
		 WHERE library_id = ? AND folder = ?`+order+` LIMIT ? OFFSET ?`,
		libraryID, folder, limit, offset)
	if err != nil {
		log.Printf("library: listar fotos de la biblioteca %d: %v", libraryID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer rows.Close()
	items := make([]photoDTO, 0)
	for rows.Next() {
		var item photoDTO
		var takenAt sql.NullInt64
		if err := rows.Scan(&item.ID, &item.Name, &item.Width, &item.Height, &takenAt, &item.Size); err != nil {
			log.Printf("library: leer foto de la biblioteca %d: %v", libraryID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		item.TakenAt = int64Pointer(takenAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("library: recorrer fotos de la biblioteca %d: %v", libraryID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	folders, err := loadPhotoFolders(r.Context(), a.db, libraryID, folder)
	if err != nil {
		log.Printf("library: listar carpetas de fotos de la biblioteca %d: %v", libraryID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, photosResponse{
		Folders: folders,
		Items:   items,
		Total:   total,
	})
}

func loadPhotoFolders(ctx context.Context, db *sql.DB, libraryID int64, folder string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT DISTINCT folder FROM photos
		 WHERE library_id = ?
		 ORDER BY folder COLLATE NOCASE`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	prefix := ""
	if folder != "" {
		prefix = folder + "/"
	}
	seen := make(map[string]bool)
	folders := make([]string, 0)
	for rows.Next() {
		var current string
		if err := rows.Scan(&current); err != nil {
			return nil, err
		}
		if current == folder || !strings.HasPrefix(current, prefix) {
			continue
		}
		rest := strings.TrimPrefix(current, prefix)
		if rest == "" {
			continue
		}
		name := rest
		if slash := strings.IndexByte(rest, '/'); slash >= 0 {
			name = rest[:slash]
		}
		if name != "" && !seen[name] {
			seen[name] = true
			folders = append(folders, name)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(folders, func(i, j int) bool {
		left := strings.ToLower(folders[i])
		right := strings.ToLower(folders[j])
		if left != right {
			return left < right
		}
		return folders[i] < folders[j]
	})
	return folders, nil
}

func (a *API) loadPhotoFile(ctx context.Context, access libraryAccess, id int64) (photoFileRef, error) {
	var ref photoFileRef
	err := a.db.QueryRowContext(ctx, `
		SELECT p.path, l.id, l.path
		  FROM photos p
		  JOIN libraries l ON l.id = p.library_id
		 WHERE p.id = ? AND l.kind = 'photos'
		   AND `+access.PhotosClause("p"), id).
		Scan(&ref.path, &ref.libraryID, &ref.libraryRoot)
	if err != nil {
		return photoFileRef{}, err
	}
	ref.libraryRoot = a.libraryRootFor(ctx, ref.libraryID, ref.path, ref.libraryRoot)
	return ref, nil
}

func (a *API) handlePhotoOriginal(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "foto no encontrada")
		return
	}
	ref, err := a.loadPhotoFile(r.Context(), accessFromRequest(r), id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "foto no encontrada")
		return
	case err != nil:
		log.Printf("library: leer foto %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	safePath, err := playback.SafePath(ref.libraryRoot, ref.path)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}
	file, err := os.Open(safePath)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("Content-Type", photoMIME(safePath))
	http.ServeContent(w, r, filepath.Base(safePath), info.ModTime(), file)
}

func (a *API) handlePhotoThumb(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "foto no encontrada")
		return
	}
	ref, err := a.loadPhotoFile(r.Context(), accessFromRequest(r), id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "foto no encontrada")
		return
	case err != nil:
		log.Printf("library: leer foto %d para miniatura: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	safePath, err := playback.SafePath(ref.libraryRoot, ref.path)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}
	stat, err := os.Stat(safePath)
	if err != nil || stat.IsDir() {
		writeJSONError(w, http.StatusNotFound, "archivo no encontrado")
		return
	}
	if a.photoThumbnails == nil {
		writeJSONError(w, http.StatusNotFound, "miniatura no disponible")
		return
	}
	thumbnailPath, err := a.photoThumbnails.Get(r.Context(), id, safePath, stat.ModTime().UnixNano())
	if err != nil {
		if errors.Is(err, ErrThumbnailBusy) {
			w.Header().Set("Retry-After", "5")
			writeJSONError(w, http.StatusServiceUnavailable, "miniatura ocupada, inténtalo de nuevo")
			return
		}
		log.Printf("library: miniatura de la foto %d: %v", id, err)
		writeJSONError(w, http.StatusNotFound, "miniatura no disponible")
		return
	}
	servePhotoThumbnail(w, r, thumbnailPath)
}

func servePhotoThumbnail(w http.ResponseWriter, r *http.Request, path string) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		writeJSONError(w, http.StatusNotFound, "miniatura no disponible")
		return
	}
	etag := fmt.Sprintf(`"%x-%x"`, info.ModTime().UnixNano(), info.Size())
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("ETag", etag)
	w.Header().Set("Content-Type", "image/jpeg")
	if matchesETag(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	file, err := os.Open(path)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "miniatura no disponible")
		return
	}
	defer file.Close()
	http.ServeContent(w, r, "thumbnail.jpg", info.ModTime(), file)
}
