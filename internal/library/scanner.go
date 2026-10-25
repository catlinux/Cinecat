package library

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/catlinux/cinecat/internal/metadata"
)

// videoExtensions son las extensiones que se consideran vídeo.
var videoExtensions = map[string]bool{
	".mkv": true, ".mp4": true, ".avi": true, ".mov": true,
	".m4v": true, ".ts": true, ".m2ts": true,
}

// IsVideoFile indica si el nombre tiene una extensión de vídeo soportada.
func IsVideoFile(name string) bool {
	return videoExtensions[strings.ToLower(filepath.Ext(name))]
}

// Scanner recorre las bibliotecas dadas de alta y puebla media_items.
type Scanner struct {
	db   *sql.DB
	omdb *metadata.OMDbClient
}

// New crea un escáner sobre la base de datos indicada.
func New(db *sql.DB) *Scanner {
	return &Scanner{db: db}
}

// WithOMDb activa la consulta de metadatos de OMDb para películas. Si no se
// llama, el escáner se comporta exactamente como antes y no hace peticiones
// externas.
func (s *Scanner) WithOMDb(client *metadata.OMDbClient) *Scanner {
	s.omdb = client
	return s
}

// Result resume un escaneo. Warnings recoge rutas ilegibles o bibliotecas
// inexistentes, que no abortan el resto del escaneo.
type Result struct {
	Libraries int
	Files     int
	Upserted  int
	Warnings  []string
}

// Scan recorre todas las bibliotecas de la tabla libraries.
func (s *Scanner) Scan(ctx context.Context) (Result, error) {
	var res Result

	type library struct {
		id   int64
		path string
	}
	var libraries []library

	rows, err := s.db.QueryContext(ctx, `SELECT id, path FROM libraries ORDER BY id`)
	if err != nil {
		return res, fmt.Errorf("library: listar bibliotecas: %w", err)
	}
	for rows.Next() {
		var l library
		if err := rows.Scan(&l.id, &l.path); err != nil {
			rows.Close()
			return res, fmt.Errorf("library: leer biblioteca: %w", err)
		}
		libraries = append(libraries, l)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return res, fmt.Errorf("library: recorrer bibliotecas: %w", err)
	}
	rows.Close()

	for _, l := range libraries {
		res.Libraries++
		if err := s.scanLibrary(ctx, l.id, l.path, &res); err != nil {
			return res, err
		}
	}
	return res, nil
}

func (s *Scanner) scanLibrary(ctx context.Context, libraryID int64, root string, res *Result) error {
	info, err := os.Stat(root)
	if err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", root, err))
		return nil
	}
	if !info.IsDir() {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no es un directorio", root))
		return nil
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", path, walkErr))
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		// No seguir enlaces simbólicos: podrían salir de la biblioteca.
		if d.Type()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			if path == root {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") || junkFolders[strings.ToLower(name)] {
				return fs.SkipDir
			}
			return nil
		}

		if strings.HasPrefix(d.Name(), ".") || !IsVideoFile(d.Name()) {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", path, err))
			return nil
		}

		media := ParsePath(rel)
		if media.Title == "" {
			media.Title = strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		}

		if err := upsertMediaItem(ctx, s.db, libraryID, path, media); err != nil {
			return err
		}
		res.Files++
		res.Upserted++

		if s.omdb != nil && media.Kind == KindMovie {
			s.enrichMovie(ctx, libraryID, path, media, res)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("library: escanear %q: %w", root, err)
	}
	return nil
}

// enrichMovie busca sinopsis y cartel en OMDb solo si el ítem todavía no los
// tiene. Los fallos de metadatos se acumulan como avisos, nunca abortan el
// escaneo.
func (s *Scanner) enrichMovie(ctx context.Context, libraryID int64, path string, media Media, res *Result) {
	if s.omdb == nil {
		return
	}

	var overview, poster sql.NullString
	if err := s.db.QueryRowContext(ctx,
		`SELECT overview, poster_url FROM media_items WHERE library_id = ? AND path = ?`,
		libraryID, path,
	).Scan(&overview, &poster); err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudo comprobar los metadatos: %v", path, err))
		return
	}
	if strings.TrimSpace(overview.String) != "" || strings.TrimSpace(poster.String) != "" {
		return
	}

	result, err := s.omdb.Search(ctx, media.Title, media.Year)
	if err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", path, err))
		return
	}

	var overviewValue, posterValue any
	if strings.TrimSpace(result.Plot) != "" {
		overviewValue = result.Plot
	}
	if strings.TrimSpace(result.PosterURL) != "" {
		posterValue = result.PosterURL
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE media_items SET overview = ?, poster_url = ? WHERE library_id = ? AND path = ?`,
		overviewValue, posterValue, libraryID, path,
	); err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudieron guardar los metadatos: %v", path, err))
	}
}

func upsertMediaItem(ctx context.Context, db *sql.DB, libraryID int64, path string, media Media) error {
	var year, season, episode any
	if media.Year > 0 {
		year = media.Year
	}
	if media.Season > 0 {
		season = media.Season
	}
	if media.Episode > 0 {
		episode = media.Episode
	}

	_, err := db.ExecContext(ctx, `
		INSERT INTO media_items (library_id, path, title, year, kind, season, episode)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (library_id, path) DO UPDATE SET
			title = excluded.title,
			year = excluded.year,
			kind = excluded.kind,
			season = excluded.season,
			episode = excluded.episode`,
		libraryID, path, media.Title, year, media.Kind, season, episode,
	)
	if err != nil {
		return fmt.Errorf("library: guardar %q: %w", path, err)
	}
	return nil
}
