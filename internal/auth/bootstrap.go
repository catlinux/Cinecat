package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
)

const (
	// EnvAdminPassword permite fijar la contraseña del administrador inicial
	// sin que se genere una aleatoria. Nunca debe registrarse.
	EnvAdminPassword = "CINECAT_ADMIN_PASSWORD"
	// DefaultAdminUsername es el nombre del administrador que se crea al
	// arrancar contra una base sin usuarios.
	DefaultAdminUsername = "admin"

	// bootstrapPasswordBytes son los bytes aleatorios de la contraseña
	// inicial: 18 bytes son 144 bits y se codifican en 24 caracteres.
	bootstrapPasswordBytes = 18
)

// BootstrapResult indica si el arranque creó el administrador. GeneratedPassword
// solo lleva valor cuando la contraseña se generó y hay que mostrarla una vez;
// nunca se guarda en disco.
type BootstrapResult struct {
	Created           bool
	Username          string
	GeneratedPassword string
}

// BootstrapAdmin crea el usuario administrador inicial si la tabla users está
// vacía. Si password está vacía se genera una aleatoria con crypto/rand; si
// viene de CINECAT_ADMIN_PASSWORD se usa tal cual y no se registra en ningún
// sitio. Con usuarios ya existentes no crea ni modifica nada.
func BootstrapAdmin(ctx context.Context, db *sql.DB, password string) (BootstrapResult, error) {
	users, err := countUsers(ctx, db)
	if err != nil {
		return BootstrapResult{}, err
	}
	if users > 0 {
		return BootstrapResult{}, nil
	}

	generated := false
	if password == "" {
		password, err = randomPassword()
		if err != nil {
			return BootstrapResult{}, err
		}
		generated = true
	}

	hash, err := HashPassword(password)
	if err != nil {
		return BootstrapResult{}, err
	}

	if _, err := db.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'admin')`,
		DefaultAdminUsername, hash,
	); err != nil {
		// Otra instancia pudo crear el usuario entre la consulta y el insert.
		users, countErr := countUsers(ctx, db)
		if countErr == nil && users > 0 {
			return BootstrapResult{}, nil
		}
		return BootstrapResult{}, fmt.Errorf("auth: crear el administrador inicial: %w", err)
	}

	result := BootstrapResult{Created: true, Username: DefaultAdminUsername}
	if generated {
		result.GeneratedPassword = password
	}
	return result, nil
}

func countUsers(ctx context.Context, db *sql.DB) (int, error) {
	var users int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM users`).Scan(&users); err != nil {
		return 0, fmt.Errorf("auth: contar usuarios: %w", err)
	}
	return users, nil
}

func randomPassword() (string, error) {
	buf := make([]byte, bootstrapPasswordBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", errors.New("auth: no se pudo generar la contraseña del administrador")
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
