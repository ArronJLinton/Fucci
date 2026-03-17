-- +goose Up
-- Expose display_name and avatar_url for sqlc schema (columns may already exist from 016).
ALTER TABLE users ADD COLUMN IF NOT EXISTS display_name VARCHAR(100);
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;

-- +goose Down
-- Leave columns in place; 016 may have added them.
