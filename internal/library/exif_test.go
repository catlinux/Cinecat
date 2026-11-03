package library

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEXIFDateTimeOriginalEnSubIFD(t *testing.T) {
	got := exifDateTimeOriginal(buildPhotoTestEXIFJPEG("2020:01:02 03:04:05"))
	if got != "2020:01:02 03:04:05" {
		t.Fatalf("fecha EXIF = %q", got)
	}
	if got := exifDateTimeOriginal([]byte{0xff, 0xd8, 0xff, 0xd9}); got != "" {
		t.Fatalf("un JPEG sin EXIF no debe devolver fecha: %q", got)
	}
}

func TestPhotoTakenAtUsaHoraLocal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "foto.jpg")
	if err := os.WriteFile(path, buildPhotoTestEXIFJPEG("2020:01:02 03:04:05"), 0o644); err != nil {
		t.Fatalf("crear JPEG: %v", err)
	}
	got := photoTakenAt(path)
	want := time.Date(2020, 1, 2, 3, 4, 5, 0, time.Local).Unix()
	if got == nil || *got != want {
		t.Fatalf("taken_at = %v, se esperaba %d", got, want)
	}

	noEXIF := filepath.Join(dir, "sin-exif.jpg")
	if err := os.WriteFile(noEXIF, []byte{0xff, 0xd8, 0xff, 0xd9}, 0o644); err != nil {
		t.Fatalf("crear JPEG sin EXIF: %v", err)
	}
	if got := photoTakenAt(noEXIF); got != nil {
		t.Fatalf("sin EXIF debe ser nil: %v", *got)
	}
	if got := photoTakenAt(filepath.Join(dir, "foto.png")); got != nil {
		t.Fatalf("PNG no debe llevar EXIF JPEG: %v", *got)
	}
}

// buildPhotoTestEXIFJPEG construye un JPEG mínimo con un APP1/TIFF little-endian y
// DateTimeOriginal en el SubIFD. No necesita ser una imagen completa porque el
// lector de EXIF solo recorre los segmentos.
func buildPhotoTestEXIFJPEG(value string) []byte {
	if len(value) != 19 {
		panic("la fecha EXIF debe tener 19 caracteres")
	}
	tiff := make([]byte, 64)
	copy(tiff, "II")
	binary.LittleEndian.PutUint16(tiff[2:4], 42)
	binary.LittleEndian.PutUint32(tiff[4:8], 8)

	binary.LittleEndian.PutUint16(tiff[8:10], 1)
	binary.LittleEndian.PutUint16(tiff[10:12], 0x8769)
	binary.LittleEndian.PutUint16(tiff[12:14], 4)
	binary.LittleEndian.PutUint32(tiff[14:18], 1)
	binary.LittleEndian.PutUint32(tiff[18:22], 26)
	binary.LittleEndian.PutUint32(tiff[22:26], 0)

	binary.LittleEndian.PutUint16(tiff[26:28], 1)
	binary.LittleEndian.PutUint16(tiff[28:30], 0x9003)
	binary.LittleEndian.PutUint16(tiff[30:32], 2)
	binary.LittleEndian.PutUint32(tiff[32:36], 20)
	binary.LittleEndian.PutUint32(tiff[36:40], 44)
	binary.LittleEndian.PutUint32(tiff[40:44], 0)
	copy(tiff[44:], value+"\x00")

	payload := append([]byte("Exif\x00\x00"), tiff...)
	result := []byte{0xff, 0xd8, 0xff, 0xe1, 0, 0}
	binary.BigEndian.PutUint16(result[4:6], uint16(len(payload)+2))
	result = append(result, payload...)
	return append(result, 0xff, 0xd9)
}
