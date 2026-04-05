-- 007: signed-in user player profile (player_profile), traits, career teams.

-- name: GetPlayerProfileByUserID :one
SELECT * FROM player_profile WHERE user_id = $1;

-- name: CreatePlayerProfileRow :one
INSERT INTO player_profile (user_id, age, country_code, club_name, is_free_agent, position)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpsertPlayerProfile :one
-- POST /player-profile: single statement. New row: omitted cores -> COALESCE(NULL, 50). Conflict update:
-- NULL core param keeps player_profile.* (no stale read/merge in app code); non-NULL overwrites. photo_url unchanged.
INSERT INTO player_profile (
  user_id, age, country_code, club_name, is_free_agent, position,
  speed, shooting, passing, dribbling, defending, physical, stamina
)
VALUES (
  sqlc.arg('user_id'),
  sqlc.arg('age'),
  sqlc.arg('country_code'),
  sqlc.arg('club_name'),
  sqlc.arg('is_free_agent'),
  sqlc.arg('position'),
  COALESCE(sqlc.narg('speed'), 50),
  COALESCE(sqlc.narg('shooting'), 50),
  COALESCE(sqlc.narg('passing'), 50),
  COALESCE(sqlc.narg('dribbling'), 50),
  COALESCE(sqlc.narg('defending'), 50),
  COALESCE(sqlc.narg('physical'), 50),
  COALESCE(sqlc.narg('stamina'), 50)
)
ON CONFLICT (user_id) DO UPDATE SET
  age = EXCLUDED.age,
  country_code = EXCLUDED.country_code,
  club_name = EXCLUDED.club_name,
  is_free_agent = EXCLUDED.is_free_agent,
  position = EXCLUDED.position,
  speed = CASE WHEN sqlc.narg('speed') IS NULL THEN player_profile.speed ELSE sqlc.narg('speed')::integer END,
  shooting = CASE WHEN sqlc.narg('shooting') IS NULL THEN player_profile.shooting ELSE sqlc.narg('shooting')::integer END,
  passing = CASE WHEN sqlc.narg('passing') IS NULL THEN player_profile.passing ELSE sqlc.narg('passing')::integer END,
  dribbling = CASE WHEN sqlc.narg('dribbling') IS NULL THEN player_profile.dribbling ELSE sqlc.narg('dribbling')::integer END,
  defending = CASE WHEN sqlc.narg('defending') IS NULL THEN player_profile.defending ELSE sqlc.narg('defending')::integer END,
  physical = CASE WHEN sqlc.narg('physical') IS NULL THEN player_profile.physical ELSE sqlc.narg('physical')::integer END,
  stamina = CASE WHEN sqlc.narg('stamina') IS NULL THEN player_profile.stamina ELSE sqlc.narg('stamina')::integer END,
  updated_at = NOW()
RETURNING *;

-- name: UpdatePlayerProfileRow :one
UPDATE player_profile
SET age = $2,
    country_code = $3,
    club_name = $4,
    is_free_agent = $5,
    position = $6,
    photo_url = $7,
    speed = $8,
    shooting = $9,
    passing = $10,
    dribbling = $11,
    defending = $12,
    physical = $13,
    stamina = $14,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdatePlayerProfilePhotoRow :one
UPDATE player_profile SET photo_url = $2, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: DeletePlayerProfileRow :exec
DELETE FROM player_profile WHERE id = $1;

-- name: ListPlayerProfileTraits :many
-- id reflects insert order (matches client PUT order after delete-then-insert).
SELECT trait_code FROM player_profile_trait WHERE player_profile_id = $1 ORDER BY id;

-- name: DeletePlayerProfileTraitsByProfileID :exec
DELETE FROM player_profile_trait WHERE player_profile_id = $1;

-- name: InsertPlayerProfileTrait :one
INSERT INTO player_profile_trait (player_profile_id, trait_code) VALUES ($1, $2)
RETURNING *;

-- name: ListPlayerProfileCareerTeams :many
SELECT * FROM player_profile_career_team WHERE player_profile_id = $1 ORDER BY start_year DESC;

-- name: CreatePlayerProfileCareerTeam :one
INSERT INTO player_profile_career_team (player_profile_id, team_name, start_year, end_year)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdatePlayerProfileCareerTeam :one
UPDATE player_profile_career_team
SET team_name = $2, start_year = $3, end_year = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeletePlayerProfileCareerTeam :exec
DELETE FROM player_profile_career_team WHERE id = $1;

-- name: GetPlayerProfileCareerTeam :one
SELECT * FROM player_profile_career_team WHERE id = $1;
