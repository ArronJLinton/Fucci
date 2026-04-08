package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const googleTokenEndpoint = "https://oauth2.googleapis.com/token"

var (
	ErrGoogleInvalidRedirectURI = errors.New("invalid google redirect uri")
	ErrGoogleExchangeFailed     = errors.New("google auth code exchange failed")
	ErrGoogleTokenVerifyFailed  = errors.New("google id token verification failed")
)

type GoogleIDTokenClaims struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

type googleTokenExchangeResponse struct {
	IDToken string `json:"id_token"`
}

type googleTokenInfoResponse struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
	Audience      string `json:"aud"`
}

type GoogleOAuthVerifier struct {
	clientID     string
	clientSecret string
	allowed      map[string]struct{}
	httpClient   *http.Client
}

func NewGoogleOAuthVerifier(clientID, clientSecret string, allowedRedirectURIs []string) *GoogleOAuthVerifier {
	allowed := make(map[string]struct{}, len(allowedRedirectURIs))
	for _, raw := range allowedRedirectURIs {
		uri := strings.TrimSpace(raw)
		if uri != "" {
			allowed[uri] = struct{}{}
		}
	}
	return &GoogleOAuthVerifier{
		clientID:     strings.TrimSpace(clientID),
		clientSecret: strings.TrimSpace(clientSecret),
		allowed:      allowed,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (g *GoogleOAuthVerifier) ValidateRedirectURI(uri string) error {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return ErrGoogleInvalidRedirectURI
	}
	if _, ok := g.allowed[uri]; !ok {
		return ErrGoogleInvalidRedirectURI
	}
	return nil
}

func (g *GoogleOAuthVerifier) ExchangeCodeForIDToken(ctx context.Context, code, redirectURI string) (string, error) {
	if err := g.ValidateRedirectURI(redirectURI); err != nil {
		return "", err
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", strings.TrimSpace(code))
	form.Set("client_id", g.clientID)
	form.Set("client_secret", g.clientSecret)
	form.Set("redirect_uri", strings.TrimSpace(redirectURI))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrGoogleExchangeFailed, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrGoogleExchangeFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status=%d", ErrGoogleExchangeFailed, resp.StatusCode)
	}

	var payload googleTokenExchangeResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("%w: %v", ErrGoogleExchangeFailed, err)
	}
	if strings.TrimSpace(payload.IDToken) == "" {
		return "", fmt.Errorf("%w: missing id_token", ErrGoogleExchangeFailed)
	}
	return payload.IDToken, nil
}

func (g *GoogleOAuthVerifier) VerifyIDToken(ctx context.Context, token string) (GoogleIDTokenClaims, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://oauth2.googleapis.com/tokeninfo?id_token="+url.QueryEscape(strings.TrimSpace(token)),
		nil,
	)
	if err != nil {
		return GoogleIDTokenClaims{}, fmt.Errorf("%w: %v", ErrGoogleTokenVerifyFailed, err)
	}
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return GoogleIDTokenClaims{}, fmt.Errorf("%w: %v", ErrGoogleTokenVerifyFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return GoogleIDTokenClaims{}, fmt.Errorf("%w: status=%d", ErrGoogleTokenVerifyFailed, resp.StatusCode)
	}

	var payload googleTokenInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return GoogleIDTokenClaims{}, fmt.Errorf("%w: %v", ErrGoogleTokenVerifyFailed, err)
	}
	if strings.TrimSpace(payload.Audience) != g.clientID {
		return GoogleIDTokenClaims{}, fmt.Errorf("%w: audience mismatch", ErrGoogleTokenVerifyFailed)
	}
	claims := GoogleIDTokenClaims{
		Subject:       payload.Subject,
		Email:         strings.ToLower(strings.TrimSpace(payload.Email)),
		EmailVerified: toBool(payload.EmailVerified),
		GivenName:     payload.GivenName,
		FamilyName:    payload.FamilyName,
		Picture:       payload.Picture,
		Locale:        payload.Locale,
	}
	return claims, nil
}

func toBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.EqualFold(x, "true")
	default:
		return false
	}
}

