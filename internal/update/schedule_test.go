package update

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunBackgroundRespetaElAjuste(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tags" {
			atomic.AddInt32(&requests, 1)
			fmt.Fprint(w, `[{"name":"v0.14.0"}]`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	checker := NewChecker(CheckerOptions{
		Current: "0.13.1", TagsURL: server.URL + "/tags", RawBaseURL: server.URL + "/raw",
	})
	var enabled atomic.Bool
	changes := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go RunBackground(ctx, checker, enabled.Load, changes, 20*time.Millisecond, 40*time.Millisecond)

	time.Sleep(80 * time.Millisecond)
	if atomic.LoadInt32(&requests) != 0 {
		t.Fatalf("con el ajuste apagado hubo peticiones: %d", requests)
	}
	enabled.Store(true)
	changes <- struct{}{}
	deadline := time.Now().Add(time.Second)
	for atomic.LoadInt32(&requests) == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if atomic.LoadInt32(&requests) == 0 {
		t.Fatal("al activar el ajuste no se comprobó GitHub")
	}
	enabled.Store(false)
	changes <- struct{}{}
	time.Sleep(80 * time.Millisecond)
	before := atomic.LoadInt32(&requests)
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&requests) != before {
		t.Fatal("tras apagar el ajuste siguieron las comprobaciones automáticas")
	}
}
