-- +goose Up

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_google_id_unique_not_null
ON users (google_id)
WHERE google_id IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_users_google_id_unique_not_null;

