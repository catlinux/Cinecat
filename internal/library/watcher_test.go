package library

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func startTestWatcher(t *testing.T, db *sql.DB, job *ScanJob, quiet, maxWait time.Duration) (*LibraryWatcher, context.CancelFunc, <-chan struct{}) {
	t.Helper()
	watcher := NewLibraryWatcher(db, job, quiet, maxWait)
	watcher.tick = 10 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		watcher.Run(ctx)
		close(stopped)
	}()
	select {
	case <-watcher.ready:
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("el vigilante no terminó de cargar las carpetas")
	}
	return watcher, cancel, stopped
}

func waitTestScan(t *testing.T, scans <-chan struct{}) {
	t.Helper()
	select {
	case <-scans:
	case <-time.After(3 * time.Second):
		t.Fatal("no llegó el escaneo esperado")
	}
}

func TestLibraryWatcherEscaneaTrasQuiet(t *testing.T) {
	db := openTestDB(t)
	db.SetMaxOpenConns(1)
	root := t.TempDir()
	libraryID := createLibrary(t, db, root)
	addScannerLibraryPath(t, db, libraryID, root)

	job := NewScanJob(nil)
	t.Cleanup(job.Close)
	scans := make(chan struct{}, 4)
	job.scan = func(context.Context) (Result, error) {
		scans <- struct{}{}
		return Result{}, nil
	}

	_, cancel, stopped := startTestWatcher(t, db, job, 100*time.Millisecond, 2*time.Second)
	defer func() {
		cancel()
		<-stopped
		_ = db.Close()
	}()

	writeFile(t, filepath.Join(root, "Pelicula.mp4"))
	waitTestScan(t, scans)
}

func TestLibraryWatcherEsperaMaxWaitConCambiosContinuos(t *testing.T) {
	db := openTestDB(t)
	db.SetMaxOpenConns(1)
	root := t.TempDir()
	libraryID := createLibrary(t, db, root)
	addScannerLibraryPath(t, db, libraryID, root)

	job := NewScanJob(nil)
	t.Cleanup(job.Close)
	scans := make(chan struct{}, 4)
	job.scan = func(context.Context) (Result, error) {
		scans <- struct{}{}
		return Result{}, nil
	}

	_, cancel, stopped := startTestWatcher(t, db, job, 500*time.Millisecond, 200*time.Millisecond)
	defer func() {
		cancel()
		<-stopped
		_ = db.Close()
	}()

	writingDone := make(chan struct{})
	go func() {
		defer close(writingDone)
		for i := 0; i < 20; i++ {
			_ = os.WriteFile(filepath.Join(root, "Pelicula "+string(rune('A'+i))+".mp4"), []byte("x"), 0o644)
			time.Sleep(50 * time.Millisecond)
		}
	}()

	waitTestScan(t, scans)
	select {
	case <-writingDone:
		t.Fatal("el escaneo no debe esperar a que terminen los cambios")
	default:
	}
	<-writingDone
}

func TestLibraryWatcherVigilaCarpetaCreadaDespues(t *testing.T) {
	db := openTestDB(t)
	db.SetMaxOpenConns(1)
	root := t.TempDir()
	libraryID := createLibrary(t, db, root)
	addScannerLibraryPath(t, db, libraryID, root)

	job := NewScanJob(nil)
	t.Cleanup(job.Close)
	scans := make(chan struct{}, 4)
	job.scan = func(context.Context) (Result, error) {
		scans <- struct{}{}
		return Result{}, nil
	}

	_, cancel, stopped := startTestWatcher(t, db, job, 100*time.Millisecond, time.Second)
	defer func() {
		cancel()
		<-stopped
		_ = db.Close()
	}()

	subdir := filepath.Join(root, "Serie")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("crear carpeta: %v", err)
	}
	waitTestScan(t, scans)

	writeFile(t, filepath.Join(subdir, "Capitulo.mkv"))
	waitTestScan(t, scans)
}

func TestScanJobRepiteSiQuedanArchivosInestables(t *testing.T) {
	job := NewScanJob(nil)
	t.Cleanup(job.Close)
	job.retryGrace = 20 * time.Millisecond

	var calls atomic.Int32
	scans := make(chan struct{}, 4)
	job.scan = func(context.Context) (Result, error) {
		call := calls.Add(1)
		if call == 1 {
			return Result{SkippedUnstable: 1}, nil
		}
		scans <- struct{}{}
		return Result{}, nil
	}

	if !job.StartService() {
		t.Fatal("el primer escaneo debe arrancar")
	}
	select {
	case <-scans:
	case <-time.After(3 * time.Second):
		t.Fatal("no se repitió el escaneo tras los archivos inestables")
	}
	if calls.Load() < 2 {
		t.Fatalf("escaneos = %d, se esperaban al menos 2", calls.Load())
	}
}
