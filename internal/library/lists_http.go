package library

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"
)

func (a *API) handleItemListPut(w http.ResponseWriter, r *http.Request) {
	a.setItemList(w, r, true)
}

func (a *API) handleItemListDelete(w http.ResponseWriter, r *http.Request) {
	a.setItemList(w, r, false)
}

func (a *API) setItemList(w http.ResponseWriter, r *http.Request, add bool) {
	user, ok := sessionUserFromRequest(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "falta el token de sesión")
		return
	}
	list, ok := listFromPath(w, r)
	if !ok {
		return
	}
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	var exists int
	err := a.db.QueryRowContext(r.Context(), `SELECT 1 FROM media_items WHERE id = ?`, id).Scan(&exists)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	case err != nil:
		log.Printf("library: comprobar ítem %d para lista: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if add {
		_, err = a.db.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO item_lists (user_id, media_item_id, list) VALUES (?, ?, ?)`,
			user.ID, id, list)
	} else {
		_, err = a.db.ExecContext(r.Context(),
			`DELETE FROM item_lists WHERE user_id = ? AND media_item_id = ? AND list = ?`,
			user.ID, id, list)
	}
	if err != nil {
		log.Printf("library: actualizar lista %q del ítem %d: %v", list, id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleSeriesListPut(w http.ResponseWriter, r *http.Request) {
	a.setSeriesList(w, r, true)
}

func (a *API) handleSeriesListDelete(w http.ResponseWriter, r *http.Request) {
	a.setSeriesList(w, r, false)
}

func (a *API) setSeriesList(w http.ResponseWriter, r *http.Request, add bool) {
	user, ok := sessionUserFromRequest(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "falta el token de sesión")
		return
	}
	list, ok := listFromPath(w, r)
	if !ok {
		return
	}
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "serie no encontrada")
		return
	}
	var exists int
	err := a.db.QueryRowContext(r.Context(), `SELECT 1 FROM series WHERE id = ?`, id).Scan(&exists)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "serie no encontrada")
		return
	case err != nil:
		log.Printf("library: comprobar serie %d para lista: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if add {
		_, err = a.db.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO series_lists (user_id, series_id, list) VALUES (?, ?, ?)`,
			user.ID, id, list)
	} else {
		_, err = a.db.ExecContext(r.Context(),
			`DELETE FROM series_lists WHERE user_id = ? AND series_id = ? AND list = ?`,
			user.ID, id, list)
	}
	if err != nil {
		log.Printf("library: actualizar lista %q de la serie %d: %v", list, id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleCollectionListPut(w http.ResponseWriter, r *http.Request) {
	a.setCollectionList(w, r, true)
}

func (a *API) handleCollectionListDelete(w http.ResponseWriter, r *http.Request) {
	a.setCollectionList(w, r, false)
}

func (a *API) setCollectionList(w http.ResponseWriter, r *http.Request, add bool) {
	user, ok := sessionUserFromRequest(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "falta el token de sesión")
		return
	}
	list, ok := listFromPath(w, r)
	if !ok {
		return
	}
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "colección no encontrada")
		return
	}
	var exists int
	err := a.db.QueryRowContext(r.Context(),
		`SELECT 1 FROM collections WHERE id = ? AND deleted = 0`, id).Scan(&exists)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "colección no encontrada")
		return
	case err != nil:
		log.Printf("library: comprobar colección %d para lista: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if add {
		_, err = a.db.ExecContext(r.Context(),
			`INSERT OR IGNORE INTO collection_lists (user_id, collection_id, list) VALUES (?, ?, ?)`,
			user.ID, id, list)
	} else {
		_, err = a.db.ExecContext(r.Context(),
			`DELETE FROM collection_lists WHERE user_id = ? AND collection_id = ? AND list = ?`,
			user.ID, id, list)
	}
	if err != nil {
		log.Printf("library: actualizar lista %q de la colección %d: %v", list, id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func listFromPath(w http.ResponseWriter, r *http.Request) (string, bool) {
	list := strings.TrimSpace(r.PathValue("list"))
	if !validListName(list) {
		writeJSONError(w, http.StatusNotFound, "lista no encontrada")
		return "", false
	}
	return list, true
}

func (a *API) handleLists(w http.ResponseWriter, r *http.Request) {
	list, ok := listFromPath(w, r)
	if !ok {
		return
	}
	limit, offset, err := pagination(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	kind, err := listKindName(strings.TrimSpace(r.URL.Query().Get("kind")))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	userID := sessionUserID(r)

	totals, err := queryListTotals(r.Context(), a.db, userID, list)
	if err != nil {
		log.Printf("library: contar listas de %d: %v", userID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	response := listsResponse{
		Items:       make([]mediaItemDTO, 0),
		Series:      make([]seriesListDTO, 0),
		Collections: make([]collectionDTO, 0),
		Totals:      totals,
		Limit:       limit,
		Offset:      offset,
	}
	queryKind := kind
	if queryKind == "" {
		queryKind = "all"
		offset = 0
		response.Offset = 0
	}

	if queryKind == "all" || queryKind == "items" {
		items, total, err := queryListItems(r.Context(), a.db, userID, list, limit, offset)
		if err != nil {
			log.Printf("library: listar ítems de %q: %v", list, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		for _, item := range items {
			response.Items = append(response.Items, item.Item)
		}
		if kind == "items" {
			response.Total = &total
		}
	}
	if queryKind == "all" || queryKind == "series" {
		series, total, err := queryListSeries(r.Context(), a.db, userID, list, limit, offset)
		if err != nil {
			log.Printf("library: listar series de %q: %v", list, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		for _, item := range series {
			response.Series = append(response.Series, item.Series)
		}
		if kind == "series" {
			response.Total = &total
		}
	}
	if queryKind == "all" || queryKind == "collections" {
		collections, total, err := queryListCollections(r.Context(), a.db, userID, list, limit, offset)
		if err != nil {
			log.Printf("library: listar colecciones de %q: %v", list, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		for _, item := range collections {
			response.Collections = append(response.Collections, item.Collection)
		}
		if kind == "collections" {
			response.Total = &total
		}
	}
	writeJSON(w, http.StatusOK, response)
}
