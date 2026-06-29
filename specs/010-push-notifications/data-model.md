# Data Model: Push Notifications

**Feature**: 010-push-notifications  
**Phase**: 1 — infrastructure

## `push_devices`

One row per Expo push token registered to a user. A user may have phone + tablet.

| Column | Type | Notes |
|--------|------|-------|
| `id` | SERIAL PK | |
| `user_id` | INT FK → users(id) ON DELETE CASCADE | |
| `expo_push_token` | VARCHAR(255) UNIQUE NOT NULL | `ExponentPushToken[...]` |
| `platform` | VARCHAR(16) | `ios` \| `android` |
| `timezone` | VARCHAR(64) NOT NULL | IANA, e.g. `America/New_York` |
| `app_version` | VARCHAR(32) | From mobile Constants |
| `enabled` | BOOLEAN DEFAULT true | False when Expo returns DeviceNotRegistered |
| `last_seen_at` | TIMESTAMPTZ | Heartbeat / register |
| `created_at` | TIMESTAMPTZ | |
| `updated_at` | TIMESTAMPTZ | |

**Indexes:** `(user_id)`, `(user_id, enabled)` where enabled.

## `push_preferences`

One row per user (create on first register or signup).

| Column | Type | Default | Notes |
|--------|------|---------|-------|
| `user_id` | INT PK FK → users(id) | | |
| `master_enabled` | BOOLEAN | false | Opt-in gate |
| `debates_enabled` | BOOLEAN | false | 6pm local |
| `news_enabled` | BOOLEAN | false | 12pm local |
| `matches_enabled` | BOOLEAN | false | Post-FT |
| `updated_at` | TIMESTAMPTZ | | |

## `push_send_ledger`

Idempotency — at most one successful send per user per campaign per **local calendar date**.

| Column | Type | Notes |
|--------|------|-------|
| `id` | SERIAL PK | |
| `user_id` | INT FK | |
| `campaign_key` | VARCHAR(128) | e.g. `debate:daily`, `news:daily`, `match:1489391:highlights` |
| `local_date` | DATE | User’s local date at send time |
| `sent_at` | TIMESTAMPTZ | |
| UNIQUE | | `(user_id, campaign_key, local_date)` |

## `push_delivery_log`

Audit / debugging (retention 90 days).

| Column | Type | Notes |
|--------|------|-------|
| `id` | SERIAL PK | |
| `user_id` | INT | |
| `push_device_id` | INT NULL FK | |
| `campaign_key` | VARCHAR(128) | |
| `title` | VARCHAR(255) | |
| `expo_ticket_id` | VARCHAR(64) NULL | |
| `status` | VARCHAR(32) | `sent` \| `failed` \| `skipped_prefs` \| `skipped_dedupe` |
| `error_message` | TEXT NULL | |
| `created_at` | TIMESTAMPTZ | |

## Phase 2 tables (not migrated in Phase 1)

- `national_team_rankings` — FIFA rank by API-Football team id (marquee + debate boost)
- `news_article_opens` — `(user_id, article_url, opened_at)` for news exclusion
