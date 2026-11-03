package playback

import (
	"testing"
)

func TestParseCapabilitiesNativas(t *testing.T) {
	caps := ParseCapabilities(" MKV,mp4,h264,hevc10,ac3,eac3,mch,desconocido,dts ")
	if !caps.Native() {
		t.Fatal("con un contenedor declarado deben ser capacidades nativas")
	}
	if !caps.HEVC || !caps.HEVC10 {
		t.Fatalf("hevc10 debe implicar hevc también en nativo: %+v", caps)
	}
	if got, want := caps.QueryValue(), "hevc,hevc10,mkv,mp4,h264,ac3,eac3,dts,mch"; got != want {
		t.Fatalf("QueryValue = %q, se esperaba %q", got, want)
	}
	if again := ParseCapabilities(caps.QueryValue()); again != caps {
		t.Fatalf("QueryValue no es estable: %+v frente a %+v", again, caps)
	}
}

func TestCapacidadesDeNavegadorNoSonNativas(t *testing.T) {
	// Lo que envía hoy la web (incluidos h264 o aac sueltos) nunca activa el
	// modo nativo: sin contenedor declarado la decisión es la de navegador.
	for _, raw := range []string{"", "hevc", "hevc,hevc10,hevcmse,hev1", "h264,aac,ac3,mch"} {
		if ParseCapabilities(raw).Native() {
			t.Fatalf("%q no debe activar el modo nativo", raw)
		}
	}
}

// TestNavegadorSinCambios recorre la matriz de navegador: DecideForAudio debe
// dar lo mismo que DecideFor salvo cuando se pide una pista distinta de la
// que el navegador reproduce en directo, que pasa a HLS por el audio.
func TestNavegadorSinCambios(t *testing.T) {
	h264 := VideoStream{Codec: "h264", Profile: "High", PixelFormat: "yuv420p", BitDepth: 8}
	hevc := VideoStream{Codec: "hevc", CodecTag: "hev1", Profile: "Main", PixelFormat: "yuv420p", BitDepth: 8}
	infos := []*FileInfo{
		nil,
		{Video: []VideoStream{h264}, Audio: []AudioStream{{Codec: "aac", Channels: 2, Default: true}}},
		{Video: []VideoStream{h264}, Audio: []AudioStream{{Codec: "aac", Channels: 6, Default: true}}},
		{Video: []VideoStream{h264}, Audio: []AudioStream{{Codec: "ac3", Channels: 6, Default: true}, {Codec: "aac", Channels: 2}}},
		{Video: []VideoStream{hevc}, Audio: []AudioStream{{Codec: "aac", Channels: 2, Default: true}}},
	}
	browserCaps := []Capabilities{{}, ParseCapabilities("hevc,hevc10,hev1"), ParseCapabilities("hevcmse")}
	for _, path := range []string{"/media/a.mp4", "/media/a.mkv", "/media/a.avi"} {
		for _, info := range infos {
			for _, caps := range browserCaps {
				want, wantErr := DecideFor(path, info, caps)
				for _, audio := range []int{-1, 0, 1} {
					got, reason, err := DecideForAudio(path, info, caps, audio)
					if got != want || (err == nil) != (wantErr == nil) {
						t.Fatalf("%s %+v %+v audio %d: %q frente a %q", path, info, caps, audio, got, want)
					}
					if (got == ModeDirect) != (reason == ReasonNone) {
						t.Fatalf("motivo %q incoherente con el modo %q", reason, got)
					}
				}
			}
		}
	}
}

// TestNavegadorPideOtraPistaDeAudio cubre un MP4 con dos pistas compatibles:
// la pista por defecto va en directo y cualquier otra obliga a pasar por HLS,
// porque el navegador no puede cambiar de pista en un archivo servido tal cual.
func TestNavegadorPideOtraPistaDeAudio(t *testing.T) {
	h264 := VideoStream{Codec: "h264", Profile: "High", PixelFormat: "yuv420p", BitDepth: 8}
	info := &FileInfo{
		Video: []VideoStream{h264},
		Audio: []AudioStream{
			{Codec: "aac", Channels: 2, Language: "fr", Default: true},
			{Codec: "aac", Channels: 2, Language: "es"},
		},
	}
	tests := []struct {
		audio  int
		mode   Mode
		reason Reason
	}{
		{-1, ModeDirect, ReasonNone},
		{0, ModeDirect, ReasonNone},
		{1, ModeRemux, ReasonAudio},
		{7, ModeDirect, ReasonNone},
	}
	for _, test := range tests {
		mode, reason, err := DecideForAudio("/media/pelicula.mp4", info, Capabilities{}, test.audio)
		if err != nil || mode != test.mode || reason != test.reason {
			t.Fatalf("audio %d: %q/%q (%v), se esperaba %q/%q", test.audio, mode, reason, err, test.mode, test.reason)
		}
	}
}

func TestAudioChannelsFor(t *testing.T) {
	info := &FileInfo{Audio: []AudioStream{
		{Codec: "aac", Channels: 2},
		{Codec: "ac3", Channels: 6},
		{Codec: "eac3", Channels: 8},
	}}
	nativeMCH := ParseCapabilities("mkv,h264,aac,ac3,mch")
	native := ParseCapabilities("mkv,h264,aac,ac3")
	browserMCH := ParseCapabilities("h264,aac,mch")
	tests := []struct {
		name  string
		caps  Capabilities
		info  *FileInfo
		audio int
		want  int
	}{
		{name: "nativo mch 6 canales", caps: nativeMCH, info: info, audio: 1, want: 6},
		{name: "nativo mch 8 canales", caps: nativeMCH, info: info, audio: 2, want: 6},
		{name: "nativo mch estéreo", caps: nativeMCH, info: info, audio: 0, want: 2},
		{name: "nativo sin mch", caps: native, info: info, audio: 1, want: 2},
		{name: "navegador con mch", caps: browserMCH, info: info, audio: 1, want: 2},
		{name: "sin ficha", caps: nativeMCH, info: nil, audio: 1, want: 2},
		{name: "índice negativo usa la primera", caps: nativeMCH,
			info: &FileInfo{Audio: []AudioStream{{Codec: "ac3", Channels: 6}}}, audio: -1, want: 6},
		{name: "índice fuera de rango", caps: nativeMCH, info: info, audio: 9, want: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := AudioChannelsFor(test.caps, test.info, test.audio); got != test.want {
				t.Fatalf("canales = %d, se esperaban %d", got, test.want)
			}
		})
	}
}

func TestAudioCopyFor(t *testing.T) {
	info := &FileInfo{Audio: []AudioStream{
		{Codec: "aac", Channels: 2},
		{Codec: "ac3", Channels: 6},
		{Codec: "eac3", Channels: 8},
		{Codec: "dts", Channels: 6},
	}}
	nativeAC3 := ParseCapabilities("mkv,h264,ac3")
	nativeEAC3 := ParseCapabilities("mkv,h264,eac3")
	nativeBoth := ParseCapabilities("mkv,h264,ac3,eac3")
	browser := ParseCapabilities("h264,aac,ac3,eac3")
	tests := []struct {
		name  string
		caps  Capabilities
		info  *FileInfo
		audio int
		want  bool
	}{
		{name: "nativo ac3 con pista ac3", caps: nativeAC3, info: info, audio: 1, want: true},
		{name: "nativo eac3 con pista eac3", caps: nativeEAC3, info: info, audio: 2, want: true},
		{name: "nativo sin ac3", caps: ParseCapabilities("mkv,h264,aac"), info: info, audio: 1, want: false},
		{name: "nativo ac3 con pista eac3", caps: nativeAC3, info: info, audio: 2, want: false},
		{name: "pista dts nunca copia", caps: nativeBoth, info: info, audio: 3, want: false},
		{name: "web nunca copia", caps: browser, info: info, audio: 1, want: false},
		{name: "sin ficha", caps: nativeBoth, info: nil, audio: 1, want: false},
		{name: "índice negativo usa la primera", caps: nativeAC3,
			info:  &FileInfo{Audio: []AudioStream{{Codec: "ac3", Channels: 6}}},
			audio: -1, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := AudioCopyFor(test.caps, test.info, test.audio); got != test.want {
				t.Fatalf("AudioCopyFor = %v, se esperaba %v", got, test.want)
			}
		})
	}
}

func TestDecisionNativa(t *testing.T) {
	h264 := VideoStream{Codec: "h264", Profile: "High", PixelFormat: "yuv420p", BitDepth: 8}
	h264Hi10 := VideoStream{Codec: "h264", Profile: "High 10", PixelFormat: "yuv420p10le", BitDepth: 10}
	hevc := VideoStream{Codec: "hevc", CodecTag: "hev1", Profile: "Main", PixelFormat: "yuv420p", BitDepth: 8}
	hevc10 := VideoStream{Codec: "hevc", Profile: "Main 10", PixelFormat: "yuv420p10le", BitDepth: 10}
	av1 := VideoStream{Codec: "av1", PixelFormat: "yuv420p", BitDepth: 8}
	stereo := AudioStream{Codec: "aac", Channels: 2, Default: true}
	aac51 := AudioStream{Codec: "aac", Channels: 6, Default: true}
	ac3 := AudioStream{Codec: "ac3", Channels: 6, Default: true}
	dts := AudioStream{Codec: "dts", Channels: 6, Default: true}

	tv := ParseCapabilities("mkv,mp4,h264,hevc,hevc10,aac,mp3,ac3,eac3")
	tests := []struct {
		name   string
		path   string
		info   *FileInfo
		caps   Capabilities
		audio  int
		want   Mode
		reason Reason
	}{
		{name: "mkv hevc ac3 directo", path: "/m/a.mkv", info: &FileInfo{Video: []VideoStream{hevc}, Audio: []AudioStream{ac3}}, caps: tv, audio: -1, want: ModeDirect},
		{name: "mkv hevc 10 bits directo", path: "/m/a.mkv", info: &FileInfo{Video: []VideoStream{hevc10}, Audio: []AudioStream{stereo}}, caps: tv, audio: -1, want: ModeDirect},
		{name: "mp4 hev1 directo sin token hev1", path: "/m/a.mp4", info: &FileInfo{Video: []VideoStream{hevc}, Audio: []AudioStream{stereo}}, caps: tv, audio: -1, want: ModeDirect},
		{name: "mkv sin token mkv", path: "/m/a.mkv", info: &FileInfo{Video: []VideoStream{h264}, Audio: []AudioStream{stereo}}, caps: ParseCapabilities("mp4,h264,aac"), audio: -1, want: ModeRemux, reason: ReasonContainer},
		{name: "avi siempre hls", path: "/m/a.avi", info: &FileInfo{Video: []VideoStream{h264}, Audio: []AudioStream{stereo}}, caps: tv, audio: -1, want: ModeRemux, reason: ReasonContainer},
		{name: "dts sin token", path: "/m/a.mkv", info: &FileInfo{Video: []VideoStream{h264}, Audio: []AudioStream{dts}}, caps: tv, audio: -1, want: ModeRemux, reason: ReasonAudio},
		{name: "aac 5.1 sin mch", path: "/m/a.mp4", info: &FileInfo{Video: []VideoStream{h264}, Audio: []AudioStream{aac51}}, caps: tv, audio: -1, want: ModeRemux, reason: ReasonAudio},
		{name: "aac 5.1 con mch", path: "/m/a.mp4", info: &FileInfo{Video: []VideoStream{h264}, Audio: []AudioStream{aac51}}, caps: ParseCapabilities("mp4,h264,aac,mch"), audio: -1, want: ModeDirect},
		{name: "h264 high 10 sin token", path: "/m/a.mkv", info: &FileInfo{Video: []VideoStream{h264Hi10}, Audio: []AudioStream{stereo}}, caps: tv, audio: -1, want: ModeRemux, reason: ReasonVideo},
		{name: "av1 sin token", path: "/m/a.mkv", info: &FileInfo{Video: []VideoStream{av1}, Audio: []AudioStream{stereo}}, caps: tv, audio: -1, want: ModeRemux, reason: ReasonVideo},
		{name: "av1 con token", path: "/m/a.mkv", info: &FileInfo{Video: []VideoStream{av1}, Audio: []AudioStream{stereo}}, caps: ParseCapabilities("mkv,av1,aac"), audio: -1, want: ModeDirect},
		{name: "pista elegida no admitida", path: "/m/a.mkv", info: &FileInfo{Video: []VideoStream{h264}, Audio: []AudioStream{ac3, dts}}, caps: tv, audio: 1, want: ModeRemux, reason: ReasonAudio},
		{name: "pista elegida admitida", path: "/m/a.mkv", info: &FileInfo{Video: []VideoStream{h264}, Audio: []AudioStream{dts, {Codec: "ac3", Channels: 6}}}, caps: tv, audio: 1, want: ModeDirect},
		{name: "pista fuera de rango usa la por defecto", path: "/m/a.mkv", info: &FileInfo{Video: []VideoStream{h264}, Audio: []AudioStream{ac3}}, caps: tv, audio: 7, want: ModeDirect},
		{name: "sin audio directo", path: "/m/a.mkv", info: &FileInfo{Video: []VideoStream{h264}}, caps: tv, audio: -1, want: ModeDirect},
		{name: "mkv sin ficha hls", path: "/m/a.mkv", info: nil, caps: tv, audio: -1, want: ModeRemux, reason: ReasonVideo},
		{name: "mp4 sin ficha directo", path: "/m/a.mp4", info: nil, caps: tv, audio: -1, want: ModeDirect},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, reason, err := DecideForAudio(test.path, test.info, test.caps, test.audio)
			if err != nil {
				t.Fatalf("DecideForAudio: %v", err)
			}
			if got != test.want || reason != test.reason {
				t.Fatalf("= %q (%q), se esperaba %q (%q)", got, reason, test.want, test.reason)
			}
		})
	}

	if _, _, err := DecideForAudio("/m/notas.txt", nil, tv, -1); err == nil {
		t.Fatal("un contenedor no admitido debe seguir dando error en nativo")
	}
}

func TestCopiaHEVCNativa(t *testing.T) {
	hevc := VideoStream{Codec: "hevc", Profile: "Main", PixelFormat: "yuv420p", BitDepth: 8, Height: 1080}
	hevc10 := VideoStream{Codec: "hevc", Profile: "Main 10", PixelFormat: "yuv420p10le", BitDepth: 10, Height: 1080}
	mkv := func(video VideoStream) *FileInfo {
		return &FileInfo{FileName: "/m/a.mkv", Container: "matroska,webm", Video: []VideoStream{video}}
	}
	if CanCopy(mkv(hevc), ParseCapabilities("hevc"), 0) {
		t.Fatal("la web sin hevcmse no debe copiar HEVC (sin cambios)")
	}
	if !CanCopy(mkv(hevc), ParseCapabilities("mkv,hevc"), 0) {
		t.Fatal("una app con hevc debe poder copiar HEVC a TS")
	}
	if CanCopy(mkv(hevc10), ParseCapabilities("mkv,hevc"), 0) {
		t.Fatal("sin hevc10 no se copia HEVC de 10 bits")
	}
	if !CanCopy(mkv(hevc10), ParseCapabilities("mkv,hevc10"), 0) {
		t.Fatal("con hevc10 se copia HEVC de 10 bits")
	}
}
