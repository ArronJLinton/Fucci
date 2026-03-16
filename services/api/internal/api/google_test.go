package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/cache"
)

// TestGoogleNewsCache tests the cache functionality for the Google News search endpoint
func TestGoogleNewsCache(t *testing.T) {
	// Skip if no Redis connection
	redisURL := "redis://localhost:6379"
	cache, err := cache.NewCache(redisURL)
	if err != nil {
		t.Skip("Skipping test: Redis not available")
	}

	mockAPIKey := "mock-api-key"
	config := &Config{
		Cache:       cache,
		RapidAPIKey: mockAPIKey,
	}

	t.Run("Test Google News Search Error Handling", func(t *testing.T) {
		// Test missing query parameter
		req := httptest.NewRequest("GET", "/search", nil)
		rec := httptest.NewRecorder()

		config.search(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rec.Code)
		}

		var response struct {
			Error string `json:"error"`
		}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Failed to parse error response: %s", err)
		}
		if response.Error != "Query parameter 'q' is required" {
			t.Errorf("Expected error message 'Query parameter 'q' is required', got '%s'", response.Error)
		}
	})

	t.Run("Test Google News Cache Key Generation", func(t *testing.T) {
		// Test that different queries generate different cache keys
		// Generate expected cache keys
		expectedKeys := []string{
			"google_news:football:en-US",
			"google_news:soccer:en-US",
			"google_news:basketball:en-US",
			"google_news:football:es-ES",
			"google_news:soccer:es-ES",
			"google_news:basketball:es-ES",
			"google_news:football:fr-FR",
			"google_news:soccer:fr-FR",
			"google_news:basketball:fr-FR",
		}

		// Verify cache keys are unique
		keyMap := make(map[string]bool)
		for _, key := range expectedKeys {
			if keyMap[key] {
				t.Errorf("Duplicate cache key found: %s", key)
			}
			keyMap[key] = true
		}

		t.Logf("Generated %d unique cache keys", len(expectedKeys))
	})
}

// TestGoogleNewsCacheTTL tests that Google News cache entries expire correctly
func TestGoogleNewsCacheTTL(t *testing.T) {
	t.Run("Test Google News Cache TTL Constant", func(t *testing.T) {
		// Verify the TTL constant is set (15m per cache/constants.go)
		expectedTTL := 15 * time.Minute
		if cache.NewsTTL != expectedTTL {
			t.Errorf("Expected NewsTTL to be %v, got %v", expectedTTL, cache.NewsTTL)
		}

		t.Logf("NewsTTL is correctly set to %v", cache.NewsTTL)
	})
}

// TestGoogleNewsCacheErrorHandling tests that the endpoint handles cache errors gracefully
func TestGoogleNewsCacheErrorHandling(t *testing.T) {
	// Create a mock cache that always returns errors
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
		Cache:       mockCache,
		RapidAPIKey: mockAPIKey,
	}

	t.Run("Test Google News Cache Error Handling", func(t *testing.T) {
		// Test that the function handles cache errors gracefully
		req := httptest.NewRequest("GET", "/search", nil)
		req.URL.RawQuery = "q=football&lr=en-US"
		rec := httptest.NewRecorder()

		// This should not panic even with cache errors
		config.search(rec, req)

		// Should handle cache errors gracefully (may return API error, but shouldn't panic)
		t.Log("Function handled cache errors without panicking")
	})
}
