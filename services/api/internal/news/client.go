package news

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"
)

// Throttle RapidAPI news; interval is configurable per client (default 1 req/s for BASIC plan; e.g. 100ms for 10 req/s).
var (
	newsRateMu      sync.Mutex
	newsLastRequest time.Time
)

// RapidAPIResponse represents the response from RapidAPI Real-Time News Data
type RapidAPIResponse struct {
	Status    string            `json:"status"`
	RequestID string            `json:"request_id"`
	Data      []RapidAPIArticle `json:"data"`
}

// RapidAPIArticle represents a single article from RapidAPI
type RapidAPIArticle struct {
	Title                string `json:"title"`
	Link                 string `json:"link"`
	Snippet              string `json:"snippet,omitempty"`
	PhotoURL             string `json:"photo_url,omitempty"`
	PublishedDatetimeUTC string `json:"published_datetime_utc"`
	SourceName           string `json:"source_name"`
	SourceURL            string `json:"source_url"`
	SourceLogoURL        string `json:"source_logo_url,omitempty"`
	SourceFaviconURL     string `json:"source_favicon_url,omitempty"`
}

// Client wraps the RapidAPI Real-Time News Data API client
type Client struct {
	apiKey             string
	baseURL            string
	timeout            time.Duration
	MinRequestInterval time.Duration // Min time between requests (default 1s; use 100ms for 10 req/s plans)
}

const defaultNewsBaseURL = "https://api.openwebninja.com/realtime-news-data/search"

// NewClient creates a new RapidAPI news client (MinRequestInterval defaults to 1s for BASIC plan).
func NewClient(apiKey string) *Client {
	return NewClientWithBaseURL(apiKey, defaultNewsBaseURL)
}

// NewClientWithBaseURL creates a client with a custom base URL (e.g. for tests).
// MinRequestInterval defaults to 1s (1 req/s); set env NEWS_RATE_LIMIT_RPS to e.g. "10" for 10 req/s plans.
func NewClientWithBaseURL(apiKey, baseURL string) *Client {
	interval := time.Second
	if rps := os.Getenv("NEWS_RATE_LIMIT_RPS"); rps != "" {
		if n, err := strconv.Atoi(rps); err == nil && n > 0 {
			interval = time.Second / time.Duration(n)
		}
	}
	return &Client{
		apiKey:             apiKey,
		baseURL:            baseURL,
		timeout:            10 * time.Second,
		MinRequestInterval: interval,
	}
}

// FetchNewsOptions contains options for fetching news
type FetchNewsOptions struct {
	Query         string
	TimePublished string
	Limit         int
	Country       string
	Lang          string
}

// buildParams returns url.Values with only non-empty entries from pairs, plus limit when > 0.
func buildParams(pairs map[string]string, limit int) url.Values {
	params := url.Values{}
	for k, v := range pairs {
		if v != "" {
			params.Add(k, v)
		}
	}
	if limit > 0 {
		params.Add("limit", fmt.Sprintf("%d", limit))
	}
	return params
}

// throttleNewsRequest waits until at least minInterval has passed since the last request.
// It is context-aware: if ctx is cancelled during the wait, returns ctx.Err() without updating lastRequest.
// The mutex is not held while sleeping so other goroutines can observe/update the next-allowed time.
func throttleNewsRequest(ctx context.Context, minInterval time.Duration) error {
	if minInterval <= 0 {
		minInterval = time.Second
	}
	newsRateMu.Lock()
	elapsed := time.Since(newsLastRequest)
	sleep := time.Duration(0)
	if elapsed < minInterval {
		sleep = minInterval - elapsed
	}
	newsRateMu.Unlock()

	if sleep > 0 {
		timer := time.NewTimer(sleep)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			// waited; fall through to update lastRequest
		}
	}

	newsRateMu.Lock()
	newsLastRequest = time.Now()
	newsRateMu.Unlock()
	return nil
}

// FetchNews fetches football news from RapidAPI with custom options.
func (c *Client) FetchNews(ctx context.Context, opts FetchNewsOptions) (*RapidAPIResponse, error) {
	if err := throttleNewsRequest(ctx, c.MinRequestInterval); err != nil {
		return nil, err
	}
	params := buildParams(map[string]string{
		"query":          opts.Query,
		"time_published": opts.TimePublished,
		"country":        opts.Country,
		"lang":           opts.Lang,
	}, opts.Limit)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)

	// Make the request with timeout
	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch news: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check if response is successful
	if resp.StatusCode != http.StatusOK {
		log.Printf("news API returned status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var newsResponse RapidAPIResponse
	if err := json.Unmarshal(body, &newsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &newsResponse, nil
}

// TodayAndHistoryResponse contains both today's and historical news responses
type TodayAndHistoryResponse struct {
	TodayResponse   *RapidAPIResponse
	HistoryResponse *RapidAPIResponse
}

// FetchTodayAndHistoryNews fetches both today's news and historical news.
// Returns partial success (empty slice for a failed section) when at least one request succeeds.
// Returns an error only when both today and history requests fail, so the handler can return 5xx and avoid caching an empty payload.
func (c *Client) FetchTodayAndHistoryNews(ctx context.Context) (*TodayAndHistoryResponse, error) {
	defaultOpts := FetchNewsOptions{
		Lang:  "en",
		Limit: 5,
	}

	// Fetch today's news
	todayOpts := defaultOpts
	todayOpts.Query = "FIFA Football News"
	todayOpts.TimePublished = "1d"

	todayResp, todayErr := c.FetchNews(ctx, todayOpts)
	if todayErr != nil {
		log.Printf("Failed to fetch today's news: %v", todayErr)
		todayResp = &RapidAPIResponse{Data: []RapidAPIArticle{}}
	}

	// Fetch historical news
	historyOpts := defaultOpts
	historyOpts.Query = "World FIFA Football History"
	historyOpts.TimePublished = "anytime"

	historyResp, historyErr := c.FetchNews(ctx, historyOpts)
	if historyErr != nil {
		log.Printf("Failed to fetch historical news: %v", historyErr)
		historyResp = &RapidAPIResponse{Data: []RapidAPIArticle{}}
	}

	// If both failed, return error so handler can return 5xx and not cache empty response
	if todayErr != nil && historyErr != nil {
		return nil, fmt.Errorf("today and history news fetch failed: today=%v, history=%w", todayErr, historyErr)
	}

	return &TodayAndHistoryResponse{
		TodayResponse:   todayResp,
		HistoryResponse: historyResp,
	}, nil
}

// MatchStatusCompleted returns true if the match has finished (FT, AET, PEN, etc.)
func MatchStatusCompleted(status string) bool {
	switch status {
	case "FT", "AET", "PEN", "AWD", "WO", "CANC", "ABD", "PST":
		return true
	default:
		return false
	}
}

// FetchMatchNews fetches news for a specific match (both teams).
// Time filtering:
// - Not started / ongoing: fetches articles from past 1 day
// - Completed: fetches articles from past 7 days, then filters to only those published after match end
func (c *Client) FetchMatchNews(ctx context.Context, homeTeam, awayTeam string, limit int, matchStatus string, matchEndTime *time.Time) (*RapidAPIResponse, error) {
	// Build combined query: "Team A and Team B"
	query := fmt.Sprintf("%s and %s", homeTeam, awayTeam)

	timePublished := "1d"
	if MatchStatusCompleted(matchStatus) {
		timePublished = "7d"
	}

	opts := FetchNewsOptions{
		Query:         query,
		TimePublished: timePublished,
		Limit:         limit,
		Country:       "US",
		Lang:          "en",
	}

	resp, err := c.FetchNews(ctx, opts)
	if err != nil {
		return nil, err
	}

	// For completed matches, filter to only articles published after match end
	if MatchStatusCompleted(matchStatus) && matchEndTime != nil {
		filtered := make([]RapidAPIArticle, 0, len(resp.Data))
		for _, article := range resp.Data {
			pubTime, err := time.Parse(time.RFC3339, article.PublishedDatetimeUTC)
			if err != nil {
				pubTime, err = time.Parse("2006-01-02T15:04:05Z", article.PublishedDatetimeUTC)
				if err != nil {
					continue // Skip articles with unparseable timestamps
				}
			}
			if !pubTime.Before(*matchEndTime) {
				filtered = append(filtered, article)
			}
		}
		resp.Data = filtered
	}

	return resp, nil
}
