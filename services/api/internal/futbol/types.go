package futbol

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrUpstream     = errors.New("upstream request failed")
	ErrParse        = errors.New("upstream parse failed")
)

// Operation identifies a futbol cacheable operation category.
type Operation string

const (
	OperationMatches   Operation = "matches"
	OperationLineup    Operation = "lineup"
	OperationLeagues   Operation = "leagues"
	OperationStandings Operation = "standings"
	OperationStats     Operation = "stats"
	OperationH2H       Operation = "h2h"
)

// MatchStatus carries provider-independent status for TTL decisions.
type MatchStatus string

const (
	MatchStatusLive      MatchStatus = "live"
	MatchStatusInPlay    MatchStatus = "in_play"
	MatchStatusScheduled MatchStatus = "scheduled"
	MatchStatusFinished  MatchStatus = "finished"
)

// MatchDTO is a canonical minimal match representation used by service logic.
type MatchDTO struct {
	ID     string
	Status MatchStatus
}

// MatchesDTO is a canonical matches envelope.
type MatchesDTO struct {
	Results int
	Matches []MatchDTO
}

// LeagueStandingsDTO is a canonical standings envelope.
type LeagueStandingsDTO struct {
	LeagueID string
	Season   string
	Raw      map[string]any
}

// FutbolProvider defines the external data source contract.
type FutbolProvider interface {
	FetchMatches(ctx context.Context, date string, leagueID string) (MatchesDTO, map[string]any, error)
	FetchLineup(ctx context.Context, matchID string) (map[string]any, error)
	FetchLeagues(ctx context.Context, season string) (map[string]any, error)
	FetchTeamStandings(ctx context.Context, teamID string, season int) (map[string]any, error)
	FetchLeagueStandings(ctx context.Context, leagueID string, season string) (LeagueStandingsDTO, error)
	FetchMatchStats(ctx context.Context, matchID string) (map[string]any, error)
	FetchHeadToHead(ctx context.Context, homeTeamID string, awayTeamID string) (map[string]any, error)
}

// Cache is the minimal cache surface used by the package.
type Cache interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Exists(ctx context.Context, key string) (bool, error)
}
