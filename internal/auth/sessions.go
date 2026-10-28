package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
)

const (
	// Clientes de sesión admitidos.
	ClientWeb       = "web"
	ClientAndroid   = "android"
	ClientAndroidTV = "androidtv"

	maxDeviceNameLength  = 80
	sessionTouchInterval = 5 * time.Minute
)

// Session es una sesión activa tal como se muestra al usuario.
type Session struct {
	ID         int64
	DeviceName string
	Client     string
	CreatedAt  string
	LastSeenAt string
	ExpiresAt  string
	Current    bool
}

// ListSessions devuelve las sesiones activas de un usuario, de la última
// actividad a la más antigua. currentSessionID marca la sesión de la petición.
func (s *Service) ListSessions(ctx context.Context, userID, currentSessionID int64) ([]Session, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, device_name, client, created_at, last_seen_at, expires_at
		  FROM sessions
		 WHERE user_id = ? AND expires_at > ?
		 ORDER BY last_seen_at DESC, id DESC`, userID, formatTime(s.now()))
	if err != nil {
		return nil, fmt.Errorf("auth: listar sesiones: %w", err)
	}
	defer rows.Close()

	sessions := make([]Session, 0)
	for rows.Next() {
		var session Session
		if err := rows.Scan(&session.ID, &session.DeviceName, &session.Client,
			&session.CreatedAt, &session.LastSeenAt, &session.ExpiresAt); err != nil {
			return nil, fmt.Errorf("auth: leer sesión: %w", err)
		}
		session.Current = currentSessionID != 0 && session.ID == currentSessionID
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("auth: recorrer sesiones: %w", err)
	}
	return sessions, nil
}

// DeleteSession cierra una sesión que pertenezca al usuario indicado. Si no
// existe o es de otra cuenta devuelve ErrSessionNotFound.
func (s *Service) DeleteSession(ctx context.Context, userID, sessionID int64) error {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE id = ? AND user_id = ?`, sessionID, userID)
	if err != nil {
		return fmt.Errorf("auth: cerrar sesión: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("auth: comprobar sesión cerrada: %w", err)
	}
	if affected == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (s *Service) userExists(ctx context.Context, userID int64) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM users WHERE id = ?`, userID).Scan(&exists)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("auth: comprobar usuario: %w", err)
	default:
		return true, nil
	}
}

func (s *Service) sessionTTLFor(client string) time.Duration {
	switch client {
	case ClientAndroid, ClientAndroidTV:
		return NativeSessionTTL
	default:
		return s.sessionTTL
	}
}

func normalizeClient(raw string) (string, error) {
	client := strings.TrimSpace(raw)
	if client == "" {
		return ClientWeb, nil
	}
	switch client {
	case ClientWeb, ClientAndroid, ClientAndroidTV:
		return client, nil
	default:
		return "", ErrInvalidClient
	}
}

func cleanDeviceName(raw string) string {
	raw = strings.ToValidUTF8(raw, "")
	raw = strings.TrimSpace(raw)
	var builder strings.Builder
	builder.Grow(len(raw))
	count := 0
	for _, char := range raw {
		if unicode.IsControl(char) {
			continue
		}
		if count >= maxDeviceNameLength {
			break
		}
		builder.WriteRune(char)
		count++
	}
	return strings.TrimSpace(builder.String())
}

// deviceLabel resume el navegador y el sistema operativo del User-Agent.
func deviceLabel(userAgent string) string {
	agent := strings.ToLower(userAgent)

	browser := ""
	switch {
	case strings.Contains(agent, "edg/"), strings.Contains(agent, "edgios/"),
		strings.Contains(agent, "edga/"), strings.Contains(agent, "edge/"):
		browser = "Edge"
	case strings.Contains(agent, "opr/"), strings.Contains(agent, "opera"):
		browser = "Opera"
	case strings.Contains(agent, "chrome/"), strings.Contains(agent, "crios/"):
		browser = "Chrome"
	case strings.Contains(agent, "firefox/"), strings.Contains(agent, "fxios/"):
		browser = "Firefox"
	case strings.Contains(agent, "safari/"):
		browser = "Safari"
	}

	system := ""
	switch {
	case strings.Contains(agent, "windows"):
		system = "Windows"
	case strings.Contains(agent, "android"):
		system = "Android"
	case strings.Contains(agent, "iphone"), strings.Contains(agent, "ipad"),
		strings.Contains(agent, "ipod"):
		system = "iOS"
	case (strings.Contains(agent, "macintosh") || strings.Contains(agent, "mac os x")) &&
		strings.Contains(agent, "mobile/"):
		// iPadOS puede presentarse como un Mac de escritorio.
		system = "iOS"
	case strings.Contains(agent, "macintosh"), strings.Contains(agent, "mac os x"):
		system = "macOS"
	case strings.Contains(agent, "linux"):
		system = "Linux"
	}

	switch {
	case browser != "" && system != "":
		return browser + " · " + system
	case browser != "":
		return browser
	case system != "":
		return "Navegador · " + system
	default:
		return "Navegador"
	}
}
