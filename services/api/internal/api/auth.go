package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/google/uuid"
)

// LoginRequest represents the login request payload (email-only)
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents the login response payload
type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type GoogleAuthRequest struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirect_uri"`
}

type GoogleAuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
	IsNew bool         `json:"is_new"`
}

type googleAuthErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// UserResponse represents a user without sensitive data
type UserResponse struct {
	ID          int32  `json:"id"`
	Firstname   string `json:"firstname"`
	Lastname    string `json:"lastname"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	IsVerified  bool   `json:"is_verified"`
	IsActive    bool   `json:"is_active"`
	Role        string `json:"role"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func (c *Config) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Email) == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	// Get user by email
	var user struct {
		ID        int32
		Firstname string
		Lastname  string
		Email     string
	}
	err := c.DBConn.QueryRowContext(r.Context(),
		`SELECT id, firstname, lastname, email FROM users WHERE email = $1 AND is_active = true LIMIT 1`,
		strings.TrimSpace(req.Email),
	).Scan(&user.ID, &user.Firstname, &user.Lastname, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	// Verify password
	var passwordHash string
	err = c.DBConn.QueryRow(
		"SELECT password_hash FROM users WHERE id = $1",
		user.ID,
	).Scan(&passwordHash)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if err := auth.VerifyPassword(req.Password, passwordHash); err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	// Update last login
	_, err = c.DBConn.Exec(
		"UPDATE users SET last_login_at = $1 WHERE id = $2",
		time.Now(), user.ID,
	)
	if err != nil {
		// Log error but don't fail the login
		fmt.Printf("Failed to update last_login_at: %v\n", err)
	}

	// Generate JWT token
	// Default role is 'fan' if not set
	role := "fan"
	if err := c.DBConn.QueryRow(
		"SELECT role FROM users WHERE id = $1",
		user.ID,
	).Scan(&role); err != nil {
		fmt.Printf("Failed to get user role: %v\n", err)
	}

	token, err := auth.GenerateToken(user.ID, user.Email, role, 24*time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	// Build user response
	var displayName, avatarURL, createdAt, updatedAt string
	var isVerified, isActive bool
	c.DBConn.QueryRow(
		"SELECT COALESCE(display_name, ''), COALESCE(avatar_url, ''), is_verified, is_active, created_at, updated_at FROM users WHERE id = $1",
		user.ID,
	).Scan(&displayName, &avatarURL, &isVerified, &isActive, &createdAt, &updatedAt)

	userResponse := UserResponse{
		ID:          user.ID,
		Firstname:   user.Firstname,
		Lastname:    user.Lastname,
		Email:       user.Email,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		IsVerified:  isVerified,
		IsActive:    isActive,
		Role:        role,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	response := LoginResponse{
		Token: token,
		User:  userResponse,
	}

	respondWithJSON(w, http.StatusOK, response)
}

type googleAuthProcError struct {
	status int
	code   string
	msg    string
}

func logGoogleAuthEvent(event string, kv ...any) {
	log.Printf("[auth][google] event=%s kv=%v", event, kv)
}

// googleAuthFromCode exchanges an OAuth code and returns a session (used by POST /auth/google and GET /auth/google/callback).
func (c *Config) googleAuthFromCode(ctx context.Context, code, redirectURI string) (GoogleAuthResponse, *googleAuthProcError) {
	verifier := c.googleVerifier()
	idToken, err := verifier.ExchangeCodeForIDToken(ctx, code, redirectURI)
	if err != nil {
		if errors.Is(err, auth.ErrGoogleInvalidRedirectURI) {
			return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusBadRequest, code: auth.GoogleAuthInvalidRedirectURI, msg: "redirect_uri is not allowed"}
		}
		return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusBadRequest, code: auth.GoogleAuthCodeInvalid, msg: "malformed or expired auth code"}
	}

	claims, err := verifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusUnauthorized, code: auth.GoogleAuthTokenVerifyFailed, msg: "unable to verify Google ID token"}
	}
	if !claims.EmailVerified {
		return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusBadRequest, code: auth.GoogleAuthEmailNotVerified, msg: "Google email is not verified"}
	}

	email := strings.ToLower(strings.TrimSpace(claims.Email))
	subject := strings.TrimSpace(claims.Subject)

	var userID int32
	var isNew bool
	var role string
	err = c.DBConn.QueryRowContext(
		ctx,
		`SELECT id, COALESCE(role, 'fan') FROM users WHERE google_id = $1 LIMIT 1`,
		subject,
	).Scan(&userID, &role)
	switch {
	case err == nil:
		_, _ = c.DBConn.ExecContext(
			ctx,
			`UPDATE users SET last_login_at = CURRENT_TIMESTAMP, avatar_url = COALESCE(NULLIF($2, ''), avatar_url), updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
			userID,
			strings.TrimSpace(claims.Picture),
		)
	case err == sql.ErrNoRows:
		var existingID int32
		var existingProvider sql.NullString
		qerr := c.DBConn.QueryRowContext(
			ctx,
			`SELECT id, auth_provider FROM users WHERE lower(email) = lower($1) LIMIT 1`,
			email,
		).Scan(&existingID, &existingProvider)
		if qerr == nil {
			if strings.EqualFold(existingProvider.String, "email") {
				return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusConflict, code: auth.GoogleAuthAccountExistsEmail, msg: "Email already registered via password"}
			}
			_, _ = c.DBConn.ExecContext(
				ctx,
				`UPDATE users SET google_id = COALESCE(NULLIF(google_id, ''), $2), auth_provider = COALESCE(auth_provider, 'google'), last_login_at = CURRENT_TIMESTAMP, avatar_url = COALESCE(NULLIF($3, ''), avatar_url), updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
				existingID,
				subject,
				strings.TrimSpace(claims.Picture),
			)
			userID = existingID
		} else if qerr == sql.ErrNoRows {
			err = c.DBConn.QueryRowContext(
				ctx,
				`INSERT INTO users (firstname, lastname, email, google_id, auth_provider, avatar_url, locale, is_admin, is_active, is_verified, last_login_at)
				 VALUES ($1,$2,$3,$4,'google',$5,$6,false,true,true,CURRENT_TIMESTAMP) RETURNING id`,
				strings.TrimSpace(claims.GivenName),
				strings.TrimSpace(claims.FamilyName),
				email,
				subject,
				strings.TrimSpace(claims.Picture),
				strings.TrimSpace(claims.Locale),
			).Scan(&userID)
			if err != nil {
				return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusInternalServerError, code: auth.GoogleAuthUpstreamAPIError, msg: "failed to persist google user"}
			}
			isNew = true
		} else {
			return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusInternalServerError, code: auth.GoogleAuthUpstreamAPIError, msg: "failed to check existing account"}
		}
	default:
		return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusInternalServerError, code: auth.GoogleAuthUpstreamAPIError, msg: "failed to lookup google user"}
	}

	var userResponse UserResponse
	if err := c.DBConn.QueryRowContext(
		ctx,
		`SELECT id, firstname, lastname, email, COALESCE(display_name, ''), COALESCE(avatar_url, ''), COALESCE(is_verified, false), COALESCE(is_active, true), COALESCE(role, 'fan'), created_at::text, updated_at::text FROM users WHERE id = $1`,
		userID,
	).Scan(
		&userResponse.ID,
		&userResponse.Firstname,
		&userResponse.Lastname,
		&userResponse.Email,
		&userResponse.DisplayName,
		&userResponse.AvatarURL,
		&userResponse.IsVerified,
		&userResponse.IsActive,
		&userResponse.Role,
		&userResponse.CreatedAt,
		&userResponse.UpdatedAt,
	); err != nil {
		return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusInternalServerError, code: auth.GoogleAuthUpstreamAPIError, msg: "failed to load authenticated user"}
	}

	token, err := auth.GenerateToken(userResponse.ID, userResponse.Email, userResponse.Role, 24*time.Hour)
	if err != nil {
		return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusInternalServerError, code: auth.GoogleAuthUpstreamAPIError, msg: "failed to generate auth token"}
	}

	return GoogleAuthResponse{
		Token: token,
		User:  userResponse,
		IsNew: isNew,
	}, nil
}

func (c *Config) handleGoogleAuth(w http.ResponseWriter, r *http.Request) {
	var req GoogleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logGoogleAuthEvent("post_decode_failed", "path", r.URL.Path, "method", r.Method)
		respondWithGoogleAuthError(w, http.StatusBadRequest, auth.GoogleAuthCodeInvalid, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Code) == "" || strings.TrimSpace(req.RedirectURI) == "" {
		logGoogleAuthEvent("post_validation_failed", "path", r.URL.Path, "missing_code", strings.TrimSpace(req.Code) == "", "missing_redirect_uri", strings.TrimSpace(req.RedirectURI) == "")
		respondWithGoogleAuthError(w, http.StatusBadRequest, auth.GoogleAuthCodeInvalid, "code and redirect_uri are required")
		return
	}

	out, procErr := c.googleAuthFromCode(r.Context(), req.Code, req.RedirectURI)
	if procErr != nil {
		logGoogleAuthEvent("post_failed", "path", r.URL.Path, "status", procErr.status, "code", procErr.code)
		respondWithGoogleAuthError(w, procErr.status, procErr.code, procErr.msg)
		return
	}
	logGoogleAuthEvent("post_success", "path", r.URL.Path, "user_id", out.User.ID, "is_new", out.IsNew)
	respondWithJSON(w, http.StatusOK, out)
}

func (c *Config) handleGoogleOAuthStart(w http.ResponseWriter, r *http.Request) {
	cb := strings.TrimSpace(c.GoogleOAuthCallbackURL)
	if cb == "" {
		logGoogleAuthEvent("start_missing_callback_url", "path", r.URL.Path)
		http.Error(w, "Google OAuth callback URL is not configured (GOOGLE_OAUTH_CALLBACK_URL)", http.StatusServiceUnavailable)
		return
	}
	if strings.TrimSpace(c.GoogleOAuthClientID) == "" {
		logGoogleAuthEvent("start_missing_client_id", "path", r.URL.Path)
		http.Error(w, "Google OAuth client ID is not configured", http.StatusServiceUnavailable)
		return
	}

	returnToApp := strings.TrimSpace(r.URL.Query().Get("return"))
	if returnToApp == "" {
		returnToApp = "fucci://auth"
	}
	if !auth.AllowedGoogleAppReturnURI(returnToApp) {
		logGoogleAuthEvent("start_invalid_return_url", "path", r.URL.Path, "return", returnToApp)
		http.Error(w, "return URL is not allowed", http.StatusBadRequest)
		return
	}

	state, err := auth.SignGoogleOAuthState(returnToApp)
	if err != nil {
		logGoogleAuthEvent("start_state_sign_failed", "path", r.URL.Path)
		http.Error(w, "failed to start OAuth", http.StatusInternalServerError)
		return
	}

	u, err := url.Parse("https://accounts.google.com/o/oauth2/v2/auth")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	q := u.Query()
	q.Set("client_id", strings.TrimSpace(c.GoogleOAuthClientID))
	q.Set("redirect_uri", cb)
	q.Set("response_type", "code")
	q.Set("scope", "openid email profile")
	q.Set("state", state)
	q.Set("access_type", "online")
	u.RawQuery = q.Encode()
	logGoogleAuthEvent("start_redirect_google", "path", r.URL.Path, "callback_url", cb, "return_url", returnToApp)
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func redirectGoogleOAuthApp(w http.ResponseWriter, r *http.Request, appReturn, code, message string) {
	u, err := url.Parse(appReturn)
	if err != nil {
		http.Error(w, "invalid app return URL", http.StatusInternalServerError)
		return
	}
	q := u.Query()
	q.Set("google_error", code)
	if message != "" {
		q.Set("google_error_description", message)
	}
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (c *Config) handleGoogleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if errMsg := q.Get("error"); errMsg != "" {
		ret := "fucci://auth"
		if ru, err := auth.ParseGoogleOAuthState(q.Get("state")); err == nil && strings.TrimSpace(ru) != "" {
			ret = ru
		}
		logGoogleAuthEvent("callback_provider_error", "path", r.URL.Path, "provider_error", errMsg, "return_url", ret)
		redirectGoogleOAuthApp(w, r, ret, errMsg, q.Get("error_description"))
		return
	}
	code := q.Get("code")
	state := q.Get("state")
	if code == "" || state == "" {
		logGoogleAuthEvent("callback_missing_code_or_state", "path", r.URL.Path, "missing_code", code == "", "missing_state", state == "")
		http.Error(w, "missing code or state", http.StatusBadRequest)
		return
	}
	appReturn, err := auth.ParseGoogleOAuthState(state)
	if err != nil {
		logGoogleAuthEvent("callback_invalid_state", "path", r.URL.Path)
		http.Error(w, "invalid or expired OAuth state", http.StatusBadRequest)
		return
	}

	cb := strings.TrimSpace(c.GoogleOAuthCallbackURL)
	if cb == "" {
		logGoogleAuthEvent("callback_missing_callback_url", "path", r.URL.Path)
		http.Error(w, "Google OAuth callback URL is not configured", http.StatusServiceUnavailable)
		return
	}

	out, procErr := c.googleAuthFromCode(r.Context(), code, cb)
	if procErr != nil {
		logGoogleAuthEvent("callback_failed", "path", r.URL.Path, "status", procErr.status, "code", procErr.code, "return_url", appReturn)
		redirectGoogleOAuthApp(w, r, appReturn, procErr.code, procErr.msg)
		return
	}

	u, err := url.Parse(appReturn)
	if err != nil {
		http.Error(w, "invalid app return URL", http.StatusInternalServerError)
		return
	}
	qq := u.Query()
	qq.Set("token", out.Token)
	if out.IsNew {
		qq.Set("is_new", "1")
	} else {
		qq.Set("is_new", "0")
	}
	u.RawQuery = qq.Encode()
	logGoogleAuthEvent("callback_success", "path", r.URL.Path, "user_id", out.User.ID, "is_new", out.IsNew, "return_url", appReturn)
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func respondWithGoogleAuthError(w http.ResponseWriter, status int, code, message string) {
	respondWithJSON(w, status, googleAuthErrorPayload{Code: code, Message: message})
}

func (c *Config) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		Firstname   *string `json:"firstname"`
		Lastname    *string `json:"lastname"`
		DisplayName *string `json:"display_name"`
		AvatarURL   *string `json:"avatar_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update only provided fields
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if req.Firstname != nil {
		updates = append(updates, fmt.Sprintf("firstname = $%d", argPos))
		args = append(args, *req.Firstname)
		argPos++
	}

	if req.Lastname != nil {
		updates = append(updates, fmt.Sprintf("lastname = $%d", argPos))
		args = append(args, *req.Lastname)
		argPos++
	}

	if req.DisplayName != nil {
		updates = append(updates, fmt.Sprintf("display_name = $%d", argPos))
		args = append(args, *req.DisplayName)
		argPos++
	}

	if req.AvatarURL != nil {
		avatarURL := strings.TrimSpace(*req.AvatarURL)
		if avatarURL == "" {
			respondWithError(w, http.StatusBadRequest, "avatar_url cannot be empty")
			return
		}
		if err := c.validateCloudinaryMediaURLForContext(avatarURL, "avatar"); err != nil {
			if errors.Is(err, ErrCloudinaryURLValidationNotConfigured) {
				respondWithError(w, http.StatusInternalServerError, err.Error())
				return
			}
			respondWithError(w, http.StatusBadRequest, fmt.Sprintf("invalid avatar_url: %v", err))
			return
		}
		updates = append(updates, fmt.Sprintf("avatar_url = $%d", argPos))
		args = append(args, avatarURL)
		argPos++
	}

	if len(updates) == 0 {
		respondWithError(w, http.StatusBadRequest, "no fields to update")
		return
	}

	// Add updated_at timestamp (direct SQL is safe for CURRENT_TIMESTAMP)
	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")

	// Build SET clause with comma-separated updates
	setClause := strings.Join(updates, ", ")

	// Add WHERE clause with parameterized user ID
	args = append(args, userID)
	whereClause := fmt.Sprintf("WHERE id = $%d", argPos)

	// Construct final query with proper separation of SET and WHERE clauses
	query := fmt.Sprintf("UPDATE users SET %s %s", setClause, whereClause)

	persist := c.profileUpdatePersistence()
	if persist == nil {
		respondWithError(w, http.StatusInternalServerError, "database not configured")
		return
	}
	if err := persist.ExecUpdate(r.Context(), query, args...); err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update profile: %s", err))
		return
	}

	userResponse, err := persist.LoadUserResponse(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to fetch updated user")
		return
	}

	respondWithJSON(w, http.StatusOK, userResponse)
}

// FollowingItem represents a single followed entity for GET /users/me/following
type FollowingItem struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	FollowableID string `json:"followable_id"`
}

// GetFollowingResponse is the response for GET /users/me/following
type GetFollowingResponse struct {
	Items []FollowingItem `json:"items"`
}

func (c *Config) handleGetFollowing(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	rows, err := c.DBConn.QueryContext(r.Context(),
		`SELECT id, followable_type, followable_id FROM user_follows WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to list following")
		return
	}
	defer rows.Close()

	var items []FollowingItem
	for rows.Next() {
		var id, followableID uuid.UUID
		var followableType string
		if err := rows.Scan(&id, &followableType, &followableID); err != nil {
			respondWithError(w, http.StatusInternalServerError, "failed to scan following")
			return
		}
		items = append(items, FollowingItem{ID: id.String(), Type: followableType, FollowableID: followableID.String()})
	}
	if err := rows.Err(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to list following")
		return
	}
	if items == nil {
		items = []FollowingItem{}
	}
	respondWithJSON(w, http.StatusOK, GetFollowingResponse{Items: items})
}

func (c *Config) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int32)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	user, err := c.DB.GetUser(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	// Get additional fields
	var displayName, avatarURL, role, createdAt, updatedAt string
	var isVerified, isActive bool
	err = c.DBConn.QueryRow(
		"SELECT COALESCE(display_name, ''), COALESCE(avatar_url, ''), is_verified, is_active, role, created_at, updated_at FROM users WHERE id = $1",
		userID,
	).Scan(&displayName, &avatarURL, &isVerified, &isActive, &role, &createdAt, &updatedAt)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to fetch user details")
		return
	}

	userResponse := UserResponse{
		ID:          user.ID,
		Firstname:   user.Firstname,
		Lastname:    user.Lastname,
		Email:       user.Email,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		IsVerified:  isVerified,
		IsActive:    isActive,
		Role:        role,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	respondWithJSON(w, http.StatusOK, userResponse)
}
