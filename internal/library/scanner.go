package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

// TMDBClient es lo que necesita el escáner para identificar y guardar fichas.
type TMDBClient interface {
	MovieSearcher
	MovieDetails(ctx context.Context, id int) (*metadata.Details, error)
}

// Scanner recorre las bibliotecas dadas de alta y puebla media_items.
type Scanner struct {
	db   *sql.DB
	omdb *metadata.OMDbClient
	tmdb TMDBClient
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

// WithTMDB activa la identificación y las fichas de TMDB para películas. Si se
// llama, OMDb deja de usarse aunque también se haya configurado.
func (s *Scanner) WithTMDB(client TMDBClient) *Scanner {
	s.tmdb = client
	return s
}

// Result resume un escaneo. Warnings recoge rutas ilegibles o bibliotecas
// inexistentes, que no abortan el resto del escaneo.
type Result struct {
	Libraries    int
	Files        int
	Upserted     int
	Identified   int
	Unidentified int
	Warnings     []string
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

		if media.Kind == KindMovie {
			switch {
			case s.tmdb != nil:
				s.enrichMovieTMDB(ctx, libraryID, path, media, res)
			case s.omdb != nil:
				s.enrichMovie(ctx, libraryID, path, media, res)
			}
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
		`UPDATE media_items SET overview = ?, poster_url = ?, metadata_source = 'omdb'
		 WHERE library_id = ? AND path = ?`,
		overviewValue, posterValue, libraryID, path,
	); err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudieron guardar los metadatos: %v", path, err))
	}
}

// enrichMovieTMDB identifica la película y guarda su ficha. Respeta el estado
// de corrección manual: lo que el usuario ha fijado no se vuelve a consultar.
func (s *Scanner) enrichMovieTMDB(ctx context.Context, libraryID int64, path string, media Media, res *Result) {
	var itemID int64
	var matchStatus, source, displayTitle sql.NullString
	var tmdbID sql.NullInt64
	var matchScore sql.NullFloat64
	if err := s.db.QueryRowContext(ctx,
		`SELECT id, match_status, metadata_source, tmdb_id, match_score, display_title
		   FROM media_items WHERE library_id = ? AND path = ?`,
		libraryID, path,
	).Scan(&itemID, &matchStatus, &source, &tmdbID, &matchScore, &displayTitle); err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudo comprobar los metadatos: %v", path, err))
		return
	}

	if matchStatus.Valid {
		// Relleno de las fichas guardadas antes de que existiera
		// display_title: se conserva el estado y la puntuación de la fila.
		if (matchStatus.String == "auto" || matchStatus.String == "manual") &&
			source.Valid && source.String == "tmdb" &&
			tmdbID.Valid && tmdbID.Int64 > 0 && !displayTitle.Valid {
			score := 0.0
			if matchScore.Valid {
				score = matchScore.Float64
			}
			details, err := s.tmdb.MovieDetails(ctx, int(tmdbID.Int64))
			if err != nil {
				res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudo obtener la ficha: %v", path, err))
				return
			}
			if err := ApplyMovieDetails(ctx, s.db, itemID, details, matchStatus.String, score); err != nil {
				res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudieron guardar los metadatos: %v", path, err))
				return
			}
			if !matchScore.Valid {
				if _, err := s.db.ExecContext(ctx,
					`UPDATE media_items SET match_score = NULL WHERE id = ?`, itemID); err != nil {
					res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudieron guardar los metadatos: %v", path, err))
				}
			}
			return
		}

		switch matchStatus.String {
		case "manual", "none":
			return
		case "auto":
			if source.Valid && source.String == "tmdb" {
				return
			}
		}
	}

	candidate, score, ok, err := IdentifyMovie(ctx, s.tmdb, media.Title, media.Year)
	if err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudo identificar: %v", path, err))
		return
	}
	if !ok {
		// Los metadatos anteriores podían ser de otra película o estar en
		// inglés, así que no se conservan.
		if err := ClearMovieMetadata(ctx, s.db, itemID, nil); err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudieron borrar los metadatos: %v", path, err))
		}
		res.Unidentified++
		return
	}

	details, err := s.tmdb.MovieDetails(ctx, candidate.TMDBID)
	if err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudo obtener la ficha: %v", path, err))
		return
	}
	if err := ApplyMovieDetails(ctx, s.db, itemID, details, "auto", score); err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudieron guardar los metadatos: %v", path, err))
		return
	}
	res.Identified++
}

// ApplyMovieDetails guarda la ficha completa en la fila. El título analizado
// (title) no se toca; el de TMDB se guarda en display_title.
func ApplyMovieDetails(ctx context.Context, db *sql.DB, itemID int64, d *metadata.Details, status string, score float64) error {
	sortSource := d.Title
	if strings.TrimSpace(sortSource) == "" {
		if err := db.QueryRowContext(ctx, `SELECT title FROM media_items WHERE id = ?`, itemID).Scan(&sortSource); err != nil {
			return fmt.Errorf("library: leer el título de la película %d: %w", itemID, err)
		}
	}
	sortTitle, sortLetter := sortKeys(sortSource)
	_, err := db.ExecContext(ctx, `
		UPDATE media_items SET
			tmdb_id = ?, original_title = ?, imdb_id = ?, tagline = ?,
			overview = ?, overview_language = ?, year = ?, display_title = ?,
			sort_title = ?, sort_letter = ?, genres = ?, runtime_minutes = ?,
			rating = ?, directors = ?, cast_names = ?, poster_url = ?,
			backdrop_url = ?, metadata_source = 'tmdb', match_status = ?,
			match_score = ?
		WHERE id = ?`,
		nullableInt(d.TMDBID), nullableText(d.OriginalTitle), nullableText(d.IMDbID), nullableText(d.Tagline),
		nullableText(d.Overview), nullableText(d.OverviewLanguage), nullableInt(d.Year), nullableText(d.Title),
		sortTitle, sortLetter, jsonArray(d.Genres), nullableInt(d.RuntimeMinutes), nullableFloat(d.Rating),
		jsonArray(d.Directors), jsonArray(d.Cast), nullableText(d.PosterURL), nullableText(d.BackdropURL),
		nullableText(status), score, itemID,
	)
	if err != nil {
		return fmt.Errorf("library: guardar la ficha de la película %d: %w", itemID, err)
	}
	return nil
}

// ClearMovieMetadata deja la fila sin ficha, conservando el título y el año
// analizados del archivo.
func ClearMovieMetadata(ctx context.Context, db *sql.DB, itemID int64, status any) error {
	var title string
	err := db.QueryRowContext(ctx, `SELECT title FROM media_items WHERE id = ?`, itemID).Scan(&title)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("library: leer el título de la película %d: %w", itemID, err)
	}
	sortTitle, sortLetter := sortKeys(title)

	_, err = db.ExecContext(ctx, `
		UPDATE media_items SET
			tmdb_id = NULL, original_title = NULL, imdb_id = NULL, tagline = NULL,
			overview = NULL, overview_language = NULL, display_title = NULL,
			sort_title = ?, sort_letter = ?, genres = NULL,
			runtime_minutes = NULL, rating = NULL, directors = NULL,
			cast_names = NULL, poster_url = NULL, backdrop_url = NULL,
			metadata_source = NULL, match_score = NULL, match_status = ?
		WHERE id = ?`,
		sortTitle, sortLetter, status, itemID,
	)
	if err != nil {
		return fmt.Errorf("library: borrar la ficha de la película %d: %w", itemID, err)
	}
	return nil
}

func nullableText(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nullableInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func nullableFloat(value float64) any {
	if value == 0 {
		return nil
	}
	return value
}

func jsonArray(values []string) any {
	if len(values) == 0 {
		return nil
	}
	data, err := json.Marshal(values)
	if err != nil {
		return nil
	}
	return string(data)
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
			-- El año de una ficha de TMDB es más fiable que el del nombre del
			-- archivo, que casi nunca lo lleva.
			year = CASE WHEN media_items.metadata_source = 'tmdb' THEN media_items.year ELSE excluded.year END,
			kind = excluded.kind,
			season = excluded.season,
			episode = excluded.episode`,
		libraryID, path, media.Title, year, media.Kind, season, episode,
	)
	if err != nil {
		return fmt.Errorf("library: guardar %q: %w", path, err)
	}
	return refreshSortKeysByPath(ctx, db, libraryID, path)
}
