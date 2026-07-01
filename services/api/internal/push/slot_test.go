package push

import (
	"testing"
	"time"
)

func TestInDeliveryWindow(t *testing.T) {
	t.Parallel()
	loc := time.UTC
	target := TargetLocalTime{Hour: 18, Minute: 0}
	window := 7 * time.Minute

	atTarget := time.Date(2026, 6, 30, 18, 0, 0, 0, loc)
	if !InDeliveryWindow(atTarget, target, window) {
		t.Fatal("expected in window at target")
	}

	edge := time.Date(2026, 6, 30, 18, 7, 0, 0, loc)
	if !InDeliveryWindow(edge, target, window) {
		t.Fatal("expected in window at +7m edge")
	}

	outside := time.Date(2026, 6, 30, 18, 8, 0, 0, loc)
	if InDeliveryWindow(outside, target, window) {
		t.Fatal("expected outside window at +8m")
	}
}

func TestInDeliveryWindowForTimezone(t *testing.T) {
	t.Parallel()
	// 2026-06-30 16:00 UTC = 12:00 America/New_York (EDT)
	now := time.Date(2026, 6, 30, 16, 0, 0, 0, time.UTC)
	ok, err := InDeliveryWindowForTimezone("America/New_York", TargetLocalTime{Hour: 12, Minute: 0}, DefaultDeliveryWindow, now)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected 12:00 local news window")
	}
}

func TestCurrentScanSlot(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 30, 12, 7, 0, 0, time.UTC)
	if got := CurrentScanSlot(now); got != "2026-06-30T12:00" {
		t.Fatalf("got %q", got)
	}
}

func TestScanLockKey(t *testing.T) {
	t.Parallel()
	key := ScanLockKey(CampaignDebateDaily, ScanSlot("2026-06-30T12:00"))
	if key != "push:scan:debate:daily:2026-06-30T12:00" {
		t.Fatalf("unexpected key %q", key)
	}
}
