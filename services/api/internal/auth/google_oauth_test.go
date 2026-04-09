package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestGoogleOAuthVerifier_ValidateRedirectURI(t *testing.T) {
	verifier := NewGoogleOAuthVerifier("client-id", "secret", []string{
		"fucci://auth",
		"com.fucci.app:/oauth2redirect",
	})

	tests := []struct {
		name    string
		uri     string
		wantErr bool
	}{
		{name: "ios allowed", uri: "fucci://auth"},
		{name: "android allowed", uri: "com.fucci.app:/oauth2redirect"},
		{name: "unknown uri rejected", uri: "https://example.com/callback", wantErr: true},
		{name: "empty uri rejected", uri: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := verifier.ValidateRedirectURI(tc.uri)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for uri %q", tc.uri)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error for uri %q, got %v", tc.uri, err)
			}
		})
	}
}

func TestToBool(t *testing.T) {
	if !toBool(true) {
		t.Fatal("expected true for bool true")
	}
	if !toBool("true") {
		t.Fatal("expected true for string true")
	}
	if toBool("false") {
		t.Fatal("expected false for string false")
	}
}

func TestGoogleOAuthVerifier_VerifyIDToken_ValidatesIssuerAndTimeClaims(t *testing.T) {
	now := time.Now().Unix()
	basePayload := func(iss string, exp int64, iat int64) string {
		return fmt.Sprintf(`{
			"sub":"sub-1",
			"email":"user@example.com",
			"email_verified":"true",
			"given_name":"First",
			"family_name":"Last",
			"picture":"https://example.com/p.png",
			"locale":"en",
			"aud":"client-id",
			"iss":"%s",
			"exp":"%d",
			"iat":"%d"
		}`, iss, exp, iat)
	}

	tests := []struct {
		name      string
		body      string
		shouldErr bool
	}{
		{
			name:      "valid tokeninfo payload",
			body:      basePayload("https://accounts.google.com", now+600, now-60),
			shouldErr: false,
		},
		{
			name:      "issuer mismatch",
			body:      basePayload("https://evil.example.com", now+600, now-60),
			shouldErr: true,
		},
		{
			name:      "expired token",
			body:      basePayload("accounts.google.com", now-1, now-100),
			shouldErr: true,
		},
		{
			name:      "iat too far in future",
			body:      basePayload("accounts.google.com", now+600, now+1000),
			shouldErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			verifier := NewGoogleOAuthVerifier("client-id", "secret", nil)
			verifier.httpClient = &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     make(http.Header),
						Body:       io.NopCloser(strings.NewReader(tc.body)),
					}, nil
				}),
			}

			_, err := verifier.VerifyIDToken(context.Background(), "dummy-token")
			if tc.shouldErr && err == nil {
				t.Fatalf("expected error but got none")
			}
			if !tc.shouldErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
