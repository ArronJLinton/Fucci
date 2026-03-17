-- name: AddCommentReaction :one
INSERT INTO comment_reactions (comment_id, user_id, emoji)
VALUES ($1, $2, $3)
ON CONFLICT (comment_id, user_id, emoji)
DO UPDATE SET created_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: RemoveCommentReaction :exec
DELETE FROM comment_reactions
WHERE comment_id = $1 AND user_id = $2 AND emoji = $3;

-- name: GetCommentReactionsByCommentID :many
SELECT emoji, COUNT(*)::int AS count
FROM comment_reactions
WHERE comment_id = $1
GROUP BY emoji;

-- name: GetCommentReactionsByCommentIDsBatch :many
SELECT comment_id, emoji, COUNT(*)::int AS count
FROM comment_reactions
WHERE comment_id = ANY($1::int[])
GROUP BY comment_id, emoji;

-- name: GetUserCommentReaction :one
SELECT * FROM comment_reactions
WHERE comment_id = $1 AND user_id = $2 AND emoji = $3;
