//go:build linux

package library

import (
	"math"
	"syscall"
)

// diskFreeBytes devuelve el espacio disponible para el usuario en el sistema
// de archivos de path. El segundo valor indica si se pudo consultar.
func diskFreeBytes(path string) (int64, bool) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, false
	}
	if stat.Bsize <= 0 {
		return 0, false
	}
	blockSize := uint64(stat.Bsize)
	if stat.Bavail > uint64(math.MaxInt64)/blockSize {
		return math.MaxInt64, true
	}
	return int64(stat.Bavail * blockSize), true
}
