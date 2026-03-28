package api

import (
	"context"
	"database/sql"

	"github.com/ArronJLinton/fucci-api/internal/database"
)

// ProfileUpdatePersistence implements PUT /users/profile persistence (dynamic UPDATE + LoadUserResponse).
// Production uses sqlProfileUpdateStore; tests inject fakes (e.g. FR-007 avatar URL persistence).
type ProfileUpdatePersistence interface {
	ExecUpdate(ctx context.Context, query string, args ...interface{}) error
	LoadUserResponse(ctx context.Context, userID int32) (UserResponse, error)
}

type sqlProfileUpdateStore struct {
	db *sql.DB
	q  *database.Queries
}

func (s *sqlProfileUpdateStore) ExecUpdate(ctx context.Context, query string, args ...interface{}) error {
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *sqlProfileUpdateStore) LoadUserResponse(ctx context.Context, userID int32) (UserResponse, error) {
	user, err := s.q.GetUser(ctx, userID)
	if err != nil {
		return UserResponse{}, err
	}
	var displayName, avatarURL, role, createdAt, updatedAt string
	var isVerified, isActive bool
	err = s.db.QueryRowContext(ctx,
		"SELECT COALESCE(display_name, ''), COALESCE(avatar_url, ''), is_verified, is_active, role, created_at, updated_at FROM users WHERE id = $1",
		userID,
	).Scan(&displayName, &avatarURL, &isVerified, &isActive, &role, &createdAt, &updatedAt)
	if err != nil {
		return UserResponse{}, err
	}
	return UserResponse{
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
	}, nil
}

func (c *Config) profileUpdatePersistence() ProfileUpdatePersistence {
	if c.ProfileUpdateDB != nil {
		return c.ProfileUpdateDB
	}
	if c.DBConn == nil || c.DB == nil {
		return nil
	}
	return &sqlProfileUpdateStore{db: c.DBConn, q: c.DB}
}
