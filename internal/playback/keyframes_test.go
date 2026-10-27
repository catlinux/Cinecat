package playback

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func ebmlIDBytes(id uint32) []byte {
	length := 1
	switch {
	case id > 0xFFFFFF:
		length = 4
	case id > 0xFFFF:
		length = 3
	case id > 0xFF:
		length = 2
	}
	result := make([]byte, length)
	for index := length - 1; index >= 0; index-- {
		result[index] = byte(id)
		id >>= 8
	}
	return result
}

func ebmlSizeBytes(value int) []byte {
	length := 1
	for value >= (1<<(7*length))-1 {
		length++
	}
	result := make([]byte, length)
	for index := length - 1; index >= 0; index-- {
		result[index] = byte(value)
		value >>= 8
	}
	result[0] |= byte(1) << (8 - length)
	return result
}

func testElement(id uint32, parts ...[]byte) []byte {
	payload := bytes.Join(parts, nil)
	result := append(ebmlIDBytes(id), ebmlSizeBytes(len(payload))...)
	return append(result, payload...)
}

func ebmlUint(value uint64) []byte {
	if value == 0 {
		return []byte{0}
	}
	length := 1
	for value >= (uint64(1) << (8 * uint(length))) {
		length++
	}
	result := make([]byte, length)
	for index := length - 1; index >= 0; index-- {
		result[index] = byte(value)
		value >>= 8
	}
	return result
}

func ebmlFixedUint(value uint64, length int) []byte {
	result := make([]byte, length)
	for index := length - 1; index >= 0; index-- {
		result[index] = byte(value)
		value >>= 8
	}
	return result
}

type testCue struct {
	time  uint64
	track uint64
}

func testSeekHead(cuesPosition int) []byte {
	seek := testElement(ebmlSeek,
		testElement(ebmlSeekID, ebmlIDBytes(ebmlCues)),
		testElement(ebmlSeekPosition, ebmlFixedUint(uint64(cuesPosition), 4)),
	)
	return testElement(ebmlSeekHead, seek)
}

func testTrackEntry(number, kind uint64) []byte {
	return testElement(ebmlTrackEntry,
		testElement(ebmlTrackNumber, ebmlUint(number)),
		testElement(ebmlTrackType, ebmlUint(kind)),
	)
}

func testMKVParts(scale uint64, cues []testCue, videoTrack, audioTrack uint64) ([]byte, []byte) {
	seekHead := testSeekHead(0)
	cuesPosition := len(seekHead)
	seekHead = testSeekHead(cuesPosition)
	info := testElement(ebmlInfo, testElement(ebmlTimestampScale, ebmlUint(scale)))
	tracks := testElement(ebmlTracks,
		testTrackEntry(videoTrack, 1),
		testTrackEntry(audioTrack, 2),
	)
	cueParts := make([][]byte, 0, len(cues))
	for _, cue := range cues {
		cueParts = append(cueParts, testElement(ebmlCuePoint,
			testElement(ebmlCueTime, ebmlUint(cue.time)),
			testElement(ebmlCueTrackPos,
				testElement(ebmlCueTrack, ebmlUint(cue.track)),
			),
		))
	}
	cuesElement := testElement(ebmlCues, cueParts...)
	return testElement(ebmlHeader), bytes.Join([][]byte{seekHead, cuesElement, info, tracks}, nil)
}

func testMKV(scale uint64, cues []testCue, videoTrack, audioTrack uint64) []byte {
	header, payload := testMKVParts(scale, cues, videoTrack, audioTrack)
	return append(header, testElement(ebmlSegment, payload)...)
}

func testMKVCuesDespuesDelCluster(scale uint64, cues []testCue, videoTrack, audioTrack uint64) []byte {
	seekHead := testSeekHead(0)
	info := testElement(ebmlInfo, testElement(ebmlTimestampScale, ebmlUint(scale)))
	tracks := testElement(ebmlTracks,
		testTrackEntry(videoTrack, 1),
		testTrackEntry(audioTrack, 2),
	)
	cueParts := make([][]byte, 0, len(cues))
	for _, cue := range cues {
		cueParts = append(cueParts, testElement(ebmlCuePoint,
			testElement(ebmlCueTime, ebmlUint(cue.time)),
			testElement(ebmlCueTrackPos, testElement(ebmlCueTrack, ebmlUint(cue.track))),
		))
	}
	cluster := testElement(ebmlCluster)
	cuesElement := testElement(ebmlCues, cueParts...)
	cuesPosition := len(seekHead) + len(info) + len(tracks) + len(cluster)
	seekHead = testSeekHead(cuesPosition)
	payload := bytes.Join([][]byte{seekHead, info, tracks, cluster, cuesElement}, nil)
	return append(testElement(ebmlHeader), testElement(ebmlSegment, payload)...)
}

func testMKVUnknownSize(scale uint64, cues []testCue, videoTrack, audioTrack uint64) []byte {
	header, payload := testMKVParts(scale, cues, videoTrack, audioTrack)
	unknownSize := []byte{0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	segment := append(ebmlIDBytes(ebmlSegment), unknownSize...)
	return append(append(header, segment...), payload...)
}

func testMKVNoSeekHead(cues []testCue) []byte {
	info := testElement(ebmlInfo, testElement(ebmlTimestampScale, ebmlUint(1_000_000)))
	tracks := testElement(ebmlTracks, testTrackEntry(1, 1), testTrackEntry(2, 2))
	cueParts := make([][]byte, 0, len(cues))
	for _, cue := range cues {
		cueParts = append(cueParts, testElement(ebmlCuePoint,
			testElement(ebmlCueTime, ebmlUint(cue.time)),
			testElement(ebmlCueTrackPos, testElement(ebmlCueTrack, ebmlUint(cue.track))),
		))
	}
	segment := testElement(ebmlSegment, info, tracks, testElement(ebmlCues, cueParts...))
	return append(testElement(ebmlHeader), segment...)
}

func writeKeyframeTestFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("escribir archivo: %v", err)
	}
	return path
}

func TestMatroskaKeyframes(t *testing.T) {
	data := testMKV(1_000_000, []testCue{
		{2500, 1},
		{0, 1},
		{1000, 2},
		{5000, 1},
		{2500, 1},
	}, 1, 2)
	path := writeKeyframeTestFile(t, "peli.mp4", data)
	got, err := MatroskaKeyframes(path)
	if err != nil {
		t.Fatalf("MatroskaKeyframes: %v", err)
	}
	want := []float64{0, 2.5, 5}
	if len(got) != len(want) {
		t.Fatalf("claves = %v, se esperaba %v", got, want)
	}
	for index := range want {
		if math.Abs(got[index]-want[index]) > 1e-9 {
			t.Fatalf("claves = %v, se esperaba %v", got, want)
		}
	}
}

func TestMatroskaKeyframesCuesDespuesDelCluster(t *testing.T) {
	path := writeKeyframeTestFile(t, "real.mkv",
		testMKVCuesDespuesDelCluster(1_000_000, []testCue{{0, 1}, {2000, 1}}, 1, 2))
	got, err := MatroskaKeyframes(path)
	if err != nil {
		t.Fatalf("MatroskaKeyframes: %v", err)
	}
	if !equalFloats(got, []float64{0, 2}) {
		t.Fatalf("claves = %v", got)
	}
}

func TestMatroskaKeyframesTimestampScaleUnknownSize(t *testing.T) {
	data := testMKV(2_000_000, []testCue{{0, 1}, {1000, 1}, {2000, 1}}, 1, 2)
	path := writeKeyframeTestFile(t, "escala.mkv", data)
	got, err := MatroskaKeyframes(path)
	if err != nil {
		t.Fatalf("MatroskaKeyframes: %v", err)
	}
	if len(got) != 3 || math.Abs(got[1]-2) > 1e-9 || math.Abs(got[2]-4) > 1e-9 {
		t.Fatalf("claves con escala = %v", got)
	}

	unknownPath := writeKeyframeTestFile(t, "desconocido.mkv",
		testMKVUnknownSize(1_000_000, []testCue{{0, 1}, {2000, 1}}, 1, 2))
	unknown, err := MatroskaKeyframes(unknownPath)
	if err != nil {
		t.Fatalf("MKV con Segment de tamaño desconocido: %v", err)
	}
	if !equalFloats(unknown, []float64{0, 2}) {
		t.Fatalf("claves con Segment desconocido = %v", unknown)
	}
}

func TestMatroskaKeyframesErrores(t *testing.T) {
	for name, data := range map[string][]byte{
		"no EBML":     []byte("esto no es EBML"),
		"sin Segment": testElement(ebmlHeader),
		"truncado":    testMKV(1_000_000, []testCue{{0, 1}, {2000, 1}}, 1, 2)[:20],
		"tamaño absurdo": append(
			append(testElement(ebmlHeader), ebmlIDBytes(ebmlSegment)...),
			0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFE, 'x',
		),
	} {
		t.Run(name, func(t *testing.T) {
			path := writeKeyframeTestFile(t, "error.mkv", data)
			if _, err := MatroskaKeyframes(path); err == nil {
				t.Fatal("se esperaba un error")
			}
		})
	}

	noSeek := testMKVNoSeekHead([]testCue{{0, 1}, {2000, 1}})
	if _, err := MatroskaKeyframes(writeKeyframeTestFile(t, "sin-seek.mkv", noSeek)); !errors.Is(err, ErrNoKeyframeIndex) {
		t.Fatalf("sin SeekHead: %v", err)
	}

	audioOnly := testMKV(1_000_000, []testCue{{0, 2}, {2000, 2}}, 1, 2)
	if _, err := MatroskaKeyframes(writeKeyframeTestFile(t, "solo-audio.mkv", audioOnly)); !errors.Is(err, ErrNoKeyframeIndex) {
		t.Fatalf("Cues sin vídeo: %v", err)
	}
}

func TestSegmentBoundaries(t *testing.T) {
	even := make([]float64, 0, 16)
	for value := 0; value <= 30; value += 2 {
		even = append(even, float64(value))
	}
	if got := SegmentBoundaries(even, 30); !equalFloats(got, []float64{0, 6, 12, 18, 24}) {
		t.Fatalf("claves cada 2 s: %v", got)
	}

	if got := SegmentBoundaries([]float64{0, 10.43, 20.86, 31.29}, 31.29); !equalFloats(got, []float64{0, 10.43, 20.86}) {
		t.Fatalf("claves cada 10,43 s: %v", got)
	}

	if got := SegmentBoundaries([]float64{0, 4, 6, 6.1, 12, 13, 20}, 20); !equalFloats(got, []float64{0, 12}) {
		t.Fatalf("claves irregulares: %v", got)
	}

	if got := SegmentBoundaries([]float64{0, 5.9, 6, 6.1, 6.2, 12}, 12); !equalFloats(got, []float64{0}) {
		t.Fatalf("claves muy juntas: %v", got)
	}

	// El límite en 24 dejaría un último segmento de 0,6 s: se une al anterior
	// y debe coincidir con el número de duraciones.
	bounds := SegmentBoundaries(even, 24.6)
	if !equalFloats(bounds, []float64{0, 6, 12, 18}) {
		t.Fatalf("resto corto: %v", bounds)
	}
	if durations := BoundaryDurations(bounds, 24.6); len(durations) != len(bounds) {
		t.Fatalf("límites %v y duraciones %v no coinciden", bounds, durations)
	}
}

func TestBoundaryDurations(t *testing.T) {
	if got := BoundaryDurations([]float64{0, 6, 12, 20}, 20.5); !equalFloats(got, []float64{6, 6, 8.5}) {
		t.Fatalf("duraciones: %v", got)
	}
	if got := BoundaryDurations([]float64{0, 6, 12, 20}, 22); !equalFloats(got, []float64{6, 6, 8, 2}) {
		t.Fatalf("duraciones: %v", got)
	}
	if got := BoundaryDurations(nil, 22); len(got) != 0 {
		t.Fatalf("sin límites: %v", got)
	}
}

func equalFloats(left, right []float64) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if math.Abs(left[index]-right[index]) > 1e-6 {
			return false
		}
	}
	return true
}

func waitKeyframeLookup(t *testing.T, index *KeyframeIndex, path string) (*KeyframeEntry, error) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		entry, err := index.Lookup(path)
		if !errors.Is(err, ErrKeyframesPending) {
			return entry, err
		}
		if time.Now().After(deadline) {
			t.Fatal("el índice de claves no terminó en segundo plano")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestKeyframeIndexCuesYCacheEnDisco(t *testing.T) {
	dir := t.TempDir()
	path := writeKeyframeTestFile(t, "cues.mkv", testMKV(1_000_000, []testCue{{0, 1}, {2000, 1}}, 1, 2))
	index := NewKeyframeIndex(dir)
	defer index.Close()

	entry, err := index.Lookup(path)
	if err != nil {
		t.Fatalf("Lookup de Cues: %v", err)
	}
	if entry.Source != "cues" || !equalFloats(entry.Keyframes, []float64{0, 2}) {
		t.Fatalf("entrada de Cues: %+v", entry)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(files) != 1 {
		t.Fatalf("archivos de caché: %v, %v", files, err)
	}
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("leer caché: %v", err)
	}
	var persisted KeyframeEntry
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("JSON de caché: %v", err)
	}
	if persisted.Version != keyframeCacheVersion || persisted.Source != "cues" {
		t.Fatalf("caché inesperada: %+v", persisted)
	}

	if err := os.WriteFile(path, append(testMKV(1_000_000, []testCue{{0, 1}, {3000, 1}}, 1, 2), 0), 0o644); err != nil {
		t.Fatalf("cambiar archivo: %v", err)
	}
	entry, err = index.Lookup(path)
	if err != nil {
		t.Fatalf("Lookup tras cambiar tamaño: %v", err)
	}
	if !equalFloats(entry.Keyframes, []float64{0, 3}) {
		t.Fatalf("caché no invalidada: %+v", entry)
	}

	stamp := time.Now().Add(2 * time.Hour)
	if err := os.Chtimes(path, stamp, stamp); err != nil {
		t.Fatalf("cambiar fecha: %v", err)
	}
	entry, err = index.Lookup(path)
	if err != nil || !equalFloats(entry.Keyframes, []float64{0, 3}) {
		t.Fatalf("caché no invalidada por fecha: %+v, %v", entry, err)
	}
}

func TestKeyframeIndexSoloMemoriaSinDirectorio(t *testing.T) {
	path := writeKeyframeTestFile(t, "memoria.mkv", testMKV(1_000_000, []testCue{{0, 1}, {2000, 1}}, 1, 2))
	index := NewKeyframeIndex("")
	defer index.Close()
	entry, err := index.Lookup(path)
	if err != nil || entry.Source != "cues" || !equalFloats(entry.Keyframes, []float64{0, 2}) {
		t.Fatalf("caché en memoria: %+v, %v", entry, err)
	}
	if entry, err = index.Lookup(path); err != nil || entry.Source != "cues" {
		t.Fatalf("segunda lectura en memoria: %+v, %v", entry, err)
	}
}

func TestKeyframeIndexFfprobePendienteYFalloRecordado(t *testing.T) {
	path := writeKeyframeTestFile(t, "video.mp4", []byte("no es EBML"))
	index := NewKeyframeIndex(t.TempDir())
	defer index.Close()

	var calls int32
	index.probe = func(context.Context, string) ([]float64, error) {
		atomic.AddInt32(&calls, 1)
		return []float64{0, 2, 4}, nil
	}
	if _, err := index.Lookup(path); !errors.Is(err, ErrKeyframesPending) {
		t.Fatalf("primer Lookup: %v", err)
	}
	entry, err := waitKeyframeLookup(t, index, path)
	if err != nil || entry.Source != "ffprobe" {
		t.Fatalf("Lookup ffprobe: %+v, %v", entry, err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("llamadas a probe = %d", calls)
	}

	failPath := writeKeyframeTestFile(t, "fallo.mp4", []byte("tampoco EBML"))
	var failCalls int32
	index.probe = func(context.Context, string) ([]float64, error) {
		atomic.AddInt32(&failCalls, 1)
		return nil, errors.New("sin claves")
	}
	if _, err := index.Lookup(failPath); !errors.Is(err, ErrKeyframesPending) {
		t.Fatalf("primer Lookup fallo: %v", err)
	}
	if _, err := waitKeyframeLookup(t, index, failPath); !errors.Is(err, ErrNoKeyframeIndex) {
		t.Fatalf("fallo esperado: %v", err)
	}
	if _, err := index.Lookup(failPath); !errors.Is(err, ErrNoKeyframeIndex) {
		t.Fatalf("fallo no recordado: %v", err)
	}
	if atomic.LoadInt32(&failCalls) != 1 {
		t.Fatalf("reintentos de probe = %d", failCalls)
	}
}

func TestKeyframeIndexSinDuplicadosEnCola(t *testing.T) {
	path := writeKeyframeTestFile(t, "cola.mp4", []byte("no EBML"))
	index := NewKeyframeIndex(t.TempDir())
	defer index.Close()
	started := make(chan struct{})
	release := make(chan struct{})
	var calls int32
	var once sync.Once
	index.probe = func(context.Context, string) ([]float64, error) {
		atomic.AddInt32(&calls, 1)
		once.Do(func() { close(started) })
		<-release
		return []float64{0, 2}, nil
	}
	if _, err := index.Lookup(path); !errors.Is(err, ErrKeyframesPending) {
		t.Fatalf("primer Lookup: %v", err)
	}
	<-started
	if _, err := index.Lookup(path); !errors.Is(err, ErrKeyframesPending) {
		t.Fatalf("segundo Lookup: %v", err)
	}
	close(release)
	if _, err := waitKeyframeLookup(t, index, path); err != nil {
		t.Fatalf("Lookup final: %v", err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("se encoló %d veces", calls)
	}
}

func TestMatroskaYProbeCoincidenSiHayFFmpeg(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg no está en PATH")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe no está en PATH")
	}
	path := filepath.Join(t.TempDir(), "generado.mkv")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=size=160x120:rate=10:duration=3",
		"-c:v", "libx264", "-g", "10", "-pix_fmt", "yuv420p", "-an", path)
	if output, err := command.CombinedOutput(); err != nil {
		t.Skipf("no se pudo generar el MKV de prueba: %v (%s)", err, output)
	}
	cues, err := MatroskaKeyframes(path)
	if err != nil {
		t.Fatalf("MatroskaKeyframes del MKV generado: %v", err)
	}
	probe, err := ProbeKeyframes(context.Background(), path)
	if err != nil {
		t.Fatalf("ProbeKeyframes del MKV generado: %v", err)
	}
	sort.Float64s(cues)
	sort.Float64s(probe)
	if len(cues) != len(probe) {
		t.Fatalf("Cues = %v, ffprobe = %v", cues, probe)
	}
	for index := range cues {
		if math.Abs(cues[index]-probe[index]) > 0.05 {
			t.Fatalf("Cues = %v, ffprobe = %v", cues, probe)
		}
	}
}
