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
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	defaultMediaRoot = "/media"
	maxLibraryName   = 80
)

type libraryPathDTO struct {
	ID   int64  `json:"id"`
	Path string `json:"path,omitempty"`
}

type libraryDTO struct {
	ID        int64            `json:"id"`
	Name      string           `json:"name"`
	Kind      string           `json:"kind"`
	Path      string           `json:"path,omitempty"`
	Paths     []libraryPathDTO `json:"paths,omitempty"`
	ItemCount int64            `json:"item_count"`
}

type librariesResponse struct {
	Libraries []libraryDTO `json:"libraries"`
}

type createLibraryRequest struct {
	Name  string   `json:"name"`
	Kind  string   `json:"kind"`
	Path  string   `json:"path"`
	Paths []string `json:"paths"`
}

type updateLibraryRequest struct {
	Name string `json:"name"`
}

type libraryPathRequest struct {
	Path string `json:"path"`
}

// WithMediaRoots sustituye las raíces permitidas para el navegador y las
// rutas de biblioteca.
func (a *API) WithMediaRoots(roots []string) *API {
	filtered := make([]string, 0, len(roots))
	for _, root := range roots {
		if root = strings.TrimSpace(root); root != "" {
			filtered = append(filtered, root)
		}
	}
	if len(filtered) == 0 {
		filtered = defaultMediaRoots()
	}
	a.mediaRoots = filtered
	return a
}

func defaultMediaRoots() []string {
	raw := strings.TrimSpace(os.Getenv("CINECAT_MEDIA_ROOTS"))
	if raw == "" {
		return []string{defaultMediaRoot}
	}
	roots := make([]string, 0)
	for _, root := range filepath.SplitList(raw) {
		if root = strings.TrimSpace(root); root != "" {
			roots = append(roots, root)
		}
	}
	if len(roots) == 0 {
		return []string{defaultMediaRoot}
	}
	return roots
}

func (a *API) allowedMediaRoots() []string {
	roots := a.mediaRoots
	if len(roots) == 0 {
		roots = defaultMediaRoots()
	}
	result := make([]string, 0, len(roots))
	for _, root := range roots {
		clean := filepath.Clean(strings.TrimSpace(root))
		if !filepath.IsAbs(clean) {
			continue
		}
		resolved, err := filepath.EvalSymlinks(clean)
		if err != nil {
			continue
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			continue
		}
		result = append(result, resolved)
	}
	return result
}

func hasParentComponent(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func pathWithin(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if parent == child {
		return true
	}
	rel, err := filepath.Rel(parent, child)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func pathsOverlap(first, second string) bool {
	return pathWithin(first, second) || pathWithin(second, first)
}

type checkedLibraryPath struct {
	clean    string
	resolved string
}

func (a *API) checkLibraryPath(path string) (checkedLibraryPath, error) {
	raw := strings.TrimSpace(path)
	if raw == "" {
		return checkedLibraryPath{}, errors.New("la ruta no puede estar vacía")
	}
	if hasParentComponent(raw) {
		return checkedLibraryPath{}, errors.New("la ruta no puede contener «..»")
	}
	clean := filepath.Clean(raw)
	if !filepath.IsAbs(clean) {
		return checkedLibraryPath{}, errors.New("la ruta debe ser absoluta")
	}
	info, err := os.Stat(clean)
	if err != nil {
		return checkedLibraryPath{}, errors.New("la ruta no existe o no es accesible")
	}
	if !info.IsDir() {
		return checkedLibraryPath{}, errors.New("la ruta no es un directorio")
	}
	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return checkedLibraryPath{}, errors.New("la ruta no se puede resolver")
	}
	if !pathWithinAny(a.allowedMediaRoots(), resolved) {
		return checkedLibraryPath{}, errors.New("la ruta está fuera de las raíces permitidas")
	}
	return checkedLibraryPath{clean: clean, resolved: resolved}, nil
}

func pathWithinAny(roots []string, path string) bool {
	for _, root := range roots {
		if pathWithin(root, path) {
			return true
		}
	}
	return false
}

func normalizeLibraryKind(kind string) (string, error) {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return "video", nil
	}
	switch kind {
	case "video", "clips", "music":
		return kind, nil
	default:
		return "", errors.New("el tipo debe ser video, clips o music")
	}
}

func normalizeLibraryName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("el nombre no puede estar vacío")
	}
	if utf8.RuneCountInString(name) > maxLibraryName {
		return "", fmt.Errorf("el nombre no puede superar los %d caracteres", maxLibraryName)
	}
	return name, nil
}

func (a *API) checkLibraryPathFree(ctx context.Context, db *sql.DB, candidate checkedLibraryPath) error {
	existing, err := registeredLibraryPaths(ctx, db, 0)
	if err != nil {
		return err
	}
	for _, path := range existing {
		resolved := path
		if value, err := filepath.EvalSymlinks(path); err == nil {
			resolved = value
		}
		if pathsOverlap(candidate.clean, path) || pathsOverlap(candidate.resolved, resolved) {
			return errors.New("la carpeta ya está registrada o se solapa con otra biblioteca")
		}
	}
	return nil
}

func registeredLibraryPaths(ctx context.Context, db *sql.DB, excludeLibraryID int64) ([]string, error) {
	query := `SELECT path FROM library_paths`
	if excludeLibraryID > 0 {
		query += ` WHERE library_id <> ?`
	}
	query += ` UNION SELECT path FROM libraries`
	if excludeLibraryID > 0 {
		query += ` WHERE id <> ?`
	}
	args := []any{}
	if excludeLibraryID > 0 {
		args = append(args, excludeLibraryID, excludeLibraryID)
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	paths := make([]string, 0)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return paths, nil
}

func (a *API) handleLibraries(w http.ResponseWriter, r *http.Request) {
	libraries, err := a.listLibraries(r.Context())
	if err != nil {
		logLibraryError("listar bibliotecas", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if !isAdminRequest(r) {
		for i := range libraries {
			libraries[i].Path = ""
			libraries[i].Paths = nil
		}
	}
	writeJSON(w, http.StatusOK, librariesResponse{Libraries: libraries})
}

func (a *API) listLibraries(ctx context.Context) ([]libraryDTO, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT l.id, l.name, l.kind, l.path,
		       CASE WHEN l.kind = 'music'
		            THEN (SELECT count(*) FROM music_tracks mt WHERE mt.library_id = l.id)
		            ELSE (SELECT count(*) FROM media_items mi WHERE mi.library_id = l.id)
		       END
		  FROM libraries l
		 ORDER BY l.name COLLATE NOCASE, l.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	libraries := make([]libraryDTO, 0)
	byID := make(map[int64]int)
	for rows.Next() {
		var item libraryDTO
		if err := rows.Scan(&item.ID, &item.Name, &item.Kind, &item.Path, &item.ItemCount); err != nil {
			return nil, err
		}
		item.Paths = make([]libraryPathDTO, 0)
		libraries = append(libraries, item)
		byID[item.ID] = len(libraries) - 1
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	pathRows, err := a.db.QueryContext(ctx,
		`SELECT id, library_id, path FROM library_paths ORDER BY library_id, id`)
	if err != nil {
		return nil, err
	}
	defer pathRows.Close()
	for pathRows.Next() {
		var id, libraryID int64
		var path string
		if err := pathRows.Scan(&id, &libraryID, &path); err != nil {
			return nil, err
		}
		if index, ok := byID[libraryID]; ok {
			libraries[index].Paths = append(libraries[index].Paths,
				libraryPathDTO{ID: id, Path: path})
		}
	}
	if err := pathRows.Err(); err != nil {
		return nil, err
	}
	for index := range libraries {
		if len(libraries[index].Paths) == 0 {
			// Compatibilidad con filas antiguas que aún no tengan library_paths.
			libraries[index].Paths = append(libraries[index].Paths,
				libraryPathDTO{ID: 0, Path: libraries[index].Path})
		}
	}
	return libraries, nil
}

func (a *API) handleCreateLibrary(w http.ResponseWriter, r *http.Request) {
	var req createLibraryRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSONBodyError(w, err)
		return
	}
	name, err := normalizeLibraryName(req.Name)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	kind, err := normalizeLibraryKind(req.Kind)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	rawPaths := append([]string{}, req.Paths...)
	if strings.TrimSpace(req.Path) != "" {
		rawPaths = append(rawPaths, req.Path)
	}
	if len(rawPaths) == 0 {
		writeJSONError(w, http.StatusBadRequest, "la biblioteca debe tener al menos una carpeta")
		return
	}
	paths := make([]checkedLibraryPath, 0, len(rawPaths))
	for _, raw := range rawPaths {
		path, err := a.checkLibraryPath(raw)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		for _, previous := range paths {
			if pathsOverlap(path.clean, previous.clean) ||
				pathsOverlap(path.resolved, previous.resolved) {
				writeJSONError(w, http.StatusBadRequest, "hay carpetas repetidas o solapadas")
				return
			}
		}
		paths = append(paths, path)
	}
	for _, path := range paths {
		if err := a.checkLibraryPathFree(r.Context(), a.db, path); err != nil {
			writeJSONError(w, http.StatusConflict, err.Error())
			return
		}
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		logLibraryError("iniciar alta de biblioteca", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(r.Context(), `
		INSERT INTO libraries (name, path, kind) VALUES (?, ?, ?)`,
		name, paths[0].clean, kind)
	if err != nil {
		if isUniqueError(err) {
			writeJSONError(w, http.StatusConflict, "ya existe una biblioteca con esa carpeta")
			return
		}
		logLibraryError("crear biblioteca", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	libraryID, err := result.LastInsertId()
	if err != nil {
		logLibraryError("id de la biblioteca", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	for _, path := range paths {
		if _, err := tx.ExecContext(r.Context(), `
			INSERT INTO library_paths (library_id, path) VALUES (?, ?)`,
			libraryID, path.clean); err != nil {
			if isUniqueError(err) {
				writeJSONError(w, http.StatusConflict, "ya existe una biblioteca con esa carpeta")
				return
			}
			logLibraryError("guardar carpeta de biblioteca", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		logLibraryError("confirmar alta de biblioteca", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	item, err := a.libraryByID(r.Context(), libraryID)
	if err != nil {
		logLibraryError("leer biblioteca creada", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) handleUpdateLibrary(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
		return
	}
	var req updateLibraryRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSONBodyError(w, err)
		return
	}
	name, err := normalizeLibraryName(req.Name)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := a.db.ExecContext(r.Context(),
		`UPDATE libraries SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		logLibraryError("actualizar biblioteca", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
		return
	}
	item, err := a.libraryByID(r.Context(), id)
	if err != nil {
		logLibraryError("leer biblioteca actualizada", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) handleAddLibraryPath(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
		return
	}
	var exists int
	if err := a.db.QueryRowContext(r.Context(),
		`SELECT 1 FROM libraries WHERE id = ?`, id).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
		} else {
			logLibraryError("comprobar biblioteca", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
		}
		return
	}
	var req libraryPathRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSONBodyError(w, err)
		return
	}
	path, err := a.checkLibraryPath(req.Path)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.checkLibraryPathFree(r.Context(), a.db, path); err != nil {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}
	if _, err := a.db.ExecContext(r.Context(), `
		INSERT INTO library_paths (library_id, path) VALUES (?, ?)`, id, path.clean); err != nil {
		if isUniqueError(err) {
			writeJSONError(w, http.StatusConflict, "ya existe una biblioteca con esa carpeta")
			return
		}
		logLibraryError("añadir carpeta a biblioteca", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	item, err := a.libraryByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
			return
		}
		logLibraryError("leer biblioteca tras añadir carpeta", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) handleDeleteLibraryPath(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
		return
	}
	pathID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("pathId")), 10, 64)
	if err != nil || pathID <= 0 {
		writeJSONError(w, http.StatusNotFound, "carpeta no encontrada")
		return
	}
	paths, err := loadLibraryPathDTOs(r.Context(), a.db, id)
	if err != nil {
		logLibraryError("leer carpetas de biblioteca", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if len(paths) == 0 {
		writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
		return
	}
	var removed string
	found := false
	for _, path := range paths {
		if path.ID == pathID {
			removed = path.Path
			found = true
			break
		}
	}
	if !found {
		writeJSONError(w, http.StatusNotFound, "carpeta no encontrada")
		return
	}
	if len(paths) <= 1 {
		writeJSONError(w, http.StatusBadRequest, "la biblioteca debe conservar al menos una carpeta")
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		logLibraryError("iniciar retirada de carpeta", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(r.Context(),
		`SELECT id, path FROM media_items WHERE library_id = ?`, id)
	if err != nil {
		logLibraryError("leer ítems de la carpeta", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	itemIDs := make([]int64, 0)
	for rows.Next() {
		var itemID int64
		var itemPath string
		if err := rows.Scan(&itemID, &itemPath); err != nil {
			rows.Close()
			logLibraryError("leer ítem de la carpeta", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		if pathWithin(removed, itemPath) {
			itemIDs = append(itemIDs, itemID)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		logLibraryError("recorrer ítems de la carpeta", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	rows.Close()
	for _, itemID := range itemIDs {
		if _, err := tx.ExecContext(r.Context(),
			`DELETE FROM media_items WHERE id = ?`, itemID); err != nil {
			logLibraryError("borrar ítem de la carpeta", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
	}
	if err := deleteMusicTracksUnderPath(r.Context(), tx, id, removed); err != nil {
		logLibraryError("borrar pistas de la carpeta", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if _, err := tx.ExecContext(r.Context(),
		`DELETE FROM library_paths WHERE id = ? AND library_id = ?`, pathID, id); err != nil {
		logLibraryError("borrar carpeta de biblioteca", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	var current string
	if err := tx.QueryRowContext(r.Context(),
		`SELECT path FROM libraries WHERE id = ?`, id).Scan(&current); err != nil {
		logLibraryError("leer carpeta principal", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if current == removed {
		var next string
		if err := tx.QueryRowContext(r.Context(), `
			SELECT path FROM library_paths WHERE library_id = ? ORDER BY id LIMIT 1`, id).
			Scan(&next); err == nil {
			if _, err := tx.ExecContext(r.Context(),
				`UPDATE libraries SET path = ? WHERE id = ?`, next, id); err != nil {
				logLibraryError("actualizar carpeta principal", err)
				writeJSONError(w, http.StatusInternalServerError, "error interno")
				return
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			logLibraryError("leer siguiente carpeta principal", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
	}
	if err := cleanupOrphans(r.Context(), tx); err != nil {
		logLibraryError("limpiar huérfanos tras retirar carpeta", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if err := cleanupMusicOrphans(r.Context(), tx); err != nil {
		logLibraryError("limpiar huérfanos de música tras retirar carpeta", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if err := tx.Commit(); err != nil {
		logLibraryError("confirmar retirada de carpeta", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleDeleteLibrary(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
		return
	}
	result, err := a.db.ExecContext(r.Context(), `DELETE FROM libraries WHERE id = ?`, id)
	if err != nil {
		logLibraryError("borrar biblioteca", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
		return
	}
	if err := cleanupMusicOrphans(r.Context(), a.db); err != nil {
		logLibraryError("limpiar artistas tras borrar biblioteca", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) libraryByID(ctx context.Context, id int64) (libraryDTO, error) {
	var item libraryDTO
	err := a.db.QueryRowContext(ctx, `
		SELECT l.id, l.name, l.kind, l.path,
		       CASE WHEN l.kind = 'music'
		            THEN (SELECT count(*) FROM music_tracks mt WHERE mt.library_id = l.id)
		            ELSE (SELECT count(*) FROM media_items mi WHERE mi.library_id = l.id)
		       END
		  FROM libraries l WHERE l.id = ?`, id).
		Scan(&item.ID, &item.Name, &item.Kind, &item.Path, &item.ItemCount)
	if err != nil {
		return libraryDTO{}, err
	}
	item.Paths, err = loadLibraryPathDTOs(ctx, a.db, id)
	if err != nil {
		return libraryDTO{}, err
	}
	if len(item.Paths) == 0 {
		item.Paths = []libraryPathDTO{{ID: 0, Path: item.Path}}
	}
	return item, nil
}

func loadLibraryPathDTOs(ctx context.Context, db *sql.DB, libraryID int64) ([]libraryPathDTO, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, path FROM library_paths WHERE library_id = ? ORDER BY id`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	paths := make([]libraryPathDTO, 0)
	for rows.Next() {
		var path libraryPathDTO
		if err := rows.Scan(&path.ID, &path.Path); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return paths, nil
}

func loadLibraryPathStrings(ctx context.Context, db *sql.DB, libraryID int64, fallback string) ([]string, error) {
	paths, err := loadLibraryPathDTOs(ctx, db, libraryID)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return []string{fallback}, nil
	}
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		result = append(result, path.Path)
	}
	return result, nil
}

func isUniqueError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique")
}

func logLibraryError(action string, err error) {
	log.Printf("library: %s: %v", action, err)
}
