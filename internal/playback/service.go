package playback

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// DefaultURLTTL es la vida por defecto de una URL de reproducción firmada. Los
// segmentos HLS y los saltos en reproducción directa reutilizan la firma de la
// URL inicial, así que debe cubrir una película entera con pausas: con 15
// minutos la reproducción se cortaba a mitad.
const DefaultURLTTL = 8 * time.Hour

// hlsIdleTTL es el tiempo que se conserva un HLS ya generado sin usarse.
const hlsIdleTTL = time.Hour

// defaultMaxTranscodes es el número de procesos FFmpeg que pueden convivir si
// no se configura otro límite.
const defaultMaxTranscodes = 2

// TranscodeSettings son los parámetros que afectan al remux HLS.
type TranscodeSettings struct {
	MaxHeight int
	CRF       int
	Preset    string
	// Threads limita los hilos de x264. Cero significa usar el valor por
	// defecto, calculado a partir de los núcleos disponibles.
	Threads int
}

func defaultTranscodeSettings() TranscodeSettings {
	return TranscodeSettings{
		MaxHeight: 1080,
		CRF:       20,
		Preset:    "veryfast",
		Threads:   defaultTranscodeThreads(),
	}
}

// defaultTranscodeThreads deja la mitad de los núcleos para el servidor y el
// resto de procesos del sistema.
func defaultTranscodeThreads() int {
	threads := runtime.NumCPU() / 2
	if threads < 1 {
		threads = 1
	}
	return threads
}

func normalizeTranscodeSettings(settings TranscodeSettings) TranscodeSettings {
	defaults := defaultTranscodeSettings()
	switch settings.MaxHeight {
	case 480, 720, 1080:
	default:
		settings.MaxHeight = defaults.MaxHeight
	}
	if settings.CRF < 18 || settings.CRF > 28 {
		settings.CRF = defaults.CRF
	}
	switch settings.Preset {
	case "veryfast", "faster", "fast":
	default:
		settings.Preset = defaults.Preset
	}
	if settings.Threads <= 0 {
		settings.Threads = defaults.Threads
	}
	return settings
}

// Options configura el servicio de reproducción.
type Options struct {
	// TTL de las URLs firmadas; por defecto, 15 minutos.
	TTL time.Duration
	// Now permite sustituir el reloj en las pruebas.
	Now func() time.Time
	// WorkDir es el directorio de trabajo de los remux; si está vacío se crea
	// uno temporal. El servicio lo elimina al cerrarse.
	WorkDir string
	// Key es la clave HMAC; si está vacía se genera una aleatoria por proceso.
	Key []byte
	// Prober sustituye el detector de keyframes; si es nil se usa el de por
	// defecto (ffprobe).
	Prober *Prober
	// Settings devuelve los parámetros de transcodificación vigentes. Si es
	// nil se usan 1080p, CRF 20 y veryfast.
	Settings func() TranscodeSettings
	// Threads sobrescribe los hilos de x264; cero usa el valor por defecto.
	Threads int
	// MaxTranscodes es el máximo de procesos FFmpeg vivos. Cero usa 2.
	MaxTranscodes int
	// MaxTranscodesFunc permite consultar el límite en caliente; tiene
	// prioridad sobre MaxTranscodes.
	MaxTranscodesFunc func() int
	// RunIdleTimeout es cuánto se espera sin peticiones antes de detener un
	// proceso. Cero usa hlsRunIdleTimeout.
	RunIdleTimeout time.Duration
	// MaxAheadSegments es cuántos segmentos puede adelantarse FFmpeg. Cero usa
	// hlsMaxAheadSegments.
	MaxAheadSegments int
	// MonitorInterval es la frecuencia del vigilante. Cero usa
	// hlsMonitorInterval.
	MonitorInterval time.Duration
	// Command sustituye la creación del proceso FFmpeg en las pruebas.
	Command func(ctx context.Context, name string, args ...string) *exec.Cmd
	// command es el alias que usan algunas pruebas internas.
	command func(ctx context.Context, name string, args ...string) *exec.Cmd
}

// Service firma URLs, prepara remux HLS y sirve sus archivos.
type Service struct {
	key      []byte
	ttl      time.Duration
	now      func() time.Time
	workDir  string
	prober   *Prober
	settings func() TranscodeSettings
	threads  int

	maxTranscodes     int
	maxTranscodesFunc func() int
	runIdleTimeout    time.Duration
	maxAheadSegments  int
	monitorInterval   time.Duration
	command           func(ctx context.Context, name string, args ...string) *exec.Cmd

	procCtx    context.Context
	procCancel context.CancelFunc

	mu             sync.Mutex
	entries        map[hlsEntryKey]*hlsEntry
	liveTranscodes int
	stop           chan struct{}
	done           chan struct{}
	closeOnce      sync.Once
}

type hlsEntryKey struct {
	itemID    int64
	transcode string
}

// hlsEntry es el HLS de un ítem. Sus campos se protegen con Service.mu.
type hlsEntry struct {
	itemID       int64
	dir          string
	inputPath    string
	lastUsed     time.Time
	lastRequest  time.Time
	playlistFile string
	transcode    TranscodeSettings
	// highestRequested es el mayor segmento que ha pedido el cliente.
	highestRequested int
	// segments es el total de la playlist VOD propia; 0 si se sirve la de
	// FFmpeg y por tanto no se puede relanzar en un segmento concreto.
	segments int
	// run es el proceso FFmpeg actual; nil hasta la primera petición de
	// segmento. Un salto lo sustituye por otro que empieza más adelante.
	run *hlsRun
	// removed indica que la entrada ya se borró y no debe relanzar procesos.
	removed bool
}

// hlsRun es un proceso FFmpeg que genera segmentos desde start.
type hlsRun struct {
	start   int
	started time.Time
	// done se cierra cuando termina el proceso. err se escribe antes de
	// cerrar done, así que leerlo después de recibir del canal es seguro.
	done   chan struct{}
	err    error
	cancel context.CancelFunc
	// interrupted marca las paradas deliberadas por inactividad, adelanto o
	// concurrencia: la próxima petición puede relanzar el proceso.
	interrupted bool
	// counted indica que este proceso suma en Service.liveTranscodes.
	counted bool
}

func (r *hlsRun) finished() bool {
	select {
	case <-r.done:
		return true
	default:
		return false
	}
}

// New crea el servicio de reproducción.
func New(opts Options) (*Service, error) {
	ttl := opts.TTL
	if ttl <= 0 {
		ttl = DefaultURLTTL
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}

	key := opts.Key
	if len(key) == 0 {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("playback: generar clave de firma: %w", err)
		}
	}

	workDir := opts.WorkDir
	if workDir == "" {
		dir, err := os.MkdirTemp("", "cinecat-hls-*")
		if err != nil {
			return nil, fmt.Errorf("playback: crear directorio temporal: %w", err)
		}
		workDir = dir
	} else if err := os.MkdirAll(workDir, 0o700); err != nil {
		return nil, fmt.Errorf("playback: crear directorio de trabajo: %w", err)
	}

	prober := opts.Prober
	if prober == nil {
		prober = NewProber()
	}
	settings := opts.Settings
	if settings == nil {
		settings = defaultTranscodeSettings
	}

	runIdleTimeout := opts.RunIdleTimeout
	if runIdleTimeout <= 0 {
		runIdleTimeout = hlsRunIdleTimeout
	}
	maxAheadSegments := opts.MaxAheadSegments
	if maxAheadSegments <= 0 {
		maxAheadSegments = hlsMaxAheadSegments
	}
	monitorInterval := opts.MonitorInterval
	if monitorInterval <= 0 {
		monitorInterval = hlsMonitorInterval
	}
	maxTranscodes := opts.MaxTranscodes
	if maxTranscodes <= 0 {
		maxTranscodes = defaultMaxTranscodes
	}
	threads := opts.Threads
	if threads < 0 {
		threads = 0
	}

	command := opts.Command
	if command == nil {
		command = opts.command
	}

	procCtx, procCancel := context.WithCancel(context.Background())
	svc := &Service{
		key:               key,
		ttl:               ttl,
		now:               now,
		workDir:           workDir,
		prober:            prober,
		settings:          settings,
		threads:           threads,
		maxTranscodes:     maxTranscodes,
		maxTranscodesFunc: opts.MaxTranscodesFunc,
		runIdleTimeout:    runIdleTimeout,
		maxAheadSegments:  maxAheadSegments,
		monitorInterval:   monitorInterval,
		command:           command,
		procCtx:           procCtx,
		procCancel:        procCancel,
		entries:           make(map[hlsEntryKey]*hlsEntry),
		stop:              make(chan struct{}),
		done:              make(chan struct{}),
	}
	go svc.cleanupLoop()
	return svc, nil
}

func (s *Service) currentTranscodeSettings() TranscodeSettings {
	original := s.settings()
	settings := normalizeTranscodeSettings(original)
	if s.threads > 0 {
		settings.Threads = s.threads
	} else if original.Threads <= 0 {
		// Sin límite explícito se deja sin fijar; los argumentos aplican el
		// valor por defecto calculado a partir de los núcleos.
		settings.Threads = original.Threads
	}
	return settings
}

// Close detiene la limpieza periódica y elimina el directorio de trabajo con
// los remux pendientes.
func (s *Service) Close() error {
	s.closeOnce.Do(func() {
		close(s.stop)
		s.procCancel()
	})
	<-s.done

	// Los procesos ya se cancelaron con procCtx; removeEntryLocked espera a
	// que terminen antes de borrar sus archivos.
	s.mu.Lock()
	for key, entry := range s.entries {
		s.removeEntryLocked(key, entry)
	}
	s.recountLiveTranscodesLocked()
	s.mu.Unlock()
	return os.RemoveAll(s.workDir)
}

// cleanupLoop borra los remux antiguos y vigila los procesos vivos: los para
// cuando llevan demasiado sin recibir peticiones o se adelantan al cliente.
func (s *Service) cleanupLoop() {
	defer close(s.done)

	cleanupTicker := time.NewTicker(15 * time.Minute)
	defer cleanupTicker.Stop()
	monitorInterval := s.monitorInterval
	if monitorInterval <= 0 {
		monitorInterval = hlsMonitorInterval
	}
	monitorTicker := time.NewTicker(monitorInterval)
	defer monitorTicker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-cleanupTicker.C:
			s.mu.Lock()
			s.cleanupLocked()
			s.mu.Unlock()
		case <-monitorTicker.C:
			s.mu.Lock()
			s.monitorLocked()
			s.mu.Unlock()
		}
	}
}

// ffmpegAvailable comprueba el binario real salvo que las pruebas hayan
// sustituido el comando.
func (s *Service) ffmpegAvailable() error {
	if s.command != nil {
		return nil
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return ErrFFmpegNotFound
	}
	return nil
}

func (s *Service) maxTranscodeLimit() int {
	if s.maxTranscodesFunc != nil {
		if limit := s.maxTranscodesFunc(); limit > 0 {
			return limit
		}
	}
	if s.maxTranscodes > 0 {
		return s.maxTranscodes
	}
	return defaultMaxTranscodes
}

// Sign devuelve la firma y la caducidad de una URL de reproducción para el
// ítem indicado.
func (s *Service) Sign(itemID int64) (string, time.Time) {
	expires := s.now().Add(s.ttl).UTC().Truncate(time.Second)
	return s.signature(itemID, expires.Unix()), expires
}

// Verify comprueba que la firma corresponde al ítem y que la URL no ha
// caducado.
func (s *Service) Verify(itemID int64, expRaw, signature string) error {
	expires, err := strconv.ParseInt(expRaw, 10, 64)
	if err != nil {
		return ErrInvalidSignature
	}
	if s.now().Unix() >= expires {
		return ErrExpired
	}

	got, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return ErrInvalidSignature
	}
	expected := s.signatureBytes(itemID, expires)
	if !hmac.Equal(got, expected) {
		return ErrInvalidSignature
	}
	return nil
}

func (s *Service) signature(itemID, expires int64) string {
	return base64.RawURLEncoding.EncodeToString(s.signatureBytes(itemID, expires))
}

func (s *Service) signatureBytes(itemID, expires int64) []byte {
	mac := hmac.New(sha256.New, s.key)
	// Los dos enteros van en binario para que la firma no dependa de cómo se
	// formateen como texto.
	var message [16]byte
	binary.BigEndian.PutUint64(message[:8], uint64(itemID))
	binary.BigEndian.PutUint64(message[8:], uint64(expires))
	mac.Write(message[:])
	return mac.Sum(nil)
}

// CheckFFmpeg comprueba que ffmpeg existe en el PATH.
func (s *Service) CheckFFmpeg() error {
	return s.ffmpegAvailable()
}
