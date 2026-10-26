//go:build windows

package playback

import (
	"context"
	"os/exec"
)

// ffmpegCommand no cambia la prioridad en Windows; el límite de hilos y de
// procesos sigue protegiendo al servidor.
func ffmpegCommand(ctx context.Context, ffmpeg string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, ffmpeg, args...)
}
