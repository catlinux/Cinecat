package playback

import "testing"

func TestDownloadableFor(t *testing.T) {
	h264 := VideoStream{Codec: "h264", Profile: "High", PixelFormat: "yuv420p", BitDepth: 8}
	hevc := VideoStream{Codec: "hevc", Profile: "Main", PixelFormat: "yuv420p", BitDepth: 8}
	hevc10 := VideoStream{Codec: "hevc", Profile: "Main 10", PixelFormat: "yuv420p10le", BitDepth: 10}
	info := func(name string, video VideoStream, audio string, channels int) *FileInfo {
		result := &FileInfo{FileName: name, Video: []VideoStream{video}}
		if audio != "" {
			result.Audio = []AudioStream{{Codec: audio, Channels: channels, Default: true}}
		}
		return result
	}
	tests := []struct {
		name    string
		info    *FileInfo
		caps    string
		want    bool
		blocked string
	}{
		{name: "h264 y aac en mkv", info: info("a.mkv", h264, "aac", 2), caps: "mkv,h264,aac", want: true},
		{name: "h264 y ac3 declarado", info: info("a.mkv", h264, "ac3", 6), caps: "mkv,h264,ac3", want: true},
		{name: "ac3 sin declarar", info: info("a.mkv", h264, "ac3", 6), caps: "mkv,h264,aac", blocked: DownloadBlockedAudio},
		{name: "dts declarado", info: info("a.mkv", h264, "dts", 6), caps: "mkv,h264,dts", want: true},
		{name: "dts sin declarar", info: info("a.mkv", h264, "dts", 6), caps: "mkv,h264,aac", blocked: DownloadBlockedAudio},
		{name: "aac multicanal sin mch", info: info("a.mkv", h264, "aac", 6), caps: "mkv,h264,aac", blocked: DownloadBlockedAudio},
		{name: "aac multicanal con mch", info: info("a.mkv", h264, "aac", 6), caps: "mkv,h264,aac,mch", want: true},
		{name: "hevc sin declarar", info: info("a.mkv", hevc, "aac", 2), caps: "mkv,h264,aac", blocked: DownloadBlockedVideo},
		{name: "hevc declarado aunque no tenga etiqueta", info: info("a.mkv", hevc, "aac", 2), caps: "mkv,hevc,aac", want: true},
		{name: "hevc 10 bits sin declarar", info: info("a.mkv", hevc10, "aac", 2), caps: "mkv,hevc,aac", blocked: DownloadBlockedVideo},
		{name: "hevc 10 bits declarado", info: info("a.mkv", hevc10, "aac", 2), caps: "mkv,hevc10,aac", want: true},
		{name: "h264 sin declarar", info: info("a.mkv", h264, "aac", 2), caps: "mkv,aac", blocked: DownloadBlockedVideo},
		{name: "web sin capacidades nativas", info: info("a.mkv", h264, "aac", 2), caps: "h264,aac", blocked: DownloadBlockedClient},
		{name: "avi no admitido", info: info("a.avi", h264, "aac", 2), caps: "mkv,h264,aac", blocked: DownloadBlockedContainer},
		{name: "mkv sin declarar el contenedor", info: info("a.mkv", h264, "aac", 2), caps: "mp4,h264,aac", blocked: DownloadBlockedContainer},
		{name: "mov sin pista de audio", info: info("a.mov", h264, "", 0), caps: "mp4,h264", want: true},
		{name: "sin ficha", caps: "mkv,h264,aac", blocked: DownloadBlockedVideo},
		{
			name: "pista por defecto aunque no sea la primera",
			info: &FileInfo{
				FileName: "a.mkv",
				Video:    []VideoStream{h264},
				Audio: []AudioStream{
					{Codec: "dts", Channels: 6},
					{Codec: "flac", Channels: 6, Default: true},
				},
			},
			caps: "mkv,h264,flac,mch",
			want: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, blocked := DownloadableFor(test.info, ParseCapabilities(test.caps))
			if got != test.want || blocked != test.blocked {
				t.Fatalf("DownloadableFor = %v, %q; se esperaba %v, %q", got, blocked, test.want, test.blocked)
			}
		})
	}
}
