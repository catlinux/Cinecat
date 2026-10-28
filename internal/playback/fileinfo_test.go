package playback

import (
	"context"
	"errors"
	"math"
	"testing"
)

func mustParseFileInfo(t *testing.T, data string) *FileInfo {
	t.Helper()
	info, err := parseFFprobeJSON([]byte(data))
	if err != nil {
		t.Fatalf("parseFFprobeJSON: %v", err)
	}
	return info
}

func TestInspectSinFFprobe(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	if _, err := NewProber().Inspect(context.Background(), "x.mkv"); !errors.Is(err, ErrFFprobeNotFound) {
		t.Fatalf("error = %v, se esperaba ErrFFprobeNotFound", err)
	}
}

func TestParseFFprobeJSONMKV(t *testing.T) {
	info := mustParseFileInfo(t, `{
	  "streams": [
	    {
	      "index": 0,
	      "codec_name": "h264",
	      "codec_tag_string": "AVC1",
	      "codec_long_name": "H.264 / AVC / MPEG-4 AVC",
	      "profile": "High",
	      "codec_type": "video",
	      "width": 1920,
	      "height": 1080,
	      "display_aspect_ratio": "16:9",
	      "pix_fmt": "yuv420p",
	      "field_order": "progressive",
	      "bits_per_raw_sample": "8",
	      "avg_frame_rate": "24000/1001",
	      "r_frame_rate": "24000/1001",
	      "bit_rate": "5000000",
	      "color_transfer": "bt709",
	      "tags": {"language": "eng", "title": "Principal"},
	      "disposition": {"default": 1, "forced": 0}
	    },
	    {
	      "index": 1,
	      "codec_name": "ac3",
	      "codec_long_name": "ATSC A/52A (AC-3)",
	      "profile": "",
	      "codec_type": "audio",
	      "channels": 6,
	      "channel_layout": "5.1(side)",
	      "sample_rate": "48000",
	      "bit_rate": "448000",
	      "tags": {"language": "spa", "title": "Castellano"},
	      "disposition": {"default": 1, "forced": 0}
	    },
	    {
	      "index": 2,
	      "codec_name": "aac",
	      "codec_long_name": "AAC (Advanced Audio Coding)",
	      "profile": "LC",
	      "codec_type": "audio",
	      "channels": 2,
	      "channel_layout": "stereo",
	      "sample_rate": "48000",
	      "bit_rate": "192000",
	      "tags": {"language": "eng"},
	      "disposition": {"default": 0, "forced": 0}
	    },
	    {
	      "index": 3,
	      "codec_name": "subrip",
	      "codec_type": "subtitle",
	      "tags": {"language": "spa", "title": "Forzados"},
	      "disposition": {"default": 0, "forced": 1}
	    },
	    {
	      "index": 4,
	      "codec_name": "hdmv_pgs_subtitle",
	      "codec_type": "subtitle",
	      "tags": {"language": "eng"},
	      "disposition": {"default": 0, "forced": 0}
	    }
	  ],
	  "format": {
	    "filename": "/media/pelis/Peli.mkv",
	    "format_name": "matroska,webm",
	    "format_long_name": "Matroska / WebM",
	    "duration": "7200.125000",
	    "size": "1234567890",
	    "bit_rate": "1371742"
	  },
	  "chapters": [{}, {}]
	}`)

	if info.Container != "matroska" || info.ContainerLongName != "Matroska / WebM" {
		t.Fatalf("contenedor = %q / %q", info.Container, info.ContainerLongName)
	}
	if info.SizeBytes != 1234567890 || info.DurationSeconds != 7200.125 || info.BitRate != 1371742 {
		t.Fatalf("datos del contenedor = %+v", info)
	}
	if info.FileName != "Peli.mkv" || info.Chapters != 2 || info.Quality != "Full HD" {
		t.Fatalf("resumen = %+v", info)
	}
	if len(info.Video) != 1 {
		t.Fatalf("vídeo = %+v", info.Video)
	}
	video := info.Video[0]
	if video.Codec != "h264" || video.Profile != "High" || video.Width != 1920 ||
		video.Height != 1080 || video.DisplayAspectRatio != "16:9" ||
		video.BitRate != 5000000 || video.PixelFormat != "yuv420p" ||
		video.CodecTag != "avc1" || video.BitDepth != 8 || video.HDR != "" || video.FieldOrder != "progressive" ||
		video.Language != "eng" ||
		video.Title != "Principal" || !video.Default {
		t.Fatalf("vídeo inesperado: %+v", video)
	}
	if math.Abs(video.FrameRate-23.976023976) > 0.001 {
		t.Fatalf("frame_rate = %f", video.FrameRate)
	}

	if len(info.Audio) != 2 {
		t.Fatalf("audio = %+v", info.Audio)
	}
	ac3 := info.Audio[0]
	if ac3.Codec != "ac3" || ac3.Channels != 6 || ac3.ChannelLayout != "5.1(side)" ||
		ac3.SampleRate != 48000 || ac3.BitRate != 448000 || ac3.Language != "spa" ||
		ac3.Title != "Castellano" || !ac3.Default {
		t.Fatalf("audio AC-3 inesperado: %+v", ac3)
	}
	aac := info.Audio[1]
	if aac.Codec != "aac" || aac.Channels != 2 || aac.Language != "eng" || aac.Default {
		t.Fatalf("audio AAC inesperado: %+v", aac)
	}

	if len(info.Subtitles) != 2 {
		t.Fatalf("subtítulos = %+v", info.Subtitles)
	}
	if sub := info.Subtitles[0]; sub.Codec != "subrip" || !sub.Text || !sub.Forced || sub.Default {
		t.Fatalf("subtítulo de texto inesperado: %+v", sub)
	}
	if sub := info.Subtitles[1]; sub.Codec != "hdmv_pgs_subtitle" || sub.Text || sub.Language != "eng" {
		t.Fatalf("subtítulo de imagen inesperado: %+v", sub)
	}
}

func TestParseFFprobeJSONAVI(t *testing.T) {
	info := mustParseFileInfo(t, `{
	  "streams": [
	    {
	      "index": 0,
	      "codec_name": "mpeg4",
	      "codec_long_name": "MPEG-4 part 2",
	      "codec_type": "video",
	      "width": 720,
	      "height": 576,
	      "pix_fmt": "yuv420p",
	      "avg_frame_rate": "25/1",
	      "bit_rate": "N/A"
	    },
	    {
	      "index": 1,
	      "codec_name": "mp3",
	      "codec_type": "audio",
	      "channels": 2,
	      "sample_rate": "44100",
	      "bit_rate": "128000"
	    }
	  ],
	  "format": {
	    "filename": "/media/pelis/DivX.avi",
	    "format_name": "avi",
	    "format_long_name": "AVI (Audio Video Interleaved)",
	    "duration": "N/A",
	    "size": "734003200",
	    "bit_rate": "N/A"
	  }
	}`)

	if info.Container != "avi" || info.DurationSeconds != 0 || info.BitRate != 0 {
		t.Fatalf("contenedor AVI inesperado: %+v", info)
	}
	if info.Quality != "SD" || len(info.Audio) != 1 || len(info.Subtitles) != 0 {
		t.Fatalf("resumen AVI inesperado: %+v", info)
	}
	video := info.Video[0]
	if video.Codec != "mpeg4" || video.Width != 720 || video.Height != 576 ||
		video.FrameRate != 25 || video.BitDepth != 8 {
		t.Fatalf("vídeo AVI inesperado: %+v", video)
	}
	if audio := info.Audio[0]; audio.Codec != "mp3" || audio.SampleRate != 44100 || audio.BitRate != 128000 {
		t.Fatalf("audio AVI inesperado: %+v", audio)
	}
}

func TestParseFFprobeJSONHDR(t *testing.T) {
	info := mustParseFileInfo(t, `{
	  "streams": [
	    {
	      "index": 0,
	      "codec_name": "hevc",
	      "codec_long_name": "H.265 / HEVC",
	      "profile": "Main 10",
	      "codec_type": "video",
	      "width": 3840,
	      "height": 2160,
	      "pix_fmt": "yuv420p10le",
	      "avg_frame_rate": "24000/1001",
	      "color_transfer": "smpte2084"
	    }
	  ],
	  "format": {
	    "filename": "/media/pelis/HDR.mkv",
	    "format_name": "matroska,webm",
	    "duration": "5400",
	    "size": "9000000000",
	    "bit_rate": "13333333"
	  }
	}`)

	video := info.Video[0]
	if video.BitDepth != 10 || video.HDR != "HDR10" || info.Quality != "4K" {
		t.Fatalf("vídeo HDR inesperado: %+v (calidad %q)", video, info.Quality)
	}
	if len(info.Audio) != 0 || info.Audio == nil {
		t.Fatalf("audio vacío debe ser []: %#v", info.Audio)
	}
	if len(info.Subtitles) != 0 || info.Subtitles == nil {
		t.Fatalf("subtítulos vacíos deben ser []: %#v", info.Subtitles)
	}
}

func TestParseFFprobeJSONDolbyVisionYHLG(t *testing.T) {
	dolby := mustParseFileInfo(t, `{
	  "streams": [{
	    "index": 0,
	    "codec_name": "hevc",
	    "codec_type": "video",
	    "width": 3840,
	    "height": 2160,
	    "pix_fmt": "yuv420p10le",
	    "color_transfer": "smpte2084",
	    "side_data_list": [{"side_data_type": "DOVI configuration record"}]
	  }],
	  "format": {"filename": "dolby.mkv", "format_name": "matroska", "size": "10"}
	}`)
	if dolby.Video[0].HDR != "Dolby Vision" {
		t.Fatalf("HDR Dolby Vision = %q", dolby.Video[0].HDR)
	}

	hlg := mustParseFileInfo(t, `{
	  "streams": [{
	    "index": 0,
	    "codec_name": "hevc",
	    "codec_type": "video",
	    "width": 1280,
	    "height": 720,
	    "pix_fmt": "yuv420p",
	    "color_transfer": "arib-std-b67"
	  }],
	  "format": {"filename": "hlg.mkv", "format_name": "matroska"}
	}`)
	if hlg.Video[0].HDR != "HLG" || hlg.Quality != "HD" {
		t.Fatalf("HLG inesperado: %+v (calidad %q)", hlg.Video[0], hlg.Quality)
	}
}

func TestParseFFprobeJSONMinimo(t *testing.T) {
	info := mustParseFileInfo(t, `{
	  "format": {"format_name": "mp4", "filename": "sin-pistas.mp4"}
	}`)

	if info.Container != "mp4" || info.FileName != "sin-pistas.mp4" {
		t.Fatalf("contenedor mínimo inesperado: %+v", info)
	}
	if info.Video == nil || info.Audio == nil || info.Subtitles == nil {
		t.Fatalf("los arrays deben ser [] y no nil: %+v", info)
	}
	if info.Quality != "" || info.Chapters != 0 {
		t.Fatalf("resumen mínimo inesperado: %+v", info)
	}
}
