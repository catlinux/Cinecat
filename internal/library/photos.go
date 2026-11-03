package library

import (
	"path/filepath"
	"strings"
)

var photoExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true,
}

// IsPhotoFile indica si la extensión es una imagen admitida por las
// bibliotecas de fotos. HEIC y RAW quedan fuera porque necesitan dependencias
// externas.
func IsPhotoFile(name string) bool {
	return photoExtensions[strings.ToLower(filepath.Ext(name))]
}

func photoMIME(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "image/jpeg"
	}
}
