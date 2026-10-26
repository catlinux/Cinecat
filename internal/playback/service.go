package playback

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

// DefaultURLTTL es la vida por defecto de una URL de reproducción firmada.
const DefaultURLTTL = 15 * time.Minute

// hlsIdleTTL es el tiempo que se conserva un remux ya generado sin usarse.
const hlsIdleTTL = time.Hour

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
}

// Service firma URLs, prepara remux HLS y sirve sus archivos.
type Service struct {
	key     []byte
	ttl     time.Duration
	now     func() time.Time
	workDir string
	prober  *Prober

	mu        sync.Mutex
	entries   map[int64]*hlsEntry
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

type hlsEntry struct {
	dir      string
	lastUsed time.Time
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

	svc := &Service{
		key:     key,
		ttl:     ttl,
		now:     now,
		workDir: workDir,
		prober:  prober,
		entries: make(map[int64]*hlsEntry),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	go svc.cleanupLoop()
	return svc, nil
}

// Close detiene la limpieza periódica y elimina el directorio de trabajo con
// los remux pendientes.
func (s *Service) Close() error {
	s.closeOnce.Do(func() { close(s.stop) })
	<-s.done

	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = make(map[int64]*hlsEntry)
	return os.RemoveAll(s.workDir)
}

// cleanupLoop borra los remux que llevan un rato sin usarse, aunque no lleguen
// más peticiones.
func (s *Service) cleanupLoop() {
	defer close(s.done)

	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.mu.Lock()
			s.cleanupLocked()
			s.mu.Unlock()
		}
	}
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
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return ErrFFmpegNotFound
	}
	return nil
}
