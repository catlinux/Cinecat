package library

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
)

// clipTitle construye el título visible de un vídeo sin identificar. No usa
// el analizador de películas: los clips no tienen temporadas, año ni ficha.
func clipTitle(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	title := strings.NewReplacer("_", " ", ".", " ").Replace(base)
	title = strings.Join(strings.Fields(title), " ")
	if title == "" {
		title = strings.TrimSpace(base)
	}
	return title
}

// upsertClipItem guarda un vídeo sin metadatos. No toca una ficha existente:
// los clips no se reidentifican en escaneos posteriores.
func upsertClipItem(ctx context.Context, db *sql.DB, libraryID int64, path, title string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO media_items (
			library_id, path, title, year, kind, season, episode, series_id,
			match_status, metadata_source
		) VALUES (?, ?, ?, NULL, 'movie', NULL, NULL, NULL, 'none', NULL)
		ON CONFLICT (library_id, path) DO UPDATE SET
			title = excluded.title,
			kind = excluded.kind,
			season = NULL,
			episode = NULL,
			series_id = NULL`,
		libraryID, path, title,
	)
	if err != nil {
		return fmt.Errorf("library: guardar clip %q: %w", path, err)
	}
	return refreshSortKeysByPath(ctx, db, libraryID, path)
}
