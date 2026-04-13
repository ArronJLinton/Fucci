package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/news"
)

// validNewsSearchJSON is a minimal valid search API response the news client can parse.
const validNewsSearchJSON = `{"status":"OK","request_id":"test","data":[{"title":"Test","link":"https://example.com/article/1","snippet":"","photo_url":"","published_datetime_utc":"2025-01-01T12:00:00Z","source_name":"Source","source_url":"https://example.com"}]}`

func TestGetFootballNews_CacheHit(t *testing.T) {
	cacheKey := news.GenerateCacheKey()
	cachedBody := news.NewsAPIResponse{
		TodayArticles:   []news.NewsArticle{{ID: "1", Title: "Cached", SourceURL: "https://example.com", SourceName: "S", PublishedAt: "2025-01-01T12:00:00Z", RelativeTime: "1 hour ago"}},
		HistoryArticles: []news.NewsArticle{},
		Cached:          false,
	}
	mockCache := &MockCache{
		existsFunc: func(ctx context.Context, key string) (bool, error) {
			return key == cacheKey, nil
		},
		getFunc: func(ctx context.Context, key string, value interface{}) error {
			if key != cacheKey {
				return nil
			}
			if ptr, ok := value.(*news.NewsAPIResponse); ok {
				*ptr = cachedBody
			}
			return nil
		},
		setFunc: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
			return nil
		},
	}
	config := &Config{Cache: mockCache, RapidAPIKey: "key"}
	req := httptest.NewRequest(http.MethodGet, "/news/football", nil)
	rec := httptest.NewRecorder()

	config.getFootballNews(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	var body news.NewsAPIResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Cached {
		t.Error("expected cached=true on cache hit")
	}
	if len(body.TodayArticles) != 1 || body.TodayArticles[0].Title != "Cached" {
		t.Errorf("expected cached article, got %+v", body.TodayArticles)
	}
}

func TestGetFootballNews_CacheMiss_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "key" {
			t.Errorf("expected X-API-Key: key, got %q", r.Header.Get("X-API-Key"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validNewsSearchJSON))
	}))
	defer server.Close()

	mockCache := &MockCache{
		existsFunc: func(ctx context.Context, key string) (bool, error) { return false, nil },
		getFunc:    func(ctx context.Context, key string, value interface{}) error { return nil },
		setFunc:    func(ctx context.Context, key string, value interface{}, ttl time.Duration) error { return nil },
	}
	config := &Config{Cache: mockCache, RapidAPIKey: "key", NewsBaseURL: server.URL}
	req := httptest.NewRequest(http.MethodGet, "/news/football", nil)
	rec := httptest.NewRecorder()

	config.getFootballNews(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body news.NewsAPIResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Cached {
		t.Error("expected cached=false on fresh fetch")
	}
}

func TestGetFootballNews_UpstreamNewsFailure_NoCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	mockCache := &MockCache{
		existsFunc: func(ctx context.Context, key string) (bool, error) { return false, nil },
		getFunc:    func(ctx context.Context, key string, value interface{}) error { return nil },
		setFunc:    func(ctx context.Context, key string, value interface{}, ttl time.Duration) error { return nil },
	}
	config := &Config{Cache: mockCache, RapidAPIKey: "key", NewsBaseURL: server.URL}
	req := httptest.NewRequest(http.MethodGet, "/news/football", nil)
	rec := httptest.NewRecorder()

	config.getFootballNews(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 when API fails and no cache, got %d", rec.Code)
	}
}

// Note: 503 with cached fallback is not reachable in the current handler: on cache hit we return 200
// and never call the news client; when we do, we only reach the error path with exists=false
// (because Get failure sets exists=false). So upstream news failure always yields 500 when no cache.

func TestGetMatchNews_MissingParams(t *testing.T) {
	config := &Config{Cache: &MockCache{}, RapidAPIKey: "key"}
	req := httptest.NewRequest(http.MethodGet, "/news/football/match", nil)
	rec := httptest.NewRecorder()

	config.getMatchNews(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing params, got %d", rec.Code)
	}
}

func TestGetMatchNews_CacheHit(t *testing.T) {
	matchID := "123"
	matchStatus := "FT"
	matchEndTime := "2025-01-01T15:00:00Z"
	cacheKey := news.GenerateMatchCacheKey(matchID, matchStatus, matchEndTime)
	cachedBody := news.MatchNewsAPIResponse{
		Articles: []news.NewsArticle{{ID: "1", Title: "Match Cached", SourceURL: "https://example.com", SourceName: "S", PublishedAt: "2025-01-01T12:00:00Z", RelativeTime: "1 hour ago"}},
		Cached:   false,
	}
	mockCache := &MockCache{
		existsFunc: func(ctx context.Context, key string) (bool, error) {
			return key == cacheKey, nil
		},
		getFunc: func(ctx context.Context, key string, value interface{}) error {
			if key != cacheKey {
				return nil
			}
			if ptr, ok := value.(*news.MatchNewsAPIResponse); ok {
				*ptr = cachedBody
			}
			return nil
		},
		setFunc: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
			return nil
		},
	}
	config := &Config{Cache: mockCache, RapidAPIKey: "key"}
	req := httptest.NewRequest(http.MethodGet, "/news/football/match", nil)
	req.URL.RawQuery = "homeTeam=TeamA&awayTeam=TeamB&matchId=" + matchID + "&matchStatus=" + matchStatus + "&matchEndTime=" + matchEndTime
	rec := httptest.NewRecorder()

	config.getMatchNews(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 on cache hit, got %d", rec.Code)
	}
	var body news.MatchNewsAPIResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Cached {
		t.Error("expected cached=true on cache hit")
	}
}

func TestGetMatchNews_UpstreamNewsFailure_NoCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	mockCache := &MockCache{
		existsFunc: func(ctx context.Context, key string) (bool, error) { return false, nil },
		getFunc:    func(ctx context.Context, key string, value interface{}) error { return nil },
		setFunc:    func(ctx context.Context, key string, value interface{}, ttl time.Duration) error { return nil },
	}
	config := &Config{Cache: mockCache, RapidAPIKey: "key", NewsBaseURL: server.URL}
	req := httptest.NewRequest(http.MethodGet, "/news/football/match", nil)
	req.URL.RawQuery = "homeTeam=TeamA&awayTeam=TeamB&matchId=123"
	rec := httptest.NewRecorder()

	config.getMatchNews(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 when API fails and no cache, got %d", rec.Code)
	}
}

// Match-news 503 with cached fallback is similarly unreachable (same flow as football news).

func TestGetMatchNews_CacheMiss_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(validNewsSearchJSON))
	}))
	defer server.Close()

	mockCache := &MockCache{
		existsFunc: func(ctx context.Context, key string) (bool, error) { return false, nil },
		getFunc:    func(ctx context.Context, key string, value interface{}) error { return nil },
		setFunc:    func(ctx context.Context, key string, value interface{}, ttl time.Duration) error { return nil },
	}
	config := &Config{Cache: mockCache, RapidAPIKey: "key", NewsBaseURL: server.URL}
	req := httptest.NewRequest(http.MethodGet, "/news/football/match", nil)
	req.URL.RawQuery = "homeTeam=TeamA&awayTeam=TeamB&matchId=123"
	rec := httptest.NewRecorder()

	config.getMatchNews(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body news.MatchNewsAPIResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Cached {
		t.Error("expected cached=false on fresh fetch")
	}
}
