// Package playback decide cómo servir un vídeo y prepara el remux a HLS fMP4
// con FFmpeg, además de firmar las URLs de reproducción.
package playback

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// Mode es la forma de reproducción elegida.
type Mode string

const (
	// ModeDirect sirve el archivo tal cual, con soporte de rangos HTTP.
	ModeDirect Mode = "direct"
	// ModeRemux reempaqueta a HLS fMP4 sin recodificar.
	ModeRemux Mode = "remux"
	// ModeTranscode reempaqueta a HLS fMP4 recodificando solo el vídeo cuando
	// los keyframes del original no permiten segmentar sin tocar el códec.
	ModeTranscode Mode = "transcode"
)

// Errores del paquete.
var (
	ErrUnsupportedContainer = errors.New("playback: contenedor no soportado")
	ErrOutsideLibrary       = errors.New("playback: la ruta queda fuera de la biblioteca")
	ErrFileNotFound         = errors.New("playback: archivo no encontrado")
	ErrFFmpegNotFound       = errors.New("playback: ffmpeg no está instalado o no está en PATH")
	ErrFFmpegFailed         = errors.New("playback: falló el remux con ffmpeg")
	ErrInvalidSignature     = errors.New("playback: firma de reproducción no válida")
	ErrExpired              = errors.New("playback: URL de reproducción caducada")
)

// Decide elige el modo según la extensión del archivo. Para este paso se
// consideran servibles directamente los contenedores MP4; el resto de formatos
// de vídeo habituales se reempaquetan a HLS. Cualquier otra extensión es un
// error claro: la transcodificación queda para la Fase 3.
func Decide(path string) (Mode, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4", ".m4v":
		return ModeDirect, nil
	case ".mkv", ".avi", ".mov", ".ts", ".m2ts":
		return ModeRemux, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrUnsupportedContainer, filepath.Ext(path))
	}
}

// SafePath limpia la ruta del archivo y comprueba que sigue dentro de la
// biblioteca, como defensa adicional frente a path traversal.
func SafePath(libraryRoot, mediaPath string) (string, error) {
	root, err := filepath.Abs(libraryRoot)
	if err != nil {
		return "", fmt.Errorf("playback: ruta de biblioteca no válida: %w", err)
	}
	clean, err := filepath.Abs(filepath.Clean(mediaPath))
	if err != nil {
		return "", fmt.Errorf("playback: ruta de archivo no válida: %w", err)
	}

	rel, err := filepath.Rel(root, clean)
	if err != nil || filepath.IsAbs(rel) || rel == "." || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrOutsideLibrary
	}
	return clean, nil
}

// ValidHLSName indica si el nombre corresponde a un archivo que genera el
// remux: la playlist, el init de fMP4 o un segmento.
func ValidHLSName(name string) bool {
	if name != filepath.Base(name) || strings.ContainsAny(name, `/\`) {
		return false
	}
	switch {
	case name == "playlist.m3u8", name == "init.mp4":
		return true
	case strings.HasPrefix(name, "segment_") && strings.HasSuffix(name, ".m4s"):
		return true
	default:
		return false
	}
}
