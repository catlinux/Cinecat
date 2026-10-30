package library

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/catlinux/cinecat/internal/metadata"
)

type fakeTMDB struct {
	searchCalls   int
	detailsCalls  int
	lastQuery     string
	lastYear      int
	lastDetailsID int
	searchErr     error
	detailsErr    error
	imagesErr     error
	details       *metadata.Details
	images        *metadata.ImageOptions
	imagesCalls   int
	lastImagesID  int

	tvSearchCalls   int
	tvSearchErr     error
	tvSearchResults map[string][]metadata.Candidate
	lastTVQuery     string
	lastTVYear      int
	tvDetailsCalls  int
	tvDetailsErr    error
	tvDetails       *metadata.Details
	lastTVDetailsID int
	seasonCalls     int
	seasonErr       error
	seasons         map[int]*metadata.Season
	lastSeason      int
	lastSeasonTV    int

	tvImagesCalls  int
	tvImagesErr    error
	tvImages       *metadata.ImageOptions
	lastTVImagesID int

	searchCollectionsCalls int
	searchCollectionsErr   error
	searchCollections      []metadata.CollectionCandidate
	lastCollectionsQuery   string
	collectionDetailsCalls int
	collectionDetailsErr   error
	collectionDetails      *metadata.CollectionInfo
	lastCollectionDetails  int
}

func (f *fakeTMDB) SearchMovies(ctx context.Context, query string, year int) ([]metadata.Candidate, error) {
	f.searchCalls++
	f.lastQuery = query
	f.lastYear = year
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	return []metadata.Candidate{{
		TMDBID:        42,
		Title:         query,
		OriginalTitle: query,
		Year:          2019,
		Popularity:    10,
		VoteCount:     1000,
	}}, nil
}

func (f *fakeTMDB) MovieCollection(ctx context.Context, movieID int) ([]metadata.Candidate, error) {
	return nil, nil
}

func (f *fakeTMDB) SearchCollections(ctx context.Context, query string) ([]metadata.CollectionCandidate, error) {
	f.searchCollectionsCalls++
	f.lastCollectionsQuery = query
	if f.searchCollectionsErr != nil {
		return nil, f.searchCollectionsErr
	}
	if f.searchCollections == nil {
		return nil, metadata.ErrNotFound
	}
	return f.searchCollections, nil
}

func (f *fakeTMDB) CollectionDetails(ctx context.Context, id int) (*metadata.CollectionInfo, error) {
	f.collectionDetailsCalls++
	f.lastCollectionDetails = id
	if f.collectionDetailsErr != nil {
		return nil, f.collectionDetailsErr
	}
	if f.collectionDetails == nil {
		return &metadata.CollectionInfo{TMDBID: id}, nil
	}
	return f.collectionDetails, nil
}

func (f *fakeTMDB) MovieDetails(ctx context.Context, id int) (*metadata.Details, error) {
	f.detailsCalls++
	f.lastDetailsID = id
	if f.detailsErr != nil {
		return nil, f.detailsErr
	}
	if f.details == nil {
		return &metadata.Details{TMDBID: id}, nil
	}
	return f.details, nil
}

func (f *fakeTMDB) MovieImages(ctx context.Context, tmdbID int) (*metadata.ImageOptions, error) {
	f.imagesCalls++
	f.lastImagesID = tmdbID
	if f.imagesErr != nil {
		return nil, f.imagesErr
	}
	if f.images == nil {
		return &metadata.ImageOptions{
			Posters:   []metadata.ImageOption{},
			Backdrops: []metadata.ImageOption{},
		}, nil
	}
	return f.images, nil
}

func (f *fakeTMDB) TVImages(ctx context.Context, tvID int) (*metadata.ImageOptions, error) {
	f.tvImagesCalls++
	f.lastTVImagesID = tvID
	if f.tvImagesErr != nil {
		return nil, f.tvImagesErr
	}
	if f.tvImages == nil {
		return &metadata.ImageOptions{
			Posters:   []metadata.ImageOption{},
			Backdrops: []metadata.ImageOption{},
		}, nil
	}
	return f.tvImages, nil
}

func (f *fakeTMDB) SearchTV(ctx context.Context, query string, year int) ([]metadata.Candidate, error) {
	f.tvSearchCalls++
	f.lastTVQuery = query
	f.lastTVYear = year
	if f.tvSearchErr != nil {
		return nil, f.tvSearchErr
	}
	if results, ok := f.tvSearchResults[query]; ok {
		return results, nil
	}
	return nil, metadata.ErrNotFound
}

func (f *fakeTMDB) TVDetails(ctx context.Context, id int) (*metadata.Details, error) {
	f.tvDetailsCalls++
	f.lastTVDetailsID = id
	if f.tvDetailsErr != nil {
		return nil, f.tvDetailsErr
	}
	if f.tvDetails == nil {
		return &metadata.Details{TMDBID: id}, nil
	}
	return f.tvDetails, nil
}

func (f *fakeTMDB) SeasonDetails(ctx context.Context, tvID, season int) (*metadata.Season, error) {
	f.seasonCalls++
	f.lastSeason = season
	f.lastSeasonTV = tvID
	if f.seasonErr != nil {
		return nil, f.seasonErr
	}
	if detail, ok := f.seasons[season]; ok {
		return detail, nil
	}
	return nil, metadata.ErrNotFound
}

type tmdbMetadataRow struct {
	tmdbID           sql.NullInt64
	originalTitle    sql.NullString
	displayTitle     sql.NullString
	imdbID           sql.NullString
	tagline          sql.NullString
	overview         sql.NullString
	overviewLanguage sql.NullString
	genres           sql.NullString
	runtime          sql.NullInt64
	rating           sql.NullFloat64
	directors        sql.NullString
	castNames        sql.NullString
	posterURL        sql.NullString
	posterLocked     int
	backdropURL      sql.NullString
	backdropLocked   int
	source           sql.NullString
	matchStatus      sql.NullString
	matchScore       sql.NullFloat64
}

func readTMDBMetadata(t *testing.T, db *sql.DB, path string) tmdbMetadataRow {
	t.Helper()
	var row tmdbMetadataRow
	if err := db.QueryRow(`
		SELECT tmdb_id, original_title, display_title, imdb_id, tagline,
			overview, overview_language, genres, runtime_minutes, rating,
			directors, cast_names, poster_url, poster_locked, backdrop_url,
			backdrop_locked, metadata_source, match_status, match_score
		FROM media_items WHERE path = ?`, path,
	).Scan(
		&row.tmdbID, &row.originalTitle, &row.displayTitle, &row.imdbID,
		&row.tagline, &row.overview, &row.overviewLanguage, &row.genres,
		&row.runtime, &row.rating, &row.directors, &row.castNames,
		&row.posterURL, &row.posterLocked, &row.backdropURL, &row.backdropLocked,
		&row.source, &row.matchStatus, &row.matchScore,
	); err != nil {
		t.Fatalf("leer metadatos de %q: %v", path, err)
	}
	return row
}

func readSortKeys(t *testing.T, db *sql.DB, path string) (string, string) {
	t.Helper()
	var title, letter sql.NullString
	if err := db.QueryRow(
		`SELECT sort_title, sort_letter FROM media_items WHERE path = ?`, path,
	).Scan(&title, &letter); err != nil {
		t.Fatalf("leer claves de orden de %q: %v", path, err)
	}
	return title.String, letter.String
}

func newMetadataTestClient(t *testing.T, baseURL string) *metadata.OMDbClient {
	t.Helper()
	client, err := metadata.NewOMDbClient(metadata.OMDbOptions{
		APIKey:   "clave-de-prueba",
		BaseURL:  baseURL,
		CacheDir: filepath.Join(t.TempDir(), "cache"),
	})
	if err != nil {
		t.Fatalf("NewOMDbClient: %v", err)
	}
	return client
}

func readMetadata(t *testing.T, db *sql.DB, path string) (sql.NullString, sql.NullString) {
	t.Helper()
	var overview, poster sql.NullString
	if err := db.QueryRow(
		`SELECT overview, poster_url FROM media_items WHERE path = ?`, path,
	).Scan(&overview, &poster); err != nil {
		t.Fatalf("leer metadatos de %q: %v", path, err)
	}
	return overview, poster
}

func TestScannerConOMDbRellenaYNoRepite(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	root := t.TempDir()
	file := filepath.Join(root, "Pelicula (2019).mkv")
	writeFile(t, file)
	createLibrary(t, db, root)

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"Title": "Pelicula",
			"Year": "2019",
			"Plot": "Una sinopsis de prueba.",
			"Poster": "https://images.example/pelicula.jpg",
			"Response": "True"
		}`))
	}))
	t.Cleanup(server.Close)

	scanner := New(db).WithOMDb(newMetadataTestClient(t, server.URL))
	res, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Files != 1 || res.Upserted != 1 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("avisos inesperados: %v", res.Warnings)
	}

	overview, poster := readMetadata(t, db, file)
	if !overview.Valid || overview.String != "Una sinopsis de prueba." {
		t.Fatalf("overview = %+v", overview)
	}
	if !poster.Valid || poster.String != "https://images.example/pelicula.jpg" {
		t.Fatalf("poster_url = %+v", poster)
	}
	if calls.Load() != 1 {
		t.Fatalf("peticiones = %d, se esperaba 1", calls.Load())
	}

	// Un segundo escaneo no vuelve a consultar OMDb.
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("segundo Scan: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("el segundo escaneo hizo %d peticiones a OMDb", calls.Load())
	}
}

func TestScannerConOMDbNoEncontrado(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	root := t.TempDir()
	file := filepath.Join(root, "Desconocida (1999).mkv")
	writeFile(t, file)
	createLibrary(t, db, root)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":"False","Error":"Movie not found!"}`))
	}))
	t.Cleanup(server.Close)

	res, err := New(db).WithOMDb(newMetadataTestClient(t, server.URL)).Scan(ctx)
	if err != nil {
		t.Fatalf("el escaneo no debe fallar: %v", err)
	}
	if res.Files != 1 {
		t.Fatalf("archivos = %d", res.Files)
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("avisos = %v, se esperaba uno", res.Warnings)
	}

	overview, poster := readMetadata(t, db, file)
	if overview.Valid || poster.Valid {
		t.Fatalf("no debe haber metadatos: %+v, %+v", overview, poster)
	}
}

func TestScannerConOMDbIgnoraSeriesYEpisodios(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Serie", "Season 1", "Serie - 1x01.mkv"))
	createLibrary(t, db, root)

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":"True","Title":"Serie","Plot":"x"}`))
	}))
	t.Cleanup(server.Close)

	res, err := New(db).WithOMDb(newMetadataTestClient(t, server.URL)).Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Files != 1 {
		t.Fatalf("archivos = %d", res.Files)
	}
	if calls.Load() != 0 {
		t.Fatalf("no se debe consultar OMDb para episodios: %d peticiones", calls.Load())
	}
}

func TestScannerConTMDBIdentificaYPersisteFicha(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	root := t.TempDir()
	file := filepath.Join(root, "Pelicula (2019).mkv")
	writeFile(t, file)
	createLibrary(t, db, root)

	fake := &fakeTMDB{details: &metadata.Details{
		TMDBID:           42,
		IMDbID:           "tt0123456",
		Title:            "Ficha de TMDB",
		OriginalTitle:    "Original title",
		Year:             2019,
		Tagline:          "Un lema",
		Overview:         "Una sinopsis en castellano.",
		OverviewLanguage: "es-ES",
		Genres:           []string{"Drama", "Acción"},
		RuntimeMinutes:   121,
		Rating:           8.4,
		Directors:        []string{"Directora"},
		Cast:             []string{"Actor uno", "Actor dos"},
		PosterURL:        "https://images.example/poster.jpg",
		BackdropURL:      "https://images.example/backdrop.jpg",
	}}
	scanner := New(db).WithTMDB(fake)

	res, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Files != 1 || res.Identified != 1 || res.Unidentified != 0 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("avisos inesperados: %v", res.Warnings)
	}

	row := readTMDBMetadata(t, db, file)
	if !row.tmdbID.Valid || row.tmdbID.Int64 != 42 {
		t.Fatalf("tmdb_id = %+v", row.tmdbID)
	}
	if !row.originalTitle.Valid || row.originalTitle.String != "Original title" {
		t.Fatalf("original_title = %+v", row.originalTitle)
	}
	if !row.displayTitle.Valid || row.displayTitle.String != "Ficha de TMDB" {
		t.Fatalf("display_title = %+v", row.displayTitle)
	}
	if sortTitle, letter := readSortKeys(t, db, file); sortTitle != "Ficha de TMDB" || letter != "F" {
		t.Fatalf("claves de orden = %q, %q", sortTitle, letter)
	}
	if !row.imdbID.Valid || row.imdbID.String != "tt0123456" {
		t.Fatalf("imdb_id = %+v", row.imdbID)
	}
	if !row.tagline.Valid || row.tagline.String != "Un lema" {
		t.Fatalf("tagline = %+v", row.tagline)
	}
	if !row.overview.Valid || row.overview.String != "Una sinopsis en castellano." {
		t.Fatalf("overview = %+v", row.overview)
	}
	if !row.overviewLanguage.Valid || row.overviewLanguage.String != "es-ES" {
		t.Fatalf("overview_language = %+v", row.overviewLanguage)
	}
	if !row.genres.Valid || row.genres.String != `["Drama","Acción"]` {
		t.Fatalf("genres = %+v", row.genres)
	}
	if !row.runtime.Valid || row.runtime.Int64 != 121 {
		t.Fatalf("runtime_minutes = %+v", row.runtime)
	}
	if !row.rating.Valid || row.rating.Float64 != 8.4 {
		t.Fatalf("rating = %+v", row.rating)
	}
	if !row.directors.Valid || row.directors.String != `["Directora"]` {
		t.Fatalf("directors = %+v", row.directors)
	}
	if !row.castNames.Valid || row.castNames.String != `["Actor uno","Actor dos"]` {
		t.Fatalf("cast_names = %+v", row.castNames)
	}
	if !row.posterURL.Valid || row.posterURL.String != "https://images.example/poster.jpg" {
		t.Fatalf("poster_url = %+v", row.posterURL)
	}
	if !row.backdropURL.Valid || row.backdropURL.String != "https://images.example/backdrop.jpg" {
		t.Fatalf("backdrop_url = %+v", row.backdropURL)
	}
	if !row.source.Valid || row.source.String != "tmdb" {
		t.Fatalf("metadata_source = %+v", row.source)
	}
	if !row.matchStatus.Valid || row.matchStatus.String != "auto" {
		t.Fatalf("match_status = %+v", row.matchStatus)
	}
	if !row.matchScore.Valid || row.matchScore.Float64 < MatchThreshold {
		t.Fatalf("match_score = %+v", row.matchScore)
	}

	// El título del archivo no se cambia por el de la ficha.
	media := readMediaRow(t, db, file)
	if media.title != "Pelicula" {
		t.Fatalf("title = %q", media.title)
	}

	// Un segundo escaneo no vuelve a consultar TMDB para una fila auto+tmdb.
	searches := fake.searchCalls
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("segundo Scan: %v", err)
	}
	if fake.searchCalls != searches || fake.detailsCalls != 1 {
		t.Fatalf("segundo escaneo: búsquedas %d (antes %d), fichas %d",
			fake.searchCalls, searches, fake.detailsCalls)
	}
}

func TestScannerConTMDBLimpiaMetadatosSinCoincidencia(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	root := t.TempDir()
	file := filepath.Join(root, "Desconocida (1999).mkv")
	writeFile(t, file)
	libraryID := createLibrary(t, db, root)

	if _, err := db.Exec(`
		INSERT INTO media_items (
			library_id, path, title, year, kind, tmdb_id, overview, poster_url,
			poster_locked, original_title, display_title, imdb_id, tagline,
			genres, runtime_minutes, rating, directors, cast_names, backdrop_url,
			backdrop_locked, overview_language, metadata_source, match_score
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?)`,
		libraryID, file, "Desconocida", 1999, KindMovie, 7,
		"Sinopsis de OMDb", "https://images.example/omdb.jpg",
		"Old original", "Título antiguo", "tt0000007", "Old tagline", `["Old genre"]`, 100, 7.1,
		`["Old director"]`, `["Old actor"]`, "https://images.example/old-backdrop.jpg",
		"en-US", "omdb", 0.9,
	); err != nil {
		t.Fatalf("insertar fila previa: %v", err)
	}

	fake := &fakeTMDB{searchErr: metadata.ErrNotFound}
	res, err := New(db).WithTMDB(fake).Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Identified != 0 || res.Unidentified != 1 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("avisos inesperados: %v", res.Warnings)
	}
	if fake.detailsCalls != 0 {
		t.Fatalf("no se debe pedir la ficha sin coincidencia: %d", fake.detailsCalls)
	}

	row := readTMDBMetadata(t, db, file)
	if row.tmdbID.Valid || row.originalTitle.Valid || row.displayTitle.Valid ||
		row.imdbID.Valid || row.tagline.Valid || row.overview.Valid ||
		row.overviewLanguage.Valid || row.genres.Valid || row.runtime.Valid ||
		row.rating.Valid || row.directors.Valid || row.castNames.Valid ||
		row.posterURL.Valid || row.posterLocked != 0 ||
		row.backdropURL.Valid || row.backdropLocked != 0 || row.source.Valid ||
		row.matchStatus.Valid || row.matchScore.Valid {
		t.Fatalf("los metadatos debían quedar vacíos: %+v", row)
	}
	if sortTitle, letter := readSortKeys(t, db, file); sortTitle != "Desconocida" || letter != "D" {
		t.Fatalf("claves de orden = %q, %q", sortTitle, letter)
	}
}

func TestScannerConTMDBNoTocaManualNiNone(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	root := t.TempDir()
	manual := filepath.Join(root, "Manual (2001).mkv")
	none := filepath.Join(root, "None (2002).mkv")
	writeFile(t, manual)
	writeFile(t, none)
	libraryID := createLibrary(t, db, root)

	for _, item := range []struct {
		path   string
		title  string
		year   int
		status string
		source string
	}{
		{manual, "Manual", 2001, "manual", "tmdb"},
		{none, "None", 2002, "none", "omdb"},
	} {
		if _, err := db.Exec(`
			INSERT INTO media_items (
				library_id, path, title, year, kind, tmdb_id, overview,
				poster_url, display_title, metadata_source, match_status, match_score,
				collection_checked, credits
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?)`,
			libraryID, item.path, item.title, item.year, KindMovie, 99,
			"Ficha fijada", "https://images.example/fijada.jpg", "Ficha fijada",
			item.source, item.status, 0.8, `{"cast":[],"crew":[]}`,
		); err != nil {
			t.Fatalf("insertar %q: %v", item.path, err)
		}
	}

	fake := &fakeTMDB{details: &metadata.Details{TMDBID: 99, Overview: "Otra"}}
	res, err := New(db).WithTMDB(fake).Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Identified != 0 || res.Unidentified != 0 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("avisos inesperados: %v", res.Warnings)
	}
	if fake.searchCalls != 0 || fake.detailsCalls != 0 {
		t.Fatalf("no se debe consultar TMDB: búsquedas %d, fichas %d",
			fake.searchCalls, fake.detailsCalls)
	}

	for _, path := range []string{manual, none} {
		row := readTMDBMetadata(t, db, path)
		if !row.tmdbID.Valid || row.tmdbID.Int64 != 99 ||
			!row.overview.Valid || row.overview.String != "Ficha fijada" ||
			!row.displayTitle.Valid || row.displayTitle.String != "Ficha fijada" ||
			!row.matchScore.Valid || row.matchScore.Float64 != 0.8 {
			t.Fatalf("la fila %q cambió: %+v", path, row)
		}
	}
}

func TestScannerConTMDBRellenaDisplayTitlePendiente(t *testing.T) {
	for _, status := range []string{"auto", "manual"} {
		t.Run(status, func(t *testing.T) {
			ctx := context.Background()
			db := openTestDB(t)

			root := t.TempDir()
			file := filepath.Join(root, "Pelicula (1999).mkv")
			writeFile(t, file)
			libraryID := createLibrary(t, db, root)

			if _, err := db.Exec(`
				INSERT INTO media_items (
					library_id, path, title, year, kind, tmdb_id, overview,
					poster_url, poster_locked, backdrop_url, backdrop_locked,
					metadata_source, match_status, match_score
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?, 1, 'tmdb', ?, ?)`,
				libraryID, file, "Pelicula", 1999, KindMovie, 77,
				"Sinopsis antigua", "https://images.example/poster-fijado.jpg",
				"https://images.example/fondo-fijado.jpg", status, 0.87,
			); err != nil {
				t.Fatalf("insertar fila previa: %v", err)
			}

			fake := &fakeTMDB{details: &metadata.Details{
				TMDBID:      77,
				PosterURL:   "https://images.example/poster-tmdb.jpg",
				BackdropURL: "https://images.example/fondo-tmdb.jpg",
				Title:       "Título de TMDB",
				Year:        1999,
				Overview:    "Sinopsis nueva",
				Credits: metadata.Credits{
					Cast: []metadata.CastCredit{{
						Name:      "Actriz",
						Character: "Heroína",
						PhotoURL:  "https://images.example/w185/actriz.jpg",
					}},
				},
				Collection: &metadata.CollectionRef{
					TMDBID: 900,
					Name:   "La saga",
				},
			}}
			res, err := New(db).WithTMDB(fake).Scan(ctx)
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}
			if res.Identified != 0 || res.Unidentified != 0 {
				t.Fatalf("el relleno no debe contar como identificación: %+v", res)
			}
			if fake.searchCalls != 0 || fake.detailsCalls != 1 {
				t.Fatalf("búsquedas %d, fichas %d", fake.searchCalls, fake.detailsCalls)
			}

			row := readTMDBMetadata(t, db, file)
			if !row.displayTitle.Valid || row.displayTitle.String != "Título de TMDB" {
				t.Fatalf("display_title = %+v", row.displayTitle)
			}
			if !row.matchStatus.Valid || row.matchStatus.String != status {
				t.Fatalf("match_status = %+v", row.matchStatus)
			}
			if !row.matchScore.Valid || row.matchScore.Float64 != 0.87 {
				t.Fatalf("match_score = %+v", row.matchScore)
			}
			if !row.overview.Valid || row.overview.String != "Sinopsis nueva" {
				t.Fatalf("overview = %+v", row.overview)
			}
			if !row.posterURL.Valid || row.posterURL.String != "https://images.example/poster-fijado.jpg" ||
				row.posterLocked != 1 {
				t.Fatalf("el escaneo pisó el póster fijado: %+v", row)
			}
			if !row.backdropURL.Valid || row.backdropURL.String != "https://images.example/fondo-fijado.jpg" ||
				row.backdropLocked != 1 {
				t.Fatalf("el escaneo pisó el fondo fijado: %+v", row)
			}
			if sortTitle, letter := readSortKeys(t, db, file); sortTitle != "Título de TMDB" || letter != "T" {
				t.Fatalf("claves de orden = %q, %q", sortTitle, letter)
			}
			var rawCredits sql.NullString
			if err := db.QueryRow(`SELECT credits FROM media_items WHERE id = (
				SELECT id FROM media_items WHERE path = ?)`, file).Scan(&rawCredits); err != nil {
				t.Fatalf("leer credits: %v", err)
			}
			if !rawCredits.Valid {
				t.Fatal("el relleno debe guardar credits")
			}
			decodedCredits := decodeCredits(rawCredits)
			if len(decodedCredits.Cast) != 1 ||
				decodedCredits.Cast[0].Character != "Heroína" ||
				decodedCredits.Cast[0].PhotoURL != "https://images.example/w185/actriz.jpg" {
				t.Fatalf("créditos rellenados inesperados: %+v", decodedCredits)
			}
			var collectionChecked int
			if err := db.QueryRow(
				`SELECT collection_checked FROM media_items WHERE path = ?`, file,
			).Scan(&collectionChecked); err != nil {
				t.Fatalf("leer collection_checked: %v", err)
			}
			if collectionChecked != 1 {
				t.Fatalf("collection_checked = %d, se esperaba 1", collectionChecked)
			}
			var collectionName string
			if err := db.QueryRow(`
				SELECT c.name FROM collections c
				  JOIN collection_items ci ON ci.collection_id = c.id
				  JOIN media_items mi ON mi.id = ci.media_item_id
				 WHERE mi.path = ?`, file).Scan(&collectionName); err != nil {
				t.Fatalf("leer colección rellenada: %v", err)
			}
			if collectionName != "La saga" {
				t.Fatalf("colección = %q", collectionName)
			}

			fake.searchCalls = 0
			fake.detailsCalls = 0
			if _, err := New(db).WithTMDB(fake).Scan(ctx); err != nil {
				t.Fatalf("segundo Scan: %v", err)
			}
			if fake.searchCalls != 0 || fake.detailsCalls != 0 {
				t.Fatalf("el segundo escaneo no debe pedir TMDB: %d, %d",
					fake.searchCalls, fake.detailsCalls)
			}
		})
	}
}

func TestScannerConTMDBRellenaColeccionPendiente(t *testing.T) {
	for _, status := range []string{"auto", "manual"} {
		t.Run(status, func(t *testing.T) {
			ctx := context.Background()
			db := openTestDB(t)

			root := t.TempDir()
			file := filepath.Join(root, "Pelicula (1999).mkv")
			writeFile(t, file)
			libraryID := createLibrary(t, db, root)
			itemID := insertItemFixture(t, db, itemFixture{
				libraryID: libraryID, path: file, title: "Pelicula",
				displayTitle: "Pelicula", year: 1999, kind: KindMovie,
				tmdbID: 77, metadataSource: "tmdb", matchStatus: status, matchScore: 0.87,
			})

			fake := &fakeTMDB{details: &metadata.Details{
				TMDBID: 77,
				Title:  "Pelicula",
				Year:   1999,
				Collection: &metadata.CollectionRef{
					TMDBID: 900,
					Name:   "La saga",
				},
			}}
			scanner := New(db).WithTMDB(fake)
			res, err := scanner.Scan(ctx)
			if err != nil {
				t.Fatalf("Scan: %v", err)
			}
			if res.Identified != 0 || res.Unidentified != 0 {
				t.Fatalf("el relleno no debe contar como identificación: %+v", res)
			}
			if fake.searchCalls != 0 || fake.detailsCalls != 1 {
				t.Fatalf("búsquedas %d, fichas %d", fake.searchCalls, fake.detailsCalls)
			}

			collection := readCollection(t, db, 900)
			if collection.Name != "La saga" || collection.Source != "tmdb" {
				t.Fatalf("colección inesperada: %+v", collection)
			}
			if _, _, ok := readMembership(t, db, collection.ID, itemID); !ok {
				t.Fatal("la película no entró en su colección")
			}
			if checked := collectionChecked(t, db, itemID); checked != 1 {
				t.Fatalf("collection_checked = %d, se esperaba 1", checked)
			}
			row := readTMDBMetadata(t, db, file)
			if !row.matchStatus.Valid || row.matchStatus.String != status ||
				!row.matchScore.Valid || row.matchScore.Float64 != 0.87 {
				t.Fatalf("el estado cambió: %+v %+v", row.matchStatus, row.matchScore)
			}

			if _, err := scanner.Scan(ctx); err != nil {
				t.Fatalf("segundo Scan: %v", err)
			}
			if fake.detailsCalls != 1 || fake.searchCalls != 0 {
				t.Fatalf("no debe repetirse la ficha: búsquedas %d, fichas %d",
					fake.searchCalls, fake.detailsCalls)
			}
		})
	}
}

func TestScannerRellenaSinopsisDeColeccionUnaSolaVez(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	file := filepath.Join(root, "Pelicula (1999).mkv")
	writeFile(t, file)
	libraryID := createLibrary(t, db, root)
	itemID := insertItemFixture(t, db, itemFixture{
		libraryID: libraryID, path: file, title: "Pelicula",
		displayTitle: "Pelicula", year: 1999, kind: KindMovie,
		tmdbID: 77, metadataSource: "tmdb", matchStatus: "auto", matchScore: 0.87,
	})

	fake := &fakeTMDB{
		details: &metadata.Details{
			TMDBID: 77,
			Title:  "Pelicula",
			Year:   1999,
			Collection: &metadata.CollectionRef{
				TMDBID: 900,
				Name:   "La saga",
			},
		},
		collectionDetails: &metadata.CollectionInfo{
			TMDBID:   900,
			Name:     "La saga",
			Overview: "Sinopsis de la saga",
		},
	}
	scanner := New(db).WithTMDB(fake)
	res, err := scanner.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("avisos inesperados: %v", res.Warnings)
	}
	if fake.collectionDetailsCalls != 1 || fake.lastCollectionDetails != 900 {
		t.Fatalf("fichas de colección = %d, última = %d",
			fake.collectionDetailsCalls, fake.lastCollectionDetails)
	}
	overview := collectionOverview(t, db, 900)
	if !overview.Valid || overview.String != "Sinopsis de la saga" {
		t.Fatalf("sinopsis = %+v", overview)
	}

	fake.collectionDetailsCalls = 0
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("segundo Scan: %v", err)
	}
	if fake.collectionDetailsCalls != 0 {
		t.Fatalf("no debe repetirse la sinopsis: %d", fake.collectionDetailsCalls)
	}

	if _, err := db.Exec(`UPDATE collections SET edited = 1, overview = NULL
		WHERE tmdb_collection_id = 900`); err != nil {
		t.Fatalf("marcar colección editada: %v", err)
	}
	if _, err := db.Exec(`UPDATE media_items SET credits = NULL WHERE id = ?`, itemID); err != nil {
		t.Fatalf("forzar ficha: %v", err)
	}
	fake.collectionDetailsCalls = 0
	if _, err := scanner.Scan(ctx); err != nil {
		t.Fatalf("Scan de colección editada: %v", err)
	}
	if fake.collectionDetailsCalls != 0 {
		t.Fatalf("una colección editada no debe pedir sinopsis: %d", fake.collectionDetailsCalls)
	}
}

func TestScannerAvisaSiFallaLaSinopsisDeColeccion(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	file := filepath.Join(root, "Pelicula (1999).mkv")
	writeFile(t, file)
	libraryID := createLibrary(t, db, root)
	insertItemFixture(t, db, itemFixture{
		libraryID: libraryID, path: file, title: "Pelicula",
		displayTitle: "Pelicula", year: 1999, kind: KindMovie,
		tmdbID: 77, metadataSource: "tmdb", matchStatus: "auto", matchScore: 0.87,
	})

	fake := &fakeTMDB{
		details: &metadata.Details{
			TMDBID: 77,
			Title:  "Pelicula",
			Year:   1999,
			Collection: &metadata.CollectionRef{
				TMDBID: 900,
				Name:   "La saga",
			},
		},
		collectionDetailsErr: errors.New("TMDB no responde"),
	}
	res, err := New(db).WithTMDB(fake).Scan(ctx)
	if err != nil {
		t.Fatalf("el fallo de TMDB no debe abortar: %v", err)
	}
	if fake.collectionDetailsCalls != 1 {
		t.Fatalf("fichas de colección = %d", fake.collectionDetailsCalls)
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("avisos = %v, se esperaba uno", res.Warnings)
	}
	if overview := collectionOverview(t, db, 900); overview.Valid {
		t.Fatalf("la sinopsis no debe guardarse: %+v", overview)
	}
}

func collectionOverview(t *testing.T, db *sql.DB, tmdbID int) sql.NullString {
	t.Helper()
	var overview sql.NullString
	if err := db.QueryRow(`SELECT overview FROM collections WHERE tmdb_collection_id = ?`,
		tmdbID).Scan(&overview); err != nil {
		t.Fatalf("leer sinopsis de la colección %d: %v", tmdbID, err)
	}
	return overview
}

func TestApplyMovieDetailsGuardaYClearCredits(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	libraryID := createLibrary(t, db, t.TempDir())
	itemID := insertItemFixture(t, db, itemFixture{
		libraryID: libraryID, path: filepath.Join(t.TempDir(), "creditos.mkv"),
		title: "Créditos", kind: KindMovie,
	})

	details := &metadata.Details{
		TMDBID: 1,
		Title:  "Créditos",
		Credits: metadata.Credits{
			Cast: []metadata.CastCredit{{Name: "Actriz", Character: "Heroína", PhotoURL: "https://images.example/a.jpg"}},
			Crew: []metadata.CrewCredit{{Name: "Directora", Job: "Director", Department: "Directing", PhotoURL: "https://images.example/d.jpg"}},
		},
	}
	if err := ApplyMovieDetails(ctx, db, itemID, details, "auto", 0.9); err != nil {
		t.Fatalf("ApplyMovieDetails: %v", err)
	}
	var raw sql.NullString
	if err := db.QueryRow(`SELECT credits FROM media_items WHERE id = ?`, itemID).Scan(&raw); err != nil {
		t.Fatalf("leer credits: %v", err)
	}
	decoded := decodeCredits(raw)
	if len(decoded.Cast) != 1 || decoded.Cast[0].Character != "Heroína" ||
		decoded.Cast[0].PhotoURL != "https://images.example/a.jpg" ||
		len(decoded.Crew) != 1 || decoded.Crew[0].Job != "Director" {
		t.Fatalf("credits guardados inesperados: %+v", decoded)
	}

	if err := ClearMovieMetadata(ctx, db, itemID, nil); err != nil {
		t.Fatalf("ClearMovieMetadata: %v", err)
	}
	if err := db.QueryRow(`SELECT credits FROM media_items WHERE id = ?`, itemID).Scan(&raw); err != nil {
		t.Fatalf("leer credits tras limpiar: %v", err)
	}
	if raw.Valid {
		t.Fatalf("credits debe quedar NULL: %+v", raw)
	}
}

func TestApplyMovieDetailsRespetaBloqueosYLosDescartaAlCambiarTMDB(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	root := t.TempDir()
	file := filepath.Join(root, "Pelicula (2019).mkv")
	writeFile(t, file)
	libraryID := createLibrary(t, db, root)

	result, err := db.Exec(`
		INSERT INTO media_items (
			library_id, path, title, kind, tmdb_id, poster_url, poster_locked,
			backdrop_url, backdrop_locked
		) VALUES (?, ?, 'Pelicula', ?, 10, 'https://images.example/poster-propio.jpg', 1,
			'https://images.example/fondo-propio.jpg', 1)`,
		libraryID, file, KindMovie)
	if err != nil {
		t.Fatalf("insertar fila: %v", err)
	}
	itemID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de la fila: %v", err)
	}

	same := &metadata.Details{
		TMDBID:      10,
		Title:       "Pelicula",
		PosterURL:   "https://images.example/poster-tmdb.jpg",
		BackdropURL: "https://images.example/fondo-tmdb.jpg",
	}
	if err := ApplyMovieDetails(ctx, db, itemID, same, "auto", 1); err != nil {
		t.Fatalf("ApplyMovieDetails mismo TMDB: %v", err)
	}
	row := readTMDBMetadata(t, db, file)
	if !row.posterURL.Valid || row.posterURL.String != "https://images.example/poster-propio.jpg" ||
		row.posterLocked != 1 ||
		!row.backdropURL.Valid || row.backdropURL.String != "https://images.example/fondo-propio.jpg" ||
		row.backdropLocked != 1 {
		t.Fatalf("las imágenes fijadas cambiaron con el mismo TMDB: %+v", row)
	}

	changed := &metadata.Details{
		TMDBID:      11,
		Title:       "Pelicula",
		PosterURL:   "https://images.example/poster-nuevo.jpg",
		BackdropURL: "https://images.example/fondo-nuevo.jpg",
	}
	if err := ApplyMovieDetails(ctx, db, itemID, changed, "auto", 1); err != nil {
		t.Fatalf("ApplyMovieDetails con otro TMDB: %v", err)
	}
	row = readTMDBMetadata(t, db, file)
	if !row.posterURL.Valid || row.posterURL.String != "https://images.example/poster-nuevo.jpg" ||
		row.posterLocked != 0 ||
		!row.backdropURL.Valid || row.backdropURL.String != "https://images.example/fondo-nuevo.jpg" ||
		row.backdropLocked != 0 {
		t.Fatalf("las imágenes no se sustituyeron al cambiar de TMDB: %+v", row)
	}

	unlocked := &metadata.Details{
		TMDBID:      11,
		Title:       "Pelicula",
		PosterURL:   "https://images.example/poster-sin-bloqueo.jpg",
		BackdropURL: "https://images.example/fondo-sin-bloqueo.jpg",
	}
	if err := ApplyMovieDetails(ctx, db, itemID, unlocked, "auto", 1); err != nil {
		t.Fatalf("ApplyMovieDetails sin bloqueo: %v", err)
	}
	row = readTMDBMetadata(t, db, file)
	if !row.posterURL.Valid || row.posterURL.String != "https://images.example/poster-sin-bloqueo.jpg" ||
		row.posterLocked != 0 ||
		!row.backdropURL.Valid || row.backdropURL.String != "https://images.example/fondo-sin-bloqueo.jpg" ||
		row.backdropLocked != 0 {
		t.Fatalf("sin bloqueo debían actualizarse las imágenes: %+v", row)
	}
}

func TestScannerConTMDBErrorDeRedNoTocaLaFila(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	root := t.TempDir()
	file := filepath.Join(root, "Fallo (2020).mkv")
	writeFile(t, file)
	libraryID := createLibrary(t, db, root)

	if _, err := db.Exec(`
		INSERT INTO media_items (
			library_id, path, title, year, kind, tmdb_id, overview,
			metadata_source, match_score
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		libraryID, file, "Fallo", 2020, KindMovie, 5,
		"Sinopsis anterior", "omdb", 0.81,
	); err != nil {
		t.Fatalf("insertar fila previa: %v", err)
	}

	fake := &fakeTMDB{searchErr: errors.New("red caída")}
	res, err := New(db).WithTMDB(fake).Scan(ctx)
	if err != nil {
		t.Fatalf("el escaneo no debe fallar: %v", err)
	}
	if res.Identified != 0 || res.Unidentified != 0 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("avisos = %v, se esperaba uno", res.Warnings)
	}
	if fake.detailsCalls != 0 {
		t.Fatalf("no se debe pedir la ficha si falla la búsqueda: %d", fake.detailsCalls)
	}

	row := readTMDBMetadata(t, db, file)
	if !row.tmdbID.Valid || row.tmdbID.Int64 != 5 ||
		!row.overview.Valid || row.overview.String != "Sinopsis anterior" ||
		!row.source.Valid || row.source.String != "omdb" ||
		!row.matchScore.Valid || row.matchScore.Float64 != 0.81 {
		t.Fatalf("la fila cambió: %+v", row)
	}
}

func TestScannerConTMDBNoUsaOMDb(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)

	root := t.TempDir()
	file := filepath.Join(root, "Pelicula TMDB (2019).mkv")
	writeFile(t, file)
	createLibrary(t, db, root)

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":"True","Title":"Pelicula TMDB","Plot":"x"}`))
	}))
	t.Cleanup(server.Close)

	fake := &fakeTMDB{details: &metadata.Details{
		TMDBID:   42,
		Title:    "Pelicula TMDB",
		Overview: "Sinopsis de TMDB.",
	}}
	res, err := New(db).WithOMDb(newMetadataTestClient(t, server.URL)).WithTMDB(fake).Scan(ctx)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if res.Identified != 1 || res.Unidentified != 0 {
		t.Fatalf("resultado inesperado: %+v", res)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("avisos inesperados: %v", res.Warnings)
	}
	if calls.Load() != 0 {
		t.Fatalf("con TMDB no se debe consultar OMDb: %d peticiones", calls.Load())
	}

	row := readTMDBMetadata(t, db, file)
	if !row.source.Valid || row.source.String != "tmdb" {
		t.Fatalf("metadata_source = %+v", row.source)
	}
}
