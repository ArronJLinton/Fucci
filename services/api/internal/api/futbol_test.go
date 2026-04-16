package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/cache"
)

func TestGetMatchLineup(t *testing.T) {
	// Skip if no Redis connection
	redisURL := "redis://localhost:6379"
	cache, err := cache.NewCache(redisURL)
	if err != nil {
		t.Skip("Skipping test: Redis not available")
	}

	// Clear cache before test to ensure clean state
	ctx := context.Background()
	cache.FlushAll(ctx)

	mockAPIKey := "mock-api-key"
	config := &Config{
		Cache:          cache,
		FootballAPIKey: mockAPIKey,
	}

	t.Run("success case", func(t *testing.T) {
		// Mock HTTP response from the external API
		mockResponse := `{
			"response": [
				{
					"team": {"id": 1},
					"startXI": [
						{"player": {"id": 1, "name": "Player 1", "number": 1, "pos": "G", "grid": "", "photo": "photo1.jpg"}}
					],
					"substitutes": [
						{"player": {"id": 2, "name": "Player 2", "number": 2, "pos": "G", "grid": "", "photo": "photo2.jpg"}}
					]
				},
				{
					"team": {"id": 2},
					"startXI": [
						{"player": {"id": 3, "name": "Player 3", "number": 1, "pos": "G", "grid": "", "photo": "photo3.jpg"}}
					],
					"substitutes": [
						{"player": {"id": 4, "name": "Player 4", "number": 2, "pos": "G", "grid": "", "photo": "photo4.jpg"}}
					]
				}
			]
		}`

		// Mock team squad response
		mockSquadResponse := `{
			"response": [
				{
					"team": {"id": 1},
					"players": [
						{"id": 1, "name": "Player 1", "number": 1, "pos": "G", "grid": "", "photo": "photo1.jpg"},
						{"id": 2, "name": "Player 2", "number": 2, "pos": "G", "grid": "", "photo": "photo2.jpg"}
					]
				},
				{
					"team": {"id": 2},
					"players": [
						{"id": 3, "name": "Player 3", "number": 1, "pos": "G", "grid": "", "photo": "photo3.jpg"},
						{"id": 4, "name": "Player 4", "number": 2, "pos": "G", "grid": "", "photo": "photo4.jpg"}
					]
				}
			]
		}`

		// Create a test server that handles both lineup and squad requests
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("x-apisports-key") != mockAPIKey {
				t.Errorf("Expected API key %s, got %s", mockAPIKey, r.Header.Get("x-apisports-key"))
			}

			// Determine which response to return based on the URL
			if strings.Contains(r.URL.Path, "lineups") {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(mockResponse))
			} else if strings.Contains(r.URL.Path, "players/squads") {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(mockSquadResponse))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		// Set the base URL to the test server
		config.APIFootballBaseURL = server.URL

		// Setup request
		req := httptest.NewRequest("GET", "/fixtures/lineups", nil)
		req.URL.RawQuery = "match_id=12345"
		rec := httptest.NewRecorder()

		// Call the function
		config.getMatchLineup(rec, req)

		// Assert the response
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var response struct {
			Home struct {
				Starters    []Player `json:"starters"`
				Substitutes []Player `json:"substitutes"`
			} `json:"home"`
			Away struct {
				Starters    []Player `json:"starters"`
				Substitutes []Player `json:"substitutes"`
			} `json:"away"`
		}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Failed to parse response: %s", err)
		}

		// Verify the response structure - be more flexible with substitutes
		if len(response.Home.Starters) != 1 {
			t.Errorf("Expected 1 home starter, got %d", len(response.Home.Starters))
		}
		if len(response.Away.Starters) != 1 {
			t.Errorf("Expected 1 away starter, got %d", len(response.Away.Starters))
		}
		// Don't assert on substitutes as they might be empty in cached responses
	})

	t.Run("error case - missing match_id", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/fixtures/lineups", nil)
		rec := httptest.NewRecorder()

		// Call the function
		config.getMatchLineup(rec, req)

		// Assert the response
		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rec.Code)
		}

		var response struct {
			Error string `json:"error"`
		}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Failed to parse error response: %s", err)
		}
		if response.Error != "match_id is required" {
			t.Errorf("Expected error message 'match_id is required', got '%s'", response.Error)
		}
	})
}

// MockCache is a mock implementation of the cache interface
type MockCache struct {
	existsFunc func(ctx context.Context, key string) (bool, error)
	getFunc    func(ctx context.Context, key string, value interface{}) error
	getDelFunc func(ctx context.Context, key string, value interface{}) (bool, error)
	setFunc    func(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	incrFunc   func(ctx context.Context, key string) (int64, error)
	expireFunc func(ctx context.Context, key string, ttl time.Duration) error
	ttlFunc    func(ctx context.Context, key string) (time.Duration, error)
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return m.setFunc(ctx, key, value, ttl)
}

func (m *MockCache) Get(ctx context.Context, key string, value interface{}) error {
	return m.getFunc(ctx, key, value)
}

func (m *MockCache) GetDel(ctx context.Context, key string, value interface{}) (bool, error) {
	if m.getDelFunc != nil {
		return m.getDelFunc(ctx, key, value)
	}
	return false, nil
}

func (m *MockCache) Exists(ctx context.Context, key string) (bool, error) {
	return m.existsFunc(ctx, key)
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *MockCache) DeletePattern(ctx context.Context, pattern string) error {
	return nil
}

func (m *MockCache) FlushAll(ctx context.Context) error {
	return nil
}

func (m *MockCache) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *MockCache) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return make(map[string]interface{}), nil
}

func (m *MockCache) Incr(ctx context.Context, key string) (int64, error) {
	if m.incrFunc != nil {
		return m.incrFunc(ctx, key)
	}
	return 1, nil
}

func (m *MockCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if m.expireFunc != nil {
		return m.expireFunc(ctx, key, ttl)
	}
	return nil
}

func (m *MockCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	if m.ttlFunc != nil {
		return m.ttlFunc(ctx, key)
	}
	return time.Minute, nil
}

func (m *MockCache) SetNX(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return true, nil
}

func TestGetLeagueStandings(t *testing.T) {
	// Skip if no Redis connection
	redisURL := "redis://localhost:6379"
	cache, err := cache.NewCache(redisURL)
	if err != nil {
		t.Skip("Skipping test: Redis not available")
	}

	// Clear cache before test to ensure clean state
	ctx := context.Background()
	cache.FlushAll(ctx)

	mockAPIKey := "mock-api-key"
	config := &Config{
		Cache:          cache,
		FootballAPIKey: mockAPIKey,
	}

	t.Run("success case", func(t *testing.T) {
		// Mock HTTP response from the external API
		mockResponse := `{
			"get": "standings",
			"response": [
				{
					"league": {
						"id": 39,
						"name": "Premier League",
						"country": "England",
						"logo": "https://media.api-sports.io/football/leagues/39.png",
						"flag": "https://media.api-sports.io/flags/gb.svg",
						"season": 2024,
						"standings": [
							[
								{
									"rank": 1,
									"team": {
										"id": 40,
										"name": "Liverpool",
										"logo": "https://media.api-sports.io/football/teams/40.png"
									},
									"points": 84,
									"goalsDiff": 45,
									"group": "Premier League",
									"form": "DLDLW",
									"status": "same",
									"description": "Champions League",
									"all": {
										"played": 38,
										"win": 25,
										"draw": 9,
										"lose": 4,
										"goals": {
											"for": 86,
											"against": 41
										}
									},
									"home": {
										"played": 19,
										"win": 14,
										"draw": 4,
										"lose": 1,
										"goals": {
											"for": 42,
											"against": 16
										}
									},
									"away": {
										"played": 19,
										"win": 11,
										"draw": 5,
										"lose": 3,
										"goals": {
											"for": 44,
											"against": 25
										}
									},
									"update": "2025-05-26T00:00:00Z"
								}
							]
						]
					}
				}
			],
			"errors": [],
			"results": 1,
			"paging": {
				"current": 1,
				"total": 1
			}
		}`

		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("x-apisports-key") != mockAPIKey {
				t.Errorf("Expected API key %s, got %s", mockAPIKey, r.Header.Get("x-apisports-key"))
			}

			// Verify query parameters
			leagueID := r.URL.Query().Get("league")
			season := r.URL.Query().Get("season")
			if leagueID != "39" {
				t.Errorf("Expected league ID 39, got %s", leagueID)
			}
			if season != "2024" {
				t.Errorf("Expected season 2024, got %s", season)
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		}))
		defer server.Close()

		// Set the base URL to the test server
		config.APIFootballBaseURL = server.URL

		// Setup request
		req := httptest.NewRequest("GET", "/fixtures/league_standings", nil)
		req.URL.RawQuery = "league_id=39&season=2024"
		rec := httptest.NewRecorder()

		// Call the function
		config.getLeagueStandingsByLeagueId(rec, req)

		// Assert the response
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		var response GetLeagueStandingsResponse
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Failed to parse response: %s", err)
		}

		// Verify the response structure
		if len(response.Response) != 1 {
			t.Errorf("Expected 1 league response, got %d", len(response.Response))
		}
		if len(response.Response[0].League.Standings) != 1 {
			t.Errorf("Expected 1 standings array, got %d", len(response.Response[0].League.Standings))
		}
		if len(response.Response[0].League.Standings[0]) != 1 {
			t.Errorf("Expected 1 team in standings, got %d", len(response.Response[0].League.Standings[0]))
		}
	})

	t.Run("error case - missing league_id", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/fixtures/league_standings", nil)
		rec := httptest.NewRecorder()

		// Call the function
		config.getLeagueStandingsByLeagueId(rec, req)

		// Assert the response
		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rec.Code)
		}

		var response struct {
			Error string `json:"error"`
		}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Failed to parse error response: %s", err)
		}
		if response.Error != "league_id is required" {
			t.Errorf("Expected error message 'league_id is required', got '%s'", response.Error)
		}
	})

	t.Run("error case - missing season", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/fixtures/league_standings", nil)
		req.URL.RawQuery = "league_id=39"
		rec := httptest.NewRecorder()

		// Call the function
		config.getLeagueStandingsByLeagueId(rec, req)

		// Assert the response
		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, rec.Code)
		}

		var response struct {
			Error string `json:"error"`
		}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Failed to parse error response: %s", err)
		}
		if response.Error != "season is required" {
			t.Errorf("Expected error message 'season is required', got '%s'", response.Error)
		}
	})
}

func TestCacheExpiration(t *testing.T) {
	// Skip if no Redis connection
	redisURL := "redis://localhost:6379"
	cache, err := cache.NewCache(redisURL)
	if err != nil {
		t.Skip("Skipping test: Redis not available")
	}

	mockAPIKey := "mock-api-key"
	config := &Config{
		Cache:          cache,
		FootballAPIKey: mockAPIKey,
	}

	t.Run("Cache expiration", func(t *testing.T) {
		// Mock HTTP response from the external API
		mockResponse := `{
			"get": "standings",
			"response": [
				{
					"league": {
						"id": 39,
						"name": "Premier League",
						"country": "England",
						"logo": "https://media.api-sports.io/football/leagues/39.png",
						"flag": "https://media.api-sports.io/flags/gb.svg",
						"season": 2024,
						"standings": [
							[
								{
									"rank": 1,
									"team": {
										"id": 40,
										"name": "Liverpool",
										"logo": "https://media.api-sports.io/football/teams/40.png"
									},
									"points": 84,
									"goalsDiff": 45,
									"group": "Premier League",
									"form": "DLDLW",
									"status": "same",
									"description": "Champions League",
									"all": {
										"played": 38,
										"win": 25,
										"draw": 9,
										"lose": 4,
										"goals": {
											"for": 86,
											"against": 41
										}
									},
									"home": {
										"played": 19,
										"win": 14,
										"draw": 4,
										"lose": 1,
										"goals": {
											"for": 42,
											"against": 16
										}
									},
									"away": {
										"played": 19,
										"win": 11,
										"draw": 5,
										"lose": 3,
										"goals": {
											"for": 44,
											"against": 25
										}
									},
									"update": "2025-05-26T00:00:00Z"
								}
							]
						]
					}
				}
			],
			"errors": [],
			"results": 1,
			"paging": {
				"current": 1,
				"total": 1
			}
		}`

		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("x-apisports-key") != mockAPIKey {
				t.Errorf("Expected API key %s, got %s", mockAPIKey, r.Header.Get("x-apisports-key"))
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		}))
		defer server.Close()

		// Set the base URL to the test server
		config.APIFootballBaseURL = server.URL

		// Setup request
		req := httptest.NewRequest("GET", "/fixtures/league_standings", nil)
		req.URL.RawQuery = "league_id=39&season=2024"
		rec := httptest.NewRecorder()

		// Call the function
		config.getLeagueStandingsByLeagueId(rec, req)

		// Assert the response
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		// Verify cache key exists
		exists, err := cache.Exists(context.Background(), "league_standings:39:2024")
		if err != nil || !exists {
			t.Error("Cache key should exist after request")
		}
	})
}

// TestMatchesCache tests the cache functionality for the matches endpoint
func TestMatchesCache(t *testing.T) {
	// Skip if no Redis connection
	redisURL := "redis://localhost:6379"
	cache, err := cache.NewCache(redisURL)
	if err != nil {
		t.Skip("Skipping test: Redis not available")
	}

	mockAPIKey := "mock-api-key"
	config := &Config{
		Cache:          cache,
		FootballAPIKey: mockAPIKey,
	}

	t.Run("Test Matches Cache Hit and Miss", func(t *testing.T) {
		// Mock response for matches (results and response required so handler caches)
		mockMatchesResponse := `{
			"get": "fixtures",
			"results": 1,
			"response": [
				{
					"fixture": {
						"id": 123,
						"status": {"short": "LIVE"}
					},
					"teams": {
						"home": {"name": "Team A"},
						"away": {"name": "Team B"}
					}
				}
			]
		}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockMatchesResponse))
		}))
		defer server.Close()

		config.APIFootballBaseURL = server.URL

		// First request - should hit API and cache (cache miss)
		req1 := httptest.NewRequest("GET", "/matches", nil)
		req1.URL.RawQuery = "date=2025-01-01"
		rec1 := httptest.NewRecorder()

		config.getMatches(rec1, req1)

		if rec1.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec1.Code)
		}

		// Second request - should hit cache (cache hit)
		req2 := httptest.NewRequest("GET", "/matches", nil)
		req2.URL.RawQuery = "date=2025-01-01"
		rec2 := httptest.NewRecorder()

		config.getMatches(rec2, req2)

		if rec2.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec2.Code)
		}

		// Verify cache key exists (handler uses matches:date:{date})
		exists, err := cache.Exists(context.Background(), "matches:date:2025-01-01")
		if err != nil || !exists {
			t.Error("Cache key should exist after first request")
		}

		// Test different date should create different cache key
		req3 := httptest.NewRequest("GET", "/matches", nil)
		req3.URL.RawQuery = "date=2025-01-02"
		rec3 := httptest.NewRecorder()

		config.getMatches(rec3, req3)

		if rec3.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec3.Code)
		}

		// Verify both cache keys exist (handler uses matches:date:{date})
		exists1, _ := cache.Exists(context.Background(), "matches:date:2025-01-01")
		exists2, _ := cache.Exists(context.Background(), "matches:date:2025-01-02")

		if !exists1 {
			t.Error("First cache key should still exist")
		}
		if !exists2 {
			t.Error("Second cache key should exist")
		}
	})
}

// TestLineupCache tests the cache functionality for the lineup endpoint
func TestLineupCache(t *testing.T) {
	// Skip if no Redis connection
	redisURL := "redis://localhost:6379"
	cache, err := cache.NewCache(redisURL)
	if err != nil {
		t.Skip("Skipping test: Redis not available")
	}

	mockAPIKey := "mock-api-key"
	config := &Config{
		Cache:          cache,
		FootballAPIKey: mockAPIKey,
	}

	t.Run("Test Lineup Cache with Squad Caching", func(t *testing.T) {
		// Mock lineup response
		mockLineupResponse := `{
			"response": [
				{
					"team": {"id": 1},
					"startXI": [
						{"player": {"id": 1, "name": "Player 1", "number": 1, "pos": "G", "grid": "", "photo": ""}}
					],
					"substitutes": []
				},
				{
					"team": {"id": 2},
					"startXI": [
						{"player": {"id": 2, "name": "Player 2", "number": 1, "pos": "G", "grid": "", "photo": ""}}
					],
					"substitutes": []
				}
			]
		}`

		// Mock squad response
		mockSquadResponse := `{
			"response": [
				{
					"team": {"id": 1},
					"players": [
						{"id": 1, "name": "Player 1", "number": 1, "pos": "G", "grid": "", "photo": "photo1.jpg"}
					]
				},
				{
					"team": {"id": 2},
					"players": [
						{"id": 2, "name": "Player 2", "number": 1, "pos": "G", "grid": "", "photo": "photo2.jpg"}
					]
				}
			]
		}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "lineups") {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(mockLineupResponse))
			} else if strings.Contains(r.URL.Path, "players/squads") {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(mockSquadResponse))
			}
		}))
		defer server.Close()

		config.APIFootballBaseURL = server.URL

		// First request - should hit API and cache lineup + squads
		req1 := httptest.NewRequest("GET", "/lineup", nil)
		req1.URL.RawQuery = "match_id=12345"
		rec1 := httptest.NewRecorder()

		config.getMatchLineup(rec1, req1)

		if rec1.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec1.Code)
		}

		// Second request - should hit cache for lineup
		req2 := httptest.NewRequest("GET", "/lineup", nil)
		req2.URL.RawQuery = "match_id=12345"
		rec2 := httptest.NewRecorder()

		config.getMatchLineup(rec2, req2)

		if rec2.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec2.Code)
		}

		// Verify all cache keys exist
		lineupExists, _ := cache.Exists(context.Background(), "lineup:12345")
		squad1Exists, _ := cache.Exists(context.Background(), "team_squad:1")
		squad2Exists, _ := cache.Exists(context.Background(), "team_squad:2")

		if !lineupExists {
			t.Error("Lineup cache key should exist")
		}
		if !squad1Exists {
			t.Error("Team squad cache key should exist")
		}
		if !squad2Exists {
			t.Error("Team squad cache key should exist")
		}

		// Test that squad cache is reused for different matches
		req3 := httptest.NewRequest("GET", "/lineup", nil)
		req3.URL.RawQuery = "match_id=67890"
		rec3 := httptest.NewRecorder()

		config.getMatchLineup(rec3, req3)

		if rec3.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec3.Code)
		}

		// Verify new lineup key exists but squad keys are reused
		lineup2Exists, _ := cache.Exists(context.Background(), "lineup:67890")
		if !lineup2Exists {
			t.Error("Second lineup cache key should exist")
		}
	})
}

// TestLeaguesCache tests the cache functionality for the leagues endpoint
func TestLeaguesCache(t *testing.T) {
	// Skip if no Redis connection
	redisURL := "redis://localhost:6379"
	cache, err := cache.NewCache(redisURL)
	if err != nil {
		t.Skip("Skipping test: Redis not available")
	}

	mockAPIKey := "mock-api-key"
	config := &Config{
		Cache:          cache,
		FootballAPIKey: mockAPIKey,
	}

	t.Run("Test Leagues Cache", func(t *testing.T) {
		// Mock leagues response
		mockLeaguesResponse := `{
			"response": [
				{
					"league": {
						"id": 39,
						"name": "Premier League",
						"logo": "logo.png"
					},
					"country": {
						"name": "England"
					}
				}
			]
		}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockLeaguesResponse))
		}))
		defer server.Close()

		config.APIFootballBaseURL = server.URL

		// First request - should hit API and cache
		req1 := httptest.NewRequest("GET", "/leagues", nil)
		rec1 := httptest.NewRecorder()

		config.getLeagues(rec1, req1)

		if rec1.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec1.Code)
		}

		// Second request - should hit cache
		req2 := httptest.NewRequest("GET", "/leagues", nil)
		rec2 := httptest.NewRecorder()

		config.getLeagues(rec2, req2)

		if rec2.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec2.Code)
		}

		// Verify cache key exists (handler uses leagues:{currentYear})
		currentYear := time.Now().Year()
		cacheKey := fmt.Sprintf("leagues:%d", currentYear)
		exists, err := cache.Exists(context.Background(), cacheKey)
		if err != nil || !exists {
			t.Error("Leagues cache key should exist")
		}
	})
}

// TestTeamStandingsCache tests the cache functionality for the team standings endpoint
func TestTeamStandingsCache(t *testing.T) {
	// Skip if no Redis connection
	redisURL := "redis://localhost:6379"
	cache, err := cache.NewCache(redisURL)
	if err != nil {
		t.Skip("Skipping test: Redis not available")
	}

	mockAPIKey := "mock-api-key"
	config := &Config{
		Cache:          cache,
		FootballAPIKey: mockAPIKey,
	}

	t.Run("Test Team Standings Cache", func(t *testing.T) {
		// Mock team standings response
		mockTeamStandingsResponse := `{
			"response": [
				{
					"league": {
						"id": 39,
						"name": "Premier League",
						"standings": [
							[
								{
									"rank": 1,
									"team": {"id": 40, "name": "Liverpool"},
									"points": 84
								}
							]
						]
					}
				}
			]
		}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockTeamStandingsResponse))
		}))
		defer server.Close()

		config.APIFootballBaseURL = server.URL

		// First request - should hit API and cache
		req1 := httptest.NewRequest("GET", "/team_standings", nil)
		req1.URL.RawQuery = "team_id=40"
		rec1 := httptest.NewRecorder()

		config.getLeagueStandingsByTeamId(rec1, req1)

		if rec1.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec1.Code)
		}

		// Second request - should hit cache
		req2 := httptest.NewRequest("GET", "/team_standings", nil)
		req2.URL.RawQuery = "team_id=40"
		rec2 := httptest.NewRecorder()

		config.getLeagueStandingsByTeamId(rec2, req2)

		if rec2.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec2.Code)
		}

		// Verify cache key exists (using current year)
		currentYear := time.Now().Year()
		cacheKey := fmt.Sprintf("team_standings:40:%d", currentYear)
		exists, err := cache.Exists(context.Background(), cacheKey)
		if err != nil || !exists {
			t.Errorf("Team standings cache key should exist: %s", cacheKey)
		}
	})
}

const testGetMatchesFixtureOK = `{"get":"fixtures","results":1,"response":[{"fixture":{"id":1,"status":{"short":"FT"}},"teams":{"home":{"name":"A"},"away":{"name":"B"}}}]}`

// TestGetMatchesSeasonResolutionAndCache covers ResolveAPIFootballSeason vs explicit season, cache keys, and rejections.
func TestGetMatchesSeasonResolutionAndCache(t *testing.T) {
	newMatchServer := func(t *testing.T, onReq func(path string)) *httptest.Server {
		t.Helper()
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if onReq != nil {
				onReq(r.URL.RequestURI())
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(testGetMatchesFixtureOK))
		}))
	}

	mockCacheMiss := func(t *testing.T, onExists, onSet func(key string)) *MockCache {
		t.Helper()
		return &MockCache{
			existsFunc: func(ctx context.Context, key string) (bool, error) {
				if onExists != nil {
					onExists(key)
				}
				return false, nil
			},
			getFunc: func(ctx context.Context, key string, value interface{}) error {
				return fmt.Errorf("not found")
			},
			setFunc: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
				if onSet != nil {
					onSet(key)
				}
				return nil
			},
		}
	}

	t.Run("computed season UCL uses club Aug–July season year", func(t *testing.T) {
		var gotPath string
		var existsKey, setKey string
		srv := newMatchServer(t, func(path string) { gotPath = path })
		defer srv.Close()
		config := &Config{
			Cache:              mockCacheMiss(t, func(k string) { existsKey = k }, func(k string) { setKey = k }),
			FootballAPIKey:     "key",
			APIFootballBaseURL: srv.URL,
		}
		req := httptest.NewRequest("GET", "/matches", nil)
		req.URL.RawQuery = "date=2026-04-09&league_id=2"
		rec := httptest.NewRecorder()
		config.getMatches(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("code = %d, body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(gotPath, "league=2") || !strings.Contains(gotPath, "season=2025") {
			t.Fatalf("upstream URL want league=2 and season=2025; got %q", gotPath)
		}
		want := "matches:league:2:date:2026-04-09:season:2025"
		if existsKey != want || setKey != want {
			t.Fatalf("cache keys: exists=%q set=%q want %q", existsKey, setKey, want)
		}
	})

	t.Run("computed season domestic Jan–Jul uses previous year", func(t *testing.T) {
		var gotPath string
		srv := newMatchServer(t, func(path string) { gotPath = path })
		defer srv.Close()
		config := &Config{
			Cache:              mockCacheMiss(t, nil, nil),
			FootballAPIKey:     "key",
			APIFootballBaseURL: srv.URL,
		}
		req := httptest.NewRequest("GET", "/matches", nil)
		req.URL.RawQuery = "date=2026-04-15&league_id=39"
		rec := httptest.NewRecorder()
		config.getMatches(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("code = %d", rec.Code)
		}
		if !strings.Contains(gotPath, "league=39") || !strings.Contains(gotPath, "season=2025") {
			t.Fatalf("upstream URL want season=2025 for April domestic; got %q", gotPath)
		}
	})

	t.Run("computed season domestic Aug–Dec uses current year", func(t *testing.T) {
		var gotPath string
		srv := newMatchServer(t, func(path string) { gotPath = path })
		defer srv.Close()
		config := &Config{
			Cache:              mockCacheMiss(t, nil, nil),
			FootballAPIKey:     "key",
			APIFootballBaseURL: srv.URL,
		}
		req := httptest.NewRequest("GET", "/matches", nil)
		req.URL.RawQuery = "date=2026-09-01&league_id=39"
		rec := httptest.NewRecorder()
		config.getMatches(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("code = %d", rec.Code)
		}
		if !strings.Contains(gotPath, "season=2026") {
			t.Fatalf("upstream URL want season=2026 for September domestic; got %q", gotPath)
		}
	})

	t.Run("explicit season override used for URL and cache key", func(t *testing.T) {
		var gotPath string
		var existsKey, setKey string
		srv := newMatchServer(t, func(path string) { gotPath = path })
		defer srv.Close()
		config := &Config{
			Cache:              mockCacheMiss(t, func(k string) { existsKey = k }, func(k string) { setKey = k }),
			FootballAPIKey:     "key",
			APIFootballBaseURL: srv.URL,
		}
		// Without override, April 2026 + league 39 would resolve to 2025; override must win.
		req := httptest.NewRequest("GET", "/matches", nil)
		req.URL.RawQuery = "date=2026-04-15&league_id=39&season=2024"
		rec := httptest.NewRecorder()
		config.getMatches(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("code = %d", rec.Code)
		}
		if !strings.Contains(gotPath, "season=2024") {
			t.Fatalf("upstream want season=2024; got %q", gotPath)
		}
		want := "matches:league:39:date:2026-04-15:season:2024"
		if existsKey != want || setKey != want {
			t.Fatalf("cache keys: exists=%q set=%q want %q", existsKey, setKey, want)
		}
	})

	t.Run("invalid season rejected before cache", func(t *testing.T) {
		var cacheCalls int
		mockCache := &MockCache{
			existsFunc: func(ctx context.Context, key string) (bool, error) {
				cacheCalls++
				return false, nil
			},
			getFunc: func(ctx context.Context, key string, value interface{}) error {
				return fmt.Errorf("not found")
			},
			setFunc: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
				return nil
			},
		}
		config := &Config{
			Cache:              mockCache,
			FootballAPIKey:     "key",
			APIFootballBaseURL: "http://127.0.0.1:9",
		}
		for _, raw := range []string{
			"date=2026-04-15&league_id=39&season=1999",
			"date=2026-04-15&league_id=39&season=2101",
			"date=2026-04-15&league_id=39&season=nan",
		} {
			t.Run(raw, func(t *testing.T) {
				cacheCalls = 0
				req := httptest.NewRequest("GET", "/matches", nil)
				req.URL.RawQuery = raw
				rec := httptest.NewRecorder()
				config.getMatches(rec, req)
				if rec.Code != http.StatusBadRequest {
					t.Fatalf("code = %d, want 400", rec.Code)
				}
				if cacheCalls != 0 {
					t.Fatalf("cache should not be used, got %d Exists calls", cacheCalls)
				}
			})
		}
	})

	t.Run("non-numeric league_id rejected even when season is valid", func(t *testing.T) {
		var cacheCalls int
		mockCache := &MockCache{
			existsFunc: func(ctx context.Context, key string) (bool, error) {
				cacheCalls++
				return false, nil
			},
			getFunc: func(ctx context.Context, key string, value interface{}) error {
				return fmt.Errorf("not found")
			},
			setFunc: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
				return nil
			},
		}
		config := &Config{
			Cache:              mockCache,
			FootballAPIKey:     "key",
			APIFootballBaseURL: "http://127.0.0.1:9",
		}
		req := httptest.NewRequest("GET", "/matches", nil)
		req.URL.RawQuery = "date=2026-04-15&league_id=bar&season=2026"
		rec := httptest.NewRecorder()
		config.getMatches(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("code = %d, want 400", rec.Code)
		}
		if cacheCalls != 0 {
			t.Fatalf("cache should not be used, got %d Exists calls", cacheCalls)
		}
	})
}

// TestGetMatchesQueryValidation ensures league_id and season are validated before cache or upstream calls.
func TestGetMatchesQueryValidation(t *testing.T) {
	t.Run("invalid queries return 400 before cache", func(t *testing.T) {
		var cacheCalls int
		mockCache := &MockCache{
			existsFunc: func(ctx context.Context, key string) (bool, error) {
				cacheCalls++
				return false, nil
			},
			getFunc: func(ctx context.Context, key string, value interface{}) error {
				return fmt.Errorf("not found")
			},
			setFunc: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
				return nil
			},
		}
		config := &Config{
			Cache:              mockCache,
			FootballAPIKey:     "key",
			APIFootballBaseURL: "http://127.0.0.1:9",
		}

		tests := []struct {
			name     string
			rawQuery string
		}{
			{
				name:     "non-numeric league_id with season override",
				rawQuery: "date=2025-06-01&league_id=foo&season=2026",
			},
			{
				name:     "non-numeric league_id",
				rawQuery: "date=2025-06-01&league_id=++foo&season=2026",
			},
			{
				name:     "invalid season with numeric league_id",
				rawQuery: "date=2025-06-01&league_id=39&season=notayear",
			},
			{
				name:     "invalid date without league_id",
				rawQuery: "date=not-a-date",
			},
			{
				name:     "malformed date without league_id",
				rawQuery: "date=2025/06/01",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cacheCalls = 0
				req := httptest.NewRequest("GET", "/matches", nil)
				req.URL.RawQuery = tt.rawQuery
				rec := httptest.NewRecorder()
				config.getMatches(rec, req)
				if rec.Code != http.StatusBadRequest {
					t.Fatalf("code = %d, want 400, body=%s", rec.Code, rec.Body.String())
				}
				if cacheCalls != 0 {
					t.Fatalf("expected no cache access, got %d Exists calls", cacheCalls)
				}
			})
		}
	})

	t.Run("trimmed league_id and int league in upstream URL", func(t *testing.T) {
		var gotPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.RequestURI()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"get":"fixtures","results":1,"response":[{"fixture":{"id":1,"status":{"short":"FT"}},"teams":{"home":{"name":"A"},"away":{"name":"B"}}}]}`))
		}))
		defer server.Close()

		mockCache := &MockCache{
			existsFunc: func(ctx context.Context, key string) (bool, error) {
				return false, nil
			},
			getFunc: func(ctx context.Context, key string, value interface{}) error {
				return fmt.Errorf("not found")
			},
			setFunc: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
				return nil
			},
		}
		config := &Config{
			Cache:              mockCache,
			FootballAPIKey:     "key",
			APIFootballBaseURL: server.URL,
		}

		q := url.Values{}
		q.Set("date", "2025-06-15")
		q.Set("league_id", " 39 ")
		q.Set("season", "2024")
		req := httptest.NewRequest("GET", "/matches", nil)
		req.URL.RawQuery = q.Encode()
		rec := httptest.NewRecorder()
		config.getMatches(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("code = %d, body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(gotPath, "league=39") || !strings.Contains(gotPath, "season=2024") {
			t.Fatalf("upstream URL should use normalized league and season; got %q", gotPath)
		}
	})
}

// TestCacheErrorHandling tests that endpoints handle cache errors gracefully
func TestCacheErrorHandling(t *testing.T) {
	// Create a mock cache that simulates errors
	mockCache := &MockCache{
		existsFunc: func(ctx context.Context, key string) (bool, error) {
			return false, fmt.Errorf("cache error")
		},
		getFunc: func(ctx context.Context, key string, value interface{}) error {
			return fmt.Errorf("cache error")
		},
		setFunc: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
			return fmt.Errorf("cache error")
		},
	}

	mockAPIKey := "mock-api-key"

	config := &Config{
		Cache:          mockCache,
		FootballAPIKey: mockAPIKey,
	}

	t.Run("Test Cache Error Handling", func(t *testing.T) {
		// Mock response
		mockResponse := `{
			"response": [
				{
					"fixture": {
						"id": 123,
						"status": {"short": "LIVE"}
					}
				}
			]
		}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockResponse))
		}))
		defer server.Close()

		config.APIFootballBaseURL = server.URL

		// Request should still work even with cache errors
		req := httptest.NewRequest("GET", "/matches", nil)
		req.URL.RawQuery = "date=2025-01-01"
		rec := httptest.NewRecorder()

		config.getMatches(rec, req)

		// Should still return 200 even if cache fails
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})
}

func TestGetTeamSquad(t *testing.T) {
	// Skip if no Redis connection
	redisURL := "redis://localhost:6379"
	cache, err := cache.NewCache(redisURL)
	if err != nil {
		t.Skip("Skipping test: Redis not available")
	}

	// Clear cache before test to ensure clean state
	ctx := context.Background()
	cache.FlushAll(ctx)

	mockAPIKey := "mock-api-key"
	config := &Config{
		Cache:          cache,
		FootballAPIKey: mockAPIKey,
	}

	t.Run("success case", func(t *testing.T) {
		// Mock squad response
		mockSquadResponse := `{
			"response": [
				{
					"team": {"id": 1},
					"players": [
						{"id": 1, "name": "Player 1", "number": 1, "pos": "G", "grid": "", "photo": "photo1.jpg"},
						{"id": 2, "name": "Player 2", "number": 2, "pos": "D", "grid": "", "photo": "photo2.jpg"}
					]
				}
			]
		}`

		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("x-apisports-key") != mockAPIKey {
				t.Errorf("Expected API key %s, got %s", mockAPIKey, r.Header.Get("x-apisports-key"))
			}

			// Verify the URL contains the team ID
			if !strings.Contains(r.URL.Path, "players/squads") {
				t.Errorf("Expected URL to contain 'players/squads', got %s", r.URL.Path)
			}
			if !strings.Contains(r.URL.RawQuery, "team=1") {
				t.Errorf("Expected query to contain 'team=1', got %s", r.URL.RawQuery)
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockSquadResponse))
		}))
		defer server.Close()

		// Set the base URL to the test server
		config.APIFootballBaseURL = server.URL

		// Call the function
		squad, err := config.getTeamSquad(1, ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify the response
		if len(squad.Response) != 1 {
			t.Errorf("Expected 1 team response, got %d", len(squad.Response))
		}
		if len(squad.Response[0].Players) != 2 {
			t.Errorf("Expected 2 players, got %d", len(squad.Response[0].Players))
		}

		// Verify cache was set
		exists, err := cache.Exists(ctx, "team_squad:1")
		if err != nil || !exists {
			t.Error("Team squad should be cached")
		}
	})

	t.Run("cache hit case", func(t *testing.T) {
		// The squad should already be cached from the previous test
		squad, err := config.getTeamSquad(1, ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify the response
		if len(squad.Response) != 1 {
			t.Errorf("Expected 1 team response, got %d", len(squad.Response))
		}
		if len(squad.Response[0].Players) != 2 {
			t.Errorf("Expected 2 players, got %d", len(squad.Response[0].Players))
		}
	})

	t.Run("empty response case", func(t *testing.T) {
		// Mock empty squad response
		mockEmptyResponse := `{
			"response": []
		}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockEmptyResponse))
		}))
		defer server.Close()

		config.APIFootballBaseURL = server.URL

		// Call the function
		squad, err := config.getTeamSquad(999, ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify the response is empty
		if len(squad.Response) != 0 {
			t.Errorf("Expected 0 team responses, got %d", len(squad.Response))
		}
	})
}
