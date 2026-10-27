package library

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/catlinux/cinecat/internal/settings"
)

var (
	errUploadLibraryNotFound = errors.New("biblioteca no encontrada")
	errUploadMusic           = errors.New("la subida solo admite bibliotecas de vídeo o clips")
	errUploadInvalid         = errors.New("ruta o nombre de archivo no válido")
	errUploadConflict        = errors.New("el archivo ya existe")
	errUploadNoLength        = errors.New("falta la cabecera Content-Length")
	errUploadTooLarge        = errors.New("el archivo supera el tamaño máximo")
	errUploadNoSpace         = errors.New("no hay espacio libre suficiente")
	errUploadIncomplete      = errors.New("el cuerpo no coincide con Content-Length")
)

const uploadSpaceMargin = int64(1) << 30

// diskFree es una variable para poder simular la falta de espacio en las
// pruebas; en producción apunta a la implementación del sistema.
var diskFree = diskFreeBytes

type uploadTarget struct {
	LibraryID int64
	Kind      string
	Dir       string
	Subdir    string
	FinalDir  string
	Name      string
	FinalPath string
}

type uploadResult struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func (a *API) uploadMaxBytes() int64 {
	gigabytes := 50
	if a.settingsStore != nil {
		gigabytes = a.settingsStore.Int(settings.KeyUploadMaxGB)
	}
	if gigabytes < 1 {
		gigabytes = 1
	}
	if gigabytes > 500 {
		gigabytes = 500
	}
	return int64(gigabytes) << 30
}

func hasTrashComponent(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(filepath.Clean(path)), "/") {
		if part == trashFolderName {
			return true
		}
	}
	return false
}

func (a *API) uploadLibraryRoots(ctx context.Context, libraryID int64, fallback string) ([]string, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT path FROM library_paths WHERE library_id = ?
		UNION
		SELECT path FROM libraries WHERE id = ?`, libraryID, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	roots := make([]string, 0)
	for rows.Next() {
		var root string
		if err := rows.Scan(&root); err != nil {
			return nil, err
		}
		roots = append(roots, root)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(roots) == 0 && strings.TrimSpace(fallback) != "" {
		roots = append(roots, fallback)
	}
	return roots, nil
}

func validateUploadName(name string, requireExtension bool) error {
	if name == "" || len(name) > 255 {
		return errUploadInvalid
	}
	if filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return errUploadInvalid
	}
	if strings.HasPrefix(name, ".") {
		return errUploadInvalid
	}
	for _, char := range name {
		if unicode.IsControl(char) {
			return errUploadInvalid
		}
	}
	if requireExtension && !IsVideoFile(name) && !isSubtitleFile(name) {
		return errUploadInvalid
	}
	return nil
}

func (a *API) uploadTarget(ctx context.Context, libraryID int64, query url.Values) (uploadTarget, error) {
	var target uploadTarget
	var libraryKind, libraryPath string
	err := a.db.QueryRowContext(ctx,
		`SELECT kind, path FROM libraries WHERE id = ?`, libraryID,
	).Scan(&libraryKind, &libraryPath)
	if errors.Is(err, sql.ErrNoRows) {
		return target, errUploadLibraryNotFound
	}
	if err != nil {
		return target, err
	}
	if libraryKind != "video" && libraryKind != "clips" {
		return target, errUploadMusic
	}

	rawDir := query.Get("dir")
	if rawDir == "" || !filepath.IsAbs(rawDir) || filepath.Clean(rawDir) != rawDir ||
		hasParentComponent(rawDir) || hasTrashComponent(rawDir) {
		return target, errUploadInvalid
	}
	info, err := os.Lstat(rawDir)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return target, errUploadInvalid
	}
	resolvedDir, err := filepath.EvalSymlinks(rawDir)
	if err != nil || hasTrashComponent(resolvedDir) {
		return target, errUploadInvalid
	}
	roots, err := a.uploadLibraryRoots(ctx, libraryID, libraryPath)
	if err != nil {
		return target, err
	}
	withinLibrary := false
	for _, root := range roots {
		resolvedRoot, err := filepath.EvalSymlinks(filepath.Clean(root))
		if err != nil {
			continue
		}
		if pathWithin(resolvedRoot, resolvedDir) {
			withinLibrary = true
			break
		}
	}
	if !withinLibrary {
		return target, errUploadInvalid
	}

	subdir := query.Get("subdir")
	if subdir != "" {
		if err := validateUploadName(subdir, false); err != nil {
			return target, err
		}
	}
	name := query.Get("name")
	if err := validateUploadName(name, true); err != nil {
		return target, err
	}

	finalDir := rawDir
	if subdir != "" {
		finalDir = filepath.Join(rawDir, subdir)
		if info, err := os.Lstat(finalDir); err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return target, errUploadInvalid
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return target, errUploadInvalid
		}
	}
	target = uploadTarget{
		LibraryID: libraryID,
		Kind:      libraryKind,
		Dir:       rawDir,
		Subdir:    subdir,
		FinalDir:  finalDir,
		Name:      name,
		FinalPath: filepath.Join(finalDir, name),
	}
	return target, nil
}

func (a *API) ensureUploadDirectory(target *uploadTarget) error {
	if target.Subdir == "" {
		return nil
	}
	return ensureUploadDir(target.FinalDir)
}

// ensureUploadDir crea la carpeta de destino si no existe, con los permisos y
// el propietario configurados. Nunca acepta enlaces simbólicos ni archivos.
func ensureUploadDir(path string) error {
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return errUploadInvalid
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Mkdir(path, 0o775); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return errUploadInvalid
		}
		return nil
	}
	if err := os.Chmod(path, 0o775); err != nil {
		return err
	}
	applyConfiguredOwner(path)
	return nil
}

func (a *API) writeUpload(w http.ResponseWriter, r *http.Request, target uploadTarget, length int64) (uploadResult, error) {
	var result uploadResult
	if _, err := os.Lstat(target.FinalPath); err == nil {
		return result, errUploadConflict
	} else if !errors.Is(err, os.ErrNotExist) {
		return result, err
	}

	partPath := filepath.Join(target.FinalDir, "."+target.Name+".part")
	part, err := os.OpenFile(partPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o664)
	if errors.Is(err, os.ErrExist) {
		return result, errUploadConflict
	}
	if err != nil {
		return result, err
	}
	success := false
	defer func() {
		if !success {
			if err := os.Remove(partPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				log.Printf("library: subida: no se pudo borrar %s: %v", partPath, err)
			}
		}
	}()

	if err := os.Chmod(partPath, 0o664); err != nil {
		_ = part.Close()
		return result, err
	}
	applyConfiguredOwner(partPath)

	limited := http.MaxBytesReader(w, r.Body, length)
	written, copyErr := io.Copy(part, limited)
	if copyErr != nil || written != length {
		_ = part.Close()
		return result, errUploadIncomplete
	}
	if err := part.Sync(); err != nil {
		_ = part.Close()
		return result, err
	}
	if err := part.Close(); err != nil {
		return result, err
	}
	if _, err := os.Lstat(target.FinalPath); err == nil {
		return result, errUploadConflict
	} else if !errors.Is(err, os.ErrNotExist) {
		return result, err
	}
	if err := os.Rename(partPath, target.FinalPath); err != nil {
		return result, err
	}
	success = true
	applyConfiguredOwner(target.FinalPath)
	return uploadResult{Path: target.FinalPath, Size: written}, nil
}

// receiveUpload comprueba tamaño y espacio, prepara la carpeta de destino y
// escribe el cuerpo en un .part que después renombra, sin cargarlo en memoria.
func (a *API) receiveUpload(w http.ResponseWriter, r *http.Request, target uploadTarget) (uploadResult, error) {
	length := r.ContentLength
	if length < 0 {
		return uploadResult{}, errUploadNoLength
	}
	if length > a.uploadMaxBytes() {
		return uploadResult{}, errUploadTooLarge
	}
	statDir := target.FinalDir
	if _, err := os.Lstat(statDir); errors.Is(err, os.ErrNotExist) {
		statDir = target.Dir
	}
	if free, known := diskFree(statDir); known && free < length+uploadSpaceMargin {
		return uploadResult{}, errUploadNoSpace
	}
	if err := a.ensureUploadDirectory(&target); err != nil {
		return uploadResult{}, err
	}
	return a.writeUpload(w, r, target, length)
}

func (a *API) startUploadScan() {
	if a.scanJob != nil {
		a.scanJob.StartService()
	}
}

func (a *API) handleLibraryFileUpload(w http.ResponseWriter, r *http.Request) {
	_ = http.NewResponseController(w).SetReadDeadline(time.Time{})

	id, ok := idFromPath(r)
	if !ok {
		writeUploadError(w, errUploadInvalid)
		return
	}
	if !a.allowFileChanges() {
		writeUploadError(w, errTrashDisabled)
		return
	}
	target, err := a.uploadTarget(r.Context(), id, r.URL.Query())
	if err != nil {
		writeUploadError(w, err)
		return
	}
	result, err := a.receiveUpload(w, r, target)
	if err != nil {
		writeUploadError(w, err)
		return
	}
	log.Printf("library: subida: %s subió %s (%d bytes)", trashActor(r), result.Path, result.Size)
	a.startUploadScan()
	writeJSON(w, http.StatusCreated, result)
}

func writeUploadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errTrashDisabled):
		writeJSONError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, errUploadLibraryNotFound):
		writeJSONError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, errUploadMusic), errors.Is(err, errUploadInvalid):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, errUploadConflict):
		writeJSONError(w, http.StatusConflict, err.Error())
	case errors.Is(err, errUploadNoLength):
		writeJSONError(w, http.StatusLengthRequired, err.Error())
	case errors.Is(err, errUploadTooLarge):
		writeJSONError(w, http.StatusRequestEntityTooLarge, err.Error())
	case errors.Is(err, errUploadNoSpace):
		writeJSONError(w, http.StatusInsufficientStorage, err.Error())
	case errors.Is(err, errUploadIncomplete):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("library: subida: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
	}
}
