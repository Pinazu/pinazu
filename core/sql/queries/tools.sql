-- ==============================================
-- TOOL QUERIES FOR SQLC
-- ==============================================

-- name: ListTools :many
SELECT *
FROM tools t
ORDER BY t.created_at DESC;

-- name: GetToolById :one
SELECT *
FROM tools
WHERE id = $1;

-- name: CreateTool :one
INSERT INTO tools (
    name,
    description, 
    config,
    created_by
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: UpdateTool :one
UPDATE tools SET
    description = COALESCE($2, description),
    config = COALESCE($3, config)
WHERE id = $1
RETURNING *;

-- name: DeleteTool :exec
DELETE FROM tools WHERE id = $1;

-- name: GetToolInfoByName :one
SELECT * FROM tools WHERE name = $1;

-- name: GetToolsByIDs :many
SELECT * FROM tools 
WHERE id = ANY($1::uuid[])
ORDER BY name;
