package library

import (
	"bytes"
	"context"
	"database/sql"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPhotosMigracionYBiblioteca(t *testing.T) {
	db := openTestDB(t)
	var schema string
	if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'photos'`).Scan(&schema); err != nil {
		t.Fatalf("tabla photos: %v", err)
	}
	if !strings.Contains(schema, "taken_at") {
		t.Fatalf("esquema de photos inesperado: %s", schema)
	}
	if kind, err := normalizeLibraryKind("photos"); err != nil || kind != "photos" {
		t.Fatalf("normalizeLibraryKind(photos) = %q, %v", kind, err)
	}

	res, err := db.Exec(`INSERT INTO libraries (path, name, kind) VALUES (?, ?, 'photos')`,
		t.TempDir(), "Fotos")
	if err != nil {
		t.Fatalf("crear biblioteca photos: %v", err)
	}
	libraryID, _ := res.LastInsertId()
	var path string
	if err := db.QueryRow(`SELECT path FROM libraries WHERE id = ?`, libraryID).Scan(&path); err != nil {
		t.Fatalf("leer biblioteca photos: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO library_paths (library_id, path) VALUES (?, ?)`, libraryID, path); err != nil {
		t.Fatalf("añadir carpeta photos: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO photos (library_id, path, folder, name, size, mtime, width, height)
		VALUES (?, ?, '', 'foto.jpg', 10, 20, 30, 40)`, libraryID, filepath.Join(path, "foto.jpg")); err != nil {
		t.Fatalf("insertar foto: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM libraries WHERE id = ?`, libraryID); err != nil {
		t.Fatalf("borrar biblioteca photos: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM photos`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("la cascada debe borrar las fotos: %d, %v", count, err)
	}
}

func TestPhotosEscaneoDimensionesEXIFCarpetasYPoda(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	rootB := t.TempDir()

	jpegPath := filepath.Join(root, "2024", "Captura.JPG")
	pngPath := filepath.Join(root, "inicio.png")
	webpPath := filepath.Join(root, "sin-dimensiones.webp")
	secondRootPath := filepath.Join(rootB, "otra.png")
	writePhotoTestJPEGWithEXIF(t, jpegPath, 12, 8, "2020:01:02 03:04:05")
	writePhotoTestPNG(t, pngPath, 10, 20)
	if err := os.WriteFile(webpPath, []byte("webp de prueba"), 0o644); err != nil {
		t.Fatalf("crear webp: %v", err)
	}
	writePhotoTestPNG(t, secondRootPath, 7, 9)
	writeFile(t, filepath.Join(root, "no-es-foto.txt"))

	res, err := db.Exec(`INSERT INTO libraries (path, name, kind) VALUES (?, ?, 'photos')`, root, "Fotos")
	if err != nil {
		t.Fatalf("crear biblioteca: %v", err)
	}
	libraryID, _ := res.LastInsertId()
	addScannerLibraryPath(t, db, libraryID, root)
	addScannerLibraryPath(t, db, libraryID, rootB)

	result, err := New(db).Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if result.Files != 4 || result.Upserted != 4 {
		t.Fatalf("resultado inesperado: %+v", result)
	}

	var folder, name string
	var width, height int
	var takenAt sql.NullInt64
	if err := db.QueryRow(`
		SELECT folder, name, width, height, taken_at FROM photos WHERE path = ?`, jpegPath).
		Scan(&folder, &name, &width, &height, &takenAt); err != nil {
		t.Fatalf("leer JPEG: %v", err)
	}
	if folder != "2024" || name != "Captura.JPG" || width != 12 || height != 8 {
		t.Fatalf("JPEG inesperado: folder=%q name=%q %dx%d", folder, name, width, height)
	}
	wantTaken := time.Date(2020, 1, 2, 3, 4, 5, 0, time.Local).Unix()
	if !takenAt.Valid || takenAt.Int64 != wantTaken {
		t.Fatalf("taken_at = %+v, se esperaba %d", takenAt, wantTaken)
	}
	if err := db.QueryRow(`SELECT width, height, taken_at FROM photos WHERE path = ?`, pngPath).
		Scan(&width, &height, &takenAt); err != nil {
		t.Fatalf("leer PNG: %v", err)
	}
	if width != 10 || height != 20 || takenAt.Valid {
		t.Fatalf("PNG inesperado: %dx%d taken_at=%+v", width, height, takenAt)
	}
	if err := db.QueryRow(`SELECT width, height FROM photos WHERE path = ?`, webpPath).
		Scan(&width, &height); err != nil {
		t.Fatalf("leer WebP: %v", err)
	}
	if width != 0 || height != 0 {
		t.Fatalf("WebP sin decodificador debe quedar en 0x0: %dx%d", width, height)
	}
	if err := db.QueryRow(`SELECT count(*) FROM photos WHERE path = ?`, filepath.Join(root, "no-es-foto.txt")).
		Scan(&width); err != nil || width != 0 {
		t.Fatalf("el texto no debe indexarse: %d, %v", width, err)
	}

	if err := os.Remove(jpegPath); err != nil {
		t.Fatalf("retirar JPEG: %v", err)
	}
	if _, err := New(db).Scan(ctx); err != nil {
		t.Fatalf("Scan tras retirar: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM photos WHERE path = ?`, jpegPath).Scan(&count); err != nil || count != 0 {
		t.Fatalf("la foto retirada debe podarse: %d, %v", count, err)
	}
	if count := countRows(t, &libraryTestServer{db: db}, `SELECT count(*) FROM photos`); count != 3 {
		t.Fatalf("fotos restantes = %d", count)
	}
}

func TestPhotosNoReinspeccionaSinCambios(t *testing.T) {
	db := openTestDB(t)
	root := t.TempDir()
	path := filepath.Join(root, "foto.png")
	writePhotoTestPNG(t, path, 3, 4)
	res, _ := db.Exec(`INSERT INTO libraries (path, name, kind) VALUES (?, ?, 'photos')`, root, "Fotos")
	libraryID, _ := res.LastInsertId()
	addScannerLibraryPath(t, db, libraryID, root)

	if _, err := New(db).Scan(context.Background()); err != nil {
		t.Fatalf("primer Scan: %v", err)
	}
	if _, err := db.Exec(`UPDATE photos SET width = 0, height = 0`); err != nil {
		t.Fatalf("preparar segundo escaneo: %v", err)
	}
	result, err := New(db).Scan(context.Background())
	if err != nil {
		t.Fatalf("segundo Scan: %v", err)
	}
	if result.Files != 1 || result.Upserted != 1 {
		t.Fatalf("resultado del segundo escaneo: %+v", result)
	}
	var width int
	if err := db.QueryRow(`SELECT width FROM photos WHERE path = ?`, path).Scan(&width); err != nil || width != 0 {
		t.Fatalf("un archivo sin cambios no debe redecodificarse: %d, %v", width, err)
	}
}

func writePhotoTestPNG(t *testing.T, path string, width, height int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("crear carpeta: %v", err)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 7), G: uint8(y * 11), B: 80, A: 255})
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("crear PNG: %v", err)
	}
	if err := png.Encode(file, img); err != nil {
		file.Close()
		t.Fatalf("codificar PNG: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("cerrar PNG: %v", err)
	}
	setPhotoTestModTime(t, path)
}

func writePhotoTestJPEGWithEXIF(t *testing.T, path string, width, height int, date string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("crear carpeta: %v", err)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 40, G: uint8(x * 9), B: uint8(y * 13), A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, img, nil); err != nil {
		t.Fatalf("codificar JPEG: %v", err)
	}
	data := encoded.Bytes()
	exif := buildPhotoTestEXIFJPEG(date)
	segment := exif[2 : len(exif)-2]
	data = append(append(append([]byte{}, data[:2]...), segment...), data[2:]...)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("crear JPEG con EXIF: %v", err)
	}
	setPhotoTestModTime(t, path)
}

func setPhotoTestModTime(t *testing.T, path string) {
	t.Helper()
	stamp := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatalf("fijar fecha de %s: %v", path, err)
	}
}
