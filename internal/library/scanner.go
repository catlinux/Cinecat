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
	"time"

	"github.com/catlinux/cinecat/internal/metadata"
	"github.com/catlinux/cinecat/internal/playback"
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

// TMDBClient es lo que necesita el escáner para identificar y guardar fichas
// de películas y series.
type TMDBClient interface {
	MovieSearcher
	TVSearcher
	MovieDetails(ctx context.Context, id int) (*metadata.Details, error)
	TVDetails(ctx context.Context, id int) (*metadata.Details, error)
	SeasonDetails(ctx context.Context, tvID, season int) (*metadata.Season, error)
	MovieImages(ctx context.Context, tmdbID int) (*metadata.ImageOptions, error)
	TVImages(ctx context.Context, tvID int) (*metadata.ImageOptions, error)
	SearchCollections(ctx context.Context, query string) ([]metadata.CollectionCandidate, error)
	CollectionDetails(ctx context.Context, id int) (*metadata.CollectionInfo, error)
}

// Scanner recorre las bibliotecas dadas de alta y puebla sus tablas.
type Scanner struct {
	db             *sql.DB
	omdb           *metadata.OMDbClient
	tmdb           TMDBClient
	musicInspector MusicInspector
	stableAge      time.Duration
}

// New crea un escáner sobre la base de datos indicada.
func New(db *sql.DB) *Scanner {
	return &Scanner{
		db:             db,
		musicInspector: playback.NewProber(),
		stableAge:      2 * time.Minute,
	}
}

// WithStableAge cambia el tiempo mínimo que debe llevar quieto un archivo
// para actualizarlo. Un valor de cero desactiva la comprobación.
func (s *Scanner) WithStableAge(d time.Duration) *Scanner {
	s.stableAge = d
	return s
}

// WithOMDb activa la consulta de metadatos de OMDb para películas. Si no se
// llama, el escáner se comporta exactamente como antes y no hace peticiones
// externas.
func (s *Scanner) WithOMDb(client *metadata.OMDbClient) *Scanner {
	s.omdb = client
	return s
}

// WithTMDB activa la identificación y las fichas de TMDB para películas y
// series. Si se llama, OMDb deja de usarse aunque también se haya configurado.
func (s *Scanner) WithTMDB(client TMDBClient) *Scanner {
	s.tmdb = client
	return s
}

// Result resume un escaneo. Warnings recoge rutas ilegibles o bibliotecas
// inexistentes, que no abortan el resto del escaneo.
type Result struct {
	Libraries          int
	Files              int
	Upserted           int
	Relinked           int
	SkippedUnstable    int
	Identified         int
	Unidentified       int
	SeriesIdentified   int
	SeriesUnidentified int
	Warnings           []string
}

// Scan recorre las bibliotecas de tipo vídeo y clips. Cada biblioteca puede
// tener varias carpetas; si una carpeta no está disponible no se podan sus
// fichas y se avisa en el resultado.
func (s *Scanner) Scan(ctx context.Context) (Result, error) {
	var res Result

	type library struct {
		id   int64
		name string
		path string
		kind string
	}
	var libraries []library

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, path, kind FROM libraries ORDER BY id`)
	if err != nil {
		return res, fmt.Errorf("library: listar bibliotecas: %w", err)
	}
	for rows.Next() {
		var l library
		if err := rows.Scan(&l.id, &l.name, &l.path, &l.kind); err != nil {
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
		if l.kind != "video" && l.kind != "clips" && l.kind != "music" {
			res.Warnings = append(res.Warnings,
				fmt.Sprintf("%s: tipo %s pendiente de implementar", l.name, l.kind))
			continue
		}
		paths, err := loadLibraryPathStrings(ctx, s.db, l.id, l.path)
		if err != nil {
			return res, fmt.Errorf("library: leer carpetas de %s: %w", l.name, err)
		}
		seriesIDs := map[string]int64{}
		seen := make(map[string]bool)
		unstableDirs := make(map[string]bool)
		scannedRoots := make([]string, 0, len(paths))
		for _, root := range paths {
			ok, err := s.scanLibraryPath(ctx, l.id, l.kind, root, seriesIDs, seen, unstableDirs, &res)
			if err != nil {
				return res, err
			}
			if ok {
				scannedRoots = append(scannedRoots, filepath.Clean(root))
			}
		}
		if l.kind == "music" {
			if err := s.pruneMusicLibrary(ctx, l.id, seen, scannedRoots, &res); err != nil {
				return res, err
			}
			continue
		}
		if err := s.pruneLibrary(ctx, l.id, seen, scannedRoots, unstableDirs, &res); err != nil {
			return res, err
		}
		if l.kind == "video" && s.tmdb != nil {
			ids, err := s.seriesIDsWithEpisodes(ctx, l.id)
			if err != nil {
				return res, err
			}
			for _, id := range ids {
				s.enrichSeriesTMDB(ctx, id, &res)
			}
		}
	}
	if err := cleanupOrphans(ctx, s.db); err != nil {
		return res, err
	}
	if err := cleanupMusicOrphans(ctx, s.db); err != nil {
		return res, err
	}
	return res, nil
}

func (s *Scanner) scanLibraryPath(
	ctx context.Context,
	libraryID int64,
	libraryKind string,
	root string,
	seriesIDs map[string]int64,
	seen map[string]bool,
	unstableDirs map[string]bool,
	res *Result,
) (bool, error) {
	info, err := os.Lstat(root)
	if err != nil {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("La carpeta %s no está disponible; se conservan sus fichas", root))
		return false, nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("La carpeta %s no está disponible; se conservan sus fichas", root))
		return false, nil
	}
	if !info.IsDir() {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("La carpeta %s no está disponible; se conservan sus fichas", root))
		return false, nil
	}

	musicFiles := make([]musicFile, 0)
	walkFailed := false
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", path, walkErr))
			walkFailed = true
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

		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		if libraryKind == "music" {
			if IsAudioFile(d.Name()) {
				musicFiles = append(musicFiles, musicFile{path: path, root: root})
			}
			return nil
		}
		if !IsVideoFile(d.Name()) {
			return nil
		}

		if libraryKind == "clips" {
			if err := upsertClipItem(ctx, s.db, libraryID, path, clipTitle(d.Name())); err != nil {
				return err
			}
			seen[filepath.Clean(path)] = true
			res.Files++
			res.Upserted++
			return nil
		}

		if libraryKind == "video" && s.stableAge > 0 {
			info, err := d.Info()
			if err != nil {
				res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", path, err))
				walkFailed = true
				return nil
			}
			if time.Since(info.ModTime()) < s.stableAge {
				res.SkippedUnstable++
				cleanPath := filepath.Clean(path)
				exists, err := mediaItemExists(ctx, s.db, libraryID, path)
				if err != nil {
					return err
				}
				if exists {
					seen[cleanPath] = true
				} else {
					unstableDirs[filepath.Dir(cleanPath)] = true
				}
				return nil
			}
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", path, err))
			walkFailed = true
			return nil
		}

		media := ParsePath(rel)
		if media.Title == "" {
			media.Title = strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
		}

		relinked, err := s.relinkMediaItem(ctx, libraryID, root, path, media)
		if err != nil {
			return err
		}
		if relinked {
			res.Relinked++
		}

		var seriesID any
		if media.Kind == KindEpisode && media.SeriesFolder != "" {
			key := strings.ToLower(media.SeriesFolder)
			id, ok := seriesIDs[key]
			if !ok {
				id, err = ensureSeries(ctx, s.db, libraryID, media.SeriesFolder, media.Title, media.Year)
				if err != nil {
					return err
				}
				seriesIDs[key] = id
			}
			seriesID = id
		}

		if err := upsertMediaItem(ctx, s.db, libraryID, path, media, seriesID); err != nil {
			return err
		}
		seen[filepath.Clean(path)] = true
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
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s: %v", root, err))
		walkFailed = true
	}
	if libraryKind == "music" {
		if err := s.scanMusicFiles(ctx, libraryID, musicFiles, seen, res); err != nil {
			return false, err
		}
	}
	if walkFailed {
		return false, nil
	}
	return true, nil
}

func (s *Scanner) pruneLibrary(
	ctx context.Context,
	libraryID int64,
	seen map[string]bool,
	scannedRoots []string,
	unstableDirs map[string]bool,
	res *Result,
) error {
	if len(scannedRoots) == 0 {
		return nil
	}
	// Un disco sin montar deja su punto de montaje como una carpeta vacía que
	// se recorre «con éxito» y sin archivos: no se poda una carpeta de la que
	// no se ha visto ni un solo archivo si la base tiene fichas suyas.
	rootsWithFiles := make(map[string]bool, len(scannedRoots))
	for path := range seen {
		for _, root := range scannedRoots {
			if pathWithin(root, path) {
				rootsWithFiles[root] = true
			}
		}
	}
	emptyRoots := make(map[string]bool)
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, path FROM media_items WHERE library_id = ?`, libraryID)
	if err != nil {
		return fmt.Errorf("library: listar ítems para podar: %w", err)
	}
	ids := make([]int64, 0)
	prunedPaths := make(map[int64]string)
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			rows.Close()
			return fmt.Errorf("library: leer ítem para podar: %w", err)
		}
		cleanPath := filepath.Clean(path)
		if !pathWithinAny(scannedRoots, cleanPath) || seen[cleanPath] ||
			unstableDirs[filepath.Dir(cleanPath)] {
			continue
		}
		skipped := false
		for _, root := range scannedRoots {
			if pathWithin(root, path) && !rootsWithFiles[root] {
				emptyRoots[root] = true
				skipped = true
				break
			}
		}
		if !skipped {
			ids = append(ids, id)
			prunedPaths[id] = cleanPath
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("library: recorrer ítems para podar: %w", err)
	}
	rows.Close()
	for root := range emptyRoots {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("La carpeta %s está vacía pero tiene fichas; se conservan por si el disco no está montado", root))
	}
	if len(ids) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("library: iniciar poda: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, id := range ids {
		if err := transferToSuccessor(ctx, tx, libraryID, scannedRoots, id, prunedPaths[id], seen); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM media_items WHERE id = ?`, id); err != nil {
			return fmt.Errorf("library: borrar ítem %d: %w", id, err)
		}
	}
	if err := cleanupOrphans(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("library: confirmar poda: %w", err)
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
	var matchStatus, source, displayTitle, credits sql.NullString
	var tmdbID sql.NullInt64
	var matchScore sql.NullFloat64
	var collectionChecked int
	if err := s.db.QueryRowContext(ctx,
		`SELECT id, match_status, metadata_source, tmdb_id, match_score,
		        display_title, collection_checked, credits
		   FROM media_items WHERE library_id = ? AND path = ?`,
		libraryID, path,
	).Scan(&itemID, &matchStatus, &source, &tmdbID, &matchScore,
		&displayTitle, &collectionChecked, &credits); err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudo comprobar los metadatos: %v", path, err))
		return
	}

	if matchStatus.Valid {
		// Las fichas antiguas pueden necesitar display_title, la colección o el
		// reparto. Se unifican las comprobaciones para pedir la ficha una vez.
		if (matchStatus.String == "auto" || matchStatus.String == "manual") &&
			source.Valid && source.String == "tmdb" &&
			tmdbID.Valid && tmdbID.Int64 > 0 &&
			(!displayTitle.Valid || collectionChecked == 0 || !credits.Valid) {
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
			s.fillCollectionOverview(ctx, itemID, path, res)
			if !matchScore.Valid {
				if _, err := s.db.ExecContext(ctx,
					`UPDATE media_items SET match_score = NULL WHERE id = ?`, itemID); err != nil {
					res.Warnings = append(res.Warnings, fmt.Sprintf("%s: no se pudieron guardar los metadatos: %v", path, err))
				}
			}
			return
		}

		// Las fichas completas no vuelven a pedir MovieDetails, pero pueden
		// tener pendiente la sinopsis de una colección creada antes.
		s.fillCollectionOverview(ctx, itemID, path, res)

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
	s.fillCollectionOverview(ctx, itemID, path, res)
	res.Identified++
}

// fillCollectionOverview completa una sola vez la sinopsis de una colección
// automática. Un fallo de TMDB se apunta como aviso y no interrumpe el escaneo.
func (s *Scanner) fillCollectionOverview(ctx context.Context, itemID int64, path string, res *Result) {
	if s.tmdb == nil {
		return
	}

	var collectionID int64
	var tmdbID sql.NullInt64
	var overview sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT c.id, c.tmdb_collection_id, c.overview
		  FROM collections c
		  JOIN collection_items ci ON ci.collection_id = c.id
		 WHERE ci.media_item_id = ?
		   AND c.source = 'tmdb' AND c.edited = 0 AND c.deleted = 0
		 LIMIT 1`, itemID).Scan(&collectionID, &tmdbID, &overview)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return
	case err != nil:
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("%s: no se pudo comprobar la colección: %v", path, err))
		return
	}
	if !tmdbID.Valid || tmdbID.Int64 <= 0 || overview.Valid {
		return
	}

	details, err := s.tmdb.CollectionDetails(ctx, int(tmdbID.Int64))
	if err != nil {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("%s: no se pudo obtener la sinopsis de la colección: %v", path, err))
		return
	}
	if details == nil || strings.TrimSpace(details.Overview) == "" {
		return
	}
	if _, err := s.db.ExecContext(ctx, `
		UPDATE collections SET overview = ?
		 WHERE id = ? AND overview IS NULL`,
		details.Overview, collectionID); err != nil {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("%s: no se pudo guardar la sinopsis de la colección: %v", path, err))
	}
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
			rating = ?, directors = ?, cast_names = ?, credits = ?,
			trailer_url = ?, trailer_language = ?,
			poster_url = CASE WHEN poster_locked = 1 AND tmdb_id = ? THEN poster_url ELSE ? END,
			backdrop_url = CASE WHEN backdrop_locked = 1 AND tmdb_id = ? THEN backdrop_url ELSE ? END,
			poster_locked = CASE WHEN tmdb_id = ? THEN poster_locked ELSE 0 END,
			backdrop_locked = CASE WHEN tmdb_id = ? THEN backdrop_locked ELSE 0 END,
			metadata_source = 'tmdb', match_status = ?, match_score = ?
		WHERE id = ?`,
		nullableInt(d.TMDBID), nullableText(d.OriginalTitle), nullableText(d.IMDbID), nullableText(d.Tagline),
		nullableText(d.Overview), nullableText(d.OverviewLanguage), nullableInt(d.Year), nullableText(d.Title),
		sortTitle, sortLetter, jsonArray(d.Genres), nullableInt(d.RuntimeMinutes), nullableFloat(d.Rating),
		jsonArray(d.Directors), jsonArray(d.Cast), creditsJSON(d.Credits),
		nullableText(d.TrailerURL), nullableText(d.TrailerLanguage),
		d.TMDBID, nullableText(d.PosterURL),
		d.TMDBID, nullableText(d.BackdropURL),
		d.TMDBID, d.TMDBID,
		nullableText(status), score, itemID,
	)
	if err != nil {
		return fmt.Errorf("library: guardar la ficha de la película %d: %w", itemID, err)
	}
	if err := ApplyCollection(ctx, db, itemID, d.Collection); err != nil {
		return err
	}
	return nil
}

// ClearMovieMetadata deja la fila sin ficha, conservando el título y el año
// analizados del archivo.
func ClearMovieMetadata(ctx context.Context, db *sql.DB, itemID int64, status any) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("library: iniciar limpieza de la película %d: %w", itemID, err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var title string
	err = tx.QueryRowContext(ctx, `SELECT title FROM media_items WHERE id = ?`, itemID).Scan(&title)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("library: leer el título de la película %d: %w", itemID, err)
	}
	sortTitle, sortLetter := sortKeys(title)

	if err := clearTMDBCollectionItems(ctx, tx, itemID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE media_items SET
			tmdb_id = NULL, original_title = NULL, imdb_id = NULL, tagline = NULL,
			overview = NULL, overview_language = NULL, display_title = NULL,
			sort_title = ?, sort_letter = ?, genres = NULL,
			runtime_minutes = NULL, rating = NULL, directors = NULL,
			cast_names = NULL, credits = NULL, trailer_url = NULL,
			trailer_language = NULL, poster_url = NULL, backdrop_url = NULL,
			poster_locked = 0, backdrop_locked = 0, collection_checked = 0,
			metadata_source = NULL, match_score = NULL, match_status = ?
		WHERE id = ?`,
		sortTitle, sortLetter, status, itemID,
	)
	if err != nil {
		return fmt.Errorf("library: borrar la ficha de la película %d: %w", itemID, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("library: confirmar limpieza de la película %d: %w", itemID, err)
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

func upsertMediaItem(ctx context.Context, db *sql.DB, libraryID int64, path string, media Media, seriesID any) error {
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
		INSERT INTO media_items (library_id, path, title, year, kind, season, episode, series_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (library_id, path) DO UPDATE SET
			title = excluded.title,
			-- El año de una ficha de TMDB es más fiable que el del nombre del
			-- archivo, que casi nunca lo lleva. En episodios ya enriquecidos se
			-- conserva el año de la temporada.
			year = CASE
				WHEN media_items.metadata_source = 'tmdb' THEN media_items.year
				WHEN media_items.series_id IS NOT NULL AND media_items.year IS NOT NULL
					THEN media_items.year
				ELSE excluded.year
			END,
			kind = excluded.kind,
			season = excluded.season,
			episode = excluded.episode,
			series_id = excluded.series_id`,
		libraryID, path, media.Title, year, media.Kind, season, episode, seriesID,
	)
	if err != nil {
		return fmt.Errorf("library: guardar %q: %w", path, err)
	}
	return refreshSortKeysByPath(ctx, db, libraryID, path)
}
