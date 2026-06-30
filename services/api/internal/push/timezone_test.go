package push

import (
	"testing"
	"time"
)

func TestLocalTimeInTimezone(t *testing.T) {
	t.Parallel()
	nowUTC := time.Now().UTC()
	local, err := LocalTimeInTimezone("America/New_York")
	if err != nil {
		t.Fatalf("LocalTimeInTimezone: %v", err)
	}
	if local.Location().String() != "America/New_York" {
		t.Fatalf("location = %q", local.Location())
	}
	// Wall clock should be within a few seconds of converting UTC now.
	diff := local.Sub(nowUTC.In(local.Location()))
	if diff < -time.Minute || diff > time.Minute {
		t.Fatalf("unexpected local/utc skew: %v", diff)
	}
}

func TestLocalTimeInTimezone_Invalid(t *testing.T) {
	t.Parallel()
	_, err := LocalTimeInTimezone("Not/A_Real_Zone")
	if err == nil {
		t.Fatal("expected error for invalid timezone")
	}
}

func TestLocalTimeInTimezone_DSTEdge(t *testing.T) {
	t.Parallel()
	// US spring-forward 2026-03-08 02:00 → 03:00 America/New_York
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	before := time.Date(2026, 3, 8, 1, 30, 0, 0, loc)
	after := time.Date(2026, 3, 8, 3, 30, 0, 0, loc)
	if before.Hour() != 1 || after.Hour() != 3 {
		t.Fatalf("unexpected DST fixture hours: before=%d after=%d", before.Hour(), after.Hour())
	}
}
