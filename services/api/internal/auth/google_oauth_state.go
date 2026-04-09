package auth

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const googleOAuthStatePurpose = "google_oauth_start"

// GoogleOAuthStateClaims binds the browser OAuth round-trip to a safe app return URL.
type GoogleOAuthStateClaims struct {
	Purpose   string `json:"purpose"`
	ReturnURL string `json:"return_url"`
	jwt.RegisteredClaims
}

// SignGoogleOAuthState issues a short-lived JWT used as the OAuth `state` parameter.
func SignGoogleOAuthState(returnURL string) (string, error) {
	if len(jwtSecret) == 0 {
		return "", errors.New("jwt not initialized")
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
		return "", errors.New("jwt not initialized")
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

// AllowedGoogleAppReturnURI rejects open redirects while allowing dev (exp://) and Expo proxy URLs.
func AllowedGoogleAppReturnURI(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" {
		return false
	}
	switch u.Scheme {
	case "fucci":
		return true
	case "exp":
		return true
	case "https":
		return u.Host == "auth.expo.io"
	case "http":
		h := strings.ToLower(u.Hostname())
		return h == "localhost" || h == "127.0.0.1" || h == "10.0.2.2" ||
			strings.HasPrefix(h, "192.168.") || strings.HasPrefix(h, "10.")
	default:
		if strings.HasPrefix(u.Scheme, "com.") {
			return strings.Contains(raw, ":/")
		}
		return false
	}
}
