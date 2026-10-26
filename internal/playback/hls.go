package playback

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// hlsStartupTimeout acota lo que espera prepare() a que aparezca la primera
	// playlist. Con veryfast, el primer segmento de 6 s suele tardar bastante
	// menos; 20 s evita dejar una petición HTTP colgada si FFmpeg no arranca.
	hlsStartupTimeout = 20 * time.Second
	// hlsSegmentPoll es el intervalo de comprobación de segmentos: 150 ms da
	// una latencia baja sin convertir la espera en un bucle de CPU.
	hlsSegmentPoll = 150 * time.Millisecond
	// hlsSegmentWaitTimeout limita la espera de un segmento concreto; cinco
	// minutos cubren con holgura el avance normal de una transcodificación.
	hlsSegmentWaitTimeout = 5 * time.Minute
	// hlsSegmentSeconds es el hls_time que se pide a FFmpeg y el tamaño de los
	// segmentos que Cinecat lista en su playlist VOD.
	hlsSegmentSeconds = 6
	// hlsProcessTimeout es el límite total del proceso FFmpeg en segundo plano.
	// Antes eran 30 min, pero una película larga transcodificada por software en
	// un servidor modesto puede superarlos; como ya no bloquea peticiones, se
	// deja un margen amplio (2 h) y el proceso se cancela al cerrar el servicio.
	hlsProcessTimeout = 2 * time.Hour
)

// hlsPlaylistType es event para la playlist interna que escribe FFmpeg.
// Cinecat sirve normalmente su propia playlist VOD completa (con todos los
// segmentos y ENDLIST desde el principio); la de FFmpeg solo se sirve como
// fallback si no se pudo calcular la duración. Event es necesario en ese
// fallback para que la playlist exista antes de que termine el proceso.
const hlsPlaylistType = "event"

// ffmpegPlaylistFile es la playlist interna de FFmpeg. No se sirve al cliente
// cuando Cinecat genera la suya (playlist.m3u8), así FFmpeg no la pisa.
const ffmpegPlaylistFile = "ffmpeg.m3u8"

// Errores del flujo progresivo.
var (
	ErrHLSStartupTimeout = errors.New("playback: ffmpeg no generó la playlist HLS a tiempo")
	ErrHLSSegmentTimeout = errors.New("playback: el segmento HLS no apareció a tiempo")
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
		"-hls_time", strconv.Itoa(hlsSegmentSeconds),
		"-hls_playlist_type", hlsPlaylistType,
		"-hls_segment_type", "fmp4",
		"-hls_fmp4_init_filename", "init.mp4",
		"-hls_segment_filename", "segment_%05d.m4s",
		ffmpegPlaylistFile,
	}
}

// hlsTranscodeArgs construye los argumentos para el caso en que el remux sin
// recodificar no es viable. Se recodifica solo el vídeo con libx264 veryfast y
// CRF 20: es un equilibrio razonable entre velocidad y calidad para una
// biblioteca personal, sin depender de aceleración por hardware. Se fuerzan
// keyframes cada 6 s para que los segmentos HLS ronden hls_time. El audio se
// convierte a AAC 160k porque en AVI/DivX antiguos puede ser MP3, AC-3 o DTS y
// no hay garantía de que el navegador lo reproduzca; AAC es universal y barato
// comparado con recodificar el vídeo.
func hlsTranscodeArgs(inputPath string) []string {
	return []string{
		"-hide_banner",
		"-loglevel", "error",
		"-nostdin",
		"-y",
		"-i", inputPath,
		"-map", "0:v:0",
		"-map", "0:a?",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "20",
		"-pix_fmt", "yuv420p",
		"-force_key_frames", "expr:gte(t,n_forced*6)",
		"-c:a", "aac",
		"-b:a", "160k",
		"-f", "hls",
		"-hls_time", strconv.Itoa(hlsSegmentSeconds),
		"-hls_playlist_type", hlsPlaylistType,
		"-hls_segment_type", "fmp4",
		"-hls_fmp4_init_filename", "init.mp4",
		"-hls_segment_filename", "segment_%05d.m4s",
		ffmpegPlaylistFile,
	}
}

// needsTranscode decide si hace falta recodificar el vídeo en vez de copiarlo.
//
// El análisis de intervalo entre keyframes (prober.CheckViable) no basta por
// sí solo: un .avi real de producción con 2804 keyframes bien repartidos (muy
// por debajo del umbral de MaxViableKeyframeInterval) seguía generando un
// único segmento HLS con toda la película, porque FFmpeg -f hls -c:v copy
// necesita timestamps de paquete fiables para cortar en el punto exacto, y en
// AVI/DivX antiguos buena parte de los paquetes llevan pts_time "N/A"
// (reconstruido desde dts). El intervalo entre keyframes puede ser correcto y
// aun así el contenedor no ser segmentable sin recodificar.
//
// Por eso cualquier .avi se transcodifica siempre, sin pasar por el prober: es
// el contenedor donde se vio el problema real y donde los timestamps son
// menos fiables en general. Para el resto de contenedores (.mkv, .mov, .ts,
// .m2ts) se mantiene la detección por intervalo de keyframes, que sigue siendo
// válida para el caso que resuelve (GOP realmente largo).
func (s *Service) needsTranscode(ctx context.Context, inputPath string) bool {
	if strings.EqualFold(filepath.Ext(inputPath), ".avi") {
		return true
	}
	if s.prober == nil {
		return false
	}
	return s.prober.CheckViable(ctx, inputPath) != nil
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
		data, err := os.ReadFile(filepath.Join(entry.dir, entry.playlistFile))
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

	if err := s.waitForHLSFile(r.Context(), entry, name); err != nil {
		return err
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

// prepare arranca FFmpeg en segundo plano y devuelve la entrada en cuanto
// existe la playlist inicial, sin esperar a que termine el vídeo completo.
func (s *Service) prepare(ctx context.Context, itemID int64, inputPath string) (*hlsEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked()

	if entry, ok := s.entries[itemID]; ok {
		entry.lastUsed = s.now()
		if s.playlistExists(entry) {
			return entry, nil
		}
		if s.entryFinished(entry) {
			delete(s.entries, itemID)
			_ = os.RemoveAll(entry.dir)
			if entry.err != nil {
				return nil, entry.err
			}
		} else {
			if err := s.waitForPlaylist(ctx, entry); err != nil {
				if errors.Is(err, ErrHLSStartupTimeout) {
					s.stopEntryLocked(itemID, entry)
				}
				return nil, err
			}
			return entry, nil
		}
	}

	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, ErrFFmpegNotFound
	}

	absInput, err := filepath.Abs(inputPath)
	if err != nil {
		return nil, fmt.Errorf("playback: ruta de entrada no válida: %w", err)
	}

	transcode := s.needsTranscode(ctx, absInput)
	args := hlsArgs(absInput)
	mode := ModeRemux
	if transcode {
		args = hlsTranscodeArgs(absInput)
		mode = ModeTranscode
	}

	dir := filepath.Join(s.workDir, fmt.Sprintf("item-%d", itemID))
	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf("playback: limpiar el remux anterior: %w", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("playback: crear el directorio del remux: %w", err)
	}

	// Si se conoce la duración, Cinecat escribe una playlist VOD completa con
	// todos los segmentos; FFmpeg escribe la suya aparte (ffmpeg.m3u8). Si no,
	// se cae al comportamiento anterior y se sirve la playlist event de FFmpeg
	// cuando aparezca.
	playlistFile := ffmpegPlaylistFile
	generatedPlaylist := false
	if durations, playlistErr := s.playlistDurations(ctx, absInput, mode); playlistErr == nil {
		if err := writeVODPlaylist(filepath.Join(dir, "playlist.m3u8"), durations); err != nil {
			_ = os.RemoveAll(dir)
			return nil, fmt.Errorf("playback: escribir la playlist VOD: %w", err)
		}
		playlistFile = "playlist.m3u8"
		generatedPlaylist = true
	}

	// El proceso vive ligado al contexto del servicio, no al de la petición
	// HTTP que lo arranca, para que no se cancele al terminar esa petición.
	procCtx, cancel := context.WithTimeout(s.procCtx, hlsProcessTimeout)
	cmd := exec.CommandContext(procCtx, ffmpeg, args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		cancel()
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("%w: %v", ErrFFmpegFailed, err)
	}

	entry := &hlsEntry{
		dir:          dir,
		lastUsed:     s.now(),
		mode:         mode,
		playlistFile: playlistFile,
		done:         make(chan struct{}),
		cancel:       cancel,
	}
	s.entries[itemID] = entry

	go func() {
		err := cmd.Wait()
		if err != nil {
			entry.err = fmt.Errorf("%w: %v: %s", ErrFFmpegFailed, err, tailString(stderr.String(), 500))
		} else if _, statErr := os.Stat(filepath.Join(dir, "playlist.m3u8")); statErr != nil {
			entry.err = fmt.Errorf("%w: no se generó la playlist", ErrFFmpegFailed)
		}
		cancel()
		close(entry.done)
	}()

	if !generatedPlaylist {
		if err := s.waitForPlaylist(ctx, entry); err != nil {
			if ctx.Err() != nil {
				// El cliente se fue, pero el proceso sigue en segundo plano y otra
				// petición podrá reutilizarlo.
				return nil, ctx.Err()
			}
			s.stopEntryLocked(itemID, entry)
			return nil, err
		}
	}
	return entry, nil
}

// playlistDurations calcula los segmentos esperados. En transcodificación son
// bloques de hlsSegmentSeconds. En remux sin recodificar FFmpeg solo puede
// cortar en keyframes, así que se usan esos tiempos para que la playlist
// generada coincida con los segmentos reales.
func (s *Service) playlistDurations(ctx context.Context, path string, mode Mode) ([]time.Duration, error) {
	if s.prober == nil {
		return nil, ErrDurationUnknown
	}

	duration, err := s.prober.VideoDuration(ctx, path)
	if err != nil || duration <= 0 {
		return nil, ErrDurationUnknown
	}

	if mode == ModeTranscode {
		return uniformSegmentDurations(duration), nil
	}

	keyframes, err := s.prober.KeyframeTimes(ctx, path)
	if err != nil || len(keyframes) == 0 || keyframes[0] > time.Second {
		return nil, ErrDurationUnknown
	}
	return keyframeSegmentDurations(duration, keyframes), nil
}

func uniformSegmentDurations(duration time.Duration) []time.Duration {
	segment := time.Duration(hlsSegmentSeconds) * time.Second
	var durations []time.Duration
	for remaining := duration; remaining > 0; {
		take := segment
		if remaining < take {
			take = remaining
		}
		durations = append(durations, take)
		remaining -= take
	}
	return durations
}

func keyframeSegmentDurations(duration time.Duration, keyframes []time.Duration) []time.Duration {
	segment := time.Duration(hlsSegmentSeconds) * time.Second
	var durations []time.Duration
	start := time.Duration(0)
	for _, keyframe := range keyframes {
		if keyframe <= start {
			continue
		}
		if keyframe >= duration {
			break
		}
		if keyframe-start >= segment {
			durations = append(durations, keyframe-start)
			start = keyframe
		}
	}
	if duration > start {
		durations = append(durations, duration-start)
	}
	if len(durations) == 0 {
		durations = append(durations, duration)
	}
	return durations
}

// writeVODPlaylist escribe una playlist VOD completa antes de arrancar FFmpeg.
func writeVODPlaylist(path string, durations []time.Duration) error {
	var maxDuration time.Duration
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	b.WriteString("#EXT-X-VERSION:7\n")
	for _, duration := range durations {
		if duration > maxDuration {
			maxDuration = duration
		}
	}
	target := int(math.Ceil(maxDuration.Seconds()))
	if target < 1 {
		target = 1
	}
	fmt.Fprintf(&b, "#EXT-X-TARGETDURATION:%d\n", target)
	b.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")
	b.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")
	b.WriteString("#EXT-X-MAP:URI=\"init.mp4\"\n")
	for index, duration := range durations {
		fmt.Fprintf(&b, "#EXTINF:%.6f,\nsegment_%05d.m4s\n", duration.Seconds(), index)
	}
	b.WriteString("#EXT-X-ENDLIST\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func (s *Service) playlistExists(entry *hlsEntry) bool {
	_, err := os.Stat(filepath.Join(entry.dir, entry.playlistFile))
	return err == nil
}

func (s *Service) entryFinished(entry *hlsEntry) bool {
	select {
	case <-entry.done:
		return true
	default:
		return false
	}
}

// waitForPlaylist espera a que FFmpeg escriba la playlist inicial. Con
// hls_playlist_type event aparece tras el primer segmento, no al final.
func (s *Service) waitForPlaylist(ctx context.Context, entry *hlsEntry) error {
	playlist := filepath.Join(entry.dir, entry.playlistFile)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(hlsStartupTimeout)
	defer deadline.Stop()

	for {
		if _, err := os.Stat(playlist); err == nil {
			return nil
		}
		select {
		case <-entry.done:
			if _, err := os.Stat(playlist); err == nil {
				return nil
			}
			if entry.err != nil {
				return entry.err
			}
			return fmt.Errorf("%w: ffmpeg terminó sin generar la playlist", ErrFFmpegFailed)
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return ErrHLSStartupTimeout
		case <-ticker.C:
		}
	}
}

func (s *Service) stopEntryLocked(itemID int64, entry *hlsEntry) {
	entry.cancel()
	<-entry.done
	delete(s.entries, itemID)
	_ = os.RemoveAll(entry.dir)
}

// waitForHLSFile espera a que un segmento (o el init de fMP4) esté listo para
// servirse. Un segmento se considera completo si existe y FFmpeg ya terminó o
// ya existe el segmento siguiente; así no se sirve un archivo a medio escribir.
func (s *Service) waitForHLSFile(ctx context.Context, entry *hlsEntry, name string) error {
	next, ok := hlsCompletionFile(name)
	if !ok {
		return ErrFileNotFound
	}

	ctx, cancel := context.WithTimeout(ctx, hlsSegmentWaitTimeout)
	defer cancel()
	ticker := time.NewTicker(hlsSegmentPoll)
	defer ticker.Stop()

	path := filepath.Join(entry.dir, name)
	nextPath := filepath.Join(entry.dir, next)

	for {
		if s.hlsFileReady(entry, path, nextPath) {
			return nil
		}
		select {
		case <-entry.done:
			if s.hlsFileReady(entry, path, nextPath) {
				return nil
			}
			if entry.err != nil {
				return entry.err
			}
			return fmt.Errorf("%w: %s", ErrFileNotFound, name)
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("%w: %s", ErrHLSSegmentTimeout, name)
			}
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Service) hlsFileReady(entry *hlsEntry, path, nextPath string) bool {
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		return false
	}
	if s.entryFinished(entry) {
		return true
	}
	_, err = os.Stat(nextPath)
	return err == nil
}

// hlsCompletionFile devuelve el archivo que debe existir para dar por completo
// el solicitado: el segmento siguiente, o el primero tras init.mp4.
func hlsCompletionFile(name string) (string, bool) {
	if name == "init.mp4" {
		return "segment_00000.m4s", true
	}
	index, ok := segmentIndex(name)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("segment_%05d.m4s", index+1), true
}

func segmentIndex(name string) (int, bool) {
	if !strings.HasPrefix(name, "segment_") || !strings.HasSuffix(name, ".m4s") {
		return 0, false
	}
	index, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(name, "segment_"), ".m4s"))
	if err != nil || index < 0 {
		return 0, false
	}
	return index, true
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
