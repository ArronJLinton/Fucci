-- +goose Up

INSERT INTO team_youtube_channels (lookup_key, country, channel_id, is_verified)
VALUES
    ('ecuador', 'Ecuador', 'UCqNNxN4yHqIg-jB9fqc5Ang', true),
    ('sweden', 'Sweden', 'UCj-avF0P_UyIhGJLBJDeZbw', true),
    ('croatia', 'Croatia', 'UCsqWbe1Tp3ZkobTmcKqZmjg', true),
    ('switzerland', 'Switzerland', 'UCQMc_xMoCdug9JggI8hYa5Q', true),
    ('austria', 'Austria', 'UCdzz38PI9PKi2JVOPoij0AA', true),
    ('canada', 'Canada', 'UCPwOPM0PbkY0S1V_OqA-xjg', true),
    ('egypt', 'Egypt', 'UCHZWuCcKHtp3OLwQOHZC-eg', true),
    ('nigeria', 'Nigeria', 'UCNG7u9Z0J-8ZyZ8KjPvKqvA', true)
ON CONFLICT (lookup_key) DO UPDATE SET
    channel_id = EXCLUDED.channel_id,
    is_verified = EXCLUDED.is_verified,
    updated_at = NOW();

-- +goose Down

DELETE FROM team_youtube_channels
WHERE lookup_key IN (
    'ecuador',
    'sweden',
    'croatia',
    'switzerland',
    'austria',
    'canada',
    'egypt',
    'nigeria'
);
