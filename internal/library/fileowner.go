package library

import (
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// applyConfiguredOwner cambia el propietario de una ruta recién creada si el
// despliegue define CINECAT_FILE_UID y CINECAT_FILE_GID. Si falta alguno o no
// son números válidos, no se hace nada; un fallo solo se registra.
func applyConfiguredOwner(path string) {
	if runtime.GOOS == "windows" {
		return
	}
	uidText := strings.TrimSpace(os.Getenv("CINECAT_FILE_UID"))
	gidText := strings.TrimSpace(os.Getenv("CINECAT_FILE_GID"))
	if uidText == "" || gidText == "" {
		return
	}
	uid, err := strconv.Atoi(uidText)
	if err != nil || uid < 0 {
		return
	}
	gid, err := strconv.Atoi(gidText)
	if err != nil || gid < 0 {
		return
	}
	if err := os.Lchown(path, uid, gid); err != nil {
		log.Printf("library: no se pudo cambiar el propietario de %s: %v", path, err)
	}
}
