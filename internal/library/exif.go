package library

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const exifReadLimit = 1 << 20

// photoTakenAt lee DateTimeOriginal del EXIF de un JPEG. El resto de formatos
// no tienen un EXIF JPEG que podamos leer sin dependencias nuevas.
func photoTakenAt(path string) *int64 {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".jpg" && ext != ".jpeg" {
		return nil
	}
	data, err := readPrefix(path, exifReadLimit)
	if err != nil {
		return nil
	}
	value := exifDateTimeOriginal(data)
	if value == "" {
		return nil
	}
	parsed, err := time.ParseInLocation("2006:01:02 15:04:05", value, time.Local)
	if err != nil {
		return nil
	}
	unix := parsed.Unix()
	return &unix
}

func readPrefix(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data := make([]byte, limit)
	n, err := io.ReadFull(file, data)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return nil, err
	}
	return data[:n], nil
}

// exifDateTimeOriginal busca el segmento APP1, entra en el IFD0 y sigue el
// puntero al SubIFD de EXIF. Cualquier estructura inesperada se ignora.
func exifDateTimeOriginal(data []byte) string {
	if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 {
		return ""
	}
	pos := 2
	for pos < len(data) {
		if data[pos] != 0xff {
			return ""
		}
		for pos < len(data) && data[pos] == 0xff {
			pos++
		}
		if pos >= len(data) {
			return ""
		}
		marker := data[pos]
		pos++
		if marker == 0xd9 || marker == 0xda {
			return ""
		}
		if marker == 0x01 || (marker >= 0xd0 && marker <= 0xd7) {
			continue
		}
		if pos+2 > len(data) {
			return ""
		}
		length := int(binary.BigEndian.Uint16(data[pos : pos+2]))
		pos += 2
		if length < 2 || pos+length-2 > len(data) {
			return ""
		}
		payload := data[pos : pos+length-2]
		if marker == 0xe1 && bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
			if value := tiffDateTimeOriginal(payload[6:]); value != "" {
				return value
			}
		}
		pos += length - 2
	}
	return ""
}

func tiffDateTimeOriginal(tiff []byte) string {
	if len(tiff) < 8 {
		return ""
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return ""
	}
	if order.Uint16(tiff[2:4]) != 42 {
		return ""
	}
	ifd0 := int(order.Uint32(tiff[4:8]))
	if value := ifdString(tiff, order, ifd0, 0x9003); value != "" {
		return value
	}
	subIFD := ifdUint32(tiff, order, ifd0, 0x8769)
	if subIFD == 0 {
		return ""
	}
	return ifdString(tiff, order, int(subIFD), 0x9003)
}

func ifdString(tiff []byte, order binary.ByteOrder, offset int, target uint16) string {
	if offset < 0 || offset+2 > len(tiff) {
		return ""
	}
	count := int(order.Uint16(tiff[offset : offset+2]))
	pos := offset + 2
	for index := 0; index < count; index++ {
		if pos+12 > len(tiff) {
			return ""
		}
		tag := order.Uint16(tiff[pos : pos+2])
		typ := order.Uint16(tiff[pos+2 : pos+4])
		length := int(order.Uint32(tiff[pos+4 : pos+8]))
		valueField := tiff[pos+8 : pos+12]
		if tag == target {
			return asciiValue(tiff, order, typ, length, valueField)
		}
		pos += 12
	}
	return ""
}

func ifdUint32(tiff []byte, order binary.ByteOrder, offset int, target uint16) uint32 {
	if offset < 0 || offset+2 > len(tiff) {
		return 0
	}
	count := int(order.Uint16(tiff[offset : offset+2]))
	pos := offset + 2
	for index := 0; index < count; index++ {
		if pos+12 > len(tiff) {
			return 0
		}
		tag := order.Uint16(tiff[pos : pos+2])
		typ := order.Uint16(tiff[pos+2 : pos+4])
		length := int(order.Uint32(tiff[pos+4 : pos+8]))
		valueField := tiff[pos+8 : pos+12]
		if tag == target {
			if typ != 4 || length != 1 {
				return 0
			}
			return order.Uint32(valueField)
		}
		pos += 12
	}
	return 0
}

func asciiValue(tiff []byte, order binary.ByteOrder, typ uint16, length int, valueField []byte) string {
	if typ != 2 || length <= 0 || length > 4096 {
		return ""
	}
	var value []byte
	if length <= 4 {
		value = valueField[:length]
	} else {
		offset := int(order.Uint32(valueField))
		if offset < 0 || offset+length > len(tiff) {
			return ""
		}
		value = tiff[offset : offset+length]
	}
	return strings.TrimSpace(strings.TrimRight(string(value), "\x00"))
}
