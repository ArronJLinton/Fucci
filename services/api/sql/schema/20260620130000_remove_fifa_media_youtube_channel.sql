-- +goose Up

DELETE FROM media_youtube_channels
WHERE lookup_key = 'fifa';

-- +goose Down

INSERT INTO media_youtube_channels (lookup_key, display_name, channel_id, channel_handle, sort_order, is_verified)
VALUES
    ('fifa', 'FIFA', 'UCpcTrCXblq78GZrTUTLWeBw', '@FIFA', 3, true)
ON CONFLICT (lookup_key) DO NOTHING;
