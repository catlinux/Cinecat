package playback

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// MaxViableKeyframeInterval es el mayor intervalo entre keyframes que se
// considera segmentable sin recodificar. Con hls_time de 6 s, un GOP de hasta
// 15 s deja segmentos de tamaño aceptable y cubre los GOP habituales de 10 s;
// por encima, el primer segmento puede ser enorme y el navegador tarda
// demasiado en empezar a reproducir.
const MaxViableKeyframeInterval = 15 * time.Second

// probeTimeout acota el análisis de keyframes. Leer los paquetes de un archivo
// grande puede tardar, pero dos minutos evitan bloquear una petición HLS
// indefinidamente.
const probeTimeout = 2 * time.Minute

// Errores del detector de keyframes.
var (
	ErrFFprobeNotFound          = errors.New("playback: ffprobe no está instalado o no está en PATH")
	ErrProbeFailed              = errors.New("playback: ffprobe no pudo analizar el vídeo")
	ErrInsufficientKeyframes    = errors.New("playback: el vídeo no tiene suficientes keyframes")
	ErrKeyframeIntervalTooLarge = errors.New(
		"playback: el intervalo entre keyframes es demasiado grande para segmentar sin recodificar",
	)
	ErrDurationUnknown = errors.New("playback: no se pudo determinar la duración del vídeo")
)

// Prober analiza los keyframes de un vídeo con ffprobe.
type Prober struct{}

// NewProber crea el detector basado en ffprobe.
func NewProber() *Prober {
	return &Prober{}
}

// CheckViable comprueba si el intervalo máximo entre keyframes permite un
// remux HLS razonable. Devuelve ErrKeyframeIntervalTooLarge si no es viable.
func (p *Prober) CheckViable(ctx context.Context, path string) error {
	interval, err := p.MaxKeyframeInterval(ctx, path)
	if err != nil {
		if errors.Is(err, ErrInsufficientKeyframes) {
			return fmt.Errorf(
				"%w: el vídeo no tiene keyframes suficientes para segmentar sin recodificar",
				ErrKeyframeIntervalTooLarge,
			)
		}
		return err
	}
	if interval > MaxViableKeyframeInterval {
		return fmt.Errorf("%w: %s entre keyframes", ErrKeyframeIntervalTooLarge, interval)
	}
	return nil
}

// MaxKeyframeInterval devuelve el mayor tiempo entre dos keyframes consecutivos
// del primer flujo de vídeo. Devuelve ErrInsufficientKeyframes si hay menos de
// dos keyframes utilizables.
func (p *Prober) MaxKeyframeInterval(ctx context.Context, path string) (time.Duration, error) {
	times, err := p.KeyframeTimes(ctx, path)
	if err != nil {
		return 0, err
	}
	if len(times) < 2 {
		return 0, ErrInsufficientKeyframes
	}

	var maxInterval time.Duration
	for i := 1; i < len(times); i++ {
		if diff := times[i] - times[i-1]; diff > maxInterval {
			maxInterval = diff
		}
	}
	if maxInterval <= 0 {
		return 0, ErrInsufficientKeyframes
	}
	return maxInterval, nil
}

// KeyframeTimes devuelve los tiempos de los keyframes del primer flujo de vídeo,
// ordenados de menor a mayor.
func (p *Prober) KeyframeTimes(ctx context.Context, path string) ([]time.Duration, error) {
	output, err := p.runFFprobe(ctx, path,
		"-select_streams", "v:0",
		"-show_entries", "packet=pts_time,dts_time,flags",
	)
	if err != nil {
		return nil, err
	}

	seconds := parseKeyframeTimes(output)
	sort.Float64s(seconds)
	times := make([]time.Duration, 0, len(seconds))
	for _, value := range seconds {
		times = append(times, time.Duration(value*float64(time.Second)))
	}
	if len(times) == 0 {
		return nil, ErrInsufficientKeyframes
	}
	return times, nil
}

// VideoDuration lee la duración del contenedor con ffprobe. Es una consulta de
// metadatos, no decodifica el vídeo. Devuelve ErrDurationUnknown si el
// contenedor no la trae (por ejemplo, algunos AVI antiguos).
func (p *Prober) VideoDuration(ctx context.Context, path string) (time.Duration, error) {
	output, err := p.runFFprobe(ctx, path, "-show_entries", "format=duration")
	if err != nil {
		return 0, err
	}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.EqualFold(line, "N/A") {
			return 0, ErrDurationUnknown
		}
		seconds, err := strconv.ParseFloat(line, 64)
		if err != nil || seconds <= 0 {
			return 0, fmt.Errorf("%w: %q", ErrDurationUnknown, line)
		}
		return time.Duration(seconds * float64(time.Second)), nil
	}
	return 0, ErrDurationUnknown
}

func (p *Prober) runFFprobe(ctx context.Context, path string, args ...string) (string, error) {
	return p.runFFprobeOutput(ctx, path, "csv=p=0", args...)
}

func (p *Prober) runFFprobeOutput(ctx context.Context, path, format string, args ...string) (string, error) {
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return "", ErrFFprobeNotFound
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("playback: ruta de vídeo no válida: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	commandArgs := []string{"-v", "error"}
	commandArgs = append(commandArgs, args...)
	commandArgs = append(commandArgs, "-of", format, absPath)

	cmd := exec.CommandContext(ctx, ffprobe, commandArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("%w: %v", ErrProbeFailed, ctx.Err())
		}
		return "", fmt.Errorf("%w: %v: %s", ErrProbeFailed, err, tailString(stderr.String(), 300))
	}
	return stdout.String(), nil
}

// parseKeyframeTimes interpreta las líneas csv de packet=pts_time,dts_time,flags.
// Se usa pts y, si no está disponible, dts.
func parseKeyframeTimes(output string) []float64 {
	var times []float64
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 3 || !strings.Contains(fields[2], "K") {
			continue
		}

		value, err := strconv.ParseFloat(strings.TrimSpace(fields[0]), 64)
		if err != nil {
			value, err = strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)
		}
		if err != nil {
			continue
		}
		times = append(times, value)
	}
	return times
}
