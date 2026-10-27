package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFormatosDeAudioAdmitidos(t *testing.T) {
	for _, name := range []string{"a.mp3", "b.FLAC", "c.m4a", "d.aac", "e.ogg", "f.oga", "g.opus", "h.wav"} {
		if !IsAudioFile(name) {
			t.Errorf("%q debe ser audio", name)
		}
	}
	for _, name := range []string{"a.wma", "b.ape", "c.mkv", "sin-extension"} {
		if IsAudioFile(name) {
			t.Errorf("%q no debe ser audio", name)
		}
	}
}

func TestTituloDeRespaldoSinPrefijo(t *testing.T) {
	cases := map[string]string{
		"01 - Canción.mp3":      "Canción",
		"02. Otra canción.flac": "Otra canción",
		"03_Tercera.ogg":        "Tercera",
		"1979.mp3":              "1979",
		"feat. Artista.mp3":     "feat. Artista",
		"01 - .mp3":             "01 - ",
	}
	for file, want := range cases {
		if got := musicFileTitle(file); got != want {
			t.Errorf("musicFileTitle(%q) = %q, se esperaba %q", file, got, want)
		}
	}
}

func TestNormalizacionDeArtistaYOrden(t *testing.T) {
	for _, value := range []string{"Various Artists", "VV.AA.", "vvaa", "Varios artistas", "  various artists  "} {
		if got := normalizeAlbumArtist(value); got != "Varios artistas" {
			t.Errorf("normalizeAlbumArtist(%q) = %q", value, got)
		}
	}
	if got := normalizeAlbumArtist("feat. Otra"); got != "feat. Otra" {
		t.Errorf("feat. no debe tocarse: %q", got)
	}
	cases := map[string]string{
		"The Beatles": "Beatles",
		"the beatles": "beatles",
		"El Último":   "Último",
		"La Oreja":    "Oreja",
		"The":         "The",
		"Óscar":       "Óscar",
	}
	for value, want := range cases {
		if got := musicSortName(value); got != want {
			t.Errorf("musicSortName(%q) = %q, se esperaba %q", value, got, want)
		}
	}
}

func TestArtistaDeRespaldoPorEstructura(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		path string
		want string
	}{
		{filepath.Join(root, "Artista", "Álbum", "01 - Pista.mp3"), "Artista"},
		{filepath.Join(root, "Artista", "Álbum", "CD 1", "01 - Pista.mp3"), "Artista"},
		{filepath.Join(root, "Álbum", "01 - Pista.mp3"), ""},
		{filepath.Join(root, "01 - Pista.mp3"), ""},
	}
	for _, test := range cases {
		if got := musicArtistForPath(root, test.path); got != test.want {
			t.Errorf("musicArtistForPath(%q) = %q, se esperaba %q", test.path, got, test.want)
		}
	}
}

func TestCaratulaDeCarpetaSinDistinguirMayusculas(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"FOLDER.JPG", "cover.png", "otros.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("imagen"), 0o644); err != nil {
			t.Fatalf("crear %s: %v", name, err)
		}
	}
	cover := findFolderCover(root)
	if filepath.Base(cover) != "cover.png" {
		t.Fatalf("carátula = %q, se esperaba cover.png", cover)
	}
	if got := findFolderCover(t.TempDir()); got != "" {
		t.Fatalf("sin carátula = %q", got)
	}
}

func TestCaratulaEnCarpetaDeDiscoSuperior(t *testing.T) {
	albumDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(albumDir, "cover.jpg"), []byte("imagen"), 0o644); err != nil {
		t.Fatalf("crear carátula: %v", err)
	}
	discDir := filepath.Join(albumDir, "CD 2")
	if err := os.MkdirAll(discDir, 0o755); err != nil {
		t.Fatalf("crear disco: %v", err)
	}
	if got := findAlbumCover(discDir); filepath.Dir(got) != albumDir {
		t.Fatalf("carátula = %q, se esperaba la del álbum", got)
	}
}
