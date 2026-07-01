-- name: UpsertNewsArticleOpen :exec
INSERT INTO news_article_opens (user_id, article_url, opened_at)
VALUES ($1, $2, NOW())
ON CONFLICT (user_id, article_url) DO UPDATE SET opened_at = EXCLUDED.opened_at;

-- name: ListOpenedArticleURLsForUser :many
SELECT article_url FROM news_article_opens
WHERE user_id = $1
  AND opened_at >= NOW() - INTERVAL '90 days'
ORDER BY opened_at DESC
LIMIT 500;
