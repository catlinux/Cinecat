package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/catlinux/cinecat/internal/metadata"
)

// ApplyCollection sincroniza la colección de TMDB de una película. Las
// colecciones manuales nunca se modifican desde aquí.
func ApplyCollection(ctx context.Context, db *sql.DB, itemID int64, ref *metadata.CollectionRef) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("library: iniciar colección del ítem %d: %w", itemID, err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var managedCollectionID int64
	if ref != nil && ref.TMDBID > 0 {
		var source string
		var edited, deleted int
		err := tx.QueryRowContext(ctx, `
			SELECT id, source, edited, deleted
			  FROM collections
			 WHERE tmdb_collection_id = ?`, ref.TMDBID).
			Scan(&managedCollectionID, &source, &edited, &deleted)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			result, insertErr := tx.ExecContext(ctx, `
				INSERT INTO collections (
					name, poster_url, backdrop_url, tmdb_collection_id, source
				) VALUES (?, ?, ?, ?, 'tmdb')`,
				ref.Name, nullableText(ref.PosterURL), nullableText(ref.BackdropURL), ref.TMDBID)
			if insertErr != nil {
				return fmt.Errorf("library: crear colección de TMDB %d: %w", ref.TMDBID, insertErr)
			}
			managedCollectionID, err = result.LastInsertId()
			if err != nil {
				return fmt.Errorf("library: id de la colección de TMDB %d: %w", ref.TMDBID, err)
			}
			source = "tmdb"
		case err != nil:
			return fmt.Errorf("library: leer colección de TMDB %d: %w", ref.TMDBID, err)
		}

		if source == "tmdb" {
			if deleted == 0 {
				if edited == 0 {
					if _, err := tx.ExecContext(ctx, `
						UPDATE collections
						   SET name = ?, poster_url = ?, backdrop_url = ?
						 WHERE id = ?`,
						ref.Name, nullableText(ref.PosterURL), nullableText(ref.BackdropURL),
						managedCollectionID); err != nil {
						return fmt.Errorf("library: actualizar colección %d: %w", managedCollectionID, err)
					}
				}
				if _, err := tx.ExecContext(ctx, `
					INSERT OR IGNORE INTO collection_items (
						collection_id, media_item_id, position
					) VALUES (?, ?, NULL)`,
					managedCollectionID, itemID); err != nil {
					return fmt.Errorf("library: añadir ítem %d a la colección %d: %w",
						itemID, managedCollectionID, err)
				}
			}
		}
	}

	// Las filas excluidas de la colección actual se conservan: INSERT OR IGNORE
	// no las toca. En colecciones anteriores se eliminan, aunque estuvieran
	// excluidas, porque la película ya no pertenece a ellas.
	query := `
		DELETE FROM collection_items
		 WHERE media_item_id = ?
		   AND collection_id IN (SELECT id FROM collections WHERE source = 'tmdb')`
	args := []any{itemID}
	if managedCollectionID > 0 {
		query += ` AND collection_id <> ?`
		args = append(args, managedCollectionID)
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("library: limpiar colecciones anteriores del ítem %d: %w", itemID, err)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE media_items SET collection_checked = 1 WHERE id = ?`, itemID); err != nil {
		return fmt.Errorf("library: marcar colección comprobada del ítem %d: %w", itemID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("library: confirmar colección del ítem %d: %w", itemID, err)
	}
	return nil
}

// clearTMDBCollectionItems borra las pertenencias del ítem a colecciones de
// TMDB. Se eliminan también las filas excluidas: al perder la ficha, una
// identificación posterior debe poder decidir de nuevo sin exclusiones viejas.
func clearTMDBCollectionItems(ctx context.Context, tx *sql.Tx, itemID int64) error {
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM collection_items
		 WHERE media_item_id = ?
		   AND collection_id IN (SELECT id FROM collections WHERE source = 'tmdb')`,
		itemID); err != nil {
		return fmt.Errorf("library: quitar colecciones del ítem %d: %w", itemID, err)
	}
	return nil
}
