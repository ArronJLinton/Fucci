-- name: CreateUser :one
INSERT INTO users (firstname, lastname, email, is_admin)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByEmailLower :one
-- Caller must pass email already lowercased to match stored rows and use a plain index on email.
SELECT * FROM users WHERE email = $1 LIMIT 1;

-- name: GetUserByGoogleID :one
SELECT * FROM users WHERE google_id = sqlc.arg(google_id)::varchar(255);

-- name: GetUserByAppleID :one
SELECT * FROM users WHERE apple_id = sqlc.arg(apple_id)::varchar(255);

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;

-- name: CreateGoogleUser :one
INSERT INTO users (firstname, lastname, email, google_id, auth_provider, avatar_url, locale, is_admin, is_active, is_verified, last_login_at)
VALUES ($1, $2, $3, $4, 'google', $5, $6, false, true, true, CURRENT_TIMESTAMP)
RETURNING *;

-- name: CreateAppleUser :one
INSERT INTO users (firstname, lastname, email, apple_id, auth_provider, is_admin, is_active, is_verified, last_login_at, apple_refresh_token)
VALUES ($1, $2, $3, $4, 'apple', false, true, true, CURRENT_TIMESTAMP, sqlc.narg(apple_refresh_token))
RETURNING *;

-- name: LinkGoogleToExistingUser :one
UPDATE users
SET google_id = COALESCE(NULLIF(google_id::text, ''), sqlc.arg(new_google_id)::text)::varchar(255),
    auth_provider = COALESCE(auth_provider, 'google'),
    last_login_at = CURRENT_TIMESTAMP,
    avatar_url = CASE WHEN sqlc.arg(avatar_url)::text <> '' THEN sqlc.arg(avatar_url) ELSE avatar_url END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: LinkAppleToExistingUser :one
UPDATE users
SET apple_id = COALESCE(NULLIF(apple_id::text, ''), sqlc.arg(new_apple_id)::text)::varchar(255),
    auth_provider = COALESCE(auth_provider, 'apple'),
    last_login_at = CURRENT_TIMESTAMP,
    apple_refresh_token = COALESCE(sqlc.narg(apple_refresh_token), apple_refresh_token),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateGoogleLoginFields :one
UPDATE users
SET last_login_at = CURRENT_TIMESTAMP,
    avatar_url = CASE WHEN sqlc.arg(avatar_url)::text <> '' THEN sqlc.arg(avatar_url) ELSE avatar_url END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateAppleLoginFields :one
UPDATE users
SET last_login_at = CURRENT_TIMESTAMP,
    apple_refresh_token = COALESCE(sqlc.narg(apple_refresh_token), apple_refresh_token),
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
