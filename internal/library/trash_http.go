package library

import (
	"errors"
	"log"
	"net/http"
)

func trashActor(r *http.Request) string {
	user, ok := sessionUserFromRequest(r)
	if !ok {
		return "desconocido"
	}
	return user.Username
}

func (a *API) handleItemTrash(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "identificador no válido")
		return
	}
	item, err := a.trashItem(r.Context(), id, trashActor(r))
	if err != nil {
		writeTrashError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) handleTrashList(w http.ResponseWriter, r *http.Request) {
	items, err := a.listTrash(r.Context())
	if err != nil {
		log.Printf("library: listar la papelera: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, trashListResponse{Items: items})
}

func (a *API) handleTrashRestore(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "identificador no válido")
		return
	}
	itemID, err := a.restoreTrash(r.Context(), id, trashActor(r))
	if err != nil {
		writeTrashError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"item_id": itemID})
}

func (a *API) handleTrashDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "identificador no válido")
		return
	}
	if err := a.deleteTrashItem(r.Context(), id, trashActor(r)); err != nil {
		writeTrashError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleTrashPurge(w http.ResponseWriter, r *http.Request) {
	count, err := a.purgeExpiredTrash(r.Context(), trashActor(r))
	if err != nil {
		log.Printf("library: purgar la papelera: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"purged": count})
}

func writeTrashError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errTrashDisabled):
		writeJSONError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, errTrashMusic), errors.Is(err, errTrashSymlink),
		errors.Is(err, errTrashUnsafe):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, errItemNotFound), errors.Is(err, errTrashNotFound),
		errors.Is(err, errTrashFileGone):
		writeJSONError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, errTrashConflict):
		writeJSONError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("library: papelera: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
	}
}
