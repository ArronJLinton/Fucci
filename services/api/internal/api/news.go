package api

import (
	"log"
	"net/http"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/cache"
	"github.com/ArronJLinton/fucci-api/internal/news"
)

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
	newsClient := news.NewClient(c.RapidAPIKey)
	if c.NEWS_BASE_URL != "" {
		newsClient = news.NewClientWithBaseURL(c.RapidAPIKey, c.NEWS_BASE_URL)
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
func (c *Config) getMatchNews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get query parameters
	homeTeam := r.URL.Query().Get("homeTeam")
	awayTeam := r.URL.Query().Get("awayTeam")
	matchID := r.URL.Query().Get("matchId")
	matchStatus := r.URL.Query().Get("matchStatus")
	matchEndTimeStr := r.URL.Query().Get("matchEndTime")

	// Validate required parameters
	if homeTeam == "" || awayTeam == "" || matchID == "" {
		respondWithError(w, http.StatusBadRequest, "Missing required parameters: homeTeam, awayTeam, matchId")
		return
	}

	// Parse matchEndTime for completed matches (optional - nil if not provided or invalid)
	var matchEndTime *time.Time
	if matchEndTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, matchEndTimeStr); err == nil {
			matchEndTime = &t
		} else if t, err := time.Parse("2006-01-02T15:04:05Z", matchEndTimeStr); err == nil {
			matchEndTime = &t
		}
	}

	// Generate cache key based on match ID, status, and end time
	cacheKey := news.GenerateMatchCacheKey(matchID, matchStatus, matchEndTimeStr)

	// Try to get from cache first
	var cachedResponse news.MatchNewsAPIResponse
	exists, err := c.Cache.Exists(ctx, cacheKey)
	if err != nil {
		log.Printf("Cache check error: %v\n", err)
	} else if exists {
		err = c.Cache.Get(ctx, cacheKey, &cachedResponse)
		if err == nil {
			// Return cached response
			cachedResponse.Cached = true
			respondWithJSON(w, http.StatusOK, cachedResponse)
			return
		}
		log.Printf("Cache get error: %v\n", err)
		exists = false // no usable cached data for fallback
	}

	// Create news client (optional custom base URL for tests)
	newsClient := news.NewClient(c.RapidAPIKey)
	if c.NEWS_BASE_URL != "" {
		newsClient = news.NewClientWithBaseURL(c.RapidAPIKey, c.NEWS_BASE_URL)
	}

	// Fetch match news (combined query for both teams)
	// Default limit to 10 articles
	limit := 10
	matchResp, err := newsClient.FetchMatchNews(ctx, homeTeam, awayTeam, limit, matchStatus, matchEndTime)
	if err != nil {
		log.Printf("Failed to fetch match news: %v", err)

		// If we have cached data, return it even if stale
		if exists {
			cachedResponse.Cached = true
			respondWithJSON(w, http.StatusServiceUnavailable, cachedResponse)
			return
		}

		// No cached data available, return error
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch match news from external API")
		return
	}

	// Transform response to internal format (today's articles only)
	transformedResponse, err := news.TransformMatchNewsResponse(matchResp)
	if err != nil {
		log.Printf("Failed to transform match news response: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to process match news data")
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
