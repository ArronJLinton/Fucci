-- name: CreateMatchStory :one
INSERT INTO match_stories (
    user_id,
    scope_type,
    scope_id,
    team_lookup_key,
    content_type,
    media_url,
    caption
) VALUES (
    sqlc.arg(user_id),
    sqlc.arg(scope_type)::story_scope_type,
    sqlc.arg(scope_id),
    sqlc.arg(team_lookup_key),
    sqlc.arg(content_type)::story_content_type,
    sqlc.arg(media_url),
    sqlc.narg(caption)
)
RETURNING *;

-- name: ListActiveMatchStoriesForTeam :many
SELECT
    ms.*,
    u.display_name AS user_display_name,
    u.avatar_url AS user_avatar_url
FROM match_stories ms
JOIN users u ON u.id = ms.user_id
WHERE ms.scope_type = sqlc.arg(scope_type)::story_scope_type
  AND ms.scope_id = sqlc.arg(scope_id)
  AND ms.team_lookup_key = sqlc.arg(team_lookup_key)
  AND ms.is_active = TRUE
ORDER BY ms.created_at DESC
LIMIT sqlc.arg(row_limit);

-- name: GetMatchStoryByID :one
SELECT * FROM match_stories WHERE id = sqlc.arg(id);

-- name: DeactivateMatchStory :one
UPDATE match_stories
SET is_active = FALSE
WHERE id = sqlc.arg(id) AND is_active = TRUE
RETURNING *;
