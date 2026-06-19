package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/cache"
	"github.com/ArronJLinton/fucci-api/internal/youtube"
	"github.com/go-chi/chi"
)

type matchShortsTeamPayload struct {
	LookupKey string          `json:"lookup_key"`
	HasShorts bool            `json:"has_shorts"`
	Shorts    []youtube.Short `json:"shorts"`
}

type matchShortsResponse struct {
	MatchID string `json:"match_id"`
	Teams   struct {
		Home matchShortsTeamPayload `json:"home"`
		Away matchShortsTeamPayload `json:"away"`
	} `json:"teams"`
}

// GET /v1/api/matches/{matchId}/stories/shorts
func (c *Config) getMatchYouTubeShorts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	matchID := chi.URLParam(r, "matchId")
	if matchID == "" {
		respondWithError(w, http.StatusBadRequest, "matchId is required")
		return
	}

	matchInfo, err := c.lookupMatchInfo(ctx, matchID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "match not found")
		return
	}

	svc := c.youtubeShortsService()
	homeKey := youtube.LookupKeyForTeamName(matchInfo.HomeTeam)
	awayKey := youtube.LookupKeyForTeamName(matchInfo.AwayTeam)

	var homeShorts, awayShorts []youtube.Short
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		homeShorts = svc.GetShortsForTeam(ctx, matchInfo.HomeTeam)
	}()
	go func() {
		defer wg.Done()
		awayShorts = svc.GetShortsForTeam(ctx, matchInfo.AwayTeam)
	}()
	wg.Wait()

	if homeShorts == nil {
		homeShorts = []youtube.Short{}
	}
	if awayShorts == nil {
		awayShorts = []youtube.Short{}
	}

	var resp matchShortsResponse
	resp.MatchID = matchID
	resp.Teams.Home = matchShortsTeamPayload{
		LookupKey: homeKey,
		HasShorts: len(homeShorts) > 0,
		Shorts:    homeShorts,
	}
	resp.Teams.Away = matchShortsTeamPayload{
		LookupKey: awayKey,
		HasShorts: len(awayShorts) > 0,
		Shorts:    awayShorts,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (c *Config) lookupMatchInfo(ctx context.Context, matchID string) (*MatchInfo, error) {
	if c.MatchInfoLookup != nil {
		return c.MatchInfoLookup(ctx, matchID)
	}
	return c.getMatchInfo(ctx, matchID)
}

func (c *Config) youtubeShortsService() *youtube.Service {
	if c.YouTubeShortsService != nil {
		return c.YouTubeShortsService
	}
	fetcher := c.YouTubeShortsFetcher
	if fetcher == nil && c.YouTubeAPIKey != "" {
		fetcher = youtube.NewClient(c.YouTubeAPIKey)
	}
	return &youtube.Service{
		Channels: c.DB,
		Cache:    c.Cache,
		Fetcher:  fetcher,
		TTL:      c.youtubeShortsCacheTTL(),
	}
}

func (c *Config) youtubeShortsCacheTTL() time.Duration {
	if c.YouTubeCacheTTLHours > 0 {
		return time.Duration(c.YouTubeCacheTTLHours) * time.Hour
	}
	return cache.YouTubeShortsTTL
}
