package api

import (
	"time"
)

// LeagueUEFAChampionsLeague is API-Football v3 id for UEFA Champions League.
// Fixtures for this competition use the calendar year of the match date as `season`
// (e.g. April 2026 knockouts → season=2026) per product/API behavior; optional `season`
// query on /futbol/matches overrides.
const LeagueUEFAChampionsLeague = 2

// API-Football v3 league IDs for international tournaments (national teams).
// See https://www.api-football.com/documentation-v3 — verify via GET /leagues if fixtures look wrong.
const (
	LeagueWorldCup    = 1
	LeagueFriendlies  = 10
	LeagueWCQUEFA     = 32
	LeagueWCQCONMEBOL = 29
	LeagueWCQCAF      = 34
	LeagueWCQAFC      = 35
	LeagueWCQCONCACAF = 36
	LeagueWCQOFC      = 37
)

var internationalFootballLeagueIDs = map[int]struct{}{
	LeagueWorldCup:    {},
	LeagueFriendlies:  {},
	LeagueWCQUEFA:     {},
	LeagueWCQCONMEBOL: {},
	LeagueWCQCAF:      {},
	LeagueWCQAFC:      {},
	LeagueWCQCONCACAF: {},
	LeagueWCQOFC:      {},
}

// IsInternationalFootballLeague is true for World Cup, WCQ confederations, and international friendlies.
// Domestic and club competitions (EPL, UCL, etc.) use the August–July season year rule instead.
func IsInternationalFootballLeague(leagueID int) bool {
	_, ok := internationalFootballLeagueIDs[leagueID]
	return ok
}

// ResolveAPIFootballSeason returns the `season` query parameter for API-Football fixtures.
// - UEFA Champions League: calendar year of the fixture date (aligns with requested match day).
// - International tournaments (national teams): calendar year of the fixture date.
// - Other domestic/club leagues: European-style season starting year (Aug–Dec → year; Jan–Jul → year−1).
func ResolveAPIFootballSeason(leagueID int, matchDate time.Time) int {
	y, m, _ := matchDate.Date()
	if leagueID == LeagueUEFAChampionsLeague {
		return y
	}
	if IsInternationalFootballLeague(leagueID) {
		return y
	}
	if m >= time.August {
		return y
	}
	return y - 1
}
