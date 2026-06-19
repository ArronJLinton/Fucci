-- name: ListActiveMediaYouTubeChannels :many
SELECT id, lookup_key, display_name, channel_id, channel_handle, sort_order, is_active, is_verified, created_at, updated_at
FROM media_youtube_channels
WHERE is_active = true
ORDER BY sort_order ASC, display_name ASC;
