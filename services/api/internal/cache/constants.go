package cache

import (
	"strings"
	"time"
)

// Cache TTL constants for different types of data
const (
	LiveMatchTTL      = 2 * time.Minute
	PostponedMatchTTL = 30 * time.Minute
	FixtureTTL        = 6 * time.Hour
	TeamInfoTTL    = 24 * time.Hour
	LeagueTableTTL = 12 * time.Hour
	LineupTTL      = 12 * time.Hour
	MatchStatsTTL  = 12 * time.Hour // Fixture statistics cache
	StandingsTTL   = 6 * time.Hour
	H2HTTL         = 6 * time.Hour    // Head-to-head fixtures cache
	NewsTTL        = 15 * time.Minute // News feed cache TTL (matches frontend React Query gcTime/cache time)
	// YouTubeShortsTTL caches team Shorts lists (per lookup_key, daily bucket).
	YouTubeShortsTTL = 24 * time.Hour
	DefaultTTL       = 1 * time.Hour
)

// Match status constants (legacy normalized labels; API-Football uses short codes).
const (
	StatusLive      = "LIVE"
	StatusInPlay    = "IN_PLAY"
	StatusScheduled = "SCHEDULED"
	StatusFinished  = "FINISHED"
)

func normalizeMatchStatus(status string) string {
	return strings.ToUpper(strings.TrimSpace(status))
}

// isLiveMatchStatus reports whether the fixture is in progress (API-Football short codes).
func isLiveMatchStatus(status string) bool {
	switch normalizeMatchStatus(status) {
	case "1H", "2H", "HT", "ET", "P", "BT", "LIVE", "SUSP", "INT", StatusInPlay:
		return true
	default:
		return false
	}
}

// isScheduledMatchStatus reports fixtures that have not started yet.
func isScheduledMatchStatus(status string) bool {
	switch normalizeMatchStatus(status) {
	case "NS", "TBD", StatusScheduled:
		return true
	default:
		return false
	}
}

// isPostponedMatchStatus reports fixtures delayed with a new kickoff pending.
func isPostponedMatchStatus(status string) bool {
	switch normalizeMatchStatus(status) {
	case "PST", "POSTPONED":
		return true
	default:
		return false
	}
}

// isFinishedMatchStatus reports completed or abandoned fixtures.
func isFinishedMatchStatus(status string) bool {
	switch normalizeMatchStatus(status) {
	case "FT", "AET", "PEN", "FT_PEN", "AET_PEN", "AWD", "WO", "CANC", "ABD", StatusFinished:
		return true
	default:
		return false
	}
}

// GetMatchTTL returns the appropriate TTL based on match status.
func GetMatchTTL(status string) time.Duration {
	switch {
	case isLiveMatchStatus(status):
		return LiveMatchTTL
	case isScheduledMatchStatus(status):
		return FixtureTTL
	case isFinishedMatchStatus(status):
		return TeamInfoTTL
	case isPostponedMatchStatus(status):
		return PostponedMatchTTL
	default:
		return DefaultTTL
	}
}
