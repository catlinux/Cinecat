package library

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func clipBrowseURL(ts *libraryTestServer, libraryID int64, query string) string {
	base := ts.URL + "/api/v1/libraries/" + itoa(libraryID) + "/browse"
	if query == "" {
		return base
	}
	return base + "?" + query
}

func TestClipBrowseJerarquiaSinRutasAbsolutas(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	base := t.TempDir()
	root1 := filepath.Join(base, "disco1", "Series")
	root2 := filepath.Join(base, "disco2", "Series")
	for _, path := range []string{root1, root2} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("crear carpeta: %v", err)
		}
	}
	libraryID := createClipLibrary(t, ts.db, root1)
	addScannerLibraryPath(t, ts.db, libraryID, root1)
	addScannerLibraryPath(t, ts.db, libraryID, root2)

	var root1ID, root2ID int64
	if err := ts.db.QueryRow(
		`SELECT id FROM library_paths WHERE library_id = ? ORDER BY id`, libraryID,
	).Scan(&root1ID); err != nil {
		t.Fatalf("leer raíz 1: %v", err)
	}
	if err := ts.db.QueryRow(
		`SELECT id FROM library_paths WHERE library_id = ? ORDER BY id DESC LIMIT 1`, libraryID,
	).Scan(&root2ID); err != nil {
		t.Fatalf("leer raíz 2: %v", err)
	}

	clipAID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(root1, "Pelis", "Clip A.mp4"),
		title: "Clip A", kind: KindMovie, matchStatus: "none",
	})
	clipBID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(root1, "Pelis", "Sub", "Clip B.mp4"),
		title: "Clip B", kind: KindMovie, matchStatus: "none",
	})
	clipCID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(root1, "Pelis", "Sub", "Deep", "Clip C.mp4"),
		title: "Clip C", kind: KindMovie, matchStatus: "none",
	})
	insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(root1, "Suelto.mp4"),
		title: "Suelto", kind: KindMovie, matchStatus: "none",
	})
	insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(root1, "Otro.mp4"),
		title: "Otro", kind: KindMovie, matchStatus: "none",
	})
	clipDID := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: filepath.Join(root2, "Otra", "Clip D.mp4"),
		title: "Clip D", kind: KindMovie, matchStatus: "none",
	})

	status, body := libraryHTTPGet(t, clipBrowseURL(ts, libraryID, ""), ts.token)
	if status != http.StatusOK {
		t.Fatalf("raíces: código = %d, cuerpo = %s", status, body)
	}
	var roots clipRootsResponse
	if err := json.Unmarshal(body, &roots); err != nil {
		t.Fatalf("raíces: %v", err)
	}
	if len(roots.Roots) != 2 {
		t.Fatalf("raíces = %+v", roots)
	}
	if roots.Roots[0].Name == roots.Roots[1].Name {
		t.Fatalf("las raíces con el mismo nombre deben desambiguarse: %+v", roots.Roots)
	}
	if strings.Contains(roots.Roots[0].Name, root1) || strings.Contains(roots.Roots[1].Name, root2) {
		t.Fatalf("las raíces no deben exponer rutas absolutas: %+v", roots.Roots)
	}
	if roots.Roots[0].ItemCount+roots.Roots[1].ItemCount != 6 {
		t.Fatalf("recuentos de raíces inesperados: %+v", roots.Roots)
	}
	covers := map[int64]bool{}
	for _, root := range roots.Roots {
		if root.CoverItemID != nil {
			covers[*root.CoverItemID] = true
		}
	}
	if !covers[clipAID] || !covers[clipDID] {
		t.Fatalf("portadas de raíz inesperadas: %+v", roots.Roots)
	}

	rootURL := func(path string) string {
		values := url.Values{}
		values.Set("root", itoa(root1ID))
		if path != "" {
			values.Set("path", path)
		}
		return clipBrowseURL(ts, libraryID, values.Encode())
	}

	status, body = libraryHTTPGet(t, rootURL(""), ts.token)
	if status != http.StatusOK {
		t.Fatalf("raíz: código = %d, cuerpo = %s", status, body)
	}
	if strings.Contains(string(body), root1) || strings.Contains(string(body), root2) {
		t.Fatalf("la respuesta contiene una ruta absoluta: %s", body)
	}
	var root clipBrowseResponse
	if err := json.Unmarshal(body, &root); err != nil {
		t.Fatalf("raíz: %v", err)
	}
	if root.Path != "" || root.Parent != nil || len(root.Items) != 2 ||
		len(root.Folders) != 1 || root.Folders[0].Name != "Pelis" ||
		root.Folders[0].ItemCount != 3 || root.Total != 2 ||
		root.Folders[0].CoverItemID == nil || *root.Folders[0].CoverItemID != clipAID {
		t.Fatalf("raíz inesperada: %+v", root)
	}
	if root.Items[0].LibraryKind != "clips" || root.Items[0].Path != "Otro.mp4" {
		t.Fatalf("primer ítem inesperado: %+v", root.Items[0])
	}

	status, body = libraryHTTPGet(t, rootURL("Pelis"), ts.token)
	if status != http.StatusOK {
		t.Fatalf("Pelis: código = %d, cuerpo = %s", status, body)
	}
	var pelis clipBrowseResponse
	if err := json.Unmarshal(body, &pelis); err != nil {
		t.Fatalf("Pelis: %v", err)
	}
	if pelis.Path != "Pelis" || pelis.Parent == nil || *pelis.Parent != "" ||
		len(pelis.Folders) != 1 || pelis.Folders[0].Path != "Pelis/Sub" ||
		pelis.Folders[0].ItemCount != 2 || pelis.Items[0].Path != "Pelis/Clip A.mp4" ||
		pelis.Folders[0].CoverItemID == nil || *pelis.Folders[0].CoverItemID != clipBID {
		t.Fatalf("Pelis inesperada: %+v", pelis)
	}
	if len(pelis.Crumbs) != 2 || pelis.Crumbs[1].Path != "Pelis" {
		t.Fatalf("migas inesperadas: %+v", pelis.Crumbs)
	}

	status, body = libraryHTTPGet(t, rootURL("Pelis/Sub"), ts.token)
	if status != http.StatusOK {
		t.Fatalf("Sub: código = %d, cuerpo = %s", status, body)
	}
	var sub clipBrowseResponse
	if err := json.Unmarshal(body, &sub); err != nil {
		t.Fatalf("Sub: %v", err)
	}
	if sub.Parent == nil || *sub.Parent != "Pelis" || sub.Folders[0].Path != "Pelis/Sub/Deep" ||
		sub.Folders[0].ItemCount != 1 || sub.Items[0].Path != "Pelis/Sub/Clip B.mp4" ||
		sub.Folders[0].CoverItemID == nil || *sub.Folders[0].CoverItemID != clipCID {
		t.Fatalf("Sub inesperada: %+v", sub)
	}

	status, body = libraryHTTPGet(t, rootURL("Pelis/Sub/Deep"), ts.token)
	if status != http.StatusOK {
		t.Fatalf("Deep: código = %d, cuerpo = %s", status, body)
	}
	var deep clipBrowseResponse
	if err := json.Unmarshal(body, &deep); err != nil {
		t.Fatalf("Deep: %v", err)
	}
	if len(deep.Folders) != 0 || len(deep.Items) != 1 || deep.Items[0].Path != "Pelis/Sub/Deep/Clip C.mp4" {
		t.Fatalf("Deep inesperada: %+v", deep)
	}
	if deep.Items[0].ID != clipCID {
		t.Fatalf("clip de Deep inesperado: %+v", deep.Items[0])
	}

	values := url.Values{"root": {itoa(root1ID)}, "limit": {"1"}, "offset": {"1"}}
	status, body = libraryHTTPGet(t, clipBrowseURL(ts, libraryID, values.Encode()), ts.token)
	if status != http.StatusOK {
		t.Fatalf("paginación: código = %d, cuerpo = %s", status, body)
	}
	var page clipBrowseResponse
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatalf("paginación: %v", err)
	}
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].Path != "Suelto.mp4" {
		t.Fatalf("paginación inesperada: %+v", page)
	}

	if status, body := libraryHTTPGet(t, clipBrowseURL(ts, libraryID, ""), ""); status != http.StatusUnauthorized {
		t.Fatalf("sin sesión: código = %d, cuerpo = %s", status, body)
	}

	videoID := createLibrary(t, ts.db, t.TempDir())
	if status, body := libraryHTTPGet(t, clipBrowseURL(ts, videoID, ""), ts.token); status != http.StatusNotFound {
		t.Fatalf("biblioteca de vídeo: código = %d, cuerpo = %s", status, body)
	}

	for _, rawPath := range []string{"../fuera", "/media/absoluta", "a//b", "a/"} {
		values := url.Values{"root": {itoa(root1ID)}, "path": {rawPath}}
		if status, body := libraryHTTPGet(t, clipBrowseURL(ts, libraryID, values.Encode()), ts.token); status != http.StatusBadRequest {
			t.Fatalf("path %q: código = %d, cuerpo = %s", rawPath, status, body)
		}
	}
	for _, rawRoot := range []string{"abc", "-1"} {
		values := url.Values{"root": {rawRoot}}
		if status, body := libraryHTTPGet(t, clipBrowseURL(ts, libraryID, values.Encode()), ts.token); status != http.StatusBadRequest {
			t.Fatalf("root %q: código = %d, cuerpo = %s", rawRoot, status, body)
		}
	}
	if status, body := libraryHTTPGet(t, clipBrowseURL(ts, 999999, ""), ts.token); status != http.StatusNotFound {
		t.Fatalf("biblioteca inexistente: código = %d, cuerpo = %s", status, body)
	}
}
