-- name: CreateDebate :one
INSERT INTO debates (match_id, debate_type, headline, description, ai_generated, match_info)
VALUES ($1, $2, $3, $4, $5, $6)
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

-- name: GetUserSwipeVotesForCards :many
SELECT id, debate_card_id, user_id, vote_type, emoji, created_at
FROM votes
WHERE user_id = $1
  AND debate_card_id = ANY($2::int[])
  AND emoji IS NULL
  AND vote_type IN ('upvote', 'downvote');

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
-- Binary consensus for list UI: agree side = upvotes on agree cards; disagree side = downvotes on agree
-- plus all swipe votes on disagree cards (covers detail-screen “no” on agree, hero left as downvote on disagree, legacy upvote-on-disagree).
-- Vote aggregates are scoped to debates in the limited feed (not the full votes table).
-- name: ListDebatesPublicFeed :many
WITH feed_candidates AS (
    SELECT d.id
    FROM debates d
    LEFT JOIN debate_analytics da ON d.id = da.debate_id
    WHERE d.deleted_at IS NULL
      AND d.created_at >= CURRENT_TIMESTAMP - INTERVAL '6 days'
      AND EXISTS (SELECT 1 FROM debate_cards dc WHERE dc.debate_id = d.id)
    ORDER BY da.engagement_score DESC NULLS LAST, d.created_at DESC
    LIMIT $1
)
SELECT 
    d.id, d.match_id, d.debate_type, d.headline, d.description, d.ai_generated, d.match_info, d.deleted_at, d.created_at, d.updated_at,
    da.total_votes,
    da.total_comments,
    da.engagement_score,
    COALESCE(bbin.binary_agree_upvotes, 0) AS binary_agree_upvotes,
    COALESCE(bbin.binary_disagree_upvotes, 0) AS binary_disagree_upvotes
FROM debates d
INNER JOIN feed_candidates fc ON fc.id = d.id
LEFT JOIN debate_analytics da ON d.id = da.debate_id
LEFT JOIN (
    SELECT
        dc.debate_id,
        COUNT(*) FILTER (WHERE dc.stance = 'agree' AND v.vote_type = 'upvote')::bigint AS binary_agree_upvotes,
        COUNT(*) FILTER (
          WHERE (dc.stance = 'agree' AND v.vote_type = 'downvote')
             OR (dc.stance = 'disagree' AND v.vote_type IN ('upvote', 'downvote'))
        )::bigint AS binary_disagree_upvotes
    FROM votes v
    INNER JOIN debate_cards dc ON v.debate_card_id = dc.id
    WHERE v.emoji IS NULL
      AND v.vote_type IN ('upvote', 'downvote')
      AND dc.stance IN ('agree', 'disagree')
      AND dc.debate_id IN (SELECT id FROM feed_candidates)
    GROUP BY dc.debate_id
) bbin ON bbin.debate_id = d.id
ORDER BY da.engagement_score DESC NULLS LAST, d.created_at DESC;

-- Authenticated feed — "new": user has not cast any swipe vote (upvote/downvote, emoji IS NULL) on a binary card.
-- One vote per debate (first swipe on any binary card moves the debate to "voted"). Emoji reactions do not count. Wildcards do not count.
-- name: ListDebatesFeedNewForUser :many
WITH feed_candidates AS (
    SELECT d.id
    FROM debates d
    LEFT JOIN debate_analytics da ON d.id = da.debate_id
    WHERE d.deleted_at IS NULL
      AND d.created_at >= CURRENT_TIMESTAMP - INTERVAL '6 days'
      AND EXISTS (SELECT 1 FROM debate_cards dc0 WHERE dc0.debate_id = d.id AND dc0.stance IN ('agree', 'disagree'))
      AND NOT EXISTS (
        SELECT 1
        FROM debate_cards dc
        INNER JOIN votes v ON v.debate_card_id = dc.id AND v.user_id = $1
          AND v.vote_type IN ('upvote', 'downvote')
          AND v.emoji IS NULL
        WHERE dc.debate_id = d.id AND dc.stance IN ('agree', 'disagree')
      )
    ORDER BY da.engagement_score DESC NULLS LAST, d.created_at DESC
    LIMIT $2
)
SELECT 
    d.id, d.match_id, d.debate_type, d.headline, d.description, d.ai_generated, d.match_info, d.deleted_at, d.created_at, d.updated_at,
    da.total_votes,
    da.total_comments,
    da.engagement_score,
    COALESCE(bbin.binary_agree_upvotes, 0) AS binary_agree_upvotes,
    COALESCE(bbin.binary_disagree_upvotes, 0) AS binary_disagree_upvotes
FROM debates d
INNER JOIN feed_candidates fc ON fc.id = d.id
LEFT JOIN debate_analytics da ON d.id = da.debate_id
LEFT JOIN (
    SELECT
        dc.debate_id,
        COUNT(*) FILTER (WHERE dc.stance = 'agree' AND v.vote_type = 'upvote')::bigint AS binary_agree_upvotes,
        COUNT(*) FILTER (
          WHERE (dc.stance = 'agree' AND v.vote_type = 'downvote')
             OR (dc.stance = 'disagree' AND v.vote_type IN ('upvote', 'downvote'))
        )::bigint AS binary_disagree_upvotes
    FROM votes v
    INNER JOIN debate_cards dc ON v.debate_card_id = dc.id
    WHERE v.emoji IS NULL
      AND v.vote_type IN ('upvote', 'downvote')
      AND dc.stance IN ('agree', 'disagree')
      AND dc.debate_id IN (SELECT id FROM feed_candidates)
    GROUP BY dc.debate_id
) bbin ON bbin.debate_id = d.id
ORDER BY da.engagement_score DESC NULLS LAST, d.created_at DESC;

-- Authenticated feed — "voted": user has at least one swipe vote (emoji IS NULL) on any agree/disagree card.
-- name: ListDebatesFeedVotedForUser :many
WITH feed_candidates AS (
    SELECT d.id, uv.last_voted_at
    FROM debates d
    INNER JOIN (
      SELECT
        dc.debate_id,
        MAX(v.created_at)::timestamptz AS last_voted_at
      FROM votes v
      INNER JOIN debate_cards dc ON v.debate_card_id = dc.id
      WHERE dc.stance IN ('agree', 'disagree')
        AND v.user_id = $1
        AND v.vote_type IN ('upvote', 'downvote')
        AND v.emoji IS NULL
      GROUP BY dc.debate_id
    ) uv ON uv.debate_id = d.id
    LEFT JOIN debate_analytics da ON d.id = da.debate_id
    WHERE d.deleted_at IS NULL
      AND d.created_at >= CURRENT_TIMESTAMP - INTERVAL '6 days'
      AND EXISTS (SELECT 1 FROM debate_cards dc0 WHERE dc0.debate_id = d.id AND dc0.stance IN ('agree', 'disagree'))
    ORDER BY uv.last_voted_at DESC NULLS LAST
    LIMIT $2
)
SELECT 
    d.id, d.match_id, d.debate_type, d.headline, d.description, d.ai_generated, d.match_info, d.deleted_at, d.created_at, d.updated_at,
    da.total_votes,
    da.total_comments,
    da.engagement_score,
    COALESCE(bbin.binary_agree_upvotes, 0) AS binary_agree_upvotes,
    COALESCE(bbin.binary_disagree_upvotes, 0) AS binary_disagree_upvotes,
    fc.last_voted_at
FROM debates d
INNER JOIN feed_candidates fc ON fc.id = d.id
LEFT JOIN debate_analytics da ON d.id = da.debate_id
LEFT JOIN (
    SELECT
        dc.debate_id,
        COUNT(*) FILTER (WHERE dc.stance = 'agree' AND v.vote_type = 'upvote')::bigint AS binary_agree_upvotes,
        COUNT(*) FILTER (
          WHERE (dc.stance = 'agree' AND v.vote_type = 'downvote')
             OR (dc.stance = 'disagree' AND v.vote_type IN ('upvote', 'downvote'))
        )::bigint AS binary_disagree_upvotes
    FROM votes v
    INNER JOIN debate_cards dc ON v.debate_card_id = dc.id
    WHERE v.emoji IS NULL
      AND v.vote_type IN ('upvote', 'downvote')
      AND dc.stance IN ('agree', 'disagree')
      AND dc.debate_id IN (SELECT id FROM feed_candidates)
    GROUP BY dc.debate_id
) bbin ON bbin.debate_id = d.id
ORDER BY last_voted_at DESC NULLS LAST;