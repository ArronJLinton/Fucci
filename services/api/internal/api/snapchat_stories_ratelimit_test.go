package api

import (
	"testing"
	"time"
)

func TestSnapchatMemRL_pruneIfNeededLocked_DropsExpired(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
