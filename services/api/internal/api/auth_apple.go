package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/ArronJLinton/fucci-api/internal/database"
)

type AppleAuthFullName struct {
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
}

type AppleAuthRequest struct {
	IdentityToken     string             `json:"identity_token"`
	AuthorizationCode string             `json:"authorization_code,omitempty"`
	FullName          *AppleAuthFullName `json:"full_name,omitempty"`
}

// AppleAuthResponse matches GoogleAuthResponse shape for shared mobile session handling.
type AppleAuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
	IsNew bool         `json:"is_new"`
}

type appleAuthProcError struct {
	status int
	code   string
	msg    string
}

func (c *Config) appleVerifier() AppleIdentityTokenVerifier {
	if c.AppleVerifier != nil {
		return c.AppleVerifier
	}
	c.appleVerifierOnce.Do(func() {
		c.lazyAppleVerifier = auth.NewAppleIdentityVerifier(c.AppleClientID)
	})
	return c.lazyAppleVerifier
}

func (c *Config) appleTokenConfig() auth.AppleTokenConfig {
	return auth.AppleTokenConfig{
		ClientID:   c.AppleClientID,
		TeamID:     c.AppleTeamID,
		KeyID:      c.AppleKeyID,
		PrivateKey: c.ApplePrivateKey,
	}
}

// handleAppleAuth authenticates with a Sign in with Apple identity token.
// POST /auth/apple
func (c *Config) handleAppleAuth(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(c.AppleClientID) == "" {
		respondWithErrorCode(w, http.StatusServiceUnavailable, "Apple Sign In is not configured on the server", "APPLE_NOT_CONFIGURED")
		return
	}

	var req AppleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithErrorCode(w, http.StatusBadRequest, "invalid request body", "INVALID_BODY")
		return
	}
	if strings.TrimSpace(req.IdentityToken) == "" {
		respondWithErrorCode(w, http.StatusBadRequest, "identity_token is required", "INVALID_TOKEN")
		return
	}
	if c.DB == nil && c.DBConn == nil {
		respondWithError(w, http.StatusInternalServerError, "database not configured")
		return
	}

	out, procErr := c.appleAuthFromIdentityToken(r.Context(), req)
	if procErr != nil {
		respondWithErrorCode(w, procErr.status, procErr.msg, procErr.code)
		return
	}
	respondWithJSON(w, http.StatusOK, out)
}

func (c *Config) appleAuthFromIdentityToken(ctx context.Context, req AppleAuthRequest) (AppleAuthResponse, *appleAuthProcError) {
	verifier := c.appleVerifier()
	if verifier == nil {
		return AppleAuthResponse{}, &appleAuthProcError{status: http.StatusServiceUnavailable, code: "APPLE_NOT_CONFIGURED", msg: "Apple Sign In is not configured on the server"}
	}

	claims, err := verifier.VerifyIdentityToken(ctx, req.IdentityToken)
	if err != nil {
		log.Printf("apple auth verify: %v", err)
		return AppleAuthResponse{}, &appleAuthProcError{status: http.StatusUnauthorized, code: "INVALID_TOKEN", msg: "invalid Apple identity token"}
	}

	subject := strings.TrimSpace(claims.Subject)
	email := strings.ToLower(strings.TrimSpace(claims.Email))

	var refreshToken sql.NullString
	if code := strings.TrimSpace(req.AuthorizationCode); code != "" {
		if tok, exchErr := auth.ExchangeAppleAuthCode(ctx, c.appleTokenConfig(), code); exchErr != nil {
			if !errors.Is(exchErr, auth.ErrAppleTokenNotConfigured) {
				log.Printf("apple auth code exchange (non-fatal): %v", exchErr)
			}
		} else if tok != "" {
			refreshToken = sql.NullString{String: tok, Valid: true}
		}
	}

	given := ""
	family := ""
	if req.FullName != nil {
		given = strings.TrimSpace(req.FullName.GivenName)
		family = strings.TrimSpace(req.FullName.FamilyName)
	}

	q := c.dbQueries()
	var u database.Users
	var isNew bool

	u, err = q.GetUserByAppleID(ctx, subject)
	switch {
	case err == nil:
		if u.IsActive.Valid && !u.IsActive.Bool {
			return AppleAuthResponse{}, &appleAuthProcError{status: http.StatusForbidden, code: "ACCOUNT_INACTIVE", msg: "This account has been deactivated"}
		}
		u, err = q.UpdateAppleLoginFields(ctx, database.UpdateAppleLoginFieldsParams{
			ID:                u.ID,
			AppleRefreshToken: refreshToken,
		})
		if err != nil {
			return AppleAuthResponse{}, &appleAuthProcError{status: http.StatusInternalServerError, code: "UPSTREAM", msg: "failed to update apple login"}
		}
	case errors.Is(err, sql.ErrNoRows):
		if email == "" {
			return AppleAuthResponse{}, &appleAuthProcError{
				status: http.StatusBadRequest,
				code:   "EMAIL_REQUIRED",
				msg:    "Apple did not provide an email for this new account. Try Sign in with Apple again and share your email, or use another sign-in method.",
			}
		}
		byEmail, emailErr := q.GetUserByEmailLower(ctx, email)
		if emailErr == nil {
			if byEmail.IsActive.Valid && !byEmail.IsActive.Bool {
				return AppleAuthResponse{}, &appleAuthProcError{status: http.StatusForbidden, code: "ACCOUNT_INACTIVE", msg: "This account has been deactivated"}
			}
			existingAppleID := strings.TrimSpace(byEmail.AppleID.String)
			if byEmail.AppleID.Valid && existingAppleID != "" && existingAppleID != subject {
				return AppleAuthResponse{}, &appleAuthProcError{status: http.StatusConflict, code: "ACCOUNT_EXISTS_EMAIL", msg: "Email already linked to another Apple account"}
			}
			prov := strings.TrimSpace(byEmail.AuthProvider)
			if strings.EqualFold(prov, "email") {
				return AppleAuthResponse{}, &appleAuthProcError{status: http.StatusConflict, code: "ACCOUNT_EXISTS_EMAIL", msg: "Email already registered via password"}
			}
			if !strings.EqualFold(prov, "apple") {
				return AppleAuthResponse{}, &appleAuthProcError{status: http.StatusConflict, code: "ACCOUNT_EXISTS_EMAIL", msg: "Email already linked to another account provider"}
			}
			u, err = q.LinkAppleToExistingUser(ctx, database.LinkAppleToExistingUserParams{
				ID:                byEmail.ID,
				NewAppleID:        subject,
				AppleRefreshToken: refreshToken,
			})
			if err != nil {
				return AppleAuthResponse{}, &appleAuthProcError{status: http.StatusInternalServerError, code: "UPSTREAM", msg: "failed to link apple account"}
			}
		} else if errors.Is(emailErr, sql.ErrNoRows) {
			if given == "" {
				given = "Fucci"
			}
			if family == "" {
				family = "Fan"
			}
			u, err = q.CreateAppleUser(ctx, database.CreateAppleUserParams{
				Firstname:         given,
				Lastname:          family,
				Email:             email,
				AppleID:           sql.NullString{String: subject, Valid: true},
				AppleRefreshToken: refreshToken,
			})
			if err != nil {
				log.Printf("create apple user: %v", err)
				return AppleAuthResponse{}, &appleAuthProcError{status: http.StatusInternalServerError, code: "UPSTREAM", msg: "failed to create apple user"}
			}
			isNew = true
		} else {
			return AppleAuthResponse{}, &appleAuthProcError{status: http.StatusInternalServerError, code: "UPSTREAM", msg: "failed to check existing account"}
		}
	default:
		return AppleAuthResponse{}, &appleAuthProcError{status: http.StatusInternalServerError, code: "UPSTREAM", msg: "failed to lookup apple user"}
	}

	userResponse := userResponseFromDBUser(u)
	if !userResponse.IsActive {
		return AppleAuthResponse{}, &appleAuthProcError{status: http.StatusForbidden, code: "ACCOUNT_INACTIVE", msg: "This account has been deactivated"}
	}

	token, err := auth.GenerateToken(userResponse.ID, userResponse.Email, userResponse.Role, 24*time.Hour)
	if err != nil {
		return AppleAuthResponse{}, &appleAuthProcError{status: http.StatusInternalServerError, code: "UPSTREAM", msg: "failed to generate auth token"}
	}

	return AppleAuthResponse{Token: token, User: userResponse, IsNew: isNew}, nil
}
