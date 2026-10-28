package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// RoleAdmin y RoleUser son los únicos roles válidos.
	RoleAdmin = "admin"
	RoleUser  = "user"

	maxUsernameLength = 40
)

var (
	ErrInvalidUsername = errors.New("auth: el nombre de usuario debe tener entre 1 y 40 caracteres, sin espacios al principio ni al final ni caracteres de control")
	ErrUsernameTaken   = errors.New("auth: ya existe un usuario con ese nombre")
	ErrInvalidRole     = errors.New("auth: el rol debe ser admin o user")
	ErrUserNotFound    = errors.New("auth: usuario no encontrado")
	ErrLibraryNotFound = errors.New("auth: alguna de las bibliotecas indicadas no existe")
	ErrLastAdmin       = errors.New("auth: debe quedar al menos un administrador activo")
)

// UserChanges son los campos editables de una cuenta. Los nil no cambian.
type UserChanges struct {
	Role         *string
	Disabled     *bool
	AllLibraries *bool
}

// UserUpdate se conserva como alias de UserChanges.
type UserUpdate = UserChanges

type accountQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// ListUsers devuelve todas las cuentas con sus bibliotecas y sesiones activas.
func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT u.id, u.username, u.role, u.disabled, u.all_libraries, u.created_at,
		       (SELECT count(*) FROM sessions s
		         WHERE s.user_id = u.id AND s.expires_at > ?) AS active_sessions
		  FROM users u
		 ORDER BY u.username COLLATE NOCASE, u.id`, formatTime(s.now()))
	if err != nil {
		return nil, fmt.Errorf("auth: listar usuarios: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var user User
		var disabled, allLibraries int
		if err := rows.Scan(&user.ID, &user.Username, &user.Role, &disabled,
			&allLibraries, &user.CreatedAt, &user.ActiveSessions); err != nil {
			return nil, fmt.Errorf("auth: leer usuario: %w", err)
		}
		user.Disabled = disabled != 0
		user.AllLibraries = allLibraries != 0
		user.LibraryIDs = make([]int64, 0)
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("auth: recorrer usuarios: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("auth: cerrar el listado de usuarios: %w", err)
	}
	for i := range users {
		users[i].LibraryIDs, err = loadUserLibraryIDs(ctx, s.db, users[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return users, nil
}

// UserByID devuelve una cuenta con sus bibliotecas y sesiones activas.
func (s *Service) UserByID(ctx context.Context, id int64) (User, error) {
	return s.userByID(ctx, id)
}

// CreateUser crea una cuenta con sus bibliotecas.
func (s *Service) CreateUser(ctx context.Context, username, password, role string, allLibraries bool, libraryIDs []int64) (User, error) {
	username, err := validateAccountUsername(username)
	if err != nil {
		return User{}, err
	}
	if utf8.RuneCountInString(password) < minPasswordLength {
		return User{}, ErrPasswordTooShort
	}
	if !validRole(role) {
		return User{}, ErrInvalidRole
	}
	libraryIDs = uniqueLibraryIDs(libraryIDs)

	exists, err := usernameExists(ctx, s.db, username)
	if err != nil {
		return User{}, err
	}
	if exists {
		return User{}, ErrUsernameTaken
	}

	hash, err := HashPassword(password)
	if err != nil {
		return User{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("auth: iniciar creación de usuario: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	exists, err = usernameExists(ctx, tx, username)
	if err != nil {
		return User{}, err
	}
	if exists {
		return User{}, ErrUsernameTaken
	}
	if err := checkLibrariesExist(ctx, tx, libraryIDs); err != nil {
		return User{}, err
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO users (username, password_hash, role, all_libraries)
		VALUES (?, ?, ?, ?)`, username, hash, role, boolInt(allLibraries))
	if err != nil {
		if isUniqueConstraintError(err) {
			return User{}, ErrUsernameTaken
		}
		return User{}, fmt.Errorf("auth: crear usuario: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return User{}, fmt.Errorf("auth: id del usuario creado: %w", err)
	}
	if err := replaceUserLibraries(ctx, tx, id, libraryIDs); err != nil {
		return User{}, err
	}
	if err := tx.Commit(); err != nil {
		return User{}, fmt.Errorf("auth: confirmar creación de usuario: %w", err)
	}
	return s.UserByID(ctx, id)
}

// UpdateUser cambia el rol, el estado o el acceso global a bibliotecas.
func (s *Service) UpdateUser(ctx context.Context, id int64, changes UserChanges) (User, error) {
	if changes.Role != nil && !validRole(*changes.Role) {
		return User{}, ErrInvalidRole
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("auth: iniciar actualización de usuario: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var role string
	var disabled, allLibraries int
	err = tx.QueryRowContext(ctx,
		`SELECT role, disabled, all_libraries FROM users WHERE id = ?`, id,
	).Scan(&role, &disabled, &allLibraries)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return User{}, ErrUserNotFound
	case err != nil:
		return User{}, fmt.Errorf("auth: leer usuario: %w", err)
	}

	newRole := role
	if changes.Role != nil {
		newRole = *changes.Role
	}
	currentDisabled := disabled != 0
	newDisabled := currentDisabled
	if changes.Disabled != nil {
		newDisabled = *changes.Disabled
	}
	currentAllLibraries := allLibraries != 0
	newAllLibraries := currentAllLibraries
	if changes.AllLibraries != nil {
		newAllLibraries = *changes.AllLibraries
	}

	roleChanged := newRole != role
	disabledChanged := newDisabled != currentDisabled
	if roleChanged || disabledChanged || newAllLibraries != currentAllLibraries {
		if _, err := tx.ExecContext(ctx, `
			UPDATE users SET role = ?, disabled = ?, all_libraries = ? WHERE id = ?`,
			newRole, boolInt(newDisabled), boolInt(newAllLibraries), id); err != nil {
			return User{}, fmt.Errorf("auth: actualizar usuario: %w", err)
		}
	}

	activeAdmins, err := countActiveAdmins(ctx, tx)
	if err != nil {
		return User{}, err
	}
	if activeAdmins == 0 {
		return User{}, ErrLastAdmin
	}
	if roleChanged || (changes.Disabled != nil && newDisabled) {
		if err := deleteUserSessions(ctx, tx, id); err != nil {
			return User{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return User{}, fmt.Errorf("auth: confirmar actualización de usuario: %w", err)
	}
	return s.UserByID(ctx, id)
}

// SetUserLibraries reemplaza las bibliotecas asignadas a una cuenta.
func (s *Service) SetUserLibraries(ctx context.Context, id int64, libraryIDs []int64) (User, error) {
	libraryIDs = uniqueLibraryIDs(libraryIDs)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("auth: iniciar asignación de bibliotecas: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var exists int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM users WHERE id = ?`, id).Scan(&exists)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return User{}, ErrUserNotFound
	case err != nil:
		return User{}, fmt.Errorf("auth: comprobar usuario: %w", err)
	}
	if err := checkLibrariesExist(ctx, tx, libraryIDs); err != nil {
		return User{}, err
	}
	if err := replaceUserLibraries(ctx, tx, id, libraryIDs); err != nil {
		return User{}, err
	}
	if err := tx.Commit(); err != nil {
		return User{}, fmt.Errorf("auth: confirmar asignación de bibliotecas: %w", err)
	}
	return s.UserByID(ctx, id)
}

// ResetPassword cambia la contraseña de una cuenta y cierra todas sus sesiones.
func (s *Service) ResetPassword(ctx context.Context, id int64, newPassword string) error {
	if utf8.RuneCountInString(newPassword) < minPasswordLength {
		return ErrPasswordTooShort
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("auth: iniciar restablecimiento de contraseña: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var exists int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM users WHERE id = ?`, id).Scan(&exists)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrUserNotFound
	case err != nil:
		return fmt.Errorf("auth: comprobar usuario: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET password_hash = ? WHERE id = ?`, hash, id,
	); err != nil {
		return fmt.Errorf("auth: restablecer contraseña: %w", err)
	}
	if err := deleteUserSessions(ctx, tx, id); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("auth: confirmar restablecimiento de contraseña: %w", err)
	}
	return nil
}

// DeleteUser borra una cuenta; las tablas con user_id caen por cascada.
func (s *Service) DeleteUser(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("auth: iniciar borrado de usuario: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var role string
	var disabled int
	err = tx.QueryRowContext(ctx,
		`SELECT role, disabled FROM users WHERE id = ?`, id,
	).Scan(&role, &disabled)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return ErrUserNotFound
	case err != nil:
		return fmt.Errorf("auth: leer usuario: %w", err)
	}

	if err := deleteUserSessions(ctx, tx, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id); err != nil {
		return fmt.Errorf("auth: borrar usuario: %w", err)
	}
	activeAdmins, err := countActiveAdmins(ctx, tx)
	if err != nil {
		return err
	}
	if activeAdmins == 0 {
		return ErrLastAdmin
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("auth: confirmar borrado de usuario: %w", err)
	}
	return nil
}

func (s *Service) userByID(ctx context.Context, id int64) (User, error) {
	var user User
	var disabled, allLibraries int
	err := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.username, u.role, u.disabled, u.all_libraries, u.created_at,
		       (SELECT count(*) FROM sessions s
		         WHERE s.user_id = u.id AND s.expires_at > ?) AS active_sessions
		  FROM users u
		 WHERE u.id = ?`, formatTime(s.now()), id,
	).Scan(&user.ID, &user.Username, &user.Role, &disabled,
		&allLibraries, &user.CreatedAt, &user.ActiveSessions)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return User{}, ErrUserNotFound
	case err != nil:
		return User{}, fmt.Errorf("auth: leer usuario: %w", err)
	}
	user.Disabled = disabled != 0
	user.AllLibraries = allLibraries != 0
	user.LibraryIDs, err = loadUserLibraryIDs(ctx, s.db, id)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func validateAccountUsername(raw string) (string, error) {
	if !utf8.ValidString(raw) || raw != strings.TrimSpace(raw) {
		return "", ErrInvalidUsername
	}
	length := utf8.RuneCountInString(raw)
	if length < 1 || length > maxUsernameLength {
		return "", ErrInvalidUsername
	}
	for _, char := range raw {
		if unicode.IsControl(char) {
			return "", ErrInvalidUsername
		}
	}
	return raw, nil
}

func validRole(role string) bool {
	return role == RoleAdmin || role == RoleUser
}

func usernameExists(ctx context.Context, q accountQuerier, username string) (bool, error) {
	var exists int
	err := q.QueryRowContext(ctx,
		`SELECT 1 FROM users WHERE username = ?`, username,
	).Scan(&exists)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return false, nil
	case err != nil:
		return false, fmt.Errorf("auth: comprobar nombre de usuario: %w", err)
	default:
		return true, nil
	}
}

func uniqueLibraryIDs(ids []int64) []int64 {
	unique := make([]int64, 0, len(ids))
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		unique = append(unique, id)
	}
	return unique
}

func checkLibrariesExist(ctx context.Context, q accountQuerier, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	args := make([]any, len(ids))
	placeholders := make([]string, len(ids))
	for i, id := range ids {
		args[i] = id
		placeholders[i] = "?"
	}
	rows, err := q.QueryContext(ctx,
		`SELECT id FROM libraries WHERE id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return fmt.Errorf("auth: comprobar bibliotecas: %w", err)
	}
	defer rows.Close()

	found := make(map[int64]bool, len(ids))
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("auth: leer biblioteca: %w", err)
		}
		found[id] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("auth: recorrer bibliotecas: %w", err)
	}
	for _, id := range ids {
		if !found[id] {
			return ErrLibraryNotFound
		}
	}
	return nil
}

func replaceUserLibraries(ctx context.Context, tx *sql.Tx, userID int64, ids []int64) error {
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM user_libraries WHERE user_id = ?`, userID,
	); err != nil {
		return fmt.Errorf("auth: limpiar bibliotecas del usuario: %w", err)
	}
	for _, libraryID := range ids {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO user_libraries (user_id, library_id) VALUES (?, ?)`,
			userID, libraryID,
		); err != nil {
			return fmt.Errorf("auth: asignar biblioteca al usuario: %w", err)
		}
	}
	return nil
}

func loadUserLibraryIDs(ctx context.Context, q accountQuerier, userID int64) ([]int64, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT library_id FROM user_libraries WHERE user_id = ? ORDER BY library_id`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("auth: listar bibliotecas del usuario: %w", err)
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("auth: leer biblioteca del usuario: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("auth: recorrer bibliotecas del usuario: %w", err)
	}
	return ids, nil
}

func countActiveAdmins(ctx context.Context, tx *sql.Tx) (int, error) {
	var count int
	if err := tx.QueryRowContext(ctx,
		`SELECT count(*) FROM users WHERE role = ? AND disabled = 0`, RoleAdmin,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("auth: contar administradores activos: %w", err)
	}
	return count, nil
}

func deleteUserSessions(ctx context.Context, tx *sql.Tx, userID int64) error {
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM sessions WHERE user_id = ?`, userID,
	); err != nil {
		return fmt.Errorf("auth: cerrar sesiones del usuario: %w", err)
	}
	return nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func isUniqueConstraintError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}
