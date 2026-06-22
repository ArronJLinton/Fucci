-- +goose Up

INSERT INTO media_youtube_channels (lookup_key, display_name, channel_id, channel_handle, sort_order, is_verified)
VALUES
    ('telemundo_deportes', 'Telemundo Deportes', 'UCjZ7QPKb89R-4SxzBoceyOg', '@TelemundoDeportes', 4, true)
ON CONFLICT (lookup_key) DO NOTHING;

-- +goose Down

DELETE FROM media_youtube_channels
WHERE lookup_key = 'telemundo_deportes';
