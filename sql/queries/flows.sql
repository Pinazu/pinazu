-- name: GetFlows :many
SELECT * FROM flows ORDER BY name LIMIT $1 OFFSET $2;
-- name: GetFlowById :one
SELECT * FROM flows WHERE id = $1 LIMIT 1;
-- name: CreateFlow :one
INSERT INTO flows (id, name, description, parameters_schema, engine, additional_info, tags, code_location, entrypoint)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;
-- name: UpdateFlow :one
UPDATE flows
SET name = $1, description = $2, parameters_schema = $3, engine = $4, additional_info = $5, tags = $6, code_location = $7, entrypoint = $8, updated_at = CURRENT_TIMESTAMP
WHERE id = $9
RETURNING *;
-- name: DeleteFlow :exec
DELETE FROM flows WHERE id = $1;