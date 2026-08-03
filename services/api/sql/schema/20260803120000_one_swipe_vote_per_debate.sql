-- +goose Up
-- Spec 009: one binary swipe vote per user per debate.
-- Historical bug allowed a row on the agree card and another on the disagree card
-- because uniqueness was only (debate_card_id, user_id). Keep the newest swipe
-- vote per (debate, user) and drop older conflicting rows.

DELETE FROM votes
WHERE id IN (
    SELECT id
    FROM (
        SELECT
            v.id,
            ROW_NUMBER() OVER (
                PARTITION BY dc.debate_id, v.user_id
                ORDER BY v.created_at DESC NULLS LAST, v.id DESC
            ) AS rn
        FROM votes v
        INNER JOIN debate_cards dc ON dc.id = v.debate_card_id
        WHERE v.emoji IS NULL
          AND v.vote_type IN ('upvote', 'downvote')
          AND dc.stance IN ('agree', 'disagree')
          AND v.user_id IS NOT NULL
          AND dc.debate_id IS NOT NULL
    ) ranked
    WHERE rn > 1
);

-- +goose Down
-- Irreversible data cleanup; nothing to restore.
