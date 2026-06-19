package youtube

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/cache"
	"github.com/ArronJLinton/fucci-api/internal/database"
)

type mockChannelStore struct {
	channels map[string]database.TeamYoutubeChannels
	err      error
}

func (m *mockChannelStore) GetTeamYouTubeChannelByLookupKey(_ context.Context, lookupKey string) (database.TeamYoutubeChannels, error) {
	if m.err != nil {
		return database.TeamYoutubeChannels{}, m.err
	}
	ch, ok := m.channels[lookupKey]
	if !ok {
		return database.TeamYoutubeChannels{}, errors.New("not found")
	}
	return ch, nil
}

type mockFetcher struct {
	shorts []Short
	err    error
	calls  []string
}

func (m *mockFetcher) FetchShortsForChannel(_ context.Context, channelID string) ([]Short, error) {
	m.calls = append(m.calls, channelID)
	if m.err != nil {
		return nil, m.err
	}
	return m.shorts, nil
}

type mockCache struct {
	data   map[string][]Short
	sets   int
	exists map[string]bool
}

func (m *mockCache) Exists(_ context.Context, key string) (bool, error) {
	if m.exists != nil {
		return m.exists[key], nil
	}
	_, ok := m.data[key]
	return ok, nil
}

func (m *mockCache) Get(_ context.Context, key string, dest interface{}) error {
	shorts, ok := m.data[key]
	if !ok {
		return nil
	}
	if ptr, ok := dest.(*[]Short); ok {
		*ptr = shorts
	}
	return nil
}

func (m *mockCache) Set(_ context.Context, key string, value interface{}, _ time.Duration) error {
	if m.data == nil {
		m.data = map[string][]Short{}
	}
	if shorts, ok := value.([]Short); ok {
		m.data[key] = shorts
	}
	m.sets++
	return nil
}

func (m *mockCache) GetDel(context.Context, string, interface{}) (bool, error) {
	return false, nil
}
func (m *mockCache) Delete(context.Context, string) error          { return nil }
func (m *mockCache) DeletePattern(context.Context, string) error   { return nil }
func (m *mockCache) FlushAll(context.Context) error                { return nil }
func (m *mockCache) HealthCheck(context.Context) error             { return nil }
func (m *mockCache) Incr(context.Context, string) (int64, error)   { return 0, nil }
func (m *mockCache) Expire(context.Context, string, time.Duration) error { return nil }
func (m *mockCache) TTL(context.Context, string) (time.Duration, error)  { return 0, nil }
func (m *mockCache) SetNX(context.Context, string, time.Duration) (bool, error) {
	return false, nil
}
func (m *mockCache) GetStats(context.Context) (map[string]interface{}, error) {
	return nil, nil
}

func TestCacheKey(t *testing.T) {
	day := time.Date(2026, 6, 19, 15, 30, 0, 0, time.UTC)
	got := CacheKey("united states", day)
	want := "youtube:shorts:united states:2026-06-19"
	if got != want {
		t.Fatalf("CacheKey() = %q, want %q", got, want)
	}
}

func TestGetShortsForTeam_CacheHitSkipsFetch(t *testing.T) {
	day := time.Now()
	key := CacheKey("england", day)
	cached := []Short{{VideoID: "abc", Title: "Cached", Duration: "PT30S"}}

	mc := &mockCache{data: map[string][]Short{key: cached}}
	fetch := &mockFetcher{shorts: []Short{{VideoID: "new"}}}

	svc := &Service{
		Channels: &mockChannelStore{
			channels: map[string]database.TeamYoutubeChannels{
				"england": {ChannelID: "UC123", IsVerified: true},
			},
		},
		Cache:   mc,
		Fetcher: fetch,
	}

	got := svc.GetShortsForTeam(context.Background(), "England")
	if len(got) != 1 || got[0].VideoID != "abc" {
		t.Fatalf("cache hit = %+v", got)
	}
	if len(fetch.calls) != 0 {
		t.Fatalf("expected no fetch on cache hit, got %v", fetch.calls)
	}
}

func TestGetShortsForTeam_CacheMissFetchesAndStores(t *testing.T) {
	mc := &mockCache{data: map[string][]Short{}}
	fetch := &mockFetcher{
		shorts: []Short{{VideoID: "vid1", Title: "Short", Duration: "PT45S"}},
	}

	svc := &Service{
		Channels: &mockChannelStore{
			channels: map[string]database.TeamYoutubeChannels{
				"brazil": {ChannelID: "UC-BRA", IsVerified: true},
			},
		},
		Cache:   mc,
		Fetcher: fetch,
		TTL:     time.Hour,
	}

	got := svc.GetShortsForTeam(context.Background(), "Brazil")
	if len(got) != 1 || got[0].VideoID != "vid1" {
		t.Fatalf("fetch result = %+v", got)
	}
	if len(fetch.calls) != 1 || fetch.calls[0] != "UC-BRA" {
		t.Fatalf("fetch calls = %v", fetch.calls)
	}
	if mc.sets != 1 {
		t.Fatalf("expected 1 cache set, got %d", mc.sets)
	}
}

func TestGetShortsForTeam_UnmappedTeamCachesEmpty(t *testing.T) {
	mc := &mockCache{data: map[string][]Short{}}
	fetch := &mockFetcher{shorts: []Short{{VideoID: "x"}}}

	svc := &Service{
		Channels: &mockChannelStore{channels: map[string]database.TeamYoutubeChannels{}},
		Cache:    mc,
		Fetcher:  fetch,
	}

	got := svc.GetShortsForTeam(context.Background(), "Unknown FC")
	if len(got) != 0 {
		t.Fatalf("expected empty, got %+v", got)
	}
	if len(fetch.calls) != 0 {
		t.Fatal("fetch should not run for unmapped team")
	}
	if mc.sets != 1 {
		t.Fatalf("expected negative cache set, got sets=%d", mc.sets)
	}
}

func TestGetShortsForTeam_QuotaErrorCachesEmpty(t *testing.T) {
	mc := &mockCache{data: map[string][]Short{}}
	fetch := &mockFetcher{err: &FetchError{StatusCode: 403, Body: "quotaExceeded"}}

	svc := &Service{
		Channels: &mockChannelStore{
			channels: map[string]database.TeamYoutubeChannels{
				"france": {ChannelID: "UC-FRA", IsVerified: true},
			},
		},
		Cache:   mc,
		Fetcher: fetch,
	}

	got := svc.GetShortsForTeam(context.Background(), "France")
	if len(got) != 0 {
		t.Fatalf("expected empty on quota error, got %+v", got)
	}
	if mc.sets != 1 {
		t.Fatalf("expected empty cache write, sets=%d", mc.sets)
	}
}

func TestGetShortsForTeam_MissingCacheKeyDoesNotTreatAsHit(t *testing.T) {
	// Regression: redis Get on missing key returned nil error; must use Exists first.
	mc := &mockCache{data: map[string][]Short{}}
	fetch := &mockFetcher{shorts: []Short{{VideoID: "fresh"}}}

	svc := &Service{
		Channels: &mockChannelStore{
			channels: map[string]database.TeamYoutubeChannels{
				"united states": {ChannelID: "UC-USA", IsVerified: true},
			},
		},
		Cache:   mc,
		Fetcher: fetch,
	}

	_ = svc.GetShortsForTeam(context.Background(), "USA")
	if len(fetch.calls) == 0 {
		t.Fatal("expected fetch when cache empty")
	}
}

var _ cache.CacheInterface = (*mockCache)(nil)
