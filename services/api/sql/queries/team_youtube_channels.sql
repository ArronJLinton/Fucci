-- name: GetTeamYouTubeChannelByLookupKey :one
SELECT id, lookup_key, country, channel_id, channel_handle, is_active, is_verified, created_at, updated_at
FROM team_youtube_channels
WHERE lookup_key = $1 AND is_active = true
LIMIT 1;
