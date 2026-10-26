package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	defaultCollectionsLimit = 48
	maxCollectionsLimit     = 200
)

const collectionItemCountExpr = `(SELECT count(*) FROM collection_items ci
	WHERE ci.collection_id = c.id AND ci.excluded = 0)`

const collectionPosterExpr = `COALESCE(NULLIF(c.poster_url, ''), (
	SELECT mi.poster_url
	  FROM collection_items ci
	  JOIN media_items mi ON mi.id = ci.media_item_id
	 WHERE ci.collection_id = c.id
	   AND ci.excluded = 0
	   AND mi.poster_url IS NOT NULL
	   AND trim(mi.poster_url) <> ''
	 ORDER BY (mi.year IS NULL) ASC, mi.year ASC, mi.id ASC
	 LIMIT 1))`

type collectionDTO struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Overview    *string `json:"overview"`
	PosterURL   *string `json:"poster_url"`
	BackdropURL *string `json:"backdrop_url"`
	ItemCount   int64   `json:"item_count"`
	Source      string  `json:"source"`
	Edited      bool    `json:"edited"`
}

type collectionDetailDTO struct {
	ID          int64          `json:"id"`
	Name        string         `json:"name"`
	Overview    *string        `json:"overview"`
	PosterURL   *string        `json:"poster_url"`
	BackdropURL *string        `json:"backdrop_url"`
	ItemCount   int64          `json:"item_count"`
	Source      string         `json:"source"`
	Edited      bool           `json:"edited"`
	Items       []mediaItemDTO `json:"items"`
}

type collectionsResponse struct {
	Collections []collectionDTO `json:"collections"`
	Total       int64           `json:"total"`
	Limit       int             `json:"limit"`
	Offset      int             `json:"offset"`
}

type createCollectionRequest struct {
	Name     string `json:"name"`
	Overview string `json:"overview"`
}

type patchCollectionRequest struct {
	Name        *string `json:"name"`
	Overview    *string `json:"overview"`
	PosterURL   *string `json:"poster_url"`
	BackdropURL *string `json:"backdrop_url"`
}

type collectionItemRequest struct {
	MediaItemID int64 `json:"media_item_id"`
}

type collectionOrderRequest struct {
	MediaItemIDs *[]int64 `json:"media_item_ids"`
}

func (a *API) handleCollections(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := collectionPagination(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	query := r.URL.Query()
	q := strings.TrimSpace(query.Get("q"))
	sortOrder := strings.TrimSpace(query.Get("sort"))
	if sortOrder == "" {
		sortOrder = "name"
	}

	where := `c.deleted = 0 AND (c.source = 'manual' OR ` + collectionItemCountExpr + ` >= 2)`
	args := make([]any, 0)
	if q != "" {
		where += ` AND LOWER(c.name) LIKE LOWER(?) ESCAPE '\'`
		args = append(args, likePattern(q))
	}

	var total int64
	if err := a.db.QueryRowContext(r.Context(),
		`SELECT count(*) FROM collections c WHERE `+where, args...).Scan(&total); err != nil {
		log.Printf("library: contar colecciones: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	order := " ORDER BY c.name COLLATE NOCASE, c.id"
	switch sortOrder {
	case "name":
	case "added":
		order = " ORDER BY c.id DESC"
	case "size":
		order = " ORDER BY " + collectionItemCountExpr + " DESC, c.name COLLATE NOCASE, c.id"
	default:
		writeJSONError(w, http.StatusBadRequest, "sort debe ser name, added o size")
		return
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, limit, offset)
	rows, err := a.db.QueryContext(r.Context(), `
		SELECT c.id, c.name, c.overview, `+collectionPosterExpr+`, c.backdrop_url,
		       `+collectionItemCountExpr+`, c.source, c.edited
		  FROM collections c
		 WHERE `+where+order+` LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		log.Printf("library: listar colecciones: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer rows.Close()

	collections := make([]collectionDTO, 0)
	for rows.Next() {
		collection, err := scanCollection(rows.Scan)
		if err != nil {
			log.Printf("library: leer colección: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		collections = append(collections, collection)
	}
	if err := rows.Err(); err != nil {
		log.Printf("library: recorrer colecciones: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	writeJSON(w, http.StatusOK, collectionsResponse{
		Collections: collections,
		Total:       total,
		Limit:       limit,
		Offset:      offset,
	})
}

func collectionPagination(r *http.Request) (limit, offset int, err error) {
	limit = defaultCollectionsLimit
	query := r.URL.Query()

	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		value, convErr := strconv.Atoi(raw)
		if convErr != nil || value < 1 || value > maxCollectionsLimit {
			return 0, 0, fmt.Errorf("limit debe estar entre 1 y %d", maxCollectionsLimit)
		}
		limit = value
	}
	if raw := strings.TrimSpace(query.Get("offset")); raw != "" {
		value, convErr := strconv.Atoi(raw)
		if convErr != nil || value < 0 {
			return 0, 0, errors.New("offset debe ser un número mayor o igual que 0")
		}
		offset = value
	}
	return limit, offset, nil
}

func scanCollection(scan func(dest ...any) error) (collectionDTO, error) {
	var collection collectionDTO
	var overview, poster, backdrop sql.NullString
	var edited int
	if err := scan(&collection.ID, &collection.Name, &overview, &poster, &backdrop,
		&collection.ItemCount, &collection.Source, &edited); err != nil {
		return collectionDTO{}, err
	}
	collection.Overview = stringPointer(overview)
	collection.PosterURL = stringPointer(poster)
	collection.BackdropURL = stringPointer(backdrop)
	collection.Edited = edited != 0
	return collection, nil
}

func queryCollectionBase(ctx context.Context, db *sql.DB, id int64) (collectionDTO, bool, error) {
	row := db.QueryRowContext(ctx, `
		SELECT c.id, c.name, c.overview, `+collectionPosterExpr+`, c.backdrop_url,
		       `+collectionItemCountExpr+`, c.source, c.edited, c.deleted
		  FROM collections c
		 WHERE c.id = ?`, id)

	var collection collectionDTO
	var overview, poster, backdrop sql.NullString
	var edited, deleted int
	if err := row.Scan(&collection.ID, &collection.Name, &overview, &poster,
		&backdrop, &collection.ItemCount, &collection.Source, &edited, &deleted); err != nil {
		return collectionDTO{}, false, err
	}
	collection.Overview = stringPointer(overview)
	collection.PosterURL = stringPointer(poster)
	collection.BackdropURL = stringPointer(backdrop)
	collection.Edited = edited != 0
	return collection, deleted != 0, nil
}

func (a *API) handleCollection(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "colección no encontrada")
		return
	}
	collection, deleted, err := queryCollectionBase(r.Context(), a.db, id)
	if errors.Is(err, sql.ErrNoRows) || deleted {
		writeJSONError(w, http.StatusNotFound, "colección no encontrada")
		return
	}
	if err != nil {
		log.Printf("library: leer colección %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if collection.Source == "tmdb" && collection.ItemCount < 2 {
		writeJSONError(w, http.StatusNotFound, "colección no encontrada")
		return
	}

	items, err := a.collectionMovieItems(r.Context(), id, sessionUserID(r))
	if err != nil {
		log.Printf("library: leer películas de la colección %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	writeJSON(w, http.StatusOK, collectionDetailDTO{
		ID:          collection.ID,
		Name:        collection.Name,
		Overview:    collection.Overview,
		PosterURL:   collection.PosterURL,
		BackdropURL: collection.BackdropURL,
		ItemCount:   collection.ItemCount,
		Source:      collection.Source,
		Edited:      collection.Edited,
		Items:       items,
	})
}

func (a *API) collectionMovieItems(ctx context.Context, collectionID, userID int64) ([]mediaItemDTO, error) {
	rows, err := a.db.QueryContext(ctx, `
		SELECT `+mediaItemColumns+watchedColumn+progressColumn+`
		  FROM media_items
		  JOIN collection_items ci
		    ON ci.media_item_id = media_items.id
		   AND ci.collection_id = ?
		   AND ci.excluded = 0
		  LEFT JOIN watch_state ws
		    ON ws.media_item_id = media_items.id AND ws.user_id = ?
		  LEFT JOIN playback_progress pp
		    ON pp.media_item_id = media_items.id AND pp.user_id = ?
		 ORDER BY ci.position IS NULL, ci.position ASC,
		          media_items.year IS NULL, media_items.year ASC, media_items.id`,
		collectionID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]mediaItemDTO, 0)
	for rows.Next() {
		item, err := scanItem(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	itemIDs := make([]int64, 0, len(items))
	for _, item := range items {
		itemIDs = append(itemIDs, item.ID)
	}
	collectionsByItem, err := loadItemCollections(ctx, a.db, itemIDs)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if collections, ok := collectionsByItem[items[i].ID]; ok {
			items[i].Collections = collections
		}
	}
	return items, nil
}

func normalizeCollectionName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	length := utf8.RuneCountInString(name)
	if length < 1 || length > 120 {
		return "", errors.New("el nombre debe tener entre 1 y 120 caracteres")
	}
	return name, nil
}

func normalizeCollectionOverview(raw string) (any, error) {
	overview := strings.TrimSpace(raw)
	if utf8.RuneCountInString(overview) > 2000 {
		return nil, errors.New("overview no puede superar los 2000 caracteres")
	}
	return nullableText(overview), nil
}

func (a *API) handleCollectionCreate(w http.ResponseWriter, r *http.Request) {
	var req createCollectionRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "petición no válida")
		return
	}
	name, err := normalizeCollectionName(req.Name)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	overview, err := normalizeCollectionOverview(req.Overview)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := a.db.ExecContext(r.Context(), `
		INSERT INTO collections (name, overview, source)
		VALUES (?, ?, 'manual')`, name, overview)
	if err != nil {
		log.Printf("library: crear colección: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("library: id de la colección creada: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	collection, _, err := queryCollectionBase(r.Context(), a.db, id)
	if err != nil {
		log.Printf("library: leer colección creada %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusCreated, collection)
}

func (a *API) handleCollectionPatch(w http.ResponseWriter, r *http.Request) {
	collection, ok := a.loadAdminCollection(w, r)
	if !ok {
		return
	}

	var req patchCollectionRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "petición no válida")
		return
	}
	if req.Name == nil && req.Overview == nil && req.PosterURL == nil && req.BackdropURL == nil {
		writeJSONError(w, http.StatusBadRequest, "no hay ningún campo que cambiar")
		return
	}

	updates := make([]string, 0, 5)
	args := make([]any, 0, 5)
	if req.Name != nil {
		name, err := normalizeCollectionName(*req.Name)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		updates = append(updates, "name = ?")
		args = append(args, name)
	}
	if req.Overview != nil {
		overview, err := normalizeCollectionOverview(*req.Overview)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		updates = append(updates, "overview = ?")
		args = append(args, overview)
	}
	if req.PosterURL != nil {
		if *req.PosterURL != "" && !validImageURL(*req.PosterURL) {
			writeJSONError(w, http.StatusBadRequest, "URL no válida")
			return
		}
		updates = append(updates, "poster_url = ?")
		args = append(args, nullableText(*req.PosterURL))
	}
	if req.BackdropURL != nil {
		if *req.BackdropURL != "" && !validImageURL(*req.BackdropURL) {
			writeJSONError(w, http.StatusBadRequest, "URL no válida")
			return
		}
		updates = append(updates, "backdrop_url = ?")
		args = append(args, nullableText(*req.BackdropURL))
	}
	if collection.Source == "tmdb" {
		updates = append(updates, "edited = 1")
	}

	args = append(args, collection.ID)
	if _, err := a.db.ExecContext(r.Context(),
		`UPDATE collections SET `+strings.Join(updates, ", ")+` WHERE id = ?`, args...); err != nil {
		log.Printf("library: actualizar colección %d: %v", collection.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	a.writeCollection(w, r, collection.ID)
}

func (a *API) handleCollectionDelete(w http.ResponseWriter, r *http.Request) {
	collection, ok := a.loadAdminCollection(w, r)
	if !ok {
		return
	}

	var err error
	if collection.Source == "manual" {
		_, err = a.db.ExecContext(r.Context(), `DELETE FROM collections WHERE id = ?`, collection.ID)
	} else {
		_, err = a.db.ExecContext(r.Context(), `UPDATE collections SET deleted = 1 WHERE id = ?`, collection.ID)
	}
	if err != nil {
		log.Printf("library: borrar colección %d: %v", collection.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) loadAdminCollection(w http.ResponseWriter, r *http.Request) (collectionDTO, bool) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "colección no encontrada")
		return collectionDTO{}, false
	}
	collection, deleted, err := queryCollectionBase(r.Context(), a.db, id)
	if errors.Is(err, sql.ErrNoRows) || deleted {
		writeJSONError(w, http.StatusNotFound, "colección no encontrada")
		return collectionDTO{}, false
	}
	if err != nil {
		log.Printf("library: leer colección %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return collectionDTO{}, false
	}
	return collection, true
}

func (a *API) writeCollection(w http.ResponseWriter, r *http.Request, id int64) {
	collection, _, err := queryCollectionBase(r.Context(), a.db, id)
	if err != nil {
		log.Printf("library: leer colección %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, collection)
}

func (a *API) requireMovieForCollection(w http.ResponseWriter, r *http.Request, itemID int64) bool {
	var kind string
	err := a.db.QueryRowContext(r.Context(),
		`SELECT kind FROM media_items WHERE id = ?`, itemID).Scan(&kind)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return false
	case err != nil:
		log.Printf("library: comprobar ítem %d para colección: %v", itemID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return false
	case kind != KindMovie:
		writeJSONError(w, http.StatusBadRequest, "el ítem no es una película")
		return false
	}
	return true
}

func (a *API) handleCollectionItemAdd(w http.ResponseWriter, r *http.Request) {
	collection, ok := a.loadAdminCollection(w, r)
	if !ok {
		return
	}
	var req collectionItemRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "petición no válida")
		return
	}
	if req.MediaItemID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "media_item_id debe ser un entero positivo")
		return
	}
	if !a.requireMovieForCollection(w, r, req.MediaItemID) {
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("library: iniciar alta en colección %d: %v", collection.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if collection.Source == "manual" {
		var excluded int
		err := tx.QueryRowContext(r.Context(), `
			SELECT excluded FROM collection_items
			 WHERE collection_id = ? AND media_item_id = ?`,
			collection.ID, req.MediaItemID).Scan(&excluded)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			var position int64
			if err := tx.QueryRowContext(r.Context(), `
				SELECT COALESCE(MAX(position), -1) + 1
				  FROM collection_items WHERE collection_id = ?`,
				collection.ID).Scan(&position); err != nil {
				log.Printf("library: calcular posición en colección %d: %v", collection.ID, err)
				writeJSONError(w, http.StatusInternalServerError, "error interno")
				return
			}
			if _, err := tx.ExecContext(r.Context(), `
				INSERT INTO collection_items (collection_id, media_item_id, position)
				VALUES (?, ?, ?)`, collection.ID, req.MediaItemID, position); err != nil {
				log.Printf("library: añadir ítem %d a colección %d: %v",
					req.MediaItemID, collection.ID, err)
				writeJSONError(w, http.StatusInternalServerError, "error interno")
				return
			}
		case err != nil:
			log.Printf("library: leer pertenencia a colección %d: %v", collection.ID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		case excluded != 0:
			if _, err := tx.ExecContext(r.Context(), `
				UPDATE collection_items SET excluded = 0
				 WHERE collection_id = ? AND media_item_id = ?`,
				collection.ID, req.MediaItemID); err != nil {
				log.Printf("library: reponer ítem %d en colección %d: %v",
					req.MediaItemID, collection.ID, err)
				writeJSONError(w, http.StatusInternalServerError, "error interno")
				return
			}
		}
	} else {
		if _, err := tx.ExecContext(r.Context(), `
			INSERT INTO collection_items (collection_id, media_item_id, position, excluded)
			VALUES (?, ?, NULL, 0)
			ON CONFLICT (collection_id, media_item_id) DO UPDATE SET excluded = 0`,
			collection.ID, req.MediaItemID); err != nil {
			log.Printf("library: añadir ítem %d a colección %d: %v",
				req.MediaItemID, collection.ID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		if _, err := tx.ExecContext(r.Context(),
			`UPDATE collections SET edited = 1 WHERE id = ?`, collection.ID); err != nil {
			log.Printf("library: editar colección %d: %v", collection.ID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("library: confirmar alta en colección %d: %v", collection.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleCollectionItemRemove(w http.ResponseWriter, r *http.Request) {
	collection, ok := a.loadAdminCollection(w, r)
	if !ok {
		return
	}
	itemID, err := strconv.ParseInt(r.PathValue("itemId"), 10, 64)
	if err != nil || itemID <= 0 {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	if !a.requireMovieForCollection(w, r, itemID) {
		return
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("library: iniciar baja en colección %d: %v", collection.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if collection.Source == "manual" {
		if _, err := tx.ExecContext(r.Context(), `
			DELETE FROM collection_items
			 WHERE collection_id = ? AND media_item_id = ?`,
			collection.ID, itemID); err != nil {
			log.Printf("library: quitar ítem %d de colección %d: %v", itemID, collection.ID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
	} else {
		if _, err := tx.ExecContext(r.Context(), `
			INSERT INTO collection_items (collection_id, media_item_id, position, excluded)
			VALUES (?, ?, NULL, 1)
			ON CONFLICT (collection_id, media_item_id) DO UPDATE SET excluded = 1`,
			collection.ID, itemID); err != nil {
			log.Printf("library: excluir ítem %d de colección %d: %v", itemID, collection.ID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		if _, err := tx.ExecContext(r.Context(),
			`UPDATE collections SET edited = 1 WHERE id = ?`, collection.ID); err != nil {
			log.Printf("library: editar colección %d: %v", collection.ID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("library: confirmar baja en colección %d: %v", collection.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleCollectionOrder(w http.ResponseWriter, r *http.Request) {
	collection, ok := a.loadAdminCollection(w, r)
	if !ok {
		return
	}
	var req collectionOrderRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "petición no válida")
		return
	}
	if req.MediaItemIDs == nil {
		writeJSONError(w, http.StatusBadRequest, "falta media_item_ids")
		return
	}
	ids := *req.MediaItemIDs
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id <= 0 {
			writeJSONError(w, http.StatusBadRequest, "media_item_ids contiene un id no válido")
			return
		}
		if seen[id] {
			writeJSONError(w, http.StatusBadRequest, "media_item_ids contiene repetidos")
			return
		}
		seen[id] = true
	}

	rows, err := a.db.QueryContext(r.Context(), `
		SELECT media_item_id FROM collection_items
		 WHERE collection_id = ? AND excluded = 0`, collection.ID)
	if err != nil {
		log.Printf("library: leer orden de colección %d: %v", collection.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	current := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			log.Printf("library: leer miembro de colección %d: %v", collection.ID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
		current[id] = true
	}
	if err := rows.Close(); err != nil {
		log.Printf("library: cerrar miembros de colección %d: %v", collection.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	if len(ids) != len(current) {
		writeJSONError(w, http.StatusBadRequest, "la lista debe contener exactamente las películas de la colección")
		return
	}
	for _, id := range ids {
		if !current[id] {
			writeJSONError(w, http.StatusBadRequest, "la lista debe contener exactamente las películas de la colección")
			return
		}
	}

	tx, err := a.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("library: iniciar orden de colección %d: %v", collection.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for position, id := range ids {
		if _, err := tx.ExecContext(r.Context(), `
			UPDATE collection_items SET position = ?
			 WHERE collection_id = ? AND media_item_id = ? AND excluded = 0`,
			position, collection.ID, id); err != nil {
			log.Printf("library: ordenar ítem %d de colección %d: %v", id, collection.ID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
	}
	if collection.Source == "tmdb" {
		if _, err := tx.ExecContext(r.Context(),
			`UPDATE collections SET edited = 1 WHERE id = ?`, collection.ID); err != nil {
			log.Printf("library: editar colección %d: %v", collection.ID, err)
			writeJSONError(w, http.StatusInternalServerError, "error interno")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("library: confirmar orden de colección %d: %v", collection.ID, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
