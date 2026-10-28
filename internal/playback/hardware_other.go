//go:build !linux

package playback

import (
	"context"
	"errors"
	"os/exec"
)

func hardwareDevicePath() string {
	return ""
}

func hardwareDeviceExists(string) bool {
	return false
}

func probeHardware(context.Context, func(context.Context, string, ...string) *exec.Cmd, string) error {
	return errors.New("VA-API solo está disponible en Linux")
}
