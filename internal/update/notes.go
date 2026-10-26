package update

import (
	"strings"
)

const maxNotesBytes = 8 << 10

// ExtractNotes devuelve la sección del CHANGELOG de la versión indicada, sin
// incluir las secciones siguientes.
func ExtractNotes(changelog []byte, version string) string {
	wanted := strings.TrimPrefix(strings.TrimSpace(version), "v")
	if wanted == "" {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(string(changelog), "\r\n", "\n"), "\n")
	start := -1
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "## [") {
			continue
		}
		end := strings.Index(trimmed, "]")
		if end < 0 {
			continue
		}
		label := strings.TrimPrefix(trimmed[4:end], "v")
		if label != wanted {
			continue
		}
		start = index
		break
	}
	if start < 0 {
		return ""
	}
	end := len(lines)
	for index := start + 1; index < len(lines); index++ {
		if strings.HasPrefix(strings.TrimSpace(lines[index]), "## [") {
			end = index
			break
		}
	}
	notes := strings.TrimSpace(strings.Join(lines[start:end], "\n"))
	if len(notes) > maxNotesBytes {
		notes = strings.ToValidUTF8(notes[:maxNotesBytes], "")
	}
	return notes
}
