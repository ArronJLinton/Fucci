-- +goose Up
-- Resolve the 5 MLS channels that were seeded unverified / empty, and correct
-- St. Louis handle (@stlouiscitysc → @stlcitysc). Idempotent upsert.

INSERT INTO team_youtube_channels (lookup_key, country, channel_id, channel_handle, is_verified)
VALUES
    ('columbus crew', 'USA', 'UCWaD_J0qRubgPr9DV_x9-Jw', '@ColumbusCrew', true),
    ('la galaxy', 'USA', 'UCQsasqjpe_blMBHbUoWYGZg', '@lagalaxy', true),
    ('portland timbers', 'USA', 'UCm0KnY18KTa_h9bcm3aFEBw', '@TimbersFC', true),
    ('seattle sounders', 'USA', 'UCVhbRUhe_hfmgi-UN1gcQzw', '@SoundersFC', true),
    ('st louis city', 'USA', 'UCNxHd0AvwN7uhyAh7ZpRsdg', '@stlcitysc', true)
ON CONFLICT (lookup_key) DO UPDATE SET
    channel_id = EXCLUDED.channel_id,
    channel_handle = EXCLUDED.channel_handle,
    country = EXCLUDED.country,
    is_verified = EXCLUDED.is_verified,
    updated_at = NOW();

-- +goose Down

UPDATE team_youtube_channels
SET channel_id = '', channel_handle = CASE lookup_key
        WHEN 'columbus crew' THEN '@ColumbusCrew'
        WHEN 'la galaxy' THEN '@lagalaxy'
        WHEN 'portland timbers' THEN '@TimbersFC'
        WHEN 'seattle sounders' THEN '@SoundersFC'
        WHEN 'st louis city' THEN '@stlouiscitysc'
    END,
    is_verified = false,
    updated_at = NOW()
WHERE lookup_key IN (
    'columbus crew',
    'la galaxy',
    'portland timbers',
    'seattle sounders',
    'st louis city'
);
