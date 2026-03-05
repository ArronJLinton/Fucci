package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const headtoheadResponse = `{
  "response": [
    {
      "fixture": {"id": 1, "date": "2024-01-15T15:00:00+00:00"},
      "goals": {"home": 2, "away": 1},
      "teams": {"home": {"name": "Team A"}, "away": {"name": "Team B"}}
    },
    {
      "fixture": {"id": 2, "date": "2023-10-10T14:00:00+00:00"},
      "goals": {"home": 0, "away": 0},
      "teams": {"home": {"name": "Team A"}, "away": {"name": "Team B"}}
    }
  ]
}`

const standingsResponse = `{
  "response": [
    {
      "league": {
        "id": 39,
        "name": "Premier League",
        "standings": [
          [
            {"rank": 1, "team": {"name": "Arsenal"}, "points": 50},
            {"rank": 2, "team": {"name": "Liverpool"}, "points": 48}
          ]
        ]
      }
    }
  ]
}`

func TestFetchHeadToHead_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fixtures/headtohead" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(headtoheadResponse))
	}))
	defer server.Close()

	config := &Config{
		APIFootballBaseURL: server.URL,
		FootballAPIKey:     "test-key",
		Cache:             nil,
	}

	ctx := context.Background()
	summary, err := config.FetchHeadToHead(ctx, 33, 34)
	if err != nil {
		t.Fatalf("FetchHeadToHead: %v", err)
	}
	if summary == "" {
		t.Error("expected non-empty HeadToHead summary")
	}
	if len(summary) < 20 {
		t.Errorf("summary too short: %q", summary)
	}
}

func TestFetchHeadToHead_CacheHit(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(headtoheadResponse))
	}))
	defer server.Close()

	cached := "2024-01-15: Team A 2-1 Team B"
	mockCache := &MockCache{
		existsFunc: func(ctx context.Context, key string) (bool, error) {
			return key == "h2h:33-34", nil
		},
		getFunc: func(ctx context.Context, key string, value interface{}) error {
			if key == "h2h:33-34" {
				if p, ok := value.(*string); ok {
					*p = cached
				}
			}
			return nil
		},
		setFunc: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error { return nil },
	}

	config := &Config{
		APIFootballBaseURL: server.URL,
		FootballAPIKey:     "test-key",
		Cache:             mockCache,
	}
	ctx := context.Background()
	summary, err := config.FetchHeadToHead(ctx, 33, 34)
	if err != nil {
		t.Fatalf("FetchHeadToHead: %v", err)
	}
	if summary != cached {
		t.Errorf("expected cached summary %q, got %q", cached, summary)
	}
	if called {
		t.Error("expected cache hit, server should not have been called")
	}
}

func TestFetchLeagueStandings_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/standings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(standingsResponse))
	}))
	defer server.Close()

	config := &Config{
		APIFootballBaseURL: server.URL,
		FootballAPIKey:     "test-key",
		Cache:             nil,
	}

	ctx := context.Background()
	data, err := config.GetLeagueStandingsData(ctx, "39", "2024")
	if err != nil {
		t.Fatalf("GetLeagueStandingsData: %v", err)
	}
	summary := FormatLeagueStandingsSummary(data)
	if summary == "" {
		t.Error("expected non-empty LeagueTable summary")
	}
	if len(summary) < 10 {
		t.Errorf("summary too short: %q", summary)
	}
}

func TestFetchLeagueStandings_CacheHit(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(standingsResponse))
	}))
	defer server.Close()

	// GetLeagueStandingsData uses cache key league_standings:39:2024 and stores full GetLeagueStandingsResponse
	var cachedData GetLeagueStandingsResponse
	if err := json.Unmarshal([]byte(standingsResponse), &cachedData); err != nil {
		t.Fatalf("unmarshal standings: %v", err)
	}

	mockCache := &MockCache{
		existsFunc: func(ctx context.Context, key string) (bool, error) {
			return key == "league_standings:39:2024", nil
		},
		getFunc: func(ctx context.Context, key string, value interface{}) error {
			if key == "league_standings:39:2024" {
				if p, ok := value.(*GetLeagueStandingsResponse); ok {
					*p = cachedData
				}
			}
			return nil
		},
		setFunc: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error { return nil },
	}

	config := &Config{
		APIFootballBaseURL: server.URL,
		FootballAPIKey:     "test-key",
		Cache:             mockCache,
	}

	ctx := context.Background()
	data, err := config.GetLeagueStandingsData(ctx, "39", "2024")
	if err != nil {
		t.Fatalf("GetLeagueStandingsData: %v", err)
	}
	summary := FormatLeagueStandingsSummary(data)
	expected := "1. Arsenal 50 pts\n2. Liverpool 48 pts"
	if summary != expected {
		t.Errorf("expected summary %q, got %q", expected, summary)
	}
	if called {
		t.Error("expected cache hit, server should not have been called")
	}
}

func TestAggregateMatchData_SetsH2HAndStandingsWhenIDsPresent(t *testing.T) {
	var headtoheadCalled, standingsCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/fixtures/headtohead":
			headtoheadCalled = true
			w.Write([]byte(headtoheadResponse))
		case "/standings":
			standingsCalled = true
			w.Write([]byte(standingsResponse))
		default:
			w.Write([]byte("{}"))
		}
	}))
	defer server.Close()

	config := &Config{
		APIFootballBaseURL: server.URL,
		FootballAPIKey:     "test-key",
		Cache:             nil,
		RapidAPIKey:       "key",
	}
	dda := NewDebateDataAggregator(config)

	req := MatchDataRequest{
		MatchID:      "100",
		HomeTeam:     "Team A",
		AwayTeam:     "Team B",
		Date:         "2025-02-01T15:00:00Z",
		Status:       "NS",
		LeagueID:     39,
		SeasonYear:   2024,
		HomeTeamID:   33,
		AwayTeamID:   34,
	}

	ctx := context.Background()
	matchData, err := dda.AggregateMatchData(ctx, req)
	if err != nil {
		t.Fatalf("AggregateMatchData: %v", err)
	}
	if !headtoheadCalled {
		t.Error("expected fetchHeadToHead to be called")
	}
	if !standingsCalled {
		t.Error("expected fetchLeagueStandings to be called")
	}
	if matchData.HeadToHeadSummary == "" {
		t.Error("expected HeadToHeadSummary to be set")
	}
	if matchData.LeagueTableSummary == "" {
		t.Error("expected LeagueTableSummary to be set")
	}
}
