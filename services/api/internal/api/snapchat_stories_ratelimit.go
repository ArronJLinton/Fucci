package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"sort"
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

// In-memory map bounds (tunable in tests) — when Redis is down, avoid unbounded growth from unique IP/user keys.
var (
	snapchatMemPruneAt = 4096 // run an expired-key sweep
	snapchatMemMaxKeys = 8192 // then evict oldest windows until at or under this
)

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
	allowed := e.count <= maxN
	m.pruneIfNeededLocked(now, window)
	return allowed
}

// pruneIfNeededLocked must run with m.mu held. Removes expired windows, then evict oldest keys if still over cap.
func (m *snapchatMemRL) pruneIfNeededLocked(now time.Time, window time.Duration) {
	if len(m.byKey) <= snapchatMemPruneAt {
		return
	}
	for k, e := range m.byKey {
		if now.Sub(e.start) >= window {
			delete(m.byKey, k)
		}
	}
	if len(m.byKey) <= snapchatMemMaxKeys {
		return
	}
	over := len(m.byKey) - snapchatMemMaxKeys
	if over <= 0 {
		return
	}
	type kv struct {
		k string
		t time.Time
	}
	pairs := make([]kv, 0, len(m.byKey))
	for k, e := range m.byKey {
		pairs = append(pairs, kv{k, e.start})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].t.Before(pairs[j].t) })
	for i := 0; i < over && i < len(pairs); i++ {
		delete(m.byKey, pairs[i].k)
	}
}

// clientIP returns the TCP peer host for rate limiting. We intentionally do not read
// X-Forwarded-For or X-Real-IP: without a trusted reverse proxy that strips/spoofs those,
// clients could forge them and bypass per-IP limits. Behind one or more proxies, ensure
// the platform overwrites RemoteAddr (or add middleware that sets it from trusted hops only).
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

const (
	snapchatRLKeyPrefixUser = "rl:snapchat:user:"
	snapchatRLKeyPrefixIP   = "rl:snapchat:ip:"
)

// snapchatRateLimitLogKey returns a non-identifying label for logs (no raw IP or username in Redis key).
func snapchatRateLimitLogKey(key string) string {
	var kind string
	var tail string
	switch {
	case strings.HasPrefix(key, snapchatRLKeyPrefixUser):
		kind, tail = "user", key[len(snapchatRLKeyPrefixUser):]
	case strings.HasPrefix(key, snapchatRLKeyPrefixIP):
		kind, tail = "ip", key[len(snapchatRLKeyPrefixIP):]
	default:
		h := sha256.Sum256([]byte(key))
		return "key#" + hex.EncodeToString(h[:4])
	}
	h := sha256.Sum256([]byte(tail))
	return kind + "#" + hex.EncodeToString(h[:4])
}

// snapchatStoriesRateLimitAllow enforces per-username then per-IP fixed windows
// using TTL-based counters in Redis when configured.
func snapchatStoriesRateLimitAllow(ctx context.Context, c *Config, ip, userNorm string) bool {
	if userNorm == "" {
		userNorm = "_"
	}
	if ip == "" {
		ip = "unknown"
	}
	if !snapchatRateWindowAllow(ctx, c, snapchatRLKeyPrefixUser+userNorm, snapchatStoriesUserLimitN) {
		return false
	}
	return snapchatRateWindowAllow(ctx, c, snapchatRLKeyPrefixIP+ip, snapchatStoriesIPLimitN)
}

func snapchatRateWindowAllow(ctx context.Context, c *Config, key string, maxN int) bool {
	if c != nil && c.Cache != nil {
		n, err := c.Cache.Incr(ctx, key)
		if err == nil {
			lk := snapchatRateLimitLogKey(key)
			if n == 1 {
				if err := c.Cache.Expire(ctx, key, snapchatStoriesRateWindow); err != nil {
					log.Printf("[snapchat] rate limit Expire %s: %v", lk, err)
				}
			} else {
				ttl, err := c.Cache.TTL(ctx, key)
				if err != nil {
					log.Printf("[snapchat] rate limit TTL %s: %v", lk, err)
				} else if ttl < 0 {
					if err := c.Cache.Expire(ctx, key, snapchatStoriesRateWindow); err != nil {
						log.Printf("[snapchat] rate limit fallback Expire %s: %v", lk, err)
					}
				}
			}
			return n <= int64(maxN)
		}
		log.Printf("[snapchat] rate limit Incr %s: %v; using in-memory fallback", snapchatRateLimitLogKey(key), err)
	}
	return snapchatStoryMem.allow(key, maxN, snapchatStoriesRateWindow)
}
