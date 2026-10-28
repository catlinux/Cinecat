package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"
	"unicode"
)

const (
	pairingTTL        = 10 * time.Minute
	pairingPollEvery  = 5 * time.Second
	pairingPollMin    = 4 * time.Second
	pairingCodeLength = 8
	pairingAlphabet   = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	pairingPath       = "/vincular"
	pairCreateLimit   = 10
	pairCreateWindow  = 10 * time.Minute
	pairApproveLimit  = 5
	pairTokenLimit    = 5
	pairFailureWindow = 15 * time.Minute
	pairMaxPending    = 50
	pairCleanupRetain = time.Hour
	pairPairIDBytes   = 16
)

var (
	ErrPairingNotFound     = errors.New("auth: código no válido o caducado")
	ErrPairingExpired      = errors.New("auth: vinculación caducada")
	ErrPairingPending      = errors.New("auth: vinculación pendiente")
	ErrPairingRateLimited  = errors.New("auth: demasiados intentos")
	ErrPairingTooFast      = errors.New("auth: demasiado rápido")
	ErrPairingLimit        = errors.New("auth: demasiadas vinculaciones pendientes")
	ErrPairingUserInactive = errors.New("auth: usuario no disponible")
	ErrPairingCodeInvalid  = errors.New("auth: código no válido")
	ErrPairingCode         = errors.New("auth: no se pudo generar un código único")
)

// Pairing es la información que se entrega al dispositivo al iniciar la
// vinculación.
type Pairing struct {
	PairID     string
	Secret     string
	Code       string
	DeviceName string
	Client     string
	ExpiresAt  time.Time
}

// PairingPreview muestra los datos del dispositivo antes de aprobarlo.
type PairingPreview struct {
	DeviceName string
	Client     string
	ExpiresAt  time.Time
	Approved   bool
}

// PairingSession es el resultado de canjear una vinculación aprobada.
type PairingSession struct {
	Token     string
	ExpiresAt time.Time
	SessionID int64
	Username  string
	Role      string
}

// CreatePairing inicia una vinculación para un cliente nativo.
func (s *Service) CreatePairing(
	ctx context.Context,
	deviceName, client, clientIP string,
) (Pairing, error) {
	client, err := normalizePairingClient(client)
	if err != nil {
		return Pairing{}, err
	}
	deviceName = cleanDeviceName(deviceName)
	now := s.now().UTC().Truncate(time.Millisecond)

	s.pairCreateMu.Lock()
	defer s.pairCreateMu.Unlock()
	if err := s.CleanupPairings(ctx); err != nil {
		log.Printf("auth: limpieza de vinculaciones: %v", err)
	}
	if !s.pairIPLimiter.allow(normalizeClientIP(clientIP), now) {
		return Pairing{}, ErrPairingRateLimited
	}

	var pending int
	if err := s.db.QueryRowContext(ctx,
		`SELECT count(*) FROM device_pairings
		  WHERE consumed_at IS NULL AND expires_at > ?`, formatTime(now),
	).Scan(&pending); err != nil {
		return Pairing{}, fmt.Errorf("auth: contar vinculaciones pendientes: %w", err)
	}
	if pending >= pairMaxPending {
		return Pairing{}, ErrPairingLimit
	}

	expiresAt := now.Add(pairingTTL)
	for attempt := 0; attempt < 8; attempt++ {
		pairID, err := newPairID()
		if err != nil {
			return Pairing{}, err
		}
		secret, err := newToken()
		if err != nil {
			return Pairing{}, err
		}
		code, err := newPairCode()
		if err != nil {
			return Pairing{}, err
		}
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO device_pairings (
				pair_id, secret_hash, code_hash, device_name, client, created_at, expires_at
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			pairID, hashToken(secret), hashToken(code), deviceName, client,
			formatTime(now), formatTime(expiresAt),
		); err != nil {
			if isUniqueConstraintError(err) {
				continue
			}
			return Pairing{}, fmt.Errorf("auth: crear vinculación: %w", err)
		}
		return Pairing{
			PairID:     pairID,
			Secret:     secret,
			Code:       formatPairCode(code),
			DeviceName: deviceName,
			Client:     client,
			ExpiresAt:  expiresAt,
		}, nil
	}
	return Pairing{}, ErrPairingCode
}

// ApprovePairing previsualiza o aprueba un código para el usuario indicado.
func (s *Service) ApprovePairing(
	ctx context.Context,
	userID int64,
	code string,
	confirm bool,
	clientIP string,
) (PairingPreview, error) {
	normalized, err := normalizePairCode(code)
	if err != nil {
		return PairingPreview{}, err
	}
	now := s.now().UTC().Truncate(time.Millisecond)
	key := fmt.Sprintf("%d\x00%s", userID, normalizeClientIP(clientIP))
	if s.pairApproveLimiter.blocked(key, now) {
		return PairingPreview{}, ErrPairingRateLimited
	}

	var (
		id, expiresRaw       string
		deviceName, client   string
		approvedRaw, usedRaw sql.NullString
	)
	err = s.db.QueryRowContext(ctx,
		`SELECT id, device_name, client, expires_at, approved_at, consumed_at
		   FROM device_pairings WHERE code_hash = ?`, hashToken(normalized),
	).Scan(&id, &deviceName, &client, &expiresRaw, &approvedRaw, &usedRaw)
	if err != nil {
		s.pairApproveLimiter.allow(key, now)
		if errors.Is(err, sql.ErrNoRows) {
			return PairingPreview{}, ErrPairingNotFound
		}
		return PairingPreview{}, fmt.Errorf("auth: consultar vinculación: %w", err)
	}

	expiresAt, parseErr := parseTime(expiresRaw)
	if parseErr != nil || !now.Before(expiresAt) || approvedRaw.Valid || usedRaw.Valid {
		s.pairApproveLimiter.allow(key, now)
		return PairingPreview{}, ErrPairingNotFound
	}

	preview := PairingPreview{
		DeviceName: deviceName,
		Client:     client,
		ExpiresAt:  expiresAt,
		Approved:   confirm,
	}
	if !confirm {
		s.pairApproveLimiter.reset(key)
		return preview, nil
	}

	result, err := s.db.ExecContext(ctx,
		`UPDATE device_pairings
		    SET approved_user_id = ?, approved_at = ?
		  WHERE id = ? AND approved_at IS NULL AND consumed_at IS NULL AND expires_at > ?`,
		userID, formatTime(now), id, formatTime(now),
	)
	if err != nil {
		return PairingPreview{}, fmt.Errorf("auth: aprobar vinculación: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return PairingPreview{}, fmt.Errorf("auth: comprobar aprobación: %w", err)
	}
	if affected != 1 {
		s.pairApproveLimiter.allow(key, now)
		return PairingPreview{}, ErrPairingNotFound
	}
	s.pairApproveLimiter.reset(key)
	return preview, nil
}

// RedeemPairing entrega una única vez la sesión de una vinculación aprobada.
func (s *Service) RedeemPairing(
	ctx context.Context,
	pairID, secret, clientIP string,
) (PairingSession, error) {
	now := s.now().UTC().Truncate(time.Millisecond)
	ipKey := normalizeClientIP(clientIP)
	if s.pairTokenLimiter.blocked(ipKey, now) {
		return PairingSession{}, ErrPairingRateLimited
	}

	var (
		id, secretHash, deviceName, client, expiresRaw string
		lastPolledRaw, approvedRaw, consumedRaw        sql.NullString
		approvedUserID                                 sql.NullInt64
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT id, secret_hash, device_name, client, expires_at,
		        last_polled_at, approved_user_id, approved_at, consumed_at
		   FROM device_pairings WHERE pair_id = ?`, pairID,
	).Scan(&id, &secretHash, &deviceName, &client, &expiresRaw,
		&lastPolledRaw, &approvedUserID, &approvedRaw, &consumedRaw)
	if err != nil {
		s.pairTokenLimiter.allow(ipKey, now)
		if errors.Is(err, sql.ErrNoRows) {
			return PairingSession{}, ErrPairingNotFound
		}
		return PairingSession{}, fmt.Errorf("auth: consultar vinculación: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(secretHash), []byte(hashToken(secret))) != 1 {
		s.pairTokenLimiter.allow(ipKey, now)
		return PairingSession{}, ErrPairingNotFound
	}

	expiresAt, parseErr := parseTime(expiresRaw)
	if parseErr != nil || !now.Before(expiresAt) || consumedRaw.Valid {
		return PairingSession{}, ErrPairingExpired
	}
	if !approvedRaw.Valid {
		if lastPolledRaw.Valid {
			if lastPolled, err := parseTime(lastPolledRaw.String); err == nil &&
				now.Sub(lastPolled) < pairingPollMin {
				return PairingSession{}, ErrPairingTooFast
			}
		}
		if _, err := s.db.ExecContext(ctx,
			`UPDATE device_pairings SET last_polled_at = ? WHERE id = ?`,
			formatTime(now), id,
		); err != nil {
			return PairingSession{}, fmt.Errorf("auth: registrar sondeo: %w", err)
		}
		return PairingSession{}, ErrPairingPending
	}
	if !approvedUserID.Valid {
		return PairingSession{}, ErrPairingUserInactive
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PairingSession{}, fmt.Errorf("auth: iniciar entrega de sesión: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	result, err := tx.ExecContext(ctx,
		`UPDATE device_pairings
		    SET consumed_at = ?
		  WHERE id = ? AND consumed_at IS NULL AND approved_at IS NOT NULL
		    AND approved_user_id IS NOT NULL AND expires_at > ?`,
		formatTime(now), id, formatTime(now),
	)
	if err != nil {
		return PairingSession{}, fmt.Errorf("auth: consumir vinculación: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return PairingSession{}, fmt.Errorf("auth: comprobar consumo: %w", err)
	}
	if affected != 1 {
		return PairingSession{}, ErrPairingExpired
	}

	var username, role string
	var disabled int
	err = tx.QueryRowContext(ctx,
		`SELECT username, role, disabled FROM users WHERE id = ?`, approvedUserID.Int64,
	).Scan(&username, &role, &disabled)
	if errors.Is(err, sql.ErrNoRows) || disabled != 0 {
		return PairingSession{}, ErrPairingUserInactive
	}
	if err != nil {
		return PairingSession{}, fmt.Errorf("auth: consultar usuario aprobador: %w", err)
	}

	session, err := s.createSession(ctx, tx, approvedUserID.Int64, client, deviceName, now)
	if err != nil {
		return PairingSession{}, err
	}
	if err := tx.Commit(); err != nil {
		return PairingSession{}, fmt.Errorf("auth: confirmar vinculación: %w", err)
	}
	s.pairTokenLimiter.reset(ipKey)

	return PairingSession{
		Token:     session.Token,
		ExpiresAt: session.ExpiresAt,
		SessionID: session.SessionID,
		Username:  username,
		Role:      role,
	}, nil
}

// CleanupPairings borra las vinculaciones caducadas y las consumidas hace más
// de una hora.
func (s *Service) CleanupPairings(ctx context.Context) error {
	now := s.now().UTC().Truncate(time.Millisecond)
	if _, err := s.db.ExecContext(ctx,
		`DELETE FROM device_pairings
		  WHERE expires_at <= ? OR (consumed_at IS NOT NULL AND consumed_at < ?)`,
		formatTime(now), formatTime(now.Add(-pairCleanupRetain)),
	); err != nil {
		return fmt.Errorf("auth: limpiar vinculaciones: %w", err)
	}
	return nil
}

func normalizePairingClient(client string) (string, error) {
	client = strings.TrimSpace(client)
	switch client {
	case ClientAndroid, ClientAndroidTV:
		return client, nil
	default:
		return "", ErrInvalidClient
	}
}

func normalizePairCode(raw string) (string, error) {
	var builder strings.Builder
	for _, char := range strings.ToUpper(strings.TrimSpace(raw)) {
		if char == '-' || unicode.IsSpace(char) {
			continue
		}
		if !strings.ContainsRune(pairingAlphabet, char) {
			return "", ErrPairingCodeInvalid
		}
		builder.WriteRune(char)
	}
	if builder.Len() != pairingCodeLength {
		return "", ErrPairingCodeInvalid
	}
	return builder.String(), nil
}

func formatPairCode(code string) string {
	if len(code) != pairingCodeLength {
		return code
	}
	return code[:4] + "-" + code[4:]
}

func newPairID() (string, error) {
	buffer := make([]byte, pairPairIDBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("auth: generar identificador de vinculación: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func newPairCode() (string, error) {
	maximum := big.NewInt(int64(len(pairingAlphabet)))
	var builder strings.Builder
	builder.Grow(pairingCodeLength)
	for index := 0; index < pairingCodeLength; index++ {
		value, err := rand.Int(rand.Reader, maximum)
		if err != nil {
			return "", fmt.Errorf("auth: generar código de vinculación: %w", err)
		}
		builder.WriteByte(pairingAlphabet[value.Int64()])
	}
	return builder.String(), nil
}
