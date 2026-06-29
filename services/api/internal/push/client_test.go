package push

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClient_Send_OK(t *testing.T) {
	t.Parallel()
	body := `{"data":[{"status":"ok","id":"ticket-abc"}]}`
	client := &Client{
		HTTP: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != expoPushURL {
					t.Fatalf("url = %s", r.URL.String())
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}
	results, err := client.Send(context.Background(), []Message{
		{To: "ExponentPushToken[abc]", Title: "Hi"},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(results) != 1 || results[0].TicketID != "ticket-abc" || results[0].Status != "ok" {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestClient_Send_DeviceNotRegistered(t *testing.T) {
	t.Parallel()
	body := `{"data":[{"status":"error","message":"not registered","details":{"error":"DeviceNotRegistered"}}]}`
	client := &Client{
		HTTP: &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(body)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}
	results, err := client.Send(context.Background(), []Message{{To: "ExponentPushToken[x]"}})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !results[0].DeviceNotRegistered {
		t.Fatal("expected DeviceNotRegistered")
	}
}
