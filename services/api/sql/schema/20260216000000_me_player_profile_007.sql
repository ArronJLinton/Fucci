-- +goose Up
-- 007 Player Profile (signed-in user): one profile per user, traits, career teams.
-- Separate from legacy player_profiles to avoid breaking existing usage.

CREATE TABLE player_profile (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  age INTEGER CHECK (age IS NULL OR (age >= 13 AND age <= 60)),
  country_code VARCHAR(2) NOT NULL,
  club_name TEXT,
  is_free_agent BOOLEAN NOT NULL DEFAULT FALSE,
  position VARCHAR(10) NOT NULL CHECK (position IN ('GK', 'DEF', 'MID', 'FWD')),
  photo_url TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE player_profile_trait (
  id SERIAL PRIMARY KEY,
  player_profile_id INTEGER NOT NULL REFERENCES player_profile(id) ON DELETE CASCADE,
  trait_code VARCHAR(50) NOT NULL,
  UNIQUE(player_profile_id, trait_code)
);

CREATE TABLE player_profile_career_team (
  id SERIAL PRIMARY KEY,
  player_profile_id INTEGER NOT NULL REFERENCES player_profile(id) ON DELETE CASCADE,
  team_name TEXT NOT NULL,
  start_year INTEGER NOT NULL CHECK (start_year >= 1950 AND start_year <= 2100),
  end_year INTEGER CHECK (end_year IS NULL OR (end_year >= 1950 AND end_year <= 2100)),
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_player_profile_trait_profile ON player_profile_trait(player_profile_id);
CREATE INDEX idx_player_profile_career_team_profile ON player_profile_career_team(player_profile_id);

-- +goose Down
DROP TABLE IF EXISTS player_profile_career_team;
DROP TABLE IF EXISTS player_profile_trait;
DROP TABLE IF EXISTS player_profile;
