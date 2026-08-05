-- +goose Up

ALTER TABLE content_reports
    ALTER COLUMN reportable_id TYPE TEXT USING reportable_id::TEXT;

ALTER TABLE content_reports
    ADD COLUMN reported_user_id INTEGER NULL REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_content_reports_reported_user_id
    ON content_reports (reported_user_id);

-- +goose Down

DROP INDEX IF EXISTS idx_content_reports_reported_user_id;

ALTER TABLE content_reports
    DROP COLUMN IF EXISTS reported_user_id;

ALTER TABLE content_reports
    ALTER COLUMN reportable_id TYPE UUID USING reportable_id::UUID;
