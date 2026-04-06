-- +goose Up
-- Core attributes (007 player profile): 40–99, neutral default 50 (aligned with API/mobile).

ALTER TABLE player_profile
  ADD COLUMN IF NOT EXISTS speed INTEGER NOT NULL DEFAULT 50
    CONSTRAINT player_profile_speed_check CHECK (speed >= 40 AND speed <= 99),
  ADD COLUMN IF NOT EXISTS shooting INTEGER NOT NULL DEFAULT 50
    CONSTRAINT player_profile_shooting_check CHECK (shooting >= 40 AND shooting <= 99),
  ADD COLUMN IF NOT EXISTS passing INTEGER NOT NULL DEFAULT 50
    CONSTRAINT player_profile_passing_check CHECK (passing >= 40 AND passing <= 99),
  ADD COLUMN IF NOT EXISTS dribbling INTEGER NOT NULL DEFAULT 50
    CONSTRAINT player_profile_dribbling_check CHECK (dribbling >= 40 AND dribbling <= 99),
  ADD COLUMN IF NOT EXISTS defending INTEGER NOT NULL DEFAULT 50
    CONSTRAINT player_profile_defending_check CHECK (defending >= 40 AND defending <= 99),
  ADD COLUMN IF NOT EXISTS physical INTEGER NOT NULL DEFAULT 50
    CONSTRAINT player_profile_physical_check CHECK (physical >= 40 AND physical <= 99),
  ADD COLUMN IF NOT EXISTS stamina INTEGER NOT NULL DEFAULT 50
    CONSTRAINT player_profile_stamina_check CHECK (stamina >= 40 AND stamina <= 99);

-- +goose Down
ALTER TABLE player_profile
  DROP COLUMN IF EXISTS speed,
  DROP COLUMN IF EXISTS shooting,
  DROP COLUMN IF EXISTS passing,
  DROP COLUMN IF EXISTS dribbling,
  DROP COLUMN IF EXISTS defending,
  DROP COLUMN IF EXISTS physical,
  DROP COLUMN IF EXISTS stamina;
