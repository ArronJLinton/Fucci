-- +goose Up

CREATE TABLE IF NOT EXISTS media_youtube_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lookup_key TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    channel_handle TEXT,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_media_youtube_channels_active_sort
    ON media_youtube_channels (sort_order)
    WHERE is_active = true;

INSERT INTO media_youtube_channels (lookup_key, display_name, channel_id, channel_handle, sort_order, is_verified)
VALUES
    ('fox_soccer', 'FOX SPORTS', 'UCooTLkxcpnTNx6vfOovfBFA', '@FoxSoccer', 1, true),
    ('espn_fc', 'ESPN FC', 'UC6c1z7bA__85CIWZ_jpCK-Q', '@ESPNFC', 2, true),
    ('golazo_america', 'GOLAZO', 'UCh4tni-ICN9z0eMIPcf2r2g', '@golazoamerica', 3, true)
ON CONFLICT (lookup_key) DO NOTHING;

-- +goose Down

DROP TABLE IF EXISTS media_youtube_channels;
