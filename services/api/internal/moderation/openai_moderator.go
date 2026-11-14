package moderation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultModerationModel = "omni-moderation-latest"

// ContentCheckInput is text and/or image URLs to evaluate synchronously.
type ContentCheckInput struct {
	Texts     []string
	ImageURLs []string
}

// ContentCheckResult is the soft-flag decision from the AI moderator.
type ContentCheckResult struct {
	Flagged    bool
	Categories []string
	Model      string
}

// ContentChecker evaluates UGC for policy violations (soft-flag path).
type ContentChecker interface {
	Check(ctx context.Context, input ContentCheckInput) (ContentCheckResult, error)
}

// OpenAIModerator calls OpenAI's Moderations API (omni-moderation-latest).
type OpenAIModerator struct {
	APIKey  string
	BaseURL string
	Model   string
	Client  *http.Client
}

func NewOpenAIModerator(apiKey, baseURL string) *OpenAIModerator {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	return &OpenAIModerator{
		APIKey:  strings.TrimSpace(apiKey),
		BaseURL: base,
		Model:   defaultModerationModel,
		Client:  &http.Client{Timeout: 25 * time.Second},
	}
}

type moderationRequest struct {
	Model string        `json:"model"`
	Input []interface{} `json:"input"`
}

type moderationResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Results []struct {
		Flagged        bool            `json:"flagged"`
		Categories     map[string]bool `json:"categories"`
		CategoryScores map[string]any  `json:"category_scores"`
	} `json:"results"`
}

// Check runs a synchronous moderation request. Empty input is a no-op (not flagged).
func (m *OpenAIModerator) Check(ctx context.Context, input ContentCheckInput) (ContentCheckResult, error) {
	if m == nil || strings.TrimSpace(m.APIKey) == "" {
		return ContentCheckResult{}, nil
	}

	parts := make([]interface{}, 0, len(input.Texts)+len(input.ImageURLs))
	for _, t := range input.Texts {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		parts = append(parts, map[string]string{"type": "text", "text": t})
	}
	for _, u := range input.ImageURLs {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		parts = append(parts, map[string]interface{}{
			"type":      "image_url",
			"image_url": map[string]string{"url": u},
		})
	}
	if len(parts) == 0 {
		return ContentCheckResult{Model: m.modelName()}, nil
	}

	body, err := json.Marshal(moderationRequest{
		Model: m.modelName(),
		Input: parts,
	})
	if err != nil {
		return ContentCheckResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.BaseURL+"/moderations", bytes.NewReader(body))
	if err != nil {
		return ContentCheckResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+m.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := m.Client
	if client == nil {
		client = &http.Client{Timeout: 25 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return ContentCheckResult{}, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return ContentCheckResult{}, fmt.Errorf("openai moderation status %d: %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}

	var parsed moderationResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ContentCheckResult{}, err
	}

	out := ContentCheckResult{Model: parsed.Model}
	if out.Model == "" {
		out.Model = m.modelName()
	}
	cats := map[string]struct{}{}
	for _, r := range parsed.Results {
		if r.Flagged {
			out.Flagged = true
		}
		for name, on := range r.Categories {
			if on {
				cats[name] = struct{}{}
			}
		}
	}
	for name := range cats {
		out.Categories = append(out.Categories, name)
	}
	return out, nil
}

func (m *OpenAIModerator) modelName() string {
	if m != nil && strings.TrimSpace(m.Model) != "" {
		return m.Model
	}
	return defaultModerationModel
}

// MediaPreviewURL returns a URL suitable for image moderation.
// Photos are passed through; videos use a Cloudinary still frame (so_0 → .jpg).
func MediaPreviewURL(mediaURL, contentType string) string {
	u := strings.TrimSpace(mediaURL)
	if u == "" {
		return ""
	}
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if ct != "video" {
		return u
	}
	// Insert so_0 transformation after /upload/ when present.
	const marker = "/upload/"
	idx := strings.Index(u, marker)
	if idx >= 0 {
		insertAt := idx + len(marker)
		// Avoid double-inserting.
		if !strings.Contains(u[insertAt:], "so_0") {
			u = u[:insertAt] + "so_0/" + u[insertAt:]
		}
	}
	// Prefer a still-image extension for the frame.
	for _, ext := range []string{".mp4", ".mov", ".webm", ".m3u8"} {
		if i := strings.LastIndex(strings.ToLower(u), ext); i > 0 {
			return u[:i] + ".jpg"
		}
	}
	return u
}

// FormatAIFlagDescription builds a short human-review note for content_reports.
func FormatAIFlagDescription(result ContentCheckResult) string {
	cats := result.Categories
	if len(cats) == 0 {
		cats = []string{"unspecified"}
	}
	return fmt.Sprintf(
		"AI soft-flag (model=%s). Categories: %s. Content remains published pending human review within 24 hours.",
		result.Model,
		strings.Join(cats, ", "),
	)
}
