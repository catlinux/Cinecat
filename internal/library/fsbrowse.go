package library

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxFSEntries = 1000

type fsEntryDTO struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type fsRootsResponse struct {
	Roots []fsEntryDTO `json:"roots"`
}

type fsBrowseResponse struct {
	Path      string       `json:"path"`
	Parent    *string      `json:"parent"`
	Entries   []fsEntryDTO `json:"entries"`
	Truncated bool         `json:"truncated"`
}

func (a *API) handleFS(w http.ResponseWriter, r *http.Request) {
	rawPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if rawPath == "" {
		roots := a.allowedMediaRoots()
		entries := make([]fsEntryDTO, 0, len(roots))
		for _, root := range roots {
			entries = append(entries, fsEntryDTO{Name: filepath.Base(root), Path: root})
		}
		sort.Slice(entries, func(i, j int) bool {
			return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
		})
		writeJSON(w, http.StatusOK, fsRootsResponse{Roots: entries})
		return
	}
	if hasParentComponent(rawPath) {
		writeJSONError(w, http.StatusBadRequest, "la ruta no puede contener «..»")
		return
	}
	clean := filepath.Clean(rawPath)
	if !filepath.IsAbs(clean) {
		writeJSONError(w, http.StatusBadRequest, "la ruta debe ser absoluta")
		return
	}
	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "la ruta no existe o no es accesible")
		return
	}
	roots := a.allowedMediaRoots()
	if !pathWithinAny(roots, resolved) {
		writeJSONError(w, http.StatusForbidden, "la ruta está fuera de las raíces permitidas")
		return
	}
	info, err := os.Stat(resolved)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "la ruta no existe o no es accesible")
		return
	}
	if !info.IsDir() {
		writeJSONError(w, http.StatusBadRequest, "la ruta no es un directorio")
		return
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "la carpeta no se puede leer")
		return
	}

	directories := make([]fsEntryDTO, 0, len(entries))
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.EqualFold(name, ".cinecat-papelera") {
			continue
		}
		directories = append(directories, fsEntryDTO{
			Name: name,
			Path: filepath.Join(clean, name),
		})
	}
	sort.Slice(directories, func(i, j int) bool {
		left := strings.ToLower(directories[i].Name)
		right := strings.ToLower(directories[j].Name)
		if left == right {
			return directories[i].Name < directories[j].Name
		}
		return left < right
	})
	truncated := len(directories) > maxFSEntries
	if truncated {
		directories = directories[:maxFSEntries]
	}
	var parent *string
	if !isMediaRoot(roots, resolved) {
		value := filepath.Dir(clean)
		parent = &value
	}
	writeJSON(w, http.StatusOK, fsBrowseResponse{
		Path:      clean,
		Parent:    parent,
		Entries:   directories,
		Truncated: truncated,
	})
}

func isMediaRoot(roots []string, path string) bool {
	for _, root := range roots {
		if filepath.Clean(root) == filepath.Clean(path) {
			return true
		}
	}
	return false
}
