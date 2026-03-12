package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/google/uuid"
)

// LoginRequest represents the login request payload (identifier = email or username)
type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// LoginResponse represents the login response payload
type LoginResponse struct {
	Token string `json:"token"`
	User  UserResponse
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
}

func (c *Config) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Identifier) == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "identifier and password are required")
		return
	}

	// Get user by email or username (identifier)
	var user struct {
		ID        int32
		Firstname string
		Lastname  string
		Email     string
	}
	err := c.DBConn.QueryRowContext(r.Context(),
		`SELECT id, firstname, lastname, email FROM users WHERE (email = $1 OR (username IS NOT NULL AND username = $1)) AND is_active = true LIMIT 1`,
		req.Identifier,
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
	var displayName, avatarURL, createdAt string
	var isVerified, isActive bool
	c.DBConn.QueryRow(
		"SELECT display_name, avatar_url, is_verified, is_active, created_at FROM users WHERE id = $1",
		user.ID,
	).Scan(&displayName, &avatarURL, &isVerified, &isActive, &createdAt)

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
	}

	response := LoginResponse{
		Token: token,
		User:  userResponse,
	}

	respondWithJSON(w, http.StatusOK, response)
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
		updates = append(updates, fmt.Sprintf("avatar_url = $%d", argPos))
		args = append(args, *req.AvatarURL)
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

	_, err := c.DBConn.Exec(query, args...)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("failed to update profile: %s", err))
		return
	}

	// Fetch updated user
	user, err := c.DB.GetUser(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to fetch updated user")
		return
	}

	// Get additional fields
	var displayName, avatarURL, role, createdAt string
	var isVerified, isActive bool
	c.DBConn.QueryRow(
		"SELECT display_name, avatar_url, is_verified, is_active, role, created_at FROM users WHERE id = $1",
		userID,
	).Scan(&displayName, &avatarURL, &isVerified, &isActive, &role, &createdAt)

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
	}

	respondWithJSON(w, http.StatusOK, userResponse)
}

// FollowingItem represents a single followed entity for GET /users/me/following
type FollowingItem struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	FollowableID  string `json:"followable_id"`
	Name          string `json:"name,omitempty"`
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
	var displayName, avatarURL, role, createdAt string
	var isVerified, isActive bool
	err = c.DBConn.QueryRow(
		"SELECT display_name, avatar_url, is_verified, is_active, role, created_at FROM users WHERE id = $1",
		userID,
	).Scan(&displayName, &avatarURL, &isVerified, &isActive, &role, &createdAt)
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
	}

	respondWithJSON(w, http.StatusOK, userResponse)
}
