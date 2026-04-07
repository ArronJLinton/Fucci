package futbol

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIFootballClient_MissingAPIKey(t *testing.T) {
	c := NewAPIFootballClient("https://example.com", "")
	_, err := c.FetchLineup(context.Background(), "1")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestAPIFootballClient_UpstreamStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer srv.Close()

	c := NewAPIFootballClient(srv.URL, "test-key")
	_, err := c.FetchLeagues(context.Background(), "2025")
	if !errors.Is(err, ErrUpstream) {
		t.Fatalf("expected ErrUpstream, got %v", err)
	}
}

func TestAPIFootballClient_ParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{invalid-json"))
	}))
	defer srv.Close()

	c := NewAPIFootballClient(srv.URL, "test-key")
	_, err := c.FetchMatchStats(context.Background(), "99")
	if !errors.Is(err, ErrParse) {
		t.Fatalf("expected ErrParse, got %v", err)
	}
}
