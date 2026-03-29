-- +goose Up
ALTER TABLE debates
ADD COLUMN IF NOT EXISTS match_info JSONB;

-- +goose Down
ALTER TABLE debates
DROP COLUMN IF EXISTS match_info;
