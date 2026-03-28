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

-- name: DeleteCardSwipeVotes :exec
DELETE FROM votes
WHERE debate_card_id = $1 AND user_id = $2
  AND vote_type IN ('upvote', 'downvote') AND emoji IS NULL;

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
    u.lastname,
    u.display_name,
    u.avatar_url
FROM comments c
JOIN users u ON c.user_id = u.id
WHERE c.debate_id = $1
ORDER BY c.created_at ASC;

-- name: GetComment :one
SELECT 
    c.*,
    u.firstname,
    u.lastname,
    u.display_name,
    u.avatar_url
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

-- Public browse feed: engagement desc, tie-break created_at desc; only debates with at least one card.
-- name: ListDebatesPublicFeed :many
SELECT 
    d.id, d.match_id, d.debate_type, d.headline, d.description, d.ai_generated, d.deleted_at, d.created_at, d.updated_at,
    da.total_votes,
    da.total_comments,
    da.engagement_score
FROM debates d
LEFT JOIN debate_analytics da ON d.id = da.debate_id
WHERE d.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM debate_cards dc WHERE dc.debate_id = d.id)
ORDER BY da.engagement_score DESC NULLS LAST, d.created_at DESC
LIMIT $1;

-- Authenticated feed — "new": user has not cast swipe votes on all cards (completion uses upvote/downvote per card).
-- name: ListDebatesFeedNewForUser :many
SELECT 
    d.id, d.match_id, d.debate_type, d.headline, d.description, d.ai_generated, d.deleted_at, d.created_at, d.updated_at,
    da.total_votes,
    da.total_comments,
    da.engagement_score
FROM debates d
LEFT JOIN debate_analytics da ON d.id = da.debate_id
WHERE d.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM debate_cards dc0 WHERE dc0.debate_id = d.id)
  AND (
    SELECT COUNT(*)::bigint FROM debate_cards dc_c WHERE dc_c.debate_id = d.id
  ) > COALESCE((
    SELECT COUNT(DISTINCT dc3.id)::bigint
    FROM debate_cards dc3
    INNER JOIN votes v ON v.debate_card_id = dc3.id AND v.user_id = $1
      AND v.vote_type IN ('upvote', 'downvote')
    WHERE dc3.debate_id = d.id
  ), 0)
ORDER BY da.engagement_score DESC NULLS LAST, d.created_at DESC
LIMIT $2;

-- Authenticated feed — "voted": user has swipe votes covering every card; order by last swipe time desc.
-- name: ListDebatesFeedVotedForUser :many
SELECT 
    d.id, d.match_id, d.debate_type, d.headline, d.description, d.ai_generated, d.deleted_at, d.created_at, d.updated_at,
    da.total_votes,
    da.total_comments,
    da.engagement_score,
    (
      SELECT MAX(v.created_at)
      FROM votes v
      INNER JOIN debate_cards dc ON v.debate_card_id = dc.id
      WHERE dc.debate_id = d.id AND v.user_id = $1
        AND v.vote_type IN ('upvote', 'downvote')
    ) AS last_voted_at
FROM debates d
LEFT JOIN debate_analytics da ON d.id = da.debate_id
WHERE d.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM debate_cards dc0 WHERE dc0.debate_id = d.id)
  AND (
    SELECT COUNT(*)::bigint FROM debate_cards dc_c WHERE dc_c.debate_id = d.id
  ) = (
    SELECT COUNT(DISTINCT dc4.id)::bigint
    FROM debate_cards dc4
    INNER JOIN votes v2 ON v2.debate_card_id = dc4.id AND v2.user_id = $1
      AND v2.vote_type IN ('upvote', 'downvote')
    WHERE dc4.debate_id = d.id
  )
ORDER BY last_voted_at DESC NULLS LAST
LIMIT $2;