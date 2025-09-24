-- name: GetAgents :many
SELECT * FROM agents ORDER BY name;
-- name: GetAgentByID :one
SELECT * FROM agents WHERE id = $1 LIMIT 1;
-- name: GetAgentSpecsByID :one
SELECT specs FROM agents WHERE id = $1 LIMIT 1;
-- name: CheckAgentIDValid :one
SELECT EXISTS(SELECT 1 FROM agents WHERE id = $1) AS is_valid;
-- name: CreateAgent :one
INSERT INTO agents (name, description, specs, created_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;
-- name: UpdateAgent :one
UPDATE agents
SET name = $1, description = $2, specs = $3
WHERE id = $4
RETURNING *;
-- name: ListPermissionsForAgent :many
SELECT * FROM agent_permission_mapping WHERE agent_id = $1 ORDER BY assigned_at DESC;
-- name: AddAgentPermission :one
INSERT INTO agent_permission_mapping (agent_id, permission_id, assigned_by)
VALUES ($1, $2, $3)
RETURNING *;
-- name: RemoveAgentPermission :exec
DELETE FROM agent_permission_mapping WHERE agent_id = $1 AND permission_id = $2;
-- name: DeleteAgent :exec
DELETE FROM agents WHERE id = $1;
