package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/catlinux/cinecat/internal/store"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newTestService(t *testing.T, opts ...Option) (*Service, *sql.DB, *fakeClock) {
	t.Helper()
	ctx := context.Background()

	st, err := store.Open(ctx, filepath.Join(t.TempDir(), "cinecat.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Errorf("store.Close: %v", err)
		}
	})
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	clock := newFakeClock()
	svc := New(st.DB(), append([]Option{WithClock(clock.Now)}, opts...)...)
	return svc, st.DB(), clock
}

func createUser(t *testing.T, db *sql.DB, username, password, role string) int64 {
	t.Helper()

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	res, err := db.Exec(
		`INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`,
		username, hash, role,
	)
	if err != nil {
		t.Fatalf("crear usuario: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("id de usuario: %v", err)
	}
	return id
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("Clave-segura-1")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Fatalf("hash inesperado: %q", hash)
	}
	if strings.Contains(hash, "Clave-segura-1") {
		t.Fatal("el hash no debe contener la contraseña")
	}

	ok, err := VerifyPassword(hash, "Clave-segura-1")
	if err != nil || !ok {
		t.Fatalf("VerifyPassword con la contraseña correcta = %v, %v", ok, err)
	}
	ok, err = VerifyPassword(hash, "otra")
	if err != nil || ok {
		t.Fatalf("VerifyPassword con otra contraseña = %v, %v", ok, err)
	}

	other, err := HashPassword("Clave-segura-1")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if other == hash {
		t.Fatal("dos hashes de la misma contraseña no deben coincidir (sal distinta)")
	}
}

func TestLoginCorrecto(t *testing.T) {
	svc, db, clock := newTestService(t)
	userID := createUser(t, db, "ana", "Clave-segura-1", "admin")

	ctx := context.Background()
	token, expiresAt, err := svc.Login(ctx, "ana", "Clave-segura-1")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token == "" {
		t.Fatal("el token no puede estar vacío")
	}
	if got := expiresAt.Sub(clock.Now()); got != DefaultSessionTTL {
		t.Fatalf("caducidad a %v, se esperaba %v", got, DefaultSessionTTL)
	}

	user, err := svc.Authenticate(ctx, token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if user.ID != userID || user.Username != "ana" || user.Role != "admin" {
		t.Fatalf("usuario inesperado: %+v", user)
	}

	var stored string
	if err := db.QueryRow(`SELECT token_hash FROM sessions`).Scan(&stored); err != nil {
		t.Fatalf("leer token_hash: %v", err)
	}
	if stored == token || strings.Contains(stored, token) {
		t.Fatal("el token no debe guardarse en claro")
	}
	if stored != hashToken(token) {
		t.Fatal("token_hash no coincide con el hash del token")
	}
}

func TestLoginCredencialesIncorrectas(t *testing.T) {
	svc, db, _ := newTestService(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")

	ctx := context.Background()
	_, _, errWrongPass := svc.Login(ctx, "ana", "incorrecta")
	_, _, errNoUser := svc.Login(ctx, "nadie", "incorrecta")

	if !errors.Is(errWrongPass, ErrInvalidCredentials) {
		t.Fatalf("se esperaba ErrInvalidCredentials, hay %v", errWrongPass)
	}
	if !errors.Is(errNoUser, ErrInvalidCredentials) {
		t.Fatalf("se esperaba ErrInvalidCredentials, hay %v", errNoUser)
	}
	if errWrongPass.Error() != errNoUser.Error() {
		t.Fatal("el error no debe distinguir si el usuario existe")
	}

	var sessions int
	if err := db.QueryRow(`SELECT count(*) FROM sessions`).Scan(&sessions); err != nil {
		t.Fatalf("contar sesiones: %v", err)
	}
	if sessions != 0 {
		t.Fatalf("se crearon %d sesiones con credenciales incorrectas", sessions)
	}
}

func TestBloqueoTrasIntentosFallidos(t *testing.T) {
	svc, db, clock := newTestService(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")

	ctx := context.Background()
	for i := 0; i < DefaultMaxAttempts; i++ {
		if _, _, err := svc.Login(ctx, "ana", "incorrecta"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("intento %d: se esperaba ErrInvalidCredentials, hay %v", i+1, err)
		}
	}

	if _, _, err := svc.Login(ctx, "ana", "Clave-segura-1"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("con la contraseña correcta se esperaba ErrAccountLocked, hay %v", err)
	}

	clock.Advance(DefaultLockDuration + time.Second)

	if _, _, err := svc.Login(ctx, "ana", "Clave-segura-1"); err != nil {
		t.Fatalf("tras el bloqueo el login correcto debe funcionar: %v", err)
	}
}

func TestBloqueoTambienParaUsuarioInexistente(t *testing.T) {
	svc, _, _ := newTestService(t, WithLockout(2, time.Minute))
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if _, _, err := svc.Login(ctx, "nadie", "incorrecta"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("intento %d: %v", i+1, err)
		}
	}
	if _, _, err := svc.Login(ctx, "nadie", "incorrecta"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("se esperaba ErrAccountLocked, hay %v", err)
	}
}

func TestLogoutInvalidaToken(t *testing.T) {
	svc, db, _ := newTestService(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")

	ctx := context.Background()
	token, _, err := svc.Login(ctx, "ana", "Clave-segura-1")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	if err := svc.Logout(ctx, token); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if _, err := svc.Authenticate(ctx, token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("tras logout se esperaba ErrInvalidSession, hay %v", err)
	}
	// Logout es idempotente.
	if err := svc.Logout(ctx, token); err != nil {
		t.Fatalf("segundo Logout: %v", err)
	}

	var sessions int
	if err := db.QueryRow(`SELECT count(*) FROM sessions`).Scan(&sessions); err != nil {
		t.Fatalf("contar sesiones: %v", err)
	}
	if sessions != 0 {
		t.Fatalf("quedan %d sesiones tras el logout", sessions)
	}
}

func TestTokenExpiradoSeRechaza(t *testing.T) {
	svc, db, clock := newTestService(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")

	ctx := context.Background()
	token, _, err := svc.Login(ctx, "ana", "Clave-segura-1")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	if _, err := db.Exec(`UPDATE sessions SET expires_at = ?`,
		formatTime(clock.Now().Add(-time.Minute))); err != nil {
		t.Fatalf("caducar sesión: %v", err)
	}

	if _, err := svc.Authenticate(ctx, token); !errors.Is(err, ErrSessionExpired) {
		t.Fatalf("se esperaba ErrSessionExpired, hay %v", err)
	}

	var sessions int
	if err := db.QueryRow(`SELECT count(*) FROM sessions`).Scan(&sessions); err != nil {
		t.Fatalf("contar sesiones: %v", err)
	}
	if sessions != 0 {
		t.Fatalf("la sesión caducada debería haberse limpiado, quedan %d", sessions)
	}
}

func TestRenovacionDeslizante(t *testing.T) {
	svc, db, clock := newTestService(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")

	ctx := context.Background()
	token, _, err := svc.Login(ctx, "ana", "Clave-segura-1")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	clock.Advance(DefaultSessionTTL - time.Hour)
	if _, err := svc.Authenticate(ctx, token); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}

	var raw string
	if err := db.QueryRow(`SELECT expires_at FROM sessions`).Scan(&raw); err != nil {
		t.Fatalf("leer expires_at: %v", err)
	}
	expiresAt, err := parseTime(raw)
	if err != nil {
		t.Fatalf("parseTime: %v", err)
	}
	want := clock.Now().Add(DefaultSessionTTL).Truncate(time.Millisecond)
	if !expiresAt.Equal(want) {
		t.Fatalf("caducidad renovada = %v, se esperaba %v", expiresAt, want)
	}
}

func TestLoginHandler(t *testing.T) {
	svc, db, _ := newTestService(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")

	body := `{"username":"ana","password":"Clave-segura-1"}`
	rec := httptest.NewRecorder()
	svc.LoginHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body)))

	if rec.Code != http.StatusOK {
		t.Fatalf("código = %d, cuerpo = %s", rec.Code, rec.Body.String())
	}
	var resp loginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("respuesta no JSON: %v", err)
	}
	if resp.Token == "" || resp.ExpiresAt.IsZero() {
		t.Fatalf("respuesta incompleta: %+v", resp)
	}

	rec = httptest.NewRecorder()
	badBody := `{"username":"ana","password":"incorrecta"}`
	svc.LoginHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(badBody)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("credenciales incorrectas: código = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	svc.LoginHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("no es json")))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("cuerpo no válido: código = %d", rec.Code)
	}
}

func TestLoginHandlerBloqueo(t *testing.T) {
	svc, db, _ := newTestService(t, WithLockout(2, time.Minute))
	createUser(t, db, "ana", "Clave-segura-1", "user")

	req := func(body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		svc.LoginHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body)))
		return rec
	}

	req(`{"username":"ana","password":"mala"}`)
	req(`{"username":"ana","password":"mala"}`)
	rec := req(`{"username":"ana","password":"Clave-segura-1"}`)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("bloqueado: código = %d", rec.Code)
	}
}

func TestLogoutHandler(t *testing.T) {
	svc, db, _ := newTestService(t)
	createUser(t, db, "ana", "Clave-segura-1", "user")

	ctx := context.Background()
	token, _, err := svc.Login(ctx, "ana", "Clave-segura-1")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	svc.LogoutHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("código = %d, cuerpo = %s", rec.Code, rec.Body.String())
	}
	if _, err := svc.Authenticate(ctx, token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("tras logout se esperaba ErrInvalidSession, hay %v", err)
	}

	rec = httptest.NewRecorder()
	svc.LogoutHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("sin token: código = %d", rec.Code)
	}
}

func TestChangePassword(t *testing.T) {
	svc, db, _ := newTestService(t)
	userID := createUser(t, db, "ana", "Clave-vieja-1", "user")
	ctx := context.Background()

	if err := svc.ChangePassword(ctx, userID, "Clave-vieja-1", "Clave-nueva-1"); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}

	var hash string
	if err := db.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, userID).Scan(&hash); err != nil {
		t.Fatalf("leer hash: %v", err)
	}
	if ok, err := VerifyPassword(hash, "Clave-nueva-1"); err != nil || !ok {
		t.Fatalf("la contraseña nueva no valida: %v, %v", ok, err)
	}
	if ok, _ := VerifyPassword(hash, "Clave-vieja-1"); ok {
		t.Fatal("la contraseña antigua no debe seguir valiendo")
	}
}

func TestChangePasswordActualIncorrecta(t *testing.T) {
	svc, db, _ := newTestService(t)
	userID := createUser(t, db, "ana", "Clave-vieja-1", "user")
	ctx := context.Background()

	var before string
	if err := db.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, userID).Scan(&before); err != nil {
		t.Fatalf("leer hash: %v", err)
	}

	err := svc.ChangePassword(ctx, userID, "incorrecta", "Clave-nueva-1")
	if !errors.Is(err, ErrInvalidCurrentPassword) {
		t.Fatalf("error = %v, se esperaba ErrInvalidCurrentPassword", err)
	}

	var after string
	if err := db.QueryRow(`SELECT password_hash FROM users WHERE id = ?`, userID).Scan(&after); err != nil {
		t.Fatalf("leer hash: %v", err)
	}
	if after != before {
		t.Fatal("el hash no debe cambiar con una contraseña actual incorrecta")
	}
}

func TestChangePasswordDemasiadoCorta(t *testing.T) {
	svc, db, _ := newTestService(t)
	userID := createUser(t, db, "ana", "Clave-vieja-1", "user")

	err := svc.ChangePassword(context.Background(), userID, "Clave-vieja-1", "corta")
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("error = %v, se esperaba ErrPasswordTooShort", err)
	}
}

func TestChangePasswordUsuarioInexistente(t *testing.T) {
	svc, _, _ := newTestService(t)

	err := svc.ChangePassword(context.Background(), 999999, "lo-que-sea", "Clave-nueva-1")
	if !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("error = %v, se esperaba ErrInvalidSession", err)
	}
}
