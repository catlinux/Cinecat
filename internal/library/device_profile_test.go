package library

import (
	"testing"

	"github.com/catlinux/cinecat/internal/playback"
)

func TestSameLanguageNormalizaCodigosDeDosYTresLetras(t *testing.T) {
	pairs := [][2]string{
		{"es", "spa"},
		{"en", "eng"},
		{"ca", "cat"},
		{"fr", "fra"},
		{"fr", "fre"},
		{"de", "deu"},
		{"de", "ger"},
		{"it", "ita"},
		{"pt", "por"},
		{"ja", "jpn"},
	}
	for _, pair := range pairs {
		if !sameLanguage(pair[0], pair[1]) {
			t.Errorf("sameLanguage(%q, %q) = false", pair[0], pair[1])
		}
	}
	if sameLanguage("es", "en") || sameLanguage("spa", "eng") {
		t.Fatal("idiomas distintos no deben coincidir")
	}
}

func TestPerfilEligeSubtituloDeTextoYEvitaImagenes(t *testing.T) {
	tracks := []playback.SubtitleTrack{
		{ID: "e0", Language: "spa", Image: true},
		{ID: "e1", Language: "eng", Forced: true},
		{ID: "e2", Language: "en"},
		{ID: "e3", Language: "spa", Forced: true},
	}
	profile := deviceProfile{SubtitleLanguage: "eng"}
	if got := profile.subtitleID(tracks); got != "e2" {
		t.Fatalf("subtítulo elegido = %q, se esperaba e2", got)
	}
	if got := (deviceProfile{SubtitleLanguage: "es"}).subtitleID(tracks); got != "e3" {
		t.Fatalf("subtítulo forzado de respaldo = %q, se esperaba e3", got)
	}
	if got := (deviceProfile{SubtitleLanguage: "off"}).subtitleID(tracks); got != "" {
		t.Fatalf("off no debe elegir subtítulo: %q", got)
	}
	if got := (deviceProfile{SubtitleLanguage: "spa"}).subtitleID([]playback.SubtitleTrack{
		{ID: "e0", Language: "spa", Image: true},
	}); got != "" {
		t.Fatalf("una imagen no debe elegirse por perfil: %q", got)
	}
}

func TestPerfilEligePrimeraPistaDeAudioDelIdioma(t *testing.T) {
	tracks := []playback.AudioStream{
		{Language: "eng"},
		{Language: "spa"},
		{Language: "spa"},
	}
	if got := (deviceProfile{AudioLanguage: "es"}).audioIndex(tracks); got != 1 {
		t.Fatalf("audio elegido = %d, se esperaba 1", got)
	}
	if got := (deviceProfile{}).audioIndex(tracks); got != -1 {
		t.Fatalf("sin preferencia no debe elegir audio: %d", got)
	}
	if got := (deviceProfile{AudioLanguage: "fra"}).audioIndex(tracks); got != -1 {
		t.Fatalf("un idioma ausente no debe elegir audio: %d", got)
	}
}

func TestPerfilConvierteElTopeDeCalidad(t *testing.T) {
	cases := map[string]int{"": 0, "480": 480, "720": 720, "1080": 1080, "auto": 0}
	for value, want := range cases {
		if got := (deviceProfile{MaxQuality: value}).qualityHeight(); got != want {
			t.Errorf("qualityHeight(%q) = %d, se esperaba %d", value, got, want)
		}
	}
}
