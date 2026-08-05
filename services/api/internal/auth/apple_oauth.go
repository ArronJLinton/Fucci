package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const appleJWKSURL = "https://appleid.apple.com/auth/keys"
const appleIssuer = "https://appleid.apple.com"

var (
	ErrAppleIdentityTokenInvalid = errors.New("apple identity token invalid")
	ErrAppleIdentityTokenVerify  = errors.New("apple identity token verification failed")
)

// AppleIDTokenClaims are the claims we rely on after verifying Apple's identity token.
type AppleIDTokenClaims struct {
	Subject       string
	Email         string
	EmailVerified bool
}

type appleJWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type appleJWKS struct {
	Keys []appleJWK `json:"keys"`
}

// AppleIdentityVerifier verifies Sign in with Apple identity tokens (JWKS).
type AppleIdentityVerifier struct {
	clientID   string
	jwksURL    string
	httpClient *http.Client

	mu        sync.RWMutex
	keys      map[string]any
	fetchedAt time.Time
}

func NewAppleIdentityVerifier(clientID string) *AppleIdentityVerifier {
	return &AppleIdentityVerifier{
		clientID:   strings.TrimSpace(clientID),
		jwksURL:    appleJWKSURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		keys:       make(map[string]any),
	}
}

func (a *AppleIdentityVerifier) VerifyIdentityToken(ctx context.Context, identityToken string) (AppleIDTokenClaims, error) {
	identityToken = strings.TrimSpace(identityToken)
	if identityToken == "" {
		return AppleIDTokenClaims{}, ErrAppleIdentityTokenInvalid
	}
	if strings.TrimSpace(a.clientID) == "" {
		return AppleIDTokenClaims{}, fmt.Errorf("%w: apple client id not configured", ErrAppleIdentityTokenVerify)
	}

	parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256", "ES256"}))
	token, err := parser.Parse(identityToken, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, ErrAppleIdentityTokenInvalid
		}
		key, err := a.keyForKid(ctx, kid)
		if err != nil {
			return nil, err
		}
		return key, nil
	})
	if err != nil || !token.Valid {
		return AppleIDTokenClaims{}, fmt.Errorf("%w: %v", ErrAppleIdentityTokenVerify, err)
	}

	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return AppleIDTokenClaims{}, ErrAppleIdentityTokenInvalid
	}

	iss, _ := mapClaims.GetIssuer()
	if iss != appleIssuer {
		return AppleIDTokenClaims{}, fmt.Errorf("%w: unexpected issuer", ErrAppleIdentityTokenVerify)
	}
	aud, err := mapClaims.GetAudience()
	if err != nil || len(aud) == 0 || aud[0] != a.clientID {
		return AppleIDTokenClaims{}, fmt.Errorf("%w: unexpected audience", ErrAppleIdentityTokenVerify)
	}
	exp, err := mapClaims.GetExpirationTime()
	if err != nil || exp == nil || exp.Before(time.Now().Add(-30*time.Second)) {
		return AppleIDTokenClaims{}, fmt.Errorf("%w: token expired", ErrAppleIdentityTokenVerify)
	}

	sub, _ := mapClaims.GetSubject()
	if strings.TrimSpace(sub) == "" {
		return AppleIDTokenClaims{}, ErrAppleIdentityTokenInvalid
	}

	email, _ := mapClaims["email"].(string)
	emailVerified := false
	switch v := mapClaims["email_verified"].(type) {
	case bool:
		emailVerified = v
	case string:
		emailVerified = strings.EqualFold(v, "true")
	}

	return AppleIDTokenClaims{
		Subject:       sub,
		Email:         strings.TrimSpace(email),
		EmailVerified: emailVerified,
	}, nil
}

func (a *AppleIdentityVerifier) keyForKid(ctx context.Context, kid string) (any, error) {
	a.mu.RLock()
	key, ok := a.keys[kid]
	fresh := time.Since(a.fetchedAt) < time.Hour
	a.mu.RUnlock()
	if ok && fresh {
		return key, nil
	}

	if err := a.refreshKeys(ctx); err != nil {
		if ok {
			return key, nil
		}
		return nil, err
	}

	a.mu.RLock()
	defer a.mu.RUnlock()
	key, ok = a.keys[kid]
	if !ok {
		return nil, fmt.Errorf("%w: unknown kid", ErrAppleIdentityTokenVerify)
	}
	return key, nil
}

func (a *AppleIdentityVerifier) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.jwksURL, nil)
	if err != nil {
		return err
	}
	res, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("apple jwks status %d", res.StatusCode)
	}
	var jwks appleJWKS
	if err := json.NewDecoder(res.Body).Decode(&jwks); err != nil {
		return err
	}
	next := make(map[string]any, len(jwks.Keys))
	for _, k := range jwks.Keys {
		pub, err := jwkToPublicKey(k)
		if err != nil || k.Kid == "" {
			continue
		}
		next[k.Kid] = pub
	}
	if len(next) == 0 {
		return errors.New("apple jwks empty")
	}
	a.mu.Lock()
	a.keys = next
	a.fetchedAt = time.Now()
	a.mu.Unlock()
	return nil
}

func jwkToPublicKey(k appleJWK) (any, error) {
	switch strings.ToUpper(k.Kty) {
	case "RSA":
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, err
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, err
		}
		var eInt int
		for _, b := range eBytes {
			eInt = eInt<<8 + int(b)
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: eInt}, nil
	case "EC":
		if k.Crv != "P-256" {
			return nil, fmt.Errorf("unsupported curve %s", k.Crv)
		}
		xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			return nil, err
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
		if err != nil {
			return nil, err
		}
		return &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(xBytes),
			Y:     new(big.Int).SetBytes(yBytes),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported kty %s", k.Kty)
	}
}
