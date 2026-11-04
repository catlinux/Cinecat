package playback

import "testing"

func TestDownloadableFor(t *testing.T) {
	h264 := VideoStream{Codec: "h264", Profile: "High", PixelFormat: "yuv420p", BitDepth: 8}
	hevc := VideoStream{Codec: "hevc", CodecTag: "hvc1", Profile: "Main", PixelFormat: "yuv420p", BitDepth: 8}
	info := func(name, audio string) *FileInfo {
		result := &FileInfo{FileName: name, Video: []VideoStream{h264}}
		if audio != "" {
			result.Audio = []AudioStream{{Codec: audio, Channels: 6, Default: true}}
		}
		return result
	}
	tests := []struct {
		name    string
		info    *FileInfo
		caps    Capabilities
		want    bool
		blocked string
	}{
		{
			name: "h264 y aac en mkv",
			info: info("episodio.mkv", "aac"),
			caps: ParseCapabilities("mkv,h264,aac"),
			want: true,
		},
		{
			name: "h264 y ac3 declarado",
			info: info("episodio.mkv", "ac3"),
			caps: ParseCapabilities("mkv,h264,ac3"),
			want: true,
		},
		{
			name:    "ac3 sin declarar",
			info:    info("episodio.mkv", "ac3"),
			caps:    ParseCapabilities("mkv,h264"),
			blocked: DownloadBlockedAudio,
		},
		{
			name:    "hevc sin declarar",
			info:    func() *FileInfo { i := info("episodio.mkv", "aac"); i.Video[0] = hevc; return i }(),
			caps:    ParseCapabilities("mkv,h264,aac"),
			blocked: DownloadBlockedVideo,
		},
		{
			name:    "h264 sin declarar",
			info:    info("episodio.mkv", "aac"),
			caps:    ParseCapabilities("mkv,aac"),
			blocked: DownloadBlockedVideo,
		},
		{
			name: "hevc declarado",
			info: func() *FileInfo { i := info("episodio.mkv", "aac"); i.Video[0] = hevc; return i }(),
			caps: ParseCapabilities("mkv,hevc,aac"),
			want: true,
		},
		{
			name:    "dts no se admite aunque se declare",
			info:    info("episodio.mkv", "dts"),
			caps:    ParseCapabilities("mkv,h264,dts"),
			blocked: DownloadBlockedAudio,
		},
		{
			name:    "web sin capacidades nativas",
			info:    info("episodio.mkv", "aac"),
			caps:    ParseCapabilities("h264,aac"),
			blocked: DownloadBlockedClient,
		},
		{
			name:    "avi no admitido",
			info:    info("episodio.avi", "aac"),
			caps:    ParseCapabilities("mkv,h264,aac"),
			blocked: DownloadBlockedContainer,
		},
		{
			name: "sin pista de audio",
			info: info("episodio.mov", ""),
			caps: ParseCapabilities("mp4,h264"),
			want: true,
		},
		{
			name:    "sin ficha",
			caps:    ParseCapabilities("mkv,h264,aac"),
			blocked: DownloadBlockedVideo,
		},
		{
			name: "pista por defecto aunque no sea la primera",
			info: &FileInfo{
				FileName: "episodio.mkv",
				Video:    []VideoStream{h264},
				Audio: []AudioStream{
					{Codec: "dts", Channels: 6},
					{Codec: "flac", Channels: 6, Default: true},
				},
			},
			caps: ParseCapabilities("mkv,h264"),
			want: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, blocked := DownloadableFor(test.info, test.caps)
			if got != test.want || blocked != test.blocked {
				t.Fatalf("DownloadableFor = %v, %q; se esperaba %v, %q", got, blocked, test.want, test.blocked)
			}
		})
	}
}
