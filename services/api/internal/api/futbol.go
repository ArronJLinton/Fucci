package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/ArronJLinton/fucci-api/internal/cache"
)

type GetMatchesParams struct {
	Date string
}

func (c *Config) getMatches(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Step 1: Get date and optional league_id from query parameters
	queryParams := r.URL.Query()
	date := queryParams.Get("date")
	leagueID := queryParams.Get("league_id")
	log.Printf("League ID: %s\n", leagueID)
	if date == "" {
		log.Printf("ERROR: date parameter is missing")
		respondWithError(w, http.StatusBadRequest, "date parameter is required")
		return
	}

	// Generate cache key (include league_id if provided)
	var cacheKey string
	if leagueID != "" {
		cacheKey = fmt.Sprintf("matches:league:%s:date:%s", leagueID, date)
	} else {
		cacheKey = fmt.Sprintf("matches:date:%s", date)
	}
	log.Printf("Cache key: %s\n", cacheKey)

	// Try to get from cache first
	var data GetMatchesAPIResponse
	exists, err := c.Cache.Exists(ctx, cacheKey)
	if err != nil {
		log.Printf("Cache check error: %v\n", err)
	} else if exists {
		err = c.Cache.Get(ctx, cacheKey, &data)
		if err == nil {
			log.Printf("Cache HIT: Returning cached data\n")
			respondWithJSON(w, http.StatusOK, data)
			return
		}
		log.Printf("Cache get error: %v\n", err)
	} else {
		log.Printf("Cache MISS: Fetching from API\n")
	}

	footballAPIKey := c.FootballAPIKey
	if footballAPIKey == "" {
		log.Printf("ERROR: Football API key is missing")
		respondWithError(w, http.StatusBadRequest, "Football API key is required")
		return
	}
	// If not in cache or error occurred, fetch from API
	// Build URL with optional league filter
	url := fmt.Sprintf("https://api-football-v1.p.rapidapi.com/v3/fixtures?&date=%s", date)
	if leagueID != "" {
		// When filtering by league, RapidAPI requires a season parameter
		// Extract year from date to determine season
		// dateTime, err := time.Parse("2006-01-02", date)
		if err != nil {
			log.Printf("ERROR: Invalid date format: %s\n", date)
			respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid date format: %s. Expected YYYY-MM-DD", date))
			return
		}
		// FIXME: Need to dynamically set the season year
		// season := dateTime.Year()
		season := 2025
		url = fmt.Sprintf("%s&league=%s&season=%d", url, leagueID, season)
		log.Printf("URL: %s\n", url)
		log.Printf("Filtering matches by league_id: %s, season: %d\n", leagueID, season)
	}
	headers := map[string]string{
		"Content-Type":   "application/json",
		"x-rapidapi-key": c.FootballAPIKey,
	}

	resp, err := HTTPRequest("GET", url, headers, nil)
	if err != nil {
		log.Printf("ERROR: HTTPRequest failed: %v\n", err)
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Error creating http request: %s", err))
		return
	}
	defer resp.Body.Close()

	// Read the raw response body for debugging
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read response body: %v\n", err)
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read response body: %s", err))
		return
	}

	// Create a new reader from the raw body for JSON decoding
	err = json.Unmarshal(rawBody, &data)
	if err != nil {
		log.Printf("ERROR: JSON unmarshal failed: %v\n", err)
		log.Printf("Full raw response: %s\n", string(rawBody))
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse response from football api service: %s", err))
		return
	}

	// Check if response is empty or invalid before caching
	if data.Results == 0 || len(data.Response) == 0 {
		log.Printf("WARNING: API returned empty response (results=%d, response items=%d), skipping cache\n", data.Results, len(data.Response))
		log.Printf("Response data: get=%s, errors=%v\n", data.Get, data.Errors)
		respondWithJSON(w, http.StatusOK, data)
		return
	}

	// Determine cache TTL based on match statuses
	ttl := cache.DefaultTTL
	if len(data.Response) > 0 {
		// Check if any matches are live
		for _, match := range data.Response {
			status := match.Fixture.Status.Short
			matchTTL := cache.GetMatchTTL(status)
			// Use the shortest TTL found (most conservative)
			if matchTTL < ttl {
				ttl = matchTTL
			}
		}
	}

	// Store in cache with determined TTL (only if we have valid data)
	err = c.Cache.Set(ctx, cacheKey, data, ttl)
	if err != nil {
		log.Printf("Cache set error: %v\n", err)
	} else {
		log.Printf("Cached data with TTL: %v (results=%d)\n", ttl, data.Results)
	}

	respondWithJSON(w, http.StatusOK, data)
}

func (c *Config) getMatch(w http.ResponseWriter, r *http.Request) {
}

func (c *Config) getMatchLineup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	queryParams := r.URL.Query()
	matchID := queryParams.Get("match_id")
	if matchID == "" {
		respondWithError(w, http.StatusBadRequest, "match_id is required")
		return
	}

	// Generate cache key
	cacheKey := fmt.Sprintf("lineup:%s", matchID)

	// Try to get from cache first
	var response struct {
		Home Lineup `json:"home"`
		Away Lineup `json:"away"`
	}
	exists, err := c.Cache.Exists(ctx, cacheKey)
	if err != nil {
		fmt.Printf("Cache check error: %v\n", err)
	} else if exists {
		err = c.Cache.Get(ctx, cacheKey, &response)
		if err == nil {
			fmt.Printf("Cache hit for lineup: %s\n", matchID)
			respondWithJSON(w, http.StatusOK, response)
			return
		}
		fmt.Printf("Cache get error: %v\n", err)
	}

	// Use configurable base URL with fallback
	baseURL := c.APIFootballBaseURL
	if baseURL == "" {
		baseURL = "https://api-football-v1.p.rapidapi.com/v3"
	}
	url := fmt.Sprintf("%s/fixtures/lineups?fixture=%s", baseURL, matchID)
	headers := map[string]string{
		"Content-Type":   "application/json",
		"x-rapidapi-key": c.FootballAPIKey,
	}
	resp, err := HTTPRequest("GET", url, headers, nil)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Error creating http request: %s", err))
		return
	}
	defer resp.Body.Close()

	// Read the raw response for debugging
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read response body: %s", err))
		return
	}

	// Create a new reader from the raw body for JSON decoding
	getLineUpData := &GetLineUpResponse{}
	err = json.Unmarshal(rawBody, &getLineUpData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse response from football api service - %s", err))
		return
	}

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
	// Convert to lowercase
	name = strings.ToLower(name)

	// Remove dots and extra spaces
	name = strings.ReplaceAll(name, ".", "")
	name = strings.Join(strings.Fields(name), " ")

	// Remove special characters
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, name)

	return name
}

func (c *Config) getTeamSquad(id int32, ctx context.Context) (*GetSquadResponse, error) {
	// Generate cache key
	cacheKey := fmt.Sprintf("team_squad:%d", id)

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

	// Derive current year dynamically
	currentYear := time.Now().Year()
	cacheKey := fmt.Sprintf("leagues:%d", currentYear)

	// Try to get from cache first
	var data GetLeaguesResponse
	exists, err := c.Cache.Exists(ctx, cacheKey)
	if err != nil {
		log.Printf("Cache check error: %v\n", err)
	} else if exists {
		err = c.Cache.Get(ctx, cacheKey, &data)
		if err == nil {
			// Process the cached data
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
			return
		}
		log.Printf("Cache get error: %v\n", err)
	}

	url := "https://api-football-v1.p.rapidapi.com/v3/leagues?season=2025"
	headers := map[string]string{
		"Content-Type":   "application/json",
		"x-rapidapi-key": c.FootballAPIKey,
	}
	resp, err := HTTPRequest("GET", url, headers, nil)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Error creating http request: %s", err))
		return
	}
	defer resp.Body.Close()

	responseBody := json.NewDecoder(resp.Body)
	err = responseBody.Decode(&data)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to read response from football api service: %s", err))
		return
	}

	// Store in cache for 24 hours (league data rarely changes)
	err = c.Cache.Set(ctx, cacheKey, data, cache.TeamInfoTTL)
	if err != nil {
		log.Printf("Cache set error: %v\n", err)
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

	// TODO: Dynamically set the season year
	currentYear := time.Now().Year()

	// Generate cache key
	cacheKey := fmt.Sprintf("team_standings:%s:%d", teamId, currentYear)

	// Try to get from cache first
	var data GetLeagueStandingsByTeamIdResponse
	exists, err := c.Cache.Exists(ctx, cacheKey)
	if err != nil {
		log.Printf("Cache check error: %v\n", err)
	} else if exists {
		err = c.Cache.Get(ctx, cacheKey, &data)
		if err == nil {
			respondWithJSON(w, http.StatusOK, data)
			return
		}
		log.Printf("Cache get error: %v\n", err)
	}

	url := fmt.Sprintf("https://api-football-v1.p.rapidapi.com/v3/standings?season=%d&team=%s", currentYear, teamId)
	headers := map[string]string{
		"Content-Type":   "application/json",
		"x-rapidapi-key": c.FootballAPIKey,
	}
	resp, err := HTTPRequest("GET", url, headers, nil)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Error creating http request: %s", err))
		return
	}
	defer resp.Body.Close()

	responseBody := json.NewDecoder(resp.Body)
	err = responseBody.Decode(&data)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Error parsing response body: %s", err))
		return
	}

	// Store in cache for 6 hours (standings update periodically)
	err = c.Cache.Set(ctx, cacheKey, data, cache.StandingsTTL)
	if err != nil {
		log.Printf("Cache set error: %v\n", err)
	}

	respondWithJSON(w, http.StatusOK, data)
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

	// Generate cache key
	cacheKey := fmt.Sprintf("league_standings:%s:%s", leagueID, season)

	// Try to get from cache first
	var data GetLeagueStandingsResponse
	exists, err := c.Cache.Exists(ctx, cacheKey)
	if err != nil {
		log.Printf("Cache check error: %v\n", err)
	} else if exists {
		err = c.Cache.Get(ctx, cacheKey, &data)
		if err == nil {
			respondWithJSON(w, http.StatusOK, data)
			return
		}
		log.Printf("Cache get error: %v\n", err)
	}

	// Use configurable base URL with fallback
	baseURL := c.APIFootballBaseURL
	if baseURL == "" {
		baseURL = "https://api-football-v1.p.rapidapi.com/v3"
	}
	url := fmt.Sprintf("%s/standings?league=%s&season=%s", baseURL, leagueID, season)
	headers := map[string]string{
		"Content-Type":   "application/json",
		"x-rapidapi-key": c.FootballAPIKey,
	}

	resp, err := HTTPRequest("GET", url, headers, nil)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Error creating http request: %s", err))
		return
	}
	defer resp.Body.Close()

	// Read the raw response body for debugging
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to read response body: %s", err))
		return
	}

	// Create a new reader from the raw body for JSON decoding
	err = json.Unmarshal(rawBody, &data)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse response from football api service: %s", err))
		return
	}

	// Store in cache
	err = c.Cache.Set(ctx, cacheKey, data, cache.DefaultTTL)
	if err != nil {
		log.Printf("Cache set error: %v\n", err)
	}

	respondWithJSON(w, http.StatusOK, data)
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
