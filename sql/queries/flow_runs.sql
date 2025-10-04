-- name: CreateFlowRun :one
INSERT INTO flow_runs (
    flow_run_id,
    flow_id,
    parameters,
    status,
    engine,
    task_statuses,
    success_task_results,
    max_retries
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetFlowRun :one
SELECT * FROM flow_runs WHERE flow_run_id = $1;

-- name: GetFlowRunsByFlowID :many
SELECT * FROM flow_runs 
WHERE flow_id = $1 
ORDER BY created_at DESC;

-- name: GetFlowRunsByStatus :many
SELECT * FROM flow_runs 
WHERE status = $1 
ORDER BY created_at DESC;

-- name: GetPendingFlowRuns :many
SELECT * FROM flow_runs 
WHERE status IN ('SCHEDULED', 'PENDING') 
ORDER BY created_at ASC;

-- name: GetFailedFlowRunsForRetry :many
SELECT * FROM flow_runs 
WHERE status = 'FAILED' 
AND retry_count < max_retries 
ORDER BY created_at ASC;

-- name: UpdateFlowRunStatus :exec
UPDATE flow_runs 
SET status = sqlc.arg(status), 
    updated_at = NOW()
WHERE flow_run_id = sqlc.arg(flow_run_id);

-- name: UpdateFlowRunStatusWithTimestamps :exec
UPDATE flow_runs 
SET status = sqlc.arg(status)::text, 
    updated_at = NOW(),
    started_at = CASE WHEN sqlc.arg(status)::text = 'RUNNING' AND started_at IS NULL THEN NOW() ELSE started_at END,
    finished_at = CASE WHEN sqlc.arg(status)::text IN ('SUCCESS', 'FAILED') AND finished_at IS NULL THEN NOW() ELSE finished_at END
WHERE flow_run_id = sqlc.arg(flow_run_id);

-- name: UpdateFlowRunStartedAt :exec
UPDATE flow_runs 
SET started_at = NOW(),
    updated_at = NOW()
WHERE flow_run_id = sqlc.arg(flow_run_id) AND started_at IS NULL;

-- name: UpdateFlowRunTaskStatuses :exec
UPDATE flow_runs 
SET task_statuses = $2, 
    updated_at = NOW()
WHERE flow_run_id = $1;

-- name: UpdateFlowRunSuccessResults :exec
UPDATE flow_runs 
SET success_task_results = $2, 
    updated_at = NOW()
WHERE flow_run_id = $1;

-- name: UpdateFlowRunError :exec
UPDATE flow_runs 
SET error_message = $2, 
    status = 'FAILED',
    finished_at = NOW(),
    updated_at = NOW()
WHERE flow_run_id = $1;

-- name: IncrementFlowRunRetryCount :exec
UPDATE flow_runs 
SET retry_count = retry_count + 1,
    status = 'SCHEDULED',
    started_at = NULL,
    finished_at = NULL,
    updated_at = NOW()
WHERE flow_run_id = $1;

-- name: DeleteFlowRun :exec
DELETE FROM flow_runs WHERE flow_run_id = $1;

-- name: DeleteOldFlowRuns :exec
DELETE FROM flow_runs 
WHERE created_at < $1 
AND status IN ('SUCCESS', 'FAILED');

-- name: ListFlowRuns :many
SELECT fr.*, f.name as flow_name, f.description as flow_description
FROM flow_runs fr
JOIN flows f ON fr.flow_id = f.id
ORDER BY fr.created_at DESC
LIMIT $1 OFFSET $2;