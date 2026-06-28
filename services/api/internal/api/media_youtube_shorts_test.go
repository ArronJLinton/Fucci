package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArronJLinton/fucci-api/internal/database"
	"github.com/ArronJLinton/fucci-api/internal/youtube"
)

type stubMediaChannelStore struct {
	channels []database.MediaYoutubeChannels
}

func (s *stubMediaChannelStore) ListActiveMediaYouTubeChannels(_ context.Context) ([]database.MediaYoutubeChannels, error) {
	return s.channels, nil
}

func TestGetNewsMediaYouTubeShorts_ReturnsOutlets(t *testing.T) {
	short := youtube.Short{VideoID: "vid1", Title: "Highlight", Duration: "PT45S", ThumbnailURL: "https://img/vid1.jpg"}
	cfg := &Config{
		YouTubeShortsService: &youtube.Service{
			Fetcher: &stubShortsFetcher{
				byChannel: map[string][]youtube.Short{
					"UC-FOX":  {short},
					"UC-ESPN": {},
				},
			},
		},
		MediaYouTubeChannelStore: &stubMediaChannelStore{
			channels: []database.MediaYoutubeChannels{
				{LookupKey: "fox_soccer", DisplayName: "FOX SPORTS", ChannelID: "UC-FOX", IsVerified: true, SortOrder: 1},
				{LookupKey: "espn_fc", DisplayName: "ESPN FC", ChannelID: "UC-ESPN", IsVerified: true, SortOrder: 2},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/news/stories/shorts", nil)
	rec := httptest.NewRecorder()
	cfg.getNewsMediaYouTubeShorts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	var body mediaShortsResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Outlets) != 2 {
		t.Fatalf("outlets = %d", len(body.Outlets))
	}
	if !body.Outlets[0].HasShorts || body.Outlets[0].ThumbnailURL == "" {
		t.Fatalf("fox = %+v", body.Outlets[0])
	}
	if body.Outlets[1].HasShorts {
		t.Fatalf("espn should have no shorts: %+v", body.Outlets[1])
	}
}
