package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type relinkCandidate struct {
	id      int64
	path    string
	title   string
	year    sql.NullInt64
	kind    string
	season  sql.NullInt64
	episode sql.NullInt64
}

// relinkMediaItem conserva la ficha cuando un archivo se renombra o cambia de
// extensión dentro de la misma carpeta. Solo actúa si la ruta nueva no tiene
// fila y hay una única candidata cuya ruta antigua ya no existe.
func (s *Scanner) relinkMediaItem(
	ctx context.Context,
	libraryID int64,
	root string,
	path string,
	media Media,
) (bool, error) {
	exists, err := mediaItemExists(ctx, s.db, libraryID, path)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, path, title, year, kind, season, episode
		  FROM media_items
		 WHERE library_id = ?`, libraryID)
	if err != nil {
		return false, fmt.Errorf("library: listar candidatas para %q: %w", path, err)
	}
	defer rows.Close()

	newDir := filepath.Dir(filepath.Clean(path))
	newBase := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	var exact, sameMetadata []relinkCandidate
	for rows.Next() {
		var candidate relinkCandidate
		if err := rows.Scan(
			&candidate.id,
			&candidate.path,
			&candidate.title,
			&candidate.year,
			&candidate.kind,
			&candidate.season,
			&candidate.episode,
		); err != nil {
			return false, fmt.Errorf("library: leer candidata para %q: %w", path, err)
		}
		candidatePath := filepath.Clean(candidate.path)
		if filepath.Dir(candidatePath) != newDir {
			continue
		}
		if !fileMissing(candidatePath) {
			continue
		}

		oldBase := strings.TrimSuffix(filepath.Base(candidatePath), filepath.Ext(candidatePath))
		if strings.EqualFold(oldBase, newBase) {
			exact = append(exact, candidate)
			continue
		}
		if matchesMedia(mediaFromCandidate(candidate, root), media) {
			sameMetadata = append(sameMetadata, candidate)
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("library: recorrer candidatas para %q: %w", path, err)
	}
	if err := rows.Close(); err != nil {
		return false, fmt.Errorf("library: cerrar candidatas para %q: %w", path, err)
	}

	candidates := exact
	if len(candidates) == 0 {
		candidates = sameMetadata
	}
	if len(candidates) != 1 {
		return false, nil
	}

	if _, err := s.db.ExecContext(ctx,
		`UPDATE media_items SET path = ? WHERE id = ?`,
		path, candidates[0].id,
	); err != nil {
		return false, fmt.Errorf("library: reenlazar %q: %w", path, err)
	}
	return true, nil
}

func mediaItemExists(ctx context.Context, db *sql.DB, libraryID int64, path string) (bool, error) {
	var one int
	err := db.QueryRowContext(ctx,
		`SELECT 1 FROM media_items WHERE library_id = ? AND path = ?`,
		libraryID, path,
	).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("library: comprobar %q: %w", path, err)
	}
	return true, nil
}

func fileMissing(path string) bool {
	_, err := os.Stat(path)
	return errors.Is(err, fs.ErrNotExist)
}

func mediaFromCandidate(candidate relinkCandidate, root string) Media {
	media := Media{
		Title: strings.TrimSpace(candidate.title),
		Kind:  candidate.kind,
	}
	if candidate.year.Valid {
		media.Year = int(candidate.year.Int64)
	}
	if candidate.season.Valid {
		media.Season = int(candidate.season.Int64)
	}
	if candidate.episode.Valid {
		media.Episode = int(candidate.episode.Int64)
	}

	// Las filas antiguas pueden no tener algún dato; se completa con el nombre
	// de la ruta relativa, que es la misma fuente que usa el escáner.
	needsParse := media.Title == "" || media.Kind == "" ||
		!candidate.year.Valid || media.Year == 0
	if media.Kind == KindEpisode {
		needsParse = needsParse || !candidate.season.Valid || media.Season == 0 ||
			!candidate.episode.Valid || media.Episode == 0
	}
	if needsParse {
		rel, err := filepath.Rel(root, candidate.path)
		if err == nil {
			parsed := ParsePath(rel)
			if media.Title == "" {
				media.Title = parsed.Title
			}
			if media.Kind == "" {
				media.Kind = parsed.Kind
			}
			if !candidate.year.Valid || media.Year == 0 {
				media.Year = parsed.Year
			}
			if media.Kind == KindEpisode {
				if !candidate.season.Valid || media.Season == 0 {
					media.Season = parsed.Season
				}
				if !candidate.episode.Valid || media.Episode == 0 {
					media.Episode = parsed.Episode
				}
			}
		}
	}
	return media
}

func matchesMedia(old, current Media) bool {
	if old.Kind != current.Kind || old.Year != current.Year ||
		normalizeMediaTitle(old.Title) != normalizeMediaTitle(current.Title) {
		return false
	}
	if current.Kind == KindEpisode {
		return old.Season == current.Season && old.Episode == current.Episode
	}
	return true
}

func normalizeMediaTitle(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}
