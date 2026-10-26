package playback

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
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
	// hlsStartupTimeout acota lo que espera prepare() a que aparezca la playlist
	// de FFmpeg cuando no se pudo generar la VOD propia.
	hlsStartupTimeout = 20 * time.Second
	// hlsSegmentPoll es el intervalo de comprobación de segmentos: 150 ms da
	// una latencia baja sin convertir la espera en un bucle de CPU.
	hlsSegmentPoll = 150 * time.Millisecond
	// hlsSegmentWaitTimeout limita la espera de un segmento concreto. Como un
	// salto relanza FFmpeg en el segmento pedido, nunca debería acercarse a
	// este límite salvo con un vídeo muy pesado.
	hlsSegmentWaitTimeout = 2 * time.Minute
	// hlsSegmentSeconds es el hls_time que se pide a FFmpeg y el tamaño de los
	// segmentos que Cinecat lista en su playlist VOD.
	hlsSegmentSeconds = 6
	// hlsProcessTimeout es el límite total de cada proceso FFmpeg.
	hlsProcessTimeout = 4 * time.Hour
	// hlsRestartGap es cuántos segmentos por delante de lo ya generado se
	// espera a FFmpeg antes de relanzarlo en el segmento pedido. Es el mismo
	// criterio que Jellyfin (24 s / duración de segmento).
	hlsRestartGap = 24 / hlsSegmentSeconds
	// hlsRestartMinAge es lo mínimo que debe llevar en marcha un proceso antes
	// de que otra petición lo sustituya. Sin este margen, dos peticiones
	// simultáneas a puntos lejanos se relanzarían FFmpeg la una a la otra en
	// cada comprobación sin que ninguna llegara a recibir su segmento.
	hlsRestartMinAge = 3 * time.Second
	// hlsTimestampOffsetSeconds se suma a las marcas de tiempo de salida de
	// todos los procesos. Sin él, el proceso que empieza en 0 tiene DTS
	// negativos por los B-frames y FFmpeg lo desplaza unos fotogramas, así que
	// sus segmentos no encajarían con los de un proceso relanzado más adelante.
	// Con el desplazamiento, el segmento N empieza siempre en el mismo instante
	// sea cual sea el proceso que lo generó (comprobado con FFmpeg 5.1).
	hlsTimestampOffsetSeconds = 10
	// hlsRunIdleTimeout es el tiempo sin peticiones tras el cual se para un
	// proceso FFmpeg abandonado.
	hlsRunIdleTimeout = 90 * time.Second
	// hlsMaxAheadSegments limita cuánto puede adelantarse FFmpeg al último
	// segmento pedido: 20 × 6 s ≈ 2 minutos.
	hlsMaxAheadSegments = 20
	// hlsMonitorInterval es la frecuencia con la que se revisan los procesos.
	hlsMonitorInterval = 2 * time.Second
)

// ffmpegPlaylistFile es la playlist interna de cada proceso FFmpeg. Solo lista
// los segmentos ya terminados por ese proceso, así que es la referencia para
// saber si un segmento está completo. Se sirve al cliente solo como fallback
// si no se pudo calcular la duración del vídeo.
const ffmpegPlaylistFile = "ffmpeg.m3u8"

// Errores del flujo progresivo.
var (
	ErrHLSStartupTimeout = errors.New("playback: ffmpeg no generó la playlist HLS a tiempo")
	ErrHLSSegmentTimeout = errors.New("playback: el segmento HLS no apareció a tiempo")
)

// hlsTranscodeArgs construye los argumentos de FFmpeg para generar HLS desde
// el segmento startSegment. Cada argumento va en su propio elemento de la
// lista: el nombre del archivo nunca se interpola en una cadena de comando.
//
// Se recodifica siempre: la biblioteca real es casi toda DivX/MPEG-4 con audio
// AC-3 o MP3 que ningún navegador reproduce por MSE, y en SD libx264 veryfast
// va a unas 20 veces tiempo real en el servidor. El audio se baja a AAC
// estéreo porque el AAC 5.1 no lo aceptan los reproductores del navegador
// (vídeos con AC-3 5.1 se quedaban cargando). Los segmentos son MPEG-TS y no
// fMP4 porque cada uno lleva sus marcas de tiempo absolutas: al relanzar FFmpeg
// en mitad del vídeo no hay que regenerar ningún init ni recalcular tfdt.
func hlsTranscodeArgs(inputPath string, startSegment int, transcode ...TranscodeSettings) []string {
	settings := defaultTranscodeSettings()
	if len(transcode) > 0 {
		settings = transcode[0]
	}
	settings = normalizeTranscodeSettings(settings)
	startSeconds := startSegment * hlsSegmentSeconds
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-nostdin",
		"-y",
	}
	if startSeconds > 0 {
		args = append(args, "-ss", strconv.Itoa(startSeconds))
	}
	return append(args,
		"-i", inputPath,
		// V mayúscula excluye las carátulas incrustadas como flujo de vídeo.
		"-map", "0:V:0",
		"-map", "0:a:0?",
		// libx264 con yuv420p exige dimensiones pares; la altura se limita al
		// máximo configurado y el ancho se calcula manteniendo la proporción.
		"-vf", fmt.Sprintf(`scale=-2:trunc(min(%d\,ih)/2)*2`, settings.MaxHeight),
		"-c:v", "libx264",
		"-preset", settings.Preset,
		"-crf", strconv.Itoa(settings.CRF),
		"-threads", strconv.Itoa(settings.Threads),
		"-pix_fmt", "yuv420p",
		"-force_key_frames", "expr:gte(t,n_forced*"+strconv.Itoa(hlsSegmentSeconds)+")",
		"-c:a", "aac",
		"-ac", "2",
		"-b:a", "160k",
		"-output_ts_offset", strconv.Itoa(startSeconds+hlsTimestampOffsetSeconds),
		"-f", "hls",
		"-hls_time", strconv.Itoa(hlsSegmentSeconds),
		"-hls_playlist_type", "event",
		// En FFmpeg 5.1 temp_file solo hace atómica la playlist, no los
		// segmentos; por eso un segmento se da por completo cuando aparece en
		// ella.
		"-hls_flags", "temp_file",
		"-hls_segment_type", "mpegts",
		"-start_number", strconv.Itoa(startSegment),
		"-hls_segment_filename", "segment_%05d.ts",
		ffmpegPlaylistFile,
	)
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

// ServeHLSFile prepara la entrada HLS si hace falta y sirve la playlist (con
// las URLs de los segmentos firmadas) o uno de sus segmentos.
func (s *Service) ServeHLSFile(w http.ResponseWriter, r *http.Request, itemID int64, inputPath, name, exp, sig string) error {
	if !ValidHLSName(name) {
		return ErrFileNotFound
	}

	entry, err := s.prepare(r.Context(), itemID, inputPath)
	if err != nil {
		return err
	}

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

	index, ok := segmentIndex(name)
	if !ok {
		return ErrFileNotFound
	}
	if err := s.waitForSegment(r.Context(), entry, index); err != nil {
		return err
	}

	file, err := os.Open(filepath.Join(entry.dir, name))
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
	w.Header().Set("Content-Type", "video/mp2t")
	http.ServeContent(w, r, name, info.ModTime(), file)
	return nil
}

// prepare crea la entrada HLS del ítem. Si se conoce la duración, escribe una
// playlist VOD completa y no arranca FFmpeg: lo hará la primera petición de
// segmento, en el punto que pida. Si no, arranca FFmpeg desde el principio y
// espera a su playlist, que se sirve tal cual (sin posibilidad de saltos
// lejanos rápidos).
func (s *Service) prepare(ctx context.Context, itemID int64, inputPath string) (*hlsEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked()

	transcode := s.currentTranscodeSettings()
	key := hlsEntryKey{itemID: itemID, transcode: transcodeKey(transcode)}
	if entry, ok := s.entries[key]; ok {
		now := s.now()
		entry.lastUsed = now
		entry.lastRequest = now
		if s.playlistExists(entry) {
			return entry, nil
		}
		if entry.run != nil && !entry.run.finished() {
			if err := s.waitForPlaylist(ctx, entry); err != nil {
				if errors.Is(err, ErrHLSStartupTimeout) {
					s.removeEntryLocked(key, entry)
				}
				return nil, err
			}
			return entry, nil
		}
		s.removeEntryLocked(key, entry)
		if entry.run != nil && entry.run.err != nil {
			return nil, entry.run.err
		}
	}

	if err := s.ffmpegAvailable(); err != nil {
		return nil, err
	}

	absInput, err := filepath.Abs(inputPath)
	if err != nil {
		return nil, fmt.Errorf("playback: ruta de entrada no válida: %w", err)
	}

	dir := filepath.Join(s.workDir, fmt.Sprintf("item-%d-%s", itemID, key.transcode))
	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf("playback: limpiar el HLS anterior: %w", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("playback: crear el directorio HLS: %w", err)
	}

	now := s.now()
	entry := &hlsEntry{
		itemID:           itemID,
		dir:              dir,
		inputPath:        absInput,
		lastUsed:         now,
		lastRequest:      now,
		playlistFile:     ffmpegPlaylistFile,
		transcode:        transcode,
		highestRequested: -1,
	}

	if durations, err := s.playlistDurations(ctx, absInput); err == nil {
		if err := writeVODPlaylist(filepath.Join(dir, "playlist.m3u8"), durations); err != nil {
			_ = os.RemoveAll(dir)
			return nil, fmt.Errorf("playback: escribir la playlist VOD: %w", err)
		}
		entry.playlistFile = "playlist.m3u8"
		entry.segments = len(durations)
		s.entries[key] = entry
		return entry, nil
	}

	if err := s.startRunLocked(entry, 0); err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	s.entries[key] = entry
	if err := s.waitForPlaylist(ctx, entry); err != nil {
		if ctx.Err() != nil {
			// El cliente se fue, pero el proceso sigue y otra petición podrá
			// reutilizarlo.
			return nil, ctx.Err()
		}
		s.removeEntryLocked(key, entry)
		return nil, err
	}
	return entry, nil
}

// transcodeKey identifica los parámetros de calidad en la caché y en el
// nombre del directorio: un cambio de ajustes no mezcla segmentos de otra
// calidad.
func transcodeKey(settings TranscodeSettings) string {
	settings = normalizeTranscodeSettings(settings)
	return fmt.Sprintf("%d-%d-%s-%d", settings.MaxHeight, settings.CRF, settings.Preset, settings.Threads)
}

// startRunLocked detiene el proceso FFmpeg actual de la entrada (si lo hay) y
// lanza otro que genera desde el segmento start.
func (s *Service) startRunLocked(entry *hlsEntry, start int) error {
	if err := s.ffmpegAvailable(); err != nil {
		return err
	}
	if old := entry.run; old != nil {
		old.cancel()
		<-old.done
		s.recountLiveTranscodesLocked()
		entry.run = nil
		discardSegments(entry.dir, old.start, start)
	}

	s.reserveTranscodeSlotLocked(entry)

	// El proceso vive ligado al contexto del servicio, no al de la petición
	// HTTP que lo arranca, para que no se cancele al terminar esa petición.
	procCtx, cancel := context.WithTimeout(s.procCtx, hlsProcessTimeout)
	args := hlsTranscodeArgs(entry.inputPath, start, entry.transcode)
	var cmd *exec.Cmd
	if s.command != nil {
		cmd = s.command(procCtx, "ffmpeg", args...)
		if cmd == nil {
			cancel()
			return fmt.Errorf("%w: comando de prueba vacío", ErrFFmpegFailed)
		}
	} else {
		ffmpeg, err := exec.LookPath("ffmpeg")
		if err != nil {
			cancel()
			return ErrFFmpegNotFound
		}
		cmd = ffmpegCommand(procCtx, ffmpeg, args...)
	}
	cmd.Dir = entry.dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("%w: %v", ErrFFmpegFailed, err)
	}

	run := &hlsRun{
		start:   start,
		started: time.Now(),
		done:    make(chan struct{}),
		cancel:  cancel,
		counted: true,
	}
	entry.run = run
	s.liveTranscodes++

	go func() {
		err := cmd.Wait()
		// Un proceso detenido por un salto o por el cierre del servicio no es
		// un fallo.
		if err != nil && procCtx.Err() == nil {
			run.err = fmt.Errorf("%w: %v: %s", ErrFFmpegFailed, err, tailString(stderr.String(), 500))
		}
		cancel()
		close(run.done)
		// El canal se cierra antes de pedir el mutex para que quien espera la
		// salida no se bloquee con el contador.
		s.mu.Lock()
		if run.counted {
			run.counted = false
			s.liveTranscodes--
			if s.liveTranscodes < 0 {
				s.liveTranscodes = 0
			}
		}
		s.mu.Unlock()
	}()
	return nil
}

// reserveTranscodeSlotLocked deja sitio para un proceso nuevo. Si se alcanzó
// el tope, para el proceso ajeno que lleve más tiempo sin recibir peticiones;
// la entrada que se está lanzando nunca es la víctima.
func (s *Service) reserveTranscodeSlotLocked(entry *hlsEntry) {
	s.recountLiveTranscodesLocked()
	limit := s.maxTranscodeLimit()
	for s.liveTranscodes >= limit {
		victim := s.oldestLiveRunLocked(entry)
		if victim == nil {
			return
		}
		s.stopRunLocked(victim, "límite de concurrencia")
	}
}

func (s *Service) oldestLiveRunLocked(except *hlsEntry) *hlsEntry {
	var victim *hlsEntry
	var victimTime time.Time
	for _, candidate := range s.entries {
		if candidate == except || candidate.run == nil || candidate.run.finished() {
			continue
		}
		requested := candidate.lastRequest
		if requested.IsZero() {
			requested = candidate.lastUsed
		}
		if victim == nil || requested.Before(victimTime) {
			victim = candidate
			victimTime = requested
		}
	}
	return victim
}

// stopRunLocked para un proceso deliberadamente y conserva sus segmentos para
// que la próxima petición pueda reanudar por el punto que pida el cliente.
func (s *Service) stopRunLocked(entry *hlsEntry, reason string) {
	if entry == nil || entry.run == nil || entry.run.finished() {
		s.recountLiveTranscodesLocked()
		return
	}
	entry.run.interrupted = true
	entry.run.cancel()
	<-entry.run.done
	s.recountLiveTranscodesLocked()
	log.Printf("playback: detenido ffmpeg del ítem %d por %s", entry.itemID, reason)
}

// recountLiveTranscodesLocked reconcilia el contador con los procesos que
// siguen vivos. Los procesos terminados dejan de sumar aunque su goroutine
// todavía no haya podido tomar el mutex.
func (s *Service) recountLiveTranscodesLocked() {
	count := 0
	for _, entry := range s.entries {
		if entry.run == nil {
			continue
		}
		if entry.run.finished() {
			entry.run.counted = false
			continue
		}
		count++
		entry.run.counted = true
	}
	s.liveTranscodes = count
}

// discardSegments se llama con el proceso anterior ya terminado. Borra el
// segmento que dejó a medias (los suyos que no llegaron a su playlist) y todos
// los que el nuevo proceso, que empieza en newStart, va a volver a escribir:
// así cualquier segmento en disco anterior a newStart está completo, y los
// posteriores solo se dan por buenos cuando el nuevo proceso los lista.
func discardSegments(dir string, oldStart, newStart int) {
	last := lastListedSegment(dir)
	matches, _ := filepath.Glob(filepath.Join(dir, "segment_*.ts"))
	for _, match := range matches {
		index, ok := segmentIndex(filepath.Base(match))
		if !ok {
			continue
		}
		if index >= newStart || (index >= oldStart && index > last) {
			_ = os.Remove(match)
		}
	}
	_ = os.Remove(filepath.Join(dir, ffmpegPlaylistFile))
}

// waitForSegment espera a que el segmento esté completo. Si FFmpeg no va a
// llegar pronto a él (salto adelante lejano, o salto atrás a un hueco), lo
// relanza en ese segmento, como hace Jellyfin.
func (s *Service) waitForSegment(ctx context.Context, entry *hlsEntry, index int) error {
	ctx, cancel := context.WithTimeout(ctx, hlsSegmentWaitTimeout)
	defer cancel()
	ticker := time.NewTicker(hlsSegmentPoll)
	defer ticker.Stop()

	for {
		s.mu.Lock()
		now := s.now()
		entry.lastUsed = now
		entry.lastRequest = now
		if index > entry.highestRequested {
			entry.highestRequested = index
		}
		ready, err := s.checkSegmentLocked(entry, index)
		s.mu.Unlock()
		if err != nil {
			return err
		}
		if ready {
			return nil
		}
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("%w: segmento %d", ErrHLSSegmentTimeout, index)
			}
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// checkSegmentLocked indica si el segmento se puede servir ya. Si no, y
// esperar no tiene sentido, relanza FFmpeg en él o devuelve un error.
func (s *Service) checkSegmentLocked(entry *hlsEntry, index int) (bool, error) {
	if entry.removed {
		return false, fmt.Errorf("%w: segmento %d", ErrFileNotFound, index)
	}

	run := entry.run
	exists := fileNotEmpty(filepath.Join(entry.dir, segmentName(index)))
	inRun := run != nil && index >= run.start
	last := -1
	if inRun {
		last = lastListedSegment(entry.dir)
		if last < run.start-1 {
			last = run.start - 1
		}
	}

	if exists && (!inRun || index <= last) {
		return true, nil
	}

	if inRun {
		if !run.finished() {
			if index <= last+1+hlsRestartGap {
				return false, nil
			}
		} else if !run.interrupted && (run.start == index || entry.segments == 0) {
			// El proceso empezó justo aquí (o no se puede relanzar) y terminó
			// sin generarlo: relanzarlo otra vez no lo arreglaría. Si lo
			// paramos nosotros por inactividad, adelanto o concurrencia, sí
			// hay que reanudar.
			if run.err != nil {
				return false, run.err
			}
			return false, fmt.Errorf("%w: segmento %d", ErrFileNotFound, index)
		}
	}

	if entry.segments == 0 {
		// Sin playlist VOD propia no hay forma de calcular el final, pero si
		// paramos el proceso podemos relanzarlo en el segmento pedido.
		if run != nil && !run.finished() {
			return false, nil
		}
		if run == nil || !run.interrupted {
			return false, fmt.Errorf("%w: segmento %d", ErrFileNotFound, index)
		}
	}
	if entry.segments > 0 && index >= entry.segments {
		return false, fmt.Errorf("%w: segmento %d", ErrFileNotFound, index)
	}
	if run != nil && !run.finished() && time.Since(run.started) < hlsRestartMinAge {
		return false, nil
	}
	if err := s.startRunLocked(entry, index); err != nil {
		return false, err
	}
	return false, nil
}

// lastListedSegment devuelve el mayor índice de segmento que lista la playlist
// del proceso FFmpeg actual, o -1 si aún no lista ninguno.
func lastListedSegment(dir string) int {
	file, err := os.Open(filepath.Join(dir, ffmpegPlaylistFile))
	if err != nil {
		return -1
	}
	defer file.Close()

	last := -1
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if index, ok := segmentIndex(strings.TrimSpace(scanner.Text())); ok && index > last {
			last = index
		}
	}
	return last
}

func fileNotEmpty(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Size() > 0
}

// playlistDurations calcula los segmentos esperados: bloques de
// hlsSegmentSeconds, que es donde FFmpeg fuerza los keyframes al recodificar.
func (s *Service) playlistDurations(ctx context.Context, path string) ([]time.Duration, error) {
	if s.prober == nil {
		return nil, ErrDurationUnknown
	}
	duration, err := s.prober.VideoDuration(ctx, path)
	if err != nil || duration <= 0 {
		return nil, ErrDurationUnknown
	}
	return uniformSegmentDurations(duration), nil
}

func uniformSegmentDurations(duration time.Duration) []time.Duration {
	segment := time.Duration(hlsSegmentSeconds) * time.Second
	var durations []time.Duration
	for remaining := duration; remaining > 0; {
		take := segment
		if remaining < take {
			take = remaining
		}
		// Un resto de menos de un segundo (la pista de audio suele durar unos
		// milisegundos más que el vídeo) va al segmento anterior: FFmpeg puede
		// no llegar a generarlo y el cliente fallaría al final del vídeo.
		if take < time.Second && len(durations) > 0 {
			durations[len(durations)-1] += take
			break
		}
		durations = append(durations, take)
		remaining -= take
	}
	return durations
}

// writeVODPlaylist escribe la playlist VOD completa que se sirve al cliente.
func writeVODPlaylist(path string, durations []time.Duration) error {
	var maxDuration time.Duration
	for _, duration := range durations {
		if duration > maxDuration {
			maxDuration = duration
		}
	}
	target := int(math.Ceil(maxDuration.Seconds()))
	if target < 1 {
		target = 1
	}

	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	b.WriteString("#EXT-X-VERSION:3\n")
	fmt.Fprintf(&b, "#EXT-X-TARGETDURATION:%d\n", target)
	b.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")
	b.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")
	for index, duration := range durations {
		fmt.Fprintf(&b, "#EXTINF:%.6f,\n%s\n", duration.Seconds(), segmentName(index))
	}
	b.WriteString("#EXT-X-ENDLIST\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func (s *Service) playlistExists(entry *hlsEntry) bool {
	_, err := os.Stat(filepath.Join(entry.dir, entry.playlistFile))
	return err == nil
}

// waitForPlaylist espera a que FFmpeg escriba su playlist, que aparece tras el
// primer segmento. Solo se usa cuando no hay playlist VOD propia.
func (s *Service) waitForPlaylist(ctx context.Context, entry *hlsEntry) error {
	run := entry.run
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
		case <-run.done:
			if _, err := os.Stat(playlist); err == nil {
				return nil
			}
			if run.err != nil {
				return run.err
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

// removeEntryLocked detiene el proceso de la entrada y borra sus archivos.
func (s *Service) removeEntryLocked(key hlsEntryKey, entry *hlsEntry) {
	entry.removed = true
	if entry.run != nil {
		entry.run.cancel()
		<-entry.run.done
		s.recountLiveTranscodesLocked()
	}
	if s.entries[key] == entry {
		delete(s.entries, key)
	}
	_ = os.RemoveAll(entry.dir)
}

func segmentName(index int) string {
	return fmt.Sprintf("segment_%05d.ts", index)
}

func segmentIndex(name string) (int, bool) {
	if !strings.HasPrefix(name, "segment_") || !strings.HasSuffix(name, ".ts") {
		return 0, false
	}
	index, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(name, "segment_"), ".ts"))
	if err != nil || index < 0 {
		return 0, false
	}
	return index, true
}

func (s *Service) cleanupLocked() {
	now := s.now()
	for key, entry := range s.entries {
		if now.Sub(entry.lastUsed) > hlsIdleTTL {
			s.removeEntryLocked(key, entry)
		}
	}
}

// monitorLocked para los procesos que ya no tienen sentido: los abandonados
// sin peticiones y los que se han adelantado demasiado al cliente.
func (s *Service) monitorLocked() {
	if s.procCtx != nil && s.procCtx.Err() != nil {
		return
	}
	s.recountLiveTranscodesLocked()
	now := s.now()
	for _, entry := range s.entries {
		run := entry.run
		if run == nil || run.finished() {
			continue
		}
		requested := entry.lastRequest
		if requested.IsZero() {
			requested = entry.lastUsed
		}
		if s.runIdleTimeout > 0 && !requested.IsZero() && now.Sub(requested) > s.runIdleTimeout {
			s.stopRunLocked(entry, "inactividad")
			continue
		}
		if entry.highestRequested < 0 {
			continue
		}
		if s.maxAheadSegments > 0 &&
			highestGeneratedSegment(entry.dir) > entry.highestRequested+s.maxAheadSegments {
			s.stopRunLocked(entry, "adelanto")
		}
	}
}

// highestGeneratedSegment devuelve el mayor segmento que FFmpeg ha empezado a
// escribir, aunque todavía no esté listado en su playlist.
func highestGeneratedSegment(dir string) int {
	matches, _ := filepath.Glob(filepath.Join(dir, "segment_*.ts"))
	highest := -1
	for _, match := range matches {
		index, ok := segmentIndex(filepath.Base(match))
		if ok && index > highest {
			highest = index
		}
	}
	return highest
}

// signPlaylist añade a los segmentos (y a un posible EXT-X-MAP) los mismos
// parámetros de firma que la playlist, para que el cliente pueda pedirlos.
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
