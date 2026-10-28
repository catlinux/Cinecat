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
	// compatibilidad con los clientes; según el códec y el índice de claves se
	// copia el vídeo o se recodifica a H.264 + AAC estéreo.
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

// Capabilities resume los códecs que declara el cliente. HEVC y HEVC10
// corresponden a la reproducción directa; HEVCMSE y HEVC10MSE indican que el
// navegador puede recibir HEVC por MSE, que es lo que exige el HLS en MPEG-TS.
// HEV1 indica aceptación de la etiqueta hev1 y es independiente de las demás.
type Capabilities struct {
	HEVC      bool
	HEVC10    bool
	HEVCMSE   bool
	HEVC10MSE bool
	// HEV1 indica que el navegador acepta la etiqueta hev1 en MP4. No
	// implica HEVC ni al revés.
	HEV1 bool
	// native guarda los tokens que solo envían las apps (ver native.go).
	native nativeSet
}

// ParseCapabilities interpreta la lista separada por comas que envía la web.
// Los tokens desconocidos se ignoran; hevc10 implica hevc y hevc10mse implica
// hevcmse. hev1 no implica hevc ni al revés.
func ParseCapabilities(raw string) Capabilities {
	var result Capabilities
	for _, token := range strings.Split(raw, ",") {
		switch strings.ToLower(strings.TrimSpace(token)) {
		case "hevc":
			result.HEVC = true
		case "hevc10":
			result.HEVC = true
			result.HEVC10 = true
		case "hevcmse":
			result.HEVCMSE = true
		case "hevc10mse":
			result.HEVCMSE = true
			result.HEVC10MSE = true
		case "hev1":
			result.HEV1 = true
		default:
			if bit, ok := nativeToken(strings.ToLower(strings.TrimSpace(token))); ok {
				result.native |= bit
			}
		}
	}
	return result
}

// QueryValue devuelve la lista canónica de capacidades para las URLs firmadas.
func (c Capabilities) QueryValue() string {
	values := make([]string, 0, 4)
	if c.HEVC || c.HEVC10 {
		values = append(values, "hevc")
	}
	if c.HEVC10 {
		values = append(values, "hevc10")
	}
	if c.HEVCMSE || c.HEVC10MSE {
		values = append(values, "hevcmse")
	}
	if c.HEVC10MSE {
		values = append(values, "hevc10mse")
	}
	if c.HEV1 {
		values = append(values, "hev1")
	}
	for _, token := range nativeTokens {
		if c.native.has(token.bit) {
			values = append(values, token.name)
		}
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
		if !directHEVCTag(video, caps) {
			return false
		}
		return !hevc10Bit(video) || caps.HEVC10
	default:
		return false
	}
}

func directHEVCTag(video VideoStream, caps Capabilities) bool {
	switch strings.ToLower(strings.TrimSpace(video.CodecTag)) {
	case "hvc1":
		return true
	case "hev1":
		return caps.HEV1
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

// CanCopy indica si el vídeo se puede reempaquetar a HLS sin recodificarlo.
// No decide si conviene hacerlo ahora: el índice de fotogramas clave y el
// estado del archivo se comprueban en el servicio HLS.
func CanCopy(info *FileInfo, caps Capabilities, maxHeight int) bool {
	if info == nil || copyContainerBlocked(info) {
		return false
	}
	if len(info.Video) == 0 {
		return false
	}
	video := info.Video[0]
	if maxHeight > 0 && (video.Height <= 0 || video.Height > maxHeight) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(video.Codec)) {
	case "h264":
		return copyableH264(video)
	case "hevc":
		// Una app nativa (ExoPlayer) decodifica HEVC en MPEG-TS sin MSE.
		if hevc10Bit(video) {
			return caps.HEVC10MSE || (caps.Native() && caps.HEVC10)
		}
		return caps.HEVCMSE || caps.HEVC10MSE || (caps.Native() && (caps.HEVC || caps.HEVC10))
	default:
		return false
	}
}

// copyContainerBlocked descarta los contenedores que el muxer segment no
// puede copiar de forma fiable. El formato de ffprobe manda, pero la extensión
// también cuenta porque FileInfo puede venir de una caché antigua o vacía.
func copyContainerBlocked(info *FileInfo) bool {
	if copyExtensionBlocked(info.FileName) {
		return true
	}
	// Si la ficha no trae nombre (por ejemplo, una caché antigua), el formato
	// de ffprobe sigue bastando para descartar AVI y MPEG-TS.
	for _, name := range strings.Split(strings.ToLower(info.Container), ",") {
		switch strings.TrimSpace(name) {
		case "avi", "mpegts", "m2ts", "ts":
			return true
		}
	}
	return false
}

func copyExtensionBlocked(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".avi", ".ts", ".m2ts":
		return true
	default:
		return false
	}
}

func copyableH264(video VideoStream) bool {
	if video.BitDepth > 8 {
		return false
	}
	profile := strings.ToLower(video.Profile)
	if strings.Contains(profile, "high 10") ||
		strings.Contains(profile, "high 4:2:2") ||
		strings.Contains(profile, "high 4:4:4") {
		return false
	}
	pixel := strings.ToLower(strings.TrimSpace(video.PixelFormat))
	return pixel == "yuv420p" || pixel == "yuvj420p"
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
		// Algunos navegadores (Opera) no sacan audio de AAC multicanal en
		// directo; el HLS lo baja a estéreo.
		return audio.Channels <= 2
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
