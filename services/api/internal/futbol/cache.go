package futbol

import (
	"fmt"
	"strings"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/cache"
)

const cachePrefix = "futbol:v2"

func MatchesCacheKey(date, leagueID string) string {
	date = strings.TrimSpace(date)
	leagueID = strings.TrimSpace(leagueID)
	if leagueID == "" {
		return fmt.Sprintf("%s:matches:date:%s", cachePrefix, date)
	}
	return fmt.Sprintf("%s:matches:date:%s:league:%s", cachePrefix, date, leagueID)
}

func LineupCacheKey(matchID string) string {
	return fmt.Sprintf("%s:lineup:%s", cachePrefix, strings.TrimSpace(matchID))
}

func LeaguesCacheKey(season string) string {
	return fmt.Sprintf("%s:leagues:%s", cachePrefix, strings.TrimSpace(season))
}

func TeamStandingsCacheKey(teamID string, season int) string {
	return fmt.Sprintf("%s:team-standings:%s:%d", cachePrefix, strings.TrimSpace(teamID), season)
}

func LeagueStandingsCacheKey(leagueID, season string) string {
	return fmt.Sprintf("%s:league-standings:%s:%s", cachePrefix, strings.TrimSpace(leagueID), strings.TrimSpace(season))
}

func MatchStatsCacheKey(matchID string) string {
	return fmt.Sprintf("%s:stats:%s", cachePrefix, strings.TrimSpace(matchID))
}

func H2HCacheKey(homeTeamID, awayTeamID string) string {
	return fmt.Sprintf("%s:h2h:%s:%s", cachePrefix, strings.TrimSpace(homeTeamID), strings.TrimSpace(awayTeamID))
}

func TTLForOperation(op Operation, status MatchStatus) time.Duration {
	switch op {
	case OperationMatches:
		switch status {
		case MatchStatusLive, MatchStatusInPlay:
			return cache.LiveMatchTTL
		case MatchStatusScheduled:
			return cache.FixtureTTL
		case MatchStatusFinished:
			return cache.TeamInfoTTL
		default:
			return cache.DefaultTTL
		}
	case OperationLineup:
		return cache.LineupTTL
	case OperationLeagues:
		return cache.LeagueTableTTL
	case OperationStandings:
		return cache.StandingsTTL
	case OperationStats:
		return cache.MatchStatsTTL
	case OperationH2H:
		return cache.H2HTTL
	default:
		return cache.DefaultTTL
	}
}
