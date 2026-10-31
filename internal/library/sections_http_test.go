package library

import (
	"encoding/json"
	"net/http"
	"testing"
)

func sectionsFor(t *testing.T, ts *libraryTestServer, token string) sectionsResponse {
	t.Helper()
	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/sections", token)
	if status != http.StatusOK {
		t.Fatalf("secciones: código = %d, cuerpo = %s", status, body)
	}
	var response sectionsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("respuesta de secciones: %v (%s)", err, body)
	}
	return response
}

func TestSectionsSinSesionDevuelve401(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/sections", "")
	if status != http.StatusUnauthorized {
		t.Fatalf("secciones sin sesión: código = %d, cuerpo = %s", status, body)
	}
}

func TestSectionsAdministradorSiempreTieneTodo(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	response := sectionsFor(t, ts, ts.token)
	if !response.Movies || !response.Series || !response.Collections {
		t.Fatalf("el administrador debe ver todas las secciones: %+v", response)
	}
}

func TestSectionsUsuarioSoloConSeries(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	if _, err := ts.db.Exec(`
		INSERT INTO series (library_id, folder, title)
		VALUES (?, 'Serie', 'Serie')`, libraryID); err != nil {
		t.Fatalf("crear serie: %v", err)
	}
	token, _ := partialAccessToken(t, ts, libraryID)

	response := sectionsFor(t, ts, token)
	if response.Movies || response.Collections || !response.Series {
		t.Fatalf("secciones inesperadas para un usuario de solo series: %+v", response)
	}
}

func TestSectionsUsuarioConPeliculaYColeccion(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	itemID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID,
		path:      "/media/Pelicula.mkv",
		title:     "Película",
		kind:      KindMovie,
	})
	insertManualCollection(t, ts.db, "Saga", itemID)
	token, _ := partialAccessToken(t, ts, libraryID)

	response := sectionsFor(t, ts, token)
	if !response.Movies || response.Series || !response.Collections {
		t.Fatalf("secciones inesperadas para una película en colección: %+v", response)
	}
}
