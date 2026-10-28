package library

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
)

// visibleItem es la información mínima que necesita un handler para decidir si
// puede servir o modificar un ítem.
type visibleItem struct {
	ID          int64
	LibraryID   int64
	LibraryKind string
}

// requireVisibleItem devuelve 404, con el mismo mensaje que un ítem
// inexistente, si el ítem o la serie de un episodio no son visibles.
func (a *API) requireVisibleItem(w http.ResponseWriter, r *http.Request, id int64) (visibleItem, bool) {
	access := accessFromRequest(r)
	var item visibleItem
	var seriesID, seriesLibraryID sql.NullInt64
	err := a.db.QueryRowContext(r.Context(), `
		SELECT mi.library_id, l.kind, mi.series_id, s.library_id
		  FROM media_items mi
		  JOIN libraries l ON l.id = mi.library_id
		  LEFT JOIN series s ON s.id = mi.series_id
		 WHERE mi.id = ?`, id).
		Scan(&item.LibraryID, &item.LibraryKind, &seriesID, &seriesLibraryID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return visibleItem{}, false
	case err != nil:
		log.Printf("library: comprobar acceso al ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return visibleItem{}, false
	}
	if !access.Allows(item.LibraryID) {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return visibleItem{}, false
	}
	if seriesID.Valid {
		if !seriesLibraryID.Valid || !access.Allows(seriesLibraryID.Int64) {
			writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
			return visibleItem{}, false
		}
	}
	item.ID = id
	return item, true
}
