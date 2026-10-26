package library

import (
	"context"
	"testing"

	"github.com/catlinux/cinecat/internal/metadata"
)

type wiringThresholdSearcher struct {
	candidate metadata.Candidate
}

func (s wiringThresholdSearcher) SearchMovies(context.Context, string, int) ([]metadata.Candidate, error) {
	return []metadata.Candidate{s.candidate}, nil
}

func (s wiringThresholdSearcher) SearchTV(context.Context, string, int) ([]metadata.Candidate, error) {
	return []metadata.Candidate{s.candidate}, nil
}

func (s wiringThresholdSearcher) MovieCollection(context.Context, int) ([]metadata.Candidate, error) {
	return nil, nil
}

func TestMatchThresholdConfigurable(t *testing.T) {
	t.Cleanup(func() { SetMatchThreshold(MatchThreshold) })
	searcher := wiringThresholdSearcher{candidate: metadata.Candidate{
		TMDBID: 1, Title: "Alien Resurrection", OriginalTitle: "Alien Resurrection", VoteCount: 1000,
	}}

	SetMatchThreshold(0.75)
	if _, score, ok, err := IdentifyMovie(context.Background(), searcher, "Alien", 0); err != nil ||
		ok || score < 0.65 || score > 0.75 {
		t.Fatalf("con 0.75: ok = %v, puntuación = %.4f, error = %v", ok, score, err)
	}

	SetMatchThreshold(0.65)
	if _, _, ok, err := IdentifyMovie(context.Background(), searcher, "Alien", 0); err != nil || !ok {
		t.Fatalf("con 0.65 la película debe identificarse: ok = %v, error = %v", ok, err)
	}
	if _, _, ok, err := IdentifySeries(context.Background(), searcher, "Alien", 0); err != nil || !ok {
		t.Fatalf("con 0.65 la serie debe identificarse: ok = %v, error = %v", ok, err)
	}

	SetMatchThreshold(0.5)
	if _, _, ok, err := IdentifyMovie(context.Background(), searcher, "Alien", 0); err != nil || !ok {
		t.Fatalf("un umbral fuera de rango debe ignorarse: ok = %v, error = %v", ok, err)
	}
}
