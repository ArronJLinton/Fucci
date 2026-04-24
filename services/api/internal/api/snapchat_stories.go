package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/ArronJLinton/fucci-api/internal/cache"
	"github.com/ArronJLinton/fucci-api/internal/snapchat"
)

// snapchatStoriesCached stores a prior successful upstream snapshot (only HTTP 200 responses are cached).
type snapchatStoriesCached struct {
	HTTPStatus int    `json:"http_status"`
	Body       []byte `json:"body"`
}

func snapchatStoriesCacheKey(usernameNormalized string) string {
	return "snapchat_stories:v1:" + usernameNormalized
}

func snapchatStoriesBodyHasRenderableStories(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	var payload struct {
		Stories []struct {
			SnapURLs *struct {
				MediaURL string `json:"mediaUrl"`
			} `json:"snapUrls"`
		} `json:"stories"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	for _, s := range payload.Stories {
		if s.SnapURLs != nil && strings.TrimSpace(s.SnapURLs.MediaURL) != "" {
			return true
		}
	}
	return false
}

// resolveSnapchatUserStoriesBody applies per-IP/username rate limits, Redis cache, then upstream fetch.
// rateLimited is true when the story budget is exhausted (caller should not treat as upstream failure).
func (c *Config) resolveSnapchatUserStoriesBody(ctx context.Context, rapidKey, clientIP, userNorm string) (body []byte, status int, rateLimited bool, err error) {
	if !snapchatStoriesRateLimitAllow(ctx, c, clientIP, userNorm) {
		return nil, http.StatusTooManyRequests, true, nil
	}

	if c.Cache != nil {
		cacheKey := snapchatStoriesCacheKey(userNorm)
		exists, e := c.Cache.Exists(ctx, cacheKey)
		if e == nil && exists {
			var cached snapchatStoriesCached
			if e := c.Cache.Get(ctx, cacheKey, &cached); e == nil && len(cached.Body) > 0 {
				st := cached.HTTPStatus
				if st == 0 {
					st = http.StatusOK
				}
				return cached.Body, st, false, nil
			}
		}
	}

	fetch := snapchat.FetchUserStories
	if c.SnapchatUserStoriesFetch != nil {
		fetch = c.SnapchatUserStoriesFetch
	}
	body, status, err = fetch(ctx, rapidKey, userNorm)
	if err != nil {
		return nil, snapchat.HTTPStatusForFetchError(err), false, err
	}

	if c.Cache != nil && status == http.StatusOK && len(body) > 0 {
		payload := snapchatStoriesCached{HTTPStatus: status, Body: body}
		if e := c.Cache.Set(ctx, snapchatStoriesCacheKey(userNorm), &payload, cache.SnapchatUserStoriesTTL); e != nil {
			log.Printf("[snapchat] cache set %q: %v", snapchatStoriesCacheKey(userNorm), e)
		}
	}
	return body, status, false, nil
}

// GET /api/snapchat/stories?username=psg — public proxy to RapidAPI host snapchat6.p.rapidapi.com.
// Rate-limited per client IP and per username; successful upstream 200 JSON is cached briefly in Redis.
// Upstream non-200 (e.g. 403/429) is passed through and not cached.
func (c *Config) getSnapchatUserStories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rapidKey := strings.TrimSpace(c.RapidAPIKey)
	if rapidKey == "" {
		respondWithError(w, http.StatusServiceUnavailable, "Snapchat stories are not configured (missing RAPID_API_KEY).")
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		respondWithError(w, http.StatusBadRequest, "query parameter `username` is required")
		return
	}

	userNorm, ok := snapchat.NormalizeSnapchatUsername(username)
	if !ok {
		respondWithError(w, http.StatusBadRequest, "invalid Snapchat username")
		return
	}

	body, status, rateLimited, err := c.resolveSnapchatUserStoriesBody(ctx, rapidKey, clientIP(r), userNorm)
	if rateLimited {
		respondWithError(w, http.StatusTooManyRequests, "Rate limit exceeded; try again later")
		return
	}
	if err != nil {
		respondWithError(w, status, err.Error())
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if status == 0 {
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// snapchatNewsStripLeagues: first row is strip “ALL” (league_id -1, must match mobile
// `NEWS_STRIP_ALL_LEAGUE_ID`); then apps/mobile `LEAGUES` order and API-Football ids.
// Empty SnapchatUsername skips upstream probes (e.g. UCL / international strip pills).
var snapchatNewsStripLeagues = []struct {
	LeagueID         int
	SnapchatUsername string
}{
	{LeagueID: -1, SnapchatUsername: "allarounnd"},
	{LeagueID: 39, SnapchatUsername: "premierleague"},
	{LeagueID: 140, SnapchatUsername: "laliga"},
	{LeagueID: 135, SnapchatUsername: "seriea"},
	{LeagueID: 78, SnapchatUsername: "bundesliga"},
	{LeagueID: 61, SnapchatUsername: "ligue1"},
	{LeagueID: 2, SnapchatUsername: ""},
	{LeagueID: 0, SnapchatUsername: ""},
}

type leagueSnapchatAvailabilityRow struct {
	LeagueID             int    `json:"league_id"`
	SnapchatUsername     string `json:"snapchat_username"`
	HasRenderableStories bool   `json:"has_renderable_stories"`
}

type leagueSnapchatAvailabilityResponse struct {
	Leagues []leagueSnapchatAvailabilityRow `json:"leagues"`
}

// GET /v1/api/snapchat/league-availability — batch probe for renderable Snapchat stories on
// official league accounts used by the news league strip. Reuses the same Redis cache and
// rate limits as GET /v1/api/snapchat/stories.
func (c *Config) getSnapchatLeagueStoriesAvailability(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rapidKey := strings.TrimSpace(c.RapidAPIKey)
	if rapidKey == "" {
		respondWithError(w, http.StatusServiceUnavailable, "Snapchat stories are not configured (missing RAPID_API_KEY).")
		return
	}

	ip := clientIP(r)
	out := make([]leagueSnapchatAvailabilityRow, 0, len(snapchatNewsStripLeagues))

	for _, row := range snapchatNewsStripLeagues {
		rawUser := strings.TrimSpace(row.SnapchatUsername)
		if rawUser == "" {
			out = append(out, leagueSnapchatAvailabilityRow{
				LeagueID:             row.LeagueID,
				SnapchatUsername:     "",
				HasRenderableStories: false,
			})
			continue
		}

		userNorm, ok := snapchat.NormalizeSnapchatUsername(rawUser)
		if !ok {
			out = append(out, leagueSnapchatAvailabilityRow{
				LeagueID:             row.LeagueID,
				SnapchatUsername:     rawUser,
				HasRenderableStories: false,
			})
			continue
		}

		body, status, rateLimited, fetchErr := c.resolveSnapchatUserStoriesBody(ctx, rapidKey, ip, userNorm)
		hasRenderable := false
		if !rateLimited && fetchErr == nil && status == http.StatusOK && snapchatStoriesBodyHasRenderableStories(body) {
			hasRenderable = true
		}
		out = append(out, leagueSnapchatAvailabilityRow{
			LeagueID:             row.LeagueID,
			SnapchatUsername:     userNorm,
			HasRenderableStories: hasRenderable,
		})
	}

	respondWithJSON(w, http.StatusOK, leagueSnapchatAvailabilityResponse{Leagues: out})
}
