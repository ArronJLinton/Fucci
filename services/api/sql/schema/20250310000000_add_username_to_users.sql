-- +goose Up

-- Add username column for "email or username" login (005 user registration spec)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'username') THEN
        ALTER TABLE users ADD COLUMN username VARCHAR(100) UNIQUE;
    END IF;
END
$$;

-- +goose Down

DROP INDEX IF EXISTS idx_users_username;
ALTER TABLE users DROP COLUMN IF EXISTS username;
