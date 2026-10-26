package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/catlinux/cinecat/internal/auth"
	"github.com/catlinux/cinecat/internal/store"
)

func newLibraryHTTPTestServer(t *testing.T) (*httptest.Server, *sql.DB, string) {
	t.Helper()
	ctx := context.Background()

	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "cinecat.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close: %v", err)
		}
	})
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	db := st.DB()

	hash, err := auth.HashPassword("Clave-segura-1")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'admin')`,
		"ana", hash,
	); err != nil {
		t.Fatalf("crear usuario: %v", err)
	}

	authSvc := auth.New(db)
	token, _, err := authSvc.Login(ctx, "ana", "Clave-segura-1")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	api := NewAPI(db, authSvc, nil, New(db))
	mux := http.NewServeMux()
	api.Register(mux)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server, db, token
}

func libraryHTTPGet(t *testing.T, url, token string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("nueva petición: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("leer respuesta: %v", err)
	}
	return resp.StatusCode, body
}

func TestItemsExponeMetadatosOMDb(t *testing.T) {
	server, db, token := newLibraryHTTPTestServer(t)

	libraryID, err := db.Exec(`INSERT INTO libraries (path, name) VALUES (?, ?)`, "/media/pelis", "Pelis")
	if err != nil {
		t.Fatalf("crear biblioteca: %v", err)
	}
	libID, err := libraryID.LastInsertId()
	if err != nil {
		t.Fatalf("id de biblioteca: %v", err)
	}

	res, err := db.Exec(`
		INSERT INTO media_items (library_id, path, title, year, kind, overview, poster_url)
		VALUES (?, ?, ?, ?, 'movie', ?, ?)`,
		libID, "/media/pelis/Matrix (1999).mkv", "Matrix", 1999,
		"Un hacker descubre la verdad.", "https://images.example/matrix.jpg",
	)
	if err != nil {
		t.Fatalf("crear ítem con metadatos: %v", err)
	}
	withMetadata, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("id del ítem: %v", err)
	}

	res, err = db.Exec(`
		INSERT INTO media_items (library_id, path, title, year, kind)
		VALUES (?, ?, ?, ?, 'movie')`,
		libID, "/media/pelis/Sin datos (2001).mkv", "Sin datos", 2001,
	)
	if err != nil {
		t.Fatalf("crear ítem sin metadatos: %v", err)
	}
	withoutMetadata, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("id del ítem: %v", err)
	}

	status, body := libraryHTTPGet(t, server.URL+"/api/v1/items", token)
	if status != http.StatusOK {
		t.Fatalf("lista: código = %d, cuerpo = %s", status, body)
	}
	var list itemsResponse
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("respuesta de lista: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("ítems = %d, se esperaban 2", len(list.Items))
	}

	var withItem, withoutItem mediaItemDTO
	for _, item := range list.Items {
		switch item.ID {
		case withMetadata:
			withItem = item
		case withoutMetadata:
			withoutItem = item
		}
	}
	if withItem.Overview == nil || *withItem.Overview != "Un hacker descubre la verdad." {
		t.Fatalf("overview de lista = %+v", withItem.Overview)
	}
	if withItem.PosterURL == nil || *withItem.PosterURL != "https://images.example/matrix.jpg" {
		t.Fatalf("poster_url de lista = %+v", withItem.PosterURL)
	}
	if withoutItem.Overview != nil || withoutItem.PosterURL != nil {
		t.Fatalf("los metadatos ausentes deben ser nulos: %+v", withoutItem)
	}

	status, body = libraryHTTPGet(t, server.URL+"/api/v1/items/"+strconv.FormatInt(withMetadata, 10), token)
	if status != http.StatusOK {
		t.Fatalf("ficha: código = %d, cuerpo = %s", status, body)
	}
	var item mediaItemDTO
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("respuesta de ficha: %v", err)
	}
	if item.Overview == nil || item.PosterURL == nil {
		t.Fatalf("ficha sin metadatos: %+v", item)
	}

	status, body = libraryHTTPGet(t, server.URL+"/api/v1/items/"+strconv.FormatInt(withoutMetadata, 10), token)
	if status != http.StatusOK {
		t.Fatalf("ficha sin metadatos: código = %d", status)
	}
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("respuesta de ficha: %v", err)
	}
	if item.Overview != nil || item.PosterURL != nil {
		t.Fatalf("la ficha sin metadatos debe tener nulos: %+v", item)
	}
}

func TestItemsFiltraPorBiblioteca(t *testing.T) {
	server, db, token := newLibraryHTTPTestServer(t)

	libA, err := db.Exec(`INSERT INTO libraries (path, name) VALUES (?, ?)`, "/media/a", "A")
	if err != nil {
		t.Fatalf("crear biblioteca A: %v", err)
	}
	libraryA, err := libA.LastInsertId()
	if err != nil {
		t.Fatalf("id biblioteca A: %v", err)
	}
	libB, err := db.Exec(`INSERT INTO libraries (path, name) VALUES (?, ?)`, "/media/b", "B")
	if err != nil {
		t.Fatalf("crear biblioteca B: %v", err)
	}
	libraryB, err := libB.LastInsertId()
	if err != nil {
		t.Fatalf("id biblioteca B: %v", err)
	}

	for _, item := range []struct {
		libraryID int64
		path      string
	}{
		{libraryA, "/media/a/Uno (2019).mp4"},
		{libraryA, "/media/a/Dos (2020).mp4"},
		{libraryB, "/media/b/Tres (2021).mp4"},
	} {
		if _, err := db.Exec(
			`INSERT INTO media_items (library_id, path, title, kind) VALUES (?, ?, ?, 'movie')`,
			item.libraryID, item.path, item.path,
		); err != nil {
			t.Fatalf("crear ítem %q: %v", item.path, err)
		}
	}

	status, body := libraryHTTPGet(t,
		server.URL+"/api/v1/items?library_id="+strconv.FormatInt(libraryA, 10), token)
	if status != http.StatusOK {
		t.Fatalf("filtro: código = %d, cuerpo = %s", status, body)
	}
	var filtered itemsResponse
	if err := json.Unmarshal(body, &filtered); err != nil {
		t.Fatalf("respuesta filtrada: %v", err)
	}
	if filtered.Total != 2 || len(filtered.Items) != 2 {
		t.Fatalf("total = %d, ítems = %d; se esperaban 2 de la biblioteca A", filtered.Total, len(filtered.Items))
	}
	for _, item := range filtered.Items {
		if item.LibraryID != libraryA {
			t.Fatalf("ítem de otra biblioteca: %+v", item)
		}
	}

	status, body = libraryHTTPGet(t, server.URL+"/api/v1/items", token)
	if status != http.StatusOK {
		t.Fatalf("sin filtro: código = %d", status)
	}
	var all itemsResponse
	if err := json.Unmarshal(body, &all); err != nil {
		t.Fatalf("respuesta sin filtro: %v", err)
	}
	if all.Total != 3 || len(all.Items) != 3 {
		t.Fatalf("sin filtro: total = %d, ítems = %d", all.Total, len(all.Items))
	}

	status, body = libraryHTTPGet(t,
		server.URL+"/api/v1/items?library_id="+strconv.FormatInt(libraryB, 10), token)
	if status != http.StatusOK {
		t.Fatalf("filtro B: código = %d", status)
	}
	if err := json.Unmarshal(body, &filtered); err != nil {
		t.Fatalf("respuesta filtrada B: %v", err)
	}
	if filtered.Total != 1 || len(filtered.Items) != 1 {
		t.Fatalf("filtro B: total = %d, ítems = %d", filtered.Total, len(filtered.Items))
	}

	for _, invalid := range []string{"noesnumero", "0", "-3"} {
		status, body = libraryHTTPGet(t, server.URL+"/api/v1/items?library_id="+invalid, token)
		if status != http.StatusBadRequest {
			t.Fatalf("library_id=%s: código = %d, se esperaba 400; cuerpo = %s", invalid, status, body)
		}
	}
}
