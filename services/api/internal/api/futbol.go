package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/cache"
	"github.com/ArronJLinton/fucci-api/internal/futbol"
)

type GetMatchesParams struct {
	Date string
}

func (c *Config) getMatches(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	queryParams := r.URL.Query()
	date := queryParams.Get("date")
	leagueID := queryParams.Get("league_id")
	if date == "" {
		respondWithError(w, http.StatusBadRequest, "date parameter is required")
		return
	}

	footballAPIKey := c.FootballAPIKey
	if footballAPIKey == "" {
		respondWithError(w, http.StatusBadRequest, "Football API key is required")
		return
	}
	raw, _, err := c.futbolService().GetMatches(ctx, date, leagueID)
	if err != nil {
		respondWithError(w, futbolHTTPStatus(err), fmt.Sprintf("Error creating http request: %s", err))
		return
	}
	var data GetMatchesAPIResponse
	if err := decodeRawMap(raw, &data); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse response from football api service: %s", err))
		return
	}
	respondWithJSON(w, http.StatusOK, data)
}

func (c *Config) getMatch(w http.ResponseWriter, r *http.Request) {
}

// FetchLineupData fetches raw lineup data from API-Football (with cache key lineup_raw:{matchID}).
// Reused by getMatchLineup and by the debate context aggregator.
func (c *Config) FetchLineupData(ctx context.Context, matchID string) (*GetLineUpResponse, error) {
	raw, _, err := c.futbolService().GetLineup(ctx, matchID)
	if err != nil {
		return nil, err
	}
	var data GetLineUpResponse
	if err := decodeRawMap(raw, &data); err != nil {
		return nil, fmt.Errorf("lineup parse: %w", err)
	}
	return &data, nil
}

// FetchMatchStatsData fetches fixture statistics from API-Football (with cache key match_stats:{matchID}).
// Reused by the debate context aggregator.
func (c *Config) FetchMatchStatsData(ctx context.Context, matchID string) (*GetFixtureStatisticsResponse, error) {
	raw, err := c.futbolService().FetchMatchStatsData(ctx, matchID)
	if err != nil {
		return nil, err
	}
	var data GetFixtureStatisticsResponse
	if err := decodeRawMap(raw, &data); err != nil {
		return nil, fmt.Errorf("match stats parse: %w", err)
	}
	return &data, nil
}

func (c *Config) getMatchLineup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	queryParams := r.URL.Query()
	matchID := queryParams.Get("match_id")
	if matchID == "" {
		respondWithError(w, http.StatusBadRequest, "match_id is required")
		return
	}

	// Try enriched cache first (handler response shape)
	cacheKey := futbol.LineupCacheKey(matchID)
	var response struct {
		Home Lineup `json:"home"`
		Away Lineup `json:"away"`
	}
	exists, err := c.Cache.Exists(ctx, cacheKey)
	if err == nil && exists {
		err = c.Cache.Get(ctx, cacheKey, &response)
		if err == nil {
			fmt.Printf("Cache hit for lineup: %s\n", matchID)
			respondWithJSON(w, http.StatusOK, response)
			return
		}
	}

	rawLineup, _, err := c.futbolService().GetLineup(ctx, matchID)
	if err != nil {
		respondWithError(w, futbolHTTPStatus(err), err.Error())
		return
	}
	var lineUpData GetLineUpResponse
	if err := decodeRawMap(rawLineup, &lineUpData); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse lineup response: %s", err))
		return
	}
	getLineUpData := &lineUpData

	fmt.Printf("Number of lineup responses: %d\n", len(getLineUpData.Response))

	if len(getLineUpData.Response) < 2 {
		respondWithJSON(w, http.StatusOK, "No lineup data available")
		return
	}

	// Use the same base URL for squad requests
	homeTeamSquad, err := c.getTeamSquad(int32(getLineUpData.Response[0].Team.ID), ctx)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to get team squad: %s", err))
		return
	}
	awayTeamSquad, err := c.getTeamSquad(int32(getLineUpData.Response[1].Team.ID), ctx)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to get team squad: %s", err))
		return
	}

	// Process lineups and create response
	response = struct {
		Home Lineup `json:"home"`
		Away Lineup `json:"away"`
	}{
		Home: Lineup{
			Starters:    processPlayers(getLineUpData.Response[0].StartXI, homeTeamSquad),
			Substitutes: processSubstitutes(getLineUpData.Response[0].Substitutes, homeTeamSquad),
		},
		Away: Lineup{
			Starters:    processPlayers(getLineUpData.Response[1].StartXI, awayTeamSquad),
			Substitutes: processSubstitutes(getLineUpData.Response[1].Substitutes, awayTeamSquad),
		},
	}

	// Store in cache
	err = c.Cache.Set(ctx, cacheKey, response, cache.LineupTTL)
	if err != nil {
		fmt.Printf("Cache set error: %v\n", err)
	} else {
		fmt.Printf("Stored lineup in cache: %s with TTL: %v\n", cacheKey, cache.LineupTTL)
	}

	respondWithJSON(w, http.StatusOK, response)
}

// Helper functions to process players
func processPlayers(players []struct {
	Player Player `json:"player"`
}, squad *GetSquadResponse) []Player {
	result := make([]Player, 0, len(players))
	for _, p := range players {
		squadPlayer := filterByName(squad.Response[0].Players, p.Player)
		p := Player{
			ID:     p.Player.ID,
			Name:   p.Player.Name,
			Number: p.Player.Number,
			Pos:    p.Player.Pos,
			Grid:   p.Player.Grid,
			Photo:  squadPlayer.Photo,
		}
		result = append(result, p)
	}
	return result
}

func processSubstitutes(substitutes []struct {
	Player struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Number int    `json:"number"`
		Pos    string `json:"pos"`
		Grid   any    `json:"grid"`
		Photo  string `json:"photo"`
	} `json:"player"`
}, squad *GetSquadResponse) []Player {
	result := make([]Player, 0, len(substitutes))
	for _, p := range substitutes {
		squadPlayer := filterByName(squad.Response[0].Players, Player{
			ID:     p.Player.ID,
			Name:   p.Player.Name,
			Number: p.Player.Number,
		})

		p := Player{
			ID:     p.Player.ID,
			Name:   p.Player.Name,
			Number: p.Player.Number,
			Pos:    p.Player.Pos,
			Grid:   "",
			Photo:  squadPlayer.Photo,
		}
		result = append(result, p)
	}
	return result
}

func filterByName(items []Player, player Player) Player {
	// First try to match by ID
	if player.ID != 0 {
		for _, item := range items {
			if item.ID == player.ID {
				return item
			}
		}
	}

	// If ID match fails, try name matching with various normalizations
	normalizedSearchName := normalizeName(player.Name)
	var bestMatch Player
	var maxSimilarity float32 = 0

	for _, item := range items {
		normalizedItemName := normalizeName(item.Name)

		// Try exact match first
		if normalizedItemName == normalizedSearchName {
			return item
		}

		// Check if names contain each other
		if strings.Contains(normalizedItemName, normalizedSearchName) ||
			strings.Contains(normalizedSearchName, normalizedItemName) {
			similarity := float32(len(normalizedItemName)) / float32(len(normalizedSearchName))
			if similarity > maxSimilarity {
				maxSimilarity = similarity
				bestMatch = item
			}
		}

		// If jersey numbers match and names are similar enough, consider it a match
		if player.Number != 0 && item.Number == player.Number {
			return item
		}
	}

	// If we found a good match, return it
	if maxSimilarity > 0.7 {
		return bestMatch
	}

	// Return empty player if no match found
	return Player{}
}

func normalizeName(name string) string {
	return futbol.NormalizeName(name)
}

func (c *Config) getTeamSquad(id int32, ctx context.Context) (*GetSquadResponse, error) {
	// Generate cache key
	cacheKey := futbol.TeamSquadCacheKey(fmt.Sprintf("%d", id))

	// Try to get from cache first
	var squad GetSquadResponse
	exists, err := c.Cache.Exists(ctx, cacheKey)
	if err != nil {
		log.Printf("Cache check error for squad: %v\n", err)
	} else if exists {
		err = c.Cache.Get(ctx, cacheKey, &squad)
		if err == nil {
			return &squad, nil
		}
		log.Printf("Cache get error for squad: %v\n", err)
	}

	headers := map[string]string{
		"Content-Type":   "application/json",
		"x-rapidapi-key": c.FootballAPIKey,
	}

	// Use configurable base URL with fallback
	baseURL := c.APIFootballBaseURL
	if baseURL == "" {
		baseURL = "https://api-football-v1.p.rapidapi.com/v3"
	}
	url := fmt.Sprintf("%s/players/squads?team=%d", baseURL, id)

	response, err := handleClientRequest[GetSquadResponse](url, "GET", headers)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %s", err)
	}

	// Log only if no squad data found
	if len(response.Response) == 0 {
		fmt.Printf("No squad data received for team ID: %d\n", id)
	}

	// Cache the squad data for 24 hours (team squads don't change frequently)
	err = c.Cache.Set(ctx, cacheKey, response, cache.TeamInfoTTL)
	if err != nil {
		log.Printf("Cache set error for squad: %v\n", err)
	}

	return response, nil
}

func (c *Config) getLeagues(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	season := fmt.Sprintf("%d", time.Now().Year())
	raw, _, err := c.futbolService().GetLeagues(ctx, season)
	if err != nil {
		respondWithError(w, futbolHTTPStatus(err), fmt.Sprintf("Error creating http request: %s", err))
		return
	}
	var data GetLeaguesResponse
	if err := decodeRawMap(raw, &data); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to read response from football api service: %s", err))
		return
	}

	type LeagueInfo struct {
		Name    string `json:"name"`
		Country string `json:"country"`
		Logo    string `json:"logo"`
	}
	leagueNames := []LeagueInfo{}
	for _, l := range data.Response {
		obj := LeagueInfo{
			Name:    l.League.Name,
			Country: l.Country.Name,
			Logo:    l.League.Logo,
		}
		leagueNames = append(leagueNames, obj)
	}
	respondWithJSON(w, http.StatusOK, leagueNames)
}

func (c *Config) getLeagueStandingsByTeamId(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	queryParams := r.URL.Query()
	teamId := queryParams.Get("team_id")

	if teamId == "" {
		respondWithError(w, http.StatusBadRequest, "team_id is required")
		return
	}

	currentYear := time.Now().Year()
	raw, _, err := c.futbolService().GetTeamStandings(ctx, teamId, currentYear)
	if err != nil {
		respondWithError(w, futbolHTTPStatus(err), fmt.Sprintf("Error creating http request: %s", err))
		return
	}
	var data GetLeagueStandingsByTeamIdResponse
	if err := decodeRawMap(raw, &data); err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Error parsing response body: %s", err))
		return
	}

	respondWithJSON(w, http.StatusOK, data)
}

// GetLeagueStandingsData fetches league standings from API-Football (with cache).
// Reused by the league_standings HTTP handler and by the debate context aggregator.
func (c *Config) GetLeagueStandingsData(ctx context.Context, leagueID, season string) (*GetLeagueStandingsResponse, error) {
	cacheKey := futbol.LeagueStandingsCacheKey(leagueID, season)
	if c.Cache != nil {
		var cached GetLeagueStandingsResponse
		exists, err := c.Cache.Exists(ctx, cacheKey)
		if err == nil && exists {
			if err := c.Cache.Get(ctx, cacheKey, &cached); err == nil {
				return &cached, nil
			}
		}
	}

	data, _, err := c.futbolService().GetLeagueStandings(ctx, leagueID, season)
	if err != nil {
		return nil, err
	}
	var out GetLeagueStandingsResponse
	if err := decodeRawMap(data.Raw, &out); err != nil {
		return nil, fmt.Errorf("standings parse: %w", err)
	}
	if c.Cache != nil && len(out.Response) > 0 {
		_ = c.Cache.Set(ctx, cacheKey, out, cache.StandingsTTL)
	}
	return &out, nil
}

func FormatLeagueStandingsSummary(data *GetLeagueStandingsResponse) string {
	if data == nil {
		return ""
	}
	raw := map[string]any{}
	b, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return ""
	}
	return futbol.FormatLeagueStandingsSummary(raw)
}

// FetchHeadToHead returns a text summary of the last head-to-head fixtures between two teams.
// Uses API-Football fixtures/headtohead, cache key h2h:{homeID}-{awayID}, TTL cache.H2HTTL.
// Reused by the debate context aggregator.
func (c *Config) FetchHeadToHead(ctx context.Context, homeTeamID, awayTeamID int) (string, error) {
	cacheKey := fmt.Sprintf("h2h:%d-%d", homeTeamID, awayTeamID)
	if c.Cache != nil {
		var cached string
		exists, err := c.Cache.Exists(ctx, cacheKey)
		if err == nil && exists {
			err = c.Cache.Get(ctx, cacheKey, &cached)
			if err == nil && cached != "" {
				return cached, nil
			}
		}
	}

	baseURL := c.APIFootballBaseURL
	if baseURL == "" {
		baseURL = "https://api-football-v1.p.rapidapi.com/v3"
	}
	u := fmt.Sprintf("%s/fixtures/headtohead?h2h=%d-%d&last=10", baseURL, homeTeamID, awayTeamID)
	headers := map[string]string{
		"Content-Type":   "application/json",
		"x-rapidapi-key": c.FootballAPIKey,
	}

	resp, err := HTTPRequest("GET", u, headers, nil)
	if err != nil {
		return "", fmt.Errorf("headtohead request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errorBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return "", fmt.Errorf("headtohead non-200 status %d and read body error: %w", resp.StatusCode, readErr)
		}
		return "", fmt.Errorf("headtohead non-200 status %d: %s", resp.StatusCode, strings.TrimSpace(string(errorBody)))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("headtohead read body: %w", err)
	}

	var apiResp struct {
		Response []struct {
			Fixture struct {
				Date string `json:"date"`
			} `json:"fixture"`
			Goals struct {
				Home int `json:"home"`
				Away int `json:"away"`
			} `json:"goals"`
			Teams struct {
				Home struct {
					Name string `json:"name"`
				} `json:"home"`
				Away struct {
					Name string `json:"name"`
				} `json:"away"`
			} `json:"teams"`
		} `json:"response"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("headtohead parse: %w", err)
	}

	var lines []string
	for _, m := range apiResp.Response {
		date := m.Fixture.Date
		if len(date) > 10 {
			date = date[:10]
		}
		lines = append(lines, fmt.Sprintf("%s: %s %d-%d %s",
			date, m.Teams.Home.Name, m.Goals.Home, m.Goals.Away, m.Teams.Away.Name))
	}
	summary := strings.Join(lines, "\n")
	if c.Cache != nil && summary != "" {
		_ = c.Cache.Set(ctx, cacheKey, summary, cache.H2HTTL)
	}
	return summary, nil
}

func (c *Config) getLeagueStandingsByLeagueId(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	queryParams := r.URL.Query()
	leagueID := queryParams.Get("league_id")
	season := queryParams.Get("season")

	if leagueID == "" {
		respondWithError(w, http.StatusBadRequest, "league_id is required")
		return
	}

	if season == "" {
		respondWithError(w, http.StatusBadRequest, "season is required")
		return
	}

	data, _, err := c.futbolService().GetLeagueStandings(ctx, leagueID, season)
	if err != nil {
		respondWithError(w, futbolHTTPStatus(err), err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, data.Raw)
}

type GetLeagueStandingsResponse struct {
	Get      string `json:"get"`
	Response []struct {
		League struct {
			ID        int    `json:"id"`
			Name      string `json:"name"`
			Country   string `json:"country"`
			Logo      string `json:"logo"`
			Flag      string `json:"flag"`
			Season    int    `json:"season"`
			Standings [][]struct {
				Rank int `json:"rank"`
				Team struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
					Logo string `json:"logo"`
				} `json:"team"`
				Points      int    `json:"points"`
				GoalsDiff   int    `json:"goalsDiff"`
				Group       string `json:"group"`
				Form        string `json:"form"`
				Status      string `json:"status"`
				Description string `json:"description"`
				All         struct {
					Played int `json:"played"`
					Win    int `json:"win"`
					Draw   int `json:"draw"`
					Lose   int `json:"lose"`
					Goals  struct {
						For     int `json:"for"`
						Against int `json:"against"`
					} `json:"goals"`
				} `json:"all"`
				Home struct {
					Played int `json:"played"`
					Win    int `json:"win"`
					Draw   int `json:"draw"`
					Lose   int `json:"lose"`
					Goals  struct {
						For     int `json:"for"`
						Against int `json:"against"`
					} `json:"goals"`
				} `json:"home"`
				Away struct {
					Played int `json:"played"`
					Win    int `json:"win"`
					Draw   int `json:"draw"`
					Lose   int `json:"lose"`
					Goals  struct {
						For     int `json:"for"`
						Against int `json:"against"`
					} `json:"goals"`
				} `json:"away"`
				Update string `json:"update"`
			} `json:"standings"`
		} `json:"league"`
	} `json:"response"`
	Errors  []string `json:"errors"`
	Results int      `json:"results"`
	Paging  struct {
		Current int `json:"current"`
		Total   int `json:"total"`
	} `json:"paging"`
}

func (c *Config) futbolService() *futbol.Service {
	if c.FutbolService == nil {
		provider := c.FutbolProvider
		if provider == nil {
			provider = futbol.NewAPIFootballClient(c.APIFootballBaseURL, c.FootballAPIKey)
		}
		c.FutbolService = futbol.NewService(
			provider,
			c.Cache,
		)
	}
	return c.FutbolService
}

func decodeRawMap(raw map[string]any, out interface{}) error {
	b, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func futbolHTTPStatus(err error) int {
	switch {
	case errors.Is(err, futbol.ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, futbol.ErrUpstream), errors.Is(err, futbol.ErrParse):
		// Preserve existing endpoint contract: futbol handlers return 400 on upstream/provider failures.
		return http.StatusBadRequest
	default:
		return http.StatusBadRequest
	}
}
