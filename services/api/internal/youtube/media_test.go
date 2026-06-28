package youtube

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/database"
)

type mockMediaStore struct {
	channels []database.MediaYoutubeChannels
	err      error
}

func (m *mockMediaStore) ListActiveMediaYouTubeChannels(_ context.Context) ([]database.MediaYoutubeChannels, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.channels, nil
}

func TestMediaCacheKey(t *testing.T) {
	day := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	got := MediaCacheKey("fox_soccer", day)
	want := "youtube:shorts:media:fox_soccer:2026-06-20"
	if got != want {
		t.Fatalf("MediaCacheKey() = %q, want %q", got, want)
	}
}

func TestGetMediaOutletsShorts_ReturnsAllOutlets(t *testing.T) {
	fetch := &mockFetcher{
		shorts: []Short{{VideoID: "abc", Title: "Clip", Duration: "PT30S", ThumbnailURL: "https://img/abc.jpg"}},
	}
	svc := &Service{
		Cache:   &mockCache{data: map[string][]Short{}},
		Fetcher: fetch,
		TTL:     time.Hour,
	}
	store := &mockMediaStore{
		channels: []database.MediaYoutubeChannels{
			{LookupKey: "fox_soccer", DisplayName: "FOX SPORTS", ChannelID: "UC-FOX", IsVerified: true, SortOrder: 1},
			{LookupKey: "espn_fc", DisplayName: "ESPN FC", ChannelID: "UC-ESPN", IsVerified: true, SortOrder: 2},
		},
	}

	got := svc.GetMediaOutletsShorts(context.Background(), store)
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].LookupKey != "fox_soccer" || !got[0].HasShorts || got[0].ThumbnailURL == "" {
		t.Fatalf("fox = %+v", got[0])
	}
	if got[1].LookupKey != "espn_fc" || !got[1].HasShorts {
		t.Fatalf("espn = %+v", got[1])
	}
}

func TestGetMediaOutletsShorts_EmptyWhenStoreFails(t *testing.T) {
	svc := &Service{Fetcher: &mockFetcher{shorts: []Short{{VideoID: "x"}}}}
	got := svc.GetMediaOutletsShorts(context.Background(), &mockMediaStore{err: errors.New("db")})
	if len(got) != 0 {
		t.Fatalf("expected empty, got %+v", got)
	}
}

func TestGetMediaOutletsShorts_OutletWithoutShortsStillReturned(t *testing.T) {
	svc := &Service{
		Cache:   &mockCache{data: map[string][]Short{}},
		Fetcher: &mockFetcher{shorts: []Short{}},
	}
	store := &mockMediaStore{
		channels: []database.MediaYoutubeChannels{
			{LookupKey: "fifa", DisplayName: "FIFA", ChannelID: "UC-FIFA", IsVerified: true},
		},
	}
	got := svc.GetMediaOutletsShorts(context.Background(), store)
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].HasShorts || got[0].ThumbnailURL != "" {
		t.Fatalf("expected empty outlet payload, got %+v", got[0])
	}
}
