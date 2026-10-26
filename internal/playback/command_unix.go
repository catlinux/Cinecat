//go:build !windows

package playback

import (
	"context"
	"os/exec"
)

// ffmpegCommand arranca FFmpeg con prioridad baja si el sistema tiene nice.
// El proceso no debe competir con el servidor por los núcleos.
func ffmpegCommand(ctx context.Context, ffmpeg string, args ...string) *exec.Cmd {
	if nice, err := exec.LookPath("nice"); err == nil {
		niceArgs := make([]string, 0, len(args)+3)
		niceArgs = append(niceArgs, "-n", "10", ffmpeg)
		niceArgs = append(niceArgs, args...)
		return exec.CommandContext(ctx, nice, niceArgs...)
	}
	return exec.CommandContext(ctx, ffmpeg, args...)
}
