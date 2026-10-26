package library

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func waitScanJobDone(t *testing.T, job *ScanJob) ScanStatus {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status := job.Status()
		if !status.Running {
			return status
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("el escaneo no terminó a tiempo")
	return ScanStatus{}
}

func TestScanJobUnEscaneoALaVez(t *testing.T) {
	job := NewScanJob(nil)
	t.Cleanup(job.Close)

	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseScan := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseScan)
	job.scan = func(ctx context.Context) (Result, error) {
		close(started)
		<-release
		return Result{
			Libraries:          1,
			Files:              2,
			Upserted:           3,
			Identified:         4,
			Unidentified:       5,
			SeriesIdentified:   6,
			SeriesUnidentified: 7,
			Warnings:           []string{"aviso de prueba"},
		}, nil
	}

	if !job.Start(context.Background()) {
		t.Fatal("el primer Start debe arrancar el escaneo")
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("el escaneo no llegó a arrancar")
	}
	if job.Start(context.Background()) {
		t.Fatal("el segundo Start debe devolver false mientras corre")
	}

	running := job.Status()
	if !running.Running || running.StartedAt == nil || running.FinishedAt != nil {
		t.Fatalf("estado en curso inesperado: %+v", running)
	}
	if running.Warnings == nil {
		t.Fatal("warnings no debe ser nil mientras corre")
	}
	if running.StartedAt.Location() != time.UTC {
		t.Fatalf("started_at no está en UTC: %v", running.StartedAt)
	}

	releaseScan()
	final := waitScanJobDone(t, job)
	if final.Running || final.FinishedAt == nil || final.StartedAt == nil {
		t.Fatalf("estado final inesperado: %+v", final)
	}
	if final.Libraries != 1 || final.Files != 2 || final.Upserted != 3 ||
		final.Identified != 4 || final.Unidentified != 5 ||
		final.SeriesIdentified != 6 || final.SeriesUnidentified != 7 {
		t.Fatalf("contadores inesperados: %+v", final)
	}
	if len(final.Warnings) != 1 || final.Warnings[0] != "aviso de prueba" {
		t.Fatalf("avisos = %v", final.Warnings)
	}
	if final.Error != "" {
		t.Fatalf("error = %q", final.Error)
	}
	if final.FinishedAt.Location() != time.UTC {
		t.Fatalf("finished_at no está en UTC: %v", final.FinishedAt)
	}

	// La copia devuelta no comparte el slice de avisos.
	snapshot := job.Status()
	snapshot.Warnings[0] = "cambiado"
	if job.Status().Warnings[0] != "aviso de prueba" {
		t.Fatal("Status compartió el slice de avisos")
	}
}

func TestScanJobGuardaElError(t *testing.T) {
	job := NewScanJob(nil)
	defer job.Close()

	job.scan = func(ctx context.Context) (Result, error) {
		return Result{Libraries: 1, Files: 2, Warnings: []string{"parcial"}},
			errors.New("fallo de prueba")
	}

	if !job.Start(context.Background()) {
		t.Fatal("Start debe arrancar el escaneo")
	}
	final := waitScanJobDone(t, job)
	if final.Error != "fallo de prueba" {
		t.Fatalf("error = %q", final.Error)
	}
	if final.Libraries != 1 || final.Files != 2 {
		t.Fatalf("se deben conservar los contadores parciales: %+v", final)
	}
	if len(final.Warnings) != 1 || final.Warnings[0] != "parcial" {
		t.Fatalf("avisos = %v", final.Warnings)
	}
	if final.FinishedAt == nil {
		t.Fatal("finished_at debe estar puesto tras un error")
	}
}

func TestScanJobCancelacion(t *testing.T) {
	job := NewScanJob(nil)
	defer job.Close()

	entered := make(chan struct{})
	job.scan = func(ctx context.Context) (Result, error) {
		close(entered)
		<-ctx.Done()
		return Result{}, ctx.Err()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if !job.Start(ctx) {
		t.Fatal("Start debe arrancar el escaneo")
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("el escaneo no llegó a arrancar")
	}
	cancel()

	final := waitScanJobDone(t, job)
	if final.Error == "" {
		t.Fatal("la cancelación debe quedar registrada en Error")
	}
	if !strings.Contains(final.Error, "canceled") {
		t.Fatalf("error inesperado: %q", final.Error)
	}
	if final.FinishedAt == nil {
		t.Fatal("finished_at debe estar puesto tras cancelar")
	}
}
