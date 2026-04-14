package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

type JWTClaims struct {
	UserID    int32  `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"exp"`
	jwt.RegisteredClaims
}

// InitJWTAuth initializes JWT authentication with secret from config
func InitJWTAuth(secret string) error {
	if secret == "" {
		return errors.New("JWT_SECRET is not set in config")
	}
	jwtSecret = []byte(secret)
	return nil
}

// GenerateToken generates a JWT token for a user
func GenerateToken(userID int32, email, role string, expiration time.Duration) (string, error) {
	now := time.Now()
	claims := &JWTClaims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		ExpiresAt: now.Add(expiration).Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

// ExtractToken extracts the JWT token from the Authorization header
func ExtractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header is missing")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid authorization header format")
	}

	return parts[1], nil
}

// claimsCtxKey is a private type for context keys (avoids collisions with other packages).
type claimsCtxKey int

const (
	claimsKeyUserID claimsCtxKey = iota + 1
	claimsKeyUserEmail
	claimsKeyUserRole
)

// ContextWithClaims attaches JWT claims to ctx using the same keys as RequireAuth and OptionalAuth.
// Use from tests with &JWTClaims{UserID: …}; nil claims returns ctx unchanged.
func ContextWithClaims(ctx context.Context, claims *JWTClaims) context.Context {
	if claims == nil {
		return ctx
	}
	ctx = context.WithValue(ctx, claimsKeyUserID, claims.UserID)
	ctx = context.WithValue(ctx, claimsKeyUserEmail, claims.Email)
	ctx = context.WithValue(ctx, claimsKeyUserRole, claims.Role)
	return ctx
}

// UserIDFromContext returns the authenticated user id when set by RequireAuth or OptionalAuth.
func UserIDFromContext(ctx context.Context) (int32, bool) {
	v := ctx.Value(claimsKeyUserID)
	if v == nil {
		return 0, false
	}
	id, ok := v.(int32)
	return id, ok
}

// UserEmailFromContext returns the JWT email claim when present.
func UserEmailFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(claimsKeyUserEmail)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// UserRoleFromContext returns the JWT role claim when present.
func UserRoleFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(claimsKeyUserRole)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// OptionalAuth validates Bearer JWT when present and attaches user_id (and related claims) to context.
// Invalid or missing tokens are ignored so the handler can serve public responses.
func OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := ExtractToken(r)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		claims, err := ValidateToken(tokenString)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(ContextWithClaims(r.Context(), claims)))
	})
}

// RequireAuth is a middleware that validates JWT token
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := ExtractToken(r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "authentication required"}`))
			return
		}

		claims, err := ValidateToken(tokenString)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "invalid or expired token"}`))
			return
		}

		next.ServeHTTP(w, r.WithContext(ContextWithClaims(r.Context(), claims)))
	})
}

// RequireRole is a middleware that validates user role
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole, ok := UserRoleFromContext(r.Context())
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "authentication required"}`))
				return
			}

			for _, allowedRole := range allowedRoles {
				if userRole == allowedRole {
					next.ServeHTTP(w, r)
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": "insufficient permissions"}`))
		})
	}
}
