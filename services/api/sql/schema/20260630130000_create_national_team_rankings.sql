-- +goose Up

CREATE TABLE IF NOT EXISTS national_team_rankings (
    team_id      INT PRIMARY KEY,
    fifa_rank    INT NOT NULL CHECK (fifa_rank > 0),
    team_name    TEXT NOT NULL,
    country_code VARCHAR(3),
    ranking_date DATE NOT NULL DEFAULT CURRENT_DATE,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_national_team_rankings_fifa_rank
    ON national_team_rankings (fifa_rank);

-- +goose Down

DROP INDEX IF EXISTS idx_national_team_rankings_fifa_rank;
DROP TABLE IF EXISTS national_team_rankings;
