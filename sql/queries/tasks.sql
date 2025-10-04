-- name: GetTasks :many
SELECT * FROM tasks ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: GetTaskById :one
SELECT * FROM tasks WHERE id = $1 LIMIT 1;

-- name: CreateTask :one
INSERT INTO tasks (thread_id, max_request_loop, additional_info, created_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateTaskWithID :one
INSERT INTO tasks (id, thread_id, max_request_loop, additional_info, created_by, parent_task_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateTask :one
UPDATE tasks
SET max_request_loop = $1, additional_info = $2
WHERE id = $3
RETURNING *;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = $1;

-- name: GetTasksByThreadId :many
SELECT * FROM tasks WHERE thread_id = $1 ORDER BY created_at DESC;