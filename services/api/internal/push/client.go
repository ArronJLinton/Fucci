package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	expoMaxRetries     = 3
	expoRetryBaseDelay = 500 * time.Millisecond
)

const expoPushURL = "https://exp.host/--/api/v2/push/send"

// Message is a single Expo push notification payload.
type Message struct {
	To       string                 `json:"to"`
	Title    string                 `json:"title,omitempty"`
	Body     string                 `json:"body,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Sound    string                 `json:"sound,omitempty"`
	Priority string                 `json:"priority,omitempty"`
}

type ticketResponse struct {
	Data []ticketItem `json:"data"`
}

type ticketItem struct {
	Status string `json:"status"`
	ID     string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
	Details *struct {
		Error string `json:"error,omitempty"`
	} `json:"details,omitempty"`
}

// SendResult is the outcome for one token in a batch.
type SendResult struct {
	Token      string
	TicketID   string
	Status     string
	Error      string
	DeviceNotRegistered bool
}

// HTTPDoer is satisfied by *http.Client (for tests).
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client sends batched notifications to the Expo Push API.
type Client struct {
	HTTP    HTTPDoer
	Token   string // EXPO_ACCESS_TOKEN (optional but recommended in prod)
	Timeout time.Duration
}

func (c *Client) httpClient() HTTPDoer {
	if c.HTTP != nil {
		return c.HTTP
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &http.Client{Timeout: timeout}
}

// retryAfterDelay parses the Retry-After header (integer seconds) and returns
// the appropriate wait duration, falling back to base if the header is absent or invalid.
func retryAfterDelay(header string, base time.Duration) time.Duration {
	if header != "" {
		if secs, err := strconv.Atoi(header); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return base
}

// Send delivers up to 100 messages per request (Expo limit).
// Retries up to expoMaxRetries times on 429/5xx with exponential backoff,
// honouring the Retry-After header when present.
func (c *Client) Send(ctx context.Context, messages []Message) ([]SendResult, error) {
	if len(messages) == 0 {
		return nil, nil
	}
	if len(messages) > 100 {
		return nil, fmt.Errorf("expo batch limit is 100, got %d", len(messages))
	}
	body, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}

	nextDelay := expoRetryBaseDelay
	var lastErr error

	for attempt := 0; attempt <= expoMaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(nextDelay):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, expoPushURL, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}

		resp, err := c.httpClient().Do(req)
		if err != nil {
			lastErr = err
			nextDelay *= 2
			continue
		}

		raw, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			nextDelay *= 2
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("expo push HTTP %d: %s", resp.StatusCode, string(raw))
			nextDelay = retryAfterDelay(resp.Header.Get("Retry-After"), nextDelay*2)
			continue
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("expo push HTTP %d: %s", resp.StatusCode, string(raw))
		}

		var parsed ticketResponse
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, fmt.Errorf("decode expo response: %w", err)
		}

		results := make([]SendResult, len(messages))
		for i, msg := range messages {
			results[i].Token = msg.To
			if i >= len(parsed.Data) {
				results[i].Status = "error"
				results[i].Error = "missing ticket in response"
				continue
			}
			t := parsed.Data[i]
			results[i].Status = t.Status
			results[i].TicketID = t.ID
			if t.Status == "error" {
				results[i].Error = t.Message
				if t.Details != nil && t.Details.Error == "DeviceNotRegistered" {
					results[i].DeviceNotRegistered = true
				}
			}
		}
		return results, nil
	}
	return nil, lastErr
}
