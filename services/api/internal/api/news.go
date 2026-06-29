package api

import (
	"log"
	"net/http"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/cache"
	"github.com/ArronJLinton/fucci-api/internal/news"
)

func emptyMatchNewsResponse() news.MatchNewsAPIResponse {
	return news.MatchNewsAPIResponse{
		Articles: []news.NewsArticle{},
		Cached:   false,
	}
}

// getFootballNews handles GET /api/news/football
// Fetches football news from the configured news provider with caching
func (c *Config) getFootballNews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Generate cache key
	cacheKey := news.GenerateCacheKey()

	// Try to get from cache first
	var cachedResponse news.NewsAPIResponse
	exists, err := c.Cache.Exists(ctx, cacheKey)
	if err != nil {
		log.Printf("Cache check error: %v\n", err)
	} else if exists {
		err = c.Cache.Get(ctx, cacheKey, &cachedResponse)
		if err == nil {
			cachedResponse.Cached = true
			respondWithJSON(w, http.StatusOK, cachedResponse)
			return
		}
		log.Printf("Cache get error: %v\n", err)
		exists = false // no usable cached data for fallback
	}

	// Create news client (optional custom base URL for tests)
	k := c.newsXAPIKey()
	newsClient := news.NewClient(k)
	if c.NewsBaseURL != "" {
		newsClient = news.NewClientWithBaseURL(k, c.NewsBaseURL)
	}

	todayAndHistoryResp, err := newsClient.FetchTodayAndHistoryNews(ctx)
	if err != nil {
		log.Printf("Failed to fetch news: %v", err)

		// If we have cached data, return it even if stale
		if exists {
			cachedResponse.Cached = true
			respondWithJSON(w, http.StatusServiceUnavailable, cachedResponse)
			return
		}

		// No cached data available, return error
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch news from external API")
		return
	}

	// Transform both responses to internal format
	transformedResponse, err := news.TransformTodayAndHistoryResponse(todayAndHistoryResp)
	if err != nil {
		log.Printf("Failed to transform news response: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to process news data")
		return
	}

	// Cache the response for 15 minutes (store the original cached timestamp)
	transformedResponse.CachedAt = time.Now().UTC().Format(time.RFC3339)
	err = c.Cache.Set(ctx, cacheKey, transformedResponse, cache.NewsTTL)
	if err != nil {
		log.Printf("Cache set error: %v\n", err)
		// Continue even if caching fails
	}
	// Return the response
	transformedResponse.Cached = false
	respondWithJSON(w, http.StatusOK, transformedResponse)
}

// getMatchNews handles GET /api/news/football/match
// Fetches match-specific news for both teams playing
// Query params: homeTeam, awayTeam, matchId, matchStatus, matchEndTime (ISO8601, optional for completed matches)
//
// Reads/writes the per-match cache key shared with news.GetMatchNewsCached, so
// the debate-generation pre-warm and this handler reuse one Redis entry.
func (c *Config) getMatchNews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	homeTeam := r.URL.Query().Get("homeTeam")
	awayTeam := r.URL.Query().Get("awayTeam")
	matchID := r.URL.Query().Get("matchId")
	matchStatus := r.URL.Query().Get("matchStatus")
	matchEndTimeStr := r.URL.Query().Get("matchEndTime")

	if homeTeam == "" || awayTeam == "" || matchID == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required parameters: homeTeam, awayTeam, matchId")
		return
	}

	if c.newsXAPIKey() == "" {
		log.Printf("match news: NEWS_API_KEY/RAPID_API_KEY not configured; returning empty articles for match %s", matchID)
		respondWithJSON(w, http.StatusOK, emptyMatchNewsResponse())
		return
	}

	// Parse matchEndTime for completed matches (optional; nil when missing or invalid)
	var matchEndTime *time.Time
	if matchEndTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, matchEndTimeStr); err == nil {
			matchEndTime = &t
		} else if t, err := time.Parse("2006-01-02T15:04:05Z", matchEndTimeStr); err == nil {
			matchEndTime = &t
		}
	}

	k := c.newsXAPIKey()
	newsClient := news.NewClient(k)
	if c.NewsBaseURL != "" {
		newsClient = news.NewClientWithBaseURL(k, c.NewsBaseURL)
	}

	const matchNewsLimit = 10
	resp, _, err := news.GetMatchNewsCached(
		ctx,
		c.Cache,
		newsClient,
		matchID, homeTeam, awayTeam, matchStatus, matchEndTimeStr,
		matchEndTime,
		matchNewsLimit,
		cache.NewsTTL,
	)
	if err != nil {
		log.Printf("Failed to fetch match news for match %s: %v", matchID, err)

		// Serve stale cache when upstream is down so match details still show prior headlines.
		cacheKey := news.GenerateMatchCacheKey(matchID, matchStatus, matchEndTimeStr)
		var stale news.MatchNewsAPIResponse
		if c.Cache != nil {
			if exists, _ := c.Cache.Exists(ctx, cacheKey); exists {
				if getErr := c.Cache.Get(ctx, cacheKey, &stale); getErr == nil && stale.CachedAt != "" {
					stale.Cached = true
					respondWithJSON(w, http.StatusOK, stale)
					return
				}
			}
		}

		log.Printf("match news: upstream unavailable for match %s; returning empty articles", matchID)
		respondWithJSON(w, http.StatusOK, emptyMatchNewsResponse())
		return
	}

	respondWithJSON(w, http.StatusOK, resp)
}
