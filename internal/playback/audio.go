package playback

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// AudioTags son las etiquetas de un archivo de audio normalizadas.
type AudioTags struct {
	Title       string `json:"title,omitempty"`
	Artist      string `json:"artist,omitempty"`
	AlbumArtist string `json:"album_artist,omitempty"`
	Album       string `json:"album,omitempty"`
	Genre       string `json:"genre,omitempty"`
	Track       int    `json:"track,omitempty"`
	Disc        int    `json:"disc,omitempty"`
	Year        int    `json:"year,omitempty"`
}

// AudioInfo resume lo que necesita el escáner de música de un archivo.
type AudioInfo struct {
	Tags             AudioTags
	DurationSeconds  float64
	Codec            string
	BitRate          int64
	SampleRate       int
	Channels         int
	HasEmbeddedCover bool
}

// InspectAudio lee etiquetas y datos técnicos del primer flujo de audio.
func (p *Prober) InspectAudio(ctx context.Context, path string) (*AudioInfo, error) {
	output, err := p.runFFprobeOutput(ctx, path, "json", "-show_format", "-show_streams")
	if err != nil {
		return nil, err
	}
	info, err := parseFFprobeAudioJSON([]byte(output))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProbeFailed, err)
	}
	return info, nil
}

// InspectAudio delega en el analizador del servicio.
func (s *Service) InspectAudio(ctx context.Context, path string) (*AudioInfo, error) {
	if s == nil || s.prober == nil {
		return nil, ErrFFprobeNotFound
	}
	return s.prober.InspectAudio(ctx, path)
}

func parseFFprobeAudioJSON(data []byte) (*AudioInfo, error) {
	var raw ffprobeOutput
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("playback: JSON de audio no válido: %w", err)
	}
	info := &AudioInfo{
		DurationSeconds: float64(raw.Format.Duration),
		BitRate:         int64(raw.Format.BitRate),
		Tags:            audioTagsFromTags(raw.Format.Tags),
	}
	for _, stream := range raw.Streams {
		if stream.Disposition.AttachedPic != 0 {
			info.HasEmbeddedCover = true
		}
		if info.Codec == "" && strings.EqualFold(cleanFFString(stream.CodecType), "audio") {
			info.Codec = cleanFFString(stream.CodecName)
			if stream.BitRate != 0 {
				info.BitRate = int64(stream.BitRate)
			}
			info.SampleRate = int(stream.SampleRate)
			info.Channels = int(stream.Channels)
		}
	}
	return info, nil
}

func audioTagsFromTags(tags map[string]string) AudioTags {
	return AudioTags{
		Title:       tagValueCI(tags, "title"),
		Artist:      tagValueCI(tags, "artist"),
		AlbumArtist: tagValueCI(tags, "album_artist", "albumartist", "album artist"),
		Album:       tagValueCI(tags, "album"),
		Genre:       tagValueCI(tags, "genre"),
		Track:       parseTagNumber(tagValueCI(tags, "track", "tracknumber")),
		Disc:        parseTagNumber(tagValueCI(tags, "disc", "discnumber", "disc_no")),
		Year:        parseYearTag(tagValueCI(tags, "date", "year")),
	}
}

func tagValueCI(tags map[string]string, keys ...string) string {
	for key, value := range tags {
		for _, wanted := range keys {
			if strings.EqualFold(key, wanted) {
				return cleanFFString(value)
			}
		}
	}
	return ""
}

func parseTagNumber(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if index := strings.IndexByte(value, '/'); index >= 0 {
		value = value[:index]
	}
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number <= 0 {
		return 0
	}
	return number
}

func parseYearTag(value string) int {
	value = strings.TrimSpace(value)
	for index := 0; index+4 <= len(value); index++ {
		candidate := value[index : index+4]
		year, err := strconv.Atoi(candidate)
		if err == nil && year >= 1000 && year <= 9999 {
			return year
		}
	}
	return 0
}
