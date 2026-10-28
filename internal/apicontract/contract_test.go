package apicontract

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var (
	registeredRoutePattern  = regexp.MustCompile(`\b(?:Handle|HandleFunc)\("(GET|POST|PUT|PATCH|DELETE) (/api/[^"\s]*|/healthz)"`)
	documentedPathPattern   = regexp.MustCompile(`^  (/api/\S+|/healthz):$`)
	documentedMethodPattern = regexp.MustCompile(`^    (get|post|put|patch|delete):$`)
)

func TestRoutesMatchOpenAPI(t *testing.T) {
	registered := loadRegisteredRoutes(t)
	documented := loadDocumentedRoutes(t)
	if len(registered) == 0 {
		t.Fatal("no se encontró ninguna ruta registrada en cmd o internal")
	}
	if len(documented) == 0 {
		t.Fatal("no se encontró ninguna ruta documentada en api/openapi.yaml")
	}

	missing := difference(registered, documented)
	extra := difference(documented, registered)
	var problems []string
	if len(missing) > 0 {
		problems = append(problems, "rutas registradas sin documentar:\n  "+strings.Join(missing, "\n  "))
	}
	if len(extra) > 0 {
		problems = append(problems, "rutas documentadas que ya no existen:\n  "+strings.Join(extra, "\n  "))
	}
	if len(problems) > 0 {
		t.Fatal(strings.Join(problems, "\n\n"))
	}
}

func TestExtractRegisteredRoutes(t *testing.T) {
	sample := `package example

func register(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/items/{id}", handler)
	mux.HandleFunc("POST /api/v1/trash/purge", handler)
	s.Handle("PATCH /api/v1/users/{id}", handler)
	mux.Handle("DELETE /healthz", handler)
	mux.Handle("GET /otra/ruta", handler)
	mux.Handle("get /api/v1/minusculas", handler)
	mux.Handle("/api/v1/sin-metodo", handler)
}
`
	got := extractRegisteredRoutes(sample)
	want := map[string]bool{
		"GET /api/v1/items/{id}":   true,
		"POST /api/v1/trash/purge": true,
		"PATCH /api/v1/users/{id}": true,
		"DELETE /healthz":          true,
	}
	if len(got) != len(want) {
		t.Fatalf("rutas extraídas = %v, se esperaban %v", sortedKeys(got), sortedKeys(want))
	}
	for route := range want {
		if !got[route] {
			t.Fatalf("no se extrajo %q", route)
		}
	}
}

func TestParseDocumentedRoutes(t *testing.T) {
	sample := `openapi: 3.0.3
paths:
  /api/v1/items/{id}:
    get:
      summary: Ficha
    delete:
      summary: Papelera
  /healthz:
    get:
      summary: Salud
  /otra:
    delete:
      summary: Fuera del contrato
components:
  schemas:
    get:
      type: object
`
	got := parseDocumentedRoutes(sample)
	want := map[string]bool{
		"GET /api/v1/items/{id}":    true,
		"DELETE /api/v1/items/{id}": true,
		"GET /healthz":              true,
	}
	if len(got) != len(want) {
		t.Fatalf("rutas documentadas = %v, se esperaban %v", sortedKeys(got), sortedKeys(want))
	}
	for route := range want {
		if !got[route] {
			t.Fatalf("no se documentó %q", route)
		}
	}
}

func loadRegisteredRoutes(t *testing.T) map[string]bool {
	t.Helper()
	routes := make(map[string]bool)
	for _, root := range []string{
		filepath.Join("..", "..", "cmd"),
		filepath.Join("..", "..", "internal"),
	} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for route := range extractRegisteredRoutes(string(data)) {
				routes[route] = true
			}
			return nil
		})
		if err != nil {
			t.Fatalf("recorrer %s: %v", root, err)
		}
	}
	return routes
}

func loadDocumentedRoutes(t *testing.T) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "api", "openapi.yaml"))
	if err != nil {
		t.Fatalf("leer api/openapi.yaml: %v", err)
	}
	return parseDocumentedRoutes(string(data))
}

func extractRegisteredRoutes(text string) map[string]bool {
	routes := make(map[string]bool)
	for _, match := range registeredRoutePattern.FindAllStringSubmatch(text, -1) {
		routes[match[1]+" "+match[2]] = true
	}
	return routes
}

func parseDocumentedRoutes(text string) map[string]bool {
	routes := make(map[string]bool)
	inPaths := false
	currentPath := ""
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSuffix(raw, "\r")
		if line == "paths:" {
			inPaths = true
			currentPath = ""
			continue
		}
		if !inPaths {
			continue
		}
		if line != "" && !strings.HasPrefix(line, " ") {
			inPaths = false
			currentPath = ""
			continue
		}
		if strings.HasPrefix(line, "  /") && strings.HasSuffix(line, ":") {
			currentPath = ""
			if match := documentedPathPattern.FindStringSubmatch(line); match != nil {
				currentPath = match[1]
			}
			continue
		}
		if match := documentedMethodPattern.FindStringSubmatch(line); match != nil && currentPath != "" {
			routes[strings.ToUpper(match[1])+" "+currentPath] = true
		}
	}
	return routes
}

func difference(left, right map[string]bool) []string {
	result := make([]string, 0)
	for route := range left {
		if !right[route] {
			result = append(result, route)
		}
	}
	sort.Strings(result)
	return result
}

func sortedKeys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for route := range values {
		result = append(result, route)
	}
	sort.Strings(result)
	return result
}
