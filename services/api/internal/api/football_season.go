package api

import (
	"time"
)

// LeagueUEFAChampionsLeague is API-Football v3 id for UEFA Champions League.
// `season` is the competition starting year (e.g. 2025–26 → 2025), same August–July rule
// as other club leagues; optional `season` query on /futbol/matches overrides.
const LeagueUEFAChampionsLeague = 2

// LeagueMLS is API-Football v3 id for Major League Soccer.
// MLS runs on a calendar year; API-Football `season` is that year (2026 → 2026).
const LeagueMLS = 253

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

// Club leagues whose API-Football `season` is the calendar year of the fixture
// (not the European Aug–July start year).
var calendarYearClubLeagueIDs = map[int]struct{}{
	LeagueMLS: {},
}

// IsInternationalFootballLeague is true for World Cup, WCQ confederations, and international friendlies.
// Domestic and club competitions (EPL, UCL, etc.) use the August–July season year rule instead.
func IsInternationalFootballLeague(leagueID int) bool {
	_, ok := internationalFootballLeagueIDs[leagueID]
	return ok
}

// usesCalendarYearSeason is true when API-Football expects season = fixture calendar year.
func usesCalendarYearSeason(leagueID int) bool {
	if IsInternationalFootballLeague(leagueID) {
		return true
	}
	_, ok := calendarYearClubLeagueIDs[leagueID]
	return ok
}

// ResolveAPIFootballSeason returns the `season` query parameter for API-Football fixtures.
// - UEFA Champions League and most domestic club leagues: European-style season starting year
//   (Aug–Dec → year; Jan–Jul → year−1).
// - International tournaments and calendar-year club leagues (e.g. MLS): calendar year of the fixture date.
func ResolveAPIFootballSeason(leagueID int, matchDate time.Time) int {
	y, m, _ := matchDate.Date()
	if usesCalendarYearSeason(leagueID) {
		return y
	}
	if m >= time.August {
		return y
	}
	return y - 1
}
