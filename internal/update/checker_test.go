package update

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type updateFakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *updateFakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *updateFakeClock) Advance(duration time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(duration)
	c.mu.Unlock()
}

func TestCheckerEligeLaEtiquetaMasAltaYCacha(t *testing.T) {
	var tagRequests, noteRequests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tags":
			atomic.AddInt32(&tagRequests, 1)
			if got := r.Header.Get("User-Agent"); got != "cinecat/0.13.1" {
				t.Errorf("User-Agent = %q", got)
			}
			fmt.Fprint(w, `[
				{"name":"v0.9.0"},
				{"name":"v0.14.0"},
				{"name":"v0.10.0"},
				{"name":"v0.13.1-beta"},
				{"name":"texto"}
			]`)
		case "/raw/v0.14.0/CHANGELOG.md":
			atomic.AddInt32(&noteRequests, 1)
			fmt.Fprint(w, "## [0.14.0] - 2026-10-26\n- Novedad\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	clock := &updateFakeClock{now: time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)}
	checker := NewChecker(CheckerOptions{
		Current:    "0.13.1",
		TagsURL:    server.URL + "/tags",
		RawBaseURL: server.URL + "/raw",
		Now:        clock.Now,
	})
	result, err := checker.Check(context.Background(), false)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.Latest != "v0.14.0" || !result.Available || !strings.Contains(result.Notes, "Novedad") {
		t.Fatalf("resultado inesperado: %+v", result)
	}

	if _, err := checker.Check(context.Background(), false); err != nil {
		t.Fatalf("Check cacheado: %v", err)
	}
	if tagRequests != 1 || noteRequests != 1 {
		t.Fatalf("la caché no evitó peticiones: %d, %d", tagRequests, noteRequests)
	}
	clock.Advance(time.Hour + time.Second)
	if _, err := checker.Check(context.Background(), false); err != nil {
		t.Fatalf("Check tras TTL: %v", err)
	}
	if tagRequests != 2 {
		t.Fatalf("el TTL no invalidó la caché: %d", tagRequests)
	}
	if _, err := checker.Check(context.Background(), true); err != nil {
		t.Fatalf("Check forzado: %v", err)
	}
	if tagRequests != 3 {
		t.Fatalf("la comprobación forzada reutilizó la caché: %d", tagRequests)
	}
}

func TestCheckerDevYErrores(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tags":
			fmt.Fprint(w, `[{"name":"v1.0.0"}]`)
		case "/raw/v1.0.0/CHANGELOG.md":
			http.Error(w, "caída", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	checker := NewChecker(CheckerOptions{
		Current: "dev", TagsURL: server.URL + "/tags", RawBaseURL: server.URL + "/raw",
	})
	result, err := checker.Check(context.Background(), true)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if result.Available || result.Latest != "v1.0.0" || result.Notes != "" {
		t.Fatalf("resultado dev inesperado: %+v", result)
	}

	broken := NewChecker(CheckerOptions{
		Current: "0.1.0", TagsURL: "http://127.0.0.1:1/tags", RawBaseURL: "http://127.0.0.1:1/raw",
	})
	if _, err := broken.Check(context.Background(), true); err == nil {
		t.Fatal("se esperaba un error de red")
	}

	large := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, strings.Repeat("x", maxTagsBytes+2))
	}))
	defer large.Close()
	if _, err := NewChecker(CheckerOptions{
		Current: "0.1.0", TagsURL: large.URL,
	}).Check(context.Background(), true); err == nil {
		t.Fatal("se esperaba error por cuerpo demasiado grande")
	}
}
