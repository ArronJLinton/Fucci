package news

import "fmt"

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
