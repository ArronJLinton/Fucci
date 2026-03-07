package cache

import "time"

// Cache TTL constants for different types of data
const (
	LiveMatchTTL   = 5 * time.Minute
	FixtureTTL     = 6 * time.Hour
	TeamInfoTTL    = 24 * time.Hour
	LeagueTableTTL = 12 * time.Hour
	LineupTTL       = 12 * time.Hour
	MatchStatsTTL   = 12 * time.Hour // Fixture statistics cache
	StandingsTTL    = 6 * time.Hour
	H2HTTL         = 6 * time.Hour // Head-to-head fixtures cache
	NewsTTL        = 15 * time.Minute // News feed cache TTL (matches frontend React Query gcTime/cache time)
	DefaultTTL     = 1 * time.Hour
)

// Match status constants
const (
	StatusLive      = "LIVE"
	StatusInPlay    = "IN_PLAY"
	StatusScheduled = "SCHEDULED"
	StatusFinished  = "FINISHED"
)

// GetMatchTTL returns the appropriate TTL based on match status
func GetMatchTTL(status string) time.Duration {
	switch status {
	case StatusLive, StatusInPlay:
		return LiveMatchTTL
	case StatusScheduled:
		return FixtureTTL
	case StatusFinished:
		return TeamInfoTTL // Using longer cache for finished matches
	default:
		return DefaultTTL
	}
}
