package playback

import (
	"path/filepath"
	"strings"
)

// nativeSet guarda los tokens de las apps nativas como bits para que
// Capabilities siga siendo comparable con ==.
type nativeSet uint32

const (
	nativeMKV nativeSet = 1 << iota
	nativeMP4
	nativeH264
	nativeH264Hi10
	nativeAV1
	nativeVP9
	nativeAAC
	nativeMP3
	nativeAC3
	nativeEAC3
	nativeDTS
	nativeTrueHD
	nativeOpus
	nativeFLAC
	nativeMultichannel
)

// nativeTokens fija el orden canónico de QueryValue. hevc y hevc10 no están
// aquí porque ya los usa la web con el mismo significado (decodifica HEVC).
var nativeTokens = []struct {
	name string
	bit  nativeSet
}{
	{"mkv", nativeMKV},
	{"mp4", nativeMP4},
	{"h264", nativeH264},
	{"h264hi10", nativeH264Hi10},
	{"av1", nativeAV1},
	{"vp9", nativeVP9},
	{"aac", nativeAAC},
	{"mp3", nativeMP3},
	{"ac3", nativeAC3},
	{"eac3", nativeEAC3},
	{"dts", nativeDTS},
	{"truehd", nativeTrueHD},
	{"opus", nativeOpus},
	{"flac", nativeFLAC},
	{"mch", nativeMultichannel},
}

func nativeToken(name string) (nativeSet, bool) {
	for _, token := range nativeTokens {
		if token.name == name {
			return token.bit, true
		}
	}
	return 0, false
}

func (s nativeSet) has(bit nativeSet) bool {
	return s&bit != 0
}

// Native indica que las capacidades vienen de una app nativa. Solo las apps
// declaran contenedores; la web nunca lo hace, así que sin ellos la decisión
// es exactamente la de navegador.
func (c Capabilities) Native() bool {
	return c.native.has(nativeMKV) || c.native.has(nativeMP4)
}

// Reason explica por qué no se reproduce en directo. Vacío en directo.
type Reason string

const (
	ReasonNone      Reason = ""
	ReasonContainer Reason = "container"
	ReasonVideo     Reason = "video"
	ReasonAudio     Reason = "audio"
)

// DecideForAudio es DecideFor con la pista de audio elegida (posición en
// info.Audio; negativa para la pista por defecto) y el motivo de la decisión.
// La pista solo cuenta para las apps nativas: el navegador siempre recibe la
// pista por defecto en directo.
func DecideForAudio(path string, info *FileInfo, caps Capabilities, audioIndex int) (Mode, Reason, error) {
	if !caps.Native() {
		mode, err := DecideFor(path, info, caps)
		if err != nil || mode == ModeDirect {
			return mode, ReasonNone, err
		}
		return mode, browserReason(path, info, caps), nil
	}
	return decideNative(path, info, caps, audioIndex)
}

func browserReason(path string, info *FileInfo, caps Capabilities) Reason {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4", ".m4v":
	default:
		return ReasonContainer
	}
	if !directVideo(info, caps) {
		return ReasonVideo
	}
	return ReasonAudio
}

func decideNative(path string, info *FileInfo, caps Capabilities, audioIndex int) (Mode, Reason, error) {
	// Decide valida la extensión con las mismas reglas que la web.
	if _, err := DecideFor(path, nil, Capabilities{}); err != nil {
		return "", ReasonNone, err
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4", ".m4v":
		if !caps.native.has(nativeMP4) {
			return ModeRemux, ReasonContainer, nil
		}
	case ".mkv":
		if !caps.native.has(nativeMKV) {
			return ModeRemux, ReasonContainer, nil
		}
	default:
		return ModeRemux, ReasonContainer, nil
	}
	if info == nil {
		// Sin ficha no se puede comprobar nada; el MP4 conserva el directo de
		// siempre y el MKV, más variado, va por HLS.
		if strings.EqualFold(filepath.Ext(path), ".mkv") {
			return ModeRemux, ReasonVideo, nil
		}
		return ModeDirect, ReasonNone, nil
	}
	if !nativeVideo(info, caps) {
		return ModeRemux, ReasonVideo, nil
	}
	if !nativeAudio(info, caps, audioIndex) {
		return ModeRemux, ReasonAudio, nil
	}
	return ModeDirect, ReasonNone, nil
}

func nativeVideo(info *FileInfo, caps Capabilities) bool {
	if len(info.Video) == 0 {
		return false
	}
	video := info.Video[0]
	switch strings.ToLower(strings.TrimSpace(video.Codec)) {
	case "h264":
		if copyableH264(video) {
			return caps.native.has(nativeH264)
		}
		return caps.native.has(nativeH264Hi10)
	case "hevc":
		// ExoPlayer acepta hvc1 y hev1, así que la etiqueta no importa aquí.
		if hevc10Bit(video) {
			return caps.HEVC10
		}
		return caps.HEVC || caps.HEVC10
	case "av1":
		return caps.native.has(nativeAV1)
	case "vp9":
		return caps.native.has(nativeVP9)
	default:
		return false
	}
}

// selectedAudio devuelve la pista elegida, o la por defecto si el índice no
// es válido. ok es false si el archivo no tiene audio.
func selectedAudio(info *FileInfo, audioIndex int) (AudioStream, bool) {
	if len(info.Audio) == 0 {
		return AudioStream{}, false
	}
	if audioIndex >= 0 && audioIndex < len(info.Audio) {
		return info.Audio[audioIndex], true
	}
	for _, track := range info.Audio {
		if track.Default {
			return track, true
		}
	}
	return info.Audio[0], true
}

func nativeAudio(info *FileInfo, caps Capabilities, audioIndex int) bool {
	audio, ok := selectedAudio(info, audioIndex)
	if !ok {
		return true
	}
	var bit nativeSet
	// Los formatos de cine son multicanal por naturaleza: declararlos ya
	// implica que el equipo los decodifica o los pasa por HDMI.
	inherentlyMultichannel := false
	switch strings.ToLower(strings.TrimSpace(audio.Codec)) {
	case "aac":
		bit = nativeAAC
	case "mp3":
		bit = nativeMP3
	case "ac3":
		bit, inherentlyMultichannel = nativeAC3, true
	case "eac3":
		bit, inherentlyMultichannel = nativeEAC3, true
	case "dts":
		bit, inherentlyMultichannel = nativeDTS, true
	case "truehd":
		bit, inherentlyMultichannel = nativeTrueHD, true
	case "opus":
		bit = nativeOpus
	case "flac":
		bit = nativeFLAC
	default:
		return false
	}
	if !caps.native.has(bit) {
		return false
	}
	if audio.Channels > 2 && !inherentlyMultichannel {
		return caps.native.has(nativeMultichannel)
	}
	return true
}
