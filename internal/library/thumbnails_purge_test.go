package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeThumbnailFile(t *testing.T, dir, name string, modTime time.Time) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("miniatura"), 0o644); err != nil {
		t.Fatalf("escribir %s: %v", name, err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("cambiar fecha de %s: %v", name, err)
	}
	return path
}

func TestPurgeOrphanThumbnailsConservaExistentesYAjenos(t *testing.T) {
	db := openTestDB(t)
	libraryID := createLibrary(t, db, t.TempDir())
	itemID := insertItemFixture(t, db, itemFixture{
		libraryID: libraryID,
		path:      "/media/pelicula.mkv",
		title:     "Película",
		kind:      KindMovie,
	})
	cacheDir := t.TempDir()
	api := &API{db: db, thumbnails: NewThumbnailer(cacheDir, nil)}
	now := time.Now()
	old := now.Add(-2 * time.Hour)

	existing := writeThumbnailFile(t, cacheDir,
		fmt.Sprintf("%d-100-200.jpg", itemID), old)
	orphan := writeThumbnailFile(t, cacheDir, "999999-100-200.jpg", old)
	foreignText := writeThumbnailFile(t, cacheDir, "notas.txt", old)
	foreignImage := writeThumbnailFile(t, cacheDir, "cartel.jpg", old)
	foreignSuffix := writeThumbnailFile(t, cacheDir, "999999-100-200.jpg.part", old)
	recentTemp := writeThumbnailFile(t, cacheDir, ".thumb-123.jpg", now)
	recentOrphan := writeThumbnailFile(t, cacheDir, "888888-100-200.jpg", now)

	purged, err := api.purgeOrphanThumbnails(context.Background(), now)
	if err != nil {
		t.Fatalf("purgar miniaturas: %v", err)
	}
	if purged != 1 {
		t.Fatalf("purgadas = %d, se esperaba 1", purged)
	}
	for path, want := range map[string]bool{
		existing:      true,
		orphan:        false,
		foreignText:   true,
		foreignImage:  true,
		foreignSuffix: true,
		recentTemp:    true,
		recentOrphan:  true,
	} {
		_, err := os.Stat(path)
		if want && err != nil {
			t.Errorf("%s debería conservarse: %v", path, err)
		}
		if !want && !os.IsNotExist(err) {
			t.Errorf("%s debería haberse borrado: %v", path, err)
		}
	}
}

func TestPurgeOrphanThumbnailsSinCacheNoFalla(t *testing.T) {
	db := openTestDB(t)
	missing := filepath.Join(t.TempDir(), "no-existe")
	api := &API{db: db, thumbnails: NewThumbnailer(missing, nil)}
	if purged, err := api.purgeOrphanThumbnails(context.Background(), time.Now()); err != nil || purged != 0 {
		t.Fatalf("caché inexistente: purgadas=%d err=%v", purged, err)
	}

	empty := t.TempDir()
	api.thumbnails = NewThumbnailer(empty, nil)
	if purged, err := api.purgeOrphanThumbnails(context.Background(), time.Now()); err != nil || purged != 0 {
		t.Fatalf("caché vacía: purgadas=%d err=%v", purged, err)
	}

	api.thumbnails = nil
	if purged, err := api.purgeOrphanThumbnails(context.Background(), time.Now()); err != nil || purged != 0 {
		t.Fatalf("caché desactivada: purgadas=%d err=%v", purged, err)
	}
}

func TestThumbnailItemIDExigePatronExacto(t *testing.T) {
	for name, want := range map[string]int64{
		"7-123456789-42.jpg":  7,
		"42--1-0.jpg":         42,
		"7-123456789-42.png":  0,
		".thumb-7.jpg":        0,
		"7-123456789-42.jpg2": 0,
		"7-123-42-1.jpg":      0,
		"007-123-42.jpg":      0,
		"7-0123-42.jpg":       0,
		"7-123-042.jpg":       0,
		"sin-id.jpg":          0,
	} {
		got, ok := thumbnailItemID(name)
		if want == 0 {
			if ok {
				t.Errorf("%q no debería coincidir (id=%d)", name, got)
			}
			continue
		}
		if !ok || got != want {
			t.Errorf("%q = %d, %v; se esperaba %d", name, got, ok, want)
		}
	}
}
