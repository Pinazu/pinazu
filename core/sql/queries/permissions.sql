-- name: GetAllPermissions :many
SELECT * FROM permissions ORDER BY name;
-- name: GetPermissionByID :one
SELECT * FROM permissions WHERE id = $1 LIMIT 1;
-- name: CreatePermission :one
INSERT INTO permissions (name, description, content)
VALUES ($1, $2, $3)
RETURNING *;
-- name: UpdatePermission :one
UPDATE permissions
SET name = $1, description = $2, content = $3, updated_at = NOW()
WHERE id = $4
RETURNING *;
-- name: DeletePermission :exec
DELETE FROM permissions WHERE id = $1;