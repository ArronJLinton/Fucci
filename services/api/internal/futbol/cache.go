package futbol

import (
	"fmt"
	"strings"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/cache"
)

func MatchesCacheKey(date, leagueID string) string {
	date = strings.TrimSpace(date)
	leagueID = strings.TrimSpace(leagueID)
	if leagueID == "" {
		return fmt.Sprintf("matches:date:%s", date)
	}
	return fmt.Sprintf("matches:date:%s:league:%s", date, leagueID)
}

func LineupCacheKey(matchID string) string {
	return fmt.Sprintf("lineup:%s", strings.TrimSpace(matchID))
}

func LeaguesCacheKey(season string) string {
	return fmt.Sprintf("leagues:%s", strings.TrimSpace(season))
}

func TeamStandingsCacheKey(teamID string, season int) string {
	return fmt.Sprintf("team_standings:%s:%d", strings.TrimSpace(teamID), season)
}

func LeagueStandingsCacheKey(leagueID, season string) string {
	return fmt.Sprintf("league_standings:%s:%s", strings.TrimSpace(leagueID), strings.TrimSpace(season))
}

func MatchStatsCacheKey(matchID string) string {
	return fmt.Sprintf("match_stats:%s", strings.TrimSpace(matchID))
}

func H2HCacheKey(homeTeamID, awayTeamID string) string {
	return fmt.Sprintf("h2h:%s-%s", strings.TrimSpace(homeTeamID), strings.TrimSpace(awayTeamID))
}

func TeamSquadCacheKey(teamID string) string {
	return fmt.Sprintf("team_squad:%s", strings.TrimSpace(teamID))
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
