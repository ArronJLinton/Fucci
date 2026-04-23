package api

import (
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

	if !snapchatStoriesRateLimitAllow(ctx, c, clientIP(r), userNorm) {
		respondWithError(w, http.StatusTooManyRequests, "Rate limit exceeded; try again later")
		return
	}

	if c.Cache != nil {
		cacheKey := snapchatStoriesCacheKey(userNorm)
		exists, err := c.Cache.Exists(ctx, cacheKey)
		if err == nil && exists {
			var cached snapchatStoriesCached
			if err := c.Cache.Get(ctx, cacheKey, &cached); err == nil && len(cached.Body) > 0 {
				w.Header().Add("Content-Type", "application/json")
				status := cached.HTTPStatus
				if status == 0 {
					status = http.StatusOK
				}
				w.WriteHeader(status)
				_, _ = w.Write(cached.Body)
				return
			}
		}
	}

	fetch := snapchat.FetchUserStories
	if c.SnapchatUserStoriesFetch != nil {
		fetch = c.SnapchatUserStoriesFetch
	}
	body, status, err := fetch(ctx, rapidKey, userNorm)
	if err != nil {
		code := snapchat.HTTPStatusForFetchError(err)
		respondWithError(w, code, err.Error())
		return
	}

	if c.Cache != nil && status == http.StatusOK && len(body) > 0 {
		payload := snapchatStoriesCached{HTTPStatus: status, Body: body}
		if err := c.Cache.Set(ctx, snapchatStoriesCacheKey(userNorm), &payload, cache.SnapchatUserStoriesTTL); err != nil {
			log.Printf("[snapchat] cache set %q: %v", snapchatStoriesCacheKey(userNorm), err)
		}
	}

	w.Header().Add("Content-Type", "application/json")
	if status == 0 {
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
