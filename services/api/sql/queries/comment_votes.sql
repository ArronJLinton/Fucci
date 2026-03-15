-- name: UpsertCommentVote :one
INSERT INTO comment_votes (comment_id, user_id, vote_type)
VALUES ($1, $2, $3)
ON CONFLICT (comment_id, user_id)
DO UPDATE SET vote_type = $3, created_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DeleteCommentVote :exec
DELETE FROM comment_votes WHERE comment_id = $1 AND user_id = $2;

-- name: GetCommentVoteNetScore :one
SELECT (
    COUNT(*) FILTER (WHERE vote_type = 'upvote') -
    COUNT(*) FILTER (WHERE vote_type = 'downvote')
)::int AS net_score
FROM comment_votes
WHERE comment_id = $1;

-- name: GetCommentVoteByUser :one
SELECT * FROM comment_votes WHERE comment_id = $1 AND user_id = $2;
