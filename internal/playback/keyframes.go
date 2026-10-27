package playback

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ErrNoKeyframeIndex indica que el archivo no contiene un índice de fotogramas
// clave utilizable.
var ErrNoKeyframeIndex = errors.New("playback: el archivo no tiene índice de fotogramas clave")

const (
	ebmlHeader         = 0x1A45DFA3
	ebmlSegment        = 0x18538067
	ebmlSeekHead       = 0x114D9B74
	ebmlSeek           = 0x4DBB
	ebmlSeekID         = 0x53AB
	ebmlSeekPosition   = 0x53AC
	ebmlInfo           = 0x1549A966
	ebmlTimestampScale = 0x2AD7B1
	ebmlTracks         = 0x1654AE6B
	ebmlTrackEntry     = 0xAE
	ebmlTrackNumber    = 0xD7
	ebmlTrackType      = 0x83
	ebmlCues           = 0x1C53BB6B
	ebmlCuePoint       = 0xBB
	ebmlCueTime        = 0xB3
	ebmlCueTrackPos    = 0xB7
	ebmlCueTrack       = 0xF7
	ebmlCluster        = 0x1F43B675

	defaultTimestampScale = uint64(1_000_000)
	maxKeyframes          = 2_000_000
)

type ebmlElement struct {
	id        uint32
	dataStart int64
	size      int64 // -1 si el tamaño es desconocido
}

func (e ebmlElement) end(parentEnd int64) int64 {
	if e.size < 0 {
		return parentEnd
	}
	return e.dataStart + e.size
}

type ebmlReader struct {
	r    *os.File
	size int64
}

func newEBMLReader(path string) (*ebmlReader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	return &ebmlReader{r: file, size: info.Size()}, nil
}

func (r *ebmlReader) close() error {
	return r.r.Close()
}

func (r *ebmlReader) bytesAt(offset int64, length int) ([]byte, error) {
	if offset < 0 || length < 0 || offset > r.size || int64(length) > r.size-offset {
		return nil, errors.New("elemento fuera del archivo")
	}
	if length == 0 {
		return nil, nil
	}
	buffer := make([]byte, length)
	if _, err := r.r.ReadAt(buffer, offset); err != nil {
		return nil, err
	}
	return buffer, nil
}

func (r *ebmlReader) vintAt(offset int64) (value uint64, length int, unknown bool, err error) {
	first, err := r.bytesAt(offset, 1)
	if err != nil {
		return 0, 0, false, err
	}
	length = 1
	mask := byte(0x80)
	for length <= 8 && first[0]&mask == 0 {
		mask >>= 1
		length++
	}
	if length > 8 {
		return 0, 0, false, errors.New("longitud EBML no válida")
	}
	rest, err := r.bytesAt(offset+1, length-1)
	if err != nil {
		return 0, 0, false, err
	}
	value = uint64(first[0] & (mask - 1))
	for _, b := range rest {
		value = (value << 8) | uint64(b)
	}
	if value == (uint64(1)<<(7*uint(length)))-1 {
		unknown = true
	}
	return value, length, unknown, nil
}

// idAt lee un ID EBML conservando el byte marcador: los IDs son valores
// completos, no enteros VINT con el marcador eliminado.
func (r *ebmlReader) idAt(offset int64) (uint32, int, error) {
	first, err := r.bytesAt(offset, 1)
	if err != nil {
		return 0, 0, err
	}
	length := 1
	mask := byte(0x80)
	for length <= 4 && first[0]&mask == 0 {
		mask >>= 1
		length++
	}
	if length > 4 {
		return 0, 0, errors.New("id EBML no válido")
	}
	data, err := r.bytesAt(offset, length)
	if err != nil {
		return 0, 0, err
	}
	var value uint32
	for _, b := range data {
		value = (value << 8) | uint32(b)
	}
	return value, length, nil
}

func (r *ebmlReader) elementAt(offset, parentEnd int64) (ebmlElement, error) {
	if offset < 0 || offset >= r.size || offset >= parentEnd {
		return ebmlElement{}, errors.New("elemento fuera del padre")
	}
	idValue, idLength, err := r.idAt(offset)
	if err != nil {
		return ebmlElement{}, err
	}
	sizeOffset := offset + int64(idLength)
	sizeValue, sizeLength, unknown, err := r.vintAt(sizeOffset)
	if err != nil {
		return ebmlElement{}, err
	}
	dataStart := sizeOffset + int64(sizeLength)
	if dataStart > parentEnd || dataStart > r.size {
		return ebmlElement{}, errors.New("elemento fuera del archivo")
	}
	element := ebmlElement{
		id:        uint32(idValue),
		dataStart: dataStart,
		size:      int64(sizeValue),
	}
	if unknown {
		element.size = -1
		return element, nil
	}
	if sizeValue > math.MaxInt64 || element.dataStart+int64(sizeValue) > parentEnd ||
		element.dataStart+int64(sizeValue) > r.size {
		return ebmlElement{}, errors.New("tamaño EBML fuera del archivo")
	}
	return element, nil
}

func (r *ebmlReader) uintAt(offset, length int64) (uint64, error) {
	if length < 0 || length > 8 {
		return 0, errors.New("entero EBML demasiado grande")
	}
	buffer, err := r.bytesAt(offset, int(length))
	if err != nil {
		return 0, err
	}
	var value uint64
	for _, b := range buffer {
		value = (value << 8) | uint64(b)
	}
	return value, nil
}

// MatroskaKeyframes lee los tiempos de los fotogramas clave de la primera
// pista de vídeo desde el índice Cues de un contenedor Matroska/WebM. No usa
// la extensión: comprueba la firma EBML.
func MatroskaKeyframes(path string) ([]float64, error) {
	reader, err := newEBMLReader(path)
	if err != nil {
		return nil, err
	}
	defer reader.close()

	signature, err := reader.bytesAt(0, 4)
	if err != nil || !bytes.Equal(signature, []byte{0x1A, 0x45, 0xDF, 0xA3}) {
		return nil, fmt.Errorf("%w: no es un archivo EBML", ErrNoKeyframeIndex)
	}

	segment, err := findSegment(reader)
	if err != nil {
		return nil, err
	}
	segmentEnd := segment.end(reader.size)
	positions := make(map[uint32]int64)
	inline := make(map[uint32]ebmlElement)
	for offset := segment.dataStart; offset < segmentEnd; {
		element, err := reader.elementAt(offset, segmentEnd)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrNoKeyframeIndex, err)
		}
		switch element.id {
		case ebmlSeekHead:
			if err := readSeekHead(reader, element, segment.dataStart, segmentEnd, positions); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrNoKeyframeIndex, err)
			}
		case ebmlInfo, ebmlTracks:
			if _, exists := inline[element.id]; !exists {
				inline[element.id] = element
			}
		case ebmlCluster:
			offset = segmentEnd
			continue
		}
		next := element.end(segmentEnd)
		if next <= offset || next > segmentEnd {
			return nil, fmt.Errorf("%w: recorrido EBML no válido", ErrNoKeyframeIndex)
		}
		offset = next
	}

	info, hasInfo := inline[ebmlInfo]
	if !hasInfo {
		if position, found := positions[ebmlInfo]; found {
			info, err = reader.elementAt(position, segmentEnd)
			if err != nil || info.id != ebmlInfo {
				return nil, fmt.Errorf("%w: Info no válido", ErrNoKeyframeIndex)
			}
			hasInfo = true
		}
	}
	scale := defaultTimestampScale
	if hasInfo {
		if value, found, err := findTimestampScale(reader, info, segmentEnd); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrNoKeyframeIndex, err)
		} else if found && value > 0 {
			scale = value
		}
	}

	tracks, hasTracks := inline[ebmlTracks]
	if !hasTracks {
		if position, found := positions[ebmlTracks]; found {
			tracks, err = reader.elementAt(position, segmentEnd)
			if err != nil || tracks.id != ebmlTracks {
				return nil, fmt.Errorf("%w: Tracks no válido", ErrNoKeyframeIndex)
			}
			hasTracks = true
		}
	}
	if !hasTracks {
		return nil, fmt.Errorf("%w: no hay pista de vídeo", ErrNoKeyframeIndex)
	}
	videoTrack, err := findVideoTrack(reader, tracks, segmentEnd)
	if err != nil || videoTrack == 0 {
		return nil, fmt.Errorf("%w: no hay pista de vídeo", ErrNoKeyframeIndex)
	}

	position, found := positions[ebmlCues]
	if !found {
		return nil, fmt.Errorf("%w: no hay SeekHead hacia Cues", ErrNoKeyframeIndex)
	}
	cues, err := reader.elementAt(position, segmentEnd)
	if err != nil || cues.id != ebmlCues {
		return nil, fmt.Errorf("%w: Cues no válido", ErrNoKeyframeIndex)
	}
	keyframes, err := readCues(reader, cues, segmentEnd, videoTrack, scale)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoKeyframeIndex, err)
	}
	if len(keyframes) == 0 {
		return nil, fmt.Errorf("%w: no hay CuePoints de vídeo", ErrNoKeyframeIndex)
	}
	sort.Float64s(keyframes)
	return uniqueKeyframes(keyframes), nil
}

func findSegment(r *ebmlReader) (ebmlElement, error) {
	offset := int64(0)
	for offset < r.size {
		element, err := r.elementAt(offset, r.size)
		if err != nil {
			return ebmlElement{}, fmt.Errorf("%w: cabecera EBML no válida", ErrNoKeyframeIndex)
		}
		if element.id == ebmlSegment {
			return element, nil
		}
		next := element.end(r.size)
		if next <= offset || next > r.size {
			return ebmlElement{}, fmt.Errorf("%w: cabecera EBML no válida", ErrNoKeyframeIndex)
		}
		offset = next
	}
	return ebmlElement{}, fmt.Errorf("%w: no hay Segment", ErrNoKeyframeIndex)
}

func readSeekHead(r *ebmlReader, seekHead ebmlElement, segmentStart, segmentEnd int64, positions map[uint32]int64) error {
	end := seekHead.end(segmentEnd)
	for offset := seekHead.dataStart; offset < end; {
		element, err := r.elementAt(offset, end)
		if err != nil {
			return err
		}
		if element.id == ebmlSeek {
			var target uint32
			var foundID, foundPosition bool
			var position uint64
			seekEnd := element.end(end)
			for childOffset := element.dataStart; childOffset < seekEnd; {
				child, err := r.elementAt(childOffset, seekEnd)
				if err != nil {
					return err
				}
				switch child.id {
				case ebmlSeekID:
					if child.size < 1 || child.size > 4 {
						return errors.New("SeekID no válido")
					}
					data, err := r.bytesAt(child.dataStart, int(child.size))
					if err != nil {
						return err
					}
					for _, b := range data {
						target = (target << 8) | uint32(b)
					}
					foundID = true
				case ebmlSeekPosition:
					value, err := r.uintAt(child.dataStart, child.size)
					if err != nil || value > math.MaxInt64 {
						return errors.New("SeekPosition no válido")
					}
					position = value
					foundPosition = true
				}
				next := child.end(seekEnd)
				if next <= childOffset || next > seekEnd {
					return errors.New("Seek no válido")
				}
				childOffset = next
			}
			if foundID && foundPosition {
				absolute := segmentStart + int64(position)
				if absolute >= segmentStart && absolute < segmentEnd {
					if _, exists := positions[target]; !exists {
						positions[target] = absolute
					}
				}
			}
		}
		next := element.end(end)
		if next <= offset || next > end {
			return errors.New("SeekHead no válido")
		}
		offset = next
	}
	return nil
}

func findTimestampScale(r *ebmlReader, info ebmlElement, parentEnd int64) (uint64, bool, error) {
	end := info.end(parentEnd)
	for offset := info.dataStart; offset < end; {
		element, err := r.elementAt(offset, end)
		if err != nil {
			return 0, false, err
		}
		if element.id == ebmlTimestampScale {
			value, err := r.uintAt(element.dataStart, element.size)
			return value, true, err
		}
		next := element.end(end)
		if next <= offset || next > end {
			return 0, false, errors.New("Info no válido")
		}
		offset = next
	}
	return 0, false, nil
}

func findVideoTrack(r *ebmlReader, tracks ebmlElement, parentEnd int64) (uint64, error) {
	end := tracks.end(parentEnd)
	for offset := tracks.dataStart; offset < end; {
		entry, err := r.elementAt(offset, end)
		if err != nil {
			return 0, err
		}
		if entry.id == ebmlTrackEntry {
			var number, kind uint64
			var foundNumber, foundKind bool
			trackEnd := entry.end(end)
			for childOffset := entry.dataStart; childOffset < trackEnd; {
				child, err := r.elementAt(childOffset, trackEnd)
				if err != nil {
					return 0, err
				}
				switch child.id {
				case ebmlTrackNumber:
					number, err = r.uintAt(child.dataStart, child.size)
					if err != nil {
						return 0, err
					}
					foundNumber = true
				case ebmlTrackType:
					kind, err = r.uintAt(child.dataStart, child.size)
					if err != nil {
						return 0, err
					}
					foundKind = true
				}
				next := child.end(trackEnd)
				if next <= childOffset || next > trackEnd {
					return 0, errors.New("TrackEntry no válido")
				}
				childOffset = next
			}
			if foundNumber && foundKind && kind == 1 {
				return number, nil
			}
		}
		next := entry.end(end)
		if next <= offset || next > end {
			return 0, errors.New("Tracks no válido")
		}
		offset = next
	}
	return 0, nil
}

func readCues(r *ebmlReader, cues ebmlElement, parentEnd int64, videoTrack, scale uint64) ([]float64, error) {
	end := cues.end(parentEnd)
	result := make([]float64, 0)
	for offset := cues.dataStart; offset < end; {
		point, err := r.elementAt(offset, end)
		if err != nil {
			return nil, err
		}
		if point.id == ebmlCuePoint {
			var cueTime uint64
			var foundTime bool
			var hasVideoTrack bool
			pointEnd := point.end(end)
			for childOffset := point.dataStart; childOffset < pointEnd; {
				child, err := r.elementAt(childOffset, pointEnd)
				if err != nil {
					return nil, err
				}
				switch child.id {
				case ebmlCueTime:
					cueTime, err = r.uintAt(child.dataStart, child.size)
					if err != nil {
						return nil, err
					}
					foundTime = true
				case ebmlCueTrackPos:
					trackEnd := child.end(pointEnd)
					for trackOffset := child.dataStart; trackOffset < trackEnd; {
						track, err := r.elementAt(trackOffset, trackEnd)
						if err != nil {
							return nil, err
						}
						if track.id == ebmlCueTrack {
							number, err := r.uintAt(track.dataStart, track.size)
							if err != nil {
								return nil, err
							}
							if number == videoTrack {
								hasVideoTrack = true
							}
						}
						next := track.end(trackEnd)
						if next <= trackOffset || next > trackEnd {
							return nil, errors.New("CueTrackPositions no válido")
						}
						trackOffset = next
					}
				}
				next := child.end(pointEnd)
				if next <= childOffset || next > pointEnd {
					return nil, errors.New("CuePoint no válido")
				}
				childOffset = next
			}
			if foundTime && hasVideoTrack {
				seconds := float64(cueTime) * float64(scale) / 1e9
				if !math.IsInf(seconds, 0) && !math.IsNaN(seconds) {
					result = append(result, seconds)
					if len(result) > maxKeyframes {
						return nil, errors.New("demasiados CuePoints")
					}
				}
			}
		}
		next := point.end(end)
		if next <= offset || next > end {
			return nil, errors.New("Cues no válido")
		}
		offset = next
	}
	return result, nil
}

func uniqueKeyframes(values []float64) []float64 {
	if len(values) == 0 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

// ProbeKeyframes obtiene los fotogramas clave del primer flujo de vídeo con
// ffprobe. Solo se usa como respaldo de MatroskaKeyframes.
func ProbeKeyframes(ctx context.Context, path string) ([]float64, error) {
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, ErrFFprobeNotFound
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("playback: ruta de vídeo no válida: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	command := lowPriorityCommand(ctx, ffprobe,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "packet=pts_time,flags",
		"-of", "csv=p=0",
		absPath,
	)
	output, err := command.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %v", ErrProbeFailed, ctx.Err())
		}
		return nil, fmt.Errorf("%w: %v", ErrProbeFailed, err)
	}
	times := make([]float64, 0)
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Split(strings.TrimSpace(line), ",")
		if len(fields) < 2 || !strings.Contains(fields[1], "K") {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(fields[0]), 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		times = append(times, value)
	}
	sort.Float64s(times)
	times = uniqueKeyframes(times)
	if len(times) == 0 {
		return nil, ErrInsufficientKeyframes
	}
	return times, nil
}

// SegmentBoundaries calcula los inicios de segmento a partir de las claves.
// El primero es 0; los siguientes son claves que distan al menos 6 s del
// límite anterior y tienen un margen de 0,25 s respecto a sus vecinas.
func SegmentBoundaries(keyframes []float64, duration float64) []float64 {
	bounds := []float64{0}
	if len(keyframes) < 3 {
		return bounds
	}
	keys := append([]float64(nil), keyframes...)
	sort.Float64s(keys)
	target := float64(hlsSegmentSeconds)
	for index := 1; index < len(keys)-1; index++ {
		key := keys[index]
		if key < target {
			continue
		}
		if key-keys[index-1] < 0.25 || keys[index+1]-key < 0.25 {
			continue
		}
		if key >= duration && duration > 0 {
			break
		}
		bounds = append(bounds, key)
		target = key + float64(hlsSegmentSeconds)
	}
	bounds = uniqueKeyframes(bounds)
	// Un último segmento de menos de un segundo se une al anterior aquí, y no
	// solo en BoundaryDurations: si el límite se quedara, FFmpeg cortaría en él
	// y generaría un segmento que la playlist no lista.
	if duration > 0 && len(bounds) > 1 && duration-bounds[len(bounds)-1] < 1 {
		bounds = bounds[:len(bounds)-1]
	}
	return bounds
}

// BoundaryDurations devuelve la duración de cada segmento. El último llega
// hasta duration y, como en uniformSegmentDurations, un resto menor de un
// segundo se suma al segmento anterior.
func BoundaryDurations(bounds []float64, duration float64) []float64 {
	if len(bounds) == 0 {
		return []float64{}
	}
	durations := make([]float64, 0, len(bounds))
	for index, start := range bounds {
		end := duration
		if index+1 < len(bounds) {
			end = bounds[index+1]
		}
		value := end - start
		if value <= 0 {
			continue
		}
		durations = append(durations, value)
	}
	if len(durations) > 1 && durations[len(durations)-1] < 1 {
		durations[len(durations)-2] += durations[len(durations)-1]
		durations = durations[:len(durations)-1]
	}
	return durations
}
