package api

import (
	"context"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Defaults protect RapidAPI quota; overridden in tests via snapchatStoriesIPLimitN / snapchatStoriesUserLimitN.
var (
	snapchatStoriesIPLimitN   = 60
	snapchatStoriesUserLimitN = 40
	snapchatStoriesRateWindow = time.Minute
)

type snapchatMemRateEntry struct {
	count int
	start time.Time
}

// snapchatMemRL is the in-memory fallback when Redis Incr is unavailable (same pattern as comment rate limits).
type snapchatMemRL struct {
	mu    sync.Mutex
	byKey map[string]snapchatMemRateEntry
}

var snapchatStoryMem snapchatMemRL

func resetSnapchatStoryMemRLForTest() {
	snapchatStoryMem.mu.Lock()
	snapchatStoryMem.byKey = nil
	snapchatStoryMem.mu.Unlock()
}

func (m *snapchatMemRL) allow(key string, maxN int, window time.Duration) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.byKey == nil {
		m.byKey = make(map[string]snapchatMemRateEntry)
	}
	now := time.Now()
	e := m.byKey[key]
	if now.Sub(e.start) >= window {
		e = snapchatMemRateEntry{count: 0, start: now}
	}
	e.count++
	m.byKey[key] = e
	return e.count <= maxN
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// snapchatStoriesRateLimitAllow enforces per-username then per-IP sliding windows (Redis when configured).
func snapchatStoriesRateLimitAllow(ctx context.Context, c *Config, ip, userNorm string) bool {
	if userNorm == "" {
		userNorm = "_"
	}
	if ip == "" {
		ip = "unknown"
	}
	if !snapchatRateWindowAllow(ctx, c, "rl:snapchat:user:"+userNorm, snapchatStoriesUserLimitN) {
		return false
	}
	return snapchatRateWindowAllow(ctx, c, "rl:snapchat:ip:"+ip, snapchatStoriesIPLimitN)
}

func snapchatRateWindowAllow(ctx context.Context, c *Config, key string, maxN int) bool {
	if c != nil && c.Cache != nil {
		n, err := c.Cache.Incr(ctx, key)
		if err == nil {
			if n == 1 {
				if err := c.Cache.Expire(ctx, key, snapchatStoriesRateWindow); err != nil {
					log.Printf("[snapchat] rate limit Expire %q: %v", key, err)
				}
			} else {
				ttl, err := c.Cache.TTL(ctx, key)
				if err != nil {
					log.Printf("[snapchat] rate limit TTL %q: %v", key, err)
				} else if ttl < 0 {
					if err := c.Cache.Expire(ctx, key, snapchatStoriesRateWindow); err != nil {
						log.Printf("[snapchat] rate limit fallback Expire %q: %v", key, err)
					}
				}
			}
			return n <= int64(maxN)
		}
		log.Printf("[snapchat] rate limit Incr %q: %v; using in-memory fallback", key, err)
	}
	return snapchatStoryMem.allow(key, maxN, snapchatStoriesRateWindow)
}
