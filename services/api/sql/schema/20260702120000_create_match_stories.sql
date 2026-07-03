-- +goose Up

DO $$ BEGIN
    CREATE TYPE story_scope_type AS ENUM ('match', 'tournament');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

DO $$ BEGIN
    CREATE TYPE story_content_type AS ENUM ('photo', 'video');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS match_stories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scope_type story_scope_type NOT NULL DEFAULT 'match',
    scope_id TEXT NOT NULL,
    team_lookup_key TEXT NOT NULL,
    content_type story_content_type NOT NULL,
    media_url TEXT NOT NULL,
    caption TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_match_stories_active_scope_team
    ON match_stories (scope_type, scope_id, team_lookup_key, created_at DESC)
    WHERE is_active = TRUE;

CREATE TABLE IF NOT EXISTS content_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reportable_type TEXT NOT NULL,
    reportable_id UUID NOT NULL,
    reason TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_content_reports_reportable
    ON content_reports (reportable_type, reportable_id);

-- +goose Down

DROP INDEX IF EXISTS idx_content_reports_reportable;
DROP TABLE IF EXISTS content_reports;
DROP INDEX IF EXISTS idx_match_stories_active_scope_team;
DROP TABLE IF EXISTS match_stories;
DROP TYPE IF EXISTS story_content_type;
DROP TYPE IF EXISTS story_scope_type;
