package playback

import (
	"strings"
	"sync"
	"testing"
)

func TestHLSTranscodeArgsConAjustes(t *testing.T) {
	tests := []struct {
		name     string
		settings TranscodeSettings
		filter   string
		preset   string
		crf      string
	}{
		{
			name:     "480p calidad alta",
			settings: TranscodeSettings{MaxHeight: 480, CRF: 18, Preset: "fast"},
			filter:   `scale=-2:trunc(min(480\,ih)/2)*2`,
			preset:   "fast",
			crf:      "18",
		},
		{
			name:     "720p calidad media",
			settings: TranscodeSettings{MaxHeight: 720, CRF: 23, Preset: "faster"},
			filter:   `scale=-2:trunc(min(720\,ih)/2)*2`,
			preset:   "faster",
			crf:      "23",
		},
		{
			name:     "1080p calidad baja",
			settings: TranscodeSettings{MaxHeight: 1080, CRF: 28, Preset: "veryfast"},
			filter:   `scale=-2:trunc(min(1080\,ih)/2)*2`,
			preset:   "veryfast",
			crf:      "28",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := hlsTranscodeArgs("/tmp/peli.mkv", 0, test.settings)
			joined := strings.Join(args, "\n")
			for _, expected := range []string{
				"-vf\n" + test.filter,
				"-preset\n" + test.preset,
				"-crf\n" + test.crf,
			} {
				if !strings.Contains(joined, expected) {
					t.Fatalf("faltan %q en los argumentos: %v", expected, args)
				}
			}
		})
	}
}

func TestTranscodeSettingsInvalidosUsanLosDefectos(t *testing.T) {
	settings := normalizeTranscodeSettings(TranscodeSettings{
		MaxHeight: 999,
		CRF:       99,
		Preset:    "ultrafast",
	})
	if settings != defaultTranscodeSettings() {
		t.Fatalf("ajustes normalizados = %+v, se esperaban los defectos", settings)
	}

	args := strings.Join(hlsTranscodeArgs("/tmp/peli.mkv", 0, settings), "\n")
	if !strings.Contains(args, "-vf\n"+`scale=-2:trunc(min(1080\,ih)/2)*2`) ||
		!strings.Contains(args, "-crf\n20") ||
		!strings.Contains(args, "-preset\nveryfast") {
		t.Fatalf("argumentos sin los defectos: %s", args)
	}
}

func TestServiceLeeAjustesEnCaliente(t *testing.T) {
	var mu sync.Mutex
	settings := TranscodeSettings{MaxHeight: 480, CRF: 18, Preset: "fast"}
	svc, err := New(Options{
		WorkDir: t.TempDir(),
		Settings: func() TranscodeSettings {
			mu.Lock()
			defer mu.Unlock()
			return settings
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	if got := svc.currentTranscodeSettings(); got != settings {
		t.Fatalf("ajustes iniciales = %+v", got)
	}

	mu.Lock()
	settings = TranscodeSettings{MaxHeight: 720, CRF: 28, Preset: "veryfast"}
	mu.Unlock()
	if got := svc.currentTranscodeSettings(); got != settings {
		t.Fatalf("ajustes tras el cambio = %+v", got)
	}
}

func TestTranscodeKeySeparaCalidades(t *testing.T) {
	base := defaultTranscodeSettings()
	if transcodeKey(base) != transcodeKey(base) {
		t.Fatal("la misma configuración debe generar la misma clave")
	}
	changed := base
	changed.MaxHeight = 720
	if transcodeKey(base) == transcodeKey(changed) {
		t.Fatal("cambiar la altura debe cambiar la clave de caché")
	}
	changed = base
	changed.CRF = 18
	if transcodeKey(base) == transcodeKey(changed) {
		t.Fatal("cambiar el CRF debe cambiar la clave de caché")
	}
	changed = base
	changed.Preset = "fast"
	if transcodeKey(base) == transcodeKey(changed) {
		t.Fatal("cambiar el preset debe cambiar la clave de caché")
	}
}
