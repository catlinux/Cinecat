package library

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/catlinux/cinecat/internal/auth"
)

// libraryAccess es el conjunto de bibliotecas que puede ver una petición.
// El valor cero no da acceso a nada: si falta en el contexto se deniega todo.
type libraryAccess struct {
	unrestricted bool
	ids          []int64
	set          map[int64]struct{}
}

func unrestrictedLibraryAccess() libraryAccess {
	return libraryAccess{unrestricted: true}
}

func newLibraryAccess(ids []int64) libraryAccess {
	unique := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	sort.Slice(unique, func(i, j int) bool { return unique[i] < unique[j] })
	return libraryAccess{ids: unique, set: seen}
}

// loadLibraryAccess lee el acceso efectivo del usuario una vez por petición.
// El administrador y los usuarios con all_libraries ven todo; el resto solo
// las bibliotecas enlazadas en user_libraries.
func loadLibraryAccess(ctx context.Context, db *sql.DB, user auth.User) (libraryAccess, error) {
	if user.Role == auth.RoleAdmin {
		return unrestrictedLibraryAccess(), nil
	}

	rows, err := db.QueryContext(ctx, `
		SELECT u.all_libraries, ul.library_id
		  FROM users u
		  LEFT JOIN user_libraries ul ON ul.user_id = u.id
		 WHERE u.id = ?`, user.ID)
	if err != nil {
		return libraryAccess{}, err
	}
	defer rows.Close()

	found := false
	allLibraries := false
	ids := make([]int64, 0)
	for rows.Next() {
		found = true
		var all int
		var libraryID sql.NullInt64
		if err := rows.Scan(&all, &libraryID); err != nil {
			return libraryAccess{}, err
		}
		allLibraries = all != 0
		if libraryID.Valid {
			ids = append(ids, libraryID.Int64)
		}
	}
	if err := rows.Err(); err != nil {
		return libraryAccess{}, err
	}
	if !found {
		return libraryAccess{}, errors.New("library: usuario sin acceso")
	}
	if allLibraries {
		return unrestrictedLibraryAccess(), nil
	}
	return newLibraryAccess(ids), nil
}

func (a libraryAccess) Unrestricted() bool {
	return a.unrestricted
}

func (a libraryAccess) Allows(libraryID int64) bool {
	if a.unrestricted {
		return true
	}
	_, ok := a.set[libraryID]
	return ok
}

// Clause devuelve una condición sobre library_id con los ids como literales
// enteros, nunca con texto procedente de la petición.
func (a libraryAccess) Clause(alias string) string {
	if a.unrestricted {
		return "1 = 1"
	}
	if len(a.ids) == 0 {
		return "0 = 1"
	}
	parts := make([]string, 0, len(a.ids))
	for _, id := range a.ids {
		parts = append(parts, strconv.FormatInt(id, 10))
	}
	return alias + ".library_id IN (" + strings.Join(parts, ",") + ")"
}

// LibraryClause es como Clause pero para la tabla libraries, cuyo id no se
// llama library_id.
func (a libraryAccess) LibraryClause(alias string) string {
	if a.unrestricted {
		return "1 = 1"
	}
	if len(a.ids) == 0 {
		return "0 = 1"
	}
	parts := make([]string, 0, len(a.ids))
	for _, id := range a.ids {
		parts = append(parts, strconv.FormatInt(id, 10))
	}
	return alias + ".id IN (" + strings.Join(parts, ",") + ")"
}

func (a libraryAccess) VideoClause(alias string) string {
	return "(" + alias + ".library_id IN (SELECT id FROM libraries WHERE kind = 'video')) AND (" + a.Clause(alias) + ")"
}

func (a libraryAccess) ClipsClause(alias string) string {
	return "(" + alias + ".library_id IN (SELECT id FROM libraries WHERE kind = 'clips')) AND (" + a.Clause(alias) + ")"
}

func (a libraryAccess) MusicClause(alias string) string {
	return "(" + alias + ".library_id IN (SELECT id FROM libraries WHERE kind = 'music')) AND (" + a.Clause(alias) + ")"
}

type libraryAccessKey struct{}

func withLibraryAccess(ctx context.Context, access libraryAccess) context.Context {
	return context.WithValue(ctx, libraryAccessKey{}, access)
}

// accessFromRequest devuelve el acceso calculado por el middleware. Si no
// existe, devuelve el valor cero, que no permite ver ninguna biblioteca.
func accessFromRequest(r *http.Request) libraryAccess {
	access, ok := r.Context().Value(libraryAccessKey{}).(libraryAccess)
	if !ok {
		return libraryAccess{}
	}
	return access
}
