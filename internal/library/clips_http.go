package library

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	defaultClipItemsLimit = 100
	maxClipItemsLimit     = 500
)

type clipRootDTO struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	ItemCount   int64  `json:"item_count"`
	CoverItemID *int64 `json:"cover_item_id"`
}

type clipRootsResponse struct {
	Roots []clipRootDTO `json:"roots"`
}

type clipLibraryDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type clipRootRefDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type clipBreadcrumbDTO struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type clipFolderDTO struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	ItemCount   int64  `json:"item_count"`
	CoverItemID *int64 `json:"cover_item_id"`
}

type clipBrowseResponse struct {
	Library clipLibraryDTO      `json:"library"`
	Root    clipRootRefDTO      `json:"root"`
	Path    string              `json:"path"`
	Parent  *string             `json:"parent"`
	Crumbs  []clipBreadcrumbDTO `json:"crumbs"`
	Folders []clipFolderDTO     `json:"folders"`
	Items   []mediaItemDTO      `json:"items"`
	Total   int64               `json:"total"`
	Limit   int                 `json:"limit"`
	Offset  int                 `json:"offset"`
}

type clipRoot struct {
	id          int64
	name        string
	path        string
	itemCount   int64
	coverItemID int64
}

type clipItemPath struct {
	id    int64
	path  string
	title string
}

func (a *API) handleClipBrowse(w http.ResponseWriter, r *http.Request) {
	libraryID, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
		return
	}
	if !accessFromRequest(r).Allows(libraryID) {
		writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
		return
	}

	var libraryName, libraryKind, libraryPath string
	err := a.db.QueryRowContext(r.Context(), `
		SELECT name, kind, path FROM libraries WHERE id = ?`, libraryID).
		Scan(&libraryName, &libraryKind, &libraryPath)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
		return
	case err != nil:
		log.Printf("library: leer biblioteca %d para navegar: %v", libraryID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	case libraryKind != "clips":
		writeJSONError(w, http.StatusNotFound, "biblioteca no encontrada")
		return
	}

	roots, err := loadClipRoots(r.Context(), a.db, libraryID, libraryPath)
	if err != nil {
		log.Printf("library: leer carpetas de la biblioteca %d: %v", libraryID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	rootParam := strings.TrimSpace(r.URL.Query().Get("root"))
	if rootParam == "" {
		entries := make([]clipRootDTO, 0, len(roots))
		for _, root := range roots {
			entries = append(entries, clipRootDTO{
				ID: root.id, Name: root.name, ItemCount: root.itemCount,
				CoverItemID: optionalItemID(root.coverItemID),
			})
		}
		writeJSON(w, http.StatusOK, clipRootsResponse{Roots: entries})
		return
	}

	rootID, err := strconv.ParseInt(rootParam, 10, 64)
	if err != nil || rootID < 0 {
		writeJSONError(w, http.StatusBadRequest, "root no válido")
		return
	}
	var root *clipRoot
	for index := range roots {
		if roots[index].id == rootID {
			root = &roots[index]
			break
		}
	}
	if root == nil {
		writeJSONError(w, http.StatusNotFound, "carpeta no encontrada")
		return
	}

	parts, err := clipRelativeParts(r.URL.Query().Get("path"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, offset, err := clipPagination(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	basePath := root.path
	for _, part := range parts {
		basePath = filepath.Join(basePath, part)
	}

	items, err := loadClipItemPaths(r.Context(), a.db, libraryID)
	if err != nil {
		log.Printf("library: leer vídeos de la biblioteca %d: %v", libraryID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	direct := make([]clipItemPath, 0)
	folderCounts := make(map[string]int64)
	folderCovers := make(map[string]clipItemPath)
	for _, item := range items {
		if !pathWithin(root.path, item.path) {
			continue
		}
		rel, err := filepath.Rel(basePath, item.path)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
			continue
		}
		partsForItem := strings.Split(rel, "/")
		if len(partsForItem) == 1 {
			direct = append(direct, item)
		} else {
			name := partsForItem[0]
			folderCounts[name]++
			if cover, ok := folderCovers[name]; !ok || betterClipCover(item, cover) {
				folderCovers[name] = item
			}
		}
	}

	folders := make([]clipFolderDTO, 0, len(folderCounts))
	for name, count := range folderCounts {
		pathParts := append(append([]string{}, parts...), name)
		coverID := int64(0)
		if cover, ok := folderCovers[name]; ok {
			coverID = cover.id
		}
		folders = append(folders, clipFolderDTO{
			Name:        name,
			Path:        strings.Join(pathParts, "/"),
			ItemCount:   count,
			CoverItemID: optionalItemID(coverID),
		})
	}
	sort.SliceStable(folders, func(i, j int) bool {
		left := strings.ToLower(folders[i].Name)
		right := strings.ToLower(folders[j].Name)
		if left != right {
			return left < right
		}
		return folders[i].Path < folders[j].Path
	})

	sort.SliceStable(direct, func(i, j int) bool {
		left := strings.ToLower(direct[i].title)
		right := strings.ToLower(direct[j].title)
		if left != right {
			return left < right
		}
		return direct[i].id < direct[j].id
	})

	total := int64(len(direct))
	pageIDs := make([]int64, 0)
	start := offset
	if start > len(direct) {
		start = len(direct)
	}
	end := start + limit
	if end > len(direct) {
		end = len(direct)
	}
	for _, item := range direct[start:end] {
		pageIDs = append(pageIDs, item.id)
	}
	pageItems, err := a.mediaItemsByIDs(r.Context(), pageIDs, sessionUserID(r))
	if err != nil {
		log.Printf("library: leer fichas de la biblioteca %d: %v", libraryID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	pathsByID := make(map[int64]string, len(items))
	for _, item := range items {
		pathsByID[item.id] = item.path
	}
	for index := range pageItems {
		itemPath := pathsByID[pageItems[index].ID]
		relative, err := filepath.Rel(root.path, itemPath)
		if err != nil {
			relative = filepath.Base(itemPath)
		}
		pageItems[index].Path = filepath.ToSlash(relative)
		compactMediaItemCredits(&pageItems[index])
	}
	if pageItems == nil {
		pageItems = make([]mediaItemDTO, 0)
	}

	crumbs := make([]clipBreadcrumbDTO, 0, len(parts)+1)
	crumbs = append(crumbs, clipBreadcrumbDTO{Name: root.name, Path: ""})
	for index, part := range parts {
		crumbs = append(crumbs, clipBreadcrumbDTO{
			Name: part,
			Path: strings.Join(parts[:index+1], "/"),
		})
	}

	var parent *string
	if len(parts) > 0 {
		value := strings.Join(parts[:len(parts)-1], "/")
		parent = &value
	}

	writeJSON(w, http.StatusOK, clipBrowseResponse{
		Library: clipLibraryDTO{ID: libraryID, Name: libraryName},
		Root:    clipRootRefDTO{ID: root.id, Name: root.name},
		Path:    strings.Join(parts, "/"),
		Parent:  parent,
		Crumbs:  crumbs,
		Folders: folders,
		Items:   pageItems,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	})
}

func loadClipRoots(ctx context.Context, db *sql.DB, libraryID int64, fallback string) ([]clipRoot, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, path FROM library_paths WHERE library_id = ? ORDER BY id`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roots := make([]clipRoot, 0)
	for rows.Next() {
		var root clipRoot
		if err := rows.Scan(&root.id, &root.path); err != nil {
			return nil, err
		}
		root.path = filepath.Clean(root.path)
		root.name = filepath.Base(root.path)
		roots = append(roots, root)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(roots) == 0 && strings.TrimSpace(fallback) != "" {
		path := filepath.Clean(fallback)
		roots = append(roots, clipRoot{name: filepath.Base(path), path: path})
	}

	items, err := loadClipItemPaths(ctx, db, libraryID)
	if err != nil {
		return nil, err
	}
	nameCounts := make(map[string]int)
	for _, root := range roots {
		nameCounts[strings.ToLower(root.name)]++
	}
	for index := range roots {
		var cover clipItemPath
		hasCover := false
		for _, item := range items {
			if pathWithin(roots[index].path, item.path) {
				roots[index].itemCount++
				if !hasCover || betterClipCover(item, cover) {
					cover = item
					hasCover = true
				}
			}
		}
		if hasCover {
			roots[index].coverItemID = cover.id
		}
		if nameCounts[strings.ToLower(roots[index].name)] > 1 {
			parentName := filepath.Base(filepath.Dir(roots[index].path))
			if parentName != "." && parentName != string(filepath.Separator) && parentName != "" {
				roots[index].name = parentName + "/" + roots[index].name
			}
		}
	}
	sort.SliceStable(roots, func(i, j int) bool {
		left := strings.ToLower(roots[i].name)
		right := strings.ToLower(roots[j].name)
		if left != right {
			return left < right
		}
		return roots[i].id < roots[j].id
	})
	return roots, nil
}

func betterClipCover(candidate, current clipItemPath) bool {
	left := strings.ToLower(candidate.title)
	right := strings.ToLower(current.title)
	if left != right {
		return left < right
	}
	return candidate.id < current.id
}

func optionalItemID(id int64) *int64 {
	if id <= 0 {
		return nil
	}
	value := id
	return &value
}

func loadClipItemPaths(ctx context.Context, db *sql.DB, libraryID int64) ([]clipItemPath, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id, path, title FROM media_items WHERE library_id = ?`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]clipItemPath, 0)
	for rows.Next() {
		var item clipItemPath
		if err := rows.Scan(&item.id, &item.path, &item.title); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func clipRelativeParts(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if filepath.IsAbs(raw) || strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, `\`) {
		return nil, errors.New("la ruta debe ser relativa")
	}
	if len(raw) >= 2 && raw[1] == ':' {
		return nil, errors.New("la ruta debe ser relativa")
	}
	raw = filepath.ToSlash(raw)
	parts := strings.Split(raw, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return nil, errors.New("la ruta contiene componentes no válidos")
		}
	}
	return parts, nil
}

func clipPagination(r *http.Request) (limit, offset int, err error) {
	limit = defaultClipItemsLimit
	query := r.URL.Query()
	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		value, convErr := strconv.Atoi(raw)
		if convErr != nil || value < 1 || value > maxClipItemsLimit {
			return 0, 0, errors.New("limit debe estar entre 1 y 500")
		}
		limit = value
	}
	if raw := strings.TrimSpace(query.Get("offset")); raw != "" {
		value, convErr := strconv.Atoi(raw)
		if convErr != nil || value < 0 {
			return 0, 0, errors.New("offset debe ser un número mayor o igual que 0")
		}
		offset = value
	}
	return limit, offset, nil
}
