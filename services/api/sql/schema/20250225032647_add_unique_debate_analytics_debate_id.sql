-- +goose Up
-- Add UNIQUE on debate_id so ON CONFLICT (debate_id) in CreateDebateAnalytics works.
-- Remove duplicates first if any (keep one row per debate_id with lowest id).
DELETE FROM debate_analytics a
USING debate_analytics b
WHERE a.debate_id = b.debate_id AND a.id > b.id;
ALTER TABLE debate_analytics ADD CONSTRAINT debate_analytics_debate_id_key UNIQUE (debate_id);
-- +goose Down
ALTER TABLE debate_analytics DROP CONSTRAINT IF EXISTS debate_analytics_debate_id_key;
