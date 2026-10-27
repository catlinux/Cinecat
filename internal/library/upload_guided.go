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
	"time"
)

type uploadRootTarget struct {
	LibraryID   int64  `json:"library_id"`
	LibraryName string `json:"library_name"`
	LibraryKind string `json:"library_kind"`
	Path        string `json:"path"`
	MovieCount  int64  `json:"movie_count"`
	SeriesCount int64  `json:"series_count"`
}

type uploadTargetsResponse struct {
	Allowed bool               `json:"allowed"`
	Roots   []uploadRootTarget `json:"roots"`
}

type guidedUploadResult struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Season  int    `json:"season"`
	Episode int    `json:"episode"`
}

type guidedEpisode struct {
	path   string
	season sql.NullInt64
}

type guidedSeries struct {
	libraryID   int64
	folder      string
	libraryPath string
	episodes    []guidedEpisode
}

// handleUploadTargets enumera las raíces donde la web puede ofrecer subidas.
func (a *API) handleUploadTargets(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.QueryContext(r.Context(), `
		SELECT id, name, kind, path
		  FROM libraries
		 WHERE kind IN ('video', 'clips')
		 ORDER BY name COLLATE NOCASE, id`)
	if err != nil {
		log.Printf("library: listar destinos de subida: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer rows.Close()

	type libraryInfo struct {
		id   int64
		name string
		kind string
		path string
	}
	libraries := make([]libraryInfo, 0)
	for rows.Next() {
		var library libraryInfo
		if err := rows.Scan(&library.id, &library.name, &library.kind, &library.path); err != nil {
			log.Printf("library: leer biblioteca de subida: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		libraries = append(libraries, library)
	}
	if err := rows.Err(); err != nil {
		log.Printf("library: recorrer bibliotecas de subida: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	targets := make([]uploadRootTarget, 0)
	index := make(map[string]int)
	rootsByLibrary := make(map[int64][]string)
	for _, library := range libraries {
		roots, err := a.uploadLibraryRoots(r.Context(), library.id, library.path)
		if err != nil {
			log.Printf("library: leer raíces de la biblioteca %d: %v", library.id, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		sort.Strings(roots)
		rootsByLibrary[library.id] = roots
		for _, root := range roots {
			if strings.TrimSpace(root) == "" {
				continue
			}
			index[uploadRootKey(library.id, root)] = len(targets)
			targets = append(targets, uploadRootTarget{
				LibraryID:   library.id,
				LibraryName: library.name,
				LibraryKind: library.kind,
				Path:        root,
			})
		}
	}

	movieRows, err := a.db.QueryContext(r.Context(), `
		SELECT library_id, path
		  FROM media_items
		 WHERE kind = 'movie' AND path IS NOT NULL AND trim(path) <> ''`)
	if err != nil {
		log.Printf("library: contar películas de destinos: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	for movieRows.Next() {
		var libraryID int64
		var path string
		if err := movieRows.Scan(&libraryID, &path); err != nil {
			movieRows.Close()
			log.Printf("library: leer película de destinos: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		if root := rootForPath(rootsByLibrary[libraryID], path); root != "" {
			targets[index[uploadRootKey(libraryID, root)]].MovieCount++
		}
	}
	if err := movieRows.Close(); err != nil {
		log.Printf("library: cerrar películas de destinos: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	seriesRows, err := a.db.QueryContext(r.Context(), `
		SELECT s.library_id,
		       (SELECT mi.path
		          FROM media_items mi
		         WHERE mi.series_id = s.id
		           AND mi.path IS NOT NULL
		           AND trim(mi.path) <> ''
		         ORDER BY mi.id
		         LIMIT 1)
		  FROM series s`)
	if err != nil {
		log.Printf("library: contar series de destinos: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	for seriesRows.Next() {
		var libraryID int64
		var path sql.NullString
		if err := seriesRows.Scan(&libraryID, &path); err != nil {
			seriesRows.Close()
			log.Printf("library: leer serie de destinos: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		if !path.Valid {
			continue
		}
		if root := rootForPath(rootsByLibrary[libraryID], path.String); root != "" {
			targets[index[uploadRootKey(libraryID, root)]].SeriesCount++
		}
	}
	if err := seriesRows.Close(); err != nil {
		log.Printf("library: cerrar series de destinos: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	sort.SliceStable(targets, func(i, j int) bool {
		left := strings.ToLower(targets[i].LibraryName)
		right := strings.ToLower(targets[j].LibraryName)
		if left != right {
			return left < right
		}
		return filepath.Clean(targets[i].Path) < filepath.Clean(targets[j].Path)
	})

	writeJSON(w, http.StatusOK, uploadTargetsResponse{
		Allowed: a.allowFileChanges(),
		Roots:   targets,
	})
}

func uploadRootKey(libraryID int64, root string) string {
	return strconv.FormatInt(libraryID, 10) + "\x00" + filepath.Clean(root)
}

func rootForPath(roots []string, path string) string {
	best := ""
	for _, root := range roots {
		clean := filepath.Clean(root)
		if pathWithin(clean, path) && len(clean) > len(best) {
			best = clean
		}
	}
	return best
}

// handleSeriesFileUpload guarda un capítulo o subtítulo en la carpeta que
// corresponde a la serie y temporada, sin sobrescribir nada.
func (a *API) handleSeriesFileUpload(w http.ResponseWriter, r *http.Request) {
	_ = http.NewResponseController(w).SetReadDeadline(time.Time{})

	if !a.allowFileChanges() {
		writeUploadError(w, errTrashDisabled)
		return
	}
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "serie no encontrada")
		return
	}
	season, ok := parseUploadNumber(r.URL.Query().Get("season"), 0, 99)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "season debe ser un entero entre 0 y 99")
		return
	}
	episode, ok := parseUploadNumber(r.URL.Query().Get("episode"), 1, 999)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "episode debe ser un entero entre 1 y 999")
		return
	}
	extension, ok := guidedUploadExtension(r.URL.Query().Get("name"))
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "extensión de archivo no válida")
		return
	}

	series, err := a.guidedSeriesUploadInfo(r.Context(), id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "serie no encontrada")
		return
	case err != nil:
		log.Printf("library: leer serie %d para subida: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	seriesDir, err := a.seriesUploadDir(r.Context(), series)
	if err != nil {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}
	destination, err := guidedSeasonDir(seriesDir, series.episodes, season)
	if err != nil {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}
	if err := ensureUploadDir(destination); err != nil {
		writeUploadError(w, err)
		return
	}

	finalName := fmt.Sprintf("%s S%02dE%02d%s", series.folder, season, episode, extension)
	if err := validateUploadName(finalName, true); err != nil {
		writeJSONError(w, http.StatusBadRequest, "el nombre del archivo no es válido")
		return
	}
	target := uploadTarget{
		Dir:       seriesDir,
		FinalDir:  destination,
		Name:      finalName,
		FinalPath: filepath.Join(destination, finalName),
	}
	result, err := a.receiveUpload(w, r, target)
	if err != nil {
		writeUploadError(w, err)
		return
	}
	log.Printf("library: subida guiada: %s subió %s (%d bytes)",
		trashActor(r), result.Path, result.Size)
	a.startUploadScan()
	writeJSON(w, http.StatusCreated, guidedUploadResult{
		Path:    result.Path,
		Size:    result.Size,
		Season:  season,
		Episode: episode,
	})
}

func (a *API) guidedSeriesUploadInfo(ctx context.Context, id int64) (guidedSeries, error) {
	var series guidedSeries
	err := a.db.QueryRowContext(ctx, `
		SELECT s.library_id, s.folder, l.path
		  FROM series s
		  JOIN libraries l ON l.id = s.library_id
		 WHERE s.id = ?`, id).Scan(&series.libraryID, &series.folder, &series.libraryPath)
	if err != nil {
		return guidedSeries{}, err
	}

	rows, err := a.db.QueryContext(ctx, `
		SELECT path, season
		  FROM media_items
		 WHERE series_id = ? AND path IS NOT NULL AND trim(path) <> ''
		 ORDER BY id`, id)
	if err != nil {
		return guidedSeries{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var episode guidedEpisode
		if err := rows.Scan(&episode.path, &episode.season); err != nil {
			return guidedSeries{}, err
		}
		series.episodes = append(series.episodes, episode)
	}
	if err := rows.Err(); err != nil {
		return guidedSeries{}, err
	}
	return series, nil
}

func (a *API) seriesUploadDir(ctx context.Context, series guidedSeries) (string, error) {
	if len(series.episodes) == 0 {
		return "", errors.New("la serie no tiene episodios con ruta; no se puede deducir su carpeta")
	}
	if !validSeriesFolder(series.folder) {
		return "", errors.New("la carpeta de la serie no es válida")
	}
	roots, err := a.uploadLibraryRoots(ctx, series.libraryID, series.libraryPath)
	if err != nil {
		return "", err
	}
	root := ""
	for _, episode := range series.episodes {
		if candidate := rootForPath(roots, episode.path); candidate != "" {
			root = candidate
			break
		}
	}
	if root == "" {
		return "", errors.New("no se pudo deducir la raíz de la serie a partir de sus episodios")
	}
	root = filepath.Clean(root)
	seriesDir := filepath.Join(root, series.folder)
	if !pathWithin(root, seriesDir) || filepath.Clean(seriesDir) == root {
		return "", errors.New("la carpeta de la serie queda fuera de su biblioteca")
	}
	info, err := os.Lstat(seriesDir)
	if err != nil {
		return "", errors.New("la carpeta de la serie no existe")
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", errors.New("la carpeta de la serie no es una carpeta válida")
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", errors.New("la raíz de la serie no es válida")
	}
	resolvedSeries, err := filepath.EvalSymlinks(seriesDir)
	if err != nil || !pathWithin(resolvedRoot, resolvedSeries) {
		return "", errors.New("la carpeta de la serie queda fuera de su biblioteca")
	}
	return filepath.Clean(seriesDir), nil
}

func guidedSeasonDir(seriesDir string, episodes []guidedEpisode, season int) (string, error) {
	counts := make(map[string]int)
	hasSubfolder := false
	for _, episode := range episodes {
		dir := filepath.Clean(filepath.Dir(episode.path))
		if dir != seriesDir && pathWithin(seriesDir, dir) {
			hasSubfolder = true
		}
		if episode.season.Valid && int(episode.season.Int64) == season &&
			(dir == seriesDir || pathWithin(seriesDir, dir)) {
			counts[dir]++
		}
	}
	if len(counts) > 0 {
		best := ""
		bestCount := -1
		for dir, count := range counts {
			if count > bestCount || (count == bestCount && (best == "" || dir < best)) {
				best = dir
				bestCount = count
			}
		}
		return best, nil
	}
	if hasSubfolder {
		return filepath.Join(seriesDir, fmt.Sprintf("Temporada %d", season)), nil
	}
	return seriesDir, nil
}

func validSeriesFolder(folder string) bool {
	if folder == "" || folder == "." || folder == ".." ||
		filepath.IsAbs(folder) || filepath.VolumeName(folder) != "" ||
		filepath.Base(folder) != folder || strings.ContainsAny(folder, `/\`) {
		return false
	}
	for _, char := range folder {
		if char < 32 {
			return false
		}
	}
	return true
}

func parseUploadNumber(raw string, min, max int) (int, bool) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < min || value > max {
		return 0, false
	}
	return value, true
}

func guidedUploadExtension(name string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" || (!IsVideoFile(name) && !isSubtitleFile(name)) {
		return "", false
	}
	if isSubtitleFile(name) {
		stem := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
		language := strings.TrimPrefix(filepath.Ext(stem), ".")
		if len(language) >= 2 && len(language) <= 3 && asciiLetters(language) {
			return "." + strings.ToLower(language) + ext, true
		}
	}
	return ext, true
}

func asciiLetters(value string) bool {
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') {
			return false
		}
	}
	return value != ""
}
