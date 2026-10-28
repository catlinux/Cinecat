package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

func createAuthLibrary(t *testing.T, db *sql.DB, path, name string) int64 {
	t.Helper()
	result, err := db.Exec(`INSERT INTO libraries (path, name) VALUES (?, ?)`, path, name)
	if err != nil {
		t.Fatalf("crear biblioteca: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de biblioteca: %v", err)
	}
	return id
}

func strPtr(value string) *string { return &value }
func boolPtr(value bool) *bool    { return &value }

func TestCreateUserValidaYGuardaBibliotecas(t *testing.T) {
	svc, db, _ := newTestService(t)
	ctx := context.Background()
	first := createAuthLibrary(t, db, "/media/primera", "Primera")
	second := createAuthLibrary(t, db, "/media/segunda", "Segunda")

	user, err := svc.CreateUser(ctx, "Ana", "Clave-segura-1", RoleUser, false,
		[]int64{second, first, second})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if user.ID == 0 || user.Username != "Ana" || user.Role != RoleUser ||
		user.Disabled || user.AllLibraries || user.CreatedAt == "" {
		t.Fatalf("usuario creado inesperado: %+v", user)
	}
	if len(user.LibraryIDs) != 2 || user.LibraryIDs[0] != first || user.LibraryIDs[1] != second {
		t.Fatalf("bibliotecas = %v, se esperaban [%d %d]", user.LibraryIDs, first, second)
	}

	if _, err := svc.CreateUser(ctx, "ana", "Clave-segura-1", RoleUser, false, nil); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("duplicado con otra capitalización: %v", err)
	}
	if _, err := svc.CreateUser(ctx, "bea", "corta", RoleUser, false, nil); !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("contraseña corta: %v", err)
	}
	if _, err := svc.CreateUser(ctx, "bea", "Clave-segura-1", "editor", false, nil); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("rol inválido: %v", err)
	}
	if _, err := svc.CreateUser(ctx, "bea", "Clave-segura-1", RoleUser, false, []int64{999999}); !errors.Is(err, ErrLibraryNotFound) {
		t.Fatalf("biblioteca inexistente: %v", err)
	}
	for _, invalid := range []string{"", " ana", "ana ", strings.Repeat("a", maxUsernameLength+1), "ana\x00"} {
		if _, err := svc.CreateUser(ctx, invalid, "Clave-segura-1", RoleUser, false, nil); !errors.Is(err, ErrInvalidUsername) {
			t.Fatalf("nombre %q: %v", invalid, err)
		}
	}

	users, err := svc.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 1 || users[0].ID != user.ID || users[0].ActiveSessions != 0 {
		t.Fatalf("listado inesperado: %+v", users)
	}
}

func TestUpdateUserCierraSesionesYDesactiva(t *testing.T) {
	svc, db, _ := newTestService(t)
	ctx := context.Background()
	createUser(t, db, "root", "Clave-root-1", RoleAdmin)
	anaID := createUser(t, db, "ana", "Clave-vieja-1", RoleUser)

	first, _, err := svc.Login(ctx, "ana", "Clave-vieja-1")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	second, _, err := svc.Login(ctx, "ana", "Clave-vieja-1")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	user, err := svc.UpdateUser(ctx, anaID, UserChanges{Role: strPtr(RoleAdmin)})
	if err != nil {
		t.Fatalf("cambiar rol: %v", err)
	}
	if user.Role != RoleAdmin {
		t.Fatalf("rol = %q", user.Role)
	}
	for _, token := range []string{first, second} {
		if _, err := svc.Authenticate(ctx, token); !errors.Is(err, ErrInvalidSession) {
			t.Fatalf("el cambio de rol debe cerrar sesiones: %v", err)
		}
	}

	token, _, err := svc.Login(ctx, "ana", "Clave-vieja-1")
	if err != nil {
		t.Fatalf("login tras cambio de rol: %v", err)
	}
	if _, err := svc.UpdateUser(ctx, anaID, UserChanges{AllLibraries: boolPtr(true)}); err != nil {
		t.Fatalf("cambiar all_libraries: %v", err)
	}
	if _, err := svc.Authenticate(ctx, token); err != nil {
		t.Fatalf("cambiar all_libraries no debe cerrar sesiones: %v", err)
	}

	if _, err := svc.UpdateUser(ctx, anaID, UserChanges{Disabled: boolPtr(true)}); err != nil {
		t.Fatalf("desactivar: %v", err)
	}
	if _, err := svc.Authenticate(ctx, token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("desactivar debe cerrar sesiones: %v", err)
	}
	if _, _, err := svc.Login(ctx, "ana", "Clave-vieja-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("login desactivado: %v", err)
	}
	if _, err := svc.UpdateUser(ctx, anaID, UserChanges{Disabled: boolPtr(false)}); err != nil {
		t.Fatalf("reactivar: %v", err)
	}
	if _, _, err := svc.Login(ctx, "ana", "Clave-vieja-1"); err != nil {
		t.Fatalf("login reactivado: %v", err)
	}

	if _, err := svc.UpdateUser(ctx, 999999, UserChanges{}); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("usuario inexistente: %v", err)
	}
	if _, err := svc.UpdateUser(ctx, anaID, UserChanges{Role: strPtr("otro")}); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("rol inválido: %v", err)
	}
}

func TestSiempreQuedaUnAdminActivo(t *testing.T) {
	svc, db, _ := newTestService(t)
	ctx := context.Background()
	adminID := createUser(t, db, "root", "Clave-root-1", RoleAdmin)

	if _, err := svc.UpdateUser(ctx, adminID, UserChanges{Disabled: boolPtr(true)}); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("desactivar al único admin: %v", err)
	}
	if _, err := svc.UpdateUser(ctx, adminID, UserChanges{Role: strPtr(RoleUser)}); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("quitar el rol al único admin: %v", err)
	}
	if err := svc.DeleteUser(ctx, adminID); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("borrar al único admin: %v", err)
	}

	var role string
	var disabled int
	if err := db.QueryRow(`SELECT role, disabled FROM users WHERE id = ?`, adminID).
		Scan(&role, &disabled); err != nil {
		t.Fatalf("leer admin: %v", err)
	}
	if role != RoleAdmin || disabled != 0 {
		t.Fatalf("el admin cambió pese al error: role=%q disabled=%d", role, disabled)
	}
}

func TestResetPasswordCierraTodasLasSesiones(t *testing.T) {
	svc, db, _ := newTestService(t)
	ctx := context.Background()
	createUser(t, db, "root", "Clave-root-1", RoleAdmin)
	beaID := createUser(t, db, "bea", "Clave-vieja-1", RoleUser)

	first, _, err := svc.Login(ctx, "bea", "Clave-vieja-1")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	second, _, err := svc.Login(ctx, "bea", "Clave-vieja-1")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := svc.ResetPassword(ctx, beaID, "corta"); !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("contraseña corta: %v", err)
	}
	if _, err := svc.Authenticate(ctx, first); err != nil {
		t.Fatalf("un error de validación no debe cerrar sesiones: %v", err)
	}

	if err := svc.ResetPassword(ctx, beaID, "Clave-nueva-1"); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	for _, token := range []string{first, second} {
		if _, err := svc.Authenticate(ctx, token); !errors.Is(err, ErrInvalidSession) {
			t.Fatalf("la sesión debe quedar cerrada: %v", err)
		}
	}
	if _, _, err := svc.Login(ctx, "bea", "Clave-vieja-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("contraseña antigua: %v", err)
	}
	if _, _, err := svc.Login(ctx, "bea", "Clave-nueva-1"); err != nil {
		t.Fatalf("contraseña nueva: %v", err)
	}
	if err := svc.ResetPassword(ctx, 999999, "Clave-nueva-1"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("usuario inexistente: %v", err)
	}
}

func TestLoginDesactivadoNoCuentaBloqueo(t *testing.T) {
	svc, db, _ := newTestService(t)
	ctx := context.Background()
	createUser(t, db, "root", "Clave-root-1", RoleAdmin)
	anaID := createUser(t, db, "ana", "Clave-vieja-1", RoleUser)

	token, _, err := svc.Login(ctx, "ana", "Clave-vieja-1")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if _, err := svc.UpdateUser(ctx, anaID, UserChanges{Disabled: boolPtr(true)}); err != nil {
		t.Fatalf("desactivar: %v", err)
	}
	if _, err := svc.Authenticate(ctx, token); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("la sesión de un desactivado debe ser inválida: %v", err)
	}

	for i := 0; i < DefaultMaxAttempts+2; i++ {
		if _, _, err := svc.LoginFrom(ctx, "ana", "incorrecta", "198.51.100.70"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("intento %d: %v", i+1, err)
		}
	}
	if _, _, err := svc.LoginFrom(ctx, "root", "Clave-root-1", "198.51.100.70"); err != nil {
		t.Fatalf("el desactivado no debe bloquear a otro usuario: %v", err)
	}

	if _, err := svc.UpdateUser(ctx, anaID, UserChanges{Disabled: boolPtr(false)}); err != nil {
		t.Fatalf("reactivar: %v", err)
	}
	reactivated, _, err := svc.Login(ctx, "ana", "Clave-vieja-1")
	if err != nil {
		t.Fatalf("login reactivado: %v", err)
	}
	if _, err := db.Exec(`UPDATE users SET disabled = 1 WHERE id = ?`, anaID); err != nil {
		t.Fatalf("desactivar sin cerrar sesiones: %v", err)
	}
	if _, err := svc.Authenticate(ctx, reactivated); !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("Authenticate debe rechazar al desactivado: %v", err)
	}
}

func TestDeleteUserEliminaDatosPorCascada(t *testing.T) {
	svc, db, _ := newTestService(t)
	ctx := context.Background()
	createUser(t, db, "root", "Clave-root-1", RoleAdmin)
	beaID := createUser(t, db, "bea", "Clave-vieja-1", RoleUser)
	libraryID := createAuthLibrary(t, db, "/media/bea", "Bea")
	if _, err := svc.SetUserLibraries(ctx, beaID, []int64{libraryID}); err != nil {
		t.Fatalf("SetUserLibraries: %v", err)
	}
	if _, _, err := svc.Login(ctx, "bea", "Clave-vieja-1"); err != nil {
		t.Fatalf("login: %v", err)
	}
	itemResult, err := db.Exec(`
		INSERT INTO media_items (library_id, path, title, kind)
		VALUES (?, '/media/bea/peli.mkv', 'Peli', 'movie')`, libraryID)
	if err != nil {
		t.Fatalf("crear ítem: %v", err)
	}
	itemID, _ := itemResult.LastInsertId()
	for _, statement := range []string{
		`INSERT INTO playback_state (user_id, media_item_id) VALUES (?, ?)`,
		`INSERT INTO playback_progress (user_id, media_item_id, position_seconds, duration_seconds) VALUES (?, ?, 1, 2)`,
		`INSERT INTO watch_state (user_id, media_item_id) VALUES (?, ?)`,
		`INSERT INTO item_preferences (user_id, media_item_id) VALUES (?, ?)`,
		`INSERT INTO item_lists (user_id, media_item_id, list) VALUES (?, ?, 'favorite')`,
	} {
		if _, err := db.Exec(statement, beaID, itemID); err != nil {
			t.Fatalf("preparar datos de usuario: %v", err)
		}
	}

	if err := svc.DeleteUser(ctx, beaID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	for table, query := range map[string]string{
		"users":             `SELECT count(*) FROM users WHERE id = ?`,
		"sessions":          `SELECT count(*) FROM sessions WHERE user_id = ?`,
		"user_libraries":    `SELECT count(*) FROM user_libraries WHERE user_id = ?`,
		"playback_state":    `SELECT count(*) FROM playback_state WHERE user_id = ?`,
		"playback_progress": `SELECT count(*) FROM playback_progress WHERE user_id = ?`,
		"watch_state":       `SELECT count(*) FROM watch_state WHERE user_id = ?`,
		"item_preferences":  `SELECT count(*) FROM item_preferences WHERE user_id = ?`,
		"item_lists":        `SELECT count(*) FROM item_lists WHERE user_id = ?`,
	} {
		var count int
		if err := db.QueryRow(query, beaID).Scan(&count); err != nil {
			t.Fatalf("contar %s: %v", table, err)
		}
		if count != 0 {
			t.Errorf("%s: quedan %d filas", table, count)
		}
	}
	if err := svc.DeleteUser(ctx, beaID); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("borrar dos veces: %v", err)
	}
}
