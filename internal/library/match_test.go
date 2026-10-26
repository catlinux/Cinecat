package library

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/catlinux/cinecat/internal/metadata"
)

// matchFixture son respuestas reales de TMDB (búsquedas en es-ES y
// colecciones) grabadas con el prototipo con el que se ajustó la
// identificación, para títulos reales y difíciles de la biblioteca de
// producción. Si la identificación pide una búsqueda que no está grabada, el
// test falla: significa que esta versión ya no hace lo mismo que la ajustada.
type matchFixture struct {
	Cases []struct {
		Title         string `json:"title"`
		Year          int    `json:"year"`
		ExpectedID    int    `json:"expected_id"`
		ExpectedTitle string `json:"expected_title"`
	} `json:"cases"`
	Searches    map[string][]fixtureCandidate `json:"searches"`
	Collections map[string][]fixtureCandidate `json:"collections"`
}

type fixtureCandidate struct {
	ID            int    `json:"id"`
	Title         string `json:"title"`
	OriginalTitle string `json:"original_title"`
	Year          int    `json:"year"`
	Votes         int    `json:"votes"`
}

func (c fixtureCandidate) candidate() metadata.Candidate {
	return metadata.Candidate{TMDBID: c.ID, Title: c.Title, OriginalTitle: c.OriginalTitle, Year: c.Year, VoteCount: c.Votes}
}

type fixtureSearcher struct {
	t       *testing.T
	fixture *matchFixture
	title   string
}

func (s fixtureSearcher) SearchMovies(_ context.Context, query string, year int) ([]metadata.Candidate, error) {
	if year != 0 {
		s.t.Errorf("%q: se buscó con año %d; la identificación busca sin año", s.title, year)
	}
	results, ok := s.fixture.Searches[query]
	if !ok {
		s.t.Errorf("%q: búsqueda no grabada %q", s.title, query)
		return nil, metadata.ErrNotFound
	}
	if len(results) == 0 {
		return nil, metadata.ErrNotFound
	}
	out := make([]metadata.Candidate, len(results))
	for i, result := range results {
		out[i] = result.candidate()
	}
	return out, nil
}

func (s fixtureSearcher) MovieCollection(_ context.Context, movieID int) ([]metadata.Candidate, error) {
	parts, ok := s.fixture.Collections[strconv.Itoa(movieID)]
	if !ok {
		s.t.Errorf("%q: colección no grabada de la película %d", s.title, movieID)
		return nil, nil
	}
	out := make([]metadata.Candidate, len(parts))
	for i, part := range parts {
		out[i] = part.candidate()
	}
	return out, nil
}

func TestIdentifyMovieCasosReales(t *testing.T) {
	data, err := os.ReadFile("testdata/match_fixture.json")
	if err != nil {
		t.Fatalf("leer fixture: %v", err)
	}
	var fixture matchFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("fixture no válido: %v", err)
	}
	if len(fixture.Cases) < 50 {
		t.Fatalf("el fixture solo tiene %d casos", len(fixture.Cases))
	}

	for _, tc := range fixture.Cases {
		name := tc.Title
		if tc.Year > 0 {
			name += " " + strconv.Itoa(tc.Year)
		}
		t.Run(name, func(t *testing.T) {
			searcher := fixtureSearcher{t: t, fixture: &fixture, title: tc.Title}
			got, score, ok, err := IdentifyMovie(context.Background(), searcher, tc.Title, tc.Year)
			if err != nil {
				t.Fatalf("IdentifyMovie: %v", err)
			}
			desc := fmt.Sprintf("%s (%d) [%d] puntuación %.2f", got.Title, got.Year, got.TMDBID, score)
			if tc.ExpectedID == 0 {
				if ok {
					t.Fatalf("debería quedar sin identificar, pero eligió %s", desc)
				}
				return
			}
			if !ok || got.TMDBID != tc.ExpectedID {
				t.Fatalf("se esperaba %s [%d], eligió %s (aceptada: %v)", tc.ExpectedTitle, tc.ExpectedID, desc, ok)
			}
		})
	}
}

func TestMatchVariants(t *testing.T) {
	for _, tc := range []struct {
		title string
		year  int
		want  []string
	}{
		{"La lista de Schindler", 0, []string{"La lista de Schindler"}},
		{"Harry Potter Y El Prisionero De Azkaban 3", 0, []string{
			"Harry Potter Y El Prisionero De Azkaban 3", "Harry Potter Y El Prisionero De Azkaban"}},
		{"El Señor de los Anillos - 3 - El retorno del Rey", 0, []string{
			"El Señor de los Anillos - 3 - El retorno del Rey", "El Señor de los Anillos El retorno del Rey",
			"El retorno del Rey", "El Señor de los Anillos"}},
		{"Ice Age 2 El deshielo", 0, []string{"Ice Age 2 El deshielo", "Ice Age El deshielo"}},
		{"Venganza 1 (Taken", 0, []string{"Venganza 1", "Taken", "Venganza"}},
		{"Blade Runner", 2049, []string{"Blade Runner", "Blade Runner 2049"}},
		{"Dr Strange", 0, []string{"Dr Strange", "Doctor Strange"}},
		{"HarryPotter y el misterio del principe 6", 0, []string{
			"Harry Potter y el misterio del principe 6", "Harry Potter y el misterio del principe"}},
		{"Deadpool 2 Version Extendida", 0, []string{"Deadpool 2", "Deadpool"}},
	} {
		got := matchVariants(tc.title, tc.year)
		if fmt.Sprint(got) != fmt.Sprint(tc.want) {
			t.Errorf("matchVariants(%q, %d) = %q, se esperaba %q", tc.title, tc.year, got, tc.want)
		}
	}
}

func TestSeqRatioComoDifflib(t *testing.T) {
	// Valores de difflib.SequenceMatcher(None, a, b).ratio() en Python.
	for _, tc := range []struct {
		a, b string
		want float64
	}{
		{"einfield", "enfield", 0.9333333333333333},
		{"alien", "aliens", 0.9090909090909091},
		{"abcd", "bcda", 0.75},
		{"rapsody", "rhapsody", 0.9333333333333333},
	} {
		if got := seqRatio(tc.a, tc.b); got-tc.want > 1e-9 || tc.want-got > 1e-9 {
			t.Errorf("seqRatio(%q, %q) = %v, se esperaba %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestNormTitle(t *testing.T) {
	for in, want := range map[string]string{
		"El señor de los anillos: El retorno del rey": "senor de los anillos el retorno del rey",
		"Alien³":                "alien3",
		"Fast & Furious 7":      "fast and furious 7",
		"Dr. Strange":           "doctor strange",
		"Los":                   "los",
		"¿Conoces a Joe Black?": "conoces a joe black",
	} {
		if got := normTitle(in, true); got != want {
			t.Errorf("normTitle(%q) = %q, se esperaba %q", in, got, want)
		}
	}
}
