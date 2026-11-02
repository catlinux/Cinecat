package library

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/catlinux/cinecat/internal/auth"
	"github.com/catlinux/cinecat/internal/playback"
)

// deviceProfile son los valores por defecto de la sesión que hace la petición.
// Los campos vacíos no imponen nada y dejan actuar al comportamiento actual.
type deviceProfile struct {
	MaxQuality       string
	AudioLanguage    string
	SubtitleLanguage string
}

func (a *API) loadDeviceProfile(ctx context.Context, r *http.Request) (deviceProfile, error) {
	user, ok := sessionUserFromRequest(r)
	if !ok || user.SessionID == 0 || a.auth == nil {
		return deviceProfile{}, nil
	}
	profile, err := a.auth.SessionProfileFor(ctx, user.ID, user.SessionID)
	if errors.Is(err, auth.ErrSessionNotFound) {
		return deviceProfile{}, nil
	}
	if err != nil {
		return deviceProfile{}, err
	}
	return deviceProfile{
		MaxQuality:       profile.MaxQuality,
		AudioLanguage:    profile.AudioLanguage,
		SubtitleLanguage: profile.SubtitleLanguage,
	}, nil
}

func (p deviceProfile) qualityHeight() int {
	switch p.MaxQuality {
	case "480":
		return 480
	case "720":
		return 720
	case "1080":
		return 1080
	default:
		return 0
	}
}

// subtitleID busca una pista de texto del idioma pedido, prefiriendo las no
// forzadas y sin elegir nunca una pista de imagen.
func (p deviceProfile) subtitleID(tracks []playback.SubtitleTrack) string {
	if p.SubtitleLanguage == "" || p.SubtitleLanguage == "off" {
		return ""
	}
	for _, forced := range []bool{false, true} {
		for _, track := range tracks {
			if track.Image || track.Forced != forced {
				continue
			}
			if sameLanguage(track.Language, p.SubtitleLanguage) {
				return track.ID
			}
		}
	}
	return ""
}

func (p deviceProfile) audioIndex(tracks []playback.AudioStream) int {
	if p.AudioLanguage == "" {
		return -1
	}
	for index, track := range tracks {
		if sameLanguage(track.Language, p.AudioLanguage) {
			return index
		}
	}
	return -1
}

// sameLanguage compara códigos de dos y tres letras usando la misma tabla que
// languageName. Para códigos desconocidos compara el texto sin distinguir
// mayúsculas.
func sameLanguage(left, right string) bool {
	leftName := languageName(left)
	rightName := languageName(right)
	if leftName != "Desconocido" && rightName != "Desconocido" {
		return leftName == rightName
	}
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}
