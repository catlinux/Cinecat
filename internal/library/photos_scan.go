package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type photoFile struct {
	path string
	root string
}

func (s *Scanner) scanPhotoFiles(ctx context.Context, libraryID int64, files []photoFile, seen map[string]bool, res *Result) error {
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
			if err := s.scanPhotoFile(ctx, libraryID, item.root, item.path, seen, res, &mu); err != nil {
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

func (s *Scanner) scanPhotoFile(
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
		`SELECT size, mtime FROM photos WHERE path = ?`, path).
		Scan(&storedSize, &storedMTime)
	switch {
	case err == nil && storedSize == size && storedMTime == mtime:
		mu.Lock()
		seen[filepath.Clean(path)] = true
		res.Files++
		res.Upserted++
		mu.Unlock()
		return nil
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("library: leer foto existente %q: %w", path, err)
	}

	folder, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("library: calcular carpeta de %q: %w", path, err)
	}
	folder = filepath.ToSlash(folder)
	if folder == "." {
		folder = ""
	}

	width, height := photoDimensions(path)
	if (width == 0 || height == 0) && !strings.EqualFold(filepath.Ext(path), ".webp") {
		mu.Lock()
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("%s: no se pudieron leer las dimensiones", path))
		mu.Unlock()
	}
	takenAt := photoTakenAt(path)
	var takenAtValue any
	if takenAt != nil {
		takenAtValue = *takenAt
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO photos (
			library_id, path, folder, name, size, mtime, width, height, taken_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (path) DO UPDATE SET
			library_id = excluded.library_id,
			folder = excluded.folder,
			name = excluded.name,
			size = excluded.size,
			mtime = excluded.mtime,
			width = excluded.width,
			height = excluded.height,
			taken_at = excluded.taken_at`,
		libraryID, path, folder, filepath.Base(path), size, mtime, width, height, takenAtValue,
	); err != nil {
		return fmt.Errorf("library: guardar foto %q: %w", path, err)
	}

	mu.Lock()
	seen[filepath.Clean(path)] = true
	res.Files++
	res.Upserted++
	mu.Unlock()
	return nil
}

func photoDimensions(path string) (int, int) {
	if strings.EqualFold(filepath.Ext(path), ".webp") {
		// Sin dependencia de WebP no se pueden leer las dimensiones.
		return 0, 0
	}
	file, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0
	}
	return config.Width, config.Height
}

// prunePhotosLibrary conserva las fotos de carpetas no disponibles o vacías,
// igual que la poda de música, para no borrar una biblioteca desmontada.
func (s *Scanner) prunePhotosLibrary(
	ctx context.Context,
	libraryID int64,
	seen map[string]bool,
	scannedRoots []string,
	res *Result,
) error {
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
		`SELECT id, path FROM photos WHERE library_id = ?`, libraryID)
	if err != nil {
		return fmt.Errorf("library: listar fotos para podar: %w", err)
	}
	ids := make([]int64, 0)
	emptyRoots := make(map[string]bool)
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			rows.Close()
			return fmt.Errorf("library: leer foto para podar: %w", err)
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
		return fmt.Errorf("library: recorrer fotos para podar: %w", err)
	}
	rows.Close()
	for root := range emptyRoots {
		res.Warnings = append(res.Warnings,
			fmt.Sprintf("La carpeta %s está vacía pero tiene fotos; se conservan por si el disco no está montado", root))
	}
	if len(ids) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("library: iniciar poda de fotos: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `DELETE FROM photos WHERE id = ?`, id); err != nil {
			return fmt.Errorf("library: borrar foto %d: %w", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("library: confirmar poda de fotos: %w", err)
	}
	return nil
}

func deletePhotosUnderPath(ctx context.Context, tx *sql.Tx, libraryID int64, removed string) error {
	rows, err := tx.QueryContext(ctx,
		`SELECT id, path FROM photos WHERE library_id = ?`, libraryID)
	if err != nil {
		return fmt.Errorf("library: listar fotos de la carpeta: %w", err)
	}
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			rows.Close()
			return fmt.Errorf("library: leer foto de la carpeta: %w", err)
		}
		if pathWithin(removed, path) {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("library: recorrer fotos de la carpeta: %w", err)
	}
	rows.Close()
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `DELETE FROM photos WHERE id = ?`, id); err != nil {
			return fmt.Errorf("library: borrar foto de la carpeta %d: %w", id, err)
		}
	}
	return nil
}
