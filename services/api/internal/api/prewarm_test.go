package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/ai"
)

func TestParsePrewarmLeagueIDs(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []int
	}{
		{name: "empty disables", in: "", want: nil},
		{name: "whitespace disables", in: "   ", want: nil},
		{name: "single id", in: "1", want: []int{1}},
		{name: "WC plus club leagues", in: "1,39,140,135,78,61,2", want: []int{1, 39, 140, 135, 78, 61, 2}},
		{name: "tolerates whitespace and trailing comma", in: " 1 , 39 ,140,", want: []int{1, 39, 140}},
		{name: "skips garbage entries", in: "1,foo,2", want: []int{1, 2}},
		{name: "all garbage returns nil", in: "foo,bar", want: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParsePrewarmLeagueIDs(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ParsePrewarmLeagueIDs(%q) = %v; want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsPreMatchStatus(t *testing.T) {
	preMatch := []string{"NS", "TBD", "ns", " tbd "}
	for _, s := range preMatch {
		if !isPreMatchStatus(s) {
			t.Errorf("isPreMatchStatus(%q) = false; want true", s)
		}
	}
	notPreMatch := []string{"", "FT", "AET", "PEN", "1H", "HT", "2H", "LIVE", "PST", "CANC", "ABD"}
	for _, s := range notPreMatch {
		if isPreMatchStatus(s) {
			t.Errorf("isPreMatchStatus(%q) = true; want false", s)
		}
	}
}

// --- PrewarmJob.Run end-to-end tests -----------------------------------------
//
// These tests exercise the full job loop with httptest servers standing in for
// API-Football, the news upstream, and our own /v1/api/debates/generate-set
// endpoint (the "loopback"). They assert the externally visible behavior of
// the feature: which leagues are scanned, which fixtures get a debate
// generation request, and which Redis keys get the daily SetNX lock.

// fixturesPayload returns a minimal API-Football /fixtures response with the
// given fixtures. Each fixture spec is (id, statusShort, homeTeam, awayTeam).
type fxSpec struct {
	id         int
	status     string
	home, away string
}

func fixturesPayload(t *testing.T, specs []fxSpec) string {
	t.Helper()
	type entry map[string]any
	out := make([]entry, 0, len(specs))
	for _, f := range specs {
		out = append(out, entry{
			"fixture": entry{
				"id":     f.id,
				"date":   time.Now().UTC().Format(time.RFC3339),
				"status": entry{"short": f.status, "long": f.status},
			},
			"league": entry{"id": 1, "name": "FIFA World Cup", "season": 2026},
			"teams": entry{
				"home": entry{"id": 100, "name": f.home},
				"away": entry{"id": 200, "name": f.away},
			},
		})
	}
	payload := entry{
		"get":      "fixtures",
		"errors":   []any{},
		"results":  len(out),
		"paging":   entry{"current": 1, "total": 1},
		"response": out,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal fixtures payload: %v", err)
	}
	return string(b)
}

// prewarmTestServers stands up the three upstreams the job talks to and
// returns the counters/atomics needed for assertions.
type prewarmTestServers struct {
	footballSrv *httptest.Server
	newsSrv     *httptest.Server
	loopbackSrv *httptest.Server

	fixturesCalls   int32
	newsCalls       int32
	loopbackCalls   int32
	loopbackMatches []string
	loopbackMu      sync.Mutex
	loopbackStatus  int // status code the loopback should return (default 201)
}

func newPrewarmTestServers(t *testing.T, fixturesBody string) *prewarmTestServers {
	t.Helper()
	s := &prewarmTestServers{loopbackStatus: http.StatusCreated}

	s.footballSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/fixtures") {
			http.NotFound(w, r)
			return
		}
		atomic.AddInt32(&s.fixturesCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixturesBody))
	}))

	s.newsSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&s.newsCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"OK","request_id":"t","data":[]}`))
	}))

	s.loopbackSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/api/debates/generate-set" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		atomic.AddInt32(&s.loopbackCalls, 1)
		s.loopbackMu.Lock()
		if mid, ok := body["match_id"].(string); ok {
			s.loopbackMatches = append(s.loopbackMatches, mid)
		}
		s.loopbackMu.Unlock()
		w.WriteHeader(s.loopbackStatus)
		_, _ = w.Write([]byte(`{"debates":[]}`))
	}))
	return s
}

func (s *prewarmTestServers) Close() {
	s.footballSrv.Close()
	s.newsSrv.Close()
	s.loopbackSrv.Close()
}

// newPrewarmConfig wires the test servers into an api.Config the way main.go
// would in production (sans Postgres, which the prewarm path doesn't touch).
func newPrewarmConfig(t *testing.T, srvs *prewarmTestServers, mockCache *MockCache) *Config {
	t.Helper()
	return &Config{
		Cache:              mockCache,
		FootballAPIKey:     "test-football-key",
		APIFootballBaseURL: srvs.footballSrv.URL,
		RapidAPIKey:        "test-rapid-key",
		NewsBaseURL:        srvs.newsSrv.URL,
		AIPromptGenerator:  ai.NewPromptGenerator("", "", nil),
	}
}

// newPrewarmJobFromServers constructs a PrewarmJob that targets the test
// loopback server (rather than parsing $PORT to build 127.0.0.1:{PORT}).
func newPrewarmJobFromServers(c *Config, srvs *prewarmTestServers, leagueIDs []int) *PrewarmJob {
	j := NewPrewarmJob(c, leagueIDs, "0")
	j.LoopbackBaseURL = srvs.loopbackSrv.URL
	j.httpClient = &http.Client{Timeout: 5 * time.Second}
	return j
}

// always-acquire SetNX: lets the run proceed.
func setNXAlwaysAcquire(_ context.Context, _ string, _ time.Duration) (bool, error) {
	return true, nil
}

// MockCache by default has SetNX always returning true; for the "another
// machine already ran" case we patch that behavior locally by wrapping.
type lockOverrideCache struct {
	*MockCache
	setNX func(ctx context.Context, key string, ttl time.Duration) (bool, error)
}

func (c *lockOverrideCache) SetNX(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if c.setNX != nil {
		return c.setNX(ctx, key, ttl)
	}
	return c.MockCache.SetNX(ctx, key, ttl)
}

func baselineMockCache() *MockCache {
	return &MockCache{
		existsFunc: func(_ context.Context, _ string) (bool, error) { return false, nil },
		getFunc:    func(_ context.Context, _ string, _ interface{}) error { return nil },
		setFunc:    func(_ context.Context, _ string, _ interface{}, _ time.Duration) error { return nil },
	}
}

func TestPrewarmJob_Run_NoLeaguesIsNoop(t *testing.T) {
	srvs := newPrewarmTestServers(t, fixturesPayload(t, nil))
	defer srvs.Close()
	cfg := newPrewarmConfig(t, srvs, baselineMockCache())
	job := newPrewarmJobFromServers(cfg, srvs, nil)

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&srvs.fixturesCalls); got != 0 {
		t.Errorf("no leagues configured: expected 0 fixtures calls, got %d", got)
	}
	if got := atomic.LoadInt32(&srvs.loopbackCalls); got != 0 {
		t.Errorf("no leagues configured: expected 0 loopback calls, got %d", got)
	}
}

func TestPrewarmJob_Run_NilAIPromptGeneratorIsNoop(t *testing.T) {
	srvs := newPrewarmTestServers(t, fixturesPayload(t, []fxSpec{
		{id: 1, status: "NS", home: "A", away: "B"},
	}))
	defer srvs.Close()
	cfg := newPrewarmConfig(t, srvs, baselineMockCache())
	cfg.AIPromptGenerator = nil
	job := newPrewarmJobFromServers(cfg, srvs, []int{1})

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&srvs.fixturesCalls); got != 0 {
		t.Errorf("AI not configured: expected 0 fixtures calls, got %d", got)
	}
}

func TestPrewarmJob_Run_AnotherMachineHoldsLock(t *testing.T) {
	srvs := newPrewarmTestServers(t, fixturesPayload(t, []fxSpec{
		{id: 1, status: "NS", home: "A", away: "B"},
	}))
	defer srvs.Close()

	var lockKeyAttempted string
	cfg := newPrewarmConfig(t, srvs, baselineMockCache())
	cfg.Cache = &lockOverrideCache{
		MockCache: baselineMockCache(),
		setNX: func(_ context.Context, key string, _ time.Duration) (bool, error) {
			lockKeyAttempted = key
			return false, nil
		},
	}
	job := newPrewarmJobFromServers(cfg, srvs, []int{1})

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(lockKeyAttempted, "prewarm:debates:date:") {
		t.Errorf("expected daily lock key prefix prewarm:debates:date:, got %q", lockKeyAttempted)
	}
	if got := atomic.LoadInt32(&srvs.fixturesCalls); got != 0 {
		t.Errorf("lock not acquired: expected 0 fixtures calls, got %d", got)
	}
	if got := atomic.LoadInt32(&srvs.loopbackCalls); got != 0 {
		t.Errorf("lock not acquired: expected 0 loopback calls, got %d", got)
	}
}

func TestPrewarmJob_Run_HappyPath_FetchesAndWarms(t *testing.T) {
	srvs := newPrewarmTestServers(t, fixturesPayload(t, []fxSpec{
		{id: 101, status: "NS", home: "Argentina", away: "Croatia"},
		{id: 102, status: "TBD", home: "England", away: "Spain"},
	}))
	defer srvs.Close()
	cfg := newPrewarmConfig(t, srvs, baselineMockCache())
	job := newPrewarmJobFromServers(cfg, srvs, []int{1})

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&srvs.fixturesCalls); got != 1 {
		t.Errorf("expected exactly 1 fixtures call (one league), got %d", got)
	}
	if got := atomic.LoadInt32(&srvs.loopbackCalls); got != 2 {
		t.Errorf("expected 2 debate generate-set calls (one per fixture), got %d", got)
	}
	if got := atomic.LoadInt32(&srvs.newsCalls); got != 2 {
		t.Errorf("expected 2 news warm calls (one per fixture), got %d", got)
	}
	srvs.loopbackMu.Lock()
	defer srvs.loopbackMu.Unlock()
	gotIDs := map[string]bool{}
	for _, id := range srvs.loopbackMatches {
		gotIDs[id] = true
	}
	for _, want := range []string{"101", "102"} {
		if !gotIDs[want] {
			t.Errorf("expected loopback POST for match_id=%s; saw %v", want, srvs.loopbackMatches)
		}
	}
}

func TestPrewarmJob_Run_SkipsLiveAndFinishedFixtures(t *testing.T) {
	srvs := newPrewarmTestServers(t, fixturesPayload(t, []fxSpec{
		{id: 1, status: "NS", home: "A", away: "B"},  // generate
		{id: 2, status: "1H", home: "C", away: "D"},  // skip (live)
		{id: 3, status: "FT", home: "E", away: "F"},  // skip (finished)
		{id: 4, status: "TBD", home: "G", away: "H"}, // generate
		{id: 5, status: "PST", home: "I", away: "J"}, // skip (postponed)
	}))
	defer srvs.Close()
	cfg := newPrewarmConfig(t, srvs, baselineMockCache())
	job := newPrewarmJobFromServers(cfg, srvs, []int{1})

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&srvs.loopbackCalls); got != 2 {
		t.Fatalf("expected 2 debate calls (NS+TBD only); got %d (matches=%v)", got, srvs.loopbackMatches)
	}
	srvs.loopbackMu.Lock()
	defer srvs.loopbackMu.Unlock()
	for _, m := range srvs.loopbackMatches {
		if m == "2" || m == "3" || m == "5" {
			t.Errorf("loopback should not have been called for live/finished fixture %s; saw %v", m, srvs.loopbackMatches)
		}
	}
}

func TestPrewarmJob_Run_BadLoopbackDoesNotAbortOtherFixtures(t *testing.T) {
	srvs := newPrewarmTestServers(t, fixturesPayload(t, []fxSpec{
		{id: 1, status: "NS", home: "A", away: "B"},
		{id: 2, status: "NS", home: "C", away: "D"},
		{id: 3, status: "NS", home: "E", away: "F"},
	}))
	defer srvs.Close()
	cfg := newPrewarmConfig(t, srvs, baselineMockCache())
	srvs.loopbackStatus = http.StatusInternalServerError

	job := newPrewarmJobFromServers(cfg, srvs, []int{1})
	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run should swallow per-fixture loopback errors; got %v", err)
	}
	// Even though each call 5xx'd, we still tried all 3 — proves the iterator
	// does not abort on the first error.
	if got := atomic.LoadInt32(&srvs.loopbackCalls); got != 3 {
		t.Errorf("expected all 3 fixtures attempted despite per-call 5xx; got %d", got)
	}
}

func TestPrewarmJob_Run_MultipleLeaguesIndependent(t *testing.T) {
	srvs := newPrewarmTestServers(t, fixturesPayload(t, []fxSpec{
		{id: 1, status: "NS", home: "A", away: "B"},
	}))
	defer srvs.Close()
	cfg := newPrewarmConfig(t, srvs, baselineMockCache())
	job := newPrewarmJobFromServers(cfg, srvs, []int{1, 39, 140})

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// One fixtures call per league (the mock returns the same payload regardless
	// of league parameter — that's fine; we're asserting iteration count).
	if got := atomic.LoadInt32(&srvs.fixturesCalls); got != 3 {
		t.Errorf("expected 3 fixtures calls (one per league), got %d", got)
	}
}
