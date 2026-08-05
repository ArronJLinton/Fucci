-- +goose Up

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS apple_id VARCHAR(255);

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS apple_refresh_token TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_apple_id_unique
    ON users (apple_id)
    WHERE apple_id IS NOT NULL AND apple_id <> '';

-- +goose Down

DROP INDEX IF EXISTS idx_users_apple_id_unique;

ALTER TABLE users
    DROP COLUMN IF EXISTS apple_refresh_token;

ALTER TABLE users
    DROP COLUMN IF EXISTS apple_id;
