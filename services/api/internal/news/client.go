package news

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
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
	apiKey  string
	baseURL string
	timeout time.Duration
}

// NewClient creates a new RapidAPI news client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://real-time-news-data.p.rapidapi.com/search",
		timeout: 10 * time.Second,
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

// FetchNews fetches football news from RapidAPI with custom options
func (c *Client) FetchNews(opts FetchNewsOptions) (*RapidAPIResponse, error) {
	params := buildParams(map[string]string{
		"query":          opts.Query,
		"time_published": opts.TimePublished,
		"country":        opts.Country,
		"lang":           opts.Lang,
	}, opts.Limit)

	// Create HTTP request
	req, err := http.NewRequest("GET", c.baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add RapidAPI headers
	req.Header.Add("X-RapidAPI-Key", c.apiKey)
	req.Header.Add("X-RapidAPI-Host", "real-time-news-data.p.rapidapi.com")

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
		log.Printf("RapidAPI returned status %d: %s", resp.StatusCode, string(body))
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
func (c *Client) FetchTodayAndHistoryNews() (*TodayAndHistoryResponse, error) {
	defaultOpts := FetchNewsOptions{
		Lang:  "en",
		Limit: 5,
	}

	// Fetch today's news
	todayOpts := defaultOpts
	todayOpts.Query = "FIFA Football News"
	todayOpts.TimePublished = "1d"

	todayResp, todayErr := c.FetchNews(todayOpts)
	if todayErr != nil {
		log.Printf("Failed to fetch today's news: %v", todayErr)
		todayResp = &RapidAPIResponse{Data: []RapidAPIArticle{}}
	}

	// Fetch historical news
	historyOpts := defaultOpts
	historyOpts.Query = "World FIFA Football History"
	historyOpts.TimePublished = "anytime"

	historyResp, historyErr := c.FetchNews(historyOpts)
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
func (c *Client) FetchMatchNews(homeTeam, awayTeam string, limit int, matchStatus string, matchEndTime *time.Time) (*RapidAPIResponse, error) {
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

	resp, err := c.FetchNews(opts)
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
