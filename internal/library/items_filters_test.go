package library

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

func TestItemsBusquedaPorCampos(t *testing.T) {
	ts := newLibraryHTTPTestServer(t)
	libraryID := createLibrary(t, ts.db, t.TempDir())

	padrino := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/El Padrino (1972).mkv", title: "El Padrino",
		originalTitle: "The Godfather", year: 1972,
		genres: `["Drama","Crimen"]`, directors: `["Francis Ford Coppola"]`,
		cast: `["Marlon Brando","Al Pacino"]`, tagline: "Una oferta que no podrás rechazar",
	})
	nandu := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/Ñandú (2020).mkv", title: "Ñandú", year: 2020,
		genres: `["Documental"]`, directors: `["Penélope Directora"]`,
		cast: `["Penélope Cruz"]`, tagline: "La vida salvaje",
	})
	number := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/1917 (2019).mkv", title: "1917", year: 2019,
		genres: `["Bélico"]`, cast: `["George MacKay"]`,
	})
	matrix := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/The Matrix (1999).mkv", title: "The Matrix",
		year: 1999, genres: `["Ciencia ficción","Acción"]`,
		directors: `["Lana Wachowski"]`, cast: `["Keanu Reeves"]`,
		tagline: "La realidad es una simulación",
	})
	aguila := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/Aguila.mkv", title: "Aguila",
		displayTitle: "Águila", genres: `["Drama","Comedia"]`,
	})
	episode := insertItemFixture(t, ts.db, itemFixture{
		libraryID: libraryID, path: "/media/Serie/S01E01.mkv", title: "Capítulo",
		kind: KindEpisode, genres: `["Drama"]`,
	})

	get := func(query string) itemsResponse {
		t.Helper()
		status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items?"+query, ts.token)
		if status != http.StatusOK {
			t.Fatalf("%s: código = %d, cuerpo = %s", query, status, body)
		}
		var response itemsResponse
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		if response.Total != int64(len(response.Items)) {
			t.Fatalf("%s: total = %d, ítems = %d", query, response.Total, len(response.Items))
		}
		return response
	}
	ids := func(response itemsResponse) []int64 {
		values := make([]int64, 0, len(response.Items))
		for _, item := range response.Items {
			values = append(values, item.ID)
		}
		return values
	}

	for _, test := range []struct {
		query string
		want  int64
	}{
		{"field=cast&q=" + url.QueryEscape("Penélope"), nandu},
		{"field=director&q=" + url.QueryEscape("Penélope"), nandu},
		{"field=title&q=Matrix", matrix},
		{"field=title&q=" + url.QueryEscape("Ñandú"), nandu},
		{"field=all&q=" + url.QueryEscape("simulación"), matrix},
		{"field=all&q=Godfather", padrino},
		{"field=title&q=" + url.QueryEscape("Penélope"), 0},
		{"q=" + url.QueryEscape("simulación"), matrix},
		{"field=cast&q=" + url.QueryEscape("Al Pacino"), padrino},
	} {
		response := get(test.query)
		if test.want == 0 {
			if response.Total != 0 {
				t.Fatalf("%s: total = %d", test.query, response.Total)
			}
			continue
		}
		if response.Total != 1 || response.Items[0].ID != test.want {
			t.Fatalf("%s: %+v", test.query, response)
		}
	}

	for _, invalid := range []string{"field=actor", "field=title&q=Matrix&letter=AB"} {
		if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/items?"+invalid, ts.token); status != http.StatusBadRequest {
			t.Fatalf("%s: código = %d, cuerpo = %s", invalid, status, body)
		}
	}

	if got := ids(get("letter=P")); len(got) != 1 || got[0] != padrino {
		t.Fatalf("letter=P: %v", got)
	}
	if got := ids(get("letter=n")); len(got) != 1 || got[0] != nandu {
		t.Fatalf("letter=n: %v", got)
	}
	if got := ids(get("letter=Ñ")); len(got) != 1 || got[0] != nandu {
		t.Fatalf("letter=Ñ: %v", got)
	}
	if got := ids(get("letter=" + url.QueryEscape("#"))); len(got) != 1 || got[0] != number {
		t.Fatalf("letter=#: %v", got)
	}
	if got := ids(get("letter=A")); len(got) != 1 || got[0] != aguila {
		t.Fatalf("letter=A: %v", got)
	}

	if got := ids(get("sort=title")); len(got) != 6 ||
		got[0] != number || got[1] != aguila || got[2] != episode ||
		got[3] != matrix || got[4] != nandu || got[5] != padrino {
		t.Fatalf("sort=title: %v", got)
	}

	combined := get("genre=drama&letter=P&q=padrino&kind=movie")
	if combined.Total != 1 || combined.Items[0].ID != padrino {
		t.Fatalf("filtros combinados: %+v", combined)
	}
	if got := ids(get("genre=DRAMA&kind=movie")); len(got) != 2 {
		t.Fatalf("genre=DRAMA: %v", got)
	}
	if got := ids(get("genre=Dram")); len(got) != 0 {
		t.Fatalf("genre parcial no debe coincidir: %v", got)
	}

	status, body := libraryHTTPGet(t, ts.URL+"/api/v1/genres", ts.token)
	if status != http.StatusOK {
		t.Fatalf("géneros: código = %d, cuerpo = %s", status, body)
	}
	var genres genresResponse
	if err := json.Unmarshal(body, &genres); err != nil {
		t.Fatalf("géneros: %v", err)
	}
	if len(genres.Genres) < 2 || genres.Genres[0].Name != "Acción" ||
		genres.Genres[len(genres.Genres)-1].Name != "Drama" {
		t.Fatalf("orden de géneros: %+v", genres.Genres)
	}
	counts := map[string]int64{}
	for _, genre := range genres.Genres {
		counts[genre.Name] = genre.Count
	}
	if counts["Drama"] != 2 || counts["Comedia"] != 1 || counts["Acción"] != 1 {
		t.Fatalf("conteos de géneros: %+v", counts)
	}

	status, body = libraryHTTPGet(t, ts.URL+"/api/v1/genres?kind=episode", ts.token)
	if status != http.StatusOK {
		t.Fatalf("géneros de episodios: código = %d", status)
	}
	if err := json.Unmarshal(body, &genres); err != nil {
		t.Fatalf("géneros de episodios: %v", err)
	}
	if len(genres.Genres) != 1 || genres.Genres[0].Name != "Drama" || genres.Genres[0].Count != 1 {
		t.Fatalf("géneros de episodios inesperados: %+v", genres.Genres)
	}

	for _, invalid := range []string{"?kind=serie", "?kind="} {
		status, body := libraryHTTPGet(t, ts.URL+"/api/v1/genres"+invalid, ts.token)
		if invalid == "?kind=" {
			if status != http.StatusOK {
				t.Fatalf("kind vacío: código = %d, cuerpo = %s", status, body)
			}
			continue
		}
		if status != http.StatusBadRequest {
			t.Fatalf("%s: código = %d, cuerpo = %s", invalid, status, body)
		}
	}

	if status, body := libraryHTTPGet(t, ts.URL+"/api/v1/genres", ""); status != http.StatusUnauthorized {
		t.Fatalf("géneros sin sesión: código = %d, cuerpo = %s", status, body)
	}
}
