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

// SessionProfile guarda los ajustes propios de un dispositivo. Los valores
// vacíos significan que se usan los valores por defecto.
type SessionProfile struct {
	MaxQuality       string
	AudioLanguage    string
	SubtitleLanguage string
}

func (p SessionProfile) Empty() bool {
	return p.MaxQuality == "" && p.AudioLanguage == "" && p.SubtitleLanguage == ""
}

// Session es una sesión activa tal como se muestra al usuario.
type Session struct {
	ID         int64
	DeviceName string
	Client     string
	CreatedAt  string
	LastSeenAt string
	ExpiresAt  string
	Current    bool
	Profile    SessionProfile
}

// ListSessions devuelve las sesiones activas de un usuario, de la última
// actividad a la más antigua. currentSessionID marca la sesión de la petición.
func (s *Service) ListSessions(ctx context.Context, userID, currentSessionID int64) ([]Session, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.device_name, s.client, s.created_at, s.last_seen_at, s.expires_at,
		       COALESCE(p.max_quality, ''), COALESCE(p.audio_language, ''),
		       COALESCE(p.subtitle_language, '')
		  FROM sessions s
		  LEFT JOIN session_profiles p ON p.session_id = s.id
		 WHERE s.user_id = ? AND s.expires_at > ?
		 ORDER BY s.last_seen_at DESC, s.id DESC`, userID, formatTime(s.now()))
	if err != nil {
		return nil, fmt.Errorf("auth: listar sesiones: %w", err)
	}
	defer rows.Close()

	sessions := make([]Session, 0)
	for rows.Next() {
		var session Session
		if err := rows.Scan(&session.ID, &session.DeviceName, &session.Client,
			&session.CreatedAt, &session.LastSeenAt, &session.ExpiresAt,
			&session.Profile.MaxQuality, &session.Profile.AudioLanguage,
			&session.Profile.SubtitleLanguage); err != nil {
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

// SessionProfileFor devuelve el perfil de una sesión activa del usuario. Una
// sesión ajena, caducada o inexistente se trata igual que una que no existe.
func (s *Service) SessionProfileFor(ctx context.Context, userID, sessionID int64) (SessionProfile, error) {
	var profile SessionProfile
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(p.max_quality, ''), COALESCE(p.audio_language, ''),
		       COALESCE(p.subtitle_language, '')
		  FROM sessions s
		  LEFT JOIN session_profiles p ON p.session_id = s.id
		 WHERE s.id = ? AND s.user_id = ? AND s.expires_at > ?`,
		sessionID, userID, formatTime(s.now())).Scan(
		&profile.MaxQuality, &profile.AudioLanguage, &profile.SubtitleLanguage,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return SessionProfile{}, ErrSessionNotFound
	}
	if err != nil {
		return SessionProfile{}, fmt.Errorf("auth: leer perfil de sesión: %w", err)
	}
	return profile, nil
}

// PutSessionProfile sustituye el perfil de una sesión activa del usuario. Un
// perfil vacío elimina la fila para que no se distinga de una sesión sin perfil.
func (s *Service) PutSessionProfile(
	ctx context.Context,
	userID, sessionID int64,
	profile SessionProfile,
) error {
	var exists int
	err := s.db.QueryRowContext(ctx, `
		SELECT 1 FROM sessions
		 WHERE id = ? AND user_id = ? AND expires_at > ?`,
		sessionID, userID, formatTime(s.now())).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrSessionNotFound
	}
	if err != nil {
		return fmt.Errorf("auth: comprobar sesión para el perfil: %w", err)
	}

	if profile.Empty() {
		if _, err := s.db.ExecContext(ctx,
			`DELETE FROM session_profiles WHERE session_id = ?`, sessionID); err != nil {
			return fmt.Errorf("auth: borrar perfil de sesión: %w", err)
		}
		return nil
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO session_profiles (session_id, max_quality, audio_language, subtitle_language)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (session_id) DO UPDATE SET
			max_quality = excluded.max_quality,
			audio_language = excluded.audio_language,
			subtitle_language = excluded.subtitle_language`,
		sessionID, profile.MaxQuality, profile.AudioLanguage, profile.SubtitleLanguage)
	if err != nil {
		return fmt.Errorf("auth: guardar perfil de sesión: %w", err)
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
