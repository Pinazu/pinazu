-- name: GetToolRunStatus :many
SELECT * FROM tool_runs;
-- name: GetChildToolRunStatusByParentID :many
SELECT * FROM tool_runs WHERE parent_run_id = $1 ORDER BY CASE WHEN id::text ~ '_[0-9]+$' THEN CAST(SUBSTRING(id::text FROM '_([0-9]+)$') AS INTEGER) ELSE 0 END;
-- name: GetChildToolRunStatusByID :one
SELECT * FROM tool_runs WHERE id = $1 AND parent_run_id IS NOT NULL LIMIT 1;
-- name: GetToolRunStatusByID :one
SELECT * FROM tool_runs WHERE id = $1 LIMIT 1;
-- name: CreateToolRunStatus :one
INSERT INTO tool_runs (connection_id, thread_id, agent_id, recipient_id, id, tool_id, input)
VALUES ($1, $2, $3, $4, $5, (SELECT id FROM tools WHERE name = $6), $7)
RETURNING *;
-- name: CreateChildToolRunStatus :one
INSERT INTO tool_runs (connection_id, thread_id, agent_id, recipient_id, id, tool_id, input, parent_run_id)
VALUES ($1, $2, $3, $4, $5, (SELECT id FROM tools WHERE name = $6), $7, $8)
RETURNING *;
-- name: UpdateToolRunStatusByID :one
UPDATE tool_runs
SET result = $1, status = $2, duration = $3
WHERE id = $4
RETURNING *;
-- name: UpdateToolRunStatusToRunningByID :one
UPDATE tool_runs
SET status = 'RUNNING'
WHERE id = $1
RETURNING *;
-- name: UpdateToolRunStatusToSuccessByID :one
UPDATE tool_runs
SET status = 'SUCCESS', duration = $1
WHERE id = $2
RETURNING *;
-- name: UpdateToolRunStatusToFailedByID :one
UPDATE tool_runs
SET status = 'FAILED', duration = $1
WHERE id = $2
RETURNING *;
-- name: CheckIfAllChildToolRunStatusAreCompleted :one
SELECT NOT EXISTS (
  SELECT 1
  FROM tool_runs
  WHERE parent_run_id = $1
    AND status NOT IN ('SUCCESS', 'FAILED')
) AS all_completed;
-- name: DeleteToolRunStatusByID :exec
DELETE FROM tool_runs WHERE id = $1;
-- name: GetToolExecutionCount :one
SELECT COUNT(*) as execution_count FROM tool_runs WHERE tool_id = $1;
-- name: GetLatestToolExecutionStatus :one
SELECT status, duration, created_at, updated_at
FROM tool_runs
WHERE tool_id = $1
ORDER BY created_at DESC
LIMIT 1;
-- name: IsTempParallelToolManagement :one
SELECT EXISTS (
    SELECT 1
    FROM tool_runs tr
    JOIN tools t ON tr.tool_id = t.id
    WHERE tr.id = $1
    AND t.name = 'temp_parallel_tool_management'
) AS is_temp_parallel_tool;