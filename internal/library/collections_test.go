package library

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/catlinux/cinecat/internal/metadata"
)

func collectionTestItem(t *testing.T, db *sql.DB, title string) int64 {
	t.Helper()
	libraryID := createLibrary(t, db, t.TempDir())
	return insertItemFixture(t, db, itemFixture{
		libraryID: libraryID,
		path:      "/media/" + title + ".mkv",
		title:     title,
		kind:      KindMovie,
	})
}

type collectionRow struct {
	ID               int64
	Name             string
	PosterURL        sql.NullString
	BackdropURL      sql.NullString
	TMDBCollectionID sql.NullInt64
	Source           string
	Edited           int
	Deleted          int
}

func readCollection(t *testing.T, db *sql.DB, tmdbID int) collectionRow {
	t.Helper()
	var row collectionRow
	if err := db.QueryRow(`
		SELECT id, name, poster_url, backdrop_url, tmdb_collection_id, source, edited, deleted
		  FROM collections WHERE tmdb_collection_id = ?`, tmdbID).
		Scan(&row.ID, &row.Name, &row.PosterURL, &row.BackdropURL,
			&row.TMDBCollectionID, &row.Source, &row.Edited, &row.Deleted); err != nil {
		t.Fatalf("leer colección %d: %v", tmdbID, err)
	}
	return row
}

func collectionCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM collections`).Scan(&count); err != nil {
		t.Fatalf("contar colecciones: %v", err)
	}
	return count
}

func readMembership(t *testing.T, db *sql.DB, collectionID, itemID int64) (position sql.NullInt64, excluded int, ok bool) {
	t.Helper()
	err := db.QueryRow(`
		SELECT position, excluded FROM collection_items
		 WHERE collection_id = ? AND media_item_id = ?`, collectionID, itemID).
		Scan(&position, &excluded)
	if errors.Is(err, sql.ErrNoRows) {
		return position, 0, false
	}
	if err != nil {
		t.Fatalf("leer pertenencia: %v", err)
	}
	return position, excluded, true
}

func collectionChecked(t *testing.T, db *sql.DB, itemID int64) int {
	t.Helper()
	var checked int
	if err := db.QueryRow(`SELECT collection_checked FROM media_items WHERE id = ?`, itemID).
		Scan(&checked); err != nil {
		t.Fatalf("leer collection_checked: %v", err)
	}
	return checked
}

func insertManualCollection(t *testing.T, db *sql.DB, name string, itemID int64) int64 {
	t.Helper()
	result, err := db.Exec(
		`INSERT INTO collections (name, source) VALUES (?, 'manual')`, name)
	if err != nil {
		t.Fatalf("crear colección manual: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de colección manual: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO collection_items (collection_id, media_item_id, position) VALUES (?, ?, 3)`,
		id, itemID); err != nil {
		t.Fatalf("añadir a colección manual: %v", err)
	}
	return id
}

func TestApplyCollectionCreaYReutiliza(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	first := collectionTestItem(t, db, "Primera")
	second := collectionTestItem(t, db, "Segunda")

	ref := &metadata.CollectionRef{
		TMDBID:      100,
		Name:        "La saga",
		PosterURL:   "https://images.example/saga.jpg",
		BackdropURL: "https://images.example/saga-backdrop.jpg",
	}
	if err := ApplyCollection(ctx, db, first, ref); err != nil {
		t.Fatalf("ApplyCollection primera: %v", err)
	}
	collection := readCollection(t, db, 100)
	if collection.Source != "tmdb" || collection.Edited != 0 || collection.Deleted != 0 ||
		collection.Name != "La saga" ||
		!collection.PosterURL.Valid || collection.PosterURL.String != ref.PosterURL {
		t.Fatalf("colección creada inesperada: %+v", collection)
	}
	position, excluded, ok := readMembership(t, db, collection.ID, first)
	if !ok || position.Valid || excluded != 0 || collectionChecked(t, db, first) != 1 {
		t.Fatalf("pertenencia inicial inesperada: %v %v %v", position, excluded, ok)
	}

	ref.Name = "La saga actualizada"
	if err := ApplyCollection(ctx, db, second, ref); err != nil {
		t.Fatalf("ApplyCollection segunda: %v", err)
	}
	if count := collectionCount(t, db); count != 1 {
		t.Fatalf("colecciones = %d, se esperaba 1", count)
	}
	if collection = readCollection(t, db, 100); collection.Name != "La saga actualizada" {
		t.Fatalf("la colección no se actualizó: %+v", collection)
	}
	if _, _, ok := readMembership(t, db, collection.ID, second); !ok {
		t.Fatal("la segunda película no entró en la colección")
	}
	if collectionChecked(t, db, second) != 1 {
		t.Fatal("la segunda película debe quedar comprobada")
	}
}

func TestApplyCollectionRespetaColeccionBorrada(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	itemID := collectionTestItem(t, db, "Película")
	result, err := db.Exec(`
		INSERT INTO collections (name, tmdb_collection_id, source, deleted)
		VALUES ('Borrada', 200, 'tmdb', 1)`)
	if err != nil {
		t.Fatalf("crear colección borrada: %v", err)
	}
	collectionID, _ := result.LastInsertId()

	if err := ApplyCollection(ctx, db, itemID, &metadata.CollectionRef{
		TMDBID: 200,
		Name:   "Nombre nuevo",
	}); err != nil {
		t.Fatalf("ApplyCollection: %v", err)
	}
	if count := collectionCount(t, db); count != 1 {
		t.Fatalf("no debe crearse otra colección: %d", count)
	}
	collection := readCollection(t, db, 200)
	if collection.Name != "Borrada" || collection.Deleted != 1 {
		t.Fatalf("la colección borrada cambió: %+v", collection)
	}
	if _, _, ok := readMembership(t, db, collectionID, itemID); ok {
		t.Fatal("la colección borrada no debe recibir películas")
	}
	if collectionChecked(t, db, itemID) != 1 {
		t.Fatal("la película debe quedar comprobada")
	}
}

func TestApplyCollectionConservaPeliculaExcluida(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	itemID := collectionTestItem(t, db, "Excluida")
	ref := &metadata.CollectionRef{TMDBID: 300, Name: "Saga"}
	if err := ApplyCollection(ctx, db, itemID, ref); err != nil {
		t.Fatalf("ApplyCollection inicial: %v", err)
	}
	collection := readCollection(t, db, 300)
	if _, err := db.Exec(`
		UPDATE collection_items SET excluded = 1, position = 7
		 WHERE collection_id = ? AND media_item_id = ?`, collection.ID, itemID); err != nil {
		t.Fatalf("excluir película: %v", err)
	}

	if err := ApplyCollection(ctx, db, itemID, ref); err != nil {
		t.Fatalf("ApplyCollection repetida: %v", err)
	}
	position, excluded, ok := readMembership(t, db, collection.ID, itemID)
	if !ok || excluded != 1 || !position.Valid || position.Int64 != 7 {
		t.Fatalf("la exclusión se perdió: %v %v %v", position, excluded, ok)
	}
}

func TestApplyCollectionRespetaEditedYActualizaSinEdicion(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	itemID := collectionTestItem(t, db, "Película")

	if _, err := db.Exec(`
		INSERT INTO collections (name, poster_url, tmdb_collection_id, source, edited)
		VALUES ('Mía', 'https://images.example/vieja.jpg', 400, 'tmdb', 1)`); err != nil {
		t.Fatalf("crear colección editada: %v", err)
	}
	if err := ApplyCollection(ctx, db, itemID, &metadata.CollectionRef{
		TMDBID:    400,
		Name:      "De TMDB",
		PosterURL: "https://images.example/nueva.jpg",
	}); err != nil {
		t.Fatalf("ApplyCollection editada: %v", err)
	}
	collection := readCollection(t, db, 400)
	if collection.Name != "Mía" || collection.PosterURL.String != "https://images.example/vieja.jpg" {
		t.Fatalf("la colección editada cambió: %+v", collection)
	}

	if _, err := db.Exec(`
		INSERT INTO collections (name, tmdb_collection_id, source, edited)
		VALUES ('Antigua', 401, 'tmdb', 0)`); err != nil {
		t.Fatalf("crear colección automática: %v", err)
	}
	if err := ApplyCollection(ctx, db, itemID, &metadata.CollectionRef{
		TMDBID:    401,
		Name:      "Actualizada",
		PosterURL: "https://images.example/actual.jpg",
	}); err != nil {
		t.Fatalf("ApplyCollection automática: %v", err)
	}
	if updated := readCollection(t, db, 401); updated.Name != "Actualizada" ||
		updated.PosterURL.String != "https://images.example/actual.jpg" {
		t.Fatalf("la colección automática no se actualizó: %+v", updated)
	}
}

func TestApplyCollectionReasignaYQuitaSoloTMDB(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	itemID := collectionTestItem(t, db, "Película")
	manualID := insertManualCollection(t, db, "Manual", itemID)

	if err := ApplyCollection(ctx, db, itemID, &metadata.CollectionRef{TMDBID: 500, Name: "A"}); err != nil {
		t.Fatalf("ApplyCollection A: %v", err)
	}
	collectionA := readCollection(t, db, 500)
	if _, err := db.Exec(`
		UPDATE collection_items SET excluded = 1
		 WHERE collection_id = ? AND media_item_id = ?`, collectionA.ID, itemID); err != nil {
		t.Fatalf("excluir en A: %v", err)
	}

	if err := ApplyCollection(ctx, db, itemID, &metadata.CollectionRef{TMDBID: 501, Name: "B"}); err != nil {
		t.Fatalf("ApplyCollection B: %v", err)
	}
	collectionB := readCollection(t, db, 501)
	if _, _, ok := readMembership(t, db, collectionA.ID, itemID); ok {
		t.Fatal("la película debe salir de la colección anterior de TMDB")
	}
	if _, _, ok := readMembership(t, db, collectionB.ID, itemID); !ok {
		t.Fatal("la película debe entrar en la colección nueva")
	}
	if _, _, ok := readMembership(t, db, manualID, itemID); !ok {
		t.Fatal("la colección manual no debe tocarse")
	}

	if err := ApplyCollection(ctx, db, itemID, nil); err != nil {
		t.Fatalf("ApplyCollection nil: %v", err)
	}
	if _, _, ok := readMembership(t, db, collectionB.ID, itemID); ok {
		t.Fatal("sin colección debe salir de la colección de TMDB")
	}
	if _, _, ok := readMembership(t, db, manualID, itemID); !ok {
		t.Fatal("la colección manual debe sobrevivir a ApplyCollection nil")
	}
	if checked := collectionChecked(t, db, itemID); checked != 1 {
		t.Fatalf("collection_checked = %d, se esperaba 1", checked)
	}
}

func TestClearMovieMetadataQuitaColeccionesTMDB(t *testing.T) {
	ctx := context.Background()
	db := openTestDB(t)
	itemID := collectionTestItem(t, db, "Película")
	manualID := insertManualCollection(t, db, "Manual", itemID)

	if err := ApplyCollection(ctx, db, itemID, &metadata.CollectionRef{TMDBID: 600, Name: "Saga"}); err != nil {
		t.Fatalf("ApplyCollection: %v", err)
	}
	collection := readCollection(t, db, 600)

	if err := ClearMovieMetadata(ctx, db, itemID, nil); err != nil {
		t.Fatalf("ClearMovieMetadata: %v", err)
	}
	if _, _, ok := readMembership(t, db, collection.ID, itemID); ok {
		t.Fatal("ClearMovieMetadata debe quitar la pertenencia de TMDB")
	}
	if _, _, ok := readMembership(t, db, manualID, itemID); !ok {
		t.Fatal("ClearMovieMetadata no debe tocar la colección manual")
	}
	if checked := collectionChecked(t, db, itemID); checked != 0 {
		t.Fatalf("collection_checked = %d, se esperaba 0", checked)
	}
	if count := collectionCount(t, db); count != 2 {
		t.Fatalf("las colecciones no deben borrarse al limpiar la ficha: %d", count)
	}
}
