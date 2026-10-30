package library

import (
	"context"
	"database/sql"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	defaultWatchReload = 10 * time.Minute
	defaultWatchTick   = 5 * time.Second
	watchDBTimeout     = 5 * time.Second
)

// LibraryWatcher vigila las carpetas de las bibliotecas y pide un escaneo
// cuando dejan de llegar cambios durante un rato, con un máximo de espera.
type LibraryWatcher struct {
	db      *sql.DB
	job     *ScanJob
	quiet   time.Duration
	maxWait time.Duration

	tick        time.Duration
	reloadEvery time.Duration
	watched     map[string]bool
	ready       chan struct{}
	readyOnce   sync.Once
}

// NewLibraryWatcher crea el vigilante de las carpetas de todas las
// bibliotecas. Run se encarga de cargarlas y de recargarlas periódicamente.
func NewLibraryWatcher(db *sql.DB, job *ScanJob, quiet, maxWait time.Duration) *LibraryWatcher {
	return &LibraryWatcher{
		db:          db,
		job:         job,
		quiet:       quiet,
		maxWait:     maxWait,
		tick:        defaultWatchTick,
		reloadEvery: defaultWatchReload,
		watched:     make(map[string]bool),
		ready:       make(chan struct{}),
	}
}

// Run bloquea hasta que se cancele ctx. Si fsnotify no puede vigilar alguna
// carpeta, avisa una vez y mantiene el escaneo periódico normal.
func (w *LibraryWatcher) Run(ctx context.Context) {
	if w == nil {
		return
	}
	defer w.signalReady()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("watcher: no se pudo iniciar la vigilancia de carpetas: %v; se mantiene el escaneo periódico", err)
		return
	}
	defer watcher.Close()

	var watchErrOnce sync.Once
	w.reload(ctx, watcher, &watchErrOnce)
	w.signalReady()

	tickDuration := w.tick
	if tickDuration <= 0 {
		tickDuration = defaultWatchTick
	}
	reloadDuration := w.reloadEvery
	if reloadDuration <= 0 {
		reloadDuration = defaultWatchReload
	}
	tick := time.NewTicker(tickDuration)
	defer tick.Stop()
	reloadTick := time.NewTicker(reloadDuration)
	defer reloadTick.Stop()

	pending := false
	var firstPending, lastEvent time.Time
	lastReload := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			name := filepath.Base(event.Name)
			if event.Op == fsnotify.Chmod || strings.HasPrefix(name, ".") ||
				junkFolders[strings.ToLower(name)] {
				continue
			}
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					w.addTree(watcher, event.Name, &watchErrOnce)
				}
			}
			now := time.Now()
			lastEvent = now
			if !pending {
				pending = true
				firstPending = now
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			watchErrOnce.Do(func() {
				log.Printf("watcher: error de fsnotify (%v); se sigue con las carpetas vigiladas y el escaneo periódico", err)
			})
		case now := <-tick.C:
			if ctx.Err() != nil {
				return
			}
			if pending && (now.Sub(lastEvent) >= w.quiet || now.Sub(firstPending) >= w.maxWait) {
				if w.job != nil && w.job.StartService() {
					pending = false
					w.reload(ctx, watcher, &watchErrOnce)
					lastReload = now
				}
			}
		case now := <-reloadTick.C:
			if ctx.Err() != nil {
				return
			}
			if now.Sub(lastReload) >= reloadDuration {
				w.reload(ctx, watcher, &watchErrOnce)
				lastReload = now
			}
		}
	}
}

func (w *LibraryWatcher) signalReady() {
	if w.ready == nil {
		return
	}
	w.readyOnce.Do(func() { close(w.ready) })
}

func (w *LibraryWatcher) reload(ctx context.Context, watcher *fsnotify.Watcher, watchErrOnce *sync.Once) {
	if w.db == nil || ctx.Err() != nil {
		return
	}
	// La consulta no se cancela con el apagado para no dejar una conexión
	// bloqueada a medio cerrar; el tiempo máximo evita esperas eternas.
	queryCtx, cancelQuery := context.WithTimeout(context.WithoutCancel(ctx), watchDBTimeout)
	defer cancelQuery()
	type watchedLibrary struct {
		id       int64
		fallback string
	}
	rows, err := w.db.QueryContext(queryCtx, `SELECT id, path FROM libraries ORDER BY id`)
	if err != nil {
		log.Printf("watcher: no se pudieron leer las bibliotecas: %v", err)
		return
	}
	libraries := make([]watchedLibrary, 0)
	for rows.Next() {
		var library watchedLibrary
		if err := rows.Scan(&library.id, &library.fallback); err != nil {
			rows.Close()
			log.Printf("watcher: no se pudo leer una biblioteca: %v", err)
			return
		}
		libraries = append(libraries, library)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		log.Printf("watcher: no se pudieron recorrer las bibliotecas: %v", err)
		return
	}
	if err := rows.Close(); err != nil {
		log.Printf("watcher: no se pudieron cerrar las bibliotecas: %v", err)
	}

	for _, library := range libraries {
		paths, err := loadLibraryPathStrings(queryCtx, w.db, library.id, library.fallback)
		if err != nil {
			log.Printf("watcher: no se pudieron leer las carpetas de la biblioteca %d: %v", library.id, err)
			continue
		}
		for _, root := range paths {
			w.addTree(watcher, root, watchErrOnce)
		}
	}
}

func (w *LibraryWatcher) addTree(watcher *fsnotify.Watcher, root string, watchErrOnce *sync.Once) {
	info, err := os.Lstat(root)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return
	}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			watchErrOnce.Do(func() {
				log.Printf("watcher: no se pudo recorrer %s (%v); se sigue con las carpetas vigiladas", path, walkErr)
			})
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if path != root {
			name := entry.Name()
			if strings.HasPrefix(name, ".") || junkFolders[strings.ToLower(name)] {
				return fs.SkipDir
			}
		}
		w.addWatch(watcher, path, watchErrOnce)
		return nil
	})
	if err != nil {
		watchErrOnce.Do(func() {
			log.Printf("watcher: no se pudo recorrer %s (%v); se sigue con las carpetas vigiladas", root, err)
		})
	}
}

func (w *LibraryWatcher) addWatch(watcher *fsnotify.Watcher, path string, watchErrOnce *sync.Once) {
	path = filepath.Clean(path)
	if w.watched[path] {
		return
	}
	if err := watcher.Add(path); err != nil {
		watchErrOnce.Do(func() {
			log.Printf("watcher: no se pudo vigilar %s (%v); se sigue con las carpetas vigiladas y el escaneo periódico", path, err)
		})
		return
	}
	w.watched[path] = true
}
