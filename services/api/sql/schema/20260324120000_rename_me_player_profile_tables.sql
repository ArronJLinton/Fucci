-- +goose Up
-- 007: rename me_player_profile* tables/columns to player_profile* (sqlc + API naming).
DO $$
BEGIN
  IF to_regclass('public.me_player_profile') IS NOT NULL
     AND to_regclass('public.player_profile') IS NULL THEN
    ALTER TABLE me_player_profile RENAME TO player_profile;
  END IF;

  IF to_regclass('public.me_player_profile_trait') IS NOT NULL
     AND to_regclass('public.player_profile_trait') IS NULL THEN
    ALTER TABLE me_player_profile_trait RENAME TO player_profile_trait;
    ALTER TABLE player_profile_trait RENAME COLUMN me_player_profile_id TO player_profile_id;
  END IF;

  IF to_regclass('public.me_player_profile_career_team') IS NOT NULL
     AND to_regclass('public.player_profile_career_team') IS NULL THEN
    ALTER TABLE me_player_profile_career_team RENAME TO player_profile_career_team;
    ALTER TABLE player_profile_career_team RENAME COLUMN me_player_profile_id TO player_profile_id;
  END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
             WHERE n.nspname = 'public' AND c.relkind = 'i' AND c.relname = 'idx_me_player_profile_trait_profile') THEN
    ALTER INDEX idx_me_player_profile_trait_profile RENAME TO idx_player_profile_trait_profile;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
             WHERE n.nspname = 'public' AND c.relkind = 'i' AND c.relname = 'idx_me_player_profile_career_team_profile') THEN
    ALTER INDEX idx_me_player_profile_career_team_profile RENAME TO idx_player_profile_career_team_profile;
  END IF;
END $$;

-- +goose Down
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
             WHERE n.nspname = 'public' AND c.relkind = 'i' AND c.relname = 'idx_player_profile_trait_profile') THEN
    ALTER INDEX idx_player_profile_trait_profile RENAME TO idx_me_player_profile_trait_profile;
  END IF;
  IF EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
             WHERE n.nspname = 'public' AND c.relkind = 'i' AND c.relname = 'idx_player_profile_career_team_profile') THEN
    ALTER INDEX idx_player_profile_career_team_profile RENAME TO idx_me_player_profile_career_team_profile;
  END IF;
END $$;

DO $$
BEGIN
  IF to_regclass('public.player_profile_career_team') IS NOT NULL
     AND to_regclass('public.me_player_profile_career_team') IS NULL THEN
    ALTER TABLE player_profile_career_team RENAME COLUMN player_profile_id TO me_player_profile_id;
    ALTER TABLE player_profile_career_team RENAME TO me_player_profile_career_team;
  END IF;

  IF to_regclass('public.player_profile_trait') IS NOT NULL
     AND to_regclass('public.me_player_profile_trait') IS NULL THEN
    ALTER TABLE player_profile_trait RENAME COLUMN player_profile_id TO me_player_profile_id;
    ALTER TABLE player_profile_trait RENAME TO me_player_profile_trait;
  END IF;

  IF to_regclass('public.player_profile') IS NOT NULL
     AND to_regclass('public.me_player_profile') IS NULL THEN
    ALTER TABLE player_profile RENAME TO me_player_profile;
  END IF;
END $$;
