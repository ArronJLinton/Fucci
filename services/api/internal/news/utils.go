package news

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
)

// GenerateArticleID creates a unique identifier for a news article from its URL
// Uses SHA256 hash of the URL to ensure consistent IDs for the same article
//
// Example:
//
//	articleID := GenerateArticleID("https://example.com/article/123")
//	// Returns: "a1b2c3d4e5f6..." (64 character hex string)
func GenerateArticleID(articleURL string) (string, error) {
	if articleURL == "" {
		return "", fmt.Errorf("article URL cannot be empty")
	}

	// Require absolute http(s) URL so we reject relative or invalid inputs (e.g. "foo", "/path")
	u, err := url.Parse(articleURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("article URL must be http or https, got scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("article URL must have a host")
	}

	// Generate SHA256 hash of the URL
	hash := sha256.Sum256([]byte(articleURL))

	// Convert to hex string (64 characters)
	articleID := hex.EncodeToString(hash[:])

	return articleID, nil
}
