package library

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	thumbnailPurgeMinAge = time.Hour
	thumbnailPurgeBatch  = 500
)

// El nombre lo genera Thumbnailer.Get con id, fecha y tamaño del archivo.
var thumbnailNamePattern = regexp.MustCompile(`^([0-9]+)-(-?[0-9]+)-([0-9]+)\.jpg$`)

// purgeOrphanThumbnails borra solo miniaturas con el patrón exacto cuyo ítem
// ya no existe. Los archivos recientes se dejan por si se están generando.
func (a *API) purgeOrphanThumbnails(ctx context.Context, now time.Time) (int, error) {
	if a == nil || a.db == nil || a.thumbnails == nil || a.thumbnails.cacheDir == "" {
		return 0, nil
	}
	entries, err := os.ReadDir(a.thumbnails.cacheDir)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	byID := make(map[int64][]string)
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		id, ok := thumbnailItemID(entry.Name())
		if !ok {
			continue
		}
		path := filepath.Join(a.thumbnails.cacheDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) < thumbnailPurgeMinAge {
			continue
		}
		byID[id] = append(byID[id], path)
	}
	if len(byID) == 0 {
		return 0, nil
	}

	ids := make([]int64, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	existing, err := existingMediaItemIDs(ctx, a.db, ids)
	if err != nil {
		return 0, err
	}

	purged := 0
	for id, paths := range byID {
		if _, ok := existing[id]; ok {
			continue
		}
		for _, path := range paths {
			if err := os.Remove(path); errors.Is(err, os.ErrNotExist) {
				continue
			} else if err != nil {
				log.Printf("library: miniaturas: no se pudo borrar %s: %v", path, err)
				continue
			}
			purged++
		}
	}
	if purged > 0 {
		log.Printf("library: miniaturas: purgadas %d miniatura(s) huérfana(s)", purged)
	}
	return purged, nil
}

func thumbnailItemID(name string) (int64, bool) {
	match := thumbnailNamePattern.FindStringSubmatch(name)
	if match == nil || !canonicalInt64(match[1]) ||
		!canonicalInt64(match[2]) || !canonicalInt64(match[3]) {
		return 0, false
	}
	id, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func canonicalInt64(raw string) bool {
	value, err := strconv.ParseInt(raw, 10, 64)
	return err == nil && strconv.FormatInt(value, 10) == raw
}

func existingMediaItemIDs(ctx context.Context, db *sql.DB, ids []int64) (map[int64]struct{}, error) {
	existing := make(map[int64]struct{}, len(ids))
	for start := 0; start < len(ids); start += thumbnailPurgeBatch {
		end := start + thumbnailPurgeBatch
		if end > len(ids) {
			end = len(ids)
		}
		placeholders := make([]string, end-start)
		args := make([]any, end-start)
		for i, id := range ids[start:end] {
			placeholders[i] = "?"
			args[i] = id
		}
		rows, err := db.QueryContext(ctx,
			`SELECT id FROM media_items WHERE id IN (`+strings.Join(placeholders, ",")+`)`,
			args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			existing[id] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	return existing, nil
}
