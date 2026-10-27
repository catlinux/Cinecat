//go:build !linux

package library

// diskFreeBytes no comprueba el espacio libre fuera de Linux.
func diskFreeBytes(string) (int64, bool) {
	return 0, false
}
