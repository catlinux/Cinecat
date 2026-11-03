package playback

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCapitulosReales(t *testing.T) {
	requireVideoTools(t)

	dir := t.TempDir()
	metadata := filepath.Join(dir, "chapters.ffmetadata")
	const chapters = `;FFMETADATA1

[CHAPTER]
TIMEBASE=1/1000
START=0
END=2000
title=Intro

[CHAPTER]
TIMEBASE=1/1000
START=2000
END=5000
title=Créditos
`
	if err := os.WriteFile(metadata, []byte(chapters), 0o644); err != nil {
		t.Fatalf("crear metadatos: %v", err)
	}

	target := filepath.Join(dir, "capitulos.mkv")
	runFFmpeg(t,
		"-f", "lavfi", "-i", "testsrc=duration=5:size=160x120:rate=10",
		"-f", "ffmetadata", "-i", metadata,
		"-map", "0:v", "-map_metadata", "1", "-map_chapters", "1",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-t", "5",
		target,
	)

	info, err := NewProber().Inspect(context.Background(), target)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if info.Chapters != 2 || len(info.ChapterList) != 2 {
		t.Fatalf("capítulos = %d, lista = %+v", info.Chapters, info.ChapterList)
	}
	if info.ChapterList[0].Title != "Intro" ||
		info.ChapterList[0].Start != 0 || info.ChapterList[0].End != 2 {
		t.Fatalf("primer capítulo inesperado: %+v", info.ChapterList[0])
	}
	if info.ChapterList[1].Title != "Créditos" ||
		info.ChapterList[1].Start != 2 || info.ChapterList[1].End != 5 {
		t.Fatalf("segundo capítulo inesperado: %+v", info.ChapterList[1])
	}
}
