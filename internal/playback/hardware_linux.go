//go:build linux

package playback

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func hardwareDevicePath() string {
	if value := strings.TrimSpace(os.Getenv("CINECAT_VAAPI_DEVICE")); value != "" {
		return value
	}
	return defaultVAAPIDevice
}

func hardwareDeviceExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func probeHardware(ctx context.Context, command func(context.Context, string, ...string) *exec.Cmd, device string) error {
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-nostdin",
		"-y",
		"-vaapi_device", device,
		"-f", "lavfi",
		"-i", "color=c=black:s=320x240:d=1",
		"-vf", "format=nv12,hwupload",
		"-c:v", "h264_vaapi",
		"-f", "null",
		"-",
	}
	var cmd *exec.Cmd
	if command != nil {
		cmd = command(ctx, "ffmpeg", args...)
		if cmd == nil {
			return fmt.Errorf("comando de prueba vacío")
		}
	} else {
		ffmpeg, err := exec.LookPath("ffmpeg")
		if err != nil {
			return ErrFFmpegNotFound
		}
		cmd = ffmpegCommand(ctx, ffmpeg, args...)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, tailString(stderr.String(), 300))
	}
	return nil
}
