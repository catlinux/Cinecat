package playback

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// hlsArgs construye los argumentos de FFmpeg para reempaquetar a HLS fMP4 sin
// recodificar. Cada argumento va en su propio elemento de la lista: el nombre
// del archivo nunca se interpola en una cadena de comando. Los nombres de
// salida son relativos y se resuelven con el directorio de trabajo del proceso.
func hlsArgs(inputPath string) []string {
	return []string{
		"-hide_banner",
		"-loglevel", "error",
		"-nostdin",
		"-y",
		"-i", inputPath,
		"-map", "0:v:0",
		"-map", "0:a?",
		"-c:v", "copy",
		"-c:a", "copy",
		"-f", "hls",
		"-hls_time", "6",
		"-hls_playlist_type", "vod",
		"-hls_segment_type", "fmp4",
		"-hls_fmp4_init_filename", "init.mp4",
		"-hls_segment_filename", "segment_%05d.m4s",
		"playlist.m3u8",
	}
}

// ServeDirect sirve el archivo tal cual con http.ServeContent, que admite
// peticiones con Range.
func ServeDirect(w http.ResponseWriter, r *http.Request, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFileNotFound, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFileNotFound, err)
	}
	if info.IsDir() {
		return ErrFileNotFound
	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	return nil
}

// ServeHLSFile prepara el remux si hace falta y sirve la playlist (con las
// URLs de los segmentos firmadas) o uno de sus archivos.
func (s *Service) ServeHLSFile(w http.ResponseWriter, r *http.Request, itemID int64, inputPath, name, exp, sig string) error {
	if !ValidHLSName(name) {
		return ErrFileNotFound
	}

	entry, err := s.prepare(r.Context(), itemID, inputPath)
	if err != nil {
		return err
	}

	filePath := filepath.Join(entry.dir, name)
	if name == "playlist.m3u8" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrFileNotFound, err)
		}
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(signPlaylist(data, signedQuery(exp, sig))); err != nil {
			return fmt.Errorf("playback: escribir la playlist: %w", err)
		}
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFileNotFound, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFileNotFound, err)
	}
	if info.IsDir() {
		return ErrFileNotFound
	}
	if strings.HasSuffix(name, ".m4s") {
		w.Header().Set("Content-Type", "video/iso.segment")
	}
	http.ServeContent(w, r, name, info.ModTime(), file)
	return nil
}

// prepare genera el remux del ítem si no hay uno vigente y lo reutiliza
// mientras se siga usando.
func (s *Service) prepare(ctx context.Context, itemID int64, inputPath string) (*hlsEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked()

	if entry, ok := s.entries[itemID]; ok {
		if _, err := os.Stat(filepath.Join(entry.dir, "playlist.m3u8")); err == nil {
			entry.lastUsed = s.now()
			return entry, nil
		}
		delete(s.entries, itemID)
		_ = os.RemoveAll(entry.dir)
	}

	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, ErrFFmpegNotFound
	}

	dir := filepath.Join(s.workDir, fmt.Sprintf("item-%d", itemID))
	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf("playback: limpiar el remux anterior: %w", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("playback: crear el directorio del remux: %w", err)
	}

	absInput, err := filepath.Abs(inputPath)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("playback: ruta de entrada no válida: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, ffmpeg, hlsArgs(absInput)...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("%w: %v: %s", ErrFFmpegFailed, err, tailString(stderr.String(), 500))
	}
	if _, err := os.Stat(filepath.Join(dir, "playlist.m3u8")); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("%w: no se generó la playlist", ErrFFmpegFailed)
	}

	entry := &hlsEntry{dir: dir, lastUsed: s.now()}
	s.entries[itemID] = entry
	return entry, nil
}

func (s *Service) cleanupLocked() {
	now := s.now()
	for itemID, entry := range s.entries {
		if now.Sub(entry.lastUsed) > hlsIdleTTL {
			delete(s.entries, itemID)
			_ = os.RemoveAll(entry.dir)
		}
	}
}

// signPlaylist añade a los segmentos y al init de fMP4 los mismos parámetros
// de firma que la playlist, para que el cliente pueda pedirlos.
func signPlaylist(data []byte, query string) []byte {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			continue
		case strings.HasPrefix(trimmed, "#EXT-X-MAP:"):
			lines[i] = addQueryToURI(line, query)
		case strings.HasPrefix(trimmed, "#"):
			continue
		default:
			cr := ""
			if strings.HasSuffix(line, "\r") {
				line = strings.TrimSuffix(line, "\r")
				cr = "\r"
			}
			lines[i] = line + query + cr
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func addQueryToURI(line, query string) string {
	const marker = `URI="`
	start := strings.Index(line, marker)
	if start < 0 {
		return line
	}
	start += len(marker)
	rest := line[start:]
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		return line
	}
	return line[:start] + rest[:end] + query + rest[end:]
}

func signedQuery(exp, sig string) string {
	values := url.Values{}
	values.Set("exp", exp)
	values.Set("sig", sig)
	return "?" + values.Encode()
}

func tailString(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[len(value)-max:]
}
