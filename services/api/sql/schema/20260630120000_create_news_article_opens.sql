-- +goose Up

CREATE TABLE IF NOT EXISTS news_article_opens (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    article_url TEXT NOT NULL,
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, article_url)
);

CREATE INDEX IF NOT EXISTS idx_news_article_opens_user_opened
    ON news_article_opens(user_id, opened_at DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_news_article_opens_user_opened;
DROP TABLE IF EXISTS news_article_opens;
