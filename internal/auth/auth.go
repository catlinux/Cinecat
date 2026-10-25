// Package auth implementa usuarios, contraseñas con argon2id y sesiones de
// vida corta para Cinecat.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
)

// Parámetros de argon2id (64 MiB, 3 pasadas, 2 hilos), acordes con las
// recomendaciones actuales para un servidor personal.
const (
	argonMemory  = 64 * 1024 // KiB
	argonTime    = 3
	argonThreads = 2
	argonKeyLen  = 32
	argonSaltLen = 16
)

const (
	// DefaultSessionTTL es la vida de una sesión recién creada.
	DefaultSessionTTL = 24 * time.Hour
	// DefaultMaxAttempts es el número de fallos seguidos que bloquea el login.
	DefaultMaxAttempts = 5
	// DefaultLockDuration es lo que dura el bloqueo temporal.
	DefaultLockDuration = 15 * time.Minute
	// tokenBytes es la entropía del token de sesión (256 bits).
	tokenBytes = 32
)

// Errores públicos del paquete. Los mensajes no distinguen entre usuario
// inexistente y contraseña incorrecta a propósito.
var (
	ErrInvalidCredentials = errors.New("auth: usuario o contraseña incorrectos")
	ErrAccountLocked      = errors.New("auth: acceso bloqueado temporalmente por intentos fallidos")
	ErrInvalidSession     = errors.New("auth: sesión no válida")
	ErrSessionExpired     = errors.New("auth: sesión caducada")
)

// User es un usuario autenticado.
type User struct {
	ID       int64
	Username string
	Role     string
}

// Service agrupa las operaciones de autenticación sobre una base de datos.
type Service struct {
	db           *sql.DB
	now          func() time.Time
	sessionTTL   time.Duration
	maxAttempts  int
	lockDuration time.Duration

	mu       sync.Mutex
	attempts map[string]*attempt
}

type attempt struct {
	count       int
	last        time.Time
	lockedUntil time.Time
}

// Option ajusta el comportamiento del servicio (pensado para pruebas).
type Option func(*Service)

// WithClock sustituye el reloj del servicio.
func WithClock(now func() time.Time) Option {
	return func(s *Service) {
		if now != nil {
			s.now = now
		}
	}
}

// WithSessionTTL cambia la vida de las sesiones.
func WithSessionTTL(d time.Duration) Option {
	return func(s *Service) {
		if d > 0 {
			s.sessionTTL = d
		}
	}
}

// WithLockout cambia el umbral y la duración del bloqueo por intentos
// fallidos.
func WithLockout(maxAttempts int, d time.Duration) Option {
	return func(s *Service) {
		if maxAttempts > 0 {
			s.maxAttempts = maxAttempts
		}
		if d > 0 {
			s.lockDuration = d
		}
	}
}

// New crea el servicio de autenticación. Las sesiones duran 24 horas y, si
// quedan menos de 12, se renuevan en cada uso (caducidad deslizante). Tras 5
// intentos fallidos seguidos para un mismo nombre de usuario, el login se
// bloquea 15 minutos. El bloqueo vive en memoria del proceso y se pierde al
// reiniciar, algo asumible en un servidor personal.
func New(db *sql.DB, opts ...Option) *Service {
	s := &Service{
		db:           db,
		now:          time.Now,
		sessionTTL:   DefaultSessionTTL,
		maxAttempts:  DefaultMaxAttempts,
		lockDuration: DefaultLockDuration,
		attempts:     make(map[string]*attempt),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// HashPassword devuelve un hash argon2id en formato PHC con sal aleatoria.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("auth: la contraseña no puede estar vacía")
	}

	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("auth: generar sal: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword comprueba una contraseña contra un hash argon2id del formato
// que genera HashPassword.
func VerifyPassword(encodedHash, password string) (bool, error) {
	params, salt, key, err := parseHash(encodedHash)
	if err != nil {
		return false, err
	}

	candidate := argon2.IDKey([]byte(password), salt, params.time, params.memory, params.threads, uint32(len(key)))
	return subtle.ConstantTimeCompare(candidate, key) == 1, nil
}

type hashParams struct {
	memory  uint32
	time    uint32
	threads uint8
}

func parseHash(encodedHash string) (hashParams, []byte, []byte, error) {
	var params hashParams

	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return params, nil, nil, errors.New("auth: hash de contraseña con formato desconocido")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return params, nil, nil, errors.New("auth: versión de argon2 no soportada")
	}

	values := strings.Split(parts[3], ",")
	if len(values) != 3 {
		return params, nil, nil, errors.New("auth: parámetros de argon2 no válidos")
	}
	memory, err := strconv.ParseUint(strings.TrimPrefix(values[0], "m="), 10, 32)
	if err != nil || memory == 0 {
		return params, nil, nil, errors.New("auth: memoria de argon2 no válida")
	}
	iterations, err := strconv.ParseUint(strings.TrimPrefix(values[1], "t="), 10, 32)
	if err != nil || iterations == 0 {
		return params, nil, nil, errors.New("auth: iteraciones de argon2 no válidas")
	}
	threads, err := strconv.ParseUint(strings.TrimPrefix(values[2], "p="), 10, 8)
	if err != nil || threads == 0 {
		return params, nil, nil, errors.New("auth: hilos de argon2 no válidos")
	}
	params = hashParams{memory: uint32(memory), time: uint32(iterations), threads: uint8(threads)}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return params, nil, nil, errors.New("auth: sal de argon2 no válida")
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(key) == 0 {
		return params, nil, nil, errors.New("auth: clave de argon2 no válida")
	}
	return params, salt, key, nil
}

// Login comprueba las credenciales y, si son correctas, crea una sesión y
// devuelve el token en claro (que no se guarda) y su caducidad.
func (s *Service) Login(ctx context.Context, username, password string) (string, time.Time, error) {
	key := normalizeUsername(username)
	now := s.now()

	if s.isLocked(key, now) {
		return "", time.Time{}, ErrAccountLocked
	}

	user, passwordHash, err := s.findUser(ctx, username)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Mismo coste que una verificación real, para no delatar por tiempo si
		// el usuario existe.
		_, _ = VerifyPassword(dummyHash(), password)
		s.registerFailure(key, now)
		return "", time.Time{}, ErrInvalidCredentials
	case err != nil:
		return "", time.Time{}, err
	}

	valid, err := VerifyPassword(passwordHash, password)
	if err != nil {
		return "", time.Time{}, err
	}
	if !valid {
		s.registerFailure(key, now)
		return "", time.Time{}, ErrInvalidCredentials
	}
	s.resetFailures(key)

	token, err := newToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := now.Add(s.sessionTTL).UTC().Truncate(time.Millisecond)

	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (user_id, token_hash, expires_at) VALUES (?, ?, ?)`,
		user.ID, hashToken(token), formatTime(expiresAt),
	); err != nil {
		return "", time.Time{}, fmt.Errorf("auth: crear sesión: %w", err)
	}

	return token, expiresAt, nil
}

// Authenticate valida un token de sesión y devuelve el usuario. Si la sesión
// está a menos de la mitad de su vida, la alarga (renovación deslizante).
func (s *Service) Authenticate(ctx context.Context, token string) (User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return User{}, ErrInvalidSession
	}
	hashed := hashToken(token)
	now := s.now().UTC()

	var user User
	var expiresRaw string
	err := s.db.QueryRowContext(ctx,
		`SELECT u.id, u.username, u.role, s.expires_at
		   FROM sessions s
		   JOIN users u ON u.id = s.user_id
		  WHERE s.token_hash = ?`, hashed,
	).Scan(&user.ID, &user.Username, &user.Role, &expiresRaw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return User{}, ErrInvalidSession
	case err != nil:
		return User{}, fmt.Errorf("auth: consultar sesión: %w", err)
	}

	expiresAt, err := parseTime(expiresRaw)
	if err != nil {
		return User{}, fmt.Errorf("auth: sesión con fecha no válida: %w", err)
	}
	if !now.Before(expiresAt) {
		// Limpieza perezosa: una sesión caducada ya no sirve para nada.
		_, _ = s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, hashed)
		return User{}, ErrSessionExpired
	}

	if expiresAt.Sub(now) < s.sessionTTL/2 {
		renewed := now.Add(s.sessionTTL).Truncate(time.Millisecond)
		if _, err := s.db.ExecContext(ctx,
			`UPDATE sessions SET expires_at = ? WHERE token_hash = ?`, formatTime(renewed), hashed,
		); err != nil {
			return User{}, fmt.Errorf("auth: renovar sesión: %w", err)
		}
	}

	return user, nil
}

// Logout invalida la sesión del token. Es idempotente: un token desconocido no
// produce error.
func (s *Service) Logout(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE token_hash = ?`, hashToken(token),
	); err != nil {
		return fmt.Errorf("auth: cerrar sesión: %w", err)
	}
	return nil
}

func (s *Service) findUser(ctx context.Context, username string) (User, string, error) {
	var user User
	var passwordHash string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, role, password_hash FROM users WHERE username = ?`, username,
	).Scan(&user.ID, &user.Username, &user.Role, &passwordHash)
	if err != nil {
		return User{}, "", err
	}
	return user, passwordHash, nil
}

func (s *Service) isLocked(key string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.attempts[key]
	if !ok {
		return false
	}
	if now.Before(a.lockedUntil) {
		return true
	}
	if !a.lockedUntil.IsZero() {
		// El bloqueo ya expiró: se empieza de cero.
		delete(s.attempts, key)
	}
	return false
}

func (s *Service) registerFailure(key string, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.prune(now)

	a, ok := s.attempts[key]
	if !ok {
		a = &attempt{}
		s.attempts[key] = a
	}
	a.count++
	a.last = now
	if a.count >= s.maxAttempts {
		a.count = 0
		a.lockedUntil = now.Add(s.lockDuration)
	}
}

func (s *Service) resetFailures(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.attempts, key)
}

// prune evita que el mapa de intentos crezca sin límite con nombres de usuario
// inventados.
func (s *Service) prune(now time.Time) {
	if len(s.attempts) < 1024 {
		return
	}
	for key, a := range s.attempts {
		if now.Sub(a.last) > s.lockDuration && !now.Before(a.lockedUntil) {
			delete(s.attempts, key)
		}
	}
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func newToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: generar token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func formatTime(t time.Time) string {
	return t.UTC().Truncate(time.Millisecond).Format("2006-01-02T15:04:05.000Z07:00")
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}

// dummyHash se calcula una sola vez y sirve para gastar el mismo tiempo cuando
// el usuario no existe.
var dummyHash = sync.OnceValue(func() string {
	h, err := HashPassword("cinecat-usuario-inexistente")
	if err != nil {
		return ""
	}
	return h
})
