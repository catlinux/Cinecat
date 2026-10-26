package update

import (
	"strings"
	"testing"
)

const changelogTest = `# Changelog

## [Unreleased]

## [0.14.0] - 2026-10-26
### Añadido
- Comprobación de actualizaciones.
- Otra cosa.

## [0.13.1] - 2026-10-25
### Corregido
- Algo anterior.
`

func TestExtraeLaSeccionCorrecta(t *testing.T) {
	notes := ExtractNotes([]byte(changelogTest), "v0.14.0")
	for _, expected := range []string{"0.14.0", "Comprobación de actualizaciones", "Otra cosa"} {
		if !strings.Contains(notes, expected) {
			t.Fatalf("faltan las notas %q: %s", expected, notes)
		}
	}
	if strings.Contains(notes, "Algo anterior") {
		t.Fatalf("se coló la sección siguiente: %s", notes)
	}
}

func TestNotasAusentesYRecorte(t *testing.T) {
	if notes := ExtractNotes([]byte(changelogTest), "9.9.9"); notes != "" {
		t.Fatalf("una versión ausente no debe tener notas: %q", notes)
	}
	large := "## [1.0.0] - 2026-01-01\n" + strings.Repeat("a", maxNotesBytes+100)
	if notes := ExtractNotes([]byte(large), "1.0.0"); len(notes) > maxNotesBytes {
		t.Fatalf("las notas no se recortaron: %d bytes", len(notes))
	}
}
