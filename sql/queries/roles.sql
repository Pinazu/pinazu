-- name: GetAllRoles :many
SELECT * FROM roles ORDER BY name;
-- name: GetRoleByID :one
SELECT * FROM roles WHERE id = $1 LIMIT 1;
-- name: CreateRole :one
INSERT INTO roles (name, description, is_system)
VALUES ($1, $2, $3)
RETURNING *;
-- name: ListPermissionForRole :many
SELECT * FROM role_permission_mapping WHERE role_id = $1 ORDER BY assigned_at DESC;
-- name: CheckPermissionExistsForRole :one
SELECT EXISTS(SELECT 1 FROM role_permission_mapping WHERE role_id = $1 AND permission_id = $2) AS exists;
-- name: AddPermissionToRole :one
INSERT INTO role_permission_mapping (role_id, permission_id, assigned_by)
VALUES ($1, $2, $3)
RETURNING *;
-- name: DeletePermissionFromRole :exec
DELETE FROM role_permission_mapping WHERE role_id = $1 AND permission_id = $2;
-- name: UpdateRole :one
UPDATE roles
SET name = $1, description = $2, is_system = $3, updated_at = NOW()
WHERE id = $4
RETURNING *;
-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1;