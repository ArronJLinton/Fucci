-- name: CreateContentReport :one
INSERT INTO content_reports (
    reporter_id,
    reportable_type,
    reportable_id,
    reason,
    description
) VALUES (
    sqlc.arg(reporter_id),
    sqlc.arg(reportable_type),
    sqlc.arg(reportable_id),
    sqlc.arg(reason),
    sqlc.narg(description)
)
RETURNING *;
