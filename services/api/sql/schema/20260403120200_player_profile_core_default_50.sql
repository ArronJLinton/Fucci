-- +goose Up
-- Set server-side defaults to 50 for new rows (aligned with API/mobile).
-- No-op if columns already default to 50 (e.g. from current 20260403120000_player_profile_core_attributes.sql).

ALTER TABLE player_profile
  ALTER COLUMN speed SET DEFAULT 50,
  ALTER COLUMN shooting SET DEFAULT 50,
  ALTER COLUMN passing SET DEFAULT 50,
  ALTER COLUMN dribbling SET DEFAULT 50,
  ALTER COLUMN defending SET DEFAULT 50,
  ALTER COLUMN physical SET DEFAULT 50,
  ALTER COLUMN stamina SET DEFAULT 50;

-- +goose Down
-- Restore defaults from the original core-attributes migration (72) before this adjustment.

ALTER TABLE player_profile
  ALTER COLUMN speed SET DEFAULT 72,
  ALTER COLUMN shooting SET DEFAULT 72,
  ALTER COLUMN passing SET DEFAULT 72,
  ALTER COLUMN dribbling SET DEFAULT 72,
  ALTER COLUMN defending SET DEFAULT 72,
  ALTER COLUMN physical SET DEFAULT 72,
  ALTER COLUMN stamina SET DEFAULT 72;
