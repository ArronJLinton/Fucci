package auth

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const googleOAuthStatePurpose = "google_oauth_start"

// ErrJWTNotInitialized means JWT signing cannot run because no secret was loaded (JWT_SECRET / InitJWTAuth).
var ErrJWTNotInitialized = errors.New("jwt not initialized")

// GoogleOAuthStateClaims binds the browser OAuth round-trip to a safe app return URL.
type GoogleOAuthStateClaims struct {
	Purpose   string `json:"purpose"`
	ReturnURL string `json:"return_url"`
	jwt.RegisteredClaims
}

// SignGoogleOAuthState issues a short-lived JWT used as the OAuth `state` parameter.
func SignGoogleOAuthState(returnURL string) (string, error) {
	if len(jwtSecret) == 0 {
		return "", ErrJWTNotInitialized
	}
	if strings.TrimSpace(returnURL) == "" {
		return "", errors.New("missing return url")
	}
	now := time.Now()
	claims := &GoogleOAuthStateClaims{
		Purpose:   googleOAuthStatePurpose,
		ReturnURL: strings.TrimSpace(returnURL),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "fucci-google-oauth",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(jwtSecret)
}

// ParseGoogleOAuthState validates `state` and returns the app return URL embedded at /start.
func ParseGoogleOAuthState(raw string) (string, error) {
	if raw == "" {
		return "", errors.New("missing state")
	}
	if len(jwtSecret) == 0 {
		return "", ErrJWTNotInitialized
	}
	claims := &GoogleOAuthStateClaims{}
	_, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil {
		return "", err
	}
	if claims.Purpose != googleOAuthStatePurpose {
		return "", errors.New("invalid oauth state purpose")
	}
	if strings.TrimSpace(claims.ReturnURL) == "" {
		return "", errors.New("missing return url in state")
	}
	return claims.ReturnURL, nil
}

// AllowedGoogleAppReturnURI rejects open redirects. When allowDevReturns is false (production default),
// only the app deep link scheme (fucci) and the Expo auth proxy (https://auth.expo.io) are accepted.
// When allowDevReturns is true (development, or GOOGLE_OAUTH_ALLOW_DEV_RETURN_URLS), exp:// and
// http(s):// localhost / common private LAN hosts are also allowed for Expo Go, Expo web, and emulators.
func AllowedGoogleAppReturnURI(raw string, allowDevReturns bool) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" {
		return false
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())

	switch scheme {
	case "fucci":
		return true
	case "https":
		if host == "auth.expo.io" {
			return true
		}
		if !allowDevReturns {
			return false
		}
		// Expo web dev uses https://localhost; keep to loopback / private LAN only.
		return host == "localhost" || host == "127.0.0.1" || host == "10.0.2.2" ||
			strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "10.")
	case "exp", "http":
		if !allowDevReturns {
			return false
		}
		if scheme == "exp" {
			return true
		}
		return host == "localhost" || host == "127.0.0.1" || host == "10.0.2.2" ||
			strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "10.")
	default:
		return false
	}
}
