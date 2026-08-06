package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const appleTokenURL = "https://appleid.apple.com/auth/token"
const appleRevokeURL = "https://appleid.apple.com/auth/revoke"

var ErrAppleTokenNotConfigured = errors.New("apple token API not configured")

// AppleTokenConfig holds Sign in with Apple key material for auth-code exchange / revoke.
type AppleTokenConfig struct {
	ClientID   string
	TeamID     string
	KeyID      string
	PrivateKey string // PEM contents of the .p8 key
}

func (c AppleTokenConfig) configured() bool {
	return strings.TrimSpace(c.ClientID) != "" &&
		strings.TrimSpace(c.TeamID) != "" &&
		strings.TrimSpace(c.KeyID) != "" &&
		strings.TrimSpace(c.PrivateKey) != ""
}

func buildAppleClientSecret(cfg AppleTokenConfig) (string, error) {
	if !cfg.configured() {
		return "", ErrAppleTokenNotConfigured
	}
	block, _ := pem.Decode([]byte(cfg.PrivateKey))
	if block == nil {
		// Some secret stores store raw PKCS8 without PEM headers.
		raw := strings.TrimSpace(cfg.PrivateKey)
		der, err := decodeLoosePKCS8(raw)
		if err != nil {
			return "", fmt.Errorf("invalid apple private key: %w", err)
		}
		key, err := x509.ParsePKCS8PrivateKey(der)
		if err != nil {
			return "", fmt.Errorf("parse apple private key: %w", err)
		}
		ecKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return "", errors.New("apple private key is not ECDSA")
		}
		return signAppleClientSecret(cfg, ecKey)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse apple private key: %w", err)
	}
	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return "", errors.New("apple private key is not ECDSA")
	}
	return signAppleClientSecret(cfg, ecKey)
}

func decodeLoosePKCS8(raw string) ([]byte, error) {
	// Accept base64 body of a .p8 without headers by wrapping PEM.
	pemBytes := []byte("-----BEGIN PRIVATE KEY-----\n" + raw + "\n-----END PRIVATE KEY-----")
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("could not decode private key")
	}
	return block.Bytes, nil
}

func signAppleClientSecret(cfg AppleTokenConfig, key *ecdsa.PrivateKey) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": cfg.TeamID,
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"aud": "https://appleid.apple.com",
		"sub": cfg.ClientID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = cfg.KeyID
	return token.SignedString(key)
}

type appleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	Error        string `json:"error"`
}

// ExchangeAppleAuthCode exchanges an authorization code for tokens. Returns refresh_token when present.
func ExchangeAppleAuthCode(ctx context.Context, cfg AppleTokenConfig, code string) (string, error) {
	if !cfg.configured() {
		return "", ErrAppleTokenNotConfigured
	}
	clientSecret, err := buildAppleClientSecret(cfg)
	if err != nil {
		return "", err
	}
	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", clientSecret)
	form.Set("code", strings.TrimSpace(code))
	form.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, appleTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var out appleTokenResponse
	_ = json.Unmarshal(body, &out)
	if res.StatusCode != http.StatusOK {
		if out.Error != "" {
			return "", fmt.Errorf("apple token exchange: %s", out.Error)
		}
		return "", fmt.Errorf("apple token exchange status %d", res.StatusCode)
	}
	return strings.TrimSpace(out.RefreshToken), nil
}

// RevokeAppleToken revokes a refresh or access token with Apple.
func RevokeAppleToken(ctx context.Context, cfg AppleTokenConfig, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	if !cfg.configured() {
		return ErrAppleTokenNotConfigured
	}
	clientSecret, err := buildAppleClientSecret(cfg)
	if err != nil {
		return err
	}
	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", clientSecret)
	form.Set("token", token)
	form.Set("token_type_hint", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, appleRevokeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("apple revoke status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
