package futbol

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// APIFootballClient is the default provider implementation.
type APIFootballClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

var _ FutbolProvider = (*APIFootballClient)(nil)

func NewAPIFootballClient(baseURL, apiKey string) *APIFootballClient {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = "https://api-football-v1.p.rapidapi.com/v3"
	}
	return &APIFootballClient{
		baseURL:    baseURL,
		apiKey:     strings.TrimSpace(apiKey),
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *APIFootballClient) requestJSON(ctx context.Context, path string, out any) error {
	if c.apiKey == "" {
		return fmt.Errorf("%w: api key missing", ErrInvalidInput)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-rapidapi-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUpstream, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: status %d: %s", ErrUpstream, resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%w: %v", ErrParse, err)
	}
	return nil
}

func (c *APIFootballClient) FetchMatches(ctx context.Context, date string, leagueID string) (MatchesDTO, map[string]any, error) {
	path := fmt.Sprintf("/fixtures?date=%s", date)
	if leagueID != "" {
		path += fmt.Sprintf("&league=%s&season=2025", leagueID)
	}
	var raw map[string]any
	if err := c.requestJSON(ctx, path, &raw); err != nil {
		return MatchesDTO{}, nil, err
	}
	dto := MatchesFromRaw(raw)
	return dto, raw, nil
}

func (c *APIFootballClient) FetchLineup(ctx context.Context, matchID string) (map[string]any, error) {
	var raw map[string]any
	err := c.requestJSON(ctx, fmt.Sprintf("/fixtures/lineups?fixture=%s", matchID), &raw)
	return raw, err
}

func (c *APIFootballClient) FetchLeagues(ctx context.Context, season string) (map[string]any, error) {
	var raw map[string]any
	err := c.requestJSON(ctx, fmt.Sprintf("/leagues?season=%s", season), &raw)
	return raw, err
}

func (c *APIFootballClient) FetchTeamStandings(ctx context.Context, teamID string, season int) (map[string]any, error) {
	var raw map[string]any
	err := c.requestJSON(ctx, fmt.Sprintf("/standings?season=%d&team=%s", season, teamID), &raw)
	return raw, err
}

func (c *APIFootballClient) FetchLeagueStandings(ctx context.Context, leagueID string, season string) (LeagueStandingsDTO, error) {
	var raw map[string]any
	if err := c.requestJSON(ctx, fmt.Sprintf("/standings?league=%s&season=%s", leagueID, season), &raw); err != nil {
		return LeagueStandingsDTO{}, err
	}
	return LeagueStandingsDTO{LeagueID: leagueID, Season: season, Raw: raw}, nil
}

func (c *APIFootballClient) FetchMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	var raw map[string]any
	err := c.requestJSON(ctx, fmt.Sprintf("/fixtures/statistics?fixture=%s", matchID), &raw)
	return raw, err
}

func (c *APIFootballClient) FetchHeadToHead(ctx context.Context, homeTeamID string, awayTeamID string) (map[string]any, error) {
	var raw map[string]any
	err := c.requestJSON(ctx, fmt.Sprintf("/fixtures/headtohead?h2h=%s-%s", homeTeamID, awayTeamID), &raw)
	return raw, err
}
