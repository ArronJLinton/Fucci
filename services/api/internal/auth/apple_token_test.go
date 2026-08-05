package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
)

func TestExchangeAppleAuthCode_NotConfigured(t *testing.T) {
	_, err := ExchangeAppleAuthCode(context.Background(), AppleTokenConfig{}, "code")
	if err == nil || err != ErrAppleTokenNotConfigured {
		t.Fatalf("expected ErrAppleTokenNotConfigured, got %v", err)
	}
}

func TestRevokeAppleToken_NotConfigured(t *testing.T) {
	err := RevokeAppleToken(context.Background(), AppleTokenConfig{}, "refresh-token")
	if err == nil || err != ErrAppleTokenNotConfigured {
		t.Fatalf("expected ErrAppleTokenNotConfigured, got %v", err)
	}
}

func TestRevokeAppleToken_EmptyTokenNoOp(t *testing.T) {
	if err := RevokeAppleToken(context.Background(), AppleTokenConfig{}, ""); err != nil {
		t.Fatalf("expected nil for empty token, got %v", err)
	}
}

func TestBuildAppleClientSecret_SignsWithECDSAKey(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	secret, err := buildAppleClientSecret(AppleTokenConfig{
		ClientID:   "com.magistridev.fucci",
		TeamID:     "TEAMID123",
		KeyID:      "KEYID123",
		PrivateKey: string(pemBytes),
	})
	if err != nil {
		t.Fatalf("buildAppleClientSecret: %v", err)
	}
	if secret == "" {
		t.Fatal("expected non-empty client secret JWT")
	}
}
