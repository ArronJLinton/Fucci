package cache

import "testing"

func TestGetMatchTTL_liveStatuses(t *testing.T) {
	live := []string{
		"1H", "2H", "HT", "ET", "P", "BT", "LIVE", "SUSP", "INT",
		StatusInPlay,
		"1h", " ht ", "live",
	}
	for _, status := range live {
		if got := GetMatchTTL(status); got != LiveMatchTTL {
			t.Errorf("GetMatchTTL(%q) = %v, want LiveMatchTTL (%v)", status, got, LiveMatchTTL)
		}
	}
}

func TestGetMatchTTL_scheduledStatuses(t *testing.T) {
	scheduled := []string{"NS", "TBD", StatusScheduled, "ns", " tbd "}
	for _, status := range scheduled {
		if got := GetMatchTTL(status); got != FixtureTTL {
			t.Errorf("GetMatchTTL(%q) = %v, want FixtureTTL (%v)", status, got, FixtureTTL)
		}
	}
}

func TestGetMatchTTL_finishedStatuses(t *testing.T) {
	finished := []string{
		"FT", "AET", "PEN", "FT_PEN", "AET_PEN", "AWD", "WO", "CANC", "ABD",
		StatusFinished, "ft", " aet ",
	}
	for _, status := range finished {
		if got := GetMatchTTL(status); got != TeamInfoTTL {
			t.Errorf("GetMatchTTL(%q) = %v, want TeamInfoTTL (%v)", status, got, TeamInfoTTL)
		}
	}
}

func TestGetMatchTTL_defaultStatuses(t *testing.T) {
	defaults := []string{"", "UNKNOWN"}
	for _, status := range defaults {
		if got := GetMatchTTL(status); got != DefaultTTL {
			t.Errorf("GetMatchTTL(%q) = %v, want DefaultTTL (%v)", status, got, DefaultTTL)
		}
	}
}

func TestGetMatchTTL_postponedStatuses(t *testing.T) {
	postponed := []string{"PST", "POSTPONED", "pst", " postponed "}
	for _, status := range postponed {
		if got := GetMatchTTL(status); got != PostponedMatchTTL {
			t.Errorf("GetMatchTTL(%q) = %v, want PostponedMatchTTL (%v)", status, got, PostponedMatchTTL)
		}
	}
}

func TestGetMatchTTL_liveBeatsDefault(t *testing.T) {
	// Regression: live short codes must not fall through to DefaultTTL (1hr).
	if LiveMatchTTL >= DefaultTTL {
		t.Fatalf("LiveMatchTTL must be shorter than DefaultTTL for regression test")
	}
	if got := GetMatchTTL("1H"); got != LiveMatchTTL {
		t.Errorf("1H should use LiveMatchTTL, got %v", got)
	}
}
