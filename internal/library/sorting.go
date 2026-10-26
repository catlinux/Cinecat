package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// signos que pueden abrir un título y artículos que no cuentan para ordenar.
var leadingTitleMarks = map[rune]bool{
	'¿': true, '¡': true, '"': true, '\'': true, '(': true, '[': true,
}

var leadingTitleArticles = []string{
	"unos", "unas", "los", "las", "una", "the", "el", "la", "un", "an", "a", "l'",
}

// sortKeys devuelve el título normalizado para ordenar y su inicial para el
// filtro por letra. Solo se pliega la inicial; el resto conserva sus tildes.
func sortKeys(title string) (sortTitle, sortLetter string) {
	clean := stripLeadingTitle(title)
	if clean == "" {
		return "", "#"
	}

	runes := []rune(clean)
	initial := foldInitial(runes[0])
	runes[0] = initial
	sortTitle = string(runes)
	sortLetter = "#"
	if initial >= 'A' && initial <= 'Z' {
		sortLetter = string(initial)
	}
	return sortTitle, sortLetter
}

func stripLeadingTitle(title string) string {
	value := strings.TrimSpace(title)
	for {
		changed := false
		value = strings.TrimSpace(value)
		if value == "" {
			break
		}

		r, size := utf8.DecodeRuneInString(value)
		if leadingTitleMarks[r] {
			value = strings.TrimSpace(value[size:])
			changed = true
		}
		if value == "" {
			break
		}

		lower := strings.ToLower(value)
		for _, article := range leadingTitleArticles {
			if !strings.HasPrefix(lower, article) {
				continue
			}
			rest := value[len(article):]
			if article == "l'" {
				value = strings.TrimSpace(rest)
				changed = true
				break
			}
			if rest != "" && unicode.IsSpace(rune(rest[0])) {
				value = strings.TrimSpace(rest)
				changed = true
				break
			}
		}
		if !changed {
			break
		}
	}
	return value
}

func foldInitial(r rune) rune {
	switch unicode.ToUpper(r) {
	case 'Á':
		return 'A'
	case 'É':
		return 'E'
	case 'Í':
		return 'I'
	case 'Ó':
		return 'O'
	case 'Ú', 'Ü':
		return 'U'
	case 'Ñ':
		return 'N'
	default:
		return unicode.ToUpper(r)
	}
}

func parseLetter(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	runes := []rune(value)
	if len(runes) != 1 {
		return "", errors.New("letter debe ser una letra A-Z o #")
	}
	if runes[0] == '#' {
		return "#", nil
	}
	initial := foldInitial(runes[0])
	if initial >= 'A' && initial <= 'Z' {
		return string(initial), nil
	}
	return "", errors.New("letter debe ser una letra A-Z o #")
}

// BackfillSortKeys completa las claves de las filas anteriores a la migración
// o insertadas directamente en la base de datos.
func BackfillSortKeys(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx,
		`SELECT id, COALESCE(display_title, title)
		   FROM media_items
		  WHERE sort_title IS NULL OR sort_letter IS NULL`)
	if err != nil {
		return fmt.Errorf("library: listar títulos sin normalizar: %w", err)
	}
	type pending struct {
		id    int64
		title string
	}
	var items []pending
	for rows.Next() {
		var item pending
		if err := rows.Scan(&item.id, &item.title); err != nil {
			rows.Close()
			return fmt.Errorf("library: leer título sin normalizar: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("library: recorrer títulos sin normalizar: %w", err)
	}
	rows.Close()

	for _, item := range items {
		if err := setSortKeys(ctx, db, item.id, item.title); err != nil {
			return err
		}
	}
	return nil
}

func setSortKeys(ctx context.Context, db *sql.DB, id int64, title string) error {
	sortTitle, sortLetter := sortKeys(title)
	if _, err := db.ExecContext(ctx,
		`UPDATE media_items SET sort_title = ?, sort_letter = ? WHERE id = ?`,
		sortTitle, sortLetter, id,
	); err != nil {
		return fmt.Errorf("library: guardar título normalizado del ítem %d: %w", id, err)
	}
	return nil
}

func refreshSortKeysByPath(ctx context.Context, db *sql.DB, libraryID int64, path string) error {
	var id int64
	var title string
	if err := db.QueryRowContext(ctx,
		`SELECT id, COALESCE(display_title, title)
		   FROM media_items WHERE library_id = ? AND path = ?`,
		libraryID, path,
	).Scan(&id, &title); err != nil {
		return fmt.Errorf("library: leer título del ítem %q: %w", path, err)
	}
	return setSortKeys(ctx, db, id, title)
}
