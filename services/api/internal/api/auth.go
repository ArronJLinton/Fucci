package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/ArronJLinton/fucci-api/internal/database"
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

type GoogleOAuthExchangeRequest struct {
	Code string `json:"code"`
}

type googleOAuthExchangeSession struct {
	Response  GoogleAuthResponse
	ExpiresAt time.Time
}

const googleOAuthExchangeTTL = 2 * time.Minute

// googleOAuthExchangeCacheKeyPrefix namespaces one-time exchange payloads in the shared cache (Redis).
const googleOAuthExchangeCacheKeyPrefix = "oauth:google:exchange:v1:"

func googleOAuthExchangeCacheKey(code string) string {
	return googleOAuthExchangeCacheKeyPrefix + code
}

var (
	googleOAuthExchangeMu       sync.Mutex
	googleOAuthExchangeSessions = map[string]googleOAuthExchangeSession{}
)

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

func (c *Config) dbQueries() *database.Queries {
	if c.DB != nil {
		return c.DB
	}
	return database.New(c.DBConn)
}

func userResponseFromDBUser(u database.Users) UserResponse {
	displayName := ""
	if u.DisplayName.Valid {
		displayName = u.DisplayName.String
	}
	avatarURL := ""
	if u.AvatarUrl.Valid {
		avatarURL = u.AvatarUrl.String
	}
	verified := false
	if u.IsVerified.Valid {
		verified = u.IsVerified.Bool
	}
	active := true
	if u.IsActive.Valid {
		active = u.IsActive.Bool
	}
	role := "fan"
	if u.Role.Valid && strings.TrimSpace(u.Role.String) != "" {
		role = strings.TrimSpace(u.Role.String)
	}
	return UserResponse{
		ID:          u.ID,
		Firstname:   u.Firstname,
		Lastname:    u.Lastname,
		Email:       u.Email,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		IsVerified:  verified,
		IsActive:    active,
		Role:        role,
		CreatedAt:   u.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:   u.UpdatedAt.Format(time.RFC3339Nano),
	}
}

// googleAuthFromCode exchanges an OAuth code and returns a session (used by POST /auth/google and GET /auth/google/callback).
func (c *Config) googleAuthFromCode(ctx context.Context, code, redirectURI string) (GoogleAuthResponse, *googleAuthProcError) {
	if strings.TrimSpace(c.GoogleOAuthClientID) == "" || strings.TrimSpace(c.GoogleOAuthClientSecret) == "" {
		logGoogleAuthEvent(
			"exchange_not_configured",
			"missing_client_id", strings.TrimSpace(c.GoogleOAuthClientID) == "",
			"missing_client_secret", strings.TrimSpace(c.GoogleOAuthClientSecret) == "",
		)
		return GoogleAuthResponse{}, &googleAuthProcError{
			status: http.StatusServiceUnavailable,
			code:   auth.GoogleAuthNotConfigured,
			msg:    "Google OAuth is not configured on the server",
		}
	}
	verifier := c.googleVerifier()
	idToken, err := verifier.ExchangeCodeForIDToken(ctx, code, redirectURI)
	if err != nil {
		if errors.Is(err, auth.ErrGoogleInvalidRedirectURI) {
			return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusBadRequest, code: auth.GoogleAuthInvalidRedirectURI, msg: "redirect_uri is not allowed"}
		}
		if errors.Is(err, auth.ErrGoogleInvalidCode) {
			return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusBadRequest, code: auth.GoogleAuthCodeInvalid, msg: "malformed or expired auth code"}
		}
		return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusInternalServerError, code: auth.GoogleAuthUpstreamAPIError, msg: "google token exchange failed"}
	}

	claims, err := verifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusUnauthorized, code: auth.GoogleAuthTokenVerifyFailed, msg: "unable to verify Google ID token"}
	}
	if !claims.EmailVerified {
		return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusBadRequest, code: auth.GoogleAuthEmailNotVerified, msg: "Google email is not verified"}
	}

	subject := strings.TrimSpace(claims.Subject)
	emailTrim := strings.TrimSpace(claims.Email)
	if subject == "" || emailTrim == "" {
		return GoogleAuthResponse{}, &googleAuthProcError{
			status: http.StatusUnauthorized,
			code:   auth.GoogleAuthTokenVerifyFailed,
			msg:    "Google ID token is missing required subject or email claims",
		}
	}
	if _, err := mail.ParseAddress(emailTrim); err != nil {
		return GoogleAuthResponse{}, &googleAuthProcError{
			status: http.StatusUnauthorized,
			code:   auth.GoogleAuthTokenVerifyFailed,
			msg:    "Google ID token has an invalid email claim",
		}
	}
	email := strings.ToLower(emailTrim)

	q := c.dbQueries()

	var userID int32
	var isNew bool

	existingByGoogle, err := q.GetUserByGoogleID(ctx, subject)
	switch {
	case err == nil:
		if existingByGoogle.IsActive.Valid && !existingByGoogle.IsActive.Bool {
			return GoogleAuthResponse{}, &googleAuthProcError{
				status: http.StatusForbidden,
				code:   auth.GoogleAuthAccountInactive,
				msg:    "This account has been deactivated",
			}
		}
		userID = existingByGoogle.ID
		if _, err := q.UpdateGoogleLoginFields(ctx, database.UpdateGoogleLoginFieldsParams{
			AvatarUrl: strings.TrimSpace(claims.Picture),
			ID:        userID,
		}); err != nil {
			return GoogleAuthResponse{}, &googleAuthProcError{
				status: http.StatusInternalServerError,
				code:   auth.GoogleAuthUpstreamAPIError,
				msg:    "failed to update google login fields",
			}
		}
	case errors.Is(err, sql.ErrNoRows):
		byEmail, qerr := q.GetUserByEmailLower(ctx, email)
		if qerr == nil {
			if byEmail.IsActive.Valid && !byEmail.IsActive.Bool {
				return GoogleAuthResponse{}, &googleAuthProcError{
					status: http.StatusForbidden,
					code:   auth.GoogleAuthAccountInactive,
					msg:    "This account has been deactivated",
				}
			}
			existingGoogleIDVal := strings.TrimSpace(byEmail.GoogleID.String)
			if byEmail.GoogleID.Valid && existingGoogleIDVal != "" && existingGoogleIDVal != subject {
				return GoogleAuthResponse{}, &googleAuthProcError{
					status: http.StatusConflict,
					code:   auth.GoogleAuthAccountExistsEmail,
					msg:    "Email already linked to another Google account",
				}
			}
			prov := strings.TrimSpace(byEmail.AuthProvider)
			if strings.EqualFold(prov, "email") {
				return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusConflict, code: auth.GoogleAuthAccountExistsEmail, msg: "Email already registered via password"}
			}
			if !strings.EqualFold(prov, "google") {
				return GoogleAuthResponse{}, &googleAuthProcError{
					status: http.StatusConflict,
					code:   auth.GoogleAuthAccountExistsEmail,
					msg:    "Email already linked to another account provider",
				}
			}
			if _, err := q.LinkGoogleToExistingUser(ctx, database.LinkGoogleToExistingUserParams{
				ID:          byEmail.ID,
				NewGoogleID: subject,
				AvatarUrl:   strings.TrimSpace(claims.Picture),
			}); err != nil {
				return GoogleAuthResponse{}, &googleAuthProcError{
					status: http.StatusInternalServerError,
					code:   auth.GoogleAuthUpstreamAPIError,
					msg:    "failed to update existing account with google login fields",
				}
			}
			userID = byEmail.ID
		} else if errors.Is(qerr, sql.ErrNoRows) {
			pic := strings.TrimSpace(claims.Picture)
			loc := strings.TrimSpace(claims.Locale)
			created, err := q.CreateGoogleUser(ctx, database.CreateGoogleUserParams{
				Firstname: strings.TrimSpace(claims.GivenName),
				Lastname:  strings.TrimSpace(claims.FamilyName),
				Email:     email,
				GoogleID:  sql.NullString{String: subject, Valid: true},
				AvatarUrl: sql.NullString{String: pic, Valid: pic != ""},
				Locale:    sql.NullString{String: loc, Valid: loc != ""},
			})
			if err != nil {
				return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusInternalServerError, code: auth.GoogleAuthUpstreamAPIError, msg: "failed to persist google user"}
			}
			userID = created.ID
			isNew = true
		} else {
			return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusInternalServerError, code: auth.GoogleAuthUpstreamAPIError, msg: "failed to check existing account"}
		}
	default:
		return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusInternalServerError, code: auth.GoogleAuthUpstreamAPIError, msg: "failed to lookup google user"}
	}

	u, err := q.GetUser(ctx, userID)
	if err != nil {
		return GoogleAuthResponse{}, &googleAuthProcError{status: http.StatusInternalServerError, code: auth.GoogleAuthUpstreamAPIError, msg: "failed to load authenticated user"}
	}
	userResponse := userResponseFromDBUser(u)
	if !userResponse.IsActive {
		return GoogleAuthResponse{}, &googleAuthProcError{
			status: http.StatusForbidden,
			code:   auth.GoogleAuthAccountInactive,
			msg:    "This account has been deactivated",
		}
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
	if strings.TrimSpace(c.GoogleOAuthClientID) == "" || strings.TrimSpace(c.GoogleOAuthClientSecret) == "" {
		logGoogleAuthEvent(
			"post_not_configured",
			"path", r.URL.Path,
			"missing_client_id", strings.TrimSpace(c.GoogleOAuthClientID) == "",
			"missing_client_secret", strings.TrimSpace(c.GoogleOAuthClientSecret) == "",
		)
		respondWithGoogleAuthError(
			w,
			http.StatusServiceUnavailable,
			auth.GoogleAuthNotConfigured,
			"Google OAuth is not configured on the server",
		)
		return
	}

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
	if !auth.AllowedGoogleAppReturnURI(returnToApp, c.GoogleOAuthAllowDevReturnURLs) {
		logGoogleAuthEvent("start_invalid_return_url", "path", r.URL.Path, "return", returnToApp)
		http.Error(w, "return URL is not allowed", http.StatusBadRequest)
		return
	}

	state, err := auth.SignGoogleOAuthState(returnToApp)
	if err != nil {
		if errors.Is(err, auth.ErrJWTNotInitialized) {
			logGoogleAuthEvent("start_jwt_not_configured", "path", r.URL.Path)
			http.Error(
				w,
				"JWT secret is not configured (JWT_SECRET) — required to sign Google OAuth state",
				http.StatusServiceUnavailable,
			)
			return
		}
		logGoogleAuthEvent("start_state_sign_failed", "path", r.URL.Path, "err", err.Error())
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

func redirectGoogleOAuthAppCancel(w http.ResponseWriter, r *http.Request, appReturn string) {
	u, err := url.Parse(appReturn)
	if err != nil {
		http.Error(w, "invalid app return URL", http.StatusInternalServerError)
		return
	}
	q := u.Query()
	q.Set("cancel", "1")
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func isGoogleProviderCancellation(errCode string) bool {
	switch strings.ToLower(strings.TrimSpace(errCode)) {
	case "access_denied", "user_cancelled", "user_canceled":
		return true
	default:
		return false
	}
}

func (c *Config) issueGoogleOAuthExchangeCode(ctx context.Context, response GoogleAuthResponse) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	code := base64.RawURLEncoding.EncodeToString(raw)

	if c != nil && c.Cache != nil {
		key := googleOAuthExchangeCacheKey(code)
		if err := c.Cache.Set(ctx, key, response, googleOAuthExchangeTTL); err != nil {
			return "", err
		}
		return code, nil
	}

	// Process-local fallback when no shared cache (single-instance / tests).
	now := time.Now()
	googleOAuthExchangeMu.Lock()
	for k, v := range googleOAuthExchangeSessions {
		if now.After(v.ExpiresAt) {
			delete(googleOAuthExchangeSessions, k)
		}
	}
	googleOAuthExchangeSessions[code] = googleOAuthExchangeSession{
		Response:  response,
		ExpiresAt: now.Add(googleOAuthExchangeTTL),
	}
	googleOAuthExchangeMu.Unlock()
	return code, nil
}

func (c *Config) consumeGoogleOAuthExchangeCode(ctx context.Context, code string) (GoogleAuthResponse, bool) {
	if strings.TrimSpace(code) == "" {
		return GoogleAuthResponse{}, false
	}

	if c != nil && c.Cache != nil {
		key := googleOAuthExchangeCacheKey(code)
		var out GoogleAuthResponse
		found, err := c.Cache.GetDel(ctx, key, &out)
		if err != nil || !found {
			return GoogleAuthResponse{}, false
		}
		return out, true
	}

	now := time.Now()
	googleOAuthExchangeMu.Lock()
	defer googleOAuthExchangeMu.Unlock()

	for k, v := range googleOAuthExchangeSessions {
		if now.After(v.ExpiresAt) {
			delete(googleOAuthExchangeSessions, k)
		}
	}

	session, ok := googleOAuthExchangeSessions[code]
	if !ok || now.After(session.ExpiresAt) {
		delete(googleOAuthExchangeSessions, code)
		return GoogleAuthResponse{}, false
	}
	delete(googleOAuthExchangeSessions, code)
	return session.Response, true
}

func (c *Config) handleGoogleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if errMsg := q.Get("error"); errMsg != "" {
		ret := "fucci://auth"
		if ru, err := auth.ParseGoogleOAuthState(q.Get("state")); err == nil && strings.TrimSpace(ru) != "" {
			ret = ru
		}
		if isGoogleProviderCancellation(errMsg) {
			logGoogleAuthEvent("callback_provider_cancelled", "path", r.URL.Path, "provider_error", errMsg, "return_url", ret)
			redirectGoogleOAuthAppCancel(w, r, ret)
			return
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
	oneTimeCode, err := c.issueGoogleOAuthExchangeCode(r.Context(), out)
	if err != nil {
		logGoogleAuthEvent("callback_issue_exchange_code_failed", "path", r.URL.Path, "return_url", appReturn)
		redirectGoogleOAuthApp(w, r, appReturn, auth.GoogleAuthUpstreamAPIError, "failed to create OAuth exchange code")
		return
	}

	u, err := url.Parse(appReturn)
	if err != nil {
		http.Error(w, "invalid app return URL", http.StatusInternalServerError)
		return
	}
	qq := u.Query()
	qq.Set("code", oneTimeCode)
	if out.IsNew {
		qq.Set("is_new", "1")
	} else {
		qq.Set("is_new", "0")
	}
	u.RawQuery = qq.Encode()
	logGoogleAuthEvent("callback_success", "path", r.URL.Path, "user_id", out.User.ID, "is_new", out.IsNew, "return_url", appReturn)
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (c *Config) handleGoogleOAuthExchange(w http.ResponseWriter, r *http.Request) {
	var req GoogleOAuthExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithGoogleAuthError(w, http.StatusBadRequest, auth.GoogleAuthCodeInvalid, "invalid request body")
		return
	}
	out, ok := c.consumeGoogleOAuthExchangeCode(r.Context(), req.Code)
	if !ok {
		respondWithGoogleAuthError(w, http.StatusBadRequest, auth.GoogleAuthCodeInvalid, "invalid or expired oauth exchange code")
		return
	}
	respondWithJSON(w, http.StatusOK, out)
}

func respondWithGoogleAuthError(w http.ResponseWriter, status int, code, message string) {
	respondWithErrorCode(w, status, message, code)
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
