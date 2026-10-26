package library

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
)

func TestWatchState(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	id := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/Pelicula (2019).mkv", title: "Pelicula",
	})
	base := ts.URL + "/api/v1/items/" + strconv.FormatInt(id, 10) + "/watched"

	readItem := func(token string) mediaItemDTO {
		t.Helper()
		status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(id, 10), token)
		if status != http.StatusOK {
			t.Fatalf("ficha: código = %d, cuerpo = %s", status, body)
		}
		var item mediaItemDTO
		if err := json.Unmarshal(body, &item); err != nil {
			t.Fatalf("ficha: %v", err)
		}
		return item
	}

	if item := readItem(ts.token); item.Watched {
		t.Fatal("el ítem no debe empezar visto")
	}
	if status, body := libraryHTTPDo(t, http.MethodPut, base, ts.token, nil); status != http.StatusNoContent {
		t.Fatalf("marcar: código = %d, cuerpo = %s", status, body)
	}
	if item := readItem(ts.token); !item.Watched {
		t.Fatal("el ítem debe quedar visto")
	}
	if status, body := libraryHTTPDo(t, http.MethodPut, base, ts.token, nil); status != http.StatusNoContent {
		t.Fatalf("marcar otra vez: código = %d, cuerpo = %s", status, body)
	}
	var count int
	if err := ts.db.QueryRow(`SELECT count(*) FROM watch_state WHERE media_item_id = ?`, id).Scan(&count); err != nil {
		t.Fatalf("contar visionados: %v", err)
	}
	if count != 1 {
		t.Fatalf("marcar debe ser idempotente; hay %d filas", count)
	}

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items?watched=true", ts.token)
	if status != http.StatusOK {
		t.Fatalf("filtro watched=true: código = %d, cuerpo = %s", status, body)
	}
	var list itemsResponse
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("filtro watched=true: %v", err)
	}
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].ID != id || !list.Items[0].Watched {
		t.Fatalf("filtro watched=true inesperado: %+v", list)
	}
	if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items?watched=false", ts.token); status != http.StatusOK || len(body) == 0 {
		t.Fatalf("filtro watched=false: código = %d", status)
	} else {
		var empty itemsResponse
		if err := json.Unmarshal(body, &empty); err != nil {
			t.Fatalf("filtro watched=false: %v", err)
		}
		if empty.Total != 0 {
			t.Fatalf("watched=false devolvió %d ítems", empty.Total)
		}
	}
	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items?watched=true&letter=P", ts.token)
	if status != http.StatusOK {
		t.Fatalf("watched combinado: código = %d", status)
	}
	var combined itemsResponse
	if err := json.Unmarshal(body, &combined); err != nil {
		t.Fatalf("watched combinado: %v", err)
	}
	if combined.Total != 1 || combined.Items[0].ID != id {
		t.Fatalf("watched combinado inesperado: %+v", combined)
	}

	if status, body := libraryHTTPDo(t, http.MethodDelete, base, ts.token, nil); status != http.StatusNoContent {
		t.Fatalf("desmarcar: código = %d, cuerpo = %s", status, body)
	}
	if item := readItem(ts.token); item.Watched {
		t.Fatal("el ítem no debe quedar visto")
	}
	if status, body := libraryHTTPDo(t, http.MethodDelete, base, ts.token, nil); status != http.StatusNoContent {
		t.Fatalf("desmarcar otra vez: código = %d, cuerpo = %s", status, body)
	}
}

func TestWatchStateAisladoPorUsuario(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	id := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/Pelicula (2019).mkv", title: "Pelicula",
	})
	base := ts.URL + "/api/v1/items/" + strconv.FormatInt(id, 10) + "/watched"
	userToken := nonAdminToken(t, ts)

	if status, body := libraryHTTPDo(t, http.MethodPut, base, ts.token, nil); status != http.StatusNoContent {
		t.Fatalf("marcar admin: código = %d, cuerpo = %s", status, body)
	}
	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(id, 10), userToken)
	if status != http.StatusOK {
		t.Fatalf("ficha de usuario: código = %d", status)
	}
	var item mediaItemDTO
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("ficha de usuario: %v", err)
	}
	if item.Watched {
		t.Fatal("el visionado de otro usuario no debe filtrarse")
	}

	if status, body := libraryHTTPDo(t, http.MethodPut, base, userToken, nil); status != http.StatusNoContent {
		t.Fatalf("marcar usuario: código = %d, cuerpo = %s", status, body)
	}
	if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items?watched=false", userToken); status != http.StatusOK {
		t.Fatalf("filtro de usuario: código = %d", status)
	} else {
		var list itemsResponse
		if err := json.Unmarshal(body, &list); err != nil {
			t.Fatalf("filtro de usuario: %v", err)
		}
		if list.Total != 0 {
			t.Fatalf("para el usuario el ítem está visto; watched=false devolvió %d", list.Total)
		}
	}

	if status, body := libraryHTTPDo(t, http.MethodDelete, base, userToken, nil); status != http.StatusNoContent {
		t.Fatalf("desmarcar usuario: código = %d, cuerpo = %s", status, body)
	}
	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/items/"+strconv.FormatInt(id, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("ficha de admin: código = %d", status)
	}
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("ficha de admin: %v", err)
	}
	if !item.Watched {
		t.Fatal("desmarcar a un usuario no debe afectar al admin")
	}
}

func TestWatchStateErrores(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	id := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/Pelicula.mkv", title: "Pelicula",
	})
	base := ts.URL + "/api/v1/items/" + strconv.FormatInt(id, 10) + "/watched"
	missing := ts.URL + "/api/v1/items/999999/watched"

	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		if status, body := libraryHTTPDo(t, method, base, "", nil); status != http.StatusUnauthorized {
			t.Fatalf("%s sin sesión: código = %d, cuerpo = %s", method, status, body)
		}
		if status, body := libraryHTTPDo(t, method, missing, ts.token, nil); status != http.StatusNotFound {
			t.Fatalf("%s inexistente: código = %d, cuerpo = %s", method, status, body)
		}
		if status, body := libraryHTTPDo(t, method, base, ts.token, nil); status != http.StatusNoContent {
			t.Fatalf("%s válido: código = %d, cuerpo = %s", method, status, body)
		}
	}

	if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items?watched=quiz", ts.token); status != http.StatusBadRequest {
		t.Fatalf("watched inválido: código = %d, cuerpo = %s", status, body)
	}
}
