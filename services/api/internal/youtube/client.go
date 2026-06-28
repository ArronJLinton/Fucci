package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	searchPath  = "/youtube/v3/search"
	videosPath  = "/youtube/v3/videos"
	apiBaseURL  = "https://www.googleapis.com"
	maxShortSec = 180 // 3 minutes — YouTube Shorts max length
	searchMax   = 10
	lookbackDays = 7
)

// Short is a playable YouTube Short returned to clients.
type Short struct {
	VideoID      string    `json:"video_id"`
	Title        string    `json:"title"`
	ThumbnailURL string    `json:"thumbnail_url"`
	EmbedURL     string    `json:"embed_url"`
	Duration     string    `json:"duration"`
	PublishedAt  time.Time `json:"published_at"`
}

type searchSnippet struct {
	Title       string    `json:"title"`
	PublishedAt time.Time `json:"publishedAt"`
	Thumbnails  struct {
		High struct {
			URL string `json:"url"`
		} `json:"high"`
		Medium struct {
			URL string `json:"url"`
		} `json:"medium"`
		Default struct {
			URL string `json:"url"`
		} `json:"default"`
	} `json:"thumbnails"`
}

type searchListResponse struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
		Snippet searchSnippet `json:"snippet"`
	} `json:"items"`
}

type videosListResponse struct {
	Items []struct {
		ID             string `json:"id"`
		Status         struct {
			Embeddable     bool   `json:"embeddable"`
			PrivacyStatus  string `json:"privacyStatus"`
		} `json:"status"`
		ContentDetails struct {
			Duration string `json:"duration"`
		} `json:"contentDetails"`
	} `json:"items"`
}

// Client calls the YouTube Data API v3.
type Client struct {
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		APIKey: strings.TrimSpace(apiKey),
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// FetchShortsForChannel returns embeddable public Shorts published within the lookback window.
func (c *Client) FetchShortsForChannel(ctx context.Context, channelID string) ([]Short, error) {
	if c == nil || c.APIKey == "" {
		return nil, fmt.Errorf("youtube client not configured")
	}
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return nil, fmt.Errorf("channel id required")
	}

	publishedAfter := time.Now().UTC().AddDate(0, 0, -lookbackDays).Truncate(24 * time.Hour).Format(time.RFC3339)

	searchParams := url.Values{}
	searchParams.Set("part", "snippet")
	searchParams.Set("channelId", channelID)
	searchParams.Set("type", "video")
	searchParams.Set("videoDuration", "short")
	searchParams.Set("order", "date")
	searchParams.Set("maxResults", fmt.Sprintf("%d", searchMax))
	searchParams.Set("publishedAfter", publishedAfter)
	searchParams.Set("key", c.APIKey)

	searchURL := apiBaseURL + searchPath + "?" + searchParams.Encode()
	searchBody, err := c.get(ctx, searchURL)
	if err != nil {
		return nil, err
	}

	var searchResp searchListResponse
	if err := json.Unmarshal(searchBody, &searchResp); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	if len(searchResp.Items) == 0 {
		return []Short{}, nil
	}

	snippetByID := make(map[string]searchSnippet, len(searchResp.Items))
	videoIDs := make([]string, 0, len(searchResp.Items))
	for _, item := range searchResp.Items {
		id := strings.TrimSpace(item.ID.VideoID)
		if id == "" {
			continue
		}
		videoIDs = append(videoIDs, id)
		snippetByID[id] = item.Snippet
	}
	if len(videoIDs) == 0 {
		return []Short{}, nil
	}

	videosParams := url.Values{}
	videosParams.Set("part", "status,contentDetails")
	videosParams.Set("id", strings.Join(videoIDs, ","))
	videosParams.Set("key", c.APIKey)

	videosURL := apiBaseURL + videosPath + "?" + videosParams.Encode()
	videosBody, err := c.get(ctx, videosURL)
	if err != nil {
		return nil, err
	}

	var videosResp videosListResponse
	if err := json.Unmarshal(videosBody, &videosResp); err != nil {
		return nil, fmt.Errorf("decode videos response: %w", err)
	}

	shorts := make([]Short, 0, len(videosResp.Items))
	for _, item := range videosResp.Items {
		if !item.Status.Embeddable || item.Status.PrivacyStatus != "public" {
			continue
		}
		duration := item.ContentDetails.Duration
		if !IsShortDuration(duration, maxShortSec) {
			continue
		}
		snippet, ok := snippetByID[item.ID]
		if !ok {
			continue
		}
		thumb := snippet.Thumbnails.High.URL
		if thumb == "" {
			thumb = snippet.Thumbnails.Medium.URL
		}
		if thumb == "" {
			thumb = snippet.Thumbnails.Default.URL
		}
		shorts = append(shorts, Short{
			VideoID:      item.ID,
			Title:        snippet.Title,
			ThumbnailURL: thumb,
			EmbedURL:     fmt.Sprintf("https://www.youtube.com/embed/%s", item.ID),
			Duration:     duration,
			PublishedAt:  snippet.PublishedAt,
		})
	}

	return shorts, nil
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &FetchError{
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}
	return body, nil
}
