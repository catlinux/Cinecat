package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/catlinux/cinecat/internal/settings"
)

const (
	trashFolderName          = ".cinecat-papelera"
	defaultTrashRetentionDay = 14
	trashTimeLayout          = "20060102-150405"
)

var (
	errTrashDisabled = errors.New("la modificación de archivos está desactivada")
	errTrashMusic    = errors.New("la papelera solo admite bibliotecas de vídeo o clips")
	errTrashNotFound = errors.New("elemento de la papelera no encontrado")
	errItemNotFound  = errors.New("ítem no encontrado")
	errTrashConflict = errors.New("la ruta original ya existe")
	errTrashUnsafe   = errors.New("ruta de archivo no permitida")
	errTrashSymlink  = errors.New("no se pueden mover enlaces simbólicos")
	errTrashFileGone = errors.New("archivo no encontrado")
)

type trashExtraPath struct {
	Original string `json:"original"`
	Trash    string `json:"trash"`
}

type trashSnapshot struct {
	MediaItem        map[string]any   `json:"media_item"`
	PlaybackState    []map[string]any `json:"playback_state"`
	WatchState       []map[string]any `json:"watch_state"`
	PlaybackProgress []map[string]any `json:"playback_progress"`
	ItemPreferences  []map[string]any `json:"item_preferences"`
	ItemLists        []map[string]any `json:"item_lists"`
	CollectionItems  []map[string]any `json:"collection_items"`
}

type trashItemDTO struct {
	ID           int64   `json:"id"`
	LibraryID    int64   `json:"library_id"`
	Title        string  `json:"title"`
	Year         *int    `json:"year"`
	Kind         string  `json:"kind"`
	OriginalPath string  `json:"original_path"`
	MovedKind    string  `json:"moved_kind"`
	DeletedBy    *string `json:"deleted_by"`
	DeletedAt    string  `json:"deleted_at"`
	ExpiresAt    string  `json:"expires_at"`
}

type trashListResponse struct {
	Items []trashItemDTO `json:"items"`
}

type trashMedia struct {
	ID          int64
	LibraryID   int64
	Path        string
	Title       string
	Year        sql.NullInt64
	Kind        string
	LibraryKind string
	LibraryPath string
}

type trashMove struct {
	Original string
	Trash    string
}

type trashPlan struct {
	Media     trashMedia
	Root      string
	TrashPath string
	MovedKind string
	Extra     []trashExtraPath
	Moves     []trashMove
}

func (a *API) clockNow() time.Time {
	if a.now != nil {
		return a.now().UTC()
	}
	return time.Now().UTC()
}

func (a *API) allowFileChanges() bool {
	return a.settingsStore != nil && a.settingsStore.Bool(settings.KeyAllowFileChanges)
}

func (a *API) trashRetentionDays() int {
	days := defaultTrashRetentionDay
	if a.settingsStore != nil {
		days = a.settingsStore.Int(settings.KeyTrashRetention)
	}
	if days < 1 {
		return 1
	}
	if days > 90 {
		return 90
	}
	return days
}

func (a *API) trashExpiresAt(deletedAt string) string {
	parsed, err := time.Parse(time.RFC3339, deletedAt)
	if err != nil {
		return deletedAt
	}
	return parsed.Add(time.Duration(a.trashRetentionDays()) * 24 * time.Hour).UTC().Format(time.RFC3339)
}

func (a *API) loadTrashMedia(ctx context.Context, id int64) (trashMedia, error) {
	var item trashMedia
	err := a.db.QueryRowContext(ctx, `
		SELECT mi.id, mi.library_id, mi.path, mi.title, mi.year, mi.kind, l.kind, l.path
		  FROM media_items mi
		  JOIN libraries l ON l.id = mi.library_id
		 WHERE mi.id = ?`, id,
	).Scan(&item.ID, &item.LibraryID, &item.Path, &item.Title, &item.Year,
		&item.Kind, &item.LibraryKind, &item.LibraryPath)
	if errors.Is(err, sql.ErrNoRows) {
		return item, errItemNotFound
	}
	if err != nil {
		return item, fmt.Errorf("library: leer ítem para la papelera: %w", err)
	}
	return item, nil
}

func (a *API) libraryRoot(ctx context.Context, item trashMedia) (string, error) {
	root := filepath.Clean(a.libraryRootFor(ctx, item.LibraryID, item.Path, item.LibraryPath))
	original := filepath.Clean(item.Path)
	if !filepath.IsAbs(original) || !pathWithin(root, original) {
		return "", errTrashUnsafe
	}
	return root, nil
}

func (a *API) isLibraryRoot(ctx context.Context, libraryID int64, path string) bool {
	path = filepath.Clean(path)
	rows, err := a.db.QueryContext(ctx, `
		SELECT path FROM library_paths WHERE library_id = ?
		UNION
		SELECT path FROM libraries WHERE id = ?`, libraryID, libraryID)
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var root string
		if rows.Scan(&root) != nil {
			continue
		}
		if samePath(root, path) {
			return true
		}
	}
	return false
}

func samePath(first, second string) bool {
	first = filepath.Clean(first)
	second = filepath.Clean(second)
	if first == second {
		return true
	}
	return filepath.VolumeName(first) != "" && strings.EqualFold(first, second)
}

func hasOtherVideo(folder, original string) bool {
	other := false
	err := filepath.WalkDir(folder, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if samePath(path, original) || entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if IsVideoFile(entry.Name()) {
			other = true
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return true
	}
	return other
}

func isSubtitleFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".srt", ".vtt", ".ass", ".ssa", ".sub", ".idx":
		return true
	default:
		return false
	}
}

func subtitlePaths(mediaPath string) ([]string, error) {
	dir := filepath.Dir(mediaPath)
	base := strings.TrimSuffix(filepath.Base(mediaPath), filepath.Ext(mediaPath))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	lowerBase := strings.ToLower(base)
	var result []string
	for _, entry := range entries {
		if entry.IsDir() || !isSubtitleFile(entry.Name()) {
			continue
		}
		stem := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		lowerStem := strings.ToLower(stem)
		if lowerStem != lowerBase && !strings.HasPrefix(lowerStem, lowerBase+".") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, errTrashSymlink
		}
		result = append(result, path)
	}
	sort.Strings(result)
	return result, nil
}

func uniqueTrashPath(dir, stamp, name string) (string, error) {
	base := stamp + "-" + filepath.Base(name)
	candidate := filepath.Join(dir, base)
	for index := 2; ; index++ {
		_, err := os.Lstat(candidate)
		if errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		}
		if err != nil {
			return "", err
		}
		candidate = filepath.Join(dir, fmt.Sprintf("%s-%d", base, index))
	}
}

func (a *API) buildTrashPlan(ctx context.Context, item trashMedia) (trashPlan, error) {
	plan := trashPlan{Media: item, Extra: []trashExtraPath{}}
	if item.LibraryKind != "video" && item.LibraryKind != "clips" {
		return plan, errTrashMusic
	}
	root, err := a.libraryRoot(ctx, item)
	if err != nil {
		return plan, err
	}
	plan.Root = root
	original := filepath.Clean(item.Path)
	info, err := os.Lstat(original)
	if errors.Is(err, os.ErrNotExist) {
		return plan, errTrashFileGone
	}
	if err != nil {
		return plan, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return plan, errTrashSymlink
	}

	moveFolder := item.Kind == "movie" && !a.isLibraryRoot(ctx, item.LibraryID, filepath.Dir(original))
	if moveFolder && !pathWithin(root, filepath.Dir(original)) {
		moveFolder = false
	}
	if moveFolder {
		parent := filepath.Dir(original)
		parentInfo, err := os.Lstat(parent)
		if err != nil {
			return plan, err
		}
		if parentInfo.Mode()&os.ModeSymlink != 0 {
			return plan, errTrashSymlink
		}
		if hasOtherVideo(parent, original) {
			moveFolder = false
		}
	}

	trashDir := filepath.Join(root, trashFolderName)
	stamp := a.clockNow().Format(trashTimeLayout)
	if moveFolder {
		folder := filepath.Dir(original)
		trashPath, err := uniqueTrashPath(trashDir, stamp, folder)
		if err != nil {
			return plan, err
		}
		plan.MovedKind = "folder"
		plan.TrashPath = trashPath
		plan.Moves = []trashMove{{Original: folder, Trash: trashPath}}
		return plan, nil
	}

	trashPath, err := uniqueTrashPath(trashDir, stamp, original)
	if err != nil {
		return plan, err
	}
	plan.MovedKind = "file"
	plan.TrashPath = trashPath
	plan.Moves = []trashMove{{Original: original, Trash: trashPath}}
	subtitles, err := subtitlePaths(original)
	if err != nil {
		return plan, err
	}
	for _, subtitle := range subtitles {
		trashSubtitle, err := uniqueTrashPath(trashDir, stamp, subtitle)
		if err != nil {
			return plan, err
		}
		plan.Extra = append(plan.Extra, trashExtraPath{Original: subtitle, Trash: trashSubtitle})
		plan.Moves = append(plan.Moves, trashMove{Original: subtitle, Trash: trashSubtitle})
	}
	return plan, nil
}

func (a *API) applyTrashMoves(plan trashPlan) error {
	trashDir := filepath.Join(plan.Root, trashFolderName)
	if err := os.MkdirAll(trashDir, 0o775); err != nil {
		return fmt.Errorf("library: crear la papelera: %w", err)
	}
	info, err := os.Lstat(trashDir)
	if err != nil {
		return fmt.Errorf("library: comprobar la papelera: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errTrashUnsafe
	}
	applyConfiguredOwner(trashDir)
	moved := make([]trashMove, 0, len(plan.Moves))
	for _, move := range plan.Moves {
		if err := os.Rename(move.Original, move.Trash); err != nil {
			undoTrashMoves(moved)
			return fmt.Errorf("library: mover a la papelera: %w", err)
		}
		moved = append(moved, move)
	}
	return nil
}

func undoTrashMoves(moves []trashMove) {
	for index := len(moves) - 1; index >= 0; index-- {
		move := moves[index]
		if err := os.MkdirAll(filepath.Dir(move.Original), 0o775); err != nil {
			log.Printf("library: papelera: no se pudo recrear %s: %v", filepath.Dir(move.Original), err)
			continue
		}
		if err := os.Rename(move.Trash, move.Original); err != nil {
			log.Printf("library: papelera: no se pudo deshacer el movimiento %s: %v", move.Trash, err)
		}
	}
}

func queryRowMaps(ctx context.Context, tx *sql.Tx, query string, args ...any) ([]map[string]any, error) {
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for index := range values {
			pointers[index] = &values[index]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for index, column := range columns {
			row[column] = values[index]
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func captureTrashSnapshot(ctx context.Context, tx *sql.Tx, id int64) (string, error) {
	mediaRows, err := queryRowMaps(ctx, tx, `SELECT * FROM media_items WHERE id = ?`, id)
	if err != nil {
		return "", err
	}
	if len(mediaRows) != 1 {
		return "", errItemNotFound
	}
	snapshot := trashSnapshot{MediaItem: mediaRows[0]}
	for table, target := range map[string]*[]map[string]any{
		"playback_state":    &snapshot.PlaybackState,
		"watch_state":       &snapshot.WatchState,
		"playback_progress": &snapshot.PlaybackProgress,
		"item_preferences":  &snapshot.ItemPreferences,
		"item_lists":        &snapshot.ItemLists,
		"collection_items":  &snapshot.CollectionItems,
	} {
		rows, err := queryRowMaps(ctx, tx,
			`SELECT * FROM `+table+` WHERE media_item_id = ?`, id)
		if err != nil {
			return "", err
		}
		*target = rows
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func (a *API) trashItem(ctx context.Context, id int64, actor string) (trashItemDTO, error) {
	var result trashItemDTO
	if !a.allowFileChanges() {
		return result, errTrashDisabled
	}
	item, err := a.loadTrashMedia(ctx, id)
	if err != nil {
		return result, err
	}
	plan, err := a.buildTrashPlan(ctx, item)
	if err != nil {
		return result, err
	}
	if err := a.applyTrashMoves(plan); err != nil {
		return result, err
	}
	extraJSON, err := json.Marshal(plan.Extra)
	if err != nil {
		undoTrashMoves(plan.Moves)
		return result, err
	}

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		undoTrashMoves(plan.Moves)
		return result, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	snapshot, err := captureTrashSnapshot(ctx, tx, id)
	if err != nil {
		undoTrashMoves(plan.Moves)
		return result, err
	}
	deletedAt := a.clockNow().Format(time.RFC3339)
	insert, err := tx.ExecContext(ctx, `
		INSERT INTO trash_items
			(library_id, original_path, trash_path, moved_kind, extra_paths,
			 title, year, kind, snapshot, deleted_by, deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.LibraryID, plan.Moves[0].Original, plan.TrashPath, plan.MovedKind, string(extraJSON),
		item.Title, item.Year, item.Kind, snapshot, actor, deletedAt)
	if err != nil {
		undoTrashMoves(plan.Moves)
		return result, fmt.Errorf("library: guardar en la papelera: %w", err)
	}
	trashID, err := insert.LastInsertId()
	if err != nil {
		undoTrashMoves(plan.Moves)
		return result, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM media_items WHERE id = ?`, id); err != nil {
		undoTrashMoves(plan.Moves)
		return result, fmt.Errorf("library: borrar el ítem movido: %w", err)
	}
	if err := cleanupOrphans(ctx, tx); err != nil {
		undoTrashMoves(plan.Moves)
		return result, err
	}
	if err := tx.Commit(); err != nil {
		undoTrashMoves(plan.Moves)
		return result, fmt.Errorf("library: confirmar la papelera: %w", err)
	}
	committed = true

	log.Printf("library: papelera: %s movió %s a %s", actor, plan.Moves[0].Original, plan.TrashPath)
	return trashItemDTO{
		ID:           trashID,
		LibraryID:    item.LibraryID,
		Title:        item.Title,
		Year:         nullIntPointer(item.Year),
		Kind:         item.Kind,
		OriginalPath: plan.Moves[0].Original,
		MovedKind:    plan.MovedKind,
		DeletedBy:    textPointer(actor),
		DeletedAt:    deletedAt,
		ExpiresAt:    a.trashExpiresAt(deletedAt),
	}, nil
}

func textPointer(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func nullIntPointer(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	number := int(value.Int64)
	return &number
}

func (a *API) listTrash(ctx context.Context) ([]trashItemDTO, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT id, library_id, title, year, kind, original_path, moved_kind,
		       deleted_by, deleted_at
		  FROM trash_items
		 ORDER BY deleted_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]trashItemDTO, 0)
	for rows.Next() {
		var item trashItemDTO
		var year sql.NullInt64
		var deletedBy sql.NullString
		if err := rows.Scan(&item.ID, &item.LibraryID, &item.Title, &year, &item.Kind,
			&item.OriginalPath, &item.MovedKind, &deletedBy, &item.DeletedAt); err != nil {
			return nil, err
		}
		item.Year = nullIntPointer(year)
		item.DeletedBy = textPointer(deletedBy.String)
		item.ExpiresAt = a.trashExpiresAt(item.DeletedAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

type trashRestoreRow struct {
	ID           int64
	LibraryID    int64
	OriginalPath string
	TrashPath    string
	ExtraPaths   string
	Snapshot     string
}

func (a *API) loadTrashRestore(ctx context.Context, id int64) (trashRestoreRow, error) {
	var row trashRestoreRow
	err := a.db.QueryRowContext(ctx, `
		SELECT id, library_id, original_path, trash_path, extra_paths, snapshot
		  FROM trash_items WHERE id = ?`, id,
	).Scan(&row.ID, &row.LibraryID, &row.OriginalPath, &row.TrashPath,
		&row.ExtraPaths, &row.Snapshot)
	if errors.Is(err, sql.ErrNoRows) {
		return row, errTrashNotFound
	}
	if err != nil {
		return row, err
	}
	return row, nil
}

func (a *API) restoreTrash(ctx context.Context, trashID int64, actor string) (int64, error) {
	if !a.allowFileChanges() {
		return 0, errTrashDisabled
	}
	row, err := a.loadTrashRestore(ctx, trashID)
	if err != nil {
		return 0, err
	}
	var extra []trashExtraPath
	if strings.TrimSpace(row.ExtraPaths) != "" {
		if err := json.Unmarshal([]byte(row.ExtraPaths), &extra); err != nil {
			return 0, fmt.Errorf("library: leer rutas extra de la papelera: %w", err)
		}
	}
	restores := []trashMove{{Original: row.OriginalPath, Trash: row.TrashPath}}
	for _, extraPath := range extra {
		restores = append(restores, trashMove{Original: extraPath.Original, Trash: extraPath.Trash})
	}
	for _, restore := range restores {
		if _, err := os.Lstat(restore.Original); err == nil {
			return 0, errTrashConflict
		} else if !errors.Is(err, os.ErrNotExist) {
			return 0, err
		}
		if _, err := os.Lstat(restore.Trash); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return 0, errTrashNotFound
			}
			return 0, err
		}
	}
	moved := make([]trashMove, 0, len(restores))
	for _, restore := range restores {
		parent := filepath.Dir(restore.Original)
		if err := os.MkdirAll(parent, 0o775); err != nil {
			undoTrashMoves(moved)
			return 0, err
		}
		applyConfiguredOwner(parent)
		if err := os.Rename(restore.Trash, restore.Original); err != nil {
			undoTrashMoves(moved)
			return 0, fmt.Errorf("library: restaurar desde la papelera: %w", err)
		}
		moved = append(moved, restore)
	}

	newID, err := a.restoreTrashDatabase(ctx, row)
	if err != nil {
		undoTrashMoves(moved)
		return 0, err
	}
	if a.scanJob != nil {
		a.scanJob.StartService()
	}
	log.Printf("library: papelera: %s restauró %s con el ítem %d", actor, row.OriginalPath, newID)
	return newID, nil
}

func (a *API) restoreTrashDatabase(ctx context.Context, row trashRestoreRow) (int64, error) {
	var snapshot trashSnapshot
	decoder := json.NewDecoder(strings.NewReader(row.Snapshot))
	decoder.UseNumber()
	if err := decoder.Decode(&snapshot); err != nil {
		return 0, fmt.Errorf("library: leer la instantánea de la papelera: %w", err)
	}
	if snapshot.MediaItem == nil {
		return 0, errors.New("library: la instantánea no contiene el ítem")
	}
	normalizeSnapshotNumbers(snapshot.MediaItem)
	for _, table := range [][]map[string]any{
		snapshot.PlaybackState, snapshot.WatchState, snapshot.PlaybackProgress,
		snapshot.ItemPreferences, snapshot.ItemLists, snapshot.CollectionItems,
	} {
		for _, values := range table {
			normalizeSnapshotNumbers(values)
		}
	}

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	originalID, _ := asInt64(snapshot.MediaItem["id"])
	newID := originalID
	var occupied int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM media_items WHERE id = ?`, originalID).Scan(&occupied)
	switch {
	case err == nil:
		delete(snapshot.MediaItem, "id")
		newID = 0
	case !errors.Is(err, sql.ErrNoRows):
		return 0, err
	}
	if seriesValue, ok := snapshot.MediaItem["series_id"]; ok && seriesValue != nil {
		seriesID, ok := asInt64(seriesValue)
		if !ok {
			snapshot.MediaItem["series_id"] = nil
		} else {
			var exists int
			err := tx.QueryRowContext(ctx, `SELECT 1 FROM series WHERE id = ?`, seriesID).Scan(&exists)
			if errors.Is(err, sql.ErrNoRows) {
				snapshot.MediaItem["series_id"] = nil
			} else if err != nil {
				return 0, err
			}
		}
	}
	result, err := insertSnapshotRow(ctx, tx, "media_items", snapshot.MediaItem)
	if err != nil {
		if isUniqueError(err) {
			return 0, errTrashConflict
		}
		return 0, fmt.Errorf("library: restaurar el ítem: %w", err)
	}
	if newID == 0 {
		newID, err = result.LastInsertId()
		if err != nil {
			return 0, err
		}
	}
	for table, rows := range map[string][]map[string]any{
		"playback_state":    snapshot.PlaybackState,
		"watch_state":       snapshot.WatchState,
		"playback_progress": snapshot.PlaybackProgress,
		"item_preferences":  snapshot.ItemPreferences,
		"item_lists":        snapshot.ItemLists,
		"collection_items":  snapshot.CollectionItems,
	} {
		for _, values := range rows {
			values["media_item_id"] = newID
			if _, err := insertSnapshotRow(ctx, tx, table, values); err != nil {
				if !isForeignKeyError(err) {
					return 0, fmt.Errorf("library: restaurar %s: %w", table, err)
				}
				log.Printf("library: papelera: no se pudo restaurar %s: %v", table, err)
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM trash_items WHERE id = ?`, row.ID); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	committed = true
	return newID, nil
}

func normalizeSnapshotNumbers(values map[string]any) {
	for key, value := range values {
		number, ok := value.(json.Number)
		if !ok {
			continue
		}
		if integer, err := number.Int64(); err == nil {
			values[key] = integer
			continue
		}
		if decimal, err := number.Float64(); err == nil {
			values[key] = decimal
		}
	}
}

func asInt64(value any) (int64, bool) {
	switch number := value.(type) {
	case int64:
		return number, true
	case int:
		return int64(number), true
	case float64:
		return int64(number), float64(int64(number)) == number
	case json.Number:
		integer, err := number.Int64()
		return integer, err == nil
	default:
		return 0, false
	}
}

func insertSnapshotRow(ctx context.Context, tx *sql.Tx, table string, values map[string]any) (sql.Result, error) {
	columns := make([]string, 0, len(values))
	for column := range values {
		columns = append(columns, column)
	}
	sort.Strings(columns)
	placeholders := make([]string, len(columns))
	args := make([]any, len(columns))
	for index, column := range columns {
		placeholders[index] = "?"
		args[index] = values[column]
	}
	query := `INSERT INTO ` + table + ` (` + strings.Join(columns, ", ") + `) VALUES (` +
		strings.Join(placeholders, ", ") + `)`
	return tx.ExecContext(ctx, query, args...)
}

func isForeignKeyError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "foreign key constraint")
}

func (a *API) deleteTrashItem(ctx context.Context, id int64, actor string) error {
	var trashPath string
	err := a.db.QueryRowContext(ctx, `SELECT trash_path FROM trash_items WHERE id = ?`, id).Scan(&trashPath)
	if errors.Is(err, sql.ErrNoRows) {
		return errTrashNotFound
	}
	if err != nil {
		return err
	}
	if err := removeTrashPath(trashPath); err != nil {
		return err
	}
	if _, err := a.db.ExecContext(ctx, `DELETE FROM trash_items WHERE id = ?`, id); err != nil {
		return err
	}
	log.Printf("library: papelera: %s eliminó definitivamente %s", actor, trashPath)
	return nil
}

func removeTrashPath(path string) error {
	clean := filepath.Clean(path)
	parent := filepath.Dir(clean)
	if filepath.Base(parent) != trashFolderName || !pathWithin(parent, clean) {
		return errTrashUnsafe
	}
	parentInfo, err := os.Lstat(parent)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if parentInfo.Mode()&os.ModeSymlink != 0 {
		return errTrashUnsafe
	}
	if _, err := os.Lstat(clean); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	return os.RemoveAll(clean)
}

func (a *API) purgeExpiredTrash(ctx context.Context, actor string) (int, error) {
	now := a.clockNow()
	cutoff := now.Add(-time.Duration(a.trashRetentionDays()) * 24 * time.Hour).Format(time.RFC3339)
	rows, err := a.db.QueryContext(ctx,
		`SELECT id, trash_path FROM trash_items WHERE deleted_at < ? ORDER BY id`, cutoff)
	if err != nil {
		return 0, err
	}
	type pending struct {
		id   int64
		path string
	}
	var items []pending
	for rows.Next() {
		var item pending
		if err := rows.Scan(&item.id, &item.path); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	purged := 0
	for _, item := range items {
		if err := removeTrashPath(item.path); err != nil {
			log.Printf("library: papelera: no se pudo purgar %s: %v", item.path, err)
			continue
		}
		if _, err := a.db.ExecContext(ctx, `DELETE FROM trash_items WHERE id = ?`, item.id); err != nil {
			log.Printf("library: papelera: no se pudo borrar la fila de %s: %v", item.path, err)
			continue
		}
		purged++
		log.Printf("library: papelera: %s purgó %s", actor, item.path)
	}
	return purged, nil
}

// StartTrashPurger ejecuta una purga al arrancar y después cada 24 horas hasta
// que se cancele el contexto del servicio.
func (a *API) StartTrashPurger(ctx context.Context) {
	if _, err := a.purgeExpiredTrash(ctx, "sistema"); err != nil && ctx.Err() == nil {
		log.Printf("library: papelera: primera purga: %v", err)
	}
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := a.purgeExpiredTrash(ctx, "sistema"); err != nil && ctx.Err() == nil {
				log.Printf("library: papelera: purga diaria: %v", err)
			}
		}
	}
}
