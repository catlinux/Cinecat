package library

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const unknownMusicArtist = "Artista desconocido"

var audioExtensions = map[string]bool{
	".mp3": true, ".flac": true, ".m4a": true, ".aac": true,
	".ogg": true, ".oga": true, ".opus": true, ".wav": true,
}

var audioTitlePrefix = regexp.MustCompile(`^\s*\d{1,4}\s*[-._]\s*`)

// IsAudioFile indica si la extensión es de un formato de audio admitido.
func IsAudioFile(name string) bool {
	return audioExtensions[strings.ToLower(filepath.Ext(name))]
}

func musicFileTitle(name string) string {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	base = audioTitlePrefix.ReplaceAllString(base, "")
	base = strings.TrimSpace(base)
	if base != "" {
		return base
	}
	return strings.TrimSuffix(name, filepath.Ext(name))
}

func normalizeAlbumArtist(value string) string {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "various artists", "various", "vv.aa.", "vvaa", "varios artistas", "artistas varios":
		return "Varios artistas"
	default:
		return value
	}
}

func musicSortName(name string) string {
	name = strings.TrimSpace(name)
	lower := strings.ToLower(name)
	for _, prefix := range []string{"the ", "el ", "la "} {
		if strings.HasPrefix(lower, prefix) {
			return strings.TrimSpace(name[len(prefix):])
		}
	}
	return name
}

func musicArtistForPath(root, trackPath string) string {
	albumDir := filepath.Clean(filepath.Dir(trackPath))
	root = filepath.Clean(root)
	if albumDir == root {
		return ""
	}
	if isDiscFolder(filepath.Base(albumDir)) {
		albumDir = filepath.Dir(albumDir)
	}
	if albumDir == root {
		return ""
	}
	parent := filepath.Dir(albumDir)
	if filepath.Clean(parent) == root {
		return ""
	}
	name := filepath.Base(parent)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return ""
	}
	return name
}

func audioMIME(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		return "audio/mpeg"
	case ".flac":
		return "audio/flac"
	case ".m4a":
		return "audio/mp4"
	case ".aac":
		return "audio/aac"
	case ".ogg", ".oga":
		return "audio/ogg"
	case ".opus":
		return "audio/opus"
	case ".wav":
		return "audio/wav"
	default:
		return "application/octet-stream"
	}
}

// audioPlayable deja fuera solo el WAV multicanal, que los navegadores
// habituales no reproducen de forma fiable; el resto de formatos admitidos
// tienen soporte directo.
func audioPlayable(path string, channels int) bool {
	if strings.EqualFold(filepath.Ext(path), ".wav") && channels > 2 {
		return false
	}
	return IsAudioFile(path)
}

func isDiscFolder(name string) bool {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return false
	}
	for _, prefix := range []string{"cd", "disc", "disco", "disk"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// findAlbumCover busca la carátula en la carpeta de la pista y, en álbumes con
// subcarpetas de disco, también en la carpeta del álbum.
func findAlbumCover(dir string) string {
	if cover := findFolderCover(dir); cover != "" {
		return cover
	}
	if isDiscFolder(filepath.Base(dir)) {
		return findFolderCover(filepath.Dir(dir))
	}
	return ""
}

// findFolderCover busca cover/folder/front/album con una extensión de imagen en
// la carpeta del álbum, sin distinguir mayúsculas.
func findFolderCover(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	byName := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		ext := filepath.Ext(name)
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp":
		default:
			continue
		}
		base := strings.TrimSuffix(name, ext)
		if _, exists := byName[base]; !exists {
			byName[base] = filepath.Join(dir, entry.Name())
		}
	}
	for _, name := range []string{"cover", "folder", "front", "album"} {
		if path := byName[name]; path != "" {
			return path
		}
	}
	return ""
}
