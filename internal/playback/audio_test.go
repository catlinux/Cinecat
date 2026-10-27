package playback

import (
	"context"
	"errors"
	"testing"
)

func TestParseFFprobeAudioJSON(t *testing.T) {
	info, err := parseFFprobeAudioJSON([]byte(`{
	  "streams": [
	    {"codec_name": "mjpeg", "codec_type": "video", "disposition": {"attached_pic": 1}},
	    {"codec_name": "flac", "codec_type": "audio", "bit_rate": "900000",
	     "sample_rate": "44100", "channels": 2, "disposition": {"attached_pic": 0}}
	  ],
	  "format": {
	    "duration": "245.5",
	    "tags": {
	      "TITLE": "Canción de prueba",
	      "artist": "Artista",
	      "ALBUMARTIST": "Varios artistas",
	      "album": "Álbum",
	      "track": "3/12",
	      "DISC": "2/2",
	      "date": "2019-05-03",
	      "genre": "Pop"
	    }
	  }
	}`))
	if err != nil {
		t.Fatalf("parseFFprobeAudioJSON: %v", err)
	}
	if info.Tags.Title != "Canción de prueba" || info.Tags.Artist != "Artista" ||
		info.Tags.AlbumArtist != "Varios artistas" || info.Tags.Album != "Álbum" ||
		info.Tags.Genre != "Pop" {
		t.Fatalf("etiquetas: %+v", info.Tags)
	}
	if info.Tags.Track != 3 || info.Tags.Disc != 2 || info.Tags.Year != 2019 {
		t.Fatalf("números: %+v", info.Tags)
	}
	if info.Codec != "flac" || info.BitRate != 900000 || info.SampleRate != 44100 ||
		info.Channels != 2 || info.DurationSeconds != 245.5 {
		t.Fatalf("datos técnicos: %+v", info)
	}
	if !info.HasEmbeddedCover {
		t.Fatal("debe detectar la imagen incrustada")
	}
}

func TestParseFFprobeAudioSinEtiquetasNiImagen(t *testing.T) {
	info, err := parseFFprobeAudioJSON([]byte(`{
	  "streams": [{"codec_name": "mp3", "codec_type": "audio", "channels": 1}],
	  "format": {"duration": "10", "bit_rate": "128000", "tags": {"artist": "", "title": " feat. Otra "}}
	}`))
	if err != nil {
		t.Fatalf("parseFFprobeAudioJSON: %v", err)
	}
	if info.Tags.Artist != "" || info.Tags.Title != "feat. Otra" {
		t.Fatalf("etiquetas: %+v", info.Tags)
	}
	if info.HasEmbeddedCover || info.Tags.Track != 0 || info.Tags.Disc != 0 {
		t.Fatalf("valores inesperados: %+v", info)
	}
	if info.BitRate != 128000 {
		t.Fatalf("debe usar el bitrate del contenedor: %d", info.BitRate)
	}
}

func TestParseTagNumberYYear(t *testing.T) {
	cases := []struct {
		value string
		want  int
	}{
		{"3/12", 3}, {" 7 ", 7}, {"0", 0}, {"", 0}, {"x/2", 0},
	}
	for _, test := range cases {
		if got := parseTagNumber(test.value); got != test.want {
			t.Errorf("parseTagNumber(%q) = %d, se esperaba %d", test.value, got, test.want)
		}
	}
	for _, test := range []struct {
		value string
		want  int
	}{{"2019-05-03", 2019}, {"Mayo de 1984", 1984}, {"sin año", 0}, {"", 0}} {
		if got := parseYearTag(test.value); got != test.want {
			t.Errorf("parseYearTag(%q) = %d, se esperaba %d", test.value, got, test.want)
		}
	}
}

func TestInspectAudioSinFFprobe(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := NewProber().InspectAudio(context.Background(), "x.mp3"); !errors.Is(err, ErrFFprobeNotFound) {
		t.Fatalf("error = %v, se esperaba ErrFFprobeNotFound", err)
	}
}
