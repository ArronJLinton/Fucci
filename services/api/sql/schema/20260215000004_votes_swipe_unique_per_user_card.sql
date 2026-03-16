-- +goose Up
-- One swipe vote per user per card: partial unique index for rows with emoji IS NULL.
-- Postgres treats NULLs as distinct in UNIQUE(debate_card_id, user_id, vote_type, emoji),
-- so duplicate swipe votes (emoji=NULL) were possible without this.
CREATE UNIQUE INDEX IF NOT EXISTS idx_votes_swipe_one_per_user_card
ON votes (debate_card_id, user_id)
WHERE emoji IS NULL AND vote_type IN ('upvote', 'downvote');

-- +goose Down
DROP INDEX IF EXISTS idx_votes_swipe_one_per_user_card;
