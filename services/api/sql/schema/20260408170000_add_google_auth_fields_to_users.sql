-- +goose Up

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'auth_provider_type') THEN
        CREATE TYPE auth_provider_type AS ENUM ('email', 'google', 'apple');
    END IF;
END
$$;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS google_id VARCHAR(255),
    ADD COLUMN IF NOT EXISTS auth_provider auth_provider_type,
    ADD COLUMN IF NOT EXISTS locale VARCHAR(20);

-- avatar_url and last_login_at may already exist from previous auth migration.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS avatar_url TEXT,
    ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

-- Normalize timestamp type for login tracking.
ALTER TABLE users
    ALTER COLUMN last_login_at TYPE TIMESTAMPTZ USING last_login_at::timestamptz;

UPDATE users
SET auth_provider = 'email'
WHERE auth_provider IS NULL;

ALTER TABLE users
    ALTER COLUMN auth_provider SET DEFAULT 'email',
    ALTER COLUMN auth_provider SET NOT NULL;

-- +goose Down

ALTER TABLE users
    ALTER COLUMN auth_provider DROP NOT NULL,
    ALTER COLUMN auth_provider DROP DEFAULT;

ALTER TABLE users
    DROP COLUMN IF EXISTS locale,
    DROP COLUMN IF EXISTS auth_provider,
    DROP COLUMN IF EXISTS google_id;

-- Intentionally do not DROP TYPE auth_provider_type here. Up creates the enum only if missing,
-- so the type may pre-exist or be referenced by other objects; dropping it can fail rollbacks
-- or remove a shared type. Remove the type manually if needed after confirming no pg_depend users.
