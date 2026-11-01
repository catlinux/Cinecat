package library

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/catlinux/cinecat/internal/metadata"
)

func insertRefreshMovie(
	t *testing.T,
	db *sql.DB,
	libraryID int64,
	title string,
	tmdbID int,
	status string,
	refreshedAt string,
	overview string,
) int64 {
	t.Helper()
	id := insertItemFixture(t, db, itemFixture{
		libraryID:      libraryID,
		path:           "/media/" + title + ".mkv",
		title:          title,
		kind:           KindMovie,
		tmdbID:         tmdbID,
		metadataSource: "tmdb",
		matchStatus:    status,
		overview:       overview,
	})
	if _, err := db.Exec(
		`UPDATE media_items SET metadata_refreshed_at = ? WHERE id = ?`,
		refreshedAt, id,
	); err != nil {
		t.Fatalf("fecha de refresco de %q: %v", title, err)
	}
	return id
}

func insertRefreshSeries(
	t *testing.T,
	db *sql.DB,
	libraryID int64,
	title string,
	tmdbID int,
	status string,
	refreshedAt string,
	overview string,
) int64 {
	t.Helper()
	result, err := db.Exec(`
		INSERT INTO series
			(library_id, folder, title, overview, tmdb_id, metadata_source,
			 match_status, metadata_refreshed_at)
		VALUES (?, ?, ?, ?, ?, 'tmdb', ?, ?)`,
		libraryID, title, title, overview, tmdbID, status, refreshedAt)
	if err != nil {
		t.Fatalf("crear serie %q: %v", title, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de la serie %q: %v", title, err)
	}
	return id
}

func metadataRefreshedAt(t *testing.T, db *sql.DB, table string, id int64) string {
	t.Helper()
	var value sql.NullString
	if err := db.QueryRow(
		`SELECT metadata_refreshed_at FROM `+table+` WHERE id = ?`, id,
	).Scan(&value); err != nil {
		t.Fatalf("leer metadata_refreshed_at de %s %d: %v", table, id, err)
	}
	return value.String
}

func TestRefreshStaleTMDBMetadataActualizaAntiguosYRespetaRecientesYManuales(t *testing.T) {
	db := openTestDB(t)
	libraryID := createLibrary(t, db, t.TempDir())
	oldDate := "2020-01-01T00:00:00.000Z"
	recentDate := time.Now().UTC().Format(metadataRefreshTimeLayout)

	oldMovie := insertRefreshMovie(t, db, libraryID, "Antigua", 603, "auto", oldDate, "Antigua")
	recentMovie := insertRefreshMovie(t, db, libraryID, "Reciente", 604, "auto", recentDate, "Reciente")
	manualMovie := insertRefreshMovie(t, db, libraryID, "Manual", 605, "manual", oldDate, "Manual")
	oldSeries := insertRefreshSeries(t, db, libraryID, "Serie antigua", 1399, "auto", oldDate, "Antigua")
	recentSeries := insertRefreshSeries(t, db, libraryID, "Serie reciente", 1400, "auto", recentDate, "Reciente")
	manualSeries := insertRefreshSeries(t, db, libraryID, "Serie manual", 1401, "manual", oldDate, "Manual")
	if _, err := db.Exec(
		`UPDATE media_items SET poster_url = 'manual.jpg', poster_locked = 1 WHERE id = ?`,
		oldMovie,
	); err != nil {
		t.Fatalf("bloquear cartel de película: %v", err)
	}
	if _, err := db.Exec(
		`UPDATE series SET poster_url = 'manual-serie.jpg', poster_locked = 1 WHERE id = ?`,
		oldSeries,
	); err != nil {
		t.Fatalf("bloquear cartel de serie: %v", err)
	}

	fake := &fakeTMDB{
		details: &metadata.Details{
			TMDBID: 603, Title: "Película nueva", Year: 1999,
			Overview: "Sinopsis nueva", PosterURL: "nueva.jpg",
		},
		tvDetails: &metadata.Details{
			TMDBID: 1399, Title: "Serie nueva", Year: 2011,
			Overview: "Sinopsis nueva", PosterURL: "nueva-serie.jpg",
		},
	}
	api := &API{db: db, tmdb: fake, metadataRefreshDelay: 0}

	refreshed, err := api.refreshStaleTMDBMetadata(context.Background())
	if err != nil {
		t.Fatalf("refrescar metadatos: %v", err)
	}
	if refreshed != 2 {
		t.Fatalf("refrescados = %d, se esperaban 2", refreshed)
	}
	if fake.detailsCalls != 1 || fake.tvDetailsCalls != 1 {
		t.Fatalf("llamadas inesperadas: películas=%d series=%d", fake.detailsCalls, fake.tvDetailsCalls)
	}
	if metadataRefreshedAt(t, db, "media_items", oldMovie) == oldDate {
		t.Fatal("la película antigua no actualizó su fecha")
	}
	if metadataRefreshedAt(t, db, "media_items", recentMovie) != recentDate {
		t.Fatal("la película reciente no debía refrescarse")
	}
	if metadataRefreshedAt(t, db, "media_items", manualMovie) != oldDate {
		t.Fatal("la película manual no debía refrescarse")
	}
	if metadataRefreshedAt(t, db, "series", oldSeries) == oldDate {
		t.Fatal("la serie antigua no actualizó su fecha")
	}
	if metadataRefreshedAt(t, db, "series", recentSeries) != recentDate {
		t.Fatal("la serie reciente no debía refrescarse")
	}
	if metadataRefreshedAt(t, db, "series", manualSeries) != oldDate {
		t.Fatal("la serie manual no debía refrescarse")
	}

	var movieTitle, movieOverview, moviePoster, manualOverview string
	if err := db.QueryRow(`
		SELECT display_title, COALESCE(overview, ''), COALESCE(poster_url, '')
		  FROM media_items WHERE id = ?`, oldMovie,
	).Scan(&movieTitle, &movieOverview, &moviePoster); err != nil {
		t.Fatalf("leer película refrescada: %v", err)
	}
	if movieTitle != "Película nueva" || movieOverview != "Sinopsis nueva" || moviePoster != "manual.jpg" {
		t.Fatalf("película refrescada inesperada: título=%q sinopsis=%q cartel=%q",
			movieTitle, movieOverview, moviePoster)
	}
	if err := db.QueryRow(
		`SELECT COALESCE(overview, '') FROM media_items WHERE id = ?`, manualMovie,
	).Scan(&manualOverview); err != nil {
		t.Fatalf("leer película manual: %v", err)
	}
	if manualOverview != "Manual" {
		t.Fatalf("la película manual cambió: %q", manualOverview)
	}
	var seriesOverview, seriesPoster string
	if err := db.QueryRow(`
		SELECT COALESCE(overview, ''), COALESCE(poster_url, '')
		  FROM series WHERE id = ?`, oldSeries,
	).Scan(&seriesOverview, &seriesPoster); err != nil {
		t.Fatalf("leer serie refrescada: %v", err)
	}
	if seriesOverview != "Sinopsis nueva" || seriesPoster != "manual-serie.jpg" {
		t.Fatalf("serie refrescada inesperada: sinopsis=%q cartel=%q", seriesOverview, seriesPoster)
	}
}

func TestRefreshStaleTMDBMetadataSinTMDBNoHaceNada(t *testing.T) {
	db := openTestDB(t)
	libraryID := createLibrary(t, db, t.TempDir())
	oldDate := "2020-01-01T00:00:00.000Z"
	id := insertRefreshMovie(t, db, libraryID, "Antigua", 603, "auto", oldDate, "Antigua")

	api := &API{db: db, metadataRefreshDelay: 0}
	refreshed, err := api.refreshStaleTMDBMetadata(context.Background())
	if err != nil || refreshed != 0 {
		t.Fatalf("sin TMDB: refrescados=%d err=%v", refreshed, err)
	}
	if metadataRefreshedAt(t, db, "media_items", id) != oldDate {
		t.Fatal("sin TMDB no debe cambiar la fecha")
	}
}

func TestRefreshStaleTMDBMetadataRespetaLosLimitesDelLote(t *testing.T) {
	db := openTestDB(t)
	libraryID := createLibrary(t, db, t.TempDir())
	oldDate := "2020-01-01T00:00:00.000Z"
	for i := 0; i < 41; i++ {
		insertRefreshMovie(t, db, libraryID, fmt.Sprintf("Película %02d", i),
			1000+i, "auto", oldDate, "Antigua")
	}
	for i := 0; i < 21; i++ {
		insertRefreshSeries(t, db, libraryID, fmt.Sprintf("Serie %02d", i),
			2000+i, "auto", oldDate, "Antigua")
	}

	fake := &fakeTMDB{}
	api := &API{db: db, tmdb: fake, metadataRefreshDelay: 0}
	refreshed, err := api.refreshStaleTMDBMetadata(context.Background())
	if err != nil {
		t.Fatalf("refrescar metadatos: %v", err)
	}
	if refreshed != 60 {
		t.Fatalf("refrescados = %d, se esperaban 60", refreshed)
	}
	if fake.detailsCalls != 40 || fake.tvDetailsCalls != 20 {
		t.Fatalf("límites incumplidos: películas=%d series=%d", fake.detailsCalls, fake.tvDetailsCalls)
	}
}

func TestRefreshStaleTMDBMetadataAbortaAnteUnError(t *testing.T) {
	db := openTestDB(t)
	libraryID := createLibrary(t, db, t.TempDir())
	oldDate := "2020-01-01T00:00:00.000Z"
	first := insertRefreshMovie(t, db, libraryID, "Primera", 603, "auto", oldDate, "Antigua")
	insertRefreshMovie(t, db, libraryID, "Segunda", 604, "auto", oldDate, "Antigua")

	fake := &fakeTMDB{detailsErr: metadata.ErrRateLimited}
	api := &API{db: db, tmdb: fake, metadataRefreshDelay: 0}
	refreshed, err := api.refreshStaleTMDBMetadata(context.Background())
	if err == nil {
		t.Fatal("se esperaba el error de TMDB")
	}
	if refreshed != 0 || fake.detailsCalls != 1 {
		t.Fatalf("el lote debía abortar: refrescados=%d llamadas=%d", refreshed, fake.detailsCalls)
	}
	if metadataRefreshedAt(t, db, "media_items", first) != oldDate {
		t.Fatal("un error no debe actualizar la fecha")
	}
}
