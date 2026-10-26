package library

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/catlinux/cinecat/internal/metadata"
)

type seriesHTTPFixture struct {
	libraryID        int64
	folder           string
	title            string
	displayTitle     string
	originalTitle    string
	year             int
	tagline          string
	overview         string
	overviewLanguage string
	posterURL        string
	backdropURL      string
	genres           string
	rating           float64
	directors        string
	cast             string
	tmdbID           int64
	metadataSource   string
	matchStatus      string
	matchScore       float64
	posterLocked     int
	backdropLocked   int
}

func insertSeriesHTTPFixture(t *testing.T, ts *libraryTestServer, item seriesHTTPFixture) int64 {
	t.Helper()
	sortSource := item.displayTitle
	if sortSource == "" {
		sortSource = item.title
	}
	sortTitle, sortLetter := sortKeys(sortSource)
	res, err := ts.db.Exec(`
		INSERT INTO series (
			library_id, folder, title, display_title, original_title, year,
			tagline, overview, overview_language, poster_url, backdrop_url,
			genres, rating, directors, cast_names, tmdb_id, metadata_source,
			match_status, match_score, sort_title, sort_letter, poster_locked,
			backdrop_locked
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.libraryID, item.folder, item.title,
		nullableTestString(item.displayTitle), nullableTestString(item.originalTitle),
		nullableTestInt(item.year), nullableTestString(item.tagline),
		nullableTestString(item.overview), nullableTestString(item.overviewLanguage),
		nullableTestString(item.posterURL), nullableTestString(item.backdropURL),
		nullableTestString(item.genres), nullableTestFloat(item.rating),
		nullableTestString(item.directors), nullableTestString(item.cast),
		nullableTestInt(int(item.tmdbID)), nullableTestString(item.metadataSource),
		nullableTestString(item.matchStatus), nullableTestFloat(item.matchScore),
		sortTitle, sortLetter, item.posterLocked, item.backdropLocked)
	if err != nil {
		t.Fatalf("insertar serie %q: %v", item.title, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("id de la serie %q: %v", item.title, err)
	}
	return id
}

func insertSeriesSeasonHTTP(t *testing.T, ts *libraryTestServer, seriesID int64, season int, name, overview, poster string) {
	t.Helper()
	if _, err := ts.db.Exec(`
		INSERT INTO series_seasons (series_id, season, name, overview, poster_url)
		VALUES (?, ?, ?, ?, ?)`,
		seriesID, season, nullableTestString(name), nullableTestString(overview),
		nullableTestString(poster)); err != nil {
		t.Fatalf("insertar temporada %d: %v", season, err)
	}
}

func insertSeriesEpisodeHTTP(
	t *testing.T,
	ts *libraryTestServer,
	libraryID, seriesID int64,
	title, fileTitle string,
	season, episode int,
	episodeTitle, stillURL, overview string,
) int64 {
	t.Helper()
	return insertItemFixture(t, ts.db, itemFixture{
		libraryID:    libraryID,
		path:         fmt.Sprintf("/media/series/%d/%d/%d-%d.mkv", seriesID, season, season, episode),
		title:        fileTitle,
		displayTitle: title,
		kind:         KindEpisode,
		season:       season,
		episode:      episode,
		seriesID:     seriesID,
		episodeTitle: episodeTitle,
		stillURL:     stillURL,
		overview:     overview,
	})
}

func seriesHTTPURL(ts *libraryTestServer, seriesID int64, suffix string) string {
	return ts.URL + "/api/v1/series/" + strconv.FormatInt(seriesID, 10) + suffix
}

func TestSeriesListadoYGeneros(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())

	chicago := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Chicago fire", title: "Chicago fire",
		displayTitle: "Chicago Fire", originalTitle: "Chicago Fire", year: 2012,
		posterURL: "https://images.example/chicago.jpg",
		genres:    `["Drama","Acción"]`, directors: `["Derek Haas"]`,
		cast: `["Taylor Kinney"]`, tmdbID: 44006, metadataSource: "tmdb",
		matchStatus: "auto", matchScore: 1,
	})
	mrRobot := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Mr Robot", title: "Mr Robot",
		displayTitle: "Mr. Robot", originalTitle: "Mr. Robot", year: 2015,
		genres: `["Drama"]`, cast: `["Rami Malek"]`, tmdbID: 62560,
		metadataSource: "tmdb", matchStatus: "manual", matchScore: 1,
	})
	juego := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Juego de tronos", title: "Juego de tronos",
		displayTitle: "Juego de Tronos", year: 2011,
		genres: `["Drama","Fantasía"]`, cast: `["Emilia Clarke"]`,
	})
	unknown := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Sin identificar", title: "Sin identificar",
		displayTitle: "Sin identificar", matchStatus: "none",
	})

	insertSeriesEpisodeHTTP(t, ts, libraryID, chicago, "Piloto", "ChicagoFire-1x01", 1, 1,
		"Piloto", "https://images.example/still1.jpg", "Sinopsis 1")
	insertSeriesEpisodeHTTP(t, ts, libraryID, chicago, "Segundo", "ChicagoFire-1x02", 1, 2,
		"Segundo", "https://images.example/still2.jpg", "Sinopsis 2")
	insertSeriesEpisodeHTTP(t, ts, libraryID, chicago, "Estreno", "ChicagoFire-2x01", 2, 1,
		"Estreno", "https://images.example/still3.jpg", "Sinopsis 3")
	insertSeriesEpisodeHTTP(t, ts, libraryID, mrRobot, "Hello Friend", "MrRobot-1x01", 1, 1,
		"Hello Friend", "https://images.example/mr.jpg", "Sinopsis Mr")
	insertSeriesEpisodeHTTP(t, ts, libraryID, juego, "Invierno", "Juego-1x01", 1, 1,
		"Invierno", "https://images.example/juego.jpg", "Sinopsis Juego")

	anaID := progressUserID(t, ts.db, "ana")
	var chicagoFirst int64
	if err := ts.db.QueryRow(`
		SELECT id FROM media_items WHERE series_id = ? AND season = 1 AND episode = 1`,
		chicago).Scan(&chicagoFirst); err != nil {
		t.Fatalf("leer primer episodio: %v", err)
	}
	setWatchState(t, ts.db, anaID, chicagoFirst)

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/series", ts.token)
	if status != http.StatusOK {
		t.Fatalf("listar series: código = %d, cuerpo = %s", status, body)
	}
	var list seriesResponse
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("listar series: %v", err)
	}
	if list.Total != 4 || list.Limit != 48 || list.Offset != 0 || len(list.Series) != 4 {
		t.Fatalf("listado inesperado: %+v", list)
	}
	wantOrder := []int64{chicago, juego, mrRobot, unknown}
	for i, want := range wantOrder {
		if list.Series[i].ID != want {
			t.Fatalf("orden %d = %d, se esperaba %d", i, list.Series[i].ID, want)
		}
	}
	if list.Series[0].Title != "Chicago Fire" || list.Series[0].FileTitle != "Chicago fire" ||
		list.Series[0].SeasonCount != 2 || list.Series[0].EpisodeCount != 3 ||
		list.Series[0].WatchedCount != 1 ||
		list.Series[0].MatchStatus == nil || *list.Series[0].MatchStatus != "auto" {
		t.Fatalf("serie Chicago inesperada: %+v", list.Series[0])
	}
	if list.Series[2].WatchedCount != 0 || list.Series[2].SeasonCount != 1 {
		t.Fatalf("serie Mr Robot inesperada: %+v", list.Series[2])
	}
	if list.Series[3].Genres == nil || list.Series[3].SeasonCount != 0 ||
		list.Series[3].EpisodeCount != 0 {
		t.Fatalf("serie sin géneros/episodios inesperada: %+v", list.Series[3])
	}

	// q incluye reparto y título; letter, genre, unmatched y orden.
	for query, wantID := range map[string]int64{
		"Taylor": chicago,
		"Tronos": juego,
		"Rami":   mrRobot,
	} {
		status, body := libraryHTTPGet(t, ts.URL+"/api/v1/series?q="+query, ts.token)
		if status != http.StatusOK {
			t.Fatalf("q=%s: %d %s", query, status, body)
		}
		var filtered seriesResponse
		if err := json.Unmarshal(body, &filtered); err != nil || len(filtered.Series) != 1 ||
			filtered.Series[0].ID != wantID {
			t.Fatalf("q=%s inesperado: %+v (%v)", query, filtered, err)
		}
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/series?letter=J", ts.token)
	if status != http.StatusOK {
		t.Fatalf("letter: %d %s", status, body)
	}
	var byLetter seriesResponse
	if err := json.Unmarshal(body, &byLetter); err != nil || len(byLetter.Series) != 1 ||
		byLetter.Series[0].ID != juego {
		t.Fatalf("letter inesperado: %+v (%v)", byLetter, err)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/series?genre=Acci%C3%B3n", ts.token)
	if status != http.StatusOK {
		t.Fatalf("genre: %d %s", status, body)
	}
	var byGenre seriesResponse
	if err := json.Unmarshal(body, &byGenre); err != nil || len(byGenre.Series) != 1 ||
		byGenre.Series[0].ID != chicago {
		t.Fatalf("genre inesperado: %+v (%v)", byGenre, err)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/series?unmatched=true", ts.token)
	if status != http.StatusOK {
		t.Fatalf("unmatched: %d %s", status, body)
	}
	var byUnmatched seriesResponse
	if err := json.Unmarshal(body, &byUnmatched); err != nil || len(byUnmatched.Series) != 2 {
		t.Fatalf("unmatched inesperado: %+v (%v)", byUnmatched, err)
	}
	if byUnmatched.Series[0].ID != juego || byUnmatched.Series[1].ID != unknown {
		t.Fatalf("unmatched ids inesperados: %+v", byUnmatched.Series)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/series?sort=year", ts.token)
	if status != http.StatusOK {
		t.Fatalf("sort year: %d %s", status, body)
	}
	var byYear seriesResponse
	if err := json.Unmarshal(body, &byYear); err != nil {
		t.Fatalf("sort year: %v", err)
	}
	if len(byYear.Series) != 4 || byYear.Series[0].ID != mrRobot ||
		byYear.Series[1].ID != chicago || byYear.Series[2].ID != juego ||
		byYear.Series[3].ID != unknown {
		t.Fatalf("orden por año inesperado: %+v", byYear.Series)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/series?sort=added&limit=1&offset=1", ts.token)
	if status != http.StatusOK {
		t.Fatalf("paginación: %d %s", status, body)
	}
	var page seriesResponse
	if err := json.Unmarshal(body, &page); err != nil || page.Total != 4 ||
		page.Limit != 1 || page.Offset != 1 || len(page.Series) != 1 ||
		page.Series[0].ID != juego {
		t.Fatalf("paginación inesperada: %+v (%v)", page, err)
	}

	// El recuento de vistos es por usuario.
	beaToken := nonAdminToken(t, ts)
	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/series", beaToken)
	if status != http.StatusOK {
		t.Fatalf("series de bea: %d %s", status, body)
	}
	var beaList seriesResponse
	if err := json.Unmarshal(body, &beaList); err != nil {
		t.Fatalf("series de bea: %v", err)
	}
	for _, item := range beaList.Series {
		if item.ID == chicago && item.WatchedCount != 0 {
			t.Fatalf("bea no debe heredar vistos: %+v", item)
		}
	}

	for _, query := range []string{
		"/api/v1/series?sort=otro", "/api/v1/series?letter=xy",
		"/api/v1/series?unmatched=quizá", "/api/v1/series?limit=0",
		"/api/v1/series?limit=201", "/api/v1/series?offset=-1",
	} {
		if status, body := libraryHTTPGet(t, ts.URL+query, ts.token); status != http.StatusBadRequest {
			t.Fatalf("%s: código = %d, cuerpo = %s", query, status, body)
		}
	}
	_ = anaID

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/series/genres", ts.token)
	if status != http.StatusOK {
		t.Fatalf("géneros de series: %d %s", status, body)
	}
	var genres genresResponse
	if err := json.Unmarshal(body, &genres); err != nil {
		t.Fatalf("géneros de series: %v", err)
	}
	counts := map[string]int64{}
	for _, genre := range genres.Genres {
		counts[genre.Name] = genre.Count
	}
	if counts["Drama"] != 3 || counts["Acción"] != 1 || counts["Fantasía"] != 1 {
		t.Fatalf("géneros inesperados: %+v", genres.Genres)
	}
}

func TestSeriesFichaTemporadaYNextEpisode(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	seriesID := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "The Big Bang theory (2007)",
		title: "The Big Bang theory", displayTitle: "The Big Bang Theory",
		originalTitle: "The Big Bang Theory", year: 2007,
		tagline: "Una comedia", overview: "Sinopsis de la serie",
		overviewLanguage: "es-ES", posterURL: "https://images.example/serie.jpg",
		backdropURL: "https://images.example/fondo.jpg",
		genres:      `["Comedia"]`, rating: 8.2, directors: `["Chuck Lorre"]`,
		cast: `["Jim Parsons"]`, tmdbID: 1418, metadataSource: "tmdb",
		matchStatus: "auto", matchScore: 0.99,
	})
	insertSeriesSeasonHTTP(t, ts, seriesID, 1, "Temporada uno", "Primera temporada",
		"https://images.example/t1.jpg")
	first := insertSeriesEpisodeHTTP(t, ts, libraryID, seriesID, "Piloto", "1x01",
		1, 1, "Piloto", "https://images.example/e1.jpg", "Sinopsis 1")
	second := insertSeriesEpisodeHTTP(t, ts, libraryID, seriesID, "Segundo", "1x02",
		1, 2, "Segundo", "https://images.example/e2.jpg", "Sinopsis 2")
	third := insertSeriesEpisodeHTTP(t, ts, libraryID, seriesID, "Estreno", "2x01",
		2, 1, "Estreno", "https://images.example/e3.jpg", "Sinopsis 3")

	status, body := libraryHTTPGet(t, seriesHTTPURL(ts, seriesID, ""), ts.token)
	if status != http.StatusOK {
		t.Fatalf("ficha: %d %s", status, body)
	}
	var detail seriesDetailDTO
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("ficha: %v", err)
	}
	if detail.ID != seriesID || detail.Title != "The Big Bang Theory" ||
		detail.FileTitle != "The Big Bang theory" || detail.Folder != "The Big Bang theory (2007)" ||
		detail.SeasonCount != 2 || detail.EpisodeCount != 3 || detail.WatchedCount != 0 ||
		len(detail.Seasons) != 2 || detail.NextEpisode == nil ||
		detail.NextEpisode.ID != first {
		t.Fatalf("ficha inesperada: %+v", detail)
	}
	if detail.Seasons[0].Name != "Temporada uno" || detail.Seasons[0].EpisodeCount != 2 ||
		detail.Seasons[0].WatchedCount != 0 ||
		detail.Seasons[1].Name != "Temporada 2" || detail.Seasons[1].EpisodeCount != 1 {
		t.Fatalf("temporadas inesperadas: %+v", detail.Seasons)
	}
	if detail.NextEpisode.EpisodeTitle == nil || *detail.NextEpisode.EpisodeTitle != "Piloto" ||
		detail.NextEpisode.SeriesTitle == nil || *detail.NextEpisode.SeriesTitle != "The Big Bang Theory" {
		t.Fatalf("next_episode inesperado: %+v", detail.NextEpisode)
	}

	anaID := progressUserID(t, ts.db, "ana")
	setWatchState(t, ts.db, anaID, first)
	status, body = libraryHTTPGet(t, seriesHTTPURL(ts, seriesID, ""), ts.token)
	if status != http.StatusOK {
		t.Fatalf("ficha con visto: %d %s", status, body)
	}
	detail = seriesDetailDTO{}
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("ficha con visto: %v", err)
	}
	if detail.WatchedCount != 1 || detail.NextEpisode == nil || detail.NextEpisode.ID != second ||
		detail.Seasons[0].WatchedCount != 1 {
		t.Fatalf("next_episode tras visto inesperado: %+v", detail)
	}

	setProgressRow(t, ts.db, anaID, third, 120, 600, "2026-10-26T12:00:00Z")
	status, body = libraryHTTPGet(t, seriesHTTPURL(ts, seriesID, ""), ts.token)
	if status != http.StatusOK {
		t.Fatalf("ficha con progreso: %d %s", status, body)
	}
	detail = seriesDetailDTO{}
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("ficha con progreso: %v", err)
	}
	if detail.NextEpisode == nil || detail.NextEpisode.ID != third ||
		detail.NextEpisode.Progress == nil ||
		detail.NextEpisode.Progress.PositionSeconds != 120 {
		t.Fatalf("next_episode con progreso inesperado: %+v", detail.NextEpisode)
	}

	status, body = libraryHTTPGet(t, seriesHTTPURL(ts, seriesID, "/seasons/1"), ts.token)
	if status != http.StatusOK {
		t.Fatalf("temporada 1: %d %s", status, body)
	}
	var season seriesSeasonDetailDTO
	if err := json.Unmarshal(body, &season); err != nil {
		t.Fatalf("temporada 1: %v", err)
	}
	if season.Season != 1 || season.Name != "Temporada uno" ||
		len(season.Items) != 2 || season.Items[0].ID != first || season.Items[1].ID != second {
		t.Fatalf("temporada 1 inesperada: %+v", season)
	}
	if season.Items[0].Title != "Piloto" || season.Items[0].FileTitle != "1x01" ||
		season.Items[0].EpisodeTitle == nil || *season.Items[0].EpisodeTitle != "Piloto" ||
		season.Items[0].StillURL == nil || *season.Items[0].StillURL != "https://images.example/e1.jpg" ||
		season.Items[0].SeriesTitle == nil || *season.Items[0].SeriesTitle != "The Big Bang Theory" ||
		!season.Items[0].Watched {
		t.Fatalf("episodio inesperado: %+v", season.Items[0])
	}

	status, body = libraryHTTPGet(t, seriesHTTPURL(ts, seriesID, "/seasons/2"), ts.token)
	if status != http.StatusOK {
		t.Fatalf("temporada 2: %d %s", status, body)
	}
	season = seriesSeasonDetailDTO{}
	if err := json.Unmarshal(body, &season); err != nil || season.Name != "Temporada 2" ||
		len(season.Items) != 1 || season.Items[0].ID != third {
		t.Fatalf("temporada 2 inesperada: %+v (%v)", season, err)
	}

	for _, suffix := range []string{"/seasons/9"} {
		if status, body := libraryHTTPGet(t, seriesHTTPURL(ts, seriesID, suffix), ts.token); status != http.StatusNotFound {
			t.Fatalf("%s: código = %d, cuerpo = %s", suffix, status, body)
		}
	}
	if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/series/999999", ts.token); status != http.StatusNotFound {
		t.Fatalf("serie inexistente: %d %s", status, body)
	}
}

func TestSeriesCorreccionImagenesYVisionadoEnBloque(t *testing.T) {
	fake := &fakeTMDB{
		tvSearchResults: map[string][]metadata.Candidate{
			"Chicago fire": {{TMDBID: 44006, Title: "Chicago Fire", Year: 2012, VoteCount: 5000}},
		},
		tvDetails: &metadata.Details{
			TMDBID: 44006, Title: "Chicago Fire", OriginalTitle: "Chicago Fire",
			Year: 2012, Overview: "Sinopsis", OverviewLanguage: "es-ES",
			Genres: []string{"Drama"}, Rating: 8,
			PosterURL:   "https://images.example/serie.jpg",
			BackdropURL: "https://images.example/fondo.jpg",
		},
		seasons: map[int]*metadata.Season{
			1: {Name: "Temporada 1", Episodes: []metadata.SeasonEpisode{
				{Number: 1, Name: "Piloto", Overview: "Sinopsis 1", AirYear: 2012},
				{Number: 2, Name: "Segundo", Overview: "Sinopsis 2", AirYear: 2012},
			}},
			2: {Name: "Temporada 2", Episodes: []metadata.SeasonEpisode{
				{Number: 1, Name: "T2", Overview: "Sinopsis 3", AirYear: 2012},
			}},
		},
		tvImages: &metadata.ImageOptions{
			Posters:   []metadata.ImageOption{{URL: "https://images.example/poster.jpg"}},
			Backdrops: []metadata.ImageOption{{URL: "https://images.example/backdrop.jpg"}},
		},
	}
	withTMDB := newLibraryHTTPTestServer(t, fake)
	libraryID := createLibrary(t, withTMDB.db, t.TempDir())
	seriesID := insertSeriesHTTPFixture(t, withTMDB, seriesHTTPFixture{
		libraryID: libraryID, folder: "Chicago fire (2012)", title: "Chicago fire",
		displayTitle: "Chicago fire", year: 2012,
	})
	first := insertSeriesEpisodeHTTP(t, withTMDB, libraryID, seriesID, "Piloto", "1x01",
		1, 1, "Piloto", "https://images.example/e1.jpg", "Sinopsis 1")
	second := insertSeriesEpisodeHTTP(t, withTMDB, libraryID, seriesID, "Segundo", "1x02",
		1, 2, "Segundo", "https://images.example/e2.jpg", "Sinopsis 2")
	insertSeriesEpisodeHTTP(t, withTMDB, libraryID, seriesID, "T2", "2x01",
		2, 1, "T2", "https://images.example/e3.jpg", "Sinopsis 3")

	status, body := libraryHTTPGet(t, seriesHTTPURL(withTMDB, seriesID, "/candidates"), withTMDB.token)
	if status != http.StatusOK {
		t.Fatalf("candidatos: %d %s", status, body)
	}
	if fake.lastTVQuery != "Chicago fire" {
		t.Fatalf("consulta por defecto = %q", fake.lastTVQuery)
	}

	status, body = libraryHTTPDo(t, http.MethodPut, seriesHTTPURL(withTMDB, seriesID, "/match"),
		withTMDB.token, map[string]any{"tmdb_id": 44006})
	if status != http.StatusOK {
		t.Fatalf("PUT match: %d %s", status, body)
	}
	var detail seriesDetailDTO
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("PUT match: %v", err)
	}
	if detail.MatchStatus == nil || *detail.MatchStatus != "manual" ||
		detail.TMDBID == nil || *detail.TMDBID != 44006 ||
		len(detail.Seasons) != 2 || detail.EpisodeCount != 3 {
		t.Fatalf("ficha tras PUT inesperada: %+v", detail)
	}
	episode := readEpisodeMetadata(t, withTMDB.db, fmt.Sprintf("/media/series/%d/1/1-1.mkv", seriesID))
	if !episode.title.Valid || episode.title.String != "Piloto" {
		t.Fatalf("episodio no rellenado: %+v", episode)
	}

	status, body = libraryHTTPDo(t, http.MethodPut, seriesHTTPURL(withTMDB, seriesID, "/match"),
		withTMDB.token, map[string]any{"tmdb_id": nil})
	if status != http.StatusOK {
		t.Fatalf("PUT null: %d %s", status, body)
	}
	detail = seriesDetailDTO{}
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("PUT null: %v", err)
	}
	if detail.MatchStatus == nil || *detail.MatchStatus != "none" || detail.TMDBID != nil ||
		detail.Overview != nil || len(detail.Seasons) != 2 ||
		detail.Seasons[0].Name != "Temporada 1" || detail.Seasons[0].Overview != nil {
		t.Fatalf("ficha tras PUT null inesperada: %+v", detail)
	}
	episode = readEpisodeMetadata(t, withTMDB.db, fmt.Sprintf("/media/series/%d/1/1-1.mkv", seriesID))
	if episode.title.Valid || episode.overview.Valid || episode.still.Valid {
		t.Fatalf("los episodios no se limpiaron: %+v", episode)
	}
	if status, body := libraryHTTPGet(t, seriesHTTPURL(withTMDB, seriesID, "/images"), withTMDB.token); status != http.StatusConflict {
		t.Fatalf("imágenes sin identificar: %d %s", status, body)
	}

	status, body = libraryHTTPDo(t, http.MethodDelete, seriesHTTPURL(withTMDB, seriesID, "/match"),
		withTMDB.token, nil)
	if status != http.StatusOK {
		t.Fatalf("DELETE match: %d %s", status, body)
	}
	detail = seriesDetailDTO{}
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("DELETE match: %v", err)
	}
	if detail.MatchStatus == nil || *detail.MatchStatus != "auto" ||
		detail.TMDBID == nil || *detail.TMDBID != 44006 {
		t.Fatalf("ficha tras DELETE inesperada: %+v", detail)
	}

	status, body = libraryHTTPGet(t, seriesHTTPURL(withTMDB, seriesID, "/images"), withTMDB.token)
	if status != http.StatusOK {
		t.Fatalf("imágenes: %d %s", status, body)
	}
	var images metadata.ImageOptions
	if err := json.Unmarshal(body, &images); err != nil || len(images.Posters) != 1 {
		t.Fatalf("imágenes inesperadas: %+v (%v)", images, err)
	}

	if status, body := libraryHTTPDo(t, http.MethodPut, seriesHTTPURL(withTMDB, seriesID, "/images/poster"),
		withTMDB.token, map[string]string{"url": "no-es-una-url"}); status != http.StatusBadRequest {
		t.Fatalf("URL inválida: %d %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodPut, seriesHTTPURL(withTMDB, seriesID, "/images/otra"),
		withTMDB.token, map[string]string{"url": "https://images.example/propio.jpg"}); status != http.StatusNotFound {
		t.Fatalf("tipo de imagen inválido: %d %s", status, body)
	}
	status, body = libraryHTTPDo(t, http.MethodPut, seriesHTTPURL(withTMDB, seriesID, "/images/poster"),
		withTMDB.token, map[string]string{"url": "https://images.example/propio.jpg"})
	if status != http.StatusOK {
		t.Fatalf("PUT imagen: %d %s", status, body)
	}
	row := readSeries(t, withTMDB.db, seriesID)
	if row.posterLocked != 1 ||
		!row.posterURL.Valid || row.posterURL.String != "https://images.example/propio.jpg" {
		t.Fatalf("póster propio inesperado: %+v", row)
	}
	status, body = libraryHTTPDo(t, http.MethodDelete, seriesHTTPURL(withTMDB, seriesID, "/images/poster"),
		withTMDB.token, nil)
	if status != http.StatusOK {
		t.Fatalf("DELETE imagen: %d %s", status, body)
	}
	row = readSeries(t, withTMDB.db, seriesID)
	if row.posterLocked != 0 || !row.posterURL.Valid ||
		row.posterURL.String != "https://images.example/serie.jpg" {
		t.Fatalf("póster restaurado inesperado: %+v", row)
	}

	// Visionado en bloque de la serie y de una temporada, con aislamiento.
	anaID := progressUserID(t, withTMDB.db, "ana")
	beaToken := nonAdminToken(t, withTMDB)
	beaID := progressUserID(t, withTMDB.db, "bea")
	setProgressRow(t, withTMDB.db, anaID, first, 10, 100, "2026-10-26T10:00:00Z")
	setProgressRow(t, withTMDB.db, beaID, second, 10, 100, "2026-10-26T11:00:00Z")

	status, body = libraryHTTPDo(t, http.MethodPut, seriesHTTPURL(withTMDB, seriesID, "/seasons/1/watched"),
		withTMDB.token, nil)
	if status != http.StatusNoContent {
		t.Fatalf("PUT temporada: %d %s", status, body)
	}
	if countSeriesWatchedForTest(t, withTMDB.db, anaID, seriesID, 1) != 2 ||
		countSeriesWatchedForTest(t, withTMDB.db, anaID, seriesID, 2) != 0 ||
		countSeriesWatchedForTest(t, withTMDB.db, beaID, seriesID, 1) != 0 {
		t.Fatal("el visionado de temporada no se aisló correctamente")
	}
	if _, _, _, ok := readProgress(t, withTMDB.db, anaID, first); ok {
		t.Fatal("el progreso de ana debía borrarse")
	}
	if _, _, _, ok := readProgress(t, withTMDB.db, beaID, second); !ok {
		t.Fatal("el progreso de bea no debe tocarse")
	}
	status, body = libraryHTTPDo(t, http.MethodDelete, seriesHTTPURL(withTMDB, seriesID, "/seasons/1/watched"),
		withTMDB.token, nil)
	if status != http.StatusNoContent || countSeriesWatchedForTest(t, withTMDB.db, anaID, seriesID, 1) != 0 {
		t.Fatalf("DELETE temporada: %d %s", status, body)
	}
	status, body = libraryHTTPDo(t, http.MethodPut, seriesHTTPURL(withTMDB, seriesID, "/watched"),
		withTMDB.token, nil)
	if status != http.StatusNoContent || countSeriesWatchedForTest(t, withTMDB.db, anaID, seriesID, 2) != 1 {
		t.Fatalf("PUT serie: %d %s", status, body)
	}
	status, body = libraryHTTPDo(t, http.MethodDelete, seriesHTTPURL(withTMDB, seriesID, "/watched"),
		withTMDB.token, nil)
	if status != http.StatusNoContent || countSeriesWatchedForTest(t, withTMDB.db, anaID, seriesID, 0) != 0 {
		t.Fatalf("DELETE serie: %d %s", status, body)
	}

	// Permisos.
	if status, body := libraryHTTPGet(t, seriesHTTPURL(withTMDB, seriesID, "/candidates"), beaToken); status != http.StatusForbidden {
		t.Fatalf("candidatos sin admin: %d %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodPut, seriesHTTPURL(withTMDB, seriesID, "/images/poster"),
		beaToken, map[string]string{"url": "https://images.example/x.jpg"}); status != http.StatusForbidden {
		t.Fatalf("imagen sin admin: %d %s", status, body)
	}

	// Sin TMDB y errores de TMDB.
	noTMDB := newLibraryHTTPTestServer(t)
	noLib := createLibrary(t, noTMDB.db, t.TempDir())
	noSeries := insertSeriesHTTPFixture(t, noTMDB, seriesHTTPFixture{
		libraryID: noLib, folder: "Sin TMDB", title: "Sin TMDB",
	})
	if status, body := libraryHTTPGet(t, seriesHTTPURL(noTMDB, noSeries, "/candidates"), noTMDB.token); status != http.StatusServiceUnavailable {
		t.Fatalf("candidatos sin TMDB: %d %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodPut, seriesHTTPURL(noTMDB, noSeries, "/watched"), noTMDB.token, nil); status != http.StatusNotFound {
		t.Fatalf("serie sin episodios: %d %s", status, body)
	}
	if status, body := libraryHTTPDo(t, http.MethodPut, seriesHTTPURL(noTMDB, noSeries, "/seasons/1/watched"), noTMDB.token, nil); status != http.StatusNotFound {
		t.Fatalf("temporada sin episodios: %d %s", status, body)
	}
	if status, body := libraryHTTPGet(t, seriesHTTPURL(noTMDB, noSeries, "/images"), noTMDB.token); status != http.StatusConflict {
		t.Fatalf("imágenes de serie sin identificar: %d %s", status, body)
	}
	if status, body := libraryHTTPGet(t, seriesHTTPURL(withTMDB, 999999, "/candidates"), withTMDB.token); status != http.StatusNotFound {
		t.Fatalf("candidatos de serie inexistente: %d %s", status, body)
	}
	fake.tvImagesErr = errors.New("red caída")
	if status, body := libraryHTTPGet(t, seriesHTTPURL(withTMDB, seriesID, "/images"), withTMDB.token); status != http.StatusBadGateway {
		t.Fatalf("imágenes con error: %d %s", status, body)
	}
	fake.tvImagesErr = nil
	fake.tvSearchErr = errors.New("red caída")
	if status, body := libraryHTTPGet(t, seriesHTTPURL(withTMDB, seriesID, "/candidates"), withTMDB.token); status != http.StatusBadGateway {
		t.Fatalf("candidatos con error: %d %s", status, body)
	}
	fake.tvSearchErr = nil
	fake.tvDetailsErr = errors.New("red caída")
	if status, body := libraryHTTPDo(t, http.MethodPut, seriesHTTPURL(withTMDB, seriesID, "/match"),
		withTMDB.token, map[string]any{"tmdb_id": 44006}); status != http.StatusBadGateway {
		t.Fatalf("match con error: %d %s", status, body)
	}
}

func countSeriesWatchedForTest(t *testing.T, db *sql.DB, userID, seriesID int64, season int) int {
	t.Helper()
	query := `SELECT count(*) FROM watch_state ws
		JOIN media_items mi ON mi.id = ws.media_item_id
		WHERE ws.user_id = ? AND mi.series_id = ?`
	args := []any{userID, seriesID}
	if season > 0 {
		query += " AND mi.season = ?"
		args = append(args, season)
	}
	var count int
	if err := db.QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatalf("contar vistos: %v", err)
	}
	return count
}

func TestItemsFiltroSeriesYDTOEpisodio(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	seriesID := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Serie", title: "Serie",
		displayTitle: "Serie mostrada", tmdbID: 5, metadataSource: "tmdb",
		matchStatus: "auto",
	})
	episodeID := insertSeriesEpisodeHTTP(t, ts, libraryID, seriesID, "Serie mostrada",
		"episodio-archivo", 1, 1, "Título episodio", "https://images.example/still.jpg",
		"Sinopsis episodio")
	movieID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/pelicula.mkv", title: "Película",
		displayTitle: "Película mostrada", kind: KindMovie,
	})

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items?series_id="+strconv.FormatInt(seriesID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("items series_id: %d %s", status, body)
	}
	var items itemsResponse
	if err := json.Unmarshal(body, &items); err != nil {
		t.Fatalf("items series_id: %v", err)
	}
	if items.Total != 1 || len(items.Items) != 1 || items.Items[0].ID != episodeID {
		t.Fatalf("items series_id inesperado: %+v", items)
	}
	item := items.Items[0]
	if item.SeriesID == nil || *item.SeriesID != seriesID ||
		item.SeriesTitle == nil || *item.SeriesTitle != "Serie mostrada" ||
		item.EpisodeTitle == nil || *item.EpisodeTitle != "Título episodio" ||
		item.StillURL == nil || *item.StillURL != "https://images.example/still.jpg" ||
		item.Title != "Título episodio" || item.FileTitle != "episodio-archivo" {
		t.Fatalf("DTO de episodio inesperado: %+v", item)
	}
	_ = movieID

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items?series_id=0", ts.token)
	if status != http.StatusBadRequest {
		t.Fatalf("series_id inválido: %d %s", status, body)
	}
}

func TestHomeNextUpYContadorSeries(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	anaID := progressUserID(t, ts.db, "ana")

	type homeSeries struct {
		id     int64
		first  int64
		second int64
	}
	makeSeries := func(folder string) homeSeries {
		seriesID := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
			libraryID: libraryID, folder: folder, title: folder,
			displayTitle: folder, tmdbID: 1, metadataSource: "tmdb", matchStatus: "auto",
		})
		first := insertSeriesEpisodeHTTP(t, ts, libraryID, seriesID, folder, "s1e1", 1, 1,
			"Primero", "https://images.example/1.jpg", "Sinopsis 1")
		second := insertSeriesEpisodeHTTP(t, ts, libraryID, seriesID, folder, "s1e2", 1, 2,
			"Segundo", "https://images.example/2.jpg", "Sinopsis 2")
		return homeSeries{id: seriesID, first: first, second: second}
	}
	a := makeSeries("Serie A")
	b := makeSeries("Serie B")
	c := makeSeries("Serie C")
	d := makeSeries("Serie D")

	setWatchState(t, ts.db, anaID, a.first)
	setWatchState(t, ts.db, anaID, b.first)
	setWatchState(t, ts.db, anaID, d.first)
	setWatchState(t, ts.db, anaID, d.second)
	if _, err := ts.db.Exec(`UPDATE watch_state SET watched_at = '2026-10-26T10:00:00Z'
		WHERE user_id = ? AND media_item_id = ?`, anaID, a.first); err != nil {
		t.Fatalf("fecha de A: %v", err)
	}
	if _, err := ts.db.Exec(`UPDATE watch_state SET watched_at = '2026-10-26T11:00:00Z'
		WHERE user_id = ? AND media_item_id = ?`, anaID, b.first); err != nil {
		t.Fatalf("fecha de B: %v", err)
	}
	setProgressRow(t, ts.db, anaID, c.first, 30, 100, "2026-10-26T12:00:00Z")

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/home", ts.token)
	if status != http.StatusOK {
		t.Fatalf("home: %d %s", status, body)
	}
	var home homeResponse
	if err := json.Unmarshal(body, &home); err != nil {
		t.Fatalf("home: %v", err)
	}
	if home.Counts.Series != 4 || home.Counts.Episodes != 8 {
		t.Fatalf("contadores inesperados: %+v", home.Counts)
	}
	if len(home.NextUp) != 2 {
		t.Fatalf("next_up = %d, se esperaban 2: %+v", len(home.NextUp), home.NextUp)
	}
	if home.NextUp[0].ID != b.second || home.NextUp[1].ID != a.second {
		t.Fatalf("orden de next_up inesperado: %+v", home.NextUp)
	}
	if home.NextUp[0].SeriesTitle == nil || *home.NextUp[0].SeriesTitle != "Serie B" ||
		home.NextUp[0].EpisodeTitle == nil || *home.NextUp[0].EpisodeTitle != "Segundo" {
		t.Fatalf("DTO de next_up inesperado: %+v", home.NextUp[0])
	}
	for _, item := range home.NextUp {
		if item.ID == c.first {
			t.Fatal("next_up no debe repetir una serie en continue")
		}
	}
}
