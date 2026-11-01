package playback

import (
	"context"
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

func videoFilterArg(t *testing.T, args []string) string {
	t.Helper()
	for index := 0; index+1 < len(args); index++ {
		if args[index] == "-vf" {
			return args[index+1]
		}
	}
	t.Fatalf("los argumentos no llevan -vf: %v", args)
	return ""
}

func TestHLSTranscodeArgsDesentrelazadoYHDR(t *testing.T) {
	settings := TranscodeSettings{MaxHeight: 720, CRF: 20, Preset: "veryfast", Threads: 4}
	scale := `scale=-2:trunc(min(720\,ih)/2)*2`
	interlaced := &VideoStream{FieldOrder: "tt"}
	hdr := &VideoStream{ColorTransfer: "smpte2084"}
	both := &VideoStream{FieldOrder: "bb", ColorTransfer: "arib-std-b67"}

	tests := []struct {
		name       string
		video      *VideoStream
		hdrFilters bool
		want       string
	}{
		{name: "progresivo SDR", video: &VideoStream{FieldOrder: "progressive"}, want: scale},
		{name: "entrelazado", video: interlaced, want: deinterlaceVideoFilter + "," + scale},
		{name: "HDR con filtros", video: hdr, hdrFilters: true, want: hdrToSDRVideoFilter + "," + scale},
		{name: "HDR sin filtros", video: hdr, want: scale},
		{name: "entrelazado HDR con filtros", video: both, hdrFilters: true,
			want: deinterlaceVideoFilter + "," + hdrToSDRVideoFilter + "," + scale},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := hlsTranscodeArgsWithVideo("/tmp/peli.mkv", 0, settings, test.video, test.hdrFilters)
			if got := videoFilterArg(t, args); got != test.want {
				t.Fatalf("filtro = %q, se esperaba %q", got, test.want)
			}
		})
	}

	// Sin ficha de vídeo, la salida debe ser exactamente la de siempre.
	if got, want := hlsTranscodeArgsWithVideo("/tmp/peli.mkv", 0, settings, nil, false),
		hlsTranscodeArgs("/tmp/peli.mkv", 0, settings); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("los argumentos SDR cambiaron:\n%q\n%q", got, want)
	}
}

func TestHardwareCPUFiltraAntesDeSubirALaGPU(t *testing.T) {
	settings := TranscodeSettings{MaxHeight: 720, CRF: 20, Preset: "veryfast", Threads: 4}
	args := hlsHardwareArgsWithVideo("/tmp/peli.mkv", 0, settings, false, "/dev/dri/renderD128",
		&VideoStream{FieldOrder: "tt"}, false)
	want := deinterlaceVideoFilter + `,scale=-2:trunc(min(720\,ih)/2)*2,format=nv12,hwupload`
	if got := videoFilterArg(t, args); got != want {
		t.Fatalf("filtro hardware = %q, se esperaba %q", got, want)
	}
}

func TestHLSCopyArgsNoLlevaFiltros(t *testing.T) {
	args := hlsCopyArgs("/tmp/peli.mkv", 0, []float64{0, 6, 12}, defaultTranscodeSettings())
	for _, arg := range args {
		if arg == "-vf" {
			t.Fatalf("la copia no debe llevar filtros: %v", args)
		}
	}
}

func TestFilterListHasZscaleYTonemap(t *testing.T) {
	list := []byte(`Filters:
 ... zscale            V->V       Apply zscale filter.
 ... tonemap           V->V       Apply tonemap filter.
`)
	if !filterListHas(list, "zscale") || !filterListHas(list, "tonemap") {
		t.Fatal("no se detectaron los filtros HDR")
	}
	if filterListHas(list, "yadif") {
		t.Fatal("se detectó un filtro que no está en la lista")
	}
}

func TestHDRFiltersSeCompruebanUnaSolaVez(t *testing.T) {
	for _, available := range []bool{true, false} {
		calls := 0
		svc, err := New(Options{
			WorkDir: t.TempDir(),
			HDRFilterProbe: func(context.Context) (bool, error) {
				calls++
				return available, nil
			},
		})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		for i := 0; i < 3; i++ {
			if got := svc.hdrFiltersAvailable(context.Background()); got != available {
				t.Fatalf("disponibles = %v, se esperaba %v", got, available)
			}
		}
		if calls != 1 {
			t.Fatalf("comprobaciones = %d, se esperaba 1", calls)
		}
		if err := svc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
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
