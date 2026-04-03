-- 007: signed-in user player profile (player_profile), traits, career teams.

-- name: GetPlayerProfileByUserID :one
SELECT * FROM player_profile WHERE user_id = $1;

-- name: CreatePlayerProfileRow :one
INSERT INTO player_profile (user_id, age, country_code, club_name, is_free_agent, position)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpsertPlayerProfile :one
-- Atomic create-or-update on user_id (POST /player-profile); preserves photo_url on conflict.
INSERT INTO player_profile (
  user_id, age, country_code, club_name, is_free_agent, position,
  speed, shooting, passing, dribbling, defending, physical, stamina
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (user_id) DO UPDATE SET
  age = EXCLUDED.age,
  country_code = EXCLUDED.country_code,
  club_name = EXCLUDED.club_name,
  is_free_agent = EXCLUDED.is_free_agent,
  position = EXCLUDED.position,
  speed = EXCLUDED.speed,
  shooting = EXCLUDED.shooting,
  passing = EXCLUDED.passing,
  dribbling = EXCLUDED.dribbling,
  defending = EXCLUDED.defending,
  physical = EXCLUDED.physical,
  stamina = EXCLUDED.stamina,
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
SELECT trait_code FROM player_profile_trait WHERE player_profile_id = $1 ORDER BY trait_code;

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
