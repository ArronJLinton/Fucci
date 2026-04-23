// Package snapchat calls RapidAPI snapchat6 (user stories). Keys stay server-side only.
package snapchat

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	rapidHost   = "snapchat6.p.rapidapi.com"
	storiesPath = "/user/stories"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// FetchUserStories returns the upstream status code and raw JSON body (pass-through for the app).
func FetchUserStories(ctx context.Context, rapidAPIKey, username string) (body []byte, status int, err error) {
	if strings.TrimSpace(rapidAPIKey) == "" {
		return nil, 0, fmt.Errorf("snapchat: empty RapidAPI key")
	}
	u := strings.ToLower(strings.TrimSpace(username))
	if u == "" || !isPlausibleUsername(u) {
		return nil, 0, fmt.Errorf("snapchat: invalid username")
	}
	escaped := url.QueryEscape(u)
	reqURL := fmt.Sprintf("https://%s%s?username=%s", rapidHost, storiesPath, escaped)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, 0, err
	}
	log.Println("REQ URL", reqURL)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-rapidapi-host", rapidHost)
	req.Header.Set("x-rapidapi-key", rapidAPIKey)

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 1<<24)) // 16 MiB cap
	if err != nil {
		return nil, res.StatusCode, err
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
