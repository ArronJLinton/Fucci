package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newSafeMockCache() *MockCache {
	return &MockCache{
		existsFunc: func(ctx context.Context, key string) (bool, error) { return false, nil },
		getFunc:    func(ctx context.Context, key string, value interface{}) error { return nil },
		setFunc:    func(ctx context.Context, key string, value interface{}, ttl time.Duration) error { return nil },
	}
}

func TestFutbolContract_MissingRequiredParams_ErrorShape(t *testing.T) {
	cfg := &Config{Cache: newSafeMockCache(), FootballAPIKey: "test-key"}

	tests := []struct {
		name     string
		url      string
		handler  func(http.ResponseWriter, *http.Request)
		expected string
	}{
		{
			name:     "matches requires date",
			url:      "/futbol/matches",
			handler:  cfg.getMatches,
			expected: "date parameter is required",
		},
		{
			name:     "lineup requires match_id",
			url:      "/futbol/lineup",
			handler:  cfg.getMatchLineup,
			expected: "match_id is required",
		},
		{
			name:     "league standings requires season",
			url:      "/futbol/league_standings?league_id=39",
			handler:  cfg.getLeagueStandingsByLeagueId,
			expected: "season is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()

			tc.handler(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("expected JSON body, got err=%v", err)
			}
			got, _ := body["error"].(string)
			if got != tc.expected {
				t.Fatalf("expected error %q, got %q", tc.expected, got)
			}
		})
	}
}
