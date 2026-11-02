package playback

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Muestra mínima de VobSub para probar el quemado de subtítulos de imagen.
// FFmpeg no convierte texto a dvdsub, así que se muxa esta pista bitmap con
// vídeo generado en la propia prueba.
const testVobSubIdx = `IyBWb2JTdWIgaW5kZXggZmlsZSwgdjcgKGRvIG5vdCBtb2RpZnkgdGhpcyBsaW5lISkKc2l6ZTogMTI4MHg3MjAKb3JnOiAwLCAwCnNjYWxlOiAxMDAlLCAxMDAlCmFscGhhOiAxMDAlCnNtb290aDogT0ZGCmZhZGVpbi9vdXQ6IDUwLCA1MAphbGlnbjogT0ZGIGF0IExFRlQgVE9QCnRpbWUgb2Zmc2V0OiAwCmZvcmNlZCBzdWJzOiBPRkYKcGFsZXR0ZTogMDAwMDAwLCBmZmZmZmYsIDAwMDAwMCwgMDAwMDAwLCA4MjgyODIsIDgyODI4MiwgODI4MjgyLCBmZmZmZmYsIDgyODI4MiwgYmFiYWJhLCA4MjgyODIsIDgyODI4MiwgODI4MjgyLCA4MjgyODIsIDgyODI4MiwgODI4MjgyCmN1c3RvbSBjb2xvcnM6IE9GRiwgdHJpZHg6IDAwMDAsIGNvbG9yczogMDAwMDAwLCAwMDAwMDAsIDAwMDAwMCwgMDAwMDAwCmxhbmdpZHg6IDAKaWQ6IGVuLCBpbmRleDogMAp0aW1lc3RhbXA6IDAwOjAwOjAwOjAwMCwgZmlsZXBvczogMDAwMDAwMDAwCnRpbWVzdGFtcDogMDA6MDA6MDE6MDAwLCBmaWxlcG9zOiAwMDAwMDA4MDAK`
const testVobSubData = `AAABukQCxIIEqQGJw/gAAAG9ALyBgAUhAAEAASAAswCVCMDKfXYGAMaR4FhGcVfnEXqKcResTpEnkSdXp5EnV6TGkSkampGp4MaRKehpqehp4MaRaRJ1fpEnV+DGvkpxFxpxFxYUEsKoKkAQHgYAxnEX4FhOk2FB4QBGcVenGXpGcZeoxpEpcWmpcWngxpEpEkaakSRp4MaRJ1ekqadXpKngxp5GcZcScZcSwaguQuQAAAAAAK0BAzIQBP/wBSbikCsyxQYABABO/wBXAK0C/wAAAb4HKv////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAG6RALEggSpAYnD+AAAAb0AqoGABSEABb8hIAChAIMIwIZ1dgcAht4GgIYdeopxF2Rrp1dkhtEnVyJ1etcWhtFpInmtGobeRppyGt7AhtEnV6kS2t7AhrZ9cWfXa6dX7BAuQELAgWBwht4GgIbRoQHoJkCGIXpGcZ4ZeobRaSaa0aCG3kaecdrRKIbRaakS2t7AhiF+chrewIB6wAAAAAAAmwEDMhAE//AFJuKQKzLFBgAEAEr/AFcAmwL/AAABvgc8/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////w==`

// Pruebas con FFmpeg real: generan el vídeo de prueba en el momento y pasan por
// el mismo camino que una reproducción (Inspect + prepare).

func runFFmpeg(t *testing.T, args ...string) {
	t.Helper()
	full := append([]string{"-hide_banner", "-loglevel", "error", "-y"}, args...)
	if out, err := exec.Command("ffmpeg", full...).CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg %v: %v: %s", args, err, out)
	}
}

func requireEncoder(t *testing.T, name string) {
	t.Helper()
	out, err := exec.Command("ffmpeg", "-hide_banner", "-encoders").Output()
	if err != nil || !strings.Contains(string(out), name) {
		t.Skipf("ffmpeg no incluye el codificador %s", name)
	}
}

func firstSegmentStreams(t *testing.T, source string, caps string, audio int) (*FileInfo, string, string) {
	t.Helper()
	svc := newHLSTestService(t)
	info, err := svc.Inspect(context.Background(), source)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	entry, err := svc.prepare(context.Background(), 1, source, StreamSelection{
		Audio: audio,
		Caps:  caps,
		Info:  info,
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := svc.waitForSegment(context.Background(), entry, 0); err != nil {
		t.Fatalf("segmento 0: %v", err)
	}
	segment := filepath.Join(entry.dir, segmentName(0))
	return info, strings.Join(probeStreams(t, segment), " "), segment
}

func generate51(t *testing.T, name, audioCodec string) string {
	t.Helper()
	target := filepath.Join(t.TempDir(), name)
	runFFmpeg(t,
		"-f", "lavfi", "-i", "testsrc=duration=12:size=320x240:rate=25",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=12",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
		"-c:a", audioCodec, "-ac", "6", "-channel_layout", "5.1",
		target,
	)
	return target
}

func TestAudio51Real(t *testing.T) {
	requireVideoTools(t)
	for _, codec := range []string{"ac3", "eac3", "aac"} {
		t.Run(codec, func(t *testing.T) {
			source := generate51(t, "cine51.mkv", codec)

			info, streams, _ := firstSegmentStreams(t, source, "mkv,mp4,h264,aac,mch", 0)
			if len(info.Audio) == 0 || info.Audio[0].Channels != 6 {
				t.Fatalf("el origen debería tener 6 canales: %+v", info.Audio)
			}
			if !strings.Contains(streams, "aac,audio,6") {
				t.Fatalf("con mch se esperaba AAC 5.1, hay: %s", streams)
			}

			_, streams, _ = firstSegmentStreams(t, source, "mkv,mp4,h264,aac", 0)
			if !strings.Contains(streams, "aac,audio,2") {
				t.Fatalf("sin mch se esperaba AAC estéreo, hay: %s", streams)
			}

			_, streams, _ = firstSegmentStreams(t, source, "", 0)
			if !strings.Contains(streams, "aac,audio,2") {
				t.Fatalf("el navegador debe recibir AAC estéreo, hay: %s", streams)
			}
		})
	}
}

func TestAudio51LayoutLateralReal(t *testing.T) {
	requireVideoTools(t)
	target := filepath.Join(t.TempDir(), "side.mkv")
	runFFmpeg(t,
		"-f", "lavfi", "-i", "testsrc=duration=12:size=320x240:rate=25",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=12",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
		"-c:a", "eac3", "-ac", "6", "-channel_layout", "5.1(side)",
		target,
	)
	_, streams, _ := firstSegmentStreams(t, target, "mkv,mp4,h264,aac,mch", 0)
	if !strings.Contains(streams, "aac,audio,6") {
		t.Fatalf("un 5.1(side) debe convertirse a AAC 5.1, hay: %s", streams)
	}
}

func TestDesentrelazadoReal(t *testing.T) {
	requireVideoTools(t)
	target := filepath.Join(t.TempDir(), "entrelazado.mkv")
	runFFmpeg(t,
		"-f", "lavfi", "-i", "testsrc=duration=12:size=320x240:rate=25",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=12",
		"-vf", "interlace=scan=tff",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
		"-flags", "+ilme+ildct", "-x264opts", "tff=1",
		"-c:a", "aac",
		target,
	)
	info, streams, segment := firstSegmentStreams(t, target, "", 0)
	if !videoNeedsDeinterlace(&info.Video[0]) {
		t.Fatalf("el origen debería detectarse como entrelazado: field_order=%q", info.Video[0].FieldOrder)
	}
	if !strings.Contains(streams, "h264,video") {
		t.Fatalf("se esperaba H.264, hay: %s", streams)
	}
	if got := videoProp(t, segment, "field_order"); got != "progressive" {
		t.Fatalf("la salida debería ser progresiva, field_order=%q", got)
	}
}

func TestHDRaSDRReal(t *testing.T) {
	requireVideoTools(t)
	requireEncoder(t, "libx265")
	target := filepath.Join(t.TempDir(), "hdr.mkv")
	runFFmpeg(t,
		"-f", "lavfi", "-i", "testsrc2=duration=12:size=320x240:rate=25",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=12",
		"-vf", "format=yuv420p10le,setparams=color_primaries=bt2020:color_trc=smpte2084:colorspace=bt2020nc",
		"-c:v", "libx265", "-x265-params", "log-level=error",
		"-c:a", "aac",
		target,
	)
	svc := newHLSTestService(t)
	if !svc.hdrFiltersAvailable(context.Background()) {
		t.Skip("este ffmpeg no tiene zscale/tonemap")
	}
	info, streams, segment := firstSegmentStreams(t, target, "", 0)
	if !videoIsHDR(&info.Video[0]) {
		t.Fatalf("el origen debería detectarse como HDR: transfer=%q", info.Video[0].ColorTransfer)
	}
	if !strings.Contains(streams, "h264,video") {
		t.Fatalf("se esperaba H.264, hay: %s", streams)
	}
	if got := videoProp(t, segment, "color_transfer"); got == "smpte2084" {
		t.Fatalf("la salida debería ser SDR, color_transfer=%q", got)
	}
	if got := videoProp(t, segment, "pix_fmt"); got != "yuv420p" {
		t.Fatalf("pix_fmt inesperado %q", got)
	}
}

func writeVobSubSample(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	idx, err := base64.StdEncoding.DecodeString(testVobSubIdx)
	if err != nil {
		t.Fatalf("decodificar idx de prueba: %v", err)
	}
	sub, err := base64.StdEncoding.DecodeString(testVobSubData)
	if err != nil {
		t.Fatalf("decodificar sub de prueba: %v", err)
	}
	idxPath := filepath.Join(dir, "sample.idx")
	if err := os.WriteFile(idxPath, idx, 0o644); err != nil {
		t.Fatalf("escribir idx de prueba: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sample.sub"), sub, 0o644); err != nil {
		t.Fatalf("escribir sub de prueba: %v", err)
	}
	return idxPath
}

func generateImageSubtitleVideo(t *testing.T) string {
	t.Helper()
	idx := writeVobSubSample(t)
	dir := t.TempDir()
	base := filepath.Join(dir, "base.mp4")
	runFFmpeg(t,
		"-f", "lavfi", "-i", "testsrc=duration=12:size=1280x720:rate=25",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=12",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", "-shortest",
		base,
	)
	target := filepath.Join(dir, "imagen.mkv")
	runFFmpeg(t,
		"-i", base, "-i", idx,
		"-map", "0:v", "-map", "0:a", "-map", "1:s",
		"-c:v", "copy", "-c:a", "copy", "-c:s", "copy",
		target,
	)
	return target
}

func frameMD5(t *testing.T, segment, at string) string {
	t.Helper()
	out, err := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error",
		"-ss", at, "-i", segment, "-frames:v", "1", "-f", "md5", "-").Output()
	if err != nil {
		t.Fatalf("extraer fotograma de %s: %v", segment, err)
	}
	return strings.TrimSpace(string(out))
}

func TestSubtituloImagenQuemadoReal(t *testing.T) {
	requireVideoTools(t)
	source := generateImageSubtitleVideo(t)
	svc := newHLSTestService(t)
	info, err := svc.Inspect(context.Background(), source)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	index := -1
	for position, stream := range info.Subtitles {
		if ImageSubtitleCodec(stream.Codec) {
			index = position
			break
		}
	}
	if index < 0 {
		t.Fatalf("el MKV de prueba no tiene subtítulo de imagen: %+v", info.Subtitles)
	}

	plain, err := svc.prepare(context.Background(), 1, source, StreamSelection{Info: info})
	if err != nil {
		t.Fatalf("prepare sin quemar: %v", err)
	}
	if err := svc.waitForSegment(context.Background(), plain, 0); err != nil {
		t.Fatalf("segmento sin quemar: %v", err)
	}
	burned, err := svc.prepare(context.Background(), 2, source, StreamSelection{
		Info:         info,
		BurnSubtitle: index + 1,
	})
	if err != nil {
		t.Fatalf("prepare quemando: %v", err)
	}
	if err := svc.waitForSegment(context.Background(), burned, 0); err != nil {
		t.Fatalf("segmento quemado: %v", err)
	}

	plainSegment := filepath.Join(plain.dir, segmentName(0))
	burnedSegment := filepath.Join(burned.dir, segmentName(0))
	plainFrame := frameMD5(t, plainSegment, "0")
	burnedFrame := frameMD5(t, burnedSegment, "0")
	if plainFrame == burnedFrame {
		t.Fatalf("el fotograma quemado no cambió: %s", plainFrame)
	}
}

func videoProp(t *testing.T, path, key string) string {
	t.Helper()
	out, err := exec.Command("ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream="+key, "-of", "csv=p=0", path).Output()
	if err != nil {
		t.Fatalf("ffprobe %s: %v", path, err)
	}
	// El MPEG-TS repite la línea de la pista de vídeo; basta la primera.
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}
