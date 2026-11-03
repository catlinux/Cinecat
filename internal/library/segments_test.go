package library

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/catlinux/cinecat/internal/playback"
)

func TestNormalizeSegmentTitleYPrefijos(t *testing.T) {
	if got := normalizeSegmentTitle("  GÉNÉRIQUE   DE FIN "); got != "generique de fin" {
		t.Fatalf("título normalizado = %q", got)
	}
	cases := map[string]string{
		"Introducción":       segmentIntro,
		"Opening":            segmentIntro,
		"Cortinilla":         segmentIntro,
		"Générique de début": segmentIntro,
		"Créditos":           segmentCredits,
		"End credits":        segmentCredits,
		"Générique de fin":   segmentCredits,
		"Títulos de crédito": segmentCredits,
		"op":                 "",
		"Recap":              "",
	}
	for title, want := range cases {
		if got := chapterSegmentKind(title); got != want {
			t.Errorf("%q: kind = %q, se esperaba %q", title, got, want)
		}
	}
}

func TestChapterSegmentsLimitesYPrimeroDeCadaTipo(t *testing.T) {
	info := &playback.FileInfo{
		DurationSeconds: 1000,
		ChapterList: []playback.Chapter{
			{Start: 0, End: 30, Title: "Introducción"},
			{Start: 30, End: 45, Title: "Opening"},
			{Start: 50, End: 80, Title: "Créditos"},
			{Start: 80, End: 90, Title: "Créditos finales"},
		},
	}
	got := chapterSegments(info)
	if len(got) != 2 || got[0] != (itemSegment{Kind: segmentIntro, Start: 0, End: 30}) ||
		got[1] != (itemSegment{Kind: segmentCredits, Start: 50, End: 80}) {
		t.Fatalf("segmentos detectados = %+v", got)
	}

	tests := []struct {
		name string
		info *playback.FileInfo
		want int
	}{
		{
			name: "intro de 599 segundos",
			info: &playback.FileInfo{DurationSeconds: 1000, ChapterList: []playback.Chapter{
				{Start: 0, End: 599, Title: "Intro"},
			}},
			want: 1,
		},
		{
			name: "intro de 600 segundos",
			info: &playback.FileInfo{DurationSeconds: 1000, ChapterList: []playback.Chapter{
				{Start: 0, End: 600, Title: "Intro"},
			}},
			want: 0,
		},
		{
			name: "fin posterior a la duración",
			info: &playback.FileInfo{DurationSeconds: 100, ChapterList: []playback.Chapter{
				{Start: 0, End: 101, Title: "Intro"},
			}},
			want: 0,
		},
		{
			name: "inicio negativo",
			info: &playback.FileInfo{DurationSeconds: 100, ChapterList: []playback.Chapter{
				{Start: -1, End: 50, Title: "Intro"},
			}},
			want: 0,
		},
		{
			name: "final anterior al inicio",
			info: &playback.FileInfo{DurationSeconds: 100, ChapterList: []playback.Chapter{
				{Start: 50, End: 50, Title: "Intro"},
			}},
			want: 0,
		},
		{
			name: "créditos largos sin límite",
			info: &playback.FileInfo{DurationSeconds: 1000, ChapterList: []playback.Chapter{
				{Start: 0, End: 999, Title: "Créditos"},
			}},
			want: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := chapterSegments(test.info); len(got) != test.want {
				t.Fatalf("segmentos = %+v, se esperaban %d", got, test.want)
			}
		})
	}
}

func TestManualSegmentsSustituyenCapitulosDelMismoTipo(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	id := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      filepath.Join(t.TempDir(), "Pelicula.mkv"),
		title:     "Pelicula",
	})
	if err := replaceManualSegments(context.Background(), ts.db, id, []itemSegment{
		{Kind: segmentIntro, Start: 5, End: 15},
	}); err != nil {
		t.Fatalf("guardar segmento manual: %v", err)
	}

	info := &playback.FileInfo{
		DurationSeconds: 100,
		ChapterList: []playback.Chapter{
			{Start: 0, End: 10, Title: "Intro"},
			{Start: 80, End: 95, Title: "Créditos"},
		},
	}
	got, err := ts.api.segmentsForPlay(context.Background(), id, info)
	if err != nil {
		t.Fatalf("segmentsForPlay: %v", err)
	}
	want := []playSegment{
		{Kind: segmentIntro, Start: 5, End: 15, Source: segmentSourceManual},
		{Kind: segmentCredits, Start: 80, End: 95, Source: segmentSourceChapter},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("segmentos = %+v, se esperaban %+v", got, want)
	}
}

func TestPlayIncluyeSegmentosDinamicosYManuales(t *testing.T) {
	info := fileInfoFixture()
	info.DurationSeconds = 100
	info.ChapterList = []playback.Chapter{
		{Start: 0, End: 20, Title: "Intro"},
	}
	ts := newPlayTestServer(t, info, nil)
	id, _ := insertFileInfoItem(t, ts.libraryTestServer, "Pelicula.mkv", true)

	response := playJSON(t, ts, id, "")
	if len(response.Segments) != 1 || response.Segments[0].Kind != segmentIntro ||
		response.Segments[0].Source != segmentSourceChapter ||
		response.Segments[0].Start != 0 || response.Segments[0].End != 20 {
		t.Fatalf("segmentos de capítulo inesperados: %+v", response.Segments)
	}

	status, body := libraryHTTPDo(t, http.MethodPut, segmentsURL(ts.libraryTestServer, id),
		ts.token, map[string]any{
			"segments": []map[string]any{
				{"kind": "intro", "start": 2, "end": 8},
			},
		})
	if status != http.StatusNoContent {
		t.Fatalf("PUT segmentos: %d %s", status, body)
	}
	response = playJSON(t, ts, id, "")
	if len(response.Segments) != 1 || response.Segments[0].Source != segmentSourceManual ||
		response.Segments[0].Start != 2 || response.Segments[0].End != 8 {
		t.Fatalf("segmento manual no aplicado: %+v", response.Segments)
	}
}

func TestPlaySinSegmentosDevuelveListaVacia(t *testing.T) {
	info := fileInfoFixture()
	info.ChapterList = nil
	ts := newPlayTestServer(t, info, nil)
	id, _ := insertFileInfoItem(t, ts.libraryTestServer, "Pelicula.mkv", true)
	response := playJSON(t, ts, id, "")
	if response.Segments == nil || len(response.Segments) != 0 {
		t.Fatalf("segments = %#v, se esperaba []", response.Segments)
	}
}

func TestPlaySinFichaNoDevuelveSegmentosAunqueHayaManuales(t *testing.T) {
	ts := newPlayTestServer(t, nil, playback.ErrFFprobeNotFound)
	id, _ := insertFileInfoItem(t, ts.libraryTestServer, "Pelicula.mkv", true)
	status, body := libraryHTTPDo(t, http.MethodPut, segmentsURL(ts.libraryTestServer, id),
		ts.token, map[string]any{
			"segments": []map[string]any{{"kind": "intro", "start": 1, "end": 5}},
		})
	if status != http.StatusNoContent {
		t.Fatalf("PUT segmentos: %d %s", status, body)
	}
	response := playJSON(t, ts, id, "")
	if response.Segments == nil || len(response.Segments) != 0 {
		t.Fatalf("segmentos sin ficha = %#v, se esperaba []", response.Segments)
	}
}

func TestAdminSegmentsPermisosValidacionYAplicarTemporada(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	root := t.TempDir()
	libraryID := createLibrary(t, ts.db, root)
	moviePath := filepath.Join(root, "Pelicula.mkv")
	writeFile(t, moviePath)
	movieID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: moviePath, title: "Pelicula",
	})
	info := fileInfoFixture()
	info.DurationSeconds = 100
	info.ChapterList = []playback.Chapter{
		{Start: 0, End: 20, Title: "Intro"},
		{Start: 80, End: 95, Title: "Créditos"},
	}
	ts.api.WithFileInspector(&fakeFileInspector{info: info})

	status, body := libraryHTTPGet(t, segmentsURL(ts, movieID), ts.token)
	if status != http.StatusOK {
		t.Fatalf("GET segmentos: %d %s", status, body)
	}
	var segments segmentsResponse
	if err := json.Unmarshal(body, &segments); err != nil {
		t.Fatalf("GET segmentos: %v", err)
	}
	if len(segments.Manual) != 0 || len(segments.Chapters) != 2 ||
		segments.Chapters[0].Kind != segmentIntro || segments.Chapters[1].Kind != segmentCredits {
		t.Fatalf("respuesta GET inesperada: %+v", segments)
	}

	status, body = libraryHTTPDo(t, http.MethodPut, segmentsURL(ts, movieID), ts.token,
		map[string]any{"segments": []map[string]any{
			{"kind": "credits", "start": 70, "end": 90},
			{"kind": "intro", "start": 3, "end": 13},
		}})
	if status != http.StatusNoContent {
		t.Fatalf("PUT válido: %d %s", status, body)
	}
	status, body = libraryHTTPGet(t, segmentsURL(ts, movieID), ts.token)
	if status != http.StatusOK {
		t.Fatalf("GET tras PUT: %d %s", status, body)
	}
	if err := json.Unmarshal(body, &segments); err != nil {
		t.Fatalf("GET tras PUT: %v", err)
	}
	if len(segments.Manual) != 2 || segments.Manual[0].Kind != segmentIntro ||
		segments.Manual[0].Start != 3 || segments.Manual[1].Kind != segmentCredits {
		t.Fatalf("manuales tras PUT = %+v", segments.Manual)
	}

	invalid := []map[string]any{
		{"segments": []map[string]any{{"kind": "otro", "start": 0, "end": 1}}},
		{"segments": []map[string]any{
			{"kind": "intro", "start": 0, "end": 1},
			{"kind": "intro", "start": 2, "end": 3},
		}},
		{"segments": []map[string]any{{"kind": "intro", "start": -1, "end": 1}}},
		{"segments": []map[string]any{{"kind": "intro", "start": 2, "end": 2}}},
		{"segments": nil},
		{},
	}
	for index, payload := range invalid {
		status, body = libraryHTTPDo(t, http.MethodPut, segmentsURL(ts, movieID), ts.token, payload)
		if status != http.StatusBadRequest {
			t.Fatalf("payload inválido %d: %d %s", index, status, body)
		}
	}

	status, body = libraryHTTPDo(t, http.MethodPut, segmentsURL(ts, movieID), ts.token,
		map[string]any{"segments": []any{}})
	if status != http.StatusNoContent {
		t.Fatalf("PUT vacío: %d %s", status, body)
	}
	if got, err := loadManualSegments(context.Background(), ts.db, movieID); err != nil || len(got) != 0 {
		t.Fatalf("manuales tras vaciar = %+v, %v", got, err)
	}

	seriesID := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Serie", title: "Serie",
	})
	sourceID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(root, "Serie", "S01E01.mkv"),
		title: "Serie", kind: KindEpisode, seriesID: seriesID, season: 1, episode: 1,
	})
	targetID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(root, "Serie", "S01E02.mkv"),
		title: "Serie", kind: KindEpisode, seriesID: seriesID, season: 1, episode: 2,
	})
	otherSeasonID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(root, "Serie", "S02E01.mkv"),
		title: "Serie", kind: KindEpisode, seriesID: seriesID, season: 2, episode: 1,
	})
	if err := replaceManualSegments(context.Background(), ts.db, sourceID, []itemSegment{
		{Kind: segmentIntro, Start: 4, End: 14},
	}); err != nil {
		t.Fatalf("guardar manual de origen: %v", err)
	}
	if err := replaceManualSegments(context.Background(), ts.db, targetID, []itemSegment{
		{Kind: segmentCredits, Start: 80, End: 90},
	}); err != nil {
		t.Fatalf("guardar manual de destino: %v", err)
	}
	if err := replaceManualSegments(context.Background(), ts.db, otherSeasonID, []itemSegment{
		{Kind: segmentIntro, Start: 1, End: 2},
	}); err != nil {
		t.Fatalf("guardar manual de otra temporada: %v", err)
	}

	status, body = libraryHTTPDo(t, http.MethodPost, applySegmentsURL(ts, sourceID), ts.token, nil)
	if status != http.StatusOK {
		t.Fatalf("aplicar temporada: %d %s", status, body)
	}
	var applied map[string]int
	if err := json.Unmarshal(body, &applied); err != nil || applied["updated"] != 1 {
		t.Fatalf("aplicar temporada = %s, %v", body, err)
	}
	if got, _ := loadManualSegments(context.Background(), ts.db, targetID); len(got) != 1 ||
		got[0] != (itemSegment{Kind: segmentIntro, Start: 4, End: 14}) {
		t.Fatalf("manuales del destino = %+v", got)
	}
	if got, _ := loadManualSegments(context.Background(), ts.db, otherSeasonID); len(got) != 1 ||
		got[0] != (itemSegment{Kind: segmentIntro, Start: 1, End: 2}) {
		t.Fatalf("manuales de otra temporada = %+v", got)
	}

	if status, _ := libraryHTTPDo(t, http.MethodPost, applySegmentsURL(ts, movieID), ts.token, nil); status != http.StatusBadRequest {
		t.Fatalf("aplicar en película: %d", status)
	}
	if status, _ := libraryHTTPGet(t, segmentsURL(ts, 999999), ts.token); status != http.StatusNotFound {
		t.Fatalf("GET inexistente: %d", status)
	}

	userToken := nonAdminToken(t, ts)
	if status, _ := libraryHTTPGet(t, segmentsURL(ts, movieID), userToken); status != http.StatusForbidden {
		t.Fatalf("GET sin admin: %d", status)
	}
	if status, _ := libraryHTTPDo(t, http.MethodPut, segmentsURL(ts, movieID), userToken,
		map[string]any{"segments": []any{}}); status != http.StatusForbidden {
		t.Fatalf("PUT sin admin: %d", status)
	}
	if status, _ := libraryHTTPDo(t, http.MethodPost, applySegmentsURL(ts, sourceID), userToken, nil); status != http.StatusForbidden {
		t.Fatalf("apply season sin admin: %d", status)
	}
}

func segmentsURL(ts *libraryTestServer, id int64) string {
	return ts.URL + "/api/v1/items/" + strconv.FormatInt(id, 10) + "/segments"
}

func applySegmentsURL(ts *libraryTestServer, id int64) string {
	return segmentsURL(ts, id) + "/apply-season"
}
