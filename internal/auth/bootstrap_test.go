package auth

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/catlinux/cinecat/internal/store"
)

func openBootstrapDB(t *testing.T) *sql.DB {
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
	return st.DB()
}

func countUsersInDB(t *testing.T, db *sql.DB) int {
	t.Helper()
	var users int
	if err := db.QueryRow(`SELECT count(*) FROM users`).Scan(&users); err != nil {
		t.Fatalf("contar usuarios: %v", err)
	}
	return users
}

func TestBootstrapAdminBaseVacia(t *testing.T) {
	ctx := context.Background()
	db := openBootstrapDB(t)

	result, err := BootstrapAdmin(ctx, db, "")
	if err != nil {
		t.Fatalf("BootstrapAdmin: %v", err)
	}
	if !result.Created || result.Username != DefaultAdminUsername {
		t.Fatalf("resultado inesperado: %+v", result)
	}
	if len(result.GeneratedPassword) < 20 {
		t.Fatalf("contraseña generada demasiado corta: %d caracteres", len(result.GeneratedPassword))
	}
	if countUsersInDB(t, db) != 1 {
		t.Fatalf("se esperaba exactamente un usuario")
	}

	var username, role, hash string
	if err := db.QueryRow(`SELECT username, role, password_hash FROM users`).
		Scan(&username, &role, &hash); err != nil {
		t.Fatalf("leer usuario: %v", err)
	}
	if username != DefaultAdminUsername || role != "admin" {
		t.Fatalf("usuario inesperado: %q, %q", username, role)
	}
	ok, err := VerifyPassword(hash, result.GeneratedPassword)
	if err != nil || !ok {
		t.Fatalf("la contraseña generada no valida contra el hash: %v, %v", ok, err)
	}

	// Dos bases nuevas no deben producir la misma contraseña.
	other := openBootstrapDB(t)
	otherResult, err := BootstrapAdmin(ctx, other, "")
	if err != nil {
		t.Fatalf("BootstrapAdmin en otra base: %v", err)
	}
	if otherResult.GeneratedPassword == result.GeneratedPassword {
		t.Fatal("dos contraseñas generadas no deben coincidir")
	}
}

func TestBootstrapSegundoArranqueNoCambiaNada(t *testing.T) {
	ctx := context.Background()
	db := openBootstrapDB(t)

	first, err := BootstrapAdmin(ctx, db, "")
	if err != nil {
		t.Fatalf("primer arranque: %v", err)
	}
	if !first.Created {
		t.Fatal("el primer arranque debe crear el administrador")
	}

	var hashBefore string
	if err := db.QueryRow(`SELECT password_hash FROM users`).Scan(&hashBefore); err != nil {
		t.Fatalf("leer hash: %v", err)
	}

	second, err := BootstrapAdmin(ctx, db, "")
	if err != nil {
		t.Fatalf("segundo arranque: %v", err)
	}
	if second.Created || second.GeneratedPassword != "" {
		t.Fatalf("el segundo arranque no debe crear nada: %+v", second)
	}

	// Aunque se dé otra contraseña por entorno, no se modifica el usuario.
	third, err := BootstrapAdmin(ctx, db, "Otra-clave-123")
	if err != nil {
		t.Fatalf("tercer arranque: %v", err)
	}
	if third.Created {
		t.Fatalf("no debe crearse otro administrador: %+v", third)
	}

	if countUsersInDB(t, db) != 1 {
		t.Fatal("el segundo arranque no debe crear otro usuario")
	}
	var hashAfter string
	if err := db.QueryRow(`SELECT password_hash FROM users`).Scan(&hashAfter); err != nil {
		t.Fatalf("leer hash: %v", err)
	}
	if hashAfter != hashBefore {
		t.Fatal("el segundo arranque no debe cambiar la contraseña")
	}
	if ok, _ := VerifyPassword(hashAfter, "Otra-clave-123"); ok {
		t.Fatal("la contraseña del entorno no debe aplicarse si ya hay usuarios")
	}
}

func TestBootstrapAdminConPasswordDeEntorno(t *testing.T) {
	ctx := context.Background()
	db := openBootstrapDB(t)

	const password = "Clave-admin-de-prueba-123"
	result, err := BootstrapAdmin(ctx, db, password)
	if err != nil {
		t.Fatalf("BootstrapAdmin: %v", err)
	}
	if !result.Created || result.GeneratedPassword != "" {
		t.Fatalf("resultado inesperado: %+v", result)
	}

	var hash string
	if err := db.QueryRow(`SELECT password_hash FROM users`).Scan(&hash); err != nil {
		t.Fatalf("leer hash: %v", err)
	}
	if ok, err := VerifyPassword(hash, password); err != nil || !ok {
		t.Fatalf("la contraseña del entorno no valida: %v, %v", ok, err)
	}
	if ok, _ := VerifyPassword(hash, result.GeneratedPassword); ok {
		t.Fatal("no se generó contraseña, no debe validar una vacía")
	}
}

func TestBootstrapNoTocaUsuariosExistentes(t *testing.T) {
	ctx := context.Background()
	db := openBootstrapDB(t)

	hash, err := HashPassword("Clave-usuario-123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO users (username, password_hash, role) VALUES (?, ?, 'user')`,
		"ana", hash,
	); err != nil {
		t.Fatalf("crear usuario: %v", err)
	}

	result, err := BootstrapAdmin(ctx, db, "")
	if err != nil {
		t.Fatalf("BootstrapAdmin: %v", err)
	}
	if result.Created {
		t.Fatalf("no debe crear un administrador si ya hay usuarios: %+v", result)
	}
	if countUsersInDB(t, db) != 1 {
		t.Fatal("no debe añadir usuarios")
	}
	var role string
	if err := db.QueryRow(`SELECT role FROM users WHERE username = 'ana'`).Scan(&role); err != nil {
		t.Fatalf("leer rol: %v", err)
	}
	if role != "user" {
		t.Fatalf("el rol existente no debe cambiar: %q", role)
	}
}
