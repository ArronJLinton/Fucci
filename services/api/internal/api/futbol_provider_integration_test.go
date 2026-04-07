package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArronJLinton/fucci-api/internal/futbol"
)

type testProvider struct {
	matchesCalled bool
}

func (p *testProvider) FetchMatches(ctx context.Context, date string, leagueID string) (futbol.MatchesDTO, map[string]any, error) {
	p.matchesCalled = true
	return futbol.MatchesDTO{
			Results: 1,
			Matches: []futbol.MatchDTO{{ID: "10", Status: futbol.MatchStatusScheduled}},
		},
		map[string]any{
			"get":     "fixtures",
			"results": float64(1),
			"response": []any{
				map[string]any{
					"fixture": map[string]any{
						"id": float64(10),
						"status": map[string]any{
							"short": "NS",
						},
					},
				},
			},
		}, nil
}

func (p *testProvider) FetchLineup(ctx context.Context, matchID string) (map[string]any, error) {
	return map[string]any{"response": []any{}}, nil
}
func (p *testProvider) FetchLeagues(ctx context.Context, season string) (map[string]any, error) {
	return map[string]any{"response": []any{}}, nil
}
func (p *testProvider) FetchTeamStandings(ctx context.Context, teamID string, season int) (map[string]any, error) {
	return map[string]any{"response": []any{}}, nil
}
func (p *testProvider) FetchLeagueStandings(ctx context.Context, leagueID string, season string) (futbol.LeagueStandingsDTO, error) {
	return futbol.LeagueStandingsDTO{LeagueID: leagueID, Season: season, Raw: map[string]any{"response": []any{}}}, nil
}
func (p *testProvider) FetchMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	return map[string]any{"response": []any{}}, nil
}
func (p *testProvider) FetchHeadToHead(ctx context.Context, homeTeamID string, awayTeamID string) (map[string]any, error) {
	return map[string]any{"response": []any{}}, nil
}

func TestFutbolHandler_UsesInjectedProvider(t *testing.T) {
	provider := &testProvider{}
	cfg := &Config{
		Cache:          newSafeMockCache(),
		FootballAPIKey: "test-key",
		FutbolProvider: provider,
	}

	req := httptest.NewRequest(http.MethodGet, "/futbol/matches?date=2026-04-06&league_id=39", nil)
	rec := httptest.NewRecorder()

	cfg.getMatches(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !provider.matchesCalled {
		t.Fatalf("expected injected provider to be called")
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected json response: %v", err)
	}
	if body["results"] != float64(1) {
		t.Fatalf("expected injected provider payload, got %#v", body)
	}
}

func TestFutbolHandler_MapsProviderErrors(t *testing.T) {
	cfg := &Config{
		Cache:          newSafeMockCache(),
		FootballAPIKey: "test-key",
		FutbolProvider: &fakeFutbolProvider{
			fetchMatches: func(ctx context.Context, date, leagueID string) (futbol.MatchesDTO, map[string]any, error) {
				return futbol.MatchesDTO{}, nil, errors.New("provider down")
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/futbol/matches?date=2026-04-06", nil)
	rec := httptest.NewRecorder()
	cfg.getMatches(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 for provider failure, got %d body=%s", rec.Code, rec.Body.String())
	}
}

type fakeFutbolProvider struct {
	fetchMatches func(ctx context.Context, date, leagueID string) (futbol.MatchesDTO, map[string]any, error)
}

func (f *fakeFutbolProvider) FetchMatches(ctx context.Context, date string, leagueID string) (futbol.MatchesDTO, map[string]any, error) {
	return f.fetchMatches(ctx, date, leagueID)
}
func (f *fakeFutbolProvider) FetchLineup(ctx context.Context, matchID string) (map[string]any, error) {
	return map[string]any{"response": []any{}}, nil
}
func (f *fakeFutbolProvider) FetchLeagues(ctx context.Context, season string) (map[string]any, error) {
	return map[string]any{"response": []any{}}, nil
}
func (f *fakeFutbolProvider) FetchTeamStandings(ctx context.Context, teamID string, season int) (map[string]any, error) {
	return map[string]any{"response": []any{}}, nil
}
func (f *fakeFutbolProvider) FetchLeagueStandings(ctx context.Context, leagueID string, season string) (futbol.LeagueStandingsDTO, error) {
	return futbol.LeagueStandingsDTO{LeagueID: leagueID, Season: season, Raw: map[string]any{"response": []any{}}}, nil
}
func (f *fakeFutbolProvider) FetchMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	return map[string]any{"response": []any{}}, nil
}
func (f *fakeFutbolProvider) FetchHeadToHead(ctx context.Context, homeTeamID string, awayTeamID string) (map[string]any, error) {
	return map[string]any{"response": []any{}}, nil
}
