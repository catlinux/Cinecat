package playback

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// FileInfo es la ficha técnica de un archivo de vídeo obtenida con ffprobe.
type FileInfo struct {
	Container         string           `json:"container"`
	ContainerLongName string           `json:"container_long_name"`
	SizeBytes         int64            `json:"size_bytes"`
	DurationSeconds   float64          `json:"duration_seconds"`
	BitRate           int64            `json:"bit_rate"`
	ModifiedAt        string           `json:"modified_at"`
	FileName          string           `json:"file_name"`
	Video             []VideoStream    `json:"video"`
	Audio             []AudioStream    `json:"audio"`
	Subtitles         []SubtitleStream `json:"subtitles"`
	Chapters          int              `json:"chapters"`
	Quality           string           `json:"quality"`
	Tags              AudioTags        `json:"tags,omitempty"`
	HasEmbeddedCover  bool             `json:"has_embedded_cover,omitempty"`
}

type VideoStream struct {
	Index              int     `json:"index"`
	Codec              string  `json:"codec"`
	CodecLongName      string  `json:"codec_long_name"`
	Profile            string  `json:"profile"`
	Width              int     `json:"width"`
	Height             int     `json:"height"`
	DisplayAspectRatio string  `json:"display_aspect_ratio"`
	FrameRate          float64 `json:"frame_rate"`
	BitRate            int64   `json:"bit_rate"`
	PixelFormat        string  `json:"pixel_format"`
	BitDepth           int     `json:"bit_depth"`
	HDR                string  `json:"hdr"`
	FieldOrder         string  `json:"field_order"`
	Language           string  `json:"language"`
	Title              string  `json:"title"`
	Default            bool    `json:"default"`
}

type AudioStream struct {
	Index         int    `json:"index"`
	Codec         string `json:"codec"`
	CodecLongName string `json:"codec_long_name"`
	Profile       string `json:"profile"`
	Channels      int    `json:"channels"`
	ChannelLayout string `json:"channel_layout"`
	SampleRate    int    `json:"sample_rate"`
	BitRate       int64  `json:"bit_rate"`
	Language      string `json:"language"`
	Title         string `json:"title"`
	Default       bool   `json:"default"`
}

type SubtitleStream struct {
	Index    int    `json:"index"`
	Codec    string `json:"codec"`
	Language string `json:"language"`
	Title    string `json:"title"`
	Forced   bool   `json:"forced"`
	Default  bool   `json:"default"`
	Text     bool   `json:"text"`
}

// Inspect obtiene la ficha técnica con ffprobe.
func (p *Prober) Inspect(ctx context.Context, path string) (*FileInfo, error) {
	output, err := p.runFFprobeOutput(ctx, path, "json",
		"-show_format", "-show_streams", "-show_chapters")
	if err != nil {
		return nil, err
	}
	info, err := parseFFprobeJSON([]byte(output))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProbeFailed, err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProbeFailed, err)
	}
	stat, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProbeFailed, err)
	}
	info.FileName = filepath.Base(absPath)
	info.SizeBytes = stat.Size()
	info.ModifiedAt = stat.ModTime().UTC().Format(time.RFC3339)
	if info.Container == "" {
		info.Container = strings.TrimPrefix(strings.ToLower(filepath.Ext(absPath)), ".")
	}
	return info, nil
}

// Inspect delega en el analizador del servicio para que la API no dependa de
// detalles internos de reproducción.
func (s *Service) Inspect(ctx context.Context, path string) (*FileInfo, error) {
	if s == nil || s.prober == nil {
		return nil, ErrFFprobeNotFound
	}
	return s.prober.Inspect(ctx, path)
}

type ffprobeOutput struct {
	Streams  []ffprobeStream   `json:"streams"`
	Format   ffprobeFormat     `json:"format"`
	Chapters []json.RawMessage `json:"chapters"`
}

type ffprobeFormat struct {
	Filename       string            `json:"filename"`
	FormatName     string            `json:"format_name"`
	FormatLongName string            `json:"format_long_name"`
	Duration       ffFloat           `json:"duration"`
	Size           ffInt             `json:"size"`
	BitRate        ffInt             `json:"bit_rate"`
	Tags           map[string]string `json:"tags"`
}

type ffprobeStream struct {
	Index              ffInt             `json:"index"`
	CodecName          string            `json:"codec_name"`
	CodecLongName      string            `json:"codec_long_name"`
	Profile            string            `json:"profile"`
	CodecType          string            `json:"codec_type"`
	Width              ffInt             `json:"width"`
	Height             ffInt             `json:"height"`
	DisplayAspectRatio string            `json:"display_aspect_ratio"`
	AvgFrameRate       string            `json:"avg_frame_rate"`
	RFrameRate         string            `json:"r_frame_rate"`
	BitRate            ffInt             `json:"bit_rate"`
	PixelFormat        string            `json:"pix_fmt"`
	BitsPerRawSample   ffInt             `json:"bits_per_raw_sample"`
	SampleRate         ffInt             `json:"sample_rate"`
	Channels           ffInt             `json:"channels"`
	ChannelLayout      string            `json:"channel_layout"`
	ColorTransfer      string            `json:"color_transfer"`
	FieldOrder         string            `json:"field_order"`
	Tags               map[string]string `json:"tags"`
	Disposition        struct {
		Default     ffInt `json:"default"`
		Forced      ffInt `json:"forced"`
		AttachedPic ffInt `json:"attached_pic"`
	} `json:"disposition"`
	SideDataList []struct {
		SideDataType string `json:"side_data_type"`
	} `json:"side_data_list"`
}

type ffInt int64

func (v *ffInt) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*v = 0
		return nil
	}
	raw = strings.Trim(raw, `"`)
	if raw == "" || strings.EqualFold(raw, "N/A") {
		*v = 0
		return nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		*v = 0
		return nil
	}
	*v = ffInt(int64(value))
	return nil
}

type ffFloat float64

func (v *ffFloat) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*v = 0
		return nil
	}
	raw = strings.Trim(raw, `"`)
	if raw == "" || strings.EqualFold(raw, "N/A") {
		*v = 0
		return nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		*v = 0
		return nil
	}
	*v = ffFloat(value)
	return nil
}

func parseFFprobeJSON(data []byte) (*FileInfo, error) {
	var raw ffprobeOutput
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("playback: JSON de ffprobe no válido: %w", err)
	}

	filename := cleanFFString(raw.Format.Filename)
	if filename != "" {
		filename = filepath.Base(filename)
	}
	info := &FileInfo{
		Container:         containerName(raw.Format.FormatName, raw.Format.Filename),
		ContainerLongName: cleanFFString(raw.Format.FormatLongName),
		SizeBytes:         int64(raw.Format.Size),
		DurationSeconds:   float64(raw.Format.Duration),
		BitRate:           int64(raw.Format.BitRate),
		FileName:          filename,
		Video:             make([]VideoStream, 0),
		Audio:             make([]AudioStream, 0),
		Subtitles:         make([]SubtitleStream, 0),
		Chapters:          len(raw.Chapters),
		Tags:              audioTagsFromTags(raw.Format.Tags),
	}

	for _, stream := range raw.Streams {
		if stream.Disposition.AttachedPic != 0 {
			info.HasEmbeddedCover = true
		}
		switch strings.ToLower(cleanFFString(stream.CodecType)) {
		case "video":
			streamInfo := VideoStream{
				Index:              int(stream.Index),
				Codec:              cleanFFString(stream.CodecName),
				CodecLongName:      cleanFFString(stream.CodecLongName),
				Profile:            cleanFFString(stream.Profile),
				Width:              int(stream.Width),
				Height:             int(stream.Height),
				DisplayAspectRatio: cleanFFString(stream.DisplayAspectRatio),
				FrameRate:          parseFrameRate(stream.AvgFrameRate),
				BitRate:            int64(stream.BitRate),
				PixelFormat:        cleanFFString(stream.PixelFormat),
				BitDepth:           bitDepth(stream),
				HDR:                hdrLabel(stream),
				FieldOrder:         cleanFFString(stream.FieldOrder),
				Language:           tagValue(stream.Tags, "language"),
				Title:              tagValue(stream.Tags, "title"),
				Default:            stream.Disposition.Default != 0,
			}
			if streamInfo.FrameRate == 0 {
				streamInfo.FrameRate = parseFrameRate(stream.RFrameRate)
			}
			info.Video = append(info.Video, streamInfo)
		case "audio":
			info.Audio = append(info.Audio, AudioStream{
				Index:         int(stream.Index),
				Codec:         cleanFFString(stream.CodecName),
				CodecLongName: cleanFFString(stream.CodecLongName),
				Profile:       cleanFFString(stream.Profile),
				Channels:      int(stream.Channels),
				ChannelLayout: cleanFFString(stream.ChannelLayout),
				SampleRate:    int(stream.SampleRate),
				BitRate:       int64(stream.BitRate),
				Language:      tagValue(stream.Tags, "language"),
				Title:         tagValue(stream.Tags, "title"),
				Default:       stream.Disposition.Default != 0,
			})
		case "subtitle":
			codec := cleanFFString(stream.CodecName)
			info.Subtitles = append(info.Subtitles, SubtitleStream{
				Index:    int(stream.Index),
				Codec:    codec,
				Language: tagValue(stream.Tags, "language"),
				Title:    tagValue(stream.Tags, "title"),
				Forced:   stream.Disposition.Forced != 0,
				Default:  stream.Disposition.Default != 0,
				Text:     subtitleIsText(codec),
			})
		}
	}
	info.Quality = qualityLabel(info.Video)
	return info, nil
}

func containerName(formatName, filename string) string {
	first := strings.TrimSpace(strings.Split(cleanFFString(formatName), ",")[0])
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(cleanFFString(filename))), ".")
	if ext == "m4v" {
		ext = "mp4"
	}
	if ext == "mkv" {
		return "matroska"
	}
	for _, name := range strings.Split(cleanFFString(formatName), ",") {
		if strings.EqualFold(strings.TrimSpace(name), ext) {
			return ext
		}
	}
	if first != "" {
		return first
	}
	return ext
}

func cleanFFString(value string) string {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "N/A") {
		return ""
	}
	return value
}

func tagValue(tags map[string]string, key string) string {
	if tags == nil {
		return ""
	}
	return cleanFFString(tags[key])
}

func parseFrameRate(value string) float64 {
	value = cleanFFString(value)
	if value == "" {
		return 0
	}
	parts := strings.Split(value, "/")
	if len(parts) == 2 {
		numerator, err1 := strconv.ParseFloat(parts[0], 64)
		denominator, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 == nil && err2 == nil && denominator != 0 {
			return numerator / denominator
		}
		return 0
	}
	rate, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return rate
}

func bitDepth(stream ffprobeStream) int {
	if stream.BitsPerRawSample > 0 {
		return int(stream.BitsPerRawSample)
	}
	pixelFormat := strings.ToLower(cleanFFString(stream.PixelFormat))
	if strings.HasSuffix(pixelFormat, "le") || strings.HasSuffix(pixelFormat, "be") {
		core := pixelFormat[:len(pixelFormat)-2]
		index := len(core) - 1
		for index >= 0 && core[index] >= '0' && core[index] <= '9' {
			index--
		}
		if index < len(core)-1 {
			if value, err := strconv.Atoi(core[index+1:]); err == nil {
				switch value {
				case 9, 10, 12, 14, 16:
					return value
				}
			}
		}
	}
	return 8
}

func hdrLabel(stream ffprobeStream) string {
	for _, sideData := range stream.SideDataList {
		kind := strings.ToUpper(sideData.SideDataType)
		if strings.Contains(kind, "DOVI") || strings.Contains(kind, "DOLBY VISION") {
			return "Dolby Vision"
		}
	}
	switch strings.ToLower(cleanFFString(stream.ColorTransfer)) {
	case "smpte2084":
		return "HDR10"
	case "arib-std-b67":
		return "HLG"
	default:
		return ""
	}
}

func subtitleIsText(codec string) bool {
	switch strings.ToLower(codec) {
	case "subrip", "ass", "ssa", "webvtt", "mov_text", "text":
		return true
	default:
		return false
	}
}

func qualityLabel(video []VideoStream) string {
	if len(video) == 0 {
		return ""
	}
	width, height := video[0].Width, video[0].Height
	switch {
	case width >= 3000 || height >= 2000:
		return "4K"
	case height >= 1000 || width >= 1800:
		return "Full HD"
	case height >= 700 || width >= 1200:
		return "HD"
	default:
		return "SD"
	}
}
