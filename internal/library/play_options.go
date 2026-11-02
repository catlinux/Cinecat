package library

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/catlinux/cinecat/internal/playback"
)

type audioTrackDTO struct {
	Index         int    `json:"index"`
	Language      string `json:"language"`
	LanguageName  string `json:"language_name"`
	Codec         string `json:"codec"`
	Channels      int    `json:"channels"`
	ChannelLayout string `json:"channel_layout"`
	Title         string `json:"title"`
	Default       bool   `json:"default"`
}

type qualityDTO struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type subtitleDTO struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	Language     string `json:"language"`
	LanguageName string `json:"language_name"`
	Title        string `json:"title"`
	Codec        string `json:"codec"`
	Forced       bool   `json:"forced"`
	Default      bool   `json:"default"`
	BurnIn       bool   `json:"burn_in"`
	URL          string `json:"url"`
}

type playOptions struct {
	AudioTracks      []audioTrackDTO
	SelectedAudio    int
	Qualities        []qualityDTO
	SelectedQuality  string
	Subtitles        []subtitleDTO
	SelectedSubtitle *string
	Audio            int
	Quality          string
	MaxHeight        int
}

type playOptionsError struct {
	status  int
	message string
}

func validQualityValue(value string) bool {
	switch value {
	case "auto", "original", "480", "720", "1080":
		return true
	default:
		return false
	}
}

func languageName(value string) string {
	code := strings.ToLower(strings.TrimSpace(value))
	switch code {
	case "es", "spa":
		return "Español"
	case "en", "eng":
		return "Inglés"
	case "fr", "fra", "fre":
		return "Francés"
	case "de", "deu", "ger":
		return "Alemán"
	case "it", "ita":
		return "Italiano"
	case "pt", "por":
		return "Portugués"
	case "ca", "cat":
		return "Catalán"
	case "ja", "jpn":
		return "Japonés"
	case "ko", "kor":
		return "Coreano"
	case "zh", "zho", "chi":
		return "Chino"
	case "ru", "rus":
		return "Ruso"
	default:
		return "Desconocido"
	}
}

type subtitleSelection struct {
	Tracks       []playback.SubtitleTrack
	Selected     *string
	BurnSubtitle int
}

// selectSubtitles reúne las pistas disponibles y resuelve la elegida por la
// consulta o las preferencias. BurnSubtitle guarda la posición más uno de una
// pista de imagen; cero significa que no se quema ninguna.
func (a *API) selectSubtitles(
	ctx context.Context,
	itemID int64,
	path string,
	info *playback.FileInfo,
	preferences *itemPreferencesDTO,
	query map[string][]string,
) (subtitleSelection, *playOptionsError) {
	tracks := playback.DiscoverSubtitles(path, info)
	uploaded, err := a.uploadedSubtitleTracks(ctx, itemID)
	if err != nil {
		return subtitleSelection{}, &playOptionsError{http.StatusInternalServerError, "error interno"}
	}
	tracks = append(tracks, uploaded...)
	result := subtitleSelection{Tracks: tracks}
	findTrack := func(id string) *playback.SubtitleTrack {
		for index := range tracks {
			if tracks[index].ID == id {
				return &tracks[index]
			}
		}
		return nil
	}
	selectID := func(id string) *playOptionsError {
		if !playback.ValidSubtitleID(id) {
			return &playOptionsError{http.StatusBadRequest, "subtítulo no válido"}
		}
		track := findTrack(id)
		if track == nil {
			return &playOptionsError{http.StatusBadRequest, "subtítulo no válido"}
		}
		value := id
		result.Selected = &value
		if track.Image {
			result.BurnSubtitle = track.StreamIndex + 1
		}
		return nil
	}

	if values, ok := query["subtitle"]; ok {
		value := ""
		if len(values) > 0 {
			value = values[0]
		}
		if value != "" {
			if err := selectID(value); err != nil {
				return result, err
			}
		}
	} else if preferences != nil && preferences.SubtitleID != nil && *preferences.SubtitleID != "" {
		value := *preferences.SubtitleID
		if playback.ValidSubtitleID(value) && findTrack(value) != nil {
			if err := selectID(value); err != nil {
				return result, err
			}
		}
	}
	return result, nil
}

func (a *API) maxPlaybackHeight() int {
	if a.settingsStore != nil {
		switch value := a.settingsStore.Int("playback.max_height"); value {
		case 480, 720, 1080:
			return value
		}
	}
	return 1080
}

func (a *API) playOptions(
	itemID int64,
	mode playback.Mode,
	info *playback.FileInfo,
	preferences *itemPreferencesDTO,
	query map[string][]string,
	subtitles subtitleSelection,
	exp, sig string,
) (playOptions, *playOptionsError) {
	result := playOptions{
		AudioTracks: make([]audioTrackDTO, 0),
		Qualities:   make([]qualityDTO, 0),
		Subtitles:   make([]subtitleDTO, 0),
	}
	audioDefault := 0
	audioDefaultSet := false
	if info != nil {
		for position, track := range info.Audio {
			if track.Default && !audioDefaultSet {
				audioDefault = position
				audioDefaultSet = true
			}
			result.AudioTracks = append(result.AudioTracks, audioTrackDTO{
				Index:         position,
				Language:      track.Language,
				LanguageName:  languageName(track.Language),
				Codec:         track.Codec,
				Channels:      track.Channels,
				ChannelLayout: track.ChannelLayout,
				Title:         track.Title,
				Default:       track.Default,
			})
		}
	}

	audioSet := false
	rawAudio := ""
	if values, ok := query["audio"]; ok {
		audioSet = true
		if len(values) > 0 {
			rawAudio = values[0]
		}
	}
	if audioSet {
		index, err := strconv.Atoi(rawAudio)
		if err != nil || index < 0 {
			return result, &playOptionsError{http.StatusBadRequest, "audio no válido"}
		}
		if info == nil || len(result.AudioTracks) == 0 {
			if index != 0 {
				return result, &playOptionsError{http.StatusBadRequest, "audio no válido"}
			}
		} else if index >= len(result.AudioTracks) {
			return result, &playOptionsError{http.StatusBadRequest, "audio no válido"}
		}
		result.SelectedAudio = index
	} else if preferences != nil && preferences.AudioIndex != nil &&
		*preferences.AudioIndex >= 0 && len(result.AudioTracks) > 0 &&
		*preferences.AudioIndex < len(result.AudioTracks) {
		result.SelectedAudio = *preferences.AudioIndex
	} else {
		result.SelectedAudio = audioDefault
	}
	result.Audio = result.SelectedAudio

	maxHeight := a.maxPlaybackHeight()
	originalHeight := 0
	if info != nil && len(info.Video) > 0 {
		originalHeight = info.Video[0].Height
	}
	result.Qualities = buildQualities(mode, maxHeight, originalHeight)
	requestedQuality := "auto"
	if mode == playback.ModeDirect {
		requestedQuality = "original"
	}
	if values, ok := query["quality"]; ok {
		value := ""
		if len(values) > 0 {
			value = values[0]
		}
		if !validQualityValue(value) {
			return result, &playOptionsError{http.StatusBadRequest, "calidad no válida"}
		}
		requestedQuality = value
	} else if preferences != nil && preferences.Quality != nil &&
		validQualityValue(*preferences.Quality) {
		requestedQuality = *preferences.Quality
	}
	result.SelectedQuality = selectQuality(mode, requestedQuality, result.Qualities)
	result.Quality = result.SelectedQuality
	result.MaxHeight = effectiveHeight(result.SelectedQuality, maxHeight, originalHeight)

	for _, track := range subtitles.Tracks {
		url := subtitleURL(itemID, track.ID, exp, sig)
		if track.Image {
			url = ""
		}
		result.Subtitles = append(result.Subtitles, subtitleDTO{
			ID:           track.ID,
			Source:       track.Source,
			Language:     track.Language,
			LanguageName: languageName(track.Language),
			Title:        track.Title,
			Codec:        track.Codec,
			Forced:       track.Forced,
			Default:      track.Default,
			BurnIn:       track.Image,
			URL:          url,
		})
	}
	result.SelectedSubtitle = subtitles.Selected

	return result, nil
}

func buildQualities(mode playback.Mode, maxHeight, originalHeight int) []qualityDTO {
	if mode == playback.ModeDirect {
		return []qualityDTO{{Value: "original", Label: "Original"}}
	}
	limit := maxHeight
	if originalHeight > 0 && originalHeight < limit {
		limit = originalHeight
	}
	result := make([]qualityDTO, 0, 5)
	result = append(result, qualityDTO{Value: "auto", Label: fmt.Sprintf("Auto (hasta %dp)", limit)})
	for _, height := range []int{1080, 720, 480} {
		if height <= maxHeight && (originalHeight == 0 || height <= originalHeight) {
			result = append(result, qualityDTO{
				Value: strconv.Itoa(height),
				Label: fmt.Sprintf("%dp", height),
			})
		}
	}
	result = append(result, qualityDTO{Value: "original", Label: "Original"})
	return result
}

func selectQuality(mode playback.Mode, requested string, qualities []qualityDTO) string {
	if mode == playback.ModeDirect {
		return "original"
	}
	for _, quality := range qualities {
		if quality.Value == requested {
			return requested
		}
	}
	return "auto"
}

func effectiveHeight(quality string, maxHeight, originalHeight int) int {
	limit := maxHeight
	if originalHeight > 0 && originalHeight < limit {
		limit = originalHeight
	}
	switch quality {
	case "480", "720", "1080":
		height, err := strconv.Atoi(quality)
		if err == nil && height < limit {
			return height
		}
		return limit
	default:
		return limit
	}
}

func (a *API) streamSelection(
	mode playback.Mode,
	info *playback.FileInfo,
	query map[string][]string,
) (playback.StreamSelection, *playOptionsError) {
	result := playback.StreamSelection{
		Quality: "auto",
		Caps:    playback.ParseCapabilities(firstQueryValue(query, "caps")).QueryValue(),
		Info:    info,
	}
	if mode == playback.ModeDirect {
		result.Quality = "original"
	}
	if values, ok := query["q"]; ok {
		value := ""
		if len(values) > 0 {
			value = values[0]
		}
		if !validQualityValue(value) {
			return result, &playOptionsError{http.StatusBadRequest, "calidad no válida"}
		}
		result.Quality = value
	}

	if values, ok := query["audio"]; ok {
		raw := ""
		if len(values) > 0 {
			raw = values[0]
		}
		index, err := strconv.Atoi(raw)
		if err != nil || index < 0 {
			return result, &playOptionsError{http.StatusBadRequest, "audio no válido"}
		}
		if info == nil || len(info.Audio) == 0 {
			if index != 0 {
				return result, &playOptionsError{http.StatusBadRequest, "audio no válido"}
			}
		} else if index >= len(info.Audio) {
			return result, &playOptionsError{http.StatusBadRequest, "audio no válido"}
		}
		result.Audio = index
	}

	originalHeight := 0
	if info != nil {
		if len(info.Video) > 0 {
			originalHeight = info.Video[0].Height
		}
		if info.DurationSeconds > 0 {
			result.Duration = time.Duration(info.DurationSeconds * float64(time.Second))
		}
	}
	result.MaxHeight = effectiveHeight(result.Quality, a.maxPlaybackHeight(), originalHeight)

	if values, ok := query["burnsub"]; ok {
		raw := ""
		if len(values) > 0 {
			raw = values[0]
		}
		index, err := strconv.Atoi(raw)
		if err != nil || index < 0 || info == nil || index >= len(info.Subtitles) ||
			!playback.ImageSubtitleCodec(info.Subtitles[index].Codec) {
			return result, &playOptionsError{http.StatusBadRequest, "subtítulo no válido"}
		}
		result.BurnSubtitle = index + 1
	}

	// La playlist decide una sola vez si copia o recodifica. El parámetro m de
	// las URLs firmadas mantiene esa decisión en todos los segmentos aunque el
	// índice de claves pase de pendiente a listo entre peticiones.
	switch firstQueryValue(query, "m") {
	case "copy":
		if mode == playback.ModeRemux {
			result.Copy = true
			result.ForceCopy = true
		}
	case "transcode":
		if mode == playback.ModeRemux {
			result.ForceTranscode = true
		}
	default:
		if mode == playback.ModeRemux {
			caps := playback.ParseCapabilities(result.Caps)
			result.Copy = playback.CanCopy(info, caps, result.MaxHeight)
		}
	}
	if result.BurnSubtitle > 0 {
		result.Copy = false
		result.ForceCopy = false
		result.ForceTranscode = true
	}
	return result, nil
}

func firstQueryValue(query map[string][]string, key string) string {
	if values, ok := query[key]; ok && len(values) > 0 {
		return values[0]
	}
	return ""
}

func subtitleURL(itemID int64, subtitleID, exp, sig string) string {
	return "/api/v1/stream/" + strconv.FormatInt(itemID, 10) +
		"/subtitles/" + subtitleID + ".vtt?exp=" + exp + "&sig=" + sig
}
