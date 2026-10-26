package library

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catlinux/cinecat/internal/metadata"
)

type staticTVSearcher struct {
	results []metadata.Candidate
}

func (s *staticTVSearcher) SearchTV(ctx context.Context, query string, year int) ([]metadata.Candidate, error) {
	return s.results, nil
}

func TestIdentifySeries(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		candidate metadata.Candidate
	}{
		{
			name:  "The Big Bang theory",
			title: "The Big Bang theory",
			candidate: metadata.Candidate{
				TMDBID: 1418, Title: "The Big Bang Theory", OriginalTitle: "The Big Bang Theory",
				Year: 2007, VoteCount: 12000,
			},
		},
		{
			name:  "Mr Robot",
			title: "Mr Robot",
			candidate: metadata.Candidate{
				TMDBID: 62560, Title: "Mr. Robot", OriginalTitle: "Mr. Robot",
				Year: 2015, VoteCount: 8000,
			},
		},
		{
			name:  "Chicago fire",
			title: "Chicago fire",
			candidate: metadata.Candidate{
				TMDBID: 44006, Title: "Chicago Fire", OriginalTitle: "Chicago Fire",
				Year: 2012, VoteCount: 6000,
			},
		},
		{
			name:  "Juego de tronos",
			title: "Juego de tronos",
			candidate: metadata.Candidate{
				TMDBID: 1399, Title: "Juego de Tronos", OriginalTitle: "Game of Thrones",
				Year: 2011, VoteCount: 20000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searcher := &staticTVSearcher{results: []metadata.Candidate{tt.candidate}}
			got, score, ok, err := IdentifySeries(context.Background(), searcher, tt.title, 0)
			if err != nil {
				t.Fatalf("IdentifySeries: %v", err)
			}
			if !ok || got.TMDBID != tt.candidate.TMDBID || score < MatchThreshold {
				t.Fatalf("resultado = %+v, %.3f, %v", got, score, ok)
			}
		})
	}

	t.Run("título inexistente", func(t *testing.T) {
		got, score, ok, err := IdentifySeries(
			context.Background(), &staticTVSearcher{}, "Serie que no existe", 0)
		if err != nil {
			t.Fatalf("IdentifySeries: %v", err)
		}
		if ok || got.TMDBID != 0 || score != 0 {
			t.Fatalf("resultado = %+v, %.3f, %v", got, score, ok)
		}
	})
}

func TestEnsureSeriesUneMayúsculas(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	libraryID := createLibrary(t, db, root)

	first, err := ensureSeries(ctx, db, libraryID, "Juego de tronos", "Juego de tronos", 0)
	if err != nil {
		t.Fatalf("ensureSeries: %v", err)
	}
	second, err := ensureSeries(ctx, db, libraryID, "juego de tronos", "juego de tronos", 0)
	if err != nil {
		t.Fatalf("ensureSeries con otra capitalización: %v", err)
	}
	if first != second {
		t.Fatalf("ids distintos: %d y %d", first, second)
	}

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM series`).Scan(&count); err != nil {
		t.Fatalf("contar series: %v", err)
	}
	if count != 1 {
		t.Fatalf("series = %d, se esperaba 1", count)
	}
}

func TestScannerAsociaEpisodiosASeries(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()

	first := filepath.Join(root, "Juego de tronos", "Temporada 1", "Juego de tronos 1x01.mkv")
	second := filepath.Join(root, "Juego de tronos", "Temporada 1", "Juego de tronos 1x02.mkv")
	loose := filepath.Join(root, "Suelto S01E02.mkv")
	movie := filepath.Join(root, "El Orfanato (2007)", "El Orfanato.mkv")
	for _, path := range []string{first, second, loose, movie} {
		writeFile(t, path)
	}
	createLibrary(t, db, root)

	res, err := New(db).Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Files != 4 || res.Identified != 0 || res.Unidentified != 0 {
		t.Fatalf("resultado inesperado: %+v", res)
	}

	row := readSeriesByFolder(t, db, "Juego de tronos")
	if row.id == 0 || row.folder != "Juego de tronos" || row.title != "Juego de tronos" {
		t.Fatalf("serie inesperada: %+v", row)
	}
	if !row.sortTitle.Valid || row.sortTitle.String != "Juego de tronos" ||
		!row.sortLetter.Valid || row.sortLetter.String != "J" {
		t.Fatalf("claves de orden inesperadas: %+v %+v", row.sortTitle, row.sortLetter)
	}

	for _, path := range []string{first, second} {
		var seriesID sql.NullInt64
		if err := db.QueryRow(`SELECT series_id FROM media_items WHERE path = ?`, path).
			Scan(&seriesID); err != nil {
			t.Fatalf("leer serie de %q: %v", path, err)
		}
		if !seriesID.Valid || seriesID.Int64 != row.id {
			t.Fatalf("series_id de %q = %+v, se esperaba %d", path, seriesID, row.id)
		}
	}

	for _, path := range []string{loose, movie} {
		var seriesID sql.NullInt64
		if err := db.QueryRow(`SELECT series_id FROM media_items WHERE path = ?`, path).
			Scan(&seriesID); err != nil {
			t.Fatalf("leer serie de %q: %v", path, err)
		}
		if seriesID.Valid {
			t.Fatalf("%q no debe tener serie: %+v", path, seriesID)
		}
	}

	// Un segundo escaneo actualiza pero no duplica series ni episodios.
	if _, err := New(db).Scan(ctx); err != nil {
		t.Fatalf("segundo Scan: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM series`).Scan(&count); err != nil {
		t.Fatalf("contar series: %v", err)
	}
	if count != 1 {
		t.Fatalf("series tras el segundo escaneo = %d", count)
	}
}

func TestScannerBorraSeriesHuérfanas(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	libraryID := createLibrary(t, db, root)

	for _, item := range []struct {
		folder string
		status any
	}{
		{folder: "Auto", status: "auto"},
		{folder: "Manual", status: "manual"},
		{folder: "Sin estado", status: nil},
	} {
		if _, err := db.Exec(`
			INSERT INTO series (library_id, folder, title, match_status)
			VALUES (?, ?, ?, ?)`, libraryID, item.folder, item.folder, item.status); err != nil {
			t.Fatalf("insertar serie %q: %v", item.folder, err)
		}
	}

	if _, err := New(db).Scan(ctx); err != nil {
		t.Fatalf("Scan: %v", err)
	}

	rows, err := db.Query(`SELECT folder FROM series ORDER BY folder`)
	if err != nil {
		t.Fatalf("listar series: %v", err)
	}
	defer rows.Close()
	var folders []string
	for rows.Next() {
		var folder string
		if err := rows.Scan(&folder); err != nil {
			t.Fatalf("leer serie: %v", err)
		}
		folders = append(folders, folder)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("recorrer series: %v", err)
	}
	if len(folders) != 1 || folders[0] != "Manual" {
		t.Fatalf("series conservadas = %v, se esperaba solo Manual", folders)
	}
}

func TestScannerConTMDBIdentificaSerieYRellenaTemporadas(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()

	paths := []string{
		filepath.Join(root, "The Big Bang theory", "Temporada 1", "1x01.mkv"),
		filepath.Join(root, "The Big Bang theory", "Temporada 1", "1x02.mkv"),
		filepath.Join(root, "The Big Bang theory", "Temporada 2", "2x01.mkv"),
	}
	for _, path := range paths {
		writeFile(t, path)
	}
	createLibrary(t, db, root)

	fake := &fakeTMDB{
		tvSearchResults: map[string][]metadata.Candidate{
			"The Big Bang theory": {{
				TMDBID: 1418, Title: "The Big Bang Theory", OriginalTitle: "The Big Bang Theory",
				Year: 2007, VoteCount: 12000,
			}},
		},
		tvDetails: &metadata.Details{
			TMDBID: 1418, Title: "The Big Bang Theory", OriginalTitle: "The Big Bang Theory",
			Year: 2007, Tagline: "Una comedia", Overview: "Sinopsis de la serie",
			OverviewLanguage: "es-ES", Genres: []string{"Comedia"}, Rating: 8.2,
			Directors: []string{"Chuck Lorre"}, Cast: []string{"Jim Parsons"},
			PosterURL:   "https://images.example/serie.jpg",
			BackdropURL: "https://images.example/fondo.jpg",
		},
		seasons: map[int]*metadata.Season{
			1: {
				Name: "Temporada 1", Overview: "Primera temporada", OverviewLanguage: "es-ES",
				PosterURL: "https://images.example/t1.jpg", AirYear: 2007,
				Episodes: []metadata.SeasonEpisode{
					{Number: 1, Name: "Piloto", Overview: "Sinopsis 1", AirYear: 2007,
						StillURL: "https://images.example/e1.jpg"},
					{Number: 2, Name: "Segundo", Overview: "Sinopsis 2", AirYear: 2007,
						StillURL: "https://images.example/e2.jpg"},
				},
			},
			2: {
				Name: "Temporada 2", Overview: "Segunda temporada", OverviewLanguage: "es-ES",
				PosterURL: "https://images.example/t2.jpg", AirYear: 2007,
				Episodes: []metadata.SeasonEpisode{
					{Number: 1, Name: "Estreno T2", Overview: "Sinopsis T2", AirYear: 2007,
						StillURL: "https://images.example/e3.jpg"},
				},
			},
		},
	}

	scanner := New(db).WithTMDB(fake)
	res, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Files != 3 || res.Identified != 1 || res.SeriesIdentified != 1 ||
		res.Unidentified != 0 || res.SeriesUnidentified != 0 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("avisos inesperados: %v", res.Warnings)
	}
	if fake.tvSearchCalls != 1 || fake.tvDetailsCalls != 1 || fake.seasonCalls != 2 {
		t.Fatalf("peticiones: búsquedas %d, fichas %d, temporadas %d",
			fake.tvSearchCalls, fake.tvDetailsCalls, fake.seasonCalls)
	}
	if fake.lastTVDetailsID != 1418 {
		t.Fatalf("última ficha = %d", fake.lastTVDetailsID)
	}

	row := readSeriesByFolder(t, db, "The Big Bang theory")
	if row.title != "The Big Bang theory" ||
		!row.displayTitle.Valid || row.displayTitle.String != "The Big Bang Theory" ||
		!row.originalTitle.Valid || row.originalTitle.String != "The Big Bang Theory" ||
		!row.tmdbID.Valid || row.tmdbID.Int64 != 1418 ||
		!row.source.Valid || row.source.String != "tmdb" ||
		!row.matchStatus.Valid || row.matchStatus.String != "auto" ||
		!row.matchScore.Valid || row.matchScore.Float64 < MatchThreshold ||
		!row.overview.Valid || row.overview.String != "Sinopsis de la serie" ||
		!row.sortTitle.Valid || row.sortTitle.String != "Big Bang Theory" ||
		!row.sortLetter.Valid || row.sortLetter.String != "B" {
		t.Fatalf("serie inesperada: %+v", row)
	}

	wantEpisodes := map[string]struct {
		title    string
		overview string
		still    string
	}{
		paths[0]: {"Piloto", "Sinopsis 1", "https://images.example/e1.jpg"},
		paths[1]: {"Segundo", "Sinopsis 2", "https://images.example/e2.jpg"},
		paths[2]: {"Estreno T2", "Sinopsis T2", "https://images.example/e3.jpg"},
	}
	for path, want := range wantEpisodes {
		episode := readEpisodeMetadata(t, db, path)
		if !episode.title.Valid || episode.title.String != want.title ||
			!episode.overview.Valid || episode.overview.String != want.overview ||
			!episode.still.Valid || episode.still.String != want.still ||
			!episode.year.Valid || episode.year.Int64 != 2007 {
			t.Fatalf("episodio %q inesperado: %+v", path, episode)
		}
	}

	var seasons int
	if err := db.QueryRow(`SELECT count(*) FROM series_seasons WHERE series_id = ?`,
		row.id).Scan(&seasons); err != nil {
		t.Fatalf("contar temporadas: %v", err)
	}
	if seasons != 2 {
		t.Fatalf("temporadas = %d, se esperaban 2", seasons)
	}
	var seasonName, seasonOverview, seasonPoster sql.NullString
	if err := db.QueryRow(`
		SELECT name, overview, poster_url FROM series_seasons
		 WHERE series_id = ? AND season = 1`, row.id).
		Scan(&seasonName, &seasonOverview, &seasonPoster); err != nil {
		t.Fatalf("leer temporada 1: %v", err)
	}
	if !seasonName.Valid || seasonName.String != "Temporada 1" ||
		!seasonOverview.Valid || seasonOverview.String != "Primera temporada" ||
		!seasonPoster.Valid || seasonPoster.String != "https://images.example/t1.jpg" {
		t.Fatalf("temporada 1 inesperada: %+v %+v %+v",
			seasonName, seasonOverview, seasonPoster)
	}

	// El segundo escaneo no repite búsquedas, fichas ni temporadas.
	fake.tvSearchCalls = 0
	fake.tvDetailsCalls = 0
	fake.seasonCalls = 0
	res, err = scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("segundo Scan: %v", err)
	}
	if res.Identified != 0 || res.SeriesIdentified != 0 ||
		res.Unidentified != 0 || res.SeriesUnidentified != 0 {
		t.Fatalf("contadores del segundo escaneo: %+v", res)
	}
	if fake.tvSearchCalls != 0 || fake.tvDetailsCalls != 0 || fake.seasonCalls != 0 {
		t.Fatalf("peticiones del segundo escaneo: %d, %d, %d",
			fake.tvSearchCalls, fake.tvDetailsCalls, fake.seasonCalls)
	}
}

func TestScannerRespetaSeriesManualYNone(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()

	manualPath := filepath.Join(root, "Manual", "Temporada 1", "1x01.mkv")
	nonePath := filepath.Join(root, "None", "Temporada 1", "1x01.mkv")
	writeFile(t, manualPath)
	writeFile(t, nonePath)
	createLibrary(t, db, root)

	if _, err := New(db).Scan(ctx); err != nil {
		t.Fatalf("primer Scan: %v", err)
	}
	for _, item := range []struct {
		folder string
		status string
		title  string
	}{
		{folder: "Manual", status: "manual", title: "Ficha manual"},
		{folder: "None", status: "none", title: "Ficha none"},
	} {
		if _, err := db.Exec(`
			UPDATE series
			   SET match_status = ?, metadata_source = 'tmdb', tmdb_id = 7,
			       display_title = ?, overview = 'Sinopsis fijada', year = 2001,
			       sort_title = ?, sort_letter = 'F',
			       credits = '{"cast":[],"crew":[]}'
			 WHERE folder = ?`,
			item.status, item.title, item.title, item.folder); err != nil {
			t.Fatalf("fijar serie %q: %v", item.folder, err)
		}
	}

	fake := &fakeTMDB{
		tvSearchResults: map[string][]metadata.Candidate{
			"Manual": {{TMDBID: 1, Title: "Manual", VoteCount: 1000}},
			"None":   {{TMDBID: 2, Title: "None", VoteCount: 1000}},
		},
	}
	res, err := New(db).WithTMDB(fake).Scan(ctx)
	if err != nil {
		t.Fatalf("segundo Scan: %v", err)
	}
	if res.Identified != 0 || res.Unidentified != 0 ||
		res.SeriesIdentified != 0 || res.SeriesUnidentified != 0 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if fake.tvSearchCalls != 0 || fake.tvDetailsCalls != 0 || fake.seasonCalls != 0 {
		t.Fatalf("no se debe consultar TMDB: %d, %d, %d",
			fake.tvSearchCalls, fake.tvDetailsCalls, fake.seasonCalls)
	}
	for _, folder := range []string{"Manual", "None"} {
		row := readSeriesByFolder(t, db, folder)
		if !row.displayTitle.Valid || row.displayTitle.String == "" ||
			!row.overview.Valid || row.overview.String != "Sinopsis fijada" {
			t.Fatalf("la serie %q cambió: %+v", folder, row)
		}
	}
}

func TestScannerReintentaSerieSinIdentificar(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	path := filepath.Join(root, "Desconocida", "Temporada 1", "1x01.mkv")
	writeFile(t, path)
	createLibrary(t, db, root)

	fake := &fakeTMDB{tvSearchResults: map[string][]metadata.Candidate{}}
	scanner := New(db).WithTMDB(fake)
	res, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Identified != 0 || res.Unidentified != 1 ||
		res.SeriesIdentified != 0 || res.SeriesUnidentified != 1 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	row := readSeriesByFolder(t, db, "Desconocida")
	if row.matchStatus.Valid || row.tmdbID.Valid {
		t.Fatalf("la serie no debe quedar identificada: %+v", row)
	}
	if fake.tvDetailsCalls != 0 || fake.seasonCalls != 0 {
		t.Fatalf("no se debe pedir ficha ni temporadas: %d, %d",
			fake.tvDetailsCalls, fake.seasonCalls)
	}

	fake.tvSearchResults["Desconocida"] = []metadata.Candidate{{
		TMDBID: 900, Title: "Desconocida", OriginalTitle: "Unknown", Year: 2020,
		VoteCount: 1000,
	}}
	fake.tvDetails = &metadata.Details{
		TMDBID: 900, Title: "Desconocida", Year: 2020, Overview: "Sinopsis",
	}
	fake.seasons = map[int]*metadata.Season{
		1: {Name: "Temporada 1", Episodes: []metadata.SeasonEpisode{
			{Number: 1, Name: "Uno", AirYear: 2020},
		}},
	}

	res, err = scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("segundo Scan: %v", err)
	}
	if res.Identified != 1 || res.SeriesIdentified != 1 ||
		res.Unidentified != 0 || res.SeriesUnidentified != 0 {
		t.Fatalf("resultado del reintento: %+v", res)
	}
	row = readSeriesByFolder(t, db, "Desconocida")
	if !row.matchStatus.Valid || row.matchStatus.String != "auto" ||
		!row.tmdbID.Valid || row.tmdbID.Int64 != 900 {
		t.Fatalf("la serie no se identificó en el reintento: %+v", row)
	}
	episode := readEpisodeMetadata(t, db, path)
	if !episode.title.Valid || episode.title.String != "Uno" {
		t.Fatalf("episodio sin rellenar: %+v", episode)
	}
}

func TestScannerAvisaTemporadaInexistenteUnaVez(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	paths := []string{
		filepath.Join(root, "Serie", "Temporada 3", "3x01.mkv"),
		filepath.Join(root, "Serie", "Temporada 3", "3x02.mkv"),
	}
	for _, path := range paths {
		writeFile(t, path)
	}
	createLibrary(t, db, root)

	fake := &fakeTMDB{
		tvSearchResults: map[string][]metadata.Candidate{
			"Serie": {{TMDBID: 10, Title: "Serie", VoteCount: 1000}},
		},
		tvDetails: &metadata.Details{TMDBID: 10, Title: "Serie", Year: 2000},
	}
	res, err := New(db).WithTMDB(fake).Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Identified != 1 || res.SeriesIdentified != 1 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if fake.seasonCalls != 1 {
		t.Fatalf("temporadas consultadas = %d, se esperaba 1", fake.seasonCalls)
	}
	if len(res.Warnings) != 1 || !strings.Contains(res.Warnings[0], "temporada 3") {
		t.Fatalf("avisos = %v", res.Warnings)
	}
	for _, path := range paths {
		episode := readEpisodeMetadata(t, db, path)
		if episode.title.Valid {
			t.Fatalf("el episodio %q no debería tener título: %+v", path, episode)
		}
	}
}

func TestScannerRellenaCreditsSeriesAntiguas(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	path := filepath.Join(root, "Serie", "Temporada 1", "1x01.mkv")
	writeFile(t, path)
	createLibrary(t, db, root)

	if _, err := New(db).Scan(ctx); err != nil {
		t.Fatalf("primer Scan: %v", err)
	}
	var seriesID int64
	if err := db.QueryRow(`SELECT id FROM series WHERE folder = 'Serie'`).Scan(&seriesID); err != nil {
		t.Fatalf("leer serie: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE series
		   SET tmdb_id = 7, metadata_source = 'tmdb', match_status = 'auto',
		       display_title = 'Serie', overview = 'Sinopsis antigua', year = 2001,
		       match_score = 0.9, credits = NULL
		 WHERE id = ?`, seriesID); err != nil {
		t.Fatalf("preparar serie antigua: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO series_seasons (series_id, season, name) VALUES (?, 1, 'Temporada 1')`,
		seriesID); err != nil {
		t.Fatalf("crear temporada completa: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE media_items SET episode_title = 'Episodio' WHERE series_id = ?`, seriesID); err != nil {
		t.Fatalf("completar episodio: %v", err)
	}

	fake := &fakeTMDB{
		tvDetails: &metadata.Details{
			TMDBID: 7, Title: "Serie",
			Credits: metadata.Credits{
				Cast: []metadata.CastCredit{{
					Name:      "Actriz",
					Character: "Papel",
					PhotoURL:  "https://images.example/w185/actriz.jpg",
				}},
				Crew: []metadata.CrewCredit{{Name: "Creadora", Job: "Creator"}},
			},
		},
	}
	res, err := New(db).WithTMDB(fake).Scan(ctx)
	if err != nil {
		t.Fatalf("segundo Scan: %v", err)
	}
	if res.Identified != 0 || res.SeriesIdentified != 0 {
		t.Fatalf("el relleno no debe contar: %+v", res)
	}
	if fake.tvSearchCalls != 0 || fake.tvDetailsCalls != 1 || fake.seasonCalls != 0 {
		t.Fatalf("peticiones del relleno: %d, %d, %d",
			fake.tvSearchCalls, fake.tvDetailsCalls, fake.seasonCalls)
	}

	var rawCredits sql.NullString
	var status string
	var score sql.NullFloat64
	if err := db.QueryRow(`SELECT credits, match_status, match_score FROM series WHERE id = ?`,
		seriesID).Scan(&rawCredits, &status, &score); err != nil {
		t.Fatalf("leer serie rellenada: %v", err)
	}
	if !rawCredits.Valid || status != "auto" || !score.Valid || score.Float64 != 0.9 {
		t.Fatalf("la serie antigua cambió: %+v %q %+v", rawCredits, status, score)
	}
	decoded := decodeCredits(rawCredits)
	if len(decoded.Cast) != 1 || decoded.Cast[0].Character != "Papel" ||
		decoded.Cast[0].PhotoURL != "https://images.example/w185/actriz.jpg" {
		t.Fatalf("créditos de la serie inesperados: %+v", decoded)
	}

	fake.tvSearchCalls = 0
	fake.tvDetailsCalls = 0
	fake.seasonCalls = 0
	if _, err := New(db).WithTMDB(fake).Scan(ctx); err != nil {
		t.Fatalf("tercer Scan: %v", err)
	}
	if fake.tvSearchCalls != 0 || fake.tvDetailsCalls != 0 || fake.seasonCalls != 0 {
		t.Fatalf("el tercer escaneo no debe pedir TMDB: %d, %d, %d",
			fake.tvSearchCalls, fake.tvDetailsCalls, fake.seasonCalls)
	}
}

func TestScannerRellenaPosterTemporadasSinPedirEpisodios(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	path := filepath.Join(root, "Serie", "Temporada 1", "1x01.mkv")
	writeFile(t, path)
	createLibrary(t, db, root)

	if _, err := New(db).Scan(ctx); err != nil {
		t.Fatalf("primer Scan: %v", err)
	}
	var seriesID int64
	if err := db.QueryRow(`SELECT id FROM series WHERE folder = 'Serie'`).Scan(&seriesID); err != nil {
		t.Fatalf("leer serie: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE series
		   SET tmdb_id = 7, metadata_source = 'tmdb', match_status = 'auto',
		       display_title = 'Serie', match_score = 0.8,
		       credits = '{"cast":[],"crew":[]}'
		 WHERE id = ?`, seriesID); err != nil {
		t.Fatalf("preparar serie antigua: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO series_seasons (series_id, season, name) VALUES (?, 1, 'Temporada 1')`,
		seriesID); err != nil {
		t.Fatalf("crear temporada sin cartel: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE media_items SET episode_title = 'Episodio' WHERE series_id = ?`, seriesID); err != nil {
		t.Fatalf("completar episodio: %v", err)
	}

	fake := &fakeTMDB{
		tvDetails: &metadata.Details{
			TMDBID: 7, Title: "Serie",
			SeasonPosters: map[int]string{1: "https://images.example/t1.jpg"},
		},
	}
	res, err := New(db).WithTMDB(fake).Scan(ctx)
	if err != nil {
		t.Fatalf("segundo Scan: %v", err)
	}
	if res.Identified != 0 || res.SeriesIdentified != 0 {
		t.Fatalf("el relleno no debe contar: %+v", res)
	}
	if fake.tvSearchCalls != 0 || fake.tvDetailsCalls != 1 || fake.seasonCalls != 0 {
		t.Fatalf("peticiones del relleno: %d, %d, %d",
			fake.tvSearchCalls, fake.tvDetailsCalls, fake.seasonCalls)
	}

	var poster sql.NullString
	var status string
	var score sql.NullFloat64
	if err := db.QueryRow(`
		SELECT ss.poster_url, s.match_status, s.match_score
		  FROM series_seasons ss JOIN series s ON s.id = ss.series_id
		 WHERE ss.series_id = ? AND ss.season = 1`, seriesID).
		Scan(&poster, &status, &score); err != nil {
		t.Fatalf("leer temporada rellenada: %v", err)
	}
	if !poster.Valid || poster.String != "https://images.example/t1.jpg" ||
		status != "auto" || !score.Valid || score.Float64 != 0.8 {
		t.Fatalf("temporada rellenada inesperada: %+v %q %+v", poster, status, score)
	}

	fake.tvSearchCalls = 0
	fake.tvDetailsCalls = 0
	fake.seasonCalls = 0
	if _, err := New(db).WithTMDB(fake).Scan(ctx); err != nil {
		t.Fatalf("tercer Scan: %v", err)
	}
	if fake.tvSearchCalls != 0 || fake.tvDetailsCalls != 0 || fake.seasonCalls != 0 {
		t.Fatalf("el tercer escaneo no debe pedir TMDB: %d, %d, %d",
			fake.tvSearchCalls, fake.tvDetailsCalls, fake.seasonCalls)
	}
}

func TestApplyTVDetailsGuardaYClearCredits(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	libraryID := createLibrary(t, db, t.TempDir())
	seriesID, err := ensureSeries(ctx, db, libraryID, "Créditos", "Créditos", 0)
	if err != nil {
		t.Fatalf("ensureSeries: %v", err)
	}
	if err := ApplyTVDetails(ctx, db, seriesID, &metadata.Details{
		TMDBID: 2, Title: "Créditos",
		Credits: metadata.Credits{
			Cast: []metadata.CastCredit{{Name: "Actor", Character: "Personaje"}},
			Crew: []metadata.CrewCredit{{Name: "Creador", Job: "Creator"}},
		},
	}, "auto", 1); err != nil {
		t.Fatalf("ApplyTVDetails: %v", err)
	}
	var raw sql.NullString
	if err := db.QueryRow(`SELECT credits FROM series WHERE id = ?`, seriesID).Scan(&raw); err != nil {
		t.Fatalf("leer credits: %v", err)
	}
	decoded := decodeCredits(raw)
	if len(decoded.Cast) != 1 || decoded.Cast[0].Character != "Personaje" ||
		len(decoded.Crew) != 1 || decoded.Crew[0].Job != "Creator" {
		t.Fatalf("credits de serie inesperados: %+v", decoded)
	}
	if err := ClearTVMetadata(ctx, db, seriesID, nil); err != nil {
		t.Fatalf("ClearTVMetadata: %v", err)
	}
	if err := db.QueryRow(`SELECT credits FROM series WHERE id = ?`, seriesID).Scan(&raw); err != nil {
		t.Fatalf("leer credits tras limpiar: %v", err)
	}
	if raw.Valid {
		t.Fatalf("credits debe quedar NULL: %+v", raw)
	}
}

func TestApplyTVDetailsRespetaBloqueosYLosDescartaAlCambiarTMDB(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	libraryID := createLibrary(t, db, root)
	seriesID, err := ensureSeries(ctx, db, libraryID, "Serie", "Serie", 0)
	if err != nil {
		t.Fatalf("ensureSeries: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE series
		   SET tmdb_id = 10, poster_url = 'https://images.example/propio.jpg', poster_locked = 1,
		       backdrop_url = 'https://images.example/fondo-propio.jpg', backdrop_locked = 1
		 WHERE id = ?`, seriesID); err != nil {
		t.Fatalf("preparar bloqueos: %v", err)
	}

	same := &metadata.Details{
		TMDBID: 10, Title: "Serie de TMDB", OriginalTitle: "Original", Year: 2010,
		Tagline: "Lema", Overview: "Sinopsis", OverviewLanguage: "es-ES",
		Genres: []string{"Drama"}, Rating: 8.1, Directors: []string{"Creadora"},
		Cast: []string{"Actor"}, PosterURL: "https://images.example/tmdb.jpg",
		BackdropURL: "https://images.example/fondo-tmdb.jpg",
	}
	if err := ApplyTVDetails(ctx, db, seriesID, same, "auto", 0.9); err != nil {
		t.Fatalf("ApplyTVDetails mismo TMDB: %v", err)
	}
	row := readSeries(t, db, seriesID)
	if !row.posterURL.Valid || row.posterURL.String != "https://images.example/propio.jpg" ||
		row.posterLocked != 1 ||
		!row.backdropURL.Valid || row.backdropURL.String != "https://images.example/fondo-propio.jpg" ||
		row.backdropLocked != 1 {
		t.Fatalf("las imágenes fijadas cambiaron: %+v", row)
	}
	if row.title != "Serie" || !row.displayTitle.Valid || row.displayTitle.String != "Serie de TMDB" ||
		!row.year.Valid || row.year.Int64 != 2010 ||
		!row.genres.Valid || row.genres.String != `["Drama"]` ||
		!row.directors.Valid || row.directors.String != `["Creadora"]` ||
		!row.castNames.Valid || row.castNames.String != `["Actor"]` ||
		!row.source.Valid || row.source.String != "tmdb" ||
		!row.matchStatus.Valid || row.matchStatus.String != "auto" ||
		!row.matchScore.Valid || row.matchScore.Float64 != 0.9 ||
		!row.sortTitle.Valid || row.sortTitle.String != "Serie de TMDB" ||
		!row.sortLetter.Valid || row.sortLetter.String != "S" {
		t.Fatalf("ficha inesperada: %+v", row)
	}

	changed := *same
	changed.TMDBID = 11
	changed.PosterURL = "https://images.example/tmdb-nuevo.jpg"
	changed.BackdropURL = "https://images.example/fondo-nuevo.jpg"
	if err := ApplyTVDetails(ctx, db, seriesID, &changed, "auto", 0.8); err != nil {
		t.Fatalf("ApplyTVDetails con otro TMDB: %v", err)
	}
	row = readSeries(t, db, seriesID)
	if !row.posterURL.Valid || row.posterURL.String != "https://images.example/tmdb-nuevo.jpg" ||
		row.posterLocked != 0 ||
		!row.backdropURL.Valid || row.backdropURL.String != "https://images.example/fondo-nuevo.jpg" ||
		row.backdropLocked != 0 {
		t.Fatalf("las imágenes no se sustituyeron: %+v", row)
	}
}

func TestClearTVMetadataLimpiaTemporadasYEpisodios(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	libraryID := createLibrary(t, db, root)
	seriesID, err := ensureSeries(ctx, db, libraryID, "Serie", "Serie", 2000)
	if err != nil {
		t.Fatalf("ensureSeries: %v", err)
	}
	if err := ApplyTVDetails(ctx, db, seriesID, &metadata.Details{
		TMDBID: 10, Title: "Serie de TMDB", Year: 2000, Overview: "Sinopsis",
		PosterURL: "https://images.example/poster.jpg",
	}, "auto", 0.9); err != nil {
		t.Fatalf("ApplyTVDetails: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO series_seasons (series_id, season, name, overview, poster_url)
		VALUES (?, 1, 'Temporada 1', 'Sinopsis', 'https://images.example/t1.jpg')`,
		seriesID); err != nil {
		t.Fatalf("insertar temporada: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO media_items (
			library_id, path, title, kind, season, episode, series_id,
			episode_title, overview, overview_language, still_url, year
		) VALUES (?, ?, 'Serie', 'episode', 1, 1, ?, 'Piloto', 'Sinopsis', 'es-ES',
		          'https://images.example/still.jpg', 2001)`,
		libraryID, filepath.Join(root, "Serie", "Temporada 1", "1x01.mkv"), seriesID); err != nil {
		t.Fatalf("insertar episodio: %v", err)
	}

	if err := ClearTVMetadata(ctx, db, seriesID, nil); err != nil {
		t.Fatalf("ClearTVMetadata: %v", err)
	}
	row := readSeries(t, db, seriesID)
	if row.title != "Serie" || row.folder != "Serie" ||
		row.displayTitle.Valid || row.originalTitle.Valid || row.year.Valid ||
		row.tagline.Valid || row.overview.Valid || row.overviewLanguage.Valid ||
		row.posterURL.Valid || row.backdropURL.Valid || row.genres.Valid ||
		row.rating.Valid || row.directors.Valid || row.castNames.Valid ||
		row.tmdbID.Valid || row.source.Valid || row.matchStatus.Valid ||
		row.matchScore.Valid || row.posterLocked != 0 || row.backdropLocked != 0 {
		t.Fatalf("la serie no quedó limpia: %+v", row)
	}
	if !row.sortTitle.Valid || row.sortTitle.String != "Serie" ||
		!row.sortLetter.Valid || row.sortLetter.String != "S" {
		t.Fatalf("claves de orden inesperadas: %+v %+v", row.sortTitle, row.sortLetter)
	}
	var seasons int
	if err := db.QueryRow(`SELECT count(*) FROM series_seasons WHERE series_id = ?`,
		seriesID).Scan(&seasons); err != nil {
		t.Fatalf("contar temporadas: %v", err)
	}
	if seasons != 0 {
		t.Fatalf("quedaron %d temporadas", seasons)
	}
	episode := readEpisodeMetadata(t, db, filepath.Join(root, "Serie", "Temporada 1", "1x01.mkv"))
	if episode.title.Valid || episode.overview.Valid || episode.still.Valid ||
		episode.overviewLanguage.Valid {
		t.Fatalf("el episodio no quedó limpio: %+v", episode)
	}
	if !episode.year.Valid || episode.year.Int64 != 2001 {
		t.Fatalf("el año del archivo no debe borrarse: %+v", episode.year)
	}
}

type seriesMetadataRow struct {
	id               int64
	folder           string
	title            string
	displayTitle     sql.NullString
	originalTitle    sql.NullString
	year             sql.NullInt64
	tagline          sql.NullString
	overview         sql.NullString
	overviewLanguage sql.NullString
	posterURL        sql.NullString
	backdropURL      sql.NullString
	genres           sql.NullString
	rating           sql.NullFloat64
	directors        sql.NullString
	castNames        sql.NullString
	tmdbID           sql.NullInt64
	source           sql.NullString
	matchStatus      sql.NullString
	matchScore       sql.NullFloat64
	sortTitle        sql.NullString
	sortLetter       sql.NullString
	posterLocked     int
	backdropLocked   int
}

func readSeries(t *testing.T, db *sql.DB, seriesID int64) seriesMetadataRow {
	t.Helper()
	var row seriesMetadataRow
	if err := db.QueryRow(`
		SELECT id, folder, title, display_title, original_title, year, tagline,
		       overview, overview_language, poster_url, backdrop_url, genres,
		       rating, directors, cast_names, tmdb_id, metadata_source,
		       match_status, match_score, sort_title, sort_letter,
		       poster_locked, backdrop_locked
		  FROM series WHERE id = ?`, seriesID).Scan(
		&row.id, &row.folder, &row.title, &row.displayTitle, &row.originalTitle,
		&row.year, &row.tagline, &row.overview, &row.overviewLanguage,
		&row.posterURL, &row.backdropURL, &row.genres, &row.rating, &row.directors,
		&row.castNames, &row.tmdbID, &row.source, &row.matchStatus, &row.matchScore,
		&row.sortTitle, &row.sortLetter, &row.posterLocked, &row.backdropLocked,
	); err != nil {
		t.Fatalf("leer serie %d: %v", seriesID, err)
	}
	return row
}

func readSeriesByFolder(t *testing.T, db *sql.DB, folder string) seriesMetadataRow {
	t.Helper()
	var id int64
	if err := db.QueryRow(`SELECT id FROM series WHERE folder = ?`, folder).Scan(&id); err != nil {
		t.Fatalf("leer id de la serie %q: %v", folder, err)
	}
	return readSeries(t, db, id)
}

type episodeMetadataRow struct {
	title            sql.NullString
	overview         sql.NullString
	overviewLanguage sql.NullString
	still            sql.NullString
	year             sql.NullInt64
}

func readEpisodeMetadata(t *testing.T, db *sql.DB, path string) episodeMetadataRow {
	t.Helper()
	var row episodeMetadataRow
	if err := db.QueryRow(`
		SELECT episode_title, overview, overview_language, still_url, year
		  FROM media_items WHERE path = ?`, path).Scan(
		&row.title, &row.overview, &row.overviewLanguage, &row.still, &row.year,
	); err != nil {
		t.Fatalf("leer episodio %q: %v", path, err)
	}
	return row
}
