package futbol

import (
	"context"
	"errors"
	"testing"
)

func TestServiceGetLineup_CacheHitSkipsProvider(t *testing.T) {
	cache := &fakeCache{
		existsFunc: func(ctx context.Context, key string) (bool, error) { return true, nil },
		getFunc: func(ctx context.Context, key string, dest interface{}) error {
			m := dest.(*map[string]any)
			*m = map[string]any{"source": "cache"}
			return nil
		},
	}
	provider := &fakeProvider{
		fetchLineupFunc: func(ctx context.Context, matchID string) (map[string]any, error) {
			t.Fatalf("provider should not be called on cache hit")
			return nil, nil
		},
	}
	svc := NewService(provider, cache)

	raw, fromCache, err := svc.GetLineup(context.Background(), "123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !fromCache {
		t.Fatalf("expected fromCache=true")
	}
	if raw["source"] != "cache" {
		t.Fatalf("expected cached payload, got %#v", raw)
	}
}

func TestServiceGetMatches_StaleOnErrorReturnsCached(t *testing.T) {
	cache := &fakeCache{
		existsFunc: func(ctx context.Context, key string) (bool, error) { return true, nil },
		getFunc: func(ctx context.Context, key string, dest interface{}) error {
			m := dest.(*map[string]any)
			*m = map[string]any{"stale": true}
			return nil
		},
	}
	provider := &fakeProvider{
		fetchMatchesFunc: func(ctx context.Context, date, leagueID string) (MatchesDTO, map[string]any, error) {
			return MatchesDTO{}, nil, errors.New("upstream down")
		},
	}
	svc := NewService(provider, cache)

	raw, fromCache, err := svc.GetMatches(context.Background(), "2026-04-06", "")
	if err != nil {
		t.Fatalf("expected stale cache fallback, got err=%v", err)
	}
	if !fromCache {
		t.Fatalf("expected fromCache=true when provider fails")
	}
	if raw["stale"] != true {
		t.Fatalf("expected stale payload, got %#v", raw)
	}
}
