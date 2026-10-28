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
	"math"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

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
	// DefaultMaxAttempts es el número de fallos seguidos que bloquea una
	// pareja usuario+IP.
	DefaultMaxAttempts = 5
	// DefaultMaxAttemptsPerIP es el máximo de fallos desde una sola IP, con
	// cualquier usuario, antes de bloquearla.
	DefaultMaxAttemptsPerIP = 20
	// DefaultLockDuration es lo que dura el bloqueo temporal.
	DefaultLockDuration = 15 * time.Minute
	// maxAttemptEntries acota el mapa en memoria aunque lleguen nombres e IPs
	// inventados.
	maxAttemptEntries = 10000
	// tokenBytes es la entropía del token de sesión (256 bits).
	tokenBytes = 32
)

// Errores públicos del paquete. Los mensajes no distinguen entre usuario
// inexistente y contraseña incorrecta a propósito.
var (
	ErrInvalidCredentials     = errors.New("auth: usuario o contraseña incorrectos")
	ErrAccountLocked          = errors.New("auth: acceso bloqueado temporalmente por intentos fallidos")
	ErrInvalidSession         = errors.New("auth: sesión no válida")
	ErrSessionExpired         = errors.New("auth: sesión caducada")
	ErrInvalidCurrentPassword = errors.New("auth: la contraseña actual no es correcta")
	ErrPasswordTooShort       = errors.New("auth: la contraseña nueva es demasiado corta")
)

// minPasswordLength es la longitud mínima de una contraseña nueva, en caracteres.
const minPasswordLength = 8

// User es un usuario autenticado o una cuenta gestionada por el
// administrador.
type User struct {
	ID             int64
	Username       string
	Role           string
	Disabled       bool
	AllLibraries   bool
	LibraryIDs     []int64
	CreatedAt      string
	ActiveSessions int
}

// Service agrupa las operaciones de autenticación sobre una base de datos.
type Service struct {
	db            *sql.DB
	now           func() time.Time
	sessionTTL    time.Duration
	maxAttempts   int
	maxIPAttempts int
	lockDuration  time.Duration
	trusted       []netip.Prefix

	mu         sync.Mutex
	attempts   map[string]*attempt
	ipAttempts map[string]*attempt
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

// WithIPLockout cambia el umbral y la duración del bloqueo por IP. La
// duración es la misma que la del bloqueo por pareja.
func WithIPLockout(maxAttempts int, d time.Duration) Option {
	return func(s *Service) {
		if maxAttempts > 0 {
			s.maxIPAttempts = maxAttempts
		}
		if d > 0 {
			s.lockDuration = d
		}
	}
}

// WithTrustedProxies sustituye las redes desde las que se acepta
// X-Forwarded-For. Una lista vacía desactiva la confianza en la cabecera.
func WithTrustedProxies(prefixes []netip.Prefix) Option {
	return func(s *Service) {
		s.trusted = append([]netip.Prefix(nil), prefixes...)
	}
}

// New crea el servicio de autenticación. Las sesiones duran 24 horas y, si
// quedan menos de 12, se renuevan en cada uso (caducidad deslizante). Tras 5
// intentos fallidos seguidos para una misma pareja usuario+IP, el login de esa
// pareja se bloquea 15 minutos; además, 20 fallos desde una IP bloquean todas
// las peticiones de esa IP. El bloqueo vive en memoria del proceso y se pierde
// al reiniciar, algo asumible en un servidor personal.
func New(db *sql.DB, opts ...Option) *Service {
	s := &Service{
		db:            db,
		now:           time.Now,
		sessionTTL:    DefaultSessionTTL,
		maxAttempts:   DefaultMaxAttempts,
		maxIPAttempts: DefaultMaxAttemptsPerIP,
		lockDuration:  DefaultLockDuration,
		trusted:       DefaultTrustedProxies(),
		attempts:      make(map[string]*attempt),
		ipAttempts:    make(map[string]*attempt),
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

// Login comprueba las credenciales sin IP de cliente. Se conserva para los
// llamadores que no pasan por HTTP; equivale a una IP desconocida.
func (s *Service) Login(ctx context.Context, username, password string) (string, time.Time, error) {
	return s.LoginFrom(ctx, username, password, "")
}

// LoginFrom comprueba las credenciales contando los fallos por pareja
// usuario+IP y por IP. Un login correcto limpia solo la pareja.
func (s *Service) LoginFrom(ctx context.Context, username, password, clientIP string) (string, time.Time, error) {
	key := loginPairKey(username, clientIP)
	ipKey := ""
	if strings.TrimSpace(clientIP) != "" {
		ipKey = normalizeClientIP(clientIP)
	}
	now := s.now()

	user, passwordHash, err := s.findUser(ctx, username)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if s.isLockedFor(key, ipKey, now) {
			return "", time.Time{}, ErrAccountLocked
		}
		// Mismo coste que una verificación real, para no delatar por tiempo si
		// el usuario existe.
		_, _ = VerifyPassword(dummyHash(), password)
		s.registerFailureFor(key, ipKey, now)
		return "", time.Time{}, ErrInvalidCredentials
	case err != nil:
		return "", time.Time{}, err
	}

	if user.Disabled {
		// Se verifica igualmente para no delatar por tiempo que la cuenta existe,
		// pero no se cuenta el intento: una cuenta desactivada no debe bloquear
		// a nadie.
		_, _ = VerifyPassword(passwordHash, password)
		return "", time.Time{}, ErrInvalidCredentials
	}
	if s.isLockedFor(key, ipKey, now) {
		return "", time.Time{}, ErrAccountLocked
	}

	valid, err := VerifyPassword(passwordHash, password)
	if err != nil {
		return "", time.Time{}, err
	}
	if !valid {
		s.registerFailureFor(key, ipKey, now)
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
	var disabled int
	err := s.db.QueryRowContext(ctx,
		`SELECT u.id, u.username, u.role, u.disabled, s.expires_at
		   FROM sessions s
		   JOIN users u ON u.id = s.user_id
		  WHERE s.token_hash = ?`, hashed,
	).Scan(&user.ID, &user.Username, &user.Role, &disabled, &expiresRaw)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return User{}, ErrInvalidSession
	case err != nil:
		return User{}, fmt.Errorf("auth: consultar sesión: %w", err)
	}
	if disabled != 0 {
		// Una cuenta desactivada no debe seguir usando una sesión que hubiera
		// sobrevivido al cambio de estado.
		_, _ = s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, hashed)
		return User{}, ErrInvalidSession
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

// ChangePassword comprueba la contraseña actual del usuario y, si coincide,
// guarda la nueva. Si se indica el token de la petición, cierra en la misma
// transacción las demás sesiones del usuario y conserva solo la actual.
func (s *Service) ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string, currentToken ...string) error {
	if utf8.RuneCountInString(newPassword) < minPasswordLength {
		return ErrPasswordTooShort
	}

	var currentHash string
	err := s.db.QueryRowContext(ctx,
		`SELECT password_hash FROM users WHERE id = ?`, userID,
	).Scan(&currentHash)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrInvalidSession
	case err != nil:
		return fmt.Errorf("auth: consultar la contraseña actual: %w", err)
	}

	valid, err := VerifyPassword(currentHash, currentPassword)
	if err != nil {
		return err
	}
	if !valid {
		return ErrInvalidCurrentPassword
	}

	newHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("auth: iniciar cambio de contraseña: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET password_hash = ? WHERE id = ?`, newHash, userID,
	); err != nil {
		return fmt.Errorf("auth: actualizar la contraseña: %w", err)
	}
	if len(currentToken) > 0 {
		token := strings.TrimSpace(currentToken[0])
		if token != "" {
			if _, err := tx.ExecContext(ctx,
				`DELETE FROM sessions WHERE user_id = ? AND token_hash <> ?`,
				userID, hashToken(token),
			); err != nil {
				return fmt.Errorf("auth: cerrar las demás sesiones: %w", err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("auth: confirmar cambio de contraseña: %w", err)
	}
	return nil
}

func (s *Service) findUser(ctx context.Context, username string) (User, string, error) {
	var user User
	var passwordHash string
	var disabled int
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, role, disabled, password_hash FROM users WHERE username = ?`, username,
	).Scan(&user.ID, &user.Username, &user.Role, &disabled, &passwordHash)
	if err != nil {
		return User{}, "", err
	}
	user.Disabled = disabled != 0
	return user, passwordHash, nil
}

func (s *Service) isLocked(key string, now time.Time) bool {
	return s.isLockedFor(key, "", now)
}

func (s *Service) isLockedFor(key, ipKey string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pruneLocked(now)
	if entryLocked(s.attempts[key], now) {
		return true
	}
	return ipKey != "" && entryLocked(s.ipAttempts[ipKey], now)
}

func (s *Service) registerFailure(key string, now time.Time) {
	s.registerFailureFor(key, "", now)
}

func (s *Service) registerFailureFor(key, ipKey string, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pruneLocked(now)
	registerAttempt(s.attempts, key, now, s.maxAttempts, s.lockDuration)
	if ipKey != "" {
		registerAttempt(s.ipAttempts, ipKey, now, s.maxIPAttempts, s.lockDuration)
	}
	s.trimLocked()
}

func (s *Service) resetFailures(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.attempts, key)
}

func registerAttempt(entries map[string]*attempt, key string, now time.Time, limit int, duration time.Duration) {
	if key == "" {
		return
	}
	entry, ok := entries[key]
	if !ok {
		entry = &attempt{}
		entries[key] = entry
	}
	entry.count++
	entry.last = now
	if entry.count >= limit {
		entry.count = 0
		entry.lockedUntil = now.Add(duration)
	}
}

func entryLocked(entry *attempt, now time.Time) bool {
	return entry != nil && now.Before(entry.lockedUntil)
}

// prune elimina entradas caducadas. Se conserva como método para las pruebas
// y para no exponer el bloqueo interno.
func (s *Service) prune(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked(now)
}

func (s *Service) pruneLocked(now time.Time) {
	for key, entry := range s.attempts {
		if entryExpired(entry, now, s.lockDuration) {
			delete(s.attempts, key)
		}
	}
	for key, entry := range s.ipAttempts {
		if entryExpired(entry, now, s.lockDuration) {
			delete(s.ipAttempts, key)
		}
	}
}

func entryExpired(entry *attempt, now time.Time, duration time.Duration) bool {
	if entry == nil {
		return true
	}
	if now.Before(entry.lockedUntil) {
		return false
	}
	return !now.Before(entry.last.Add(duration))
}

// trimLocked impone el tope duro de memoria. Cuando se supera, descarta las
// entradas con actividad más antigua de cualquiera de los dos mapas.
func (s *Service) trimLocked() {
	for len(s.attempts)+len(s.ipAttempts) > maxAttemptEntries {
		var oldestMap map[string]*attempt
		var oldestKey string
		var oldest time.Time
		found := false
		for _, entries := range []map[string]*attempt{s.attempts, s.ipAttempts} {
			for key, entry := range entries {
				if !found || entry.last.Before(oldest) {
					oldestMap = entries
					oldestKey = key
					oldest = entry.last
					found = true
				}
			}
		}
		if !found {
			return
		}
		delete(oldestMap, oldestKey)
	}
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

const unknownClientIP = "sin-ip"

func normalizeClientIP(clientIP string) string {
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		return unknownClientIP
	}
	if address, ok := parseClientIP(clientIP); ok {
		return address.String()
	}
	return clientIP
}

func loginPairKey(username, clientIP string) string {
	return normalizeUsername(username) + "\x00" + normalizeClientIP(clientIP)
}

func (s *Service) retryAfterSeconds() int {
	seconds := int(math.Ceil(s.lockDuration.Seconds()))
	if seconds < 1 {
		return 1
	}
	return seconds
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
