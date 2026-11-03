package library

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/catlinux/cinecat/internal/playback"
)

type segmentsResponse struct {
	Manual   []itemSegment `json:"manual"`
	Chapters []itemSegment `json:"chapters"`
}

type segmentsPutRequest struct {
	Segments *[]itemSegment `json:"segments"`
}

func (a *API) handleItemSegmentsGet(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	if _, ok := a.requireVisibleItem(w, r, id); !ok {
		return
	}

	manual, err := loadManualSegments(r.Context(), a.db, id)
	if err != nil {
		log.Printf("library: leer segmentos manuales del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	writeJSON(w, http.StatusOK, segmentsResponse{
		Manual:   manual,
		Chapters: a.itemChapterSegments(r, id),
	})
}

func (a *API) handleItemSegmentsPut(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	if _, ok := a.requireVisibleItem(w, r, id); !ok {
		return
	}

	var payload segmentsPutRequest
	if err := decodeJSONBody(w, r, &payload); err != nil {
		writeJSONBodyError(w, err)
		return
	}
	if payload.Segments == nil {
		writeJSONError(w, http.StatusBadRequest, "segments no válido")
		return
	}
	if err := validateManualSegments(*payload.Segments); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := replaceManualSegments(r.Context(), a.db, id, *payload.Segments); err != nil {
		log.Printf("library: guardar segmentos manuales del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) handleItemSegmentsApplySeason(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(r)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
		return
	}
	if _, ok := a.requireVisibleItem(w, r, id); !ok {
		return
	}

	updated, err := a.applySeasonSegments(r.Context(), id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "ítem no encontrado")
	case errors.Is(err, errSegmentNotEpisode):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	case err != nil:
		log.Printf("library: aplicar segmentos a la temporada del ítem %d: %v", id, err)
		writeJSONError(w, http.StatusInternalServerError, "error interno")
	default:
		writeJSON(w, http.StatusOK, map[string]int{"updated": updated})
	}
}

func validateManualSegments(segments []itemSegment) error {
	seen := make(map[string]bool, len(segments))
	for _, segment := range segments {
		if segment.Kind != segmentIntro && segment.Kind != segmentCredits {
			return errors.New("kind debe ser intro o credits")
		}
		if seen[segment.Kind] {
			return errors.New("no se puede repetir el mismo tipo de segmento")
		}
		seen[segment.Kind] = true
		if segment.Start < 0 {
			return errors.New("start no puede ser negativo")
		}
		if segment.End <= segment.Start {
			return errors.New("end debe ser mayor que start")
		}
	}
	return nil
}

func (a *API) itemChapterSegments(r *http.Request, id int64) []itemSegment {
	media, err := a.mediaFile(r, id)
	if err != nil {
		return []itemSegment{}
	}
	safePath, err := playback.SafePath(media.libraryPath, media.path)
	if err != nil {
		return []itemSegment{}
	}
	info, err := a.cachedFileInfo(r.Context(), safePath)
	if err != nil {
		return []itemSegment{}
	}
	return chapterSegments(&info)
}
