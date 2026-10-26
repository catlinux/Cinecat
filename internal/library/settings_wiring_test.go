package library

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"

	"github.com/catlinux/cinecat/internal/metadata"
)

func TestSettingsWiringUmbralDeIdentificacion(t *testing.T) {
	t.Cleanup(func() { SetMatchThreshold(MatchThreshold) })
	ts := newSettingsHTTPTestServer(t)
	searcher := wiringThresholdSearcher{candidate: metadata.Candidate{
		TMDBID: 1, Title: "Alien Resurrection", OriginalTitle: "Alien Resurrection", VoteCount: 1000,
	}}

	for _, test := range []struct {
		threshold float64
		ok        bool
	}{
		{threshold: 0.75, ok: false},
		{threshold: 0.65, ok: true},
	} {
		status, body := libraryHTTPDo(t, http.MethodPatch, ts.url+"/api/v1/settings", ts.token,
			map[string]any{"metadata.match_threshold": test.threshold})
		if status != http.StatusOK {
			t.Fatalf("PATCH umbral %.2f: código = %d, cuerpo = %s", test.threshold, status, body)
		}
		_, _, identified, err := IdentifyMovie(context.Background(), searcher, "Alien", 0)
		if err != nil {
			t.Fatalf("IdentifyMovie: %v", err)
		}
		if identified != test.ok {
			t.Fatalf("con umbral %.2f identificada = %v, se esperaba %v", test.threshold, identified, test.ok)
		}
	}
}

func TestSettingsWiringMinimoDeColecciones(t *testing.T) {
	ts := newSettingsHTTPTestServer(t)

	first := collectionHTTPMovie(t, ts.libraryTestServer, "Primera", 2000, "", KindMovie)
	second := collectionHTTPMovie(t, ts.libraryTestServer, "Segunda", 2001, "", KindMovie)
	one := collectionHTTPSeed(t, ts.db, "Saga de una", "tmdb", 901, "", 0, 0)
	two := collectionHTTPSeed(t, ts.db, "Saga de dos", "tmdb", 902, "", 0, 0)
	manual := collectionHTTPSeed(t, ts.db, "Manual", "manual", nil, "", 0, 0)
	collectionHTTPMember(t, ts.db, one, first, nil, 0)
	collectionHTTPMember(t, ts.db, two, first, nil, 0)
	collectionHTTPMember(t, ts.db, two, second, nil, 0)
	collectionHTTPMember(t, ts.db, manual, first, nil, 0)

	readCollections := func() map[string]collectionDTO {
		t.Helper()
		status, body := libraryHTTPGet(t, ts.url+"/api/v1/collections?limit=200", ts.token)
		if status != http.StatusOK {
			t.Fatalf("listar colecciones: código = %d, cuerpo = %s", status, body)
		}
		var response collectionsResponse
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("respuesta de colecciones: %v", err)
		}
		byName := make(map[string]collectionDTO)
		for _, collection := range response.Collections {
			byName[collection.Name] = collection
		}
		return byName
	}

	if collections := readCollections(); collections["Saga de una"].ID != 0 ||
		collections["Saga de dos"].ID == 0 || collections["Manual"].ID == 0 {
		t.Fatalf("visibilidad inicial inesperada: %+v", collections)
	}

	if status, body := libraryHTTPDo(t, http.MethodPatch, ts.url+"/api/v1/settings", ts.token,
		map[string]any{"ui.collection_min_items": 1}); status != http.StatusOK {
		t.Fatalf("PATCH mínimo 1: código = %d, cuerpo = %s", status, body)
	}
	if collections := readCollections(); collections["Saga de una"].ID == 0 {
		t.Fatalf("con mínimo 1 debe verse la saga de una: %+v", collections)
	}

	status, body := libraryHTTPGet(t, ts.url+"/api/v1/items/"+strconv.FormatInt(first, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("ficha con mínimo 1: código = %d, cuerpo = %s", status, body)
	}
	var item mediaItemDTO
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("ficha con mínimo 1: %v", err)
	}
	if len(item.Collections) != 3 {
		t.Fatalf("colecciones del ítem con mínimo 1 = %+v", item.Collections)
	}

	if status, body := libraryHTTPDo(t, http.MethodPatch, ts.url+"/api/v1/settings", ts.token,
		map[string]any{"ui.collection_min_items": 3}); status != http.StatusOK {
		t.Fatalf("PATCH mínimo 3: código = %d, cuerpo = %s", status, body)
	}
	if collections := readCollections(); collections["Saga de una"].ID != 0 ||
		collections["Saga de dos"].ID != 0 || collections["Manual"].ID == 0 {
		t.Fatalf("con mínimo 3 solo debe verse la manual: %+v", collections)
	}

	status, body = libraryHTTPGet(t, ts.url+"/api/v1/items/"+strconv.FormatInt(first, 10), ts.token)
	if status != http.StatusOK {
		t.Fatalf("ficha con mínimo 3: código = %d, cuerpo = %s", status, body)
	}
	if err := json.Unmarshal(body, &item); err != nil {
		t.Fatalf("ficha con mínimo 3: %v", err)
	}
	if len(item.Collections) != 1 || item.Collections[0].ID != manual {
		t.Fatalf("colecciones del ítem con mínimo 3 = %+v", item.Collections)
	}
}
