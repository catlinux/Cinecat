package library

import (
	"context"
	"sync"
	"time"
)

// ScanStatus es el estado del último escaneo lanzado desde ScanJob. Las
// advertencias se copian para que un lector no altere las del trabajo.
type ScanStatus struct {
	Running      bool       `json:"running"`
	StartedAt    *time.Time `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Libraries    int        `json:"libraries"`
	Files        int        `json:"files"`
	Upserted     int        `json:"upserted"`
	Identified   int        `json:"identified"`
	Unidentified int        `json:"unidentified"`
	Warnings     []string   `json:"warnings"`
	Error        string     `json:"error"`
}

// ScanJob ejecuta como máximo un escaneo a la vez y conserva el último estado
// para poder consultarlo mientras corre y cuando termina.
type ScanJob struct {
	mu      sync.Mutex
	running bool
	closed  bool
	status  ScanStatus
	scan    func(context.Context) (Result, error)
	wg      sync.WaitGroup
	service context.Context
	cancel  context.CancelFunc
}

// NewScanJob crea el trabajo de escaneo. El contexto de servicio que se usa
// como base se cancela con Close, para que un escaneo lanzado desde HTTP no
// siga vivo después de apagar el servidor.
func NewScanJob(scanner *Scanner) *ScanJob {
	ctx, cancel := context.WithCancel(context.Background())
	job := &ScanJob{
		status:  ScanStatus{Warnings: []string{}},
		service: ctx,
		cancel:  cancel,
	}
	if scanner != nil {
		job.scan = scanner.Scan
	}
	return job
}

// Start lanza un escaneo con el contexto recibido y devuelve false si ya hay
// otro en marcha o si el trabajo ya se cerró. El escaneo no queda ligado a una
// petición HTTP: el contexto debe ser el del servicio.
func (j *ScanJob) Start(ctx context.Context) bool {
	j.mu.Lock()
	if j.running || j.closed || j.scan == nil {
		j.mu.Unlock()
		return false
	}
	j.running = true
	startedAt := time.Now().UTC()
	j.status = ScanStatus{
		Running:   true,
		StartedAt: &startedAt,
		Warnings:  []string{},
	}
	scan := j.scan
	j.wg.Add(1)
	j.mu.Unlock()

	scanCtx, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-j.service.Done():
			cancel()
		case <-scanCtx.Done():
		}
	}()

	go func() {
		defer j.wg.Done()
		defer cancel()

		result, err := scan(scanCtx)
		finishedAt := time.Now().UTC()

		j.mu.Lock()
		j.running = false
		j.status.Running = false
		j.status.FinishedAt = &finishedAt
		j.status.Libraries = result.Libraries
		j.status.Files = result.Files
		j.status.Upserted = result.Upserted
		j.status.Identified = result.Identified
		j.status.Unidentified = result.Unidentified
		j.status.Warnings = append([]string{}, result.Warnings...)
		if j.status.Warnings == nil {
			j.status.Warnings = []string{}
		}
		if err != nil {
			j.status.Error = err.Error()
		} else {
			j.status.Error = ""
		}
		j.mu.Unlock()
	}()

	return true
}

// StartService lanza un escaneo con el contexto de vida del trabajo. Es el
// que usa la API HTTP para que el escaneo no quede ligado a la petición.
func (j *ScanJob) StartService() bool {
	return j.Start(j.service)
}

// Status devuelve una copia segura del estado actual.
func (j *ScanJob) Status() ScanStatus {
	j.mu.Lock()
	defer j.mu.Unlock()

	status := j.status
	if status.Warnings == nil {
		status.Warnings = []string{}
	} else {
		status.Warnings = append([]string{}, status.Warnings...)
	}
	if status.StartedAt != nil {
		startedAt := *status.StartedAt
		status.StartedAt = &startedAt
	}
	if status.FinishedAt != nil {
		finishedAt := *status.FinishedAt
		status.FinishedAt = &finishedAt
	}
	return status
}

// Close cancela el contexto de servicio y espera a que termine el escaneo en
// marcha, si lo hay.
func (j *ScanJob) Close() {
	j.mu.Lock()
	j.closed = true
	j.cancel()
	j.mu.Unlock()
	j.wg.Wait()
}
