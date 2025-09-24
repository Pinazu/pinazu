-- name: CreateTasksRun :one
INSERT INTO tasks_runs (task_id) VALUES ($1) RETURNING *;

-- name: GetTasksRun :one
SELECT * FROM tasks_runs WHERE task_run_id = $1;

-- name: GetTaskRunByTaskID :many
SELECT * FROM tasks_runs 
WHERE task_id = $1 
ORDER BY created_at DESC;

-- name: GetCurrentTaskRunByTaskID :one
SELECT * FROM tasks_runs
WHERE task_id = $1 AND status IN ('PAUSE', 'SCHEDULED', 'RUNNING');

-- name: GetTaskRunByStatus :many
SELECT * FROM tasks_runs 
WHERE status = $1 
ORDER BY created_at DESC;

-- name: GetPendingTaskRun :many
SELECT * FROM tasks_runs 
WHERE status IN ('SCHEDULED', 'PAUSE') 
ORDER BY created_at ASC;

-- name: GetRunningTaskRun :many
SELECT * FROM tasks_runs 
WHERE status = 'RUNNING' 
ORDER BY created_at ASC;

-- name: UpdateTaskRunStatus :exec
UPDATE tasks_runs
SET status = sqlc.arg(status),
    updated_at = NOW()
WHERE task_run_id = sqlc.arg(task_run_id);

-- name: UpdateTaskRunStatusByTaskID :exec
UPDATE tasks_runs
SET status = sqlc.arg(status), updated_at = NOW()
WHERE task_id = sqlc.arg(task_id) AND status IN ('SCHEDULED', 'RUNNING', 'PENDING');

-- name: UpdateTaskRunStatusWithTimestamps :exec
UPDATE tasks_runs 
SET status = sqlc.arg(status)::text, 
    updated_at = NOW(),
    started_at = CASE WHEN sqlc.arg(status)::text = 'RUNNING' AND started_at IS NULL THEN NOW() ELSE started_at END,
    finished_at = CASE WHEN sqlc.arg(status)::text IN ('FINISHED', 'FAILED') AND finished_at IS NULL THEN NOW() ELSE finished_at END
WHERE task_run_id = sqlc.arg(task_run_id);

-- name: UpdateTaskRunStartedAt :exec
UPDATE tasks_runs 
SET started_at = NOW(),
    updated_at = NOW()
WHERE task_run_id = sqlc.arg(task_run_id) AND started_at IS NULL;

-- name: UpdateTaskRunCurrentLoops :exec
UPDATE tasks_runs 
SET current_loops = $2, 
    updated_at = NOW()
WHERE task_run_id = $1;

-- name: IncrementTaskRunLoops :exec
UPDATE tasks_runs 
SET current_loops = current_loops + 1,
    updated_at = NOW()
WHERE task_run_id = $1;

-- name: DeleteTaskRun :exec
DELETE FROM tasks_runs WHERE task_run_id = $1;

-- name: DeleteOldTaskRun :exec
DELETE FROM tasks_runs 
WHERE created_at < $1 
AND status IN ('FINISHED', 'FAILED');

-- name: ListTaskRun :many
SELECT tr.*, t.thread_id, t.max_request_loop
FROM tasks_runs tr
JOIN tasks t ON tr.task_id = t.id
ORDER BY tr.created_at DESC
LIMIT $1 OFFSET $2;