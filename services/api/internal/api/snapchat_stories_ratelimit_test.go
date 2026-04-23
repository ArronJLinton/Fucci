package api

import (
	"strings"
	"testing"
	"time"
)

func TestSnapchatMemRL_pruneIfNeededLocked_DropsExpired(t *testing.T) {
	// Do not use t.Parallel: this test and EvictsOldestOverCap mutate package-level
	// snapchatMemPruneAt / snapchatMemMaxKeys, which would race and flake.
	w := time.Minute
	now := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
	m := &snapchatMemRL{
		byKey: map[string]snapchatMemRateEntry{
			"stale": {1, now.Add(-2 * w)},
			"live":  {1, now},
		},
	}
	oldP, oldM := snapchatMemPruneAt, snapchatMemMaxKeys
	snapchatMemPruneAt = 1
	snapchatMemMaxKeys = 1000
	t.Cleanup(func() {
		snapchatMemPruneAt, snapchatMemMaxKeys = oldP, oldM
	})
	m.pruneIfNeededLocked(now, w)
	if len(m.byKey) != 1 {
		t.Fatalf("want 1 entry after expired prune, got %d", len(m.byKey))
	}
	if _, ok := m.byKey["live"]; !ok {
		t.Fatalf("expected live to remain, keys=%+v", m.byKey)
	}
}

func TestSnapchatMemRL_pruneIfNeededLocked_EvictsOldestOverCap(t *testing.T) {
	// See DropsExpired: shared prune globals are not safe with t.Parallel.
	w := time.Minute
	now := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
	m := &snapchatMemRL{byKey: make(map[string]snapchatMemRateEntry)}
	// All still “inside” window, but more keys than cap — should evict oldest by start.
	for i := 0; i < 4; i++ {
		// a oldest (now-5s) … d newest (now-2s)
		m.byKey[string(rune('a'+i))] = snapchatMemRateEntry{1, now.Add(-time.Duration(5-i) * time.Second)}
	}
	oldP, oldM := snapchatMemPruneAt, snapchatMemMaxKeys
	snapchatMemPruneAt = 2
	snapchatMemMaxKeys = 3
	t.Cleanup(func() {
		snapchatMemPruneAt, snapchatMemMaxKeys = oldP, oldM
	})
	m.pruneIfNeededLocked(now, w)
	if len(m.byKey) != 3 {
		t.Fatalf("want 3 after eviction, got %d", len(m.byKey))
	}
	if _, ok := m.byKey["a"]; ok {
		t.Fatal("oldest (a) should be evicted")
	}
}

func TestSnapchatRateLimitLogKey_sanitizes(t *testing.T) {
	secret := "8.8.8.8-or-username"
	userKey := snapchatRLKeyPrefixUser + secret
	ipKey := snapchatRLKeyPrefixIP + secret
	for _, k := range []string{userKey, ipKey} {
		if strings.Contains(snapchatRateLimitLogKey(k), secret) {
			t.Fatalf("log key should not contain raw value: in=%q out=%q", k, snapchatRateLimitLogKey(k))
		}
	}
	lu := snapchatRateLimitLogKey(userKey)
	li := snapchatRateLimitLogKey(ipKey)
	if !strings.HasPrefix(lu, "user#") || !strings.HasPrefix(li, "ip#") {
		t.Fatalf("want user#/ip# prefixes, got %q and %q", lu, li)
	}
	foo := "other:key:foo:bar"
	if g := snapchatRateLimitLogKey(foo); !strings.HasPrefix(g, "key#") {
		t.Fatalf("unknown key shape should use key# hash, got %q", g)
	}
}
