package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAppleIdentityVerifier_VerifyIdentityToken(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	kid := "test-kid"
	clientID := "com.magistridev.fucci"

	n := base64.RawURLEncoding.EncodeToString(priv.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(priv.E)).Bytes())
	jwksBody, _ := json.Marshal(map[string]any{
		"keys": []map[string]string{{
			"kty": "RSA",
			"kid": kid,
			"use": "sig",
			"alg": "RS256",
			"n":   n,
			"e":   e,
		}},
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwksBody)
	}))
	defer srv.Close()

	sign := func(claims jwt.MapClaims) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		tok.Header["kid"] = kid
		s, err := tok.SignedString(priv)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		return s
	}

	now := time.Now()
	validClaims := jwt.MapClaims{
		"iss":            appleIssuer,
		"aud":            clientID,
		"sub":            "apple.user.1",
		"email":          "user@example.com",
		"email_verified": "true",
		"exp":            now.Add(10 * time.Minute).Unix(),
		"iat":            now.Unix(),
	}

	tests := []struct {
		name      string
		token     string
		shouldErr bool
	}{
		{
			name:  "valid token",
			token: sign(validClaims),
		},
		{
			name: "wrong audience",
			token: sign(jwt.MapClaims{
				"iss": appleIssuer,
				"aud": "com.other.app",
				"sub": "apple.user.1",
				"exp": now.Add(10 * time.Minute).Unix(),
			}),
			shouldErr: true,
		},
		{
			name: "wrong issuer",
			token: sign(jwt.MapClaims{
				"iss": "https://evil.example",
				"aud": clientID,
				"sub": "apple.user.1",
				"exp": now.Add(10 * time.Minute).Unix(),
			}),
			shouldErr: true,
		},
		{
			name: "expired",
			token: sign(jwt.MapClaims{
				"iss": appleIssuer,
				"aud": clientID,
				"sub": "apple.user.1",
				"exp": now.Add(-2 * time.Minute).Unix(),
			}),
			shouldErr: true,
		},
		{
			name:      "empty token",
			token:     "",
			shouldErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := NewAppleIdentityVerifier(clientID)
			v.jwksURL = srv.URL
			claims, err := v.VerifyIdentityToken(context.Background(), tc.token)
			if tc.shouldErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if claims.Subject != "apple.user.1" || claims.Email != "user@example.com" || !claims.EmailVerified {
				t.Fatalf("unexpected claims: %+v", claims)
			}
		})
	}
}

func TestAppleIdentityVerifier_MissingClientID(t *testing.T) {
	v := NewAppleIdentityVerifier("")
	_, err := v.VerifyIdentityToken(context.Background(), "not-empty")
	if err == nil {
		t.Fatal("expected error when client id missing")
	}
}
