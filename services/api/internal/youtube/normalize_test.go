package youtube

import "testing"

func TestNormalizeTeamLookupKey(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"England", "england"},
		{"  Brazil  ", "brazil"},
		{"Türkiye", "turkiye"},
		{"United States", "united states"},
		{"South Korea", "south korea"},
	}
	for _, tc := range tests {
		got := NormalizeTeamLookupKey(tc.in)
		if got != tc.want {
			t.Errorf("NormalizeTeamLookupKey(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLookupKeyForTeamName_Aliases(t *testing.T) {
	if got := LookupKeyForTeamName("Turkey"); got != "turkiye" {
		t.Fatalf("Turkey alias = %q", got)
	}
}

func TestParseISO8601DurationSeconds(t *testing.T) {
	tests := []struct {
		iso  string
		want int
	}{
		{"PT58S", 58},
		{"PT1M30S", 90},
		{"PT2M", 120},
		{"", 0},
	}
	for _, tc := range tests {
		if got := ParseISO8601DurationSeconds(tc.iso); got != tc.want {
			t.Errorf("ParseISO8601DurationSeconds(%q) = %d, want %d", tc.iso, got, tc.want)
		}
	}
}

func TestIsShortDuration(t *testing.T) {
	if !IsShortDuration("PT58S", 90) {
		t.Fatal("expected PT58S to be short")
	}
	if IsShortDuration("PT2M", 90) {
		t.Fatal("expected PT2M to exceed cap")
	}
}
