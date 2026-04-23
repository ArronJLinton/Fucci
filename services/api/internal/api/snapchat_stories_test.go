package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/snapchat"
)

func TestGetSnapchatUserStories_MissingRapidAPIKey(t *testing.T) {
	cfg := &Config{RapidAPIKey: ""}
	req := httptest.NewRequest(http.MethodGet, "/snapchat/stories?username=psg", nil)
	rec := httptest.NewRecorder()

	cfg.getSnapchatUserStories(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
	var body struct {
		Error string `json:"error"`
		Code  string `json:"code,omitempty"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if body.Error == "" {
		t.Fatal("expected non-empty error")
	}
}

func TestGetSnapchatUserStories_MissingUsername(t *testing.T) {
	cfg := &Config{RapidAPIKey: "rapid-key"}
	req := httptest.NewRequest(http.MethodGet, "/snapchat/stories", nil)
	rec := httptest.NewRecorder()

	cfg.getSnapchatUserStories(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	var body struct {
		Error string `json:"error"`
		Code  string `json:"code,omitempty"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if got := body.Error; got != "query parameter `username` is required" {
		t.Fatalf("error: want %q, got %q", "query parameter `username` is required", got)
	}
}

func TestGetSnapchatUserStories_PassThroughStatusAndBody(t *testing.T) {
	const wantJSON = `{"ok":true,"upstream":429}`
	cfg := &Config{
		RapidAPIKey: "rapid-key",
		SnapchatUserStoriesFetch: func(ctx context.Context, rapidAPIKey, username string) ([]byte, int, error) {
			if rapidAPIKey != "rapid-key" {
				t.Errorf("rapidAPIKey: got %q", rapidAPIKey)
			}
			if username != "psg" {
				t.Errorf("username: want psg, got %q", username)
			}
			return []byte(wantJSON), http.StatusTooManyRequests, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/snapchat/stories?username=psg", nil)
	rec := httptest.NewRecorder()

	cfg.getSnapchatUserStories(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type: want application/json, got %q", ct)
	}
	if got := rec.Body.String(); got != wantJSON {
		t.Fatalf("body: want %q, got %q", wantJSON, got)
	}
}

func TestGetSnapchatUserStories_InvalidUsername_NoNetwork(t *testing.T) {
	cfg := &Config{RapidAPIKey: "rapid-key"}
	req := httptest.NewRequest(http.MethodGet, "/snapchat/stories?username=bad%20spaces", nil)
	rec := httptest.NewRecorder()

	cfg.getSnapchatUserStories(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	var body struct {
		Error string `json:"error"`
		Code  string `json:"code,omitempty"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if body.Error == "" {
		t.Fatal("expected error for invalid username")
	}
}

func TestGetSnapchatUserStories_FetchErrorUpstream502(t *testing.T) {
	cfg := &Config{
		RapidAPIKey: "rapid-key",
		SnapchatUserStoriesFetch: func(ctx context.Context, rapidAPIKey, username string) ([]byte, int, error) {
			return nil, 0, snapchat.UpstreamError("upstream", io.ErrUnexpectedEOF)
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/snapchat/stories?username=psg", nil)
	rec := httptest.NewRecorder()

	cfg.getSnapchatUserStories(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, rec.Code)
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if body.Error == "" {
		t.Fatal("expected error body")
	}
}

func TestGetSnapchatUserStories_FetchErrorDeadline504(t *testing.T) {
	cfg := &Config{
		RapidAPIKey: "rapid-key",
		SnapchatUserStoriesFetch: func(ctx context.Context, rapidAPIKey, username string) ([]byte, int, error) {
			return nil, 0, snapchat.UpstreamError("timeout", context.DeadlineExceeded)
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/snapchat/stories?username=psg", nil)
	rec := httptest.NewRecorder()

	cfg.getSnapchatUserStories(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected status %d, got %d", http.StatusGatewayTimeout, rec.Code)
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if body.Error == "" {
		t.Fatal("expected error body")
	}
}

func TestGetSnapchatUserStories_UnclassifiedError500(t *testing.T) {
	cfg := &Config{
		RapidAPIKey: "rapid-key",
		SnapchatUserStoriesFetch: func(ctx context.Context, rapidAPIKey, username string) ([]byte, int, error) {
			return nil, 0, errors.New("plain failure")
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/snapchat/stories?username=psg", nil)
	rec := httptest.NewRecorder()

	cfg.getSnapchatUserStories(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if body.Error == "" {
		t.Fatal("expected error body")
	}
}

func TestGetSnapchatUserStories_CacheHit(t *testing.T) {
	const wantKey = "snapchat_stories:v1:psg"
	wantBody := []byte(`{"from":"cache"}`)
	mock := &MockCache{
		existsFunc: func(ctx context.Context, key string) (bool, error) {
			return key == wantKey, nil
		},
		getFunc: func(ctx context.Context, key string, value interface{}) error {
			if key != wantKey {
				return nil
			}
			if ptr, ok := value.(*snapchatStoriesCached); ok {
				*ptr = snapchatStoriesCached{HTTPStatus: http.StatusOK, Body: wantBody}
			}
			return nil
		},
		setFunc: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
			t.Errorf("Set should not run on cache hit")
			return nil
		},
	}
	fetchCalls := 0
	cfg := &Config{
		Cache:       mock,
		RapidAPIKey: "k",
		SnapchatUserStoriesFetch: func(ctx context.Context, rapidAPIKey, username string) ([]byte, int, error) {
			fetchCalls++
			return []byte(`{"never":"used"}`), http.StatusOK, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/snapchat/stories?username=psg", nil)
	rec := httptest.NewRecorder()
	cfg.getSnapchatUserStories(rec, req)

	if fetchCalls != 0 {
		t.Fatalf("expected fetch bypassed on cache hit, fetchCalls=%d", fetchCalls)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != string(wantBody) {
		t.Fatalf("body: %s", rec.Body.String())
	}
}

func TestGetSnapchatUserStories_RateLimit429(t *testing.T) {
	resetSnapchatStoryMemRLForTest()
	oldIP, oldUser := snapchatStoriesIPLimitN, snapchatStoriesUserLimitN
	snapchatStoriesIPLimitN, snapchatStoriesUserLimitN = 2, 2
	defer func() {
		snapchatStoriesIPLimitN, snapchatStoriesUserLimitN = oldIP, oldUser
		resetSnapchatStoryMemRLForTest()
	}()

	cfg := &Config{
		RapidAPIKey: "k",
		SnapchatUserStoriesFetch: func(ctx context.Context, rapidAPIKey, username string) ([]byte, int, error) {
			return []byte(`{}`), http.StatusOK, nil
		},
	}
	addr := "192.0.2.77:1234"
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/snapchat/stories?username=ratelimit_probe", nil)
		req.RemoteAddr = addr
		rec := httptest.NewRecorder()
		cfg.getSnapchatUserStories(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("iter %d: want 200 got %d %s", i, rec.Code, rec.Body.String())
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/snapchat/stories?username=ratelimit_probe", nil)
	req.RemoteAddr = addr
	rec := httptest.NewRecorder()
	cfg.getSnapchatUserStories(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("want 429 got %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error == "" {
		t.Fatal("expected error JSON")
	}
}
