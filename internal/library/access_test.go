package library

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/catlinux/cinecat/internal/auth"
)

func TestLibraryAccessClausulas(t *testing.T) {
	empty := libraryAccess{}
	if empty.Unrestricted() || empty.Allows(1) || empty.Clause("mi") != "0 = 1" ||
		empty.LibraryClause("l") != "0 = 1" {
		t.Fatalf("el acceso vacío debe denegar todo: %+v", empty)
	}

	restricted := newLibraryAccess([]int64{3, 1, 3, 0, -1})
	if restricted.Unrestricted() || !restricted.Allows(1) || !restricted.Allows(3) ||
		restricted.Allows(2) {
		t.Fatalf("acceso restringido inesperado: %+v", restricted)
	}
	if got := restricted.Clause("mi"); got != "mi.library_id IN (1,3)" {
		t.Fatalf("Clause = %q", got)
	}
	if got := restricted.LibraryClause("l"); got != "l.id IN (1,3)" {
		t.Fatalf("LibraryClause = %q", got)
	}
	if got := restricted.VideoClause("mi"); !strings.Contains(got, "kind = 'video'") ||
		!strings.Contains(got, "mi.library_id IN (1,3)") {
		t.Fatalf("VideoClause = %q", got)
	}
	if got := restricted.ClipsClause("mi"); !strings.Contains(got, "kind = 'clips'") {
		t.Fatalf("ClipsClause = %q", got)
	}
	if got := restricted.MusicClause("mi"); !strings.Contains(got, "kind = 'music'") {
		t.Fatalf("MusicClause = %q", got)
	}

	all := unrestrictedLibraryAccess()
	if !all.Unrestricted() || !all.Allows(999999) || all.Clause("mi") != "1 = 1" ||
		all.LibraryClause("l") != "1 = 1" {
		t.Fatalf("acceso sin restricción inesperado: %+v", all)
	}
}

func TestAccessFromRequestSinContextoDeniega(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	access := accessFromRequest(req)
	if access.Unrestricted() || access.Allows(1) || access.Clause("mi") != "0 = 1" {
		t.Fatalf("sin contexto debe denegarse: %+v", access)
	}
}

func partialAccessToken(t *testing.T, ts *libraryTestServer, libraryIDs ...int64) (string, int64) {
	t.Helper()
	hash, err := auth.HashPassword("Clave-parcial-1")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	result, err := ts.db.Exec(`
		INSERT INTO users (username, password_hash, role, all_libraries)
		VALUES ('parcial', ?, 'user', 0)`, hash)
	if err != nil {
		t.Fatalf("crear usuario parcial: %v", err)
	}
	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("id de usuario parcial: %v", err)
	}
	for _, libraryID := range libraryIDs {
		if _, err := ts.db.Exec(
			`INSERT INTO user_libraries (user_id, library_id) VALUES (?, ?)`,
			userID, libraryID); err != nil {
			t.Fatalf("asignar biblioteca %d: %v", libraryID, err)
		}
	}
	token, _, err := ts.auth.Login(context.Background(), "parcial", "Clave-parcial-1")
	if err != nil {
		t.Fatalf("Login parcial: %v", err)
	}
	return token, userID
}
