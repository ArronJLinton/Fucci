-- name: ListNationalTeamRankings :many
SELECT team_id, fifa_rank, team_name, country_code, ranking_date, updated_at
FROM national_team_rankings
ORDER BY fifa_rank ASC;

-- name: GetBestFIFARankForTeams :one
SELECT MIN(fifa_rank)::int AS best_rank
FROM national_team_rankings
WHERE team_id = ANY($1::int[]);
