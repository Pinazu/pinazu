-- name: UpsertWorkerHeartbeat :one
INSERT INTO worker_heartbeats (
    worker_id,
    worker_name,
    status,
    last_heartbeat,
    worker_info
) VALUES (
    $1, $2, $3, $4, $5
) ON CONFLICT (worker_id) DO UPDATE SET
    worker_name = EXCLUDED.worker_name,
    status = EXCLUDED.status,
    last_heartbeat = EXCLUDED.last_heartbeat,
    worker_info = EXCLUDED.worker_info,
    updated_at = NOW()
RETURNING *;

-- name: GetWorkerHeartbeat :one
SELECT * FROM worker_heartbeats WHERE worker_id = $1;

-- name: GetActiveWorkers :many
SELECT * FROM worker_heartbeats 
WHERE status = 'ACTIVE' 
AND last_heartbeat > NOW() - INTERVAL '5 minutes'
ORDER BY last_heartbeat DESC;

-- name: GetAllWorkers :many
SELECT * FROM worker_heartbeats 
ORDER BY last_heartbeat DESC;

-- name: UpdateWorkerStatus :exec
UPDATE worker_heartbeats 
SET status = $2,
    updated_at = NOW()
WHERE worker_id = $1;

-- name: MarkStaleWorkersInactive :exec
UPDATE worker_heartbeats 
SET status = 'INACTIVE',
    updated_at = NOW()
WHERE last_heartbeat < $1 
AND status = 'ACTIVE';

-- name: DeleteWorkerHeartbeat :exec
DELETE FROM worker_heartbeats WHERE worker_id = $1;

-- name: DeleteInactiveWorkers :exec
DELETE FROM worker_heartbeats 
WHERE status = 'INACTIVE' 
AND updated_at < $1;

-- name: GetWorkerStats :one
SELECT 
    COUNT(*) as total_workers,
    COUNT(*) FILTER (WHERE status = 'ACTIVE') as active_workers,
    COUNT(*) FILTER (WHERE status = 'INACTIVE') as inactive_workers,
    COUNT(*) FILTER (WHERE status = 'FAILED') as failed_workers
FROM worker_heartbeats;