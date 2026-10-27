package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/catlinux/cinecat/internal/playback"
)

// MusicInspector obtiene etiquetas y datos técnicos de un archivo de audio.
type MusicInspector interface {
	InspectAudio(ctx context.Context, path string) (*playback.AudioInfo, error)
}

// WithMusicInspector activa la lectura de etiquetas con ffprobe para las
// bibliotecas de tipo music.
func (s *Scanner) WithMusicInspector(inspector MusicInspector) *Scanner {
	s.musicInspector = inspector
	return s
}

type musicFile struct {
	path string
	root string
}

func (s *Scanner) scanMusicFiles(ctx context.Context, libraryID int64, files []musicFile, seen map[string]bool, res *Result) error {
	if len(files) == 0 {
		return nil
	}
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	for _, item := range files {
		item := item
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			if err := s.scanMusicFile(ctx, libraryID, item.root, item.path, seen, res, &mu); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return err
	}
	return firstErr
}

func (s *Scanner) scanMusicFile(
	ctx context.Context,
	libraryID int64,
	root string,
	path string,
	seen map[string]bool,
	res *Result,
	mu *sync.Mutex,
) error {
	stat, err := os.Stat(path)
	if err != nil || stat.IsDir() {
		return nil
	}
	size := stat.Size()
	mtime := stat.ModTime().Unix()
	var storedSize, storedMTime int64
	err = s.db.QueryRowContext(ctx,
		`SELECT size_bytes, mtime_unix FROM music_tracks WHERE library_id = ? AND path = ?`,
		libraryID, path).Scan(&storedSize, &storedMTime)
	switch {
	case err == nil && storedSize == size && storedMTime == mtime:
		mu.Lock()
		seen[filepath.Clean(path)] = true
		res.Files++
		res.Upserted++
		mu.Unlock()
		return nil
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("library: leer pista existente %q: %w", path, err)
	}

	var info *playback.AudioInfo
	if s.musicInspector != nil {
		inspected, inspectErr := s.musicInspector.InspectAudio(ctx, path)
		if inspectErr != nil {
			mu.Lock()
			res.Warnings = append(res.Warnings,
				fmt.Sprintf("%s: no se pudieron leer las etiquetas: %v", path, inspectErr))
			mu.Unlock()
		} else {
			info = inspected
		}
	}
	if info == nil {
		info = &playback.AudioInfo{}
	}
	tags := info.Tags
	dir := filepath.Dir(path)
	artistName := normalizeAlbumArtist(tags.AlbumArtist)
	if artistName == "" {
		artistName = normalizeAlbumArtist(tags.Artist)
	}
	if artistName == "" {
		artistName = musicArtistForPath(root, path)
	}
	if artistName == "" {
		artistName = unknownMusicArtist
	}
	albumTitle := strings.TrimSpace(tags.Album)
	if albumTitle == "" {
		albumFolder := dir
		if filepath.Clean(albumFolder) != filepath.Clean(root) && isDiscFolder(filepath.Base(albumFolder)) {
			albumFolder = filepath.Dir(albumFolder)
		}
		albumTitle = filepath.Base(albumFolder)
	}
	title := strings.TrimSpace(tags.Title)
	if title == "" {
		title = musicFileTitle(filepath.Base(path))
	}
	disc := tags.Disc
	if disc <= 0 {
		disc = 1
	}
	cover := findAlbumCover(dir)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("library: iniciar pista musical %q: %w", path, err)
	}
	defer func() { _ = tx.Rollback() }()

	artistID, err := ensureMusicArtist(ctx, tx, artistName)
	if err != nil {
		return err
	}
	albumID, err := ensureMusicAlbum(ctx, tx, libraryID, artistID, albumTitle, tags.Year, cover)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO music_tracks (
			library_id, album_id, artist_id, path, title, track_no, disc_no,
			year, genre, duration_seconds, codec, bitrate, sample_rate, channels,
			size_bytes, mtime_unix, has_embedded_cover
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (library_id, path) DO UPDATE SET
			album_id = excluded.album_id,
			artist_id = excluded.artist_id,
			title = excluded.title,
			track_no = excluded.track_no,
			disc_no = excluded.disc_no,
			year = excluded.year,
			genre = excluded.genre,
			duration_seconds = excluded.duration_seconds,
			codec = excluded.codec,
			bitrate = excluded.bitrate,
			sample_rate = excluded.sample_rate,
			channels = excluded.channels,
			size_bytes = excluded.size_bytes,
			mtime_unix = excluded.mtime_unix,
			has_embedded_cover = excluded.has_embedded_cover`,
		libraryID, albumID, artistID, path, title, nullableInt(tags.Track), disc,
		nullableInt(tags.Year), nullableText(tags.Genre), nullableFloat(info.DurationSeconds),
		nullableText(info.Codec), nullableInt64(info.BitRate), nullableInt(info.SampleRate),
		nullableInt(info.Channels), size, mtime, boolInt(info.HasEmbeddedCover),
	); err != nil {
		return fmt.Errorf("library: guardar pista musical %q: %w", path, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("library: confirmar pista musical %q: %w", path, err)
	}

	mu.Lock()
	seen[filepath.Clean(path)] = true
	res.Files++
	res.Upserted++
	mu.Unlock()
	return nil
}

func ensureMusicArtist(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	sortName := musicSortName(name)
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO music_artists (name, sort_name) VALUES (?, ?)`,
		name, sortName); err != nil {
		return 0, fmt.Errorf("library: guardar artista %q: %w", name, err)
	}
	var id int64
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM music_artists WHERE name = ? COLLATE NOCASE`, name).Scan(&id); err != nil {
		return 0, fmt.Errorf("library: leer artista %q: %w", name, err)
	}
	return id, nil
}

func ensureMusicAlbum(
	ctx context.Context,
	tx *sql.Tx,
	libraryID, artistID int64,
	title string,
	year int,
	cover string,
) (int64, error) {
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO music_albums (library_id, artist_id, title, year, cover_path)
		VALUES (?, ?, ?, ?, ?)`,
		libraryID, artistID, title, nullableInt(year), nullableText(cover)); err != nil {
		return 0, fmt.Errorf("library: guardar álbum %q: %w", title, err)
	}
	var id int64
	if err := tx.QueryRowContext(ctx, `
		SELECT id FROM music_albums
		 WHERE library_id = ? AND artist_id = ? AND title = ? COLLATE NOCASE`,
		libraryID, artistID, title).Scan(&id); err != nil {
		return 0, fmt.Errorf("library: leer álbum %q: %w", title, err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE music_albums SET
			year = COALESCE(?, year),
			cover_path = COALESCE(?, cover_path)
		 WHERE id = ?`, nullableInt(year), nullableText(cover), id); err != nil {
		return 0, fmt.Errorf("library: actualizar álbum %q: %w", title, err)
	}
	return id, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nullableInt64(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}

// pruneMusicLibrary conserva las pistas de carpetas no disponibles o vacías,
// igual que la poda de vídeo, y elimina después álbumes y artistas huérfanos.
func (s *Scanner) pruneMusicLibrary(ctx context.Context, libraryID int64, seen map[string]bool, scannedRoots []string, res *Result) error {
	if len(scannedRoots) == 0 {
		return nil
	}
	rootsWithFiles := make(map[string]bool, len(scannedRoots))
	for path := range seen {
		for _, root := range scannedRoots {
			if pathWithin(root, path) {
				rootsWithFiles[root] = true
			}
		}
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, path FROM music_tracks WHERE library_id = ?`, libraryID)
	if err != nil {
		return fmt.Errorf("library: listar pistas para podar: %w", err)
	}
	ids := make([]int64, 0)
	emptyRoots := make(map[string]bool)
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			rows.Close()
			return fmt.Errorf("library: leer pista para podar: %w", err)
		}
		if !pathWithinAny(scannedRoots, path) || seen[filepath.Clean(path)] {
			continue
		}
		skip := false
		for _, root := range scannedRoots {
			if pathWithin(root, path) && !rootsWithFiles[root] {
				emptyRoots[root] = true
				skip = true
				break
			}
		}
		if !skip {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("library: recorrer pistas para podar: %w", err)
	}
	rows.Close()
	for root := range emptyRoots {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("La carpeta %s está vacía pero tiene pistas; se conservan por si el disco no está montado", root))
	}
	if len(ids) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("library: iniciar poda de música: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `DELETE FROM music_tracks WHERE id = ?`, id); err != nil {
			return fmt.Errorf("library: borrar pista musical %d: %w", id, err)
		}
	}
	if err := cleanupMusicOrphans(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("library: confirmar poda de música: %w", err)
	}
	return nil
}

func cleanupMusicOrphans(ctx context.Context, db sqlExecer) error {
	if _, err := db.ExecContext(ctx, `
		DELETE FROM music_albums
		 WHERE NOT EXISTS (
		       SELECT 1 FROM music_tracks mt WHERE mt.album_id = music_albums.id
		   )`); err != nil {
		return fmt.Errorf("library: borrar álbumes huérfanos: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
		DELETE FROM music_artists
		 WHERE NOT EXISTS (SELECT 1 FROM music_albums ma WHERE ma.artist_id = music_artists.id)
		   AND NOT EXISTS (SELECT 1 FROM music_tracks mt WHERE mt.artist_id = music_artists.id)`); err != nil {
		return fmt.Errorf("library: borrar artistas huérfanos: %w", err)
	}
	return nil
}

func deleteMusicTracksUnderPath(ctx context.Context, tx *sql.Tx, libraryID int64, removed string) error {
	rows, err := tx.QueryContext(ctx,
		`SELECT id, path FROM music_tracks WHERE library_id = ?`, libraryID)
	if err != nil {
		return fmt.Errorf("library: listar pistas de la carpeta: %w", err)
	}
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			rows.Close()
			return fmt.Errorf("library: leer pista de la carpeta: %w", err)
		}
		if pathWithin(removed, path) {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("library: recorrer pistas de la carpeta: %w", err)
	}
	rows.Close()
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `DELETE FROM music_tracks WHERE id = ?`, id); err != nil {
			return fmt.Errorf("library: borrar pista de la carpeta %d: %w", id, err)
		}
	}
	return nil
}
