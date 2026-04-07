package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFutbolRoutes_QueryNameAndPathStability(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/fixtures":
			_, _ = w.Write([]byte(`{"results":0,"response":[]}`))
		case "/leagues":
			_, _ = w.Write([]byte(`{"response":[]}`))
		default:
			_, _ = w.Write([]byte(`{"response":[]}`))
		}
	}))
	defer upstream.Close()

	mockCache := &MockCache{
		existsFunc: func(ctx context.Context, key string) (bool, error) { return false, nil },
		getFunc:    func(ctx context.Context, key string, value interface{}) error { return nil },
		setFunc:    func(ctx context.Context, key string, value interface{}, ttl time.Duration) error { return nil },
	}

	handler := New(Config{
		Cache:              mockCache,
		FootballAPIKey:     "test-key",
		APIFootballBaseURL: upstream.URL,
	})

	tests := []struct {
		name       string
		url        string
		wantStatus int
		wantError  string
	}{
		{
			name:       "matches wrong query name still enforces date",
			url:        "/futbol/matches?day=2026-04-07",
			wantStatus: http.StatusBadRequest,
			wantError:  "date parameter is required",
		},
		{
			name:       "lineup wrong query name still enforces match_id",
			url:        "/futbol/lineup?fixture_id=123",
			wantStatus: http.StatusBadRequest,
			wantError:  "match_id is required",
		},
		{
			name:       "leagues route remains reachable",
			url:        "/futbol/leagues",
			wantStatus: http.StatusOK,
		},
		{
			name:       "unknown path under futbol remains 404",
			url:        "/futbol/match",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected %d got %d body=%s", tc.wantStatus, rec.Code, rec.Body.String())
			}
			if tc.wantError != "" {
				var body map[string]any
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("expected JSON error body: %v", err)
				}
				got, _ := body["error"].(string)
				if got != tc.wantError {
					t.Fatalf("expected error %q got %q", tc.wantError, got)
				}
			}
		})
	}
}
