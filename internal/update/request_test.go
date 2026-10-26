package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRequesterEscribePeticionAtomica(t *testing.T) {
	dir := t.TempDir()
	clock := &updateFakeClock{now: time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)}
	requester := NewRequesterWithClock(dir, clock.Now)
	if !requester.Supported() {
		t.Fatal("el directorio temporal debe estar soportado")
	}
	if err := requester.Request("v0.14.0", "ana"); err != nil {
		t.Fatalf("Request: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, requestFileName))
	if err != nil {
		t.Fatalf("leer petición: %v", err)
	}
	var request Request
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("petición no válida: %v", err)
	}
	if request.Tag != "v0.14.0" || request.RequestedBy != "ana" ||
		request.RequestedAt != "2026-10-26T12:00:00Z" {
		t.Fatalf("petición inesperada: %+v", request)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, ".update-request-*"))
	if len(matches) != 0 {
		t.Fatalf("quedaron temporales: %v", matches)
	}
	if err := requester.Request("v0.14.0", "ana"); err != ErrConflict {
		t.Fatalf("segunda petición = %v, se esperaba conflicto", err)
	}
}

func TestRequesterConflictoPorEstadoActivoYEstadoCorrupto(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	requester := NewRequesterWithClock(dir, func() time.Time { return now })
	status := Status{
		Phase: "downloading", Tag: "v0.14.0",
		StartedAt: now.Add(-5 * time.Minute).Format(time.RFC3339),
	}
	raw, _ := json.Marshal(status)
	if err := os.WriteFile(filepath.Join(dir, statusFileName), raw, 0o600); err != nil {
		t.Fatalf("escribir estado: %v", err)
	}
	if err := requester.Request("v0.14.0", "ana"); err != ErrConflict {
		t.Fatalf("estado activo = %v, se esperaba conflicto", err)
	}

	if err := os.WriteFile(filepath.Join(dir, statusFileName), []byte("roto"), 0o600); err != nil {
		t.Fatalf("corromper estado: %v", err)
	}
	if got := requester.Status(); got != (Status{}) {
		t.Fatalf("estado corrupto = %+v", got)
	}
	if err := requester.Request("v0.14.0", "ana"); err != nil {
		t.Fatalf("con estado corrupto debería permitir la petición: %v", err)
	}
}

func TestRequesterSinDirectorioYEstado(t *testing.T) {
	requester := NewRequester(filepath.Join(t.TempDir(), "no-existe"))
	if requester.Supported() {
		t.Fatal("un directorio inexistente no debe estar soportado")
	}
	if err := requester.Request("v1.2.3", "ana"); err != ErrUnsupported {
		t.Fatalf("error = %v, se esperaba no soportado", err)
	}
	if status := requester.Status(); status != (Status{}) {
		t.Fatalf("estado vacío esperado: %+v", status)
	}
}
