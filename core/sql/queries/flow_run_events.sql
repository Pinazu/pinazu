-- name: CreateFlowRunEvent :one
INSERT INTO flow_run_events (
    event_id,
    flow_run_id,
    task_name,
    event_type,
    event_data,
    event_timestamp,
    source
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetFlowRunEvent :one
SELECT * FROM flow_run_events WHERE event_id = $1;

-- name: GetFlowRunEventsByFlowRun :many
SELECT * FROM flow_run_events 
WHERE flow_run_id = $1 
ORDER BY event_timestamp ASC;

-- name: GetFlowRunEventsByType :many
SELECT * FROM flow_run_events 
WHERE flow_run_id = $1 AND event_type = $2 
ORDER BY event_timestamp ASC;

-- name: GetFlowRunEventsByTask :many
SELECT * FROM flow_run_events 
WHERE flow_run_id = $1 AND task_name = $2 
ORDER BY event_timestamp ASC;

-- name: GetRecentFlowRunEvents :many
SELECT fre.*, fr.flow_id, f.name as flow_name
FROM flow_run_events fre
JOIN flow_runs fr ON fre.flow_run_id = fr.flow_run_id
JOIN flows f ON fr.flow_id = f.id
WHERE fre.event_timestamp >= $1
ORDER BY fre.event_timestamp DESC
LIMIT $2;

-- name: GetEventsBySource :many
SELECT * FROM flow_run_events 
WHERE source = $1 
ORDER BY event_timestamp DESC
LIMIT $2;

-- name: DeleteFlowRunEvent :exec
DELETE FROM flow_run_events WHERE event_id = $1;

-- name: DeleteFlowRunEventsByFlowRun :exec
DELETE FROM flow_run_events WHERE flow_run_id = $1;

-- name: DeleteOldFlowRunEvents :exec
DELETE FROM flow_run_events 
WHERE event_timestamp < $1;

-- name: GetEventStats :one
SELECT 
    COUNT(*) as total_events,
    COUNT(DISTINCT flow_run_id) as unique_flow_runs,
    COUNT(*) FILTER (WHERE event_type = 'FlowRunRequest') as request_events,
    COUNT(*) FILTER (WHERE event_type = 'FlowRunStatusEvent') as status_events,
    COUNT(*) FILTER (WHERE event_type = 'TaskRunStatusEvent') as task_events
FROM flow_run_events 
WHERE event_timestamp >= $1;