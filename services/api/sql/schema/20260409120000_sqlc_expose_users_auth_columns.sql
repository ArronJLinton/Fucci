-- +goose Up
-- Idempotent adds for real databases. Plain ALTERs so sqlc sees these columns
-- (016_add_auth_to_users.sql wraps similar DDL in DO blocks which sqlc skips).

ALTER TABLE users ADD COLUMN IF NOT EXISTS is_verified BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT DEFAULT 'fan';

-- +goose Down
-- Intentionally empty: columns are owned by broader auth migrations.
