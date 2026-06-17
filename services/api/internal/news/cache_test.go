package news

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// validSearchJSON is a minimal but parseable Real-Time News Data search response.
const validSearchJSON = `{
	"status": "OK",
	"request_id": "test",
	"data": [
		{
			"title": "Goal of the season",
			"link": "https://example.com/article/1",
			"snippet": "Lorem ipsum",
			"photo_url": "",
			"published_datetime_utc": "2026-06-15T12:00:00Z",
			"source_name": "Source",
			"source_url": "https://example.com"
		}
	]
}`

// memCache is a tiny in-memory implementation of CacheStore used to assert
// shared-key behavior without standing up Redis.
type memCache struct {
	mu      sync.Mutex
	entries map[string][]byte
	setHits int32
}

func newMemCache() *memCache {
	return &memCache{entries: make(map[string][]byte)}
}

func (c *memCache) Exists(_ context.Context, key string) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.entries[key]
	return ok, nil
}

func (c *memCache) Get(_ context.Context, key string, dest interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	raw, ok := c.entries[key]
	if !ok {
		return errors.New("memCache: miss")
	}
	return json.Unmarshal(raw, dest)
}

func (c *memCache) Set(_ context.Context, key string, value interface{}, _ time.Duration) error {
	atomic.AddInt32(&c.setHits, 1)
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = raw
	return nil
}

func newsServer(t *testing.T, hitCount *int32, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hitCount != nil {
			atomic.AddInt32(hitCount, 1)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
}

func TestGetMatchNewsCached_CacheMissWritesAndReturns(t *testing.T) {
	var hits int32
	srv := newsServer(t, &hits, validSearchJSON)
	defer srv.Close()

	client := NewClientWithBaseURL("test-key", srv.URL)
	mem := newMemCache()

	resp, fromCache, err := GetMatchNewsCached(
		context.Background(),
		mem,
		client,
		"123", "Home FC", "Away FC", "NS", "",
		nil,
		5,
		15*time.Minute,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fromCache {
		t.Fatal("first call should not be a cache hit")
	}
	if resp == nil || len(resp.Articles) != 1 || resp.Articles[0].Title != "Goal of the season" {
		t.Fatalf("unexpected articles: %+v", resp)
	}
	if resp.Cached {
		t.Error("Cached field should be false on a fresh miss")
	}
	if got := atomic.LoadInt32(&mem.setHits); got != 1 {
		t.Errorf("expected exactly 1 cache Set, got %d", got)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("expected exactly 1 upstream call, got %d", got)
	}

	// Sanity: the key the helper wrote is the same key the public handler reads.
	expectedKey := GenerateMatchCacheKey("123", "NS", "")
	if _, ok := mem.entries[expectedKey]; !ok {
		t.Fatalf("expected cache key %q to be present; have %v", expectedKey, mem.entries)
	}
}

func TestGetMatchNewsCached_CacheHitShortCircuits(t *testing.T) {
	var hits int32
	srv := newsServer(t, &hits, validSearchJSON)
	defer srv.Close()
	client := NewClientWithBaseURL("test-key", srv.URL)
	mem := newMemCache()

	// Prime the cache via a first call.
	if _, _, err := GetMatchNewsCached(
		context.Background(), mem, client,
		"42", "X", "Y", "NS", "", nil, 5, time.Minute,
	); err != nil {
		t.Fatalf("priming call failed: %v", err)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("priming call should have hit upstream once, got %d", hits)
	}

	// Second call must come straight from the cache.
	resp, fromCache, err := GetMatchNewsCached(
		context.Background(), mem, client,
		"42", "X", "Y", "NS", "", nil, 5, time.Minute,
	)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if !fromCache {
		t.Fatal("second call should be a cache hit (fromCache=true)")
	}
	if resp == nil || !resp.Cached {
		t.Errorf("expected Cached=true on cache hit, got resp=%+v", resp)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("upstream must not be called again on cache hit; got %d total hits", got)
	}
}

func TestGetMatchNewsCached_NilClientErrors(t *testing.T) {
	_, _, err := GetMatchNewsCached(
		context.Background(), newMemCache(), nil,
		"1", "A", "B", "NS", "", nil, 5, time.Minute,
	)
	if err == nil {
		t.Fatal("expected error when client is nil")
	}
}

func TestGetMatchNewsCached_UpstreamErrorPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	client := NewClientWithBaseURL("test-key", srv.URL)
	mem := newMemCache()

	_, fromCache, err := GetMatchNewsCached(
		context.Background(), mem, client,
		"1", "A", "B", "NS", "", nil, 5, time.Minute,
	)
	if err == nil {
		t.Fatal("expected error when upstream returns 5xx")
	}
	if fromCache {
		t.Error("fromCache must be false on upstream error")
	}
	if got := atomic.LoadInt32(&mem.setHits); got != 0 {
		t.Errorf("nothing should have been cached on error; got %d Set calls", got)
	}
}

func TestGetMatchNewsCached_NilCacheStillFetches(t *testing.T) {
	var hits int32
	srv := newsServer(t, &hits, validSearchJSON)
	defer srv.Close()
	client := NewClientWithBaseURL("test-key", srv.URL)

	resp, fromCache, err := GetMatchNewsCached(
		context.Background(), nil, client,
		"1", "A", "B", "NS", "", nil, 5, time.Minute,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fromCache {
		t.Error("fromCache must be false when no cache is provided")
	}
	if resp == nil || len(resp.Articles) == 0 {
		t.Error("expected articles from upstream when cache is nil")
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("expected exactly 1 upstream call with nil cache, got %d", got)
	}
}

// TestGetMatchNewsCached_KeyIsSharedAcrossCallers locks down the contract that
// the debate aggregator and the GET /news/football/match handler write to the
// same Redis key for the same logical match. If this test breaks, the news
// cache is no longer shared and the pre-warm story silently regresses.
func TestGetMatchNewsCached_KeyIsSharedAcrossCallers(t *testing.T) {
	var hits int32
	srv := newsServer(t, &hits, validSearchJSON)
	defer srv.Close()
	client := NewClientWithBaseURL("test-key", srv.URL)
	mem := newMemCache()

	// Caller A: aggregator-style (no end-time, pre-match status).
	if _, _, err := GetMatchNewsCached(
		context.Background(), mem, client,
		"7", "Foo", "Bar", "NS", "", nil, 10, time.Minute,
	); err != nil {
		t.Fatalf("aggregator-style call failed: %v", err)
	}

	// Caller B: handler-style (same matchID/status, end-time empty because the
	// match is still pre-kickoff). MUST be a cache hit.
	_, fromCache, err := GetMatchNewsCached(
		context.Background(), mem, client,
		"7", "Foo", "Bar", "NS", "", nil, 10, time.Minute,
	)
	if err != nil {
		t.Fatalf("handler-style call failed: %v", err)
	}
	if !fromCache {
		t.Fatal("handler call should hit the cache key the aggregator just wrote")
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("upstream must be called exactly once across the two callers; got %d", got)
	}
}
