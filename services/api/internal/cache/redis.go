package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

func NewCache(redisURL string) (*Cache, error) {
	// Clean up the URL by removing any newlines and extra spaces
	redisURL = strings.TrimSpace(redisURL)
	redisURL = strings.ReplaceAll(redisURL, "\n", "")

	// If URL is empty, return error
	if redisURL == "" {
		return nil, fmt.Errorf("redis URL is empty")
	}

	// If URL is just "redis://", it's incomplete
	if redisURL == "redis://" {
		return nil, fmt.Errorf("incomplete redis URL")
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %v", err)
	}

	client := redis.NewClient(opt)

	// Test the connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to ping Redis
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return &Cache{
		client: client,
	}, nil
}

// Set stores a value in the cache with the given key and expiration time
func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal data for cache: %v", err)
	}

	return c.client.Set(ctx, key, data, expiration).Err()
}

// Get retrieves a value from the cache by key and unmarshals it into the provided interface
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil // Key not found
		}
		return err
	}

	return json.Unmarshal(data, dest)
}

// Delete removes a key from the cache
func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// DeletePattern removes all keys matching a pattern
func (c *Cache) DeletePattern(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("failed to delete key %s: %v", iter.Val(), err)
		}
	}
	return iter.Err()
}

// FlushAll removes all keys from the cache
func (c *Cache) FlushAll(ctx context.Context) error {
	return c.client.FlushAll(ctx).Err()
}

// Exists checks if a key exists in the cache
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// HealthCheck performs a health check on the Redis connection
func (c *Cache) HealthCheck(ctx context.Context) error {
	// Try to ping Redis
	err := c.client.Ping(ctx).Err()
	if err != nil {
		return fmt.Errorf("Redis ping failed: %v", err)
	}

	// Try a test key write/read
	testKey := "health_check"
	testValue := "test_value"

	// Try to write
	err = c.client.Set(ctx, testKey, testValue, 1*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("Redis write test failed: %v", err)
	}

	// Try to read
	val, err := c.client.Get(ctx, testKey).Result()
	if err != nil {
		return fmt.Errorf("Redis read test failed: %v", err)
	}

	if val != testValue {
		return fmt.Errorf("Redis value mismatch")
	}

	return nil
}

// Incr increments the key by 1; if the key does not exist it is set to 1. Returns the new value.
func (c *Cache) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// Expire sets the TTL for the key.
func (c *Cache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

// GetStats returns basic cache statistics
func (c *Cache) GetStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := c.client.Info(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %v", err)
	}

	// Parse basic stats from info
	stats := make(map[string]interface{})
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "db0:") {
			stats["database"] = line
		}
		if strings.HasPrefix(line, "used_memory_human:") {
			stats["memory_used"] = strings.TrimPrefix(line, "used_memory_human:")
		}
		if strings.HasPrefix(line, "connected_clients:") {
			stats["connected_clients"] = strings.TrimPrefix(line, "connected_clients:")
		}
	}

	// Get total keys
	keys, err := c.client.DBSize(ctx).Result()
	if err != nil {
		stats["total_keys"] = "unknown"
	} else {
		stats["total_keys"] = keys
	}

	return stats, nil
}
