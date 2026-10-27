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
	// ModeRemux sirve el vídeo por HLS. El nombre se conserva por
	// compatibilidad con los clientes; hoy el servidor siempre recodifica a
	// H.264 + AAC estéreo (ver hlsTranscodeArgs).
	ModeRemux Mode = "remux"
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

// Capabilities resume los códecs que declara el cliente.
type Capabilities struct {
	HEVC   bool
	HEVC10 bool
}

// ParseCapabilities interpreta la lista separada por comas que envía la web.
// Los tokens desconocidos se ignoran y hevc10 implica hevc.
func ParseCapabilities(raw string) Capabilities {
	var result Capabilities
	for _, token := range strings.Split(raw, ",") {
		switch strings.ToLower(strings.TrimSpace(token)) {
		case "hevc":
			result.HEVC = true
		case "hevc10":
			result.HEVC = true
			result.HEVC10 = true
		}
	}
	return result
}

// QueryValue devuelve la lista canónica de capacidades para las URLs firmadas.
func (c Capabilities) QueryValue() string {
	values := make([]string, 0, 2)
	if c.HEVC || c.HEVC10 {
		values = append(values, "hevc")
	}
	if c.HEVC10 {
		values = append(values, "hevc10")
	}
	return strings.Join(values, ",")
}

// Decide mantiene la decisión por extensión para los llamadores que no tienen
// la ficha de ffprobe. Se conserva para compatibilidad.
func Decide(path string) (Mode, error) {
	return DecideFor(path, nil, Capabilities{})
}

// DecideFor elige el modo con la ficha real del archivo y las capacidades del
// cliente. Si no hay ficha, se mantiene la decisión por extensión.
func DecideFor(path string, info *FileInfo, caps Capabilities) (Mode, error) {
	extension := strings.ToLower(filepath.Ext(path))
	switch extension {
	case ".mp4", ".m4v":
		if info == nil {
			return ModeDirect, nil
		}
		if !directVideo(info, caps) || !directAudio(info) {
			return ModeRemux, nil
		}
		return ModeDirect, nil
	case ".mkv", ".avi", ".mov", ".ts", ".m2ts":
		return ModeRemux, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrUnsupportedContainer, extension)
	}
}

func directVideo(info *FileInfo, caps Capabilities) bool {
	if info == nil || len(info.Video) == 0 {
		return false
	}
	video := info.Video[0]
	codec := strings.ToLower(strings.TrimSpace(video.Codec))
	switch codec {
	case "h264":
		profile := strings.ToLower(video.Profile)
		if strings.Contains(profile, "high 10") ||
			strings.Contains(profile, "high 4:2:2") ||
			strings.Contains(profile, "high 4:4:4") {
			return false
		}
		pixel := strings.ToLower(strings.TrimSpace(video.PixelFormat))
		return pixel == "yuv420p" || pixel == "yuvj420p"
	case "hevc":
		if !caps.HEVC && !caps.HEVC10 {
			return false
		}
		return !hevc10Bit(video) || caps.HEVC10
	default:
		return false
	}
}

func hevc10Bit(video VideoStream) bool {
	if video.BitDepth > 8 {
		return true
	}
	profile := strings.ToLower(video.Profile)
	pixel := strings.ToLower(video.PixelFormat)
	return strings.Contains(profile, "main 10") ||
		strings.Contains(profile, "main10") ||
		strings.Contains(pixel, "10") ||
		strings.Contains(pixel, "p010")
}

func directAudio(info *FileInfo) bool {
	if info == nil || len(info.Audio) == 0 {
		return true
	}
	audio := info.Audio[0]
	for _, track := range info.Audio {
		if track.Default {
			audio = track
			break
		}
	}
	switch strings.ToLower(strings.TrimSpace(audio.Codec)) {
	case "aac", "mp3":
		return true
	default:
		return false
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

// ValidHLSName indica si el nombre corresponde a un archivo HLS que se sirve:
// la playlist o un segmento MPEG-TS.
func ValidHLSName(name string) bool {
	if name != filepath.Base(name) || strings.ContainsAny(name, `/\`) {
		return false
	}
	if name == "playlist.m3u8" {
		return true
	}
	_, ok := segmentIndex(name)
	return ok
}
