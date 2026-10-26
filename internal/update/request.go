package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	EnvRunDir       = "CINECAT_RUN_DIR"
	DefaultRunDir   = "/run/cinecat-update"
	requestFileName = "update-request.json"
	statusFileName  = "update-status.json"
	activeWindow    = 30 * time.Minute
)

var (
	ErrUnsupported = errors.New("update: la actualización automática no está disponible")
	ErrConflict    = errors.New("update: ya hay una actualización en curso")
	ErrInvalidTag  = errors.New("update: etiqueta no válida")
)

type Request struct {
	Tag         string `json:"tag"`
	RequestedBy string `json:"requested_by"`
	RequestedAt string `json:"requested_at"`
}

type Status struct {
	Phase      string `json:"phase"`
	Tag        string `json:"tag"`
	Message    string `json:"message"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
	Result     string `json:"result"`
}

type Requester struct {
	dir string
	now func() time.Time
}

func NewRequester(dir string) *Requester {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = strings.TrimSpace(os.Getenv(EnvRunDir))
	}
	if dir == "" {
		dir = DefaultRunDir
	}
	return &Requester{dir: dir, now: time.Now}
}

func NewRequesterWithClock(dir string, now func() time.Time) *Requester {
	requester := NewRequester(dir)
	if now != nil {
		requester.now = now
	}
	return requester
}

func (r *Requester) Dir() string {
	if r == nil {
		return ""
	}
	return r.dir
}

func (r *Requester) requestPath() string {
	return filepath.Join(r.dir, requestFileName)
}

func (r *Requester) statusPath() string {
	return filepath.Join(r.dir, statusFileName)
}

// Supported indica si existe un directorio compartido y se puede escribir en él.
func (r *Requester) Supported() bool {
	if r == nil || strings.TrimSpace(r.dir) == "" {
		return false
	}
	info, err := os.Stat(r.dir)
	if err != nil || !info.IsDir() {
		return false
	}
	file, err := os.CreateTemp(r.dir, ".cinecat-write-*")
	if err != nil {
		return false
	}
	name := file.Name()
	file.Close()
	_ = os.Remove(name)
	return true
}

func (r *Requester) Status() Status {
	if r == nil || strings.TrimSpace(r.dir) == "" {
		return Status{}
	}
	raw, err := os.ReadFile(r.statusPath())
	if err != nil {
		return Status{}
	}
	var status Status
	if err := json.Unmarshal(raw, &status); err != nil {
		return Status{}
	}
	return status
}

// Request escribe de forma atómica la petición de actualización.
func (r *Requester) Request(tag, user string) error {
	if r == nil || !r.Supported() {
		return ErrUnsupported
	}
	if _, ok := ParseTag(tag); !ok {
		return ErrInvalidTag
	}
	if _, err := os.Stat(r.requestPath()); err == nil {
		return ErrConflict
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	status := r.Status()
	if status.Result == "" && status.Phase != "" && status.Phase != "idle" {
		started := r.statusStartedAt(status)
		if started.IsZero() || r.now().Sub(started) < activeWindow {
			return ErrConflict
		}
	}

	payload, err := json.Marshal(Request{
		Tag:         tag,
		RequestedBy: strings.TrimSpace(user),
		RequestedAt: r.now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(r.dir, ".update-request-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	// El contenedor escribe como root y el actualizador del host, con otro
	// usuario, debe poder leer la petición; no contiene nada secreto.
	if err := temp.Chmod(0o644); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(payload); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, r.requestPath()); err != nil {
		return fmt.Errorf("update: guardar petición: %w", err)
	}
	return nil
}

func (r *Requester) statusStartedAt(status Status) time.Time {
	if value, err := time.Parse(time.RFC3339, status.StartedAt); err == nil {
		return value
	}
	if info, err := os.Stat(r.statusPath()); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}
