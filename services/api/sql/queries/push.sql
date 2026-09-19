-- name: UpsertPushDevice :one
INSERT INTO push_devices (user_id, expo_push_token, platform, timezone, app_version, enabled, last_seen_at, updated_at)
VALUES ($1, $2, $3, $4, $5, true, NOW(), NOW())
ON CONFLICT (expo_push_token) DO UPDATE SET
    user_id = EXCLUDED.user_id,
    platform = EXCLUDED.platform,
    timezone = EXCLUDED.timezone,
    app_version = EXCLUDED.app_version,
    enabled = true,
    last_seen_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetPushDeviceByIDForUser :one
SELECT * FROM push_devices
WHERE id = $1 AND user_id = $2;

-- name: DeletePushDeviceForUser :exec
DELETE FROM push_devices
WHERE id = $1 AND user_id = $2;

-- name: ListPushDevicesForUser :many
SELECT * FROM push_devices
WHERE user_id = $1
ORDER BY last_seen_at DESC;

-- name: ListEnabledPushDevicesForUser :many
SELECT * FROM push_devices
WHERE user_id = $1 AND enabled = true
ORDER BY last_seen_at DESC;

-- name: CountPushDevicesForUser :one
SELECT COUNT(*)::bigint FROM push_devices
WHERE user_id = $1;

-- name: DeleteOldestPushDeviceForUser :exec
DELETE FROM push_devices
WHERE id = (
    SELECT pd.id FROM push_devices pd
    WHERE pd.user_id = $1
    ORDER BY pd.last_seen_at ASC
    LIMIT 1
);

-- name: EnsurePushPreferences :one
INSERT INTO push_preferences (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO UPDATE SET updated_at = NOW()
RETURNING *;

-- name: GetPushPreferences :one
SELECT * FROM push_preferences
WHERE user_id = $1;

-- name: UpdatePushPreferences :one
UPDATE push_preferences
SET
    master_enabled = COALESCE(sqlc.narg('master_enabled'), master_enabled),
    debates_enabled = COALESCE(sqlc.narg('debates_enabled'), debates_enabled),
    news_enabled = COALESCE(sqlc.narg('news_enabled'), news_enabled),
    matches_enabled = COALESCE(sqlc.narg('matches_enabled'), matches_enabled),
    updated_at = NOW()
WHERE user_id = $1
RETURNING *;

-- name: TryInsertPushSendLedger :one
INSERT INTO push_send_ledger (user_id, campaign_key, local_date)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, campaign_key, local_date) DO NOTHING
RETURNING *;

-- name: DeletePushSendLedger :exec
DELETE FROM push_send_ledger
WHERE user_id = $1 AND campaign_key = $2 AND local_date = $3;

-- name: InsertPushDeliveryLog :one
INSERT INTO push_delivery_log (user_id, push_device_id, campaign_key, title, expo_ticket_id, status, error_message)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: DisablePushDevice :exec
UPDATE push_devices
SET enabled = false, updated_at = NOW()
WHERE id = $1;

-- name: ListSlotCampaignCandidates :many
SELECT DISTINCT ON (pp.user_id)
    pp.user_id,
    pd.timezone,
    pp.debates_enabled,
    pp.news_enabled,
    pp.matches_enabled
FROM push_preferences pp
INNER JOIN push_devices pd ON pd.user_id = pp.user_id AND pd.enabled = true
WHERE pp.master_enabled = true
  AND (pp.debates_enabled OR pp.news_enabled OR pp.matches_enabled)
ORDER BY pp.user_id, pd.last_seen_at DESC;

-- name: ListMatchPushCandidates :many
SELECT DISTINCT ON (pp.user_id)
    pp.user_id,
    pd.timezone
FROM push_preferences pp
INNER JOIN push_devices pd ON pd.user_id = pp.user_id AND pd.enabled = true
WHERE pp.master_enabled = true AND pp.matches_enabled = true
ORDER BY pp.user_id, pd.last_seen_at DESC;

-- name: CountMatchPushSendsForUserOnDate :one
SELECT COUNT(*)::bigint AS count
FROM push_send_ledger
WHERE user_id = $1
  AND local_date = $2
  AND campaign_key LIKE 'match:%';
