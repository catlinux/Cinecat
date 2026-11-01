package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLockoutPersisteYRegistraElBloqueo(t *testing.T) {
	svc, db, clock := newTestService(t,
		WithLockout(3, time.Minute), WithIPLockout(100, time.Minute))
	createUser(t, db, "ana", "Clave-segura-1", "user")
	other := New(db, WithClock(clock.Now),
		WithLockout(3, time.Minute), WithIPLockout(100, time.Minute))
	ctx := context.Background()
	ip := "198.51.100.60"

	var logs bytes.Buffer
	previousLog := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previousLog) })

	for i := 0; i < 2; i++ {
		if _, _, err := svc.LoginFrom(ctx, "ana", "mala", ip); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("fallo %d en el primer servicio: %v", i+1, err)
		}
	}
	if _, _, err := other.LoginFrom(ctx, "ana", "mala", ip); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("fallo en el segundo servicio: %v", err)
	}

	fresh := New(db, WithClock(clock.Now))
	if _, _, err := fresh.LoginFrom(ctx, "ana", "Clave-segura-1", ip); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("el bloqueo debe sobrevivir a un servicio nuevo: %v", err)
	}

	var kind string
	var name, address, since, until string
	if err := db.QueryRow(`
		SELECT kind, username, ip, locked_at, locked_until
		  FROM login_lockouts ORDER BY id`).Scan(
		&kind, &name, &address, &since, &until,
	); err != nil {
		t.Fatalf("leer bloqueo registrado: %v", err)
	}
	if kind != "user" || name != "ana" || address != ip ||
		since == "" || until == "" {
		t.Fatalf("bloqueo registrado inesperado: kind=%q name=%q ip=%q since=%q until=%q",
			kind, name, address, since, until)
	}
	if !strings.Contains(logs.String(), "auth: bloqueo de login") {
		t.Fatalf("falta la línea de log de bloqueo: %s", logs.String())
	}
	if strings.Contains(logs.String(), "Clave-segura-1") || strings.Contains(logs.String(), "mala") {
		t.Fatalf("el log no debe incluir contraseñas: %s", logs.String())
	}

	clock.Advance(time.Minute + time.Second)
	if _, _, err := svc.LoginFrom(ctx, "ana", "Clave-segura-1", ip); err != nil {
		t.Fatalf("tras caducar el bloqueo debe poderse entrar: %v", err)
	}
	var attempts int
	if err := db.QueryRow(`
		SELECT count(*) FROM login_attempts
		 WHERE kind = 'user' AND "key" = ?`, loginPairKey("ana", ip)).Scan(&attempts); err != nil {
		t.Fatalf("contar intentos tras el login correcto: %v", err)
	}
	if attempts != 0 {
		t.Fatalf("el login correcto debe limpiar los intentos, quedan %d", attempts)
	}
}

func TestSecurityEventsEndpoint(t *testing.T) {
	svc, db, _ := newTestService(t,
		WithLockout(2, time.Minute), WithIPLockout(100, time.Minute))
	createUser(t, db, "ana", "Clave-segura-1", "user")
	createUser(t, db, "root", "Clave-root-1", "admin")
	createUser(t, db, "bea", "Clave-bea-1", "user")
	ctx := context.Background()
	ip := "198.51.100.61"
	for i := 0; i < 2; i++ {
		if _, _, err := svc.LoginFrom(ctx, "ana", "mala", ip); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("fallo %d: %v", i+1, err)
		}
	}

	adminToken, _, err := svc.Login(ctx, "root", "Clave-root-1")
	if err != nil {
		t.Fatalf("login admin: %v", err)
	}
	userToken, _, err := svc.Login(ctx, "bea", "Clave-bea-1")
	if err != nil {
		t.Fatalf("login usuario: %v", err)
	}

	mux := http.NewServeMux()
	svc.RegisterSecurityRoutes(mux)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	get := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, server.URL+"/api/v1/system/security-events", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	if rec := get(""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("sin sesión: código = %d", rec.Code)
	}
	if rec := get(userToken); rec.Code != http.StatusForbidden {
		t.Fatalf("usuario normal: código = %d, cuerpo = %s", rec.Code, rec.Body.String())
	}
	rec := get(adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin: código = %d, cuerpo = %s", rec.Code, rec.Body.String())
	}
	var response securityEventsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("respuesta: %v (%s)", err, rec.Body.String())
	}
	if response.Total7d < 1 || len(response.Lockouts) == 0 {
		t.Fatalf("respuesta sin bloqueos: %+v", response)
	}
	first := response.Lockouts[0]
	if first.Kind != "user" || first.Username == nil || *first.Username != "ana" ||
		first.IP == nil || *first.IP != ip {
		t.Fatalf("bloqueo inesperado: %+v", first)
	}
}

func TestLockoutEnMemoriaSinBase(t *testing.T) {
	clock := newFakeClock()
	svc := New(nil, WithClock(clock.Now),
		WithLockout(2, time.Minute), WithIPLockout(100, time.Minute))
	ctx := context.Background()
	if err := svc.registerFailureFor(ctx, "ana", "pareja", "198.51.100.62", clock.Now()); err != nil {
		t.Fatalf("primer fallo en memoria: %v", err)
	}
	if locked, err := svc.isLockedFor(ctx, "pareja", "198.51.100.62", clock.Now()); err != nil || locked {
		t.Fatalf("no debería estar bloqueado: locked=%v err=%v", locked, err)
	}
	if err := svc.registerFailureFor(ctx, "ana", "pareja", "198.51.100.62", clock.Now()); err != nil {
		t.Fatalf("segundo fallo en memoria: %v", err)
	}
	if locked, err := svc.isLockedFor(ctx, "pareja", "198.51.100.62", clock.Now()); err != nil || !locked {
		t.Fatalf("debería estar bloqueado: locked=%v err=%v", locked, err)
	}
	if err := svc.resetFailures(ctx, "pareja"); err != nil {
		t.Fatalf("limpiar fallos en memoria: %v", err)
	}
	if locked, _ := svc.isLockedFor(ctx, "pareja", "198.51.100.62", clock.Now()); locked {
		t.Fatal("el reinicio de fallos debe desbloquear la pareja")
	}
}
