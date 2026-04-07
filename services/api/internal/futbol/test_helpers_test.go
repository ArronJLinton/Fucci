package futbol

import (
	"context"
	"time"
)

type fakeCache struct {
	existsFunc func(ctx context.Context, key string) (bool, error)
	getFunc    func(ctx context.Context, key string, dest interface{}) error
	setFunc    func(ctx context.Context, key string, value interface{}, expiration time.Duration) error
}

func (f *fakeCache) Exists(ctx context.Context, key string) (bool, error) {
	if f.existsFunc == nil {
		return false, nil
	}
	return f.existsFunc(ctx, key)
}

func (f *fakeCache) Get(ctx context.Context, key string, dest interface{}) error {
	if f.getFunc == nil {
		return nil
	}
	return f.getFunc(ctx, key, dest)
}

func (f *fakeCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if f.setFunc == nil {
		return nil
	}
	return f.setFunc(ctx, key, value, expiration)
}

type fakeProvider struct {
	fetchMatchesFunc        func(ctx context.Context, date, leagueID string) (MatchesDTO, map[string]any, error)
	fetchLineupFunc         func(ctx context.Context, matchID string) (map[string]any, error)
	fetchLeaguesFunc        func(ctx context.Context, season string) (map[string]any, error)
	fetchTeamStandingsFunc  func(ctx context.Context, teamID string, season int) (map[string]any, error)
	fetchLeagueStandingsFun func(ctx context.Context, leagueID, season string) (LeagueStandingsDTO, error)
	fetchMatchStatsFunc     func(ctx context.Context, matchID string) (map[string]any, error)
	fetchH2HFunc            func(ctx context.Context, homeTeamID, awayTeamID string) (map[string]any, error)
}

func (f *fakeProvider) FetchMatches(ctx context.Context, date string, leagueID string) (MatchesDTO, map[string]any, error) {
	return f.fetchMatchesFunc(ctx, date, leagueID)
}
func (f *fakeProvider) FetchLineup(ctx context.Context, matchID string) (map[string]any, error) {
	return f.fetchLineupFunc(ctx, matchID)
}
func (f *fakeProvider) FetchLeagues(ctx context.Context, season string) (map[string]any, error) {
	return f.fetchLeaguesFunc(ctx, season)
}
func (f *fakeProvider) FetchTeamStandings(ctx context.Context, teamID string, season int) (map[string]any, error) {
	return f.fetchTeamStandingsFunc(ctx, teamID, season)
}
func (f *fakeProvider) FetchLeagueStandings(ctx context.Context, leagueID string, season string) (LeagueStandingsDTO, error) {
	return f.fetchLeagueStandingsFun(ctx, leagueID, season)
}
func (f *fakeProvider) FetchMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	return f.fetchMatchStatsFunc(ctx, matchID)
}
func (f *fakeProvider) FetchHeadToHead(ctx context.Context, homeTeamID string, awayTeamID string) (map[string]any, error) {
	return f.fetchH2HFunc(ctx, homeTeamID, awayTeamID)
}
