-- name: CreateUser :one
INSERT INTO users (firstname, lastname, email, is_admin)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByGoogleID :one
SELECT * FROM users WHERE google_id = sqlc.arg(google_id)::text;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;

-- name: CreateGoogleUser :one
INSERT INTO users (firstname, lastname, email, google_id, auth_provider, avatar_url, locale, is_admin)
VALUES ($1, $2, $3, $4, 'google', $5, $6, false)
RETURNING *;

-- name: UpdateGoogleLoginFields :one
UPDATE users
SET last_login_at = CURRENT_TIMESTAMP,
    avatar_url = CASE WHEN sqlc.arg(avatar_url)::text <> '' THEN sqlc.arg(avatar_url) ELSE avatar_url END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateUser :one
UPDATE users 
SET firstname = $2, lastname = $3, email = $4, is_admin = $5, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;