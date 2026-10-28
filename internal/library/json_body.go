package library

import (
	"encoding/json"
	"errors"
	"net/http"
)

const maxLibraryBody = 1 << 20 // 1 MiB

var errJSONBodyTooLarge = errors.New("library: cuerpo de petición demasiado grande")

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	if r.ContentLength > maxLibraryBody {
		return errJSONBodyTooLarge
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxLibraryBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return errJSONBodyTooLarge
	}
	return err
}

func writeJSONBodyError(w http.ResponseWriter, err error) {
	if errors.Is(err, errJSONBodyTooLarge) {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "el cuerpo de la petición supera el límite de 1 MiB")
		return
	}
	writeJSONError(w, http.StatusBadRequest, "petición no válida")
}
