//go:build windows

package playback

import (
	"context"
	"os/exec"
)

// lowPriorityCommand no cambia la prioridad en Windows; el límite de hilos y
// de procesos sigue protegiendo al servidor.
func lowPriorityCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

// ffmpegCommand arranca FFmpeg sin cambiar la prioridad en Windows.
func ffmpegCommand(ctx context.Context, ffmpeg string, args ...string) *exec.Cmd {
	return lowPriorityCommand(ctx, ffmpeg, args...)
}
