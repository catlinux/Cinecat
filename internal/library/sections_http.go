package library

import (
	"log"
	"net/http"
	"strconv"
)

type sectionsResponse struct {
	Movies      bool `json:"movies"`
	Series      bool `json:"series"`
	Collections bool `json:"collections"`
}

// handleSections indica qué secciones tienen contenido accesible para el
// usuario. Los administradores ven siempre todas, como en el resto de la API.
func (a *API) handleSections(w http.ResponseWriter, r *http.Request) {
	if isAdminRequest(r) {
		writeJSON(w, http.StatusOK, sectionsResponse{
			Movies: true, Series: true, Collections: true,
		})
		return
	}

	access := accessFromRequest(r)
	var movies, series, collections int
	err := a.db.QueryRowContext(r.Context(), `
		SELECT
		  EXISTS (
		    SELECT 1 FROM media_items mi
		     WHERE mi.kind = 'movie' AND `+access.VideoClause("mi")+`
		  ),
		  EXISTS (
		    SELECT 1 FROM series s
		     WHERE `+access.Clause("s")+`
		  ),
		  EXISTS (
		    SELECT 1
		      FROM collections c
		      JOIN collection_items ci
		        ON ci.collection_id = c.id AND ci.excluded = 0
		      JOIN media_items mi ON mi.id = ci.media_item_id
		     WHERE c.deleted = 0
		       AND mi.kind = 'movie'
		       AND `+access.VideoClause("mi")+`
		       AND (c.source = 'manual' OR `+collectionItemCountSQL(access)+` >= `+strconv.Itoa(a.collectionMin())+`)
		  )`).Scan(&movies, &series, &collections)
	if err != nil {
		log.Printf("library: calcular secciones disponibles: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}

	writeJSON(w, http.StatusOK, sectionsResponse{
		Movies:      movies != 0,
		Series:      series != 0,
		Collections: collections != 0,
	})
}
