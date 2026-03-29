package api

import (
	"testing"
	"time"
)

func TestResolveAPIFootballSeason_Domestic(t *testing.T) {
	// Premier League (39): March 2026 → season 2025 (2025/26)
	mar := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)
	if g := ResolveAPIFootballSeason(39, mar); g != 2025 {
		t.Fatalf("domestic March: got %d want 2025", g)
	}
	// August 2025 → season 2025
	aug := time.Date(2025, time.August, 1, 0, 0, 0, 0, time.UTC)
	if g := ResolveAPIFootballSeason(39, aug); g != 2025 {
		t.Fatalf("domestic August: got %d want 2025", g)
	}
}

func TestResolveAPIFootballSeason_International(t *testing.T) {
	d := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)
	if g := ResolveAPIFootballSeason(LeagueWorldCup, d); g != 2026 {
		t.Fatalf("World Cup: got %d want 2026", g)
	}
	if g := ResolveAPIFootballSeason(LeagueFriendlies, d); g != 2026 {
		t.Fatalf("Friendlies: got %d want 2026", g)
	}
	if g := ResolveAPIFootballSeason(LeagueWCQUEFA, d); g != 2026 {
		t.Fatalf("WCQ UEFA: got %d want 2026", g)
	}
}
