-- +goose Up
-- Retire legacy UUID-based profile surface; canonical profile table is player_profile.
DROP TABLE IF EXISTS verifications;
DROP TABLE IF EXISTS player_profiles;

-- +goose Down
-- Recreate legacy tables for rollback.
CREATE TABLE IF NOT EXISTS player_profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  team_id UUID REFERENCES teams(id) ON DELETE SET NULL,
  position VARCHAR(50) NOT NULL,
  age INTEGER NOT NULL CHECK (age >= 16 AND age <= 50),
  country VARCHAR(50) NOT NULL,
  height_cm INTEGER NOT NULL,
  pace INTEGER NOT NULL DEFAULT 50 CHECK (pace >= 0 AND pace <= 100),
  shooting INTEGER NOT NULL DEFAULT 50 CHECK (shooting >= 0 AND shooting <= 100),
  passing INTEGER NOT NULL DEFAULT 50 CHECK (passing >= 0 AND passing <= 100),
  stamina INTEGER NOT NULL DEFAULT 50 CHECK (stamina >= 0 AND stamina <= 100),
  dribbling INTEGER NOT NULL DEFAULT 50 CHECK (dribbling >= 0 AND dribbling <= 100),
  defending INTEGER NOT NULL DEFAULT 50 CHECK (defending >= 0 AND defending <= 100),
  physical INTEGER NOT NULL DEFAULT 50 CHECK (physical >= 0 AND physical <= 100),
  is_verified BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id)
);

CREATE TABLE IF NOT EXISTS verifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  player_profile_id UUID NOT NULL REFERENCES player_profiles(id) ON DELETE CASCADE,
  verifier_user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(player_profile_id, verifier_user_id)
);

CREATE INDEX IF NOT EXISTS idx_player_profiles_user_id ON player_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_player_profiles_team_id ON player_profiles(team_id);
CREATE INDEX IF NOT EXISTS idx_player_profiles_is_verified ON player_profiles(is_verified);
CREATE INDEX IF NOT EXISTS idx_verifications_player_profile_id ON verifications(player_profile_id);
CREATE INDEX IF NOT EXISTS idx_verifications_verifier_user_id ON verifications(verifier_user_id);
