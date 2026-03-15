package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ArronJLinton/fucci-api/internal/auth"
	"github.com/lib/pq"
)

type CreateUserRequest struct {
	Firstname   string `json:"firstname"`
	Lastname    string `json:"lastname"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
}

type CreateUserResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"`
}

func (config *Config) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, fmt.Sprintf("Error parsing JSON: %s", err))
		return
	}

	// Validate password
	if err := auth.ValidatePasswordStrength(req.Password); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	// Set display name to firstname + lastname if not provided
	displayName := req.DisplayName
	if displayName == "" {
		displayName = fmt.Sprintf("%s %s", req.Firstname, req.Lastname)
	}

	// Insert user with password hash and optional avatar_url; RETURNING all fields needed for response
	query := `INSERT INTO users (firstname, lastname, email, password_hash, display_name, role, avatar_url)
			  VALUES ($1, $2, $3, $4, $5, 'fan', NULLIF($6, ''))
			  RETURNING id, firstname, lastname, email, created_at, updated_at, role,
			  COALESCE(display_name, ''), COALESCE(avatar_url, ''), is_verified, is_active`

	var id int32
	var firstname, lastname, email, role, displayNameOut, avatarURL string
	var createdAt, updatedAt time.Time
	var isVerified, isActive bool
	err = config.DBConn.QueryRow(query, req.Firstname, req.Lastname, req.Email, passwordHash, displayName, req.AvatarURL).Scan(
		&id, &firstname, &lastname, &email, &createdAt, &updatedAt, &role,
		&displayNameOut, &avatarURL, &isVerified, &isActive,
	)
	if err != nil {
		log.Printf("create user: db error: %v", err)
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			respondWithError(w, http.StatusConflict, "email already in use")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "could not create account")
		return
	}

	token, err := auth.GenerateToken(id, email, role, 24*time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	userResponse := UserResponse{
		ID:          id,
		Firstname:   firstname,
		Lastname:    lastname,
		Email:       email,
		DisplayName: displayNameOut,
		AvatarURL:   avatarURL,
		IsVerified:  isVerified,
		IsActive:    isActive,
		Role:        role,
		CreatedAt:   createdAt.Format(time.RFC3339),
		UpdatedAt:   updatedAt.Format(time.RFC3339),
	}

	respondWithJSON(w, http.StatusCreated, CreateUserResponse{User: userResponse, Token: token})
}

func (config *Config) handleListAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := config.DB.ListUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Error listing users: %s", err))
		return
	}
	respondWithJSON(w, http.StatusOK, users)
}
