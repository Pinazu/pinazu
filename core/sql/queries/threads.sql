-- name: GetThreads :many
SELECT * FROM threads WHERE user_id = $1 ORDER BY updated_at DESC;
-- name: GetThreadByID :one
SELECT * FROM threads WHERE user_id = $1 AND id = $2 LIMIT 1;
-- name: CreateThread :one
INSERT INTO threads (title, created_at, updated_at, user_id) VALUES ($1, $2, $3, $4) RETURNING *;
-- name: UpdateThread :one
UPDATE threads
SET title = $1
WHERE id = $2
RETURNING *;
-- name: DeleteThread :exec
DELETE FROM threads WHERE id = $1;

