-- +goose Up

ALTER TABLE comments ADD COLUMN IF NOT EXISTS seeded BOOLEAN NOT NULL DEFAULT false;

-- +goose Down

ALTER TABLE comments DROP COLUMN IF EXISTS seeded;
