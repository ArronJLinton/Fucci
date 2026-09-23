package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestValidateToken_RejectsExpiredToken(t *testing.T) {
	if err := InitJWTAuth("test-secret-jwt-expiry"); err != nil {
		t.Fatal(err)
	}

	token, err := GenerateToken(42, "user@example.com", "fan", time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)

	if _, err := ValidateToken(token); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestValidateToken_RejectsPastExpInPayload(t *testing.T) {
	if err := InitJWTAuth("test-secret-jwt-expiry"); err != nil {
		t.Fatal(err)
	}

	past := time.Now().Add(-2 * time.Hour)
	rawClaims := jwt.MapClaims{
		"user_id": float64(7),
		"email":   "old@example.com",
		"role":    "fan",
		"exp":     float64(past.Unix()),
		"iat":     float64(past.Add(-time.Hour).Unix()),
		"nbf":     float64(past.Add(-time.Hour).Unix()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, rawClaims)
	signed, err := token.SignedString(jwtSecret)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ValidateToken(signed); err == nil {
		t.Fatal("expected past-exp token to be rejected")
	}
}

func TestValidateToken_AcceptsUnexpiredToken(t *testing.T) {
	if err := InitJWTAuth("test-secret-jwt-expiry"); err != nil {
		t.Fatal(err)
	}

	token, err := GenerateToken(99, "ok@example.com", "fan", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("expected unexpired token to validate: %v", err)
	}
	if claims.UserID != 99 || claims.Email != "ok@example.com" || claims.Role != "fan" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if claims.ExpiresAt == nil || claims.ExpiresAt.Time.Before(time.Now()) {
		t.Fatalf("expected RegisteredClaims.ExpiresAt to be populated in the future, got %v", claims.ExpiresAt)
	}
}

func TestValidateToken_RejectsTokenWithoutExp(t *testing.T) {
	if err := InitJWTAuth("test-secret-jwt-expiry"); err != nil {
		t.Fatal(err)
	}

	rawClaims := jwt.MapClaims{
		"user_id": float64(3),
		"email":   "noexp@example.com",
		"role":    "fan",
		"iat":     float64(time.Now().Unix()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, rawClaims)
	signed, err := token.SignedString(jwtSecret)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ValidateToken(signed); err == nil {
		t.Fatal("expected token without exp to be rejected")
	}
}

func TestGenerateToken_RejectsNonPositiveExpiration(t *testing.T) {
	if err := InitJWTAuth("test-secret-jwt-expiry"); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateToken(1, "a@b.com", "fan", 0); err == nil {
		t.Fatal("expected zero expiration to be rejected")
	}
	if _, err := GenerateToken(1, "a@b.com", "fan", -time.Hour); err == nil {
		t.Fatal("expected negative expiration to be rejected")
	}
}

func TestValidateToken_RejectsNoneAlg(t *testing.T) {
	if err := InitJWTAuth("test-secret-jwt-expiry"); err != nil {
		t.Fatal(err)
	}

	claims := &JWTClaims{
		UserID: 1,
		Email:  "x@y.com",
		Role:   "fan",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(signed, ".") {
		// none-alg tokens end with a trailing dot and empty signature
		t.Fatalf("unexpected none-alg token shape: %q", signed)
	}

	if _, err := ValidateToken(signed); err == nil {
		t.Fatal("expected alg=none token to be rejected")
	}
}
