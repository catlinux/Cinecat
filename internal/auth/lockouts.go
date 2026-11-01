package auth

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	lockoutHistoryRetention = 30 * 24 * time.Hour
	securityEventsWindow    = 7 * 24 * time.Hour
	securityEventsLimit     = 50
	lockoutUsernameLimit    = 64
)

type securityLockoutDTO struct {
	Kind        string  `json:"kind"`
	Username    *string `json:"username"`
	IP          *string `json:"ip"`
	LockedAt    string  `json:"locked_at"`
	LockedUntil string  `json:"locked_until"`
}

type securityEventsResponse struct {
	Lockouts []securityLockoutDTO `json:"lockouts"`
	Total7d  int                  `json:"total_7d"`
}

func (s *Service) isLockedInDB(
	ctx context.Context,
	key, ipKey string,
	now time.Time,
) (bool, error) {
	nowRaw := formatTime(now)
	var locked int
	var err error
	if ipKey == "" {
		err = s.db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM login_attempts
				 WHERE kind = 'user' AND "key" = ?
				   AND locked_until IS NOT NULL AND locked_until > ?)`,
			key, nowRaw).Scan(&locked)
	} else {
		err = s.db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM login_attempts
				 WHERE ((kind = 'user' AND "key" = ?) OR (kind = 'ip' AND "key" = ?))
				   AND locked_until IS NOT NULL AND locked_until > ?)`,
			key, ipKey, nowRaw).Scan(&locked)
	}
	if err != nil {
		return false, err
	}
	return locked != 0, nil
}

func (s *Service) registerFailureInDB(
	ctx context.Context,
	username, key, ipKey string,
	now time.Time,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	nowRaw := formatTime(now)
	untilRaw := formatTime(now.Add(s.lockDuration))
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM login_attempts WHERE last_at <= ?`,
		formatTime(now.Add(-s.lockDuration)),
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM login_lockouts WHERE locked_at < ?`,
		formatTime(now.Add(-lockoutHistoryRetention)),
	); err != nil {
		return err
	}

	userTriggered, err := incrementLoginAttempt(
		ctx, tx, "user", key, nowRaw, untilRaw, s.maxAttempts,
	)
	if err != nil {
		return err
	}
	ipTriggered := false
	if ipKey != "" {
		ipTriggered, err = incrementLoginAttempt(
			ctx, tx, "ip", ipKey, nowRaw, untilRaw, s.maxIPAttempts,
		)
		if err != nil {
			return err
		}
	}

	logs := make([]string, 0, 2)
	if userTriggered {
		name := truncateLockoutUsername(username)
		var ip any
		if ipKey != "" {
			ip = ipKey
		}
		if err := insertLoginLockout(ctx, tx, "user", name, ip, nowRaw, untilRaw); err != nil {
			return err
		}
		logs = append(logs, fmt.Sprintf(
			"auth: bloqueo de login por usuario %q desde %q hasta %s",
			name, ipKey, untilRaw,
		))
	}
	if ipTriggered {
		if err := insertLoginLockout(ctx, tx, "ip", nil, ipKey, nowRaw, untilRaw); err != nil {
			return err
		}
		logs = append(logs, fmt.Sprintf(
			"auth: bloqueo de login por IP %q hasta %s", ipKey, untilRaw,
		))
	}

	if err := trimLoginAttempts(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	for _, message := range logs {
		log.Print(message)
	}
	return nil
}

func incrementLoginAttempt(
	ctx context.Context,
	tx *sql.Tx,
	kind, key, nowRaw, untilRaw string,
	limit int,
) (bool, error) {
	initialCount := 1
	var initialUntil any
	if limit <= 1 {
		initialCount = 0
		initialUntil = untilRaw
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO login_attempts ("key", kind, count, last_at, locked_until)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (kind, "key") DO UPDATE SET
			count = CASE
				WHEN login_attempts.count + 1 >= ? THEN 0
				ELSE login_attempts.count + 1
			END,
			last_at = excluded.last_at,
			locked_until = CASE
				WHEN login_attempts.count + 1 >= ? THEN ?
				ELSE login_attempts.locked_until
			END`,
		key, kind, initialCount, nowRaw, initialUntil,
		limit, limit, untilRaw,
	); err != nil {
		return false, err
	}

	var count int
	var locked sql.NullString
	if err := tx.QueryRowContext(ctx, `
		SELECT count, locked_until FROM login_attempts
		 WHERE kind = ? AND "key" = ?`, kind, key,
	).Scan(&count, &locked); err != nil {
		return false, err
	}
	return count == 0 && locked.Valid && locked.String == untilRaw, nil
}

func insertLoginLockout(
	ctx context.Context,
	tx *sql.Tx,
	kind string,
	username, ip any,
	lockedAt, lockedUntil string,
) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO login_lockouts (kind, username, ip, locked_at, locked_until)
		VALUES (?, ?, ?, ?, ?)`,
		kind, username, ip, lockedAt, lockedUntil)
	return err
}

func trimLoginAttempts(ctx context.Context, tx *sql.Tx) error {
	var count int
	if err := tx.QueryRowContext(ctx,
		`SELECT count(*) FROM login_attempts`).Scan(&count); err != nil {
		return err
	}
	excess := count - maxAttemptEntries
	if excess <= 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, `
		DELETE FROM login_attempts
		 WHERE rowid IN (
			SELECT rowid FROM login_attempts
			 ORDER BY last_at ASC, kind, "key" ASC
			 LIMIT ?)`, excess)
	return err
}

func (s *Service) resetFailuresInDB(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM login_attempts WHERE kind = 'user' AND "key" = ?`, key)
	return err
}

func truncateLockoutUsername(username string) string {
	username = strings.TrimSpace(username)
	if username == "" {
		return ""
	}
	runes := []rune(username)
	if len(runes) > lockoutUsernameLimit {
		runes = runes[:lockoutUsernameLimit]
	}
	return string(runes)
}

// RegisterSecurityRoutes monta el listado de bloqueos para administradores.
func (s *Service) RegisterSecurityRoutes(mux *http.ServeMux) {
	if s == nil || mux == nil {
		return
	}
	mux.Handle("GET /api/v1/system/security-events", s.RequireAdmin(http.HandlerFunc(s.handleSecurityEvents)))
}

func (s *Service) handleSecurityEvents(w http.ResponseWriter, r *http.Request) {
	lockouts, total, err := s.SecurityEvents(r.Context())
	if err != nil {
		log.Printf("auth: listar bloqueos de login: %v", err)
		writeError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, securityEventsResponse{
		Lockouts: lockouts,
		Total7d:  total,
	})
}

// SecurityEvents devuelve los bloqueos más recientes y el total de los
// últimos siete días.
func (s *Service) SecurityEvents(ctx context.Context) ([]securityLockoutDTO, int, error) {
	lockouts := make([]securityLockoutDTO, 0)
	if s == nil || s.db == nil {
		return lockouts, 0, nil
	}
	cutoff := formatTime(s.now().Add(-securityEventsWindow))
	var total int
	if err := s.db.QueryRowContext(ctx,
		`SELECT count(*) FROM login_lockouts WHERE locked_at >= ?`, cutoff,
	).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT kind, username, ip, locked_at, locked_until
		  FROM login_lockouts
		 ORDER BY locked_at DESC, id DESC
		 LIMIT ?`, securityEventsLimit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var item securityLockoutDTO
		var username, ip sql.NullString
		if err := rows.Scan(
			&item.Kind, &username, &ip, &item.LockedAt, &item.LockedUntil,
		); err != nil {
			return nil, 0, err
		}
		item.Username = nullableStringPointer(username)
		item.IP = nullableStringPointer(ip)
		lockouts = append(lockouts, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return lockouts, total, nil
}

func nullableStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	text := value.String
	return &text
}
