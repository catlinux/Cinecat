package library

import (
	"sync"
	"time"
)

// missingScanInterval limita los escaneos que lanzan las peticiones de archivos
// desaparecidos: varios clientes pidiendo el mismo archivo no deben encadenarlos.
const missingScanInterval = time.Minute

var missingScan struct {
	mu   sync.Mutex
	last time.Time
}

// scanForMissingFile lanza un escaneo cuando se pide un archivo que ya no está
// en disco, normalmente porque se ha convertido o renombrado. El escáner
// reenlaza la ficha a la ruta nueva.
func (a *API) scanForMissingFile() {
	if a.scanJob == nil {
		return
	}
	missingScan.mu.Lock()
	defer missingScan.mu.Unlock()
	if !missingScan.last.IsZero() && time.Since(missingScan.last) < missingScanInterval {
		return
	}
	if a.scanJob.StartService() {
		missingScan.last = time.Now()
	}
}
