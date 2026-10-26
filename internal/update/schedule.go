package update

import (
	"context"
	"log"
	"time"
)

// RunBackground consulta periódicamente GitHub mientras el ajuste esté activo.
// Los cambios de ajuste despiertan el bucle: al activarlo comprueba en ese
// momento y al desactivarlo deja de programar consultas.
func RunBackground(
	ctx context.Context,
	checker *Checker,
	enabled func() bool,
	changes <-chan struct{},
	initialDelay time.Duration,
	interval time.Duration,
) {
	if checker == nil {
		return
	}
	if enabled == nil {
		enabled = func() bool { return false }
	}
	if initialDelay < 0 {
		initialDelay = 0
	}
	if interval <= 0 {
		interval = 6 * time.Hour
	}

	timer := time.NewTimer(initialDelay)
	if !enabled() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
	}
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-changes:
			if enabled() {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				// Al activar el ajuste se comprueba ya; el intervalo se aplica
				// a partir de esa comprobación.
				timer.Reset(0)
			} else if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
			if !enabled() {
				continue
			}
			if _, err := checker.Check(ctx, true); err != nil {
				log.Printf("update: comprobación automática: %v", err)
			}
			timer.Reset(interval)
		}
	}
}
