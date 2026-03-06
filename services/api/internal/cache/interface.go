package cache

import (
	"context"
	"time"
)

// CacheInterface defines the interface for cache operations
type CacheInterface interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Exists(ctx context.Context, key string) (bool, error)
	Delete(ctx context.Context, key string) error
	DeletePattern(ctx context.Context, pattern string) error
	FlushAll(ctx context.Context) error
	HealthCheck(ctx context.Context) error
	GetStats(ctx context.Context) (map[string]interface{}, error)
	// Incr increments the key by 1; if key does not exist it is set to 1. Returns the new value.
	Incr(ctx context.Context, key string) (int64, error)
	// Expire sets the TTL for the key.
	Expire(ctx context.Context, key string, ttl time.Duration) error
	// TTL returns the key's remaining TTL; if < 0 the key has no expiry or does not exist.
	TTL(ctx context.Context, key string) (time.Duration, error)
}
