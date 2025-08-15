-- name: Metric :many
SELECT *
FROM metric
WHERE key LIKE ?
  AND date > sqlc.arg(Min_Date)
  AND date < sqlc.arg(Max_Date);

-- name: MetricsByDate :many
SELECT *
FROM metric
WHERE date > sqlc.arg(Min_Date)
  AND date < sqlc.arg(Max_Date);

-- name: AddValue :one
INSERT INTO metric (
    key, type, value
) VALUES (
    ?, ?, ?
)
RETURNING id;
