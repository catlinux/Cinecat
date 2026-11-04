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
// original tras descargarlo, sin depender de la decisión de HLS del servidor.
func DownloadableFor(info *FileInfo, caps Capabilities) (bool, string) {
	if info == nil {
		return false, DownloadBlockedVideo
	}
	switch strings.ToLower(filepath.Ext(info.FileName)) {
	case ".mp4", ".m4v", ".mkv", ".mov":
	default:
		return false, DownloadBlockedContainer
	}
	if !caps.Native() {
		return false, DownloadBlockedClient
	}
	if !directVideo(info, caps) {
		return false, DownloadBlockedVideo
	}
	if strings.EqualFold(strings.TrimSpace(info.Video[0].Codec), "h264") &&
		!caps.native.has(nativeH264) {
		return false, DownloadBlockedVideo
	}
	if !downloadableAudio(info, caps) {
		return false, DownloadBlockedAudio
	}
	return true, ""
}

func downloadableAudio(info *FileInfo, caps Capabilities) bool {
	audio, ok := selectedAudio(info, -1)
	if !ok {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(audio.Codec)) {
	case "aac", "mp3", "opus", "vorbis", "flac":
		return true
	case "ac3":
		return caps.native.has(nativeAC3)
	case "eac3":
		return caps.native.has(nativeEAC3)
	default:
		return false
	}
}
