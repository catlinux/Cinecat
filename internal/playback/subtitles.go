package playback

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const subtitleConvertTimeout = 30 * time.Second

var (
	ErrSubtitleNotFound   = errors.New("playback: subtítulo no encontrado")
	ErrSubtitleConversion = errors.New("playback: no se pudo convertir el subtítulo")
)

// SubtitleTrack describe una pista de subtítulos incrustada o un archivo
// externo junto al vídeo.
type SubtitleTrack struct {
	ID          string
	Source      string
	Language    string
	Title       string
	Codec       string
	Forced      bool
	Default     bool
	StreamIndex int
	Path        string
}

// DiscoverSubtitles reúne las pistas de texto incrustadas y los archivos
// externos con el mismo nombre base que el vídeo.
func DiscoverSubtitles(videoPath string, info *FileInfo) []SubtitleTrack {
	tracks := make([]SubtitleTrack, 0)
	if info != nil {
		for position, stream := range info.Subtitles {
			if !stream.Text {
				continue
			}
			tracks = append(tracks, SubtitleTrack{
				ID:          fmt.Sprintf("e%d", position),
				Source:      "embedded",
				Language:    stream.Language,
				Title:       stream.Title,
				Codec:       stream.Codec,
				Forced:      stream.Forced,
				Default:     stream.Default,
				StreamIndex: position,
			})
		}
	}
	for index, track := range externalSubtitles(videoPath) {
		track.ID = fmt.Sprintf("x%d", index)
		track.Source = "external"
		tracks = append(tracks, track)
	}
	return tracks
}

func externalSubtitles(videoPath string) []SubtitleTrack {
	name := filepath.Base(videoPath)
	base := strings.TrimSuffix(name, filepath.Ext(name))
	dir := filepath.Dir(videoPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	type item struct {
		name  string
		track SubtitleTrack
	}
	items := make([]item, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileName := entry.Name()
		extension := strings.ToLower(filepath.Ext(fileName))
		codec := subtitleCodec(extension)
		if codec == "" {
			continue
		}
		stem := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		if !strings.HasPrefix(stem, base) {
			continue
		}
		suffix := strings.TrimPrefix(stem, base)
		if suffix != "" && !strings.HasPrefix(suffix, ".") {
			continue
		}
		language, forced, ok := parseSubtitleSuffix(suffix)
		if !ok {
			continue
		}
		items = append(items, item{name: fileName, track: SubtitleTrack{
			Language: language,
			Codec:    codec,
			Forced:   forced,
			Path:     filepath.Join(dir, fileName),
		}})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].name < items[j].name })
	result := make([]SubtitleTrack, 0, len(items))
	for _, item := range items {
		result = append(result, item.track)
	}
	return result
}

func subtitleCodec(extension string) string {
	switch extension {
	case ".srt":
		return "subrip"
	case ".vtt":
		return "webvtt"
	case ".ass", ".ssa":
		return "ass"
	default:
		return ""
	}
}

func parseSubtitleSuffix(suffix string) (language string, forced bool, ok bool) {
	if suffix == "" {
		return "", false, true
	}
	parts := strings.Split(strings.TrimPrefix(suffix, "."), ".")
	switch len(parts) {
	case 1:
		if strings.EqualFold(parts[0], "forced") {
			return "", true, true
		}
		if validLanguageCode(parts[0]) {
			return parts[0], false, true
		}
	case 2:
		if strings.EqualFold(parts[1], "forced") && validLanguageCode(parts[0]) {
			return parts[0], true, true
		}
	}
	return "", false, false
}

func validLanguageCode(value string) bool {
	if value == "" || len(value) > 8 {
		return false
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && char != '-' {
			return false
		}
	}
	return true
}

// ValidSubtitleID acepta los identificadores e<N> y x<N>.
func ValidSubtitleID(id string) bool {
	if len(id) < 2 || (id[0] != 'e' && id[0] != 'x') {
		return false
	}
	for _, char := range id[1:] {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// SubtitleWebVTT devuelve el WebVTT de una pista. Las conversiones se cachean
// por ítem, identificador y fecha de modificación del origen.
func (s *Service) SubtitleWebVTT(ctx context.Context, itemID int64, videoPath string, track SubtitleTrack) ([]byte, error) {
	if s == nil || !ValidSubtitleID(track.ID) {
		return nil, ErrSubtitleNotFound
	}
	sourcePath := videoPath
	if track.Source == "external" {
		sourcePath = track.Path
	}
	stat, err := os.Stat(sourcePath)
	if err != nil || stat.IsDir() {
		return nil, ErrSubtitleNotFound
	}

	cacheDir := filepath.Join(s.workDir, "subtitles", fmt.Sprintf("item-%d", itemID))
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSubtitleConversion, err)
	}
	cachePath := filepath.Join(cacheDir,
		fmt.Sprintf("%s-%d.vtt", track.ID, stat.ModTime().UnixNano()))
	if data, err := os.ReadFile(cachePath); err == nil && len(data) > 0 {
		return data, nil
	}

	if track.Source == "external" && strings.EqualFold(filepath.Ext(sourcePath), ".vtt") {
		raw, err := os.ReadFile(sourcePath)
		if err != nil {
			return nil, ErrSubtitleNotFound
		}
		data := decodeSubtitleBytes(raw)
		if err := os.WriteFile(cachePath, data, 0o644); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrSubtitleConversion, err)
		}
		return data, nil
	}

	if track.Source != "embedded" && track.Source != "external" {
		return nil, ErrSubtitleNotFound
	}

	args := []string{
		"-hide_banner", "-loglevel", "error", "-nostdin", "-y",
	}
	var tempInput string
	if track.Source == "embedded" {
		args = append(args, "-i", videoPath,
			"-map", fmt.Sprintf("0:s:%d", track.StreamIndex),
			"-f", "webvtt", cachePath)
	} else {
		raw, err := os.ReadFile(sourcePath)
		if err != nil {
			return nil, ErrSubtitleNotFound
		}
		temp, err := os.CreateTemp(cacheDir, "subtitle-*"+strings.ToLower(filepath.Ext(sourcePath)))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrSubtitleConversion, err)
		}
		tempInput = temp.Name()
		if _, err := temp.Write(decodeSubtitleBytes(raw)); err != nil {
			temp.Close()
			_ = os.Remove(tempInput)
			return nil, fmt.Errorf("%w: %v", ErrSubtitleConversion, err)
		}
		if err := temp.Close(); err != nil {
			_ = os.Remove(tempInput)
			return nil, fmt.Errorf("%w: %v", ErrSubtitleConversion, err)
		}
		args = append(args, "-i", tempInput, "-f", "webvtt", cachePath)
	}
	if tempInput != "" {
		defer os.Remove(tempInput)
	}

	if err := s.runSubtitleFFmpeg(ctx, args...); err != nil {
		_ = os.Remove(cachePath)
		return nil, err
	}
	data, err := os.ReadFile(cachePath)
	if err != nil || len(data) == 0 {
		_ = os.Remove(cachePath)
		return nil, fmt.Errorf("%w: salida vacía", ErrSubtitleConversion)
	}
	return data, nil
}

func (s *Service) runSubtitleFFmpeg(ctx context.Context, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, subtitleConvertTimeout)
	defer cancel()

	var cmd *exec.Cmd
	if s.command != nil {
		cmd = s.command(ctx, "ffmpeg", args...)
		if cmd == nil {
			return fmt.Errorf("%w: comando de prueba vacío", ErrSubtitleConversion)
		}
	} else {
		ffmpeg, err := exec.LookPath("ffmpeg")
		if err != nil {
			return ErrFFmpegNotFound
		}
		cmd = ffmpegCommand(ctx, ffmpeg, args...)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%w: tiempo agotado", ErrSubtitleConversion)
		}
		return fmt.Errorf("%w: %v: %s", ErrSubtitleConversion, err, tailString(stderr.String(), 300))
	}
	return nil
}

// decodeSubtitleBytes convierte Latin-1/Windows-1252 a UTF-8 si el archivo no
// es ya UTF-8 válido.
func decodeSubtitleBytes(data []byte) []byte {
	if utf8.Valid(data) {
		return data
	}
	var builder strings.Builder
	builder.Grow(len(data))
	for _, value := range data {
		if value < 0x80 {
			builder.WriteByte(value)
			continue
		}
		if value >= 0x80 && value <= 0x9F {
			r := cp1252High[value-0x80]
			if r == 0 {
				r = rune(value)
			}
			builder.WriteRune(r)
			continue
		}
		builder.WriteRune(rune(value))
	}
	return []byte(builder.String())
}

var cp1252High = [32]rune{
	'€', 0, '‚', 'ƒ', '„', '…', '†', '‡',
	'ˆ', '‰', 'Š', '‹', 'Œ', 0, 'Ž', 0,
	0, '‘', '’', '“', '”', '•', '–', '—',
	'˜', '™', 'š', '›', 'œ', 0, 'ž', 'Ÿ',
}
