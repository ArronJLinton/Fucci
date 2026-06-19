-- +goose Up

CREATE TABLE IF NOT EXISTS team_youtube_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lookup_key TEXT NOT NULL UNIQUE,
    country TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    channel_handle TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_team_youtube_channels_lookup_active
    ON team_youtube_channels (lookup_key)
    WHERE is_active = true;

INSERT INTO team_youtube_channels (lookup_key, country, channel_id, channel_handle, is_verified)
VALUES
    ('england', 'England', 'UCNT2e7Og56vm5_V-yJWvglA', '@england', true),
    ('france', 'France', 'UCeJlXGyEl7kBgQJKADAHM3A', '@fff', true),
    ('germany', 'Germany', 'UC7am34-1rGU_ky1vWYnoOJQ', '@germanfootball', true),
    ('spain', 'Spain', 'UCQBxzdEPXjy05MtpfbdtMxQ', '@rfef', true),
    ('netherlands', 'Netherlands', 'UCpnmJcBhJqKIHFkKvgdkdMQ', '@onsoranje', true),
    ('portugal', 'Portugal', 'UCsIoK3XP-cVcpoCC_SdjZdg', '@fpf', true),
    ('belgium', 'Belgium', 'UCtH7VXC4kowqhH2-RMq0F-Q', '@belgianreddevils', true),
    ('poland', 'Poland', 'UCFlGSuyqfui9DwttuET8MzQ', '@laczynaspilka', true),
    ('scotland', 'Scotland', 'UCsaS5aZjlJPJCzNwjEc3waw', '@scotlandnt', true),
    ('turkiye', 'Türkiye', 'UCVoNfzLtczc_TV8-xKjcmKg', '@tff', true),
    ('brazil', 'Brazil', 'UCdQuDaRww5NkKpQQ1BJBWww', '@cbf_futebol', true),
    ('argentina', 'Argentina', 'UC0fOQAA4EgAv5qHMNcuMxWQ', '@afaseleccion', true),
    ('united states', 'United States', 'UCk1pcWQ5E19g0Cgp4c1eI1w', '@ussoccer', true),
    ('mexico', 'Mexico', 'UCQMe9zooGEZD1XvLuaAI2fw', '@miseleccionmx', true),
    ('japan', 'Japan', 'UCgIeUSV91-FfmCayG4lSBcw', '@jfatv', true),
    ('south korea', 'South Korea', 'UCpjOmwiy88a9EV3Rv8ukJgw', '@kfatv', true),
    ('australia', 'Australia', 'UC5qRwzD58S_sTEoLpl6AzTQ', '@footballaustralia', true),
    ('uzbekistan', 'Uzbekistan', 'UChLdynAXzGMaliL5dVxwLkg', '@uzbekistanfa', true),
    ('morocco', 'Morocco', 'UCbQlejA3nCVMq-9qw-oZEtQ', '@frmf', true),
    ('senegal', 'Senegal', 'UCBnBRS_fdrIQFxW6w2BiG1g', '@fsftv', true)
ON CONFLICT (lookup_key) DO NOTHING;

-- +goose Down

DROP TABLE IF EXISTS team_youtube_channels;
