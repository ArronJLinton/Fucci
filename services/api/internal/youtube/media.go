package youtube

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/database"
)

// MediaChannelStore lists active media outlet YouTube channel registry rows.
type MediaChannelStore interface {
	ListActiveMediaYouTubeChannels(ctx context.Context) ([]database.MediaYoutubeChannels, error)
}

// MediaOutletShorts is Shorts for one news/media outlet.
type MediaOutletShorts struct {
	LookupKey     string  `json:"lookup_key"`
	DisplayName   string  `json:"display_name"`
	HasShorts     bool    `json:"has_shorts"`
	ThumbnailURL  string  `json:"thumbnail_url"`
	Shorts        []Short `json:"shorts"`
}

func MediaCacheKey(lookupKey string, day time.Time) string {
	return CacheKey("media:"+lookupKey, day)
}

// GetShortsForMediaChannel returns cached Shorts for a media outlet lookup key.
func (s *Service) GetShortsForMediaChannel(ctx context.Context, lookupKey, channelID string, verified bool) []Short {
	if s == nil || lookupKey == "" {
		return []Short{}
	}

	cacheKey := MediaCacheKey(lookupKey, time.Now())
	if s.Cache != nil {
		exists, err := s.Cache.Exists(ctx, cacheKey)
		if err == nil && exists {
			var cached []Short
			if err := s.Cache.Get(ctx, cacheKey, &cached); err == nil {
				return cached
			}
		}
	}

	if !verified || channelID == "" || s.Fetcher == nil {
		s.cacheEmpty(ctx, cacheKey)
		return []Short{}
	}

	shorts, err := s.Fetcher.FetchShortsForChannel(ctx, channelID)
	if err != nil {
		if IsQuotaExceeded(err) {
			log.Printf("[youtube] quota exceeded for media %q: %v", lookupKey, err)
		} else {
			log.Printf("[youtube] fetch media shorts for %q: %v", lookupKey, err)
		}
		s.cacheEmpty(ctx, cacheKey)
		return []Short{}
	}
	if shorts == nil {
		shorts = []Short{}
	}

	s.cacheShorts(ctx, cacheKey, shorts)
	return shorts
}

// GetMediaOutletsShorts loads Shorts for all active media outlets concurrently.
func (s *Service) GetMediaOutletsShorts(ctx context.Context, store MediaChannelStore) []MediaOutletShorts {
	if s == nil || store == nil {
		return []MediaOutletShorts{}
	}

	channels, err := store.ListActiveMediaYouTubeChannels(ctx)
	if err != nil {
		log.Printf("[youtube] list media channels: %v", err)
		return []MediaOutletShorts{}
	}
	if len(channels) == 0 {
		return []MediaOutletShorts{}
	}

	out := make([]MediaOutletShorts, len(channels))
	var wg sync.WaitGroup
	wg.Add(len(channels))
	for i, ch := range channels {
		i, ch := i, ch
		go func() {
			defer wg.Done()
			shorts := s.GetShortsForMediaChannel(ctx, ch.LookupKey, ch.ChannelID, ch.IsVerified)
			if shorts == nil {
				shorts = []Short{}
			}
			thumb := ""
			if len(shorts) > 0 {
				thumb = shorts[0].ThumbnailURL
			}
			out[i] = MediaOutletShorts{
				LookupKey:    ch.LookupKey,
				DisplayName:  ch.DisplayName,
				HasShorts:    len(shorts) > 0,
				ThumbnailURL: thumb,
				Shorts:       shorts,
			}
		}()
	}
	wg.Wait()
	return out
}
