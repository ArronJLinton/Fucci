-- 007: current user's player profile (me_player_profile), traits, career teams.

-- name: GetMePlayerProfileByUserID :one
SELECT * FROM me_player_profile WHERE user_id = $1;

-- name: CreateMePlayerProfile :one
INSERT INTO me_player_profile (user_id, age, country_code, club_name, is_free_agent, position)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpsertMePlayerProfile :one
-- Atomic create-or-update on user_id (POST /me/player-profile); preserves photo_url on conflict.
INSERT INTO me_player_profile (user_id, age, country_code, club_name, is_free_agent, position)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id) DO UPDATE SET
  age = EXCLUDED.age,
  country_code = EXCLUDED.country_code,
  club_name = EXCLUDED.club_name,
  is_free_agent = EXCLUDED.is_free_agent,
  position = EXCLUDED.position,
  updated_at = NOW()
RETURNING *;

-- name: UpdateMePlayerProfile :one
UPDATE me_player_profile
SET age = $2, country_code = $3, club_name = $4, is_free_agent = $5, position = $6, photo_url = $7, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateMePlayerProfilePhoto :one
UPDATE me_player_profile SET photo_url = $2, updated_at = NOW() WHERE id = $1 RETURNING *;

-- name: DeleteMePlayerProfile :exec
DELETE FROM me_player_profile WHERE id = $1;

-- name: ListMePlayerProfileTraits :many
SELECT trait_code FROM me_player_profile_trait WHERE me_player_profile_id = $1 ORDER BY trait_code;

-- name: DeleteMePlayerProfileTraitsByProfileID :exec
DELETE FROM me_player_profile_trait WHERE me_player_profile_id = $1;

-- name: InsertMePlayerProfileTrait :one
INSERT INTO me_player_profile_trait (me_player_profile_id, trait_code) VALUES ($1, $2)
RETURNING *;

-- name: ListMePlayerProfileCareerTeams :many
SELECT * FROM me_player_profile_career_team WHERE me_player_profile_id = $1 ORDER BY start_year DESC;

-- name: CreateMePlayerProfileCareerTeam :one
INSERT INTO me_player_profile_career_team (me_player_profile_id, team_name, start_year, end_year)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateMePlayerProfileCareerTeam :one
UPDATE me_player_profile_career_team
SET team_name = $2, start_year = $3, end_year = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteMePlayerProfileCareerTeam :exec
DELETE FROM me_player_profile_career_team WHERE id = $1;

-- name: GetMePlayerProfileCareerTeam :one
SELECT * FROM me_player_profile_career_team WHERE id = $1;
