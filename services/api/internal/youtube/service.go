package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/cache"
	"github.com/ArronJLinton/fucci-api/internal/database"
)

// ChannelStore resolves team lookup keys to YouTube channel registry rows.
type ChannelStore interface {
	GetTeamYouTubeChannelByLookupKey(ctx context.Context, lookupKey string) (database.TeamYoutubeChannels, error)
}

// ShortsFetcher loads Shorts for a YouTube channel (YouTube Data API).
type ShortsFetcher interface {
	FetchShortsForChannel(ctx context.Context, channelID string) ([]Short, error)
}

// Service resolves team names to cached YouTube Shorts.
type Service struct {
	Channels ChannelStore
	Cache    cache.CacheInterface
	Fetcher  ShortsFetcher
	TTL      time.Duration
}

func CacheKey(lookupKey string, day time.Time) string {
	return fmt.Sprintf("youtube:shorts:%s:%s", lookupKey, day.UTC().Format("2006-01-02"))
}

// GetShortsForTeam returns Shorts for a team display name; never surfaces upstream errors to callers.
func (s *Service) GetShortsForTeam(ctx context.Context, teamName string) []Short {
	if s == nil {
		return []Short{}
	}
	lookupKey := LookupKeyForTeamName(teamName)
	if lookupKey == "" {
		return []Short{}
	}

	cacheKey := CacheKey(lookupKey, time.Now())
	if s.Cache != nil {
		exists, err := s.Cache.Exists(ctx, cacheKey)
		if err == nil && exists {
			var cached []Short
			if err := s.Cache.Get(ctx, cacheKey, &cached); err == nil {
				return cached
			}
		}
	}

	if s.Channels == nil || s.Fetcher == nil {
		return []Short{}
	}

	channel, err := s.Channels.GetTeamYouTubeChannelByLookupKey(ctx, lookupKey)
	if err != nil || !channel.IsVerified || channel.ChannelID == "" {
		s.cacheEmpty(ctx, cacheKey)
		return []Short{}
	}

	shorts, err := s.Fetcher.FetchShortsForChannel(ctx, channel.ChannelID)
	if err != nil {
		if IsQuotaExceeded(err) {
			log.Printf("[youtube] quota exceeded for %q: %v", lookupKey, err)
			s.cacheEmpty(ctx, cacheKey)
		} else {
			log.Printf("[youtube] fetch shorts for %q: %v", lookupKey, err)
		}
		return []Short{}
	}
	if shorts == nil {
		shorts = []Short{}
	}

	s.cacheShorts(ctx, cacheKey, shorts)
	return shorts
}

func (s *Service) cacheShorts(ctx context.Context, key string, shorts []Short) {
	if s.Cache == nil {
		return
	}
	ttl := s.TTL
	if ttl <= 0 {
		ttl = cache.YouTubeShortsTTL
	}
	if err := s.Cache.Set(ctx, key, shorts, ttl); err != nil {
		log.Printf("[youtube] cache set %q: %v", key, err)
	}
}

func (s *Service) cacheEmpty(ctx context.Context, key string) {
	if s.Cache == nil {
		return
	}
	ttl := s.TTL
	if ttl <= 0 {
		ttl = cache.YouTubeShortsTTL
	}
	if err := s.Cache.Set(ctx, key, []Short{}, ttl); err != nil {
		log.Printf("[youtube] cache set empty %q: %v", key, err)
	}
}

// MarshalShorts is used in tests and logging.
func MarshalShorts(shorts []Short) ([]byte, error) {
	return json.Marshal(shorts)
}
