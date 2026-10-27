package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func TestBloqueoPorParejaUsuarioIPNoAfectaAOtrasIPs(t *testing.T) {
	svc, db, _ := newTestService(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")
	createUser(t, db, "bea", "Clave-segura-2", "user")
	ctx := context.Background()

	for i := 1; i <= DefaultMaxAttempts; i++ {
		if _, _, err := svc.LoginFrom(ctx, "ana", "mala", "198.51.100.10"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("intento %d desde IP atacante: %v", i, err)
		}
	}
	if _, _, err := svc.LoginFrom(ctx, "ana", "Clave-segura-1", "198.51.100.10"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("la pareja bloqueada debe rechazar el login correcto: %v", err)
	}
	if _, _, err := svc.LoginFrom(ctx, "ana", "Clave-segura-1", "198.51.100.11"); err != nil {
		t.Fatalf("otra IP no debe quedar afectada: %v", err)
	}
	if _, _, err := svc.LoginFrom(ctx, "ana", "Clave-segura-1", "198.51.100.10"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("el bloqueo de la pareja sigue activo: %v", err)
	}
	if _, _, err := svc.LoginFrom(ctx, "bea", "Clave-segura-2", "198.51.100.10"); err != nil {
		t.Fatalf("otro usuario desde la misma IP no debe quedar bloqueado por la pareja: %v", err)
	}
}

func TestLimitePorIPBloqueaTodosLosUsuarios(t *testing.T) {
	svc, db, _ := newTestService(t, WithIPLockout(3, time.Minute))
	createUser(t, db, "ana", "Clave-segura-1", "user")
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if _, _, err := svc.LoginFrom(ctx, fmt.Sprintf("inventado%d", i), "mala", "203.0.113.50"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("intento %d: %v", i, err)
		}
	}
	if _, _, err := svc.LoginFrom(ctx, "ana", "Clave-segura-1", "203.0.113.50"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("la IP bloqueada debe rechazar cualquier usuario: %v", err)
	}
	if _, _, err := svc.LoginFrom(ctx, "ana", "Clave-segura-1", "203.0.113.51"); err != nil {
		t.Fatalf("otra IP no debe quedar afectada: %v", err)
	}
}

func TestLoginCorrectoLimpiaElContadorDeLaPareja(t *testing.T) {
	svc, db, _ := newTestService(t, WithLockout(3, time.Minute), WithIPLockout(100, time.Minute))
	createUser(t, db, "ana", "Clave-segura-1", "user")
	ctx := context.Background()

	for round := 0; round < 2; round++ {
		for i := 0; i < 2; i++ {
			if _, _, err := svc.LoginFrom(ctx, "ana", "mala", "198.51.100.20"); !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("ronda %d, fallo %d: %v", round, i, err)
			}
		}
		if _, _, err := svc.LoginFrom(ctx, "ana", "Clave-segura-1", "198.51.100.20"); err != nil {
			t.Fatalf("el login correcto debe limpiar la pareja: %v", err)
		}
	}
}

func TestPurgaYtopeDeMemoria(t *testing.T) {
	svc, _, clock := newTestService(t)
	now := clock.Now()

	svc.mu.Lock()
	svc.attempts["antigua"] = &attempt{count: 1, last: now.Add(-DefaultLockDuration - time.Second)}
	svc.attempts["futura"] = &attempt{last: now, lockedUntil: now.Add(time.Minute)}
	svc.mu.Unlock()

	svc.prune(now)
	svc.mu.Lock()
	if _, exists := svc.attempts["antigua"]; exists {
		t.Error("la entrada caducada debe purgarse")
	}
	if _, exists := svc.attempts["futura"]; !exists {
		t.Error("la entrada bloqueada todavía debe conservarse")
	}
	svc.mu.Unlock()

	svc.mu.Lock()
	for i := 0; i < maxAttemptEntries+20; i++ {
		svc.attempts[fmt.Sprintf("usuario-%d", i)] = &attempt{last: now.Add(time.Duration(i) * time.Millisecond)}
	}
	svc.trimLocked()
	size := len(svc.attempts) + len(svc.ipAttempts)
	svc.mu.Unlock()
	if size > maxAttemptEntries {
		t.Fatalf("el mapa tiene %d entradas, máximo %d", size, maxAttemptEntries)
	}
}

func TestLoginHandlerRetryAfterYConfianzaEnXForwardedFor(t *testing.T) {
	svc, db, _ := newTestService(t, WithLockout(2, time.Minute), WithTrustedProxies([]netip.Prefix{
		netip.MustParsePrefix("127.0.0.0/8"),
	}))
	createUser(t, db, "ana", "Clave-segura-1", "user")

	login := func(ip string) *httptest.ResponseRecorder {
		body := strings.NewReader(`{"username":"ana","password":"mala"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
		req.RemoteAddr = "127.0.0.1:1234"
		req.Header.Set("X-Forwarded-For", ip)
		rec := httptest.NewRecorder()
		svc.LoginHandler().ServeHTTP(rec, req)
		return rec
	}
	login("198.51.100.30")
	login("198.51.100.30")
	blocked := httptest.NewRecorder()
	body := strings.NewReader(`{"username":"ana","password":"Clave-segura-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.30")
	svc.LoginHandler().ServeHTTP(blocked, req)
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("código = %d, se esperaba 429", blocked.Code)
	}
	if got := blocked.Header().Get("Retry-After"); got != "60" {
		t.Fatalf("Retry-After = %q, se esperaba 60", got)
	}
	if status := login("198.51.100.31"); status.Code != http.StatusUnauthorized {
		t.Fatalf("otra IP debe seguir pudiendo intentarlo, código = %d", status.Code)
	}
}

func TestLoginHandlerCincoFallosPorParejaYUsuarioIP(t *testing.T) {
	svc, db, _ := newTestService(t, WithTrustedProxies([]netip.Prefix{
		netip.MustParsePrefix("127.0.0.0/8"),
	}))
	createUser(t, db, "ana", "Clave-segura-1", "user")

	login := func(ip, username, password string) *httptest.ResponseRecorder {
		body := strings.NewReader(fmt.Sprintf(`{"username":%q,"password":%q}`, username, password))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
		req.RemoteAddr = "127.0.0.1:1234"
		req.Header.Set("X-Forwarded-For", ip)
		rec := httptest.NewRecorder()
		svc.LoginHandler().ServeHTTP(rec, req)
		return rec
	}

	for i := 0; i < DefaultMaxAttempts; i++ {
		if rec := login("198.51.100.40", "ana", "mala"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("fallo %d: código = %d", i+1, rec.Code)
		}
	}
	blocked := login("198.51.100.40", "ana", "Clave-segura-1")
	if blocked.Code != http.StatusTooManyRequests || blocked.Header().Get("Retry-After") == "" {
		t.Fatalf("bloqueo: código = %d, Retry-After = %q", blocked.Code, blocked.Header().Get("Retry-After"))
	}
	if rec := login("198.51.100.41", "ana", "Clave-segura-1"); rec.Code != http.StatusOK {
		t.Fatalf("otra IP debe entrar: código = %d, cuerpo = %s", rec.Code, rec.Body.String())
	}
}
