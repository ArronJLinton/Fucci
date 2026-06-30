-- +goose Up

CREATE TABLE IF NOT EXISTS push_devices (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expo_push_token VARCHAR(255) NOT NULL UNIQUE,
    platform VARCHAR(16) NOT NULL,
    timezone VARCHAR(64) NOT NULL,
    app_version VARCHAR(32),
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_push_devices_user_id ON push_devices(user_id);
CREATE INDEX IF NOT EXISTS idx_push_devices_user_enabled ON push_devices(user_id) WHERE enabled = true;

CREATE TABLE IF NOT EXISTS push_preferences (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    master_enabled BOOLEAN NOT NULL DEFAULT false,
    debates_enabled BOOLEAN NOT NULL DEFAULT false,
    news_enabled BOOLEAN NOT NULL DEFAULT false,
    matches_enabled BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS push_send_ledger (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    campaign_key VARCHAR(128) NOT NULL,
    local_date DATE NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, campaign_key, local_date)
);

CREATE INDEX IF NOT EXISTS idx_push_send_ledger_user_campaign ON push_send_ledger(user_id, campaign_key, local_date);

CREATE TABLE IF NOT EXISTS push_delivery_log (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    push_device_id INTEGER REFERENCES push_devices(id) ON DELETE SET NULL,
    campaign_key VARCHAR(128) NOT NULL,
    title VARCHAR(255) NOT NULL,
    expo_ticket_id VARCHAR(64),
    status VARCHAR(32) NOT NULL,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_push_delivery_log_user_created ON push_delivery_log(user_id, created_at DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_push_delivery_log_user_created;
DROP TABLE IF EXISTS push_delivery_log;
DROP INDEX IF EXISTS idx_push_send_ledger_user_campaign;
DROP TABLE IF EXISTS push_send_ledger;
DROP TABLE IF EXISTS push_preferences;
DROP INDEX IF EXISTS idx_push_devices_user_enabled;
DROP INDEX IF EXISTS idx_push_devices_user_id;
DROP TABLE IF EXISTS push_devices;
