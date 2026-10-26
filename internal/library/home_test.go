package library

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestHomeBaseVacia(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/home", ts.token)
	if status != http.StatusOK {
		t.Fatalf("home vacío: código = %d, cuerpo = %s", status, body)
	}
	var home homeResponse
	if err := json.Unmarshal(body, &home); err != nil {
		t.Fatalf("respuesta de home: %v", err)
	}
	if home.Continue == nil || home.NextUp == nil || home.RecentMovies == nil {
		t.Fatalf("los arrays deben ser [] y no null: %+v", home)
	}
	if len(home.Continue) != 0 || len(home.NextUp) != 0 || len(home.RecentMovies) != 0 {
		t.Fatalf("home vacío con ítems: %+v", home)
	}
	if home.Counts.Movies != 0 || home.Counts.Episodes != 0 || home.Counts.Series != 0 {
		t.Fatalf("contadores inesperados: %+v", home.Counts)
	}

	if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/home", ""); status != http.StatusUnauthorized {
		t.Fatalf("sin sesión: código = %d, cuerpo = %s", status, body)
	}
}

func TestHomeContinueOrdenLimiteYMezcla(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	userID := progressUserID(t, ts.db, "ana")

	ids := make([]int64, 0, 22)
	for i := 0; i < 22; i++ {
		kind := KindMovie
		if i%2 == 1 {
			kind = KindEpisode
		}
		id := progressItem(t, ts, libraryID, fmt.Sprintf("Item %02d", i), kind)
		ids = append(ids, id)
		setProgressRow(t, ts.db, userID, id, 60, 600,
			fmt.Sprintf("2026-10-26T10:%02d:00Z", 59-i))
	}

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/home", ts.token)
	if status != http.StatusOK {
		t.Fatalf("home: código = %d, cuerpo = %s", status, body)
	}
	var home homeResponse
	if err := json.Unmarshal(body, &home); err != nil {
		t.Fatalf("respuesta de home: %v", err)
	}
	if len(home.Continue) != 20 {
		t.Fatalf("continuar viendo = %d, se esperaban 20", len(home.Continue))
	}
	for i, item := range home.Continue {
		if item.ID != ids[i] {
			t.Fatalf("posición %d: id = %d, se esperaba %d", i, item.ID, ids[i])
		}
		if item.Progress == nil {
			t.Fatalf("el ítem %d no incluye progreso", item.ID)
		}
	}
	if home.Continue[0].Kind != KindMovie || home.Continue[1].Kind != KindEpisode {
		t.Fatalf("la lista debe mezclar películas y episodios: %+v", home.Continue[:2])
	}
	if home.NextUp == nil || len(home.NextUp) != 0 {
		t.Fatalf("next_up debe ser [] por ahora: %+v", home.NextUp)
	}
	if home.Counts.Movies != 11 || home.Counts.Episodes != 11 || home.Counts.Series != 0 {
		t.Fatalf("contadores inesperados: %+v", home.Counts)
	}
}

func TestHomeRecientesContadoresYAislamiento(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())
	anaID := progressUserID(t, ts.db, "ana")
	beaToken := nonAdminToken(t, ts)
	beaID := progressUserID(t, ts.db, "bea")

	movieIDs := make([]int64, 0, 22)
	for i := 0; i < 22; i++ {
		movieIDs = append(movieIDs, progressItem(t, ts, libraryID,
			fmt.Sprintf("Pelicula %02d", i), KindMovie))
	}
	for i := 0; i < 2; i++ {
		progressItem(t, ts, libraryID, fmt.Sprintf("Episodio %02d", i), KindEpisode)
	}

	setWatchState(t, ts.db, anaID, movieIDs[21])
	setProgressRow(t, ts.db, anaID, movieIDs[20], 300, 1000, "2026-10-26T10:00:00Z")
	setProgressRow(t, ts.db, beaID, movieIDs[18], 200, 1000, "2026-10-26T10:01:00Z")

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/home", ts.token)
	if status != http.StatusOK {
		t.Fatalf("home de ana: código = %d, cuerpo = %s", status, body)
	}
	var anaHome homeResponse
	if err := json.Unmarshal(body, &anaHome); err != nil {
		t.Fatalf("home de ana: %v", err)
	}
	if len(anaHome.RecentMovies) != 20 {
		t.Fatalf("películas recientes = %d, se esperaban 20", len(anaHome.RecentMovies))
	}
	for i, item := range anaHome.RecentMovies {
		expected := movieIDs[len(movieIDs)-1-i]
		if item.ID != expected || item.Kind != KindMovie {
			t.Fatalf("reciente %d: id = %d, se esperaba %d", i, item.ID, expected)
		}
	}
	var lastMovie, progressMovie mediaItemDTO
	for _, item := range anaHome.RecentMovies {
		switch item.ID {
		case movieIDs[21]:
			lastMovie = item
		case movieIDs[20]:
			progressMovie = item
		}
	}
	if !lastMovie.Watched || lastMovie.Progress != nil {
		t.Fatalf("la última película debía estar vista y sin progreso: %+v", lastMovie)
	}
	if progressMovie.Progress == nil || progressMovie.Progress.PositionSeconds != 300 {
		t.Fatalf("la película con progreso no lo incluye: %+v", progressMovie)
	}
	if anaHome.Counts.Movies != 22 || anaHome.Counts.Episodes != 2 || anaHome.Counts.Series != 0 {
		t.Fatalf("contadores inesperados: %+v", anaHome.Counts)
	}
	if len(anaHome.Continue) != 1 || anaHome.Continue[0].ID != movieIDs[20] {
		t.Fatalf("continuar viendo de ana inesperado: %+v", anaHome.Continue)
	}
	if anaHome.NextUp == nil || len(anaHome.NextUp) != 0 {
		t.Fatalf("next_up debe ser [] por ahora: %+v", anaHome.NextUp)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/home", beaToken)
	if status != http.StatusOK {
		t.Fatalf("home de bea: código = %d, cuerpo = %s", status, body)
	}
	var beaHome homeResponse
	if err := json.Unmarshal(body, &beaHome); err != nil {
		t.Fatalf("home de bea: %v", err)
	}
	if len(beaHome.Continue) != 1 || beaHome.Continue[0].ID != movieIDs[18] {
		t.Fatalf("continuar viendo de bea inesperado: %+v", beaHome.Continue)
	}
	var beaProgress, beaAnaProgress, beaWatched mediaItemDTO
	for _, item := range beaHome.RecentMovies {
		switch item.ID {
		case movieIDs[18]:
			beaProgress = item
		case movieIDs[20]:
			beaAnaProgress = item
		case movieIDs[21]:
			beaWatched = item
		}
	}
	if beaProgress.Progress == nil || beaProgress.Progress.PositionSeconds != 200 {
		t.Fatalf("bea no ve su propio progreso: %+v", beaProgress)
	}
	if beaAnaProgress.Progress != nil {
		t.Fatalf("bea no debe ver el progreso de ana: %+v", beaAnaProgress)
	}
	if beaWatched.Watched {
		t.Fatalf("bea no debe heredar el visionado de ana: %+v", beaWatched)
	}
}
