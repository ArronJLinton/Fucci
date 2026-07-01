package news

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/cache"
)

// GetFootballNewsCached returns the global today+history feed, using Redis when available.
func GetFootballNewsCached(
	ctx context.Context,
	cacheStore CacheStore,
	client *Client,
	ttl time.Duration,
) (*NewsAPIResponse, bool, error) {
	if client == nil {
		return nil, false, fmt.Errorf("news client is nil")
	}

	cacheKey := GenerateCacheKey()
	if cacheStore != nil {
		exists, exErr := cacheStore.Exists(ctx, cacheKey)
		if exErr != nil {
			log.Printf("news cache exists check error for %s: %v", cacheKey, exErr)
		} else if exists {
			var cached NewsAPIResponse
			if getErr := cacheStore.Get(ctx, cacheKey, &cached); getErr == nil && cached.CachedAt != "" {
				cached.Cached = true
				return &cached, true, nil
			}
		}
	}

	raw, err := client.FetchTodayAndHistoryNews(ctx)
	if err != nil {
		if cacheStore != nil {
			var stale NewsAPIResponse
			if exists, _ := cacheStore.Exists(ctx, cacheKey); exists {
				if getErr := cacheStore.Get(ctx, cacheKey, &stale); getErr == nil && stale.CachedAt != "" {
					stale.Cached = true
					return &stale, true, nil
				}
			}
		}
		return nil, false, err
	}
	transformed, err := TransformTodayAndHistoryResponse(raw)
	if err != nil {
		return nil, false, fmt.Errorf("transform football news: %w", err)
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

// RankedFeed supplies push-ranked football articles with in-process scan caching.
type RankedFeed struct {
	Cache  CacheStore
	Client *Client
	TTL    time.Duration

	mu       sync.Mutex
	articles []NewsArticle
	fetchedAt time.Time
	scanTTL  time.Duration
}

// NewRankedFeed builds a feed helper for push selectors. scanTTL controls how long
// ranked articles are reused within one dispatcher tick (default 14m).
func NewRankedFeed(cacheStore CacheStore, client *Client, ttl, scanTTL time.Duration) *RankedFeed {
	if scanTTL <= 0 {
		scanTTL = 14 * time.Minute
	}
	if ttl <= 0 {
		ttl = cache.NewsTTL
	}
	return &RankedFeed{
		Cache:   cacheStore,
		Client:  client,
		TTL:     ttl,
		scanTTL: scanTTL,
	}
}

// RankedArticles returns merged, ranked articles for push (shared across users in a scan).
func (f *RankedFeed) RankedArticles(ctx context.Context) ([]NewsArticle, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.articles) > 0 && time.Since(f.fetchedAt) < f.scanTTL {
		return f.articles, nil
	}

	resp, _, err := GetFootballNewsCached(ctx, f.Cache, f.Client, f.TTL)
	if err != nil {
		return nil, err
	}
	f.articles = RankArticlesForPush(resp.TodayArticles, resp.HistoryArticles)
	f.fetchedAt = time.Now()
	return f.articles, nil
}

// InvalidateScanCache clears the in-process cache (for tests).
func (f *RankedFeed) InvalidateScanCache() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.articles = nil
	f.fetchedAt = time.Time{}
}
