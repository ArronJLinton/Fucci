package futbol

import (
	"context"
	"errors"
	"testing"
)

func TestService_UsesInjectedProviderForMatches(t *testing.T) {
	providerCalled := false
	svc := NewService(&fakeProvider{
		fetchMatchesFunc: func(ctx context.Context, date, leagueID string) (MatchesDTO, map[string]any, error) {
			providerCalled = true
			return MatchesDTO{
					Results: 1,
					Matches: []MatchDTO{{ID: "1", Status: MatchStatusLive}},
				},
				map[string]any{"results": float64(1), "response": []any{}}, nil
		},
	}, nil)

	raw, fromCache, err := svc.GetMatches(context.Background(), "2026-04-06", "39")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if fromCache {
		t.Fatalf("expected fromCache=false for injected provider call")
	}
	if !providerCalled {
		t.Fatalf("expected injected provider to be called")
	}
	if raw["results"] != float64(1) {
		t.Fatalf("unexpected payload from provider: %#v", raw)
	}
}

func TestService_NormalizesUnknownProviderErrors(t *testing.T) {
	svc := NewService(&fakeProvider{
		fetchMatchStatsFunc: func(ctx context.Context, matchID string) (map[string]any, error) {
			return nil, errors.New("tcp timeout")
		},
	}, nil)

	_, err := svc.FetchMatchStatsData(context.Background(), "200")
	if !errors.Is(err, ErrUpstream) {
		t.Fatalf("expected error to normalize to ErrUpstream, got %v", err)
	}
}
