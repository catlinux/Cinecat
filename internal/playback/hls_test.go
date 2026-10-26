package playback

import (
	"net/url"
	"strconv"
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
				"-threads\n" + strconv.Itoa(defaultTranscodeThreads()),
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
		!strings.Contains(args, "-preset\nveryfast") ||
		!strings.Contains(args, "-threads\n"+strconv.Itoa(defaultTranscodeThreads())) {
		t.Fatalf("argumentos sin los defectos: %s", args)
	}
}

func TestHLSTranscodeArgsEligePistaDeAudio(t *testing.T) {
	for _, audio := range []int{0, 1, 2} {
		settings := defaultTranscodeSettings()
		settings.Audio = audio
		args := strings.Join(hlsTranscodeArgs("/tmp/peli.mkv", 0, settings), "\n")
		if !strings.Contains(args, "-map\n0:a:"+strconv.Itoa(audio)+"?") {
			t.Fatalf("audio %d no aparece en los argumentos: %v", audio, args)
		}
	}
}

func TestStreamKeySeparaAudioYCalidad(t *testing.T) {
	base := defaultTranscodeSettings()
	audioChanged := base
	audioChanged.Audio++
	if transcodeKey(base) == transcodeKey(audioChanged) {
		t.Fatal("cambiar el audio debe cambiar la clave")
	}
	if streamKey(base, "auto") == streamKey(base, "720") {
		t.Fatal("cambiar la calidad debe cambiar la clave")
	}
	if streamKey(base, "auto") != streamKey(base, "auto") {
		t.Fatal("la misma selección debe generar la misma clave")
	}
}

func TestSignedQueryPropagaAudioYCalidad(t *testing.T) {
	query := signedQuery("123", "firma", "audio", "2", "q", "720")
	values, err := url.ParseQuery(strings.TrimPrefix(query, "?"))
	if err != nil {
		t.Fatalf("query no válida: %v", err)
	}
	if values.Get("exp") != "123" || values.Get("sig") != "firma" ||
		values.Get("audio") != "2" || values.Get("q") != "720" {
		t.Fatalf("query inesperada: %s", query)
	}
	playlist := signPlaylist([]byte("#EXTM3U\nsegment_00000.ts\n"), query)
	if !strings.Contains(string(playlist), "segment_00000.ts"+query) {
		t.Fatalf("la playlist no propaga la selección: %s", playlist)
	}
}

func TestHLSTranscodeArgsLimitaHilos(t *testing.T) {
	settings := defaultTranscodeSettings()
	settings.Threads = 3
	args := strings.Join(hlsTranscodeArgs("/tmp/peli.mkv", 0, settings), "\n")
	if !strings.Contains(args, "-threads\n3") {
		t.Fatalf("faltan los hilos configurados: %s", args)
	}

	settings.Threads = 0
	args = strings.Join(hlsTranscodeArgs("/tmp/peli.mkv", 0, settings), "\n")
	if !strings.Contains(args, "-threads\n"+strconv.Itoa(defaultTranscodeThreads())) {
		t.Fatalf("faltan los hilos por defecto: %s", args)
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

func TestOptionsThreadsSobrescribeElDefecto(t *testing.T) {
	svc, err := New(Options{
		WorkDir: t.TempDir(),
		Threads: 2,
		Settings: func() TranscodeSettings {
			return TranscodeSettings{MaxHeight: 720, CRF: 22, Preset: "fast"}
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
	if got := svc.currentTranscodeSettings().Threads; got != 2 {
		t.Fatalf("hilos = %d, se esperaban 2", got)
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
	changed = base
	changed.Threads++
	if transcodeKey(base) == transcodeKey(changed) {
		t.Fatal("cambiar los hilos debe cambiar la clave de caché")
	}
}
