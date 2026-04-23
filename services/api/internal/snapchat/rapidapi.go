// Package snapchat calls RapidAPI snapchat6 (user stories). Keys stay server-side only.
package snapchat

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	rapidHost   = "snapchat6.p.rapidapi.com"
	storiesPath = "/user/stories"

	// maxUserStoriesBodyBytes caps the upstream JSON body; responses larger than this fail closed (no silent truncation).
	maxUserStoriesBodyBytes = 1 << 24 // 16 MiB
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// FetchUserStories returns the upstream status code and raw JSON body (pass-through for the app).
// On failure, err is *FetchError when classification is known; use HTTPStatusForFetchError(err).
func FetchUserStories(ctx context.Context, rapidAPIKey, username string) (body []byte, status int, err error) {
	if strings.TrimSpace(rapidAPIKey) == "" {
		return nil, 0, MisconfiguredError("empty RapidAPI key")
	}
	u := strings.ToLower(strings.TrimSpace(username))
	if u == "" || !isPlausibleUsername(u) {
		return nil, 0, InvalidInputError("invalid Snapchat username")
	}
	escaped := url.QueryEscape(u)
	reqURL := fmt.Sprintf("https://%s%s?username=%s", rapidHost, storiesPath, escaped)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, 0, InternalError("failed to build upstream request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-rapidapi-host", rapidHost)
	req.Header.Set("x-rapidapi-key", rapidAPIKey)

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, UpstreamError("upstream request failed", err)
	}
	defer res.Body.Close()
	// Read at most max+1 bytes so we can detect bodies strictly larger than the cap (LimitReader alone would truncate silently).
	b, err := io.ReadAll(io.LimitReader(res.Body, int64(maxUserStoriesBodyBytes+1)))
	if err != nil {
		return nil, res.StatusCode, UpstreamError("failed to read upstream response", err)
	}
	if len(b) > maxUserStoriesBodyBytes {
		_, _ = io.Copy(io.Discard, res.Body)
		return nil, res.StatusCode, UpstreamError(
			fmt.Sprintf("upstream response body exceeds maximum of %d bytes", maxUserStoriesBodyBytes),
			nil,
		)
	}
	return b, res.StatusCode, nil
}

func isPlausibleUsername(s string) bool {
	if len(s) > 64 {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}
