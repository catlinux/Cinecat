// Package backup crea copias consistentes de la base SQLite y gestiona su
// retención. Las copias viven junto a la base de datos, en una subcarpeta
// backups, para que acompañen al volumen de Docker.
package backup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultRetentionDays = 7
	backupPrefix         = "cinecat-"
	backupSuffix         = ".db"
	backupTimeLayout     = "20060102-150405"
)

// ErrBusy indica que ya hay una copia en curso.
var ErrBusy = errors.New("backup: ya hay una copia en curso")

// Copy describe una copia de seguridad sin exponer su ruta en disco.
type Copy struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
	CreatedAt string `json:"created_at"`
}

type fileCopy struct {
	Copy
	created time.Time
}

// Manager crea y enumera las copias de una base de datos SQLite.
type Manager struct {
	db            *sql.DB
	dir           string
	now           func() time.Time
	enabled       func() bool
	retentionDays func() int

	runMu   sync.Mutex
	stateMu sync.Mutex
	lastErr string
}

// New crea el gestor de copias para la base indicada. El directorio se deriva
// de la ruta del archivo y se intenta crear, pero un error de permisos no
// impide arrancar: queda en last_error y se reintenta en cada copia.
func New(db *sql.DB, databasePath string) *Manager {
	manager := &Manager{
		db:            db,
		dir:           backupDir(databasePath),
		now:           time.Now,
		enabled:       func() bool { return true },
		retentionDays: func() int { return defaultRetentionDays },
	}
	if manager.dir != "" {
		if err := manager.ensureDir(); err != nil {
			manager.setLastError(err)
			log.Printf("backup: %v", err)
		}
	}
	return manager
}

// WithClock sustituye el reloj, útil para calcular la retención en pruebas.
func (m *Manager) WithClock(now func() time.Time) *Manager {
	if m != nil && now != nil {
		m.now = now
	}
	return m
}

// WithSettings conecta los ajustes que controlan las copias automáticas.
func (m *Manager) WithSettings(enabled func() bool, retentionDays func() int) *Manager {
	if m == nil {
		return m
	}
	if enabled != nil {
		m.enabled = enabled
	}
	if retentionDays != nil {
		m.retentionDays = retentionDays
	}
	return m
}

// Dir devuelve el directorio de copias. No se incluye en las respuestas HTTP.
func (m *Manager) Dir() string {
	if m == nil {
		return ""
	}
	return m.dir
}

// Enabled indica si las copias automáticas están activadas.
func (m *Manager) Enabled() bool {
	return m != nil && m.enabled != nil && m.enabled()
}

// RetentionDays devuelve los días de retención efectivos.
func (m *Manager) RetentionDays() int {
	if m == nil || m.retentionDays == nil {
		return defaultRetentionDays
	}
	days := m.retentionDays()
	if days < 1 {
		return defaultRetentionDays
	}
	return days
}

// LastError devuelve el último error de copia o retención, si lo hubo.
func (m *Manager) LastError() string {
	if m == nil {
		return ""
	}
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	return m.lastErr
}

func (m *Manager) setLastError(err error) {
	if m == nil {
		return
	}
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	if err == nil {
		m.lastErr = ""
		return
	}
	message := err.Error()
	if m.dir != "" {
		// last_error se publica por la API: nunca debe contener la ruta
		// absoluta del volumen.
		message = strings.ReplaceAll(message, m.dir, "backups")
	}
	m.lastErr = message
}

// List devuelve las copias disponibles, de la más reciente a la más antigua.
func (m *Manager) List() ([]Copy, error) {
	files, err := m.listFiles()
	if err != nil {
		return nil, err
	}
	copies := make([]Copy, 0, len(files))
	for _, file := range files {
		copies = append(copies, file.Copy)
	}
	return copies, nil
}

// Create hace una copia ahora. Si ya hay otra en curso devuelve ErrBusy.
func (m *Manager) Create(ctx context.Context) (Copy, error) {
	if m == nil || m.db == nil {
		return Copy{}, errors.New("backup: no hay base de datos")
	}
	if m.dir == "" {
		return Copy{}, errors.New("backup: la base de datos no tiene una carpeta de respaldo")
	}
	if !m.runMu.TryLock() {
		return Copy{}, ErrBusy
	}
	defer m.runMu.Unlock()

	copy, err := m.create(ctx)
	if err != nil {
		m.setLastError(err)
		log.Printf("backup: %v", err)
		return Copy{}, err
	}
	m.setLastError(nil)
	log.Printf("backup: copia creada %s", copy.Name)

	if err := m.applyRetention(copy.Name); err != nil {
		m.setLastError(err)
		log.Printf("backup: retención: %v", err)
	}
	return copy, nil
}

func (m *Manager) create(ctx context.Context) (Copy, error) {
	if err := ctx.Err(); err != nil {
		return Copy{}, err
	}
	if err := m.ensureDir(); err != nil {
		return Copy{}, err
	}

	created := m.now().UTC()
	target, err := m.targetPath(created)
	if err != nil {
		return Copy{}, err
	}

	temp, err := os.CreateTemp(m.dir, ".cinecat-*.tmp")
	if err != nil {
		return Copy{}, fmt.Errorf("backup: crear archivo temporal: %w", err)
	}
	tempPath := temp.Name()
	if err := temp.Close(); err != nil {
		os.Remove(tempPath)
		return Copy{}, fmt.Errorf("backup: cerrar archivo temporal: %w", err)
	}
	// VACUUM INTO necesita que el destino no exista; el nombre ya está
	// reservado y el archivo se vuelve a crear con los permisos definitivos.
	if err := os.Remove(tempPath); err != nil {
		return Copy{}, fmt.Errorf("backup: preparar archivo temporal: %w", err)
	}
	defer os.Remove(tempPath)

	query := "VACUUM INTO " + quoteSQLiteString(tempPath)
	if _, err := m.db.ExecContext(ctx, query); err != nil {
		return Copy{}, fmt.Errorf("backup: copiar base de datos: %w", err)
	}
	if err := os.Chmod(tempPath, 0o600); err != nil {
		return Copy{}, fmt.Errorf("backup: permisos de la copia: %w", err)
	}
	if err := os.Rename(tempPath, target); err != nil {
		return Copy{}, fmt.Errorf("backup: publicar copia: %w", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		return Copy{}, fmt.Errorf("backup: leer copia: %w", err)
	}
	return Copy{
		Name:      filepath.Base(target),
		SizeBytes: info.Size(),
		CreatedAt: created.Format(time.RFC3339),
	}, nil
}

func (m *Manager) targetPath(now time.Time) (string, error) {
	base := backupPrefix + now.Format(backupTimeLayout)
	for suffix := 1; ; suffix++ {
		name := base + backupSuffix
		if suffix > 1 {
			name = fmt.Sprintf("%s-%d%s", base, suffix, backupSuffix)
		}
		target := filepath.Join(m.dir, name)
		_, err := os.Stat(target)
		switch {
		case errors.Is(err, os.ErrNotExist):
			return target, nil
		case err != nil:
			return "", fmt.Errorf("backup: comprobar destino: %w", err)
		}
	}
}

func (m *Manager) ensureDir() error {
	if m == nil || m.dir == "" {
		return errors.New("backup: falta el directorio de copias")
	}
	if err := os.MkdirAll(m.dir, 0o700); err != nil {
		return fmt.Errorf("backup: crear directorio de copias: %w", err)
	}
	if err := os.Chmod(m.dir, 0o700); err != nil {
		return fmt.Errorf("backup: permisos del directorio de copias: %w", err)
	}
	return nil
}

func (m *Manager) listFiles() ([]fileCopy, error) {
	if m == nil || m.dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(m.dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("backup: leer directorio: %w", err)
	}

	files := make([]fileCopy, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, backupPrefix) || !strings.HasSuffix(name, backupSuffix) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("backup: leer copia %s: %w", name, err)
		}
		created := backupNameTime(name)
		if created.IsZero() {
			created = info.ModTime().UTC()
		}
		files = append(files, fileCopy{
			Copy: Copy{
				Name:      name,
				SizeBytes: info.Size(),
				CreatedAt: created.Format(time.RFC3339),
			},
			created: created,
		})
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].created.Equal(files[j].created) {
			return files[i].Name > files[j].Name
		}
		return files[i].created.After(files[j].created)
	})
	return files, nil
}

func (m *Manager) applyRetention(newest string) error {
	days := m.RetentionDays()
	if days < 1 {
		return nil
	}
	files, err := m.listFiles()
	if err != nil {
		return err
	}
	cutoff := m.now().UTC().AddDate(0, 0, -days)
	var failures []error
	for index, file := range files {
		if index == 0 || file.Name == newest {
			continue
		}
		if !file.created.Before(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(m.dir, file.Name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			failures = append(failures, fmt.Errorf("borrar %s: %w", file.Name, err))
		}
	}
	return errors.Join(failures...)
}

// LastBackup devuelve la fecha de la copia más reciente.
func (m *Manager) LastBackup() (time.Time, error) {
	files, err := m.listFiles()
	if err != nil {
		return time.Time{}, err
	}
	if len(files) == 0 {
		return time.Time{}, nil
	}
	return files[0].created, nil
}

// RunBackground ejecuta el bucle de copias del gestor hasta que se cancele
// el contexto.
func RunBackground(
	ctx context.Context,
	manager *Manager,
	changes <-chan struct{},
	initialDelay time.Duration,
	interval time.Duration,
	retryDelay time.Duration,
) {
	if manager != nil {
		manager.RunBackground(ctx, changes, initialDelay, interval, retryDelay)
	}
}

// RunBackground hace la primera comprobación tras initialDelay y después
// comprueba cada intervalo. Solo copia si la última copia tiene más de un
// intervalo; si falla, reintenta tras retryDelay.
func (m *Manager) RunBackground(
	ctx context.Context,
	changes <-chan struct{},
	initialDelay time.Duration,
	interval time.Duration,
	retryDelay time.Duration,
) {
	if m == nil {
		return
	}
	if initialDelay < 0 {
		initialDelay = 0
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	if retryDelay <= 0 {
		retryDelay = time.Hour
	}

	timer := time.NewTimer(initialDelay)
	if !m.Enabled() {
		stopTimer(timer)
	}
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-changes:
			if m.Enabled() {
				resetTimer(timer, 0)
			} else {
				stopTimer(timer)
			}
		case <-timer.C:
			if !m.Enabled() {
				continue
			}
			due, wait := m.backupDue(interval)
			if !due {
				resetTimer(timer, wait)
				continue
			}
			if _, err := m.Create(ctx); err != nil {
				resetTimer(timer, retryDelay)
				continue
			}
			resetTimer(timer, interval)
		}
	}
}

func (m *Manager) backupDue(interval time.Duration) (bool, time.Duration) {
	last, err := m.LastBackup()
	if err != nil || last.IsZero() {
		return true, 0
	}
	remaining := interval - m.now().Sub(last)
	if remaining <= 0 {
		return true, 0
	}
	return false, remaining
}

func resetTimer(timer *time.Timer, delay time.Duration) {
	stopTimer(timer)
	timer.Reset(delay)
}

func stopTimer(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

func backupDir(databasePath string) string {
	path := strings.TrimSpace(databasePath)
	if path == "" || path == ":memory:" {
		return ""
	}
	return filepath.Join(filepath.Dir(path), "backups")
}

func backupNameTime(name string) time.Time {
	stamp := strings.TrimSuffix(strings.TrimPrefix(name, backupPrefix), backupSuffix)
	if len(stamp) < len("20060102-150405") {
		return time.Time{}
	}
	parsed, err := time.Parse(backupTimeLayout, stamp[:len("20060102-150405")])
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

func quoteSQLiteString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
