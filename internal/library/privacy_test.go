package library

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestListadosOcultosParaNoAdmin(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	userToken := nonAdminToken(t, ts)
	itemID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      filepath.Join(t.TempDir(), "privada.mkv"),
		title:     "Privada",
		cast:      `["Actor"]`,
	})
	if _, err := ts.db.Exec(`UPDATE media_items SET credits = ? WHERE id = ?`,
		`{"cast":[{"id":1,"name":"Actor","character":"Papel","photo":""}],"crew":[]}`,
		itemID); err != nil {
		t.Fatalf("credits: %v", err)
	}

	status, body := libraryHTTPGet(t,
		ts.URL+"/api/v1/items?limit=48", userToken)
	if status != http.StatusOK {
		t.Fatalf("listado no admin: %d %s", status, body)
	}
	var listing itemsResponse
	if err := json.Unmarshal(body, &listing); err != nil {
		t.Fatalf("listado: %v", err)
	}
	if len(listing.Items) != 1 {
		t.Fatalf("ítems = %d", len(listing.Items))
	}
	if listing.Items[0].Path != "" || len(listing.Items[0].Credits.Cast) != 0 ||
		len(listing.Items[0].Cast) != 0 {
		t.Fatalf("listado no debe exponer ruta ni créditos: %+v", listing.Items[0])
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10), userToken)
	if status != http.StatusOK {
		t.Fatalf("ficha no admin: %d %s", status, body)
	}
	var detail mediaItemDTO
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("ficha: %v", err)
	}
	if detail.Path != "" || len(detail.Cast) != 1 || len(detail.Credits.Cast) != 1 {
		t.Fatalf("la ficha no admin debe conservar créditos y ocultar ruta: %+v", detail)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(itemID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("ficha admin: %d %s", status, body)
	}
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("ficha admin: %v", err)
	}
	if detail.Path == "" {
		t.Fatal("el administrador debe recibir la ruta del ítem")
	}
}

func TestLibrariesOcultanRutasNoAdmin(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	root := t.TempDir()
	libraryID := createLibrary(t, ts.db, root)
	userToken := nonAdminToken(t, ts)

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/libraries", userToken)
	if status != http.StatusOK {
		t.Fatalf("bibliotecas no admin: %d %s", status, body)
	}
	var userResponse librariesResponse
	if err := json.Unmarshal(body, &userResponse); err != nil {
		t.Fatalf("bibliotecas: %v", err)
	}
	if len(userResponse.Libraries) != 1 {
		t.Fatalf("bibliotecas = %d", len(userResponse.Libraries))
	}
	library := userResponse.Libraries[0]
	if library.Path != "" || len(library.Paths) != 0 {
		t.Fatalf("el no admin no debe recibir rutas: %+v", library)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/libraries", ts.token)
	if status != http.StatusOK {
		t.Fatalf("bibliotecas admin: %d %s", status, body)
	}
	var adminResponse librariesResponse
	if err := json.Unmarshal(body, &adminResponse); err != nil {
		t.Fatalf("bibliotecas admin: %v", err)
	}
	if len(adminResponse.Libraries) != 1 ||
		adminResponse.Libraries[0].Path != root ||
		adminResponse.Libraries[0].ID != libraryID {
		t.Fatalf("el admin debe recibir la ruta: %+v", adminResponse.Libraries)
	}
}

func TestFichaTecnicaOcultaRutaNoAdmin(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	id, file := insertFileInfoItem(t, ts, "Sub/Pelicula (2019).mp4", true)
	ts.api.WithFileInspector(&fakeFileInspector{info: fileInfoFixture()})
	userToken := nonAdminToken(t, ts)

	status, body := libraryHTTPGet(t, itemFileInfoURL(ts, id), userToken)
	if status != http.StatusOK {
		t.Fatalf("ficha técnica no admin: %d %s", status, body)
	}
	var userResponse fileInfoResponse
	if err := json.Unmarshal(body, &userResponse); err != nil {
		t.Fatalf("ficha técnica: %v", err)
	}
	if userResponse.Path != "" {
		t.Fatalf("el no admin no debe recibir la ruta: %q", userResponse.Path)
	}

	status, body = libraryHTTPGet(t, itemFileInfoURL(ts, id), ts.token)
	if status != http.StatusOK {
		t.Fatalf("ficha técnica admin: %d %s", status, body)
	}
	var adminResponse fileInfoResponse
	if err := json.Unmarshal(body, &adminResponse); err != nil {
		t.Fatalf("ficha técnica admin: %v", err)
	}
	if adminResponse.Path != "Sub/Pelicula (2019).mp4" {
		t.Fatalf("el admin debe recibir la ruta; archivo=%q", file)
	}
}

func TestSerieOcultaCarpetaNoAdmin(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	seriesID := insertSeriesHTTPFixture(t, ts, seriesHTTPFixture{
		libraryID: libraryID, folder: "Serie privada", title: "Serie",
	})
	if _, err := ts.db.Exec(`UPDATE series SET credits = ? WHERE id = ?`,
		`{"cast":[{"id":1,"name":"Actor","character":"Papel","photo":""}],"crew":[]}`,
		seriesID); err != nil {
		t.Fatalf("credits: %v", err)
	}
	userToken := nonAdminToken(t, ts)

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/series/"+strconv.FormatInt(seriesID, 10), userToken)
	if status != http.StatusOK {
		t.Fatalf("serie no admin: %d %s", status, body)
	}
	var detail seriesDetailDTO
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("serie: %v", err)
	}
	if detail.Folder != "" || len(detail.Credits.Cast) != 1 {
		t.Fatalf("serie no admin inesperada: %+v", detail)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/series/"+strconv.FormatInt(seriesID, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("serie admin: %d %s", status, body)
	}
	if err := json.Unmarshal(body, &detail); err != nil {
		t.Fatalf("serie admin: %v", err)
	}
	if detail.Folder == "" {
		t.Fatal("el administrador debe recibir la carpeta de la serie")
	}
}

func TestListadoItemsGrandeSinCreditos(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	userID := progressUserID(t, ts.db, "ana")

	var cast strings.Builder
	cast.WriteString(`{"cast":[`)
	for i := 0; i < 70; i++ {
		if i > 0 {
			cast.WriteByte(',')
		}
		cast.WriteString(`{"id":`)
		cast.WriteString(strconv.Itoa(i + 1))
		cast.WriteString(`,"name":"Actor de reparto con nombre largo","character":"Personaje secundario","photo":"https://images.example/w185/foto.jpg"}`)
	}
	cast.WriteString(`],"crew":[]}`)

	ids := make([]int64, 0, 48)
	for i := 0; i < 48; i++ {
		id := insertItemFixture(t, ts.db, itemFixture{
			libraryID: libraryID,
			path:      filepath.Join(t.TempDir(), "peli.mkv"),
			title:     "Peli",
			cast:      `["Actor"]`,
		})
		if _, err := ts.db.Exec(`UPDATE media_items SET credits = ? WHERE id = ?`, cast.String(), id); err != nil {
			t.Fatalf("credits: %v", err)
		}
		ids = append(ids, id)
	}

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items?limit=48", ts.token)
	if status != http.StatusOK {
		t.Fatalf("listado: %d", status)
	}
	full := make([]mediaItemDTO, 0, len(ids))
	for _, id := range ids {
		item, err := queryMediaItem(context.Background(), ts.db, id, userID)
		if err != nil {
			t.Fatalf("ítem completo %d: %v", id, err)
		}
		full = append(full, item)
	}
	baseline, err := json.Marshal(itemsResponse{Items: full, Total: 48, Limit: 48})
	if err != nil {
		t.Fatalf("baseline: %v", err)
	}
	if len(body)*4 >= len(baseline) {
		t.Fatalf("listado demasiado grande: %d bytes frente a %d completos", len(body), len(baseline))
	}
}
