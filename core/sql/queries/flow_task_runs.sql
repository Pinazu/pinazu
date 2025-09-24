-- name: CreateFlowTaskRun :one
INSERT INTO flow_task_runs (
    flow_run_id,
    task_name,
    status,
    result_cache_key,
    max_retries
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetFlowTaskRun :one
SELECT * FROM flow_task_runs WHERE flow_run_id = $1 AND task_name = $2;

-- name: GetFlowTaskRunsByFlowRun :many
SELECT * FROM flow_task_runs 
WHERE flow_run_id = $1 
ORDER BY created_at ASC;

-- name: GetFlowTaskRunByFlowRunAndName :one
SELECT * FROM flow_task_runs 
WHERE flow_run_id = $1 AND task_name = $2;

-- name: GetFlowTaskRunsByStatus :many
SELECT * FROM flow_task_runs 
WHERE status = $1 
ORDER BY created_at DESC;

-- name: GetRunningFlowTaskRuns :many
SELECT * FROM flow_task_runs 
WHERE status = 'RUNNING' 
ORDER BY created_at ASC;

-- name: GetFailedFlowTaskRunsForRetry :many
SELECT * FROM flow_task_runs 
WHERE status = 'FAILED' 
AND retry_count < max_retries 
ORDER BY created_at ASC;

-- name: UpdateFlowTaskRunStatus :exec
UPDATE flow_task_runs 
SET status = sqlc.arg(status), 
    updated_at = NOW()
WHERE flow_run_id = sqlc.arg(flow_run_id) AND task_name = sqlc.arg(task_name);

-- name: UpdateFlowTaskRunStatusWithTimestamps :exec
UPDATE flow_task_runs 
SET status = sqlc.arg(status), 
    updated_at = NOW(),
    started_at = CASE WHEN sqlc.arg(status)::text = 'RUNNING' AND started_at IS NULL THEN NOW() ELSE started_at END,
    finished_at = CASE WHEN sqlc.arg(status)::text IN ('SUCCESS', 'FAILED', 'SKIPPED') AND finished_at IS NULL THEN NOW() ELSE finished_at END
WHERE flow_run_id = sqlc.arg(flow_run_id) AND task_name = sqlc.arg(task_name);

-- name: UpdateFlowTaskRunResult :exec
UPDATE flow_task_runs 
SET result = $3,
    result_cache_key = $4,
    status = 'SUCCESS',
    finished_at = NOW(),
    duration_seconds = EXTRACT(EPOCH FROM (NOW() - started_at)),
    updated_at = NOW()
WHERE flow_run_id = $1 AND task_name = $2;

-- name: UpdateFlowTaskRunError :exec
UPDATE flow_task_runs 
SET error_message = $3, 
    status = 'FAILED',
    finished_at = NOW(),
    duration_seconds = EXTRACT(EPOCH FROM (NOW() - started_at)),
    updated_at = NOW()
WHERE flow_run_id = $1 AND task_name = $2;

-- name: IncrementFlowTaskRunRetryCount :exec
UPDATE flow_task_runs 
SET retry_count = retry_count + 1,
    status = 'PENDING',
    started_at = NULL,
    finished_at = NULL,
    error_message = NULL,
    updated_at = NOW()
WHERE flow_run_id = $1 AND task_name = $2;

-- name: DeleteFlowTaskRun :exec
DELETE FROM flow_task_runs WHERE flow_run_id = $1 AND task_name = $2;

-- name: DeleteFlowTaskRunsByFlowRun :exec
DELETE FROM flow_task_runs WHERE flow_run_id = $1;

-- name: GetFlowTaskRunStats :one
SELECT 
    COUNT(*) as total_count,
    COUNT(*) FILTER (WHERE status = 'SUCCESS') as success_count,
    COUNT(*) FILTER (WHERE status = 'FAILED') as failed_count,
    COUNT(*) FILTER (WHERE status = 'RUNNING') as running_count,
    COUNT(*) FILTER (WHERE status = 'PENDING') as pending_count,
    AVG(duration_seconds) FILTER (WHERE duration_seconds IS NOT NULL) as avg_duration
FROM flow_task_runs 
WHERE flow_run_id = $1;