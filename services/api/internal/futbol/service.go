package futbol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	provider FutbolProvider
	cache    Cache
	now      func() time.Time
}

func NewService(provider FutbolProvider, cache Cache) *Service {
	return &Service{
		provider: provider,
		cache:    cache,
		now:      time.Now,
	}
}

func (s *Service) GetMatches(ctx context.Context, date, leagueID string) (map[string]any, bool, error) {
	if strings.TrimSpace(date) == "" {
		return nil, false, fmt.Errorf("%w: date is required", ErrInvalidInput)
	}
	key := MatchesCacheKey(date, leagueID)
	var cached map[string]any
	if s.tryCacheGet(ctx, key, &cached) {
		return cached, true, nil
	}

	dto, raw, err := s.provider.FetchMatches(ctx, date, leagueID)
	if err != nil {
		if s.tryCacheGet(ctx, key, &cached) {
			return cached, true, nil
		}
		return nil, false, normalizeProviderError(err)
	}

	ttl := cacheTTLFromMatches(dto)
	_ = s.tryCacheSet(ctx, key, raw, ttl)
	return raw, false, nil
}

func (s *Service) GetLineup(ctx context.Context, matchID string) (map[string]any, bool, error) {
	if strings.TrimSpace(matchID) == "" {
		return nil, false, fmt.Errorf("%w: match_id is required", ErrInvalidInput)
	}
	key := LineupCacheKey(matchID)
	var cached map[string]any
	if s.tryCacheGet(ctx, key, &cached) {
		return cached, true, nil
	}
	raw, err := s.provider.FetchLineup(ctx, matchID)
	if err != nil {
		if s.tryCacheGet(ctx, key, &cached) {
			return cached, true, nil
		}
		return nil, false, normalizeProviderError(err)
	}
	_ = s.tryCacheSet(ctx, key, raw, TTLForOperation(OperationLineup, MatchStatusScheduled))
	return raw, false, nil
}

func (s *Service) GetLeagues(ctx context.Context, season string) (map[string]any, bool, error) {
	if strings.TrimSpace(season) == "" {
		return nil, false, fmt.Errorf("%w: season is required", ErrInvalidInput)
	}
	key := LeaguesCacheKey(season)
	var cached map[string]any
	if s.tryCacheGet(ctx, key, &cached) {
		return cached, true, nil
	}
	raw, err := s.provider.FetchLeagues(ctx, season)
	if err != nil {
		if s.tryCacheGet(ctx, key, &cached) {
			return cached, true, nil
		}
		return nil, false, normalizeProviderError(err)
	}
	_ = s.tryCacheSet(ctx, key, raw, TTLForOperation(OperationLeagues, MatchStatusScheduled))
	return raw, false, nil
}

func (s *Service) GetTeamStandings(ctx context.Context, teamID string, season int) (map[string]any, bool, error) {
	if strings.TrimSpace(teamID) == "" || season == 0 {
		return nil, false, fmt.Errorf("%w: team_id and season are required", ErrInvalidInput)
	}
	key := TeamStandingsCacheKey(teamID, season)
	var cached map[string]any
	if s.tryCacheGet(ctx, key, &cached) {
		return cached, true, nil
	}
	raw, err := s.provider.FetchTeamStandings(ctx, teamID, season)
	if err != nil {
		if s.tryCacheGet(ctx, key, &cached) {
			return cached, true, nil
		}
		return nil, false, normalizeProviderError(err)
	}
	_ = s.tryCacheSet(ctx, key, raw, TTLForOperation(OperationStandings, MatchStatusScheduled))
	return raw, false, nil
}

func (s *Service) GetLeagueStandings(ctx context.Context, leagueID, season string) (LeagueStandingsDTO, bool, error) {
	if strings.TrimSpace(leagueID) == "" || strings.TrimSpace(season) == "" {
		return LeagueStandingsDTO{}, false, fmt.Errorf("%w: league_id and season are required", ErrInvalidInput)
	}
	key := LeagueStandingsCacheKey(leagueID, season)
	var cached LeagueStandingsDTO
	if s.tryCacheGet(ctx, key, &cached) {
		return cached, true, nil
	}
	data, err := s.provider.FetchLeagueStandings(ctx, leagueID, season)
	if err != nil {
		if s.tryCacheGet(ctx, key, &cached) {
			return cached, true, nil
		}
		return LeagueStandingsDTO{}, false, normalizeProviderError(err)
	}
	_ = s.tryCacheSet(ctx, key, data, TTLForOperation(OperationStandings, MatchStatusScheduled))
	return data, false, nil
}

func (s *Service) FetchMatchStatsData(ctx context.Context, matchID string) (map[string]any, error) {
	if strings.TrimSpace(matchID) == "" {
		return nil, fmt.Errorf("%w: match_id is required", ErrInvalidInput)
	}
	key := MatchStatsCacheKey(matchID)
	var cached map[string]any
	if s.tryCacheGet(ctx, key, &cached) {
		return cached, nil
	}
	raw, err := s.provider.FetchMatchStats(ctx, matchID)
	if err != nil {
		return nil, normalizeProviderError(err)
	}
	_ = s.tryCacheSet(ctx, key, raw, TTLForOperation(OperationStats, MatchStatusScheduled))
	return raw, nil
}

func (s *Service) FetchHeadToHead(ctx context.Context, homeTeamID, awayTeamID string) (map[string]any, error) {
	if strings.TrimSpace(homeTeamID) == "" || strings.TrimSpace(awayTeamID) == "" {
		return nil, fmt.Errorf("%w: both team ids are required", ErrInvalidInput)
	}
	key := H2HCacheKey(homeTeamID, awayTeamID)
	var cached map[string]any
	if s.tryCacheGet(ctx, key, &cached) {
		return cached, nil
	}
	raw, err := s.provider.FetchHeadToHead(ctx, homeTeamID, awayTeamID)
	if err != nil {
		return nil, normalizeProviderError(err)
	}
	_ = s.tryCacheSet(ctx, key, raw, TTLForOperation(OperationH2H, MatchStatusScheduled))
	return raw, nil
}

func (s *Service) tryCacheGet(ctx context.Context, key string, dest any) bool {
	if s.cache == nil {
		return false
	}
	ok, err := s.cache.Exists(ctx, key)
	if err != nil || !ok {
		return false
	}
	if err := s.cache.Get(ctx, key, dest); err != nil {
		return false
	}
	return true
}

func (s *Service) tryCacheSet(ctx context.Context, key string, value any, ttl time.Duration) error {
	if s.cache == nil {
		return nil
	}
	return s.cache.Set(ctx, key, value, ttl)
}

func cacheTTLFromMatches(data MatchesDTO) time.Duration {
	ttl := TTLForOperation(OperationMatches, MatchStatusScheduled)
	if len(data.Matches) == 0 {
		return ttl
	}
	for _, m := range data.Matches {
		current := TTLForOperation(OperationMatches, m.Status)
		if current < ttl {
			ttl = current
		}
	}
	return ttl
}

func (s *Service) DebugJSON(v any) string {
	b, _ := json.Marshal(v)
	return fmt.Sprintf("%s", b)
}

func normalizeProviderError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrUpstream) || errors.Is(err, ErrParse) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrUpstream, err)
}
