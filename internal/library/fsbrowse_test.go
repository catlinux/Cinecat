package library

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func fsURL(ts *libraryTestServer, path string) string {
	if path == "" {
		return ts.URL + "/api/v1/fs"
	}
	return ts.URL + "/api/v1/fs?path=" + url.QueryEscape(path)
}

func getFSRoots(t *testing.T, ts *libraryTestServer) fsRootsResponse {
	t.Helper()
	status, body := libraryHTTPGet(t, fsURL(ts, ""), ts.token)
	if status != http.StatusOK {
		t.Fatalf("raíces: código = %d, cuerpo = %s", status, body)
	}
	var response fsRootsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("raíces: %v", err)
	}
	return response
}

func TestFSRaicesYListadoConfinado(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	root := t.TempDir()
	makeDirs(t, root, "zeta", "Alfa", "sub")
	if err := os.MkdirAll(filepath.Join(root, ".oculta"), 0o755); err != nil {
		t.Fatalf("oculta: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".cinecat-papelera"), 0o755); err != nil {
		t.Fatalf("papelera: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "archivo.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("archivo: %v", err)
	}
	ts.api.WithMediaRoots([]string{root})

	roots := getFSRoots(t, ts)
	if len(roots.Roots) != 1 || roots.Roots[0].Path != root {
		t.Fatalf("raíces inesperadas: %+v", roots.Roots)
	}

	status, body := libraryHTTPGet(t, fsURL(ts, root), ts.token)
	if status != http.StatusOK {
		t.Fatalf("raíz: código = %d, cuerpo = %s", status, body)
	}
	var response fsBrowseResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("raíz: %v", err)
	}
	if response.Parent != nil || response.Path != root || response.Truncated {
		t.Fatalf("respuesta de raíz inesperada: %+v", response)
	}
	names := make([]string, 0, len(response.Entries))
	for _, entry := range response.Entries {
		names = append(names, entry.Name)
		if entry.Path != filepath.Join(root, entry.Name) {
			t.Fatalf("ruta de entrada inesperada: %+v", entry)
		}
	}
	if strings.Join(names, ",") != "Alfa,sub,zeta" {
		t.Fatalf("entradas = %v", names)
	}

	status, body = libraryHTTPGet(t, fsURL(ts, filepath.Join(root, "sub")), ts.token)
	if status != http.StatusOK {
		t.Fatalf("subcarpeta: código = %d, cuerpo = %s", status, body)
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("subcarpeta: %v", err)
	}
	if response.Parent == nil || *response.Parent != root || len(response.Entries) != 0 {
		t.Fatalf("respuesta de subcarpeta inesperada: %+v", response)
	}
}

func TestFSErroresDeConfinamiento(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	root := t.TempDir()
	outside := t.TempDir()
	makeDirs(t, root, "sub")
	file := filepath.Join(root, "archivo.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("archivo: %v", err)
	}
	ts.api.WithMediaRoots([]string{root})

	cases := []struct {
		name string
		path string
		want int
	}{
		{"relativa", "sub", http.StatusBadRequest},
		{"con puntos", root + string(os.PathSeparator) + "sub" + string(os.PathSeparator) + "..", http.StatusBadRequest},
		{"fuera", outside, http.StatusForbidden},
		{"inexistente", filepath.Join(root, "no-existe"), http.StatusNotFound},
		{"archivo", file, http.StatusBadRequest},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			status, body := libraryHTTPGet(t, fsURL(ts, test.path), ts.token)
			if status != test.want {
				t.Fatalf("código = %d, se esperaba %d; cuerpo = %s", status, test.want, body)
			}
		})
	}
}

func TestFSNoSigueEnlacesSimbolicos(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "enlace")); err != nil {
		t.Skipf("el sistema no permite crear enlaces simbólicos: %v", err)
	}
	ts.api.WithMediaRoots([]string{root})

	status, body := libraryHTTPGet(t, fsURL(ts, root), ts.token)
	if status != http.StatusOK {
		t.Fatalf("raíz: código = %d, cuerpo = %s", status, body)
	}
	var response fsBrowseResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("raíz: %v", err)
	}
	for _, entry := range response.Entries {
		if entry.Name == "enlace" {
			t.Fatalf("no debe listarse el enlace: %+v", entry)
		}
	}
	status, body = libraryHTTPGet(t, fsURL(ts, filepath.Join(root, "enlace")), ts.token)
	if status != http.StatusForbidden {
		t.Fatalf("abrir enlace externo: código = %d, cuerpo = %s", status, body)
	}
}

func TestFSTruncado(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	root := t.TempDir()
	for index := 0; index < maxFSEntries+1; index++ {
		name := filepath.Join(root, "carpeta-"+strconv.Itoa(index))
		if err := os.Mkdir(name, 0o755); err != nil {
			t.Fatalf("crear carpeta %d: %v", index, err)
		}
	}
	ts.api.WithMediaRoots([]string{root})

	status, body := libraryHTTPGet(t, fsURL(ts, root), ts.token)
	if status != http.StatusOK {
		t.Fatalf("truncado: código = %d, cuerpo = %s", status, body)
	}
	var response fsBrowseResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("truncado: %v", err)
	}
	if !response.Truncated || len(response.Entries) != maxFSEntries {
		t.Fatalf("truncado = %+v, entradas = %d", response.Truncated, len(response.Entries))
	}
}

func TestFSPermisos(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	root := t.TempDir()
	ts.api.WithMediaRoots([]string{root})

	status, body := libraryHTTPGet(t, fsURL(ts, ""), "")
	if status != http.StatusUnauthorized {
		t.Fatalf("sin sesión: código = %d, cuerpo = %s", status, body)
	}
	userToken := nonAdminToken(t, ts)
	status, body = libraryHTTPGet(t, fsURL(ts, ""), userToken)
	if status != http.StatusForbidden {
		t.Fatalf("sin admin: código = %d, cuerpo = %s", status, body)
	}
	status, body = libraryJSONDo(t, http.MethodPost, ts.URL+"/api/v1/libraries", "",
		map[string]any{"name": "X", "paths": []string{root}})
	if status != http.StatusUnauthorized {
		t.Fatalf("crear sin sesión: código = %d, cuerpo = %s", status, body)
	}
}
