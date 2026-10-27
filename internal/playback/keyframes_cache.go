package playback

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
)

const keyframeCacheVersion = 1

// ErrKeyframesPending indica que el índice se está calculando en segundo
// plano y todavía no está disponible.
var ErrKeyframesPending = errors.New("playback: índice de fotogramas clave pendiente")

// KeyframeEntry es el índice persistido de un archivo.
type KeyframeEntry struct {
	Version   int       `json:"version"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	MTime     int64     `json:"mtime"`
	Source    string    `json:"source"`
	Keyframes []float64 `json:"keyframes"`
}

type keyframeJob struct {
	key  string
	path string
}

// KeyframeIndex cachea los índices de claves en disco y calcula los que
// faltan con ffprobe en una única tarea de fondo.
type KeyframeIndex struct {
	dir string

	mu      sync.Mutex
	entries map[string]*KeyframeEntry
	pending map[string]bool
	failed  map[string]bool
	queue   []keyframeJob
	wake    chan struct{}

	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
	closeOnce sync.Once
	probe     func(context.Context, string) ([]float64, error)
}

// NewKeyframeIndex crea una caché de claves. Con un directorio vacío solo usa
// memoria.
func NewKeyframeIndex(dir string) *KeyframeIndex {
	return NewKeyframeIndexWithContext(context.Background(), dir)
}

// NewKeyframeIndexWithContext permite detener la tarea de fondo con el
// contexto del servicio.
func NewKeyframeIndexWithContext(ctx context.Context, dir string) *KeyframeIndex {
	if ctx == nil {
		ctx = context.Background()
	}
	workerCtx, cancel := context.WithCancel(ctx)
	index := &KeyframeIndex{
		dir:     dir,
		entries: make(map[string]*KeyframeEntry),
		pending: make(map[string]bool),
		failed:  make(map[string]bool),
		wake:    make(chan struct{}, 1),
		ctx:     workerCtx,
		cancel:  cancel,
		done:    make(chan struct{}),
		probe:   ProbeKeyframes,
	}
	go index.worker()
	return index
}

// Close detiene la tarea de fondo y espera a que termine.
func (k *KeyframeIndex) Close() error {
	k.closeOnce.Do(k.cancel)
	<-k.done
	return nil
}

// Lookup devuelve el índice si está en memoria, en disco o en los Cues del
// propio Matroska. Si hay que calcularlo con ffprobe, lo encola y devuelve
// ErrKeyframesPending.
func (k *KeyframeIndex) Lookup(path string) (*KeyframeEntry, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("playback: ruta de vídeo no válida: %w", err)
	}
	stat, err := os.Stat(absolute)
	if err != nil {
		return nil, err
	}
	entryInfo := keyframeFileInfo{
		path:  absolute,
		size:  stat.Size(),
		mtime: stat.ModTime().UnixNano(),
	}
	key := keyframeCacheKey(entryInfo)

	k.mu.Lock()
	if entry := k.entries[key]; entry != nil {
		copyEntry := cloneKeyframeEntry(entry)
		k.mu.Unlock()
		return copyEntry, nil
	}
	if k.pending[key] {
		k.mu.Unlock()
		return nil, ErrKeyframesPending
	}
	if k.failed[key] {
		k.mu.Unlock()
		return nil, ErrNoKeyframeIndex
	}
	k.mu.Unlock()

	if entry := k.loadDisk(key, entryInfo); entry != nil {
		k.mu.Lock()
		k.entries[key] = entry
		k.mu.Unlock()
		return cloneKeyframeEntry(entry), nil
	}

	keyframes, err := MatroskaKeyframes(absolute)
	if err == nil {
		keyframes = normalizeKeyframes(keyframes)
		if validKeyframes(keyframes) {
			entry := &KeyframeEntry{
				Version:   keyframeCacheVersion,
				Path:      absolute,
				Size:      entryInfo.size,
				MTime:     entryInfo.mtime,
				Source:    "cues",
				Keyframes: keyframes,
			}
			k.store(entry, key)
			return cloneKeyframeEntry(entry), nil
		}
	}

	k.enqueue(key, absolute)
	return nil, ErrKeyframesPending
}

type keyframeFileInfo struct {
	path  string
	size  int64
	mtime int64
}

func keyframeCacheKey(info keyframeFileInfo) string {
	sum := sha256.Sum256([]byte(info.path + "\x00" +
		strconv.FormatInt(info.size, 10) + "\x00" +
		strconv.FormatInt(info.mtime, 10)))
	return fmt.Sprintf("%x", sum[:])
}

func (k *KeyframeIndex) enqueue(key, path string) {
	k.mu.Lock()
	if k.pending[key] || k.failed[key] || k.entries[key] != nil {
		k.mu.Unlock()
		return
	}
	k.pending[key] = true
	k.queue = append(k.queue, keyframeJob{key: key, path: path})
	k.mu.Unlock()
	select {
	case k.wake <- struct{}{}:
	default:
	}
}

func (k *KeyframeIndex) store(entry *KeyframeEntry, key string) {
	k.mu.Lock()
	k.entries[key] = cloneKeyframeEntry(entry)
	delete(k.pending, key)
	delete(k.failed, key)
	k.mu.Unlock()
	if err := k.saveDisk(key, entry); err != nil {
		// La caché en disco es opcional; el índice en memoria sigue sirviendo.
	}
}

func (k *KeyframeIndex) worker() {
	defer close(k.done)
	for {
		select {
		case <-k.ctx.Done():
			return
		case <-k.wake:
		}
		for {
			k.mu.Lock()
			if len(k.queue) == 0 {
				k.mu.Unlock()
				break
			}
			job := k.queue[0]
			k.queue = k.queue[1:]
			k.mu.Unlock()

			keyframes, err := k.probe(k.ctx, job.path)
			if err == nil {
				keyframes = normalizeKeyframes(keyframes)
			}
			if err == nil && validKeyframes(keyframes) {
				stat, statErr := os.Stat(job.path)
				if statErr == nil {
					k.store(&KeyframeEntry{
						Version:   keyframeCacheVersion,
						Path:      job.path,
						Size:      stat.Size(),
						MTime:     stat.ModTime().UnixNano(),
						Source:    "ffprobe",
						Keyframes: keyframes,
					}, job.key)
				} else {
					k.markFailed(job.key)
				}
			} else {
				k.markFailed(job.key)
			}
			if k.ctx.Err() != nil {
				return
			}
		}
	}
}

func (k *KeyframeIndex) markFailed(key string) {
	k.mu.Lock()
	delete(k.pending, key)
	k.failed[key] = true
	k.mu.Unlock()
}

func cloneKeyframeEntry(entry *KeyframeEntry) *KeyframeEntry {
	if entry == nil {
		return nil
	}
	copyEntry := *entry
	copyEntry.Keyframes = append([]float64(nil), entry.Keyframes...)
	return &copyEntry
}

func normalizeKeyframes(values []float64) []float64 {
	result := append([]float64(nil), values...)
	sort.Float64s(result)
	return uniqueKeyframes(result)
}

func validKeyframes(values []float64) bool {
	return len(values) >= 2 && values[0] <= 1
}

func (k *KeyframeIndex) cachePath(key string) string {
	if k.dir == "" {
		return ""
	}
	return filepath.Join(k.dir, key+".json")
}

func (k *KeyframeIndex) saveDisk(key string, entry *KeyframeEntry) error {
	path := k.cachePath(key)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(k.dir, 0o750); err != nil {
		return err
	}
	temp, err := os.CreateTemp(k.dir, ".keyframes-*.json")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tempPath)
		}
	}()
	encoder := json.NewEncoder(temp)
	if err := encoder.Encode(entry); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tempPath, 0o640); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	success = true
	return nil
}

func (k *KeyframeIndex) loadDisk(key string, info keyframeFileInfo) *KeyframeEntry {
	path := k.cachePath(key)
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var entry KeyframeEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil
	}
	if entry.Version != keyframeCacheVersion || entry.Path != info.path ||
		entry.Size != info.size || entry.MTime != info.mtime ||
		(entry.Source != "cues" && entry.Source != "ffprobe") {
		return nil
	}
	entry.Keyframes = normalizeKeyframes(entry.Keyframes)
	if !validKeyframes(entry.Keyframes) {
		return nil
	}
	return cloneKeyframeEntry(&entry)
}
