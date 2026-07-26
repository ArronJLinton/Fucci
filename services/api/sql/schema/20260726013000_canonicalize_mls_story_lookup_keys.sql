-- +goose Up
-- Rewrite pre-alias MLS story keys so they match Aliases / seed lookup_keys.
-- Before MLS Aliases (los angeles fc → lafc, sporting kansas city → sporting kc),
-- fan stories were stored under the raw normalized API-Football names and would
-- disappear from list queries that use the canonical keys.

UPDATE match_stories
SET team_lookup_key = 'lafc'
WHERE team_lookup_key = 'los angeles fc';

UPDATE match_stories
SET team_lookup_key = 'sporting kc'
WHERE team_lookup_key = 'sporting kansas city';

-- +goose Down
-- Irreversible: cannot distinguish migrated rows from stories created with
-- canonical keys after Aliases landed. Leave keys as-is.
SELECT 1;
