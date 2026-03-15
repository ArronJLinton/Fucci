-- name: CreateDebate :one
INSERT INTO debates (match_id, debate_type, headline, description, ai_generated)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetDebate :one
SELECT * FROM debates WHERE id = $1 AND deleted_at IS NULL;

-- name: GetDebatesByMatch :many
SELECT * FROM debates 
WHERE match_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetDebatesByType :many
SELECT * FROM debates 
WHERE debate_type = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdateDebate :one
UPDATE debates 
SET headline = $2, description = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteDebate :exec
DELETE FROM debates WHERE id = $1;

-- name: SoftDeleteDebate :exec
UPDATE debates SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1;

-- name: RestoreDebate :exec
UPDATE debates SET deleted_at = NULL WHERE id = $1;

-- name: CreateDebateCard :one
INSERT INTO debate_cards (debate_id, stance, title, description, ai_generated)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetDebateCards :many
SELECT * FROM debate_cards WHERE debate_id = $1 ORDER BY stance;

-- name: GetDebateCard :one
SELECT * FROM debate_cards WHERE id = $1;

-- name: UpdateDebateCard :one
UPDATE debate_cards 
SET title = $2, description = $3, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteDebateCard :exec
DELETE FROM debate_cards WHERE id = $1;

-- name: CreateVote :one
INSERT INTO votes (debate_card_id, user_id, vote_type, emoji)
VALUES ($1, $2, $3, $4)
ON CONFLICT (debate_card_id, user_id, vote_type, emoji) 
DO UPDATE SET emoji = $4, created_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetVotesByCard :many
SELECT * FROM votes WHERE debate_card_id = $1;

-- name: GetUserVote :one
SELECT * FROM votes WHERE debate_card_id = $1 AND user_id = $2 AND vote_type = $3;

-- name: DeleteVote :exec
DELETE FROM votes WHERE debate_card_id = $1 AND user_id = $2 AND vote_type = $3;

-- name: GetVoteCounts :many
SELECT 
    debate_card_id,
    vote_type,
    emoji,
    COUNT(*) as count
FROM votes 
WHERE debate_card_id = ANY($1::int[])
GROUP BY debate_card_id, vote_type, emoji;

-- name: CreateComment :one
INSERT INTO comments (debate_id, parent_comment_id, user_id, content, seeded)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetComments :many
SELECT 
    c.*,
    u.firstname,
    u.lastname
FROM comments c
JOIN users u ON c.user_id = u.id
WHERE c.debate_id = $1
ORDER BY c.created_at ASC;

-- name: GetComment :one
SELECT 
    c.*,
    u.firstname,
    u.lastname
FROM comments c
JOIN users u ON c.user_id = u.id
WHERE c.id = $1;

-- name: UpdateComment :one
UPDATE comments 
SET content = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: DeleteComment :exec
DELETE FROM comments WHERE id = $1;

-- name: GetCommentCount :one
SELECT COUNT(*) FROM comments WHERE debate_id = $1;

-- name: CreateDebateAnalytics :one
INSERT INTO debate_analytics (debate_id, total_votes, total_comments, engagement_score)
VALUES ($1, $2, $3, $4)
ON CONFLICT (debate_id) 
DO UPDATE SET 
    total_votes = $2,
    total_comments = $3,
    engagement_score = $4,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetDebateAnalytics :one
SELECT * FROM debate_analytics WHERE debate_id = $1;

-- name: UpdateDebateAnalytics :one
UPDATE debate_analytics 
SET total_votes = $2, total_comments = $3, engagement_score = $4, updated_at = CURRENT_TIMESTAMP
WHERE debate_id = $1
RETURNING *;

-- name: GetTopDebates :many
SELECT 
    d.*,
    da.total_votes,
    da.total_comments,
    da.engagement_score
FROM debates d
LEFT JOIN debate_analytics da ON d.id = da.debate_id
WHERE d.deleted_at IS NULL
ORDER BY da.engagement_score DESC NULLS LAST
LIMIT $1; 