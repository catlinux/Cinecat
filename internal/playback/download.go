package playback

import (
	"path/filepath"
	"strings"
)

const (
	DownloadBlockedContainer = "container"
	DownloadBlockedVideo     = "video"
	DownloadBlockedAudio     = "audio"
	DownloadBlockedClient    = "client"
)

// DownloadableFor indica si una app nativa puede reproducir el archivo
// original tras descargarlo. Usa los mismos criterios que el directo nativo
// (nativeVideo y nativeAudio) para que lo descargable coincida con lo que el
// reproductor del dispositivo sabe leer, sea cual sea el contenedor.
func DownloadableFor(info *FileInfo, caps Capabilities) (bool, string) {
	if info == nil {
		return false, DownloadBlockedVideo
	}
	var container nativeSet
	switch strings.ToLower(filepath.Ext(info.FileName)) {
	case ".mp4", ".m4v", ".mov":
		container = nativeMP4
	case ".mkv":
		container = nativeMKV
	default:
		return false, DownloadBlockedContainer
	}
	if !caps.Native() {
		return false, DownloadBlockedClient
	}
	if !caps.native.has(container) {
		return false, DownloadBlockedContainer
	}
	if !nativeVideo(info, caps) {
		return false, DownloadBlockedVideo
	}
	if !nativeAudio(info, caps, -1) {
		return false, DownloadBlockedAudio
	}
	return true, ""
}
