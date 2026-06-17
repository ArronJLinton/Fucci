package news

import (
	"context"
	"fmt"
	"log"
	"time"
)

const (
	// CacheKeyPrefix is the prefix for all news cache keys
	CacheKeyPrefix = "news:football"
	// CacheKeyLatest is the key for the latest news fetch
	CacheKeyLatest = "news:football:latest"
	// CacheKeyV2 is the key for the new format with todayArticles and historyArticles
	CacheKeyV2 = "news:football:v2"
)

// GenerateCacheKey creates a cache key for news data
// Format: "news:football:v2" for the new format with separate today/history articles
func GenerateCacheKey() string {
	return CacheKeyV2
}

// GenerateCacheKeyWithTimestamp creates a cache key with a timestamp
// Format: "news:football:{timestamp}"
func GenerateCacheKeyWithTimestamp(timestamp string) string {
	return fmt.Sprintf("%s:%s", CacheKeyPrefix, timestamp)
}

// GenerateMatchCacheKey creates a cache key for match-specific news.
// Includes status and matchEndTime so completed vs not-completed results are cached separately.
// Format: "news:football:match:{matchID}:{status}:{matchEndTime}" (matchEndTime empty if not completed)
func GenerateMatchCacheKey(matchID, matchStatus, matchEndTime string) string {
	return fmt.Sprintf("%s:match:%s:%s:%s", CacheKeyPrefix, matchID, matchStatus, matchEndTime)
}

// CacheStore is the subset of cache operations needed by news helpers.
// Implemented by *internal/cache.Cache; declared here to avoid a news ↔ cache import
// coupling and to ease unit testing with fakes.
type CacheStore interface {
	Exists(ctx context.Context, key string) (bool, error)
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
}

// GetMatchNewsCached returns the transformed match news for a fixture, using the
// per-match cache key shared with the GET /news/football/match handler. Either
// the handler or the debate-data aggregator (or the daily pre-warm) can call
// this; whichever runs first warms the cache for the others.
//
// matchEndTimeStr should be empty unless the match is completed; pass the same
// RFC3339 string the public handler accepts so cache keys line up exactly.
// fromCache is true on a cache hit (no upstream call was made).
func GetMatchNewsCached(
	ctx context.Context,
	cacheStore CacheStore,
	client *Client,
	matchID, homeTeam, awayTeam, matchStatus, matchEndTimeStr string,
	matchEndTime *time.Time,
	limit int,
	ttl time.Duration,
) (resp *MatchNewsAPIResponse, fromCache bool, err error) {
	if client == nil {
		return nil, false, fmt.Errorf("news client is nil")
	}

	cacheKey := GenerateMatchCacheKey(matchID, matchStatus, matchEndTimeStr)

	if cacheStore != nil {
		exists, exErr := cacheStore.Exists(ctx, cacheKey)
		if exErr != nil {
			log.Printf("news cache exists check error for %s: %v", cacheKey, exErr)
		} else if exists {
			var cached MatchNewsAPIResponse
			if getErr := cacheStore.Get(ctx, cacheKey, &cached); getErr == nil {
				cached.Cached = true
				return &cached, true, nil
			} else {
				log.Printf("news cache get error for %s: %v", cacheKey, getErr)
			}
		}
	}

	raw, fetchErr := client.FetchMatchNews(ctx, homeTeam, awayTeam, limit, matchStatus, matchEndTime)
	if fetchErr != nil {
		return nil, false, fetchErr
	}

	transformed, tErr := TransformMatchNewsResponse(raw)
	if tErr != nil {
		return nil, false, fmt.Errorf("transform match news: %w", tErr)
	}

	if cacheStore != nil {
		transformed.CachedAt = time.Now().UTC().Format(time.RFC3339)
		if setErr := cacheStore.Set(ctx, cacheKey, transformed, ttl); setErr != nil {
			log.Printf("news cache set error for %s: %v", cacheKey, setErr)
		}
	}

	transformed.Cached = false
	return transformed, false, nil
}
