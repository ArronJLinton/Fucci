package api

import (
	"encoding/json"
	"net/http"

	"github.com/ArronJLinton/fucci-api/internal/youtube"
)

type mediaShortsResponse struct {
	Outlets []youtube.MediaOutletShorts `json:"outlets"`
}

// GET /v1/api/news/stories/shorts
func (c *Config) getNewsMediaYouTubeShorts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	svc := c.youtubeShortsService()
	store := youtube.MediaChannelStore(c.DB)
	if c.MediaYouTubeChannelStore != nil {
		store = c.MediaYouTubeChannelStore
	}
	outlets := svc.GetMediaOutletsShorts(ctx, store)
	if outlets == nil {
		outlets = []youtube.MediaOutletShorts{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(mediaShortsResponse{Outlets: outlets})
}
