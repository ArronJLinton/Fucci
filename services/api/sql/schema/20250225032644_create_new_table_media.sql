-- +goose Up
-- +goose StatementBegin
CREATE TABLE media (
  id SERIAL PRIMARY KEY,
  match_id VARCHAR(50) NOT NULL,
  media_url VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_media_match_id ON media(match_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_media_match_id;
DROP TABLE media; 
-- +goose StatementEnd
