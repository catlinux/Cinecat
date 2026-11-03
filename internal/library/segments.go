package library

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/catlinux/cinecat/internal/playback"
)

const (
	segmentIntro   = "intro"
	segmentCredits = "credits"

	segmentSourceChapter = "chapter"
	segmentSourceManual  = "manual"

	introMaxSeconds = 10 * 60
)

var errSegmentNotEpisode = errors.New("el ítem no es un episodio")

var introTitlePrefixes = []string{
	"intro",
	"introduccion",
	"opening",
	"apertura",
	"cabecera",
	"cortinilla",
	"generique",
	"generique de debut",
}

var creditsTitlePrefixes = []string{
	"credits",
	"creditos",
	"ending",
	"end credits",
	"final credits",
	"generique de fin",
	"titulos de credito",
}

type itemSegment struct {
	Kind  string  `json:"kind"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

type playSegment struct {
	Kind   string  `json:"kind"`
	Start  float64 `json:"start"`
	End    float64 `json:"end"`
	Source string  `json:"source"`
}

// normalizeSegmentTitle compara los títulos sin mayúsculas, acentos ni espacios
// repetidos. Los listados de prefijos son deliberadamente cerrados: no se
// intenta adivinar por parecido.
func normalizeSegmentTitle(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	title = strings.Map(foldSegmentRune, title)
	return strings.Join(strings.Fields(title), " ")
}

func foldSegmentRune(r rune) rune {
	switch r {
	case 'á', 'à', 'â', 'ä', 'ã', 'å':
		return 'a'
	case 'é', 'è', 'ê', 'ë':
		return 'e'
	case 'í', 'ì', 'î', 'ï':
		return 'i'
	case 'ó', 'ò', 'ô', 'ö', 'õ':
		return 'o'
	case 'ú', 'ù', 'û', 'ü':
		return 'u'
	case 'ç':
		return 'c'
	case 'ñ':
		return 'n'
	default:
		return r
	}
}

func chapterSegmentKind(title string) string {
	normalized := normalizeSegmentTitle(title)
	// "générique" es prefijo de intro y de créditos. Se comprueba primero la
	// variante de fin para no clasificar "générique de fin" como intro.
	if hasSegmentPrefix(normalized, creditsTitlePrefixes) {
		return segmentCredits
	}
	if hasSegmentPrefix(normalized, introTitlePrefixes) {
		return segmentIntro
	}
	return ""
}

func hasSegmentPrefix(title string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(title, prefix) {
			return true
		}
	}
	return false
}

func chapterSegments(info *playback.FileInfo) []itemSegment {
	if info == nil {
		return []itemSegment{}
	}
	seen := make(map[string]bool, 2)
	byKind := make(map[string]itemSegment, 2)
	for _, chapter := range info.ChapterList {
		kind := chapterSegmentKind(chapter.Title)
		if kind == "" || seen[kind] {
			continue
		}
		if !validChapterSegment(kind, chapter.Start, chapter.End, info.DurationSeconds) {
			continue
		}
		seen[kind] = true
		byKind[kind] = itemSegment{Kind: kind, Start: chapter.Start, End: chapter.End}
	}
	return orderedSegments(byKind)
}

func validChapterSegment(kind string, start, end, duration float64) bool {
	if duration <= 0 || start < 0 || end <= start || end > duration {
		return false
	}
	if kind == segmentIntro && end-start >= introMaxSeconds {
		return false
	}
	return true
}

func orderedSegments(byKind map[string]itemSegment) []itemSegment {
	result := make([]itemSegment, 0, 2)
	for _, kind := range []string{segmentIntro, segmentCredits} {
		if segment, ok := byKind[kind]; ok {
			result = append(result, segment)
		}
	}
	return result
}

func loadManualSegments(ctx context.Context, db *sql.DB, itemID int64) ([]itemSegment, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT kind, start_seconds, end_seconds
		  FROM item_segments
		 WHERE item_id = ?
		 ORDER BY CASE kind WHEN 'intro' THEN 0 ELSE 1 END, kind`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	segments := make([]itemSegment, 0, 2)
	for rows.Next() {
		var segment itemSegment
		if err := rows.Scan(&segment.Kind, &segment.Start, &segment.End); err != nil {
			return nil, err
		}
		segments = append(segments, segment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return segments, nil
}

func replaceManualSegments(
	ctx context.Context,
	db *sql.DB,
	itemID int64,
	segments []itemSegment,
) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // el Commit prevalece si se hizo

	if _, err := tx.ExecContext(ctx, `DELETE FROM item_segments WHERE item_id = ?`, itemID); err != nil {
		return err
	}
	for _, segment := range segments {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO item_segments (item_id, kind, start_seconds, end_seconds)
			VALUES (?, ?, ?, ?)`,
			itemID, segment.Kind, segment.Start, segment.End); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (a *API) segmentsForPlay(
	ctx context.Context,
	itemID int64,
	info *playback.FileInfo,
) ([]playSegment, error) {
	manual, err := loadManualSegments(ctx, a.db, itemID)
	if err != nil {
		return nil, err
	}
	manualByKind := make(map[string]itemSegment, len(manual))
	for _, segment := range manual {
		manualByKind[segment.Kind] = segment
	}
	chapters := chapterSegments(info)
	chapterByKind := make(map[string]itemSegment, len(chapters))
	for _, segment := range chapters {
		chapterByKind[segment.Kind] = segment
	}

	result := make([]playSegment, 0, 2)
	for _, kind := range []string{segmentIntro, segmentCredits} {
		if segment, ok := manualByKind[kind]; ok {
			result = append(result, playSegment{
				Kind: segment.Kind, Start: segment.Start, End: segment.End, Source: segmentSourceManual,
			})
			continue
		}
		if segment, ok := chapterByKind[kind]; ok {
			result = append(result, playSegment{
				Kind: segment.Kind, Start: segment.Start, End: segment.End, Source: segmentSourceChapter,
			})
		}
	}
	return result, nil
}

func (a *API) applySeasonSegments(ctx context.Context, itemID int64) (int, error) {
	var kind string
	var seriesID, season sql.NullInt64
	err := a.db.QueryRowContext(ctx, `
		SELECT kind, series_id, season
		  FROM media_items
		 WHERE id = ?`, itemID).Scan(&kind, &seriesID, &season)
	if err != nil {
		return 0, err
	}
	if kind != KindEpisode || !seriesID.Valid || !season.Valid {
		return 0, errSegmentNotEpisode
	}

	manual, err := loadManualSegments(ctx, a.db, itemID)
	if err != nil {
		return 0, err
	}

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint:errcheck // el Commit prevalece si se hizo

	rows, err := tx.QueryContext(ctx, `
		SELECT id
		  FROM media_items
		 WHERE series_id = ? AND season = ? AND kind = ? AND id <> ?
		 ORDER BY id`, seriesID.Int64, season.Int64, KindEpisode, itemID)
	if err != nil {
		return 0, err
	}
	targets := make([]int64, 0)
	for rows.Next() {
		var targetID int64
		if err := rows.Scan(&targetID); err != nil {
			rows.Close()
			return 0, err
		}
		targets = append(targets, targetID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()

	for _, targetID := range targets {
		if _, err := tx.ExecContext(ctx, `DELETE FROM item_segments WHERE item_id = ?`, targetID); err != nil {
			return 0, err
		}
		for _, segment := range manual {
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO item_segments (item_id, kind, start_seconds, end_seconds)
				VALUES (?, ?, ?, ?)`,
				targetID, segment.Kind, segment.Start, segment.End); err != nil {
				return 0, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(targets), nil
}
