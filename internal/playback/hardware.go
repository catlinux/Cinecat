package playback

import (
	"context"
	"log"
	"strings"
	"time"
)

// hardwareTimeout acota la comprobación inicial de VA-API. Es una prueba de un
// segundo de vídeo sintético, no una decodificación de la película.
const hardwareTimeout = 15 * time.Second

const defaultVAAPIDevice = "/dev/dri/renderD128"

// hardwareFullRoute indica si la ficha permite decodificar, escalar y
// codificar todo en la gráfica. Sin ficha, o con un códec que la gráfica no
// decodifica, se usa la ruta CPU+GPU.
func hardwareFullRoute(info *FileInfo, maxHeight int) bool {
	if info == nil || len(info.Video) == 0 || maxHeight <= 0 {
		return false
	}
	video := info.Video[0]
	if video.Height <= 0 || video.Height > maxHeight {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(video.FieldOrder)) {
	case "", "progressive", "unknown":
	default:
		return false
	}
	switch strings.ToLower(strings.TrimSpace(video.Codec)) {
	case "h264":
		return copyableH264(video)
	case "hevc":
		profile := strings.ToLower(strings.TrimSpace(video.Profile))
		return (profile == "" || profile == "main") && !hevc10Bit(video)
	default:
		return false
	}
}

// HardwareAvailable comprueba, una sola vez por proceso, si VA-API está
// disponible. Respeta el ajuste en caliente: con «off» nunca se prueba.
func (s *Service) HardwareAvailable(ctx context.Context) bool {
	if s == nil || !s.currentTranscodeSettings().Hardware {
		return false
	}
	return s.hardwareReady()
}

// HardwareAvailableFor es como HardwareAvailable, pero respeta además la
// marca por archivo que deja un fallo de hardware.
func (s *Service) HardwareAvailableFor(ctx context.Context, path string) bool {
	if s == nil || s.hardwareDisabledLocked(path) {
		return false
	}
	return s.HardwareAvailable(ctx)
}

func (s *Service) hardwareReady() bool {
	s.hardwareOnce.Do(func() {
		device := hardwareDevicePath()
		if device == "" {
			log.Printf("playback: aceleración por hardware no disponible: no hay dispositivo VA-API")
			return
		}
		if !hardwareDeviceExists(device) {
			log.Printf("playback: aceleración por hardware no disponible: no existe %s", device)
			return
		}
		ctx, cancel := context.WithTimeout(s.procCtx, hardwareTimeout)
		defer cancel()
		if err := probeHardware(ctx, s.command, device); err != nil {
			log.Printf("playback: aceleración por hardware no disponible en %s: %v", device, err)
			return
		}
		s.hardwareAvailable = true
		log.Printf("playback: aceleración por hardware activada con VA-API en %s", device)
	})
	return s.hardwareAvailable
}

func (s *Service) markHardwareDisabled(path string) {
	key := copyFileKey(path)
	s.hardwareMu.Lock()
	if s.hardwareDisabled == nil {
		s.hardwareDisabled = make(map[string]bool)
	}
	s.hardwareDisabled[key] = true
	s.hardwareMu.Unlock()
}

func (s *Service) hardwareDisabledLocked(path string) bool {
	s.hardwareMu.Lock()
	disabled := s.hardwareDisabled[copyFileKey(path)]
	s.hardwareMu.Unlock()
	return disabled
}
