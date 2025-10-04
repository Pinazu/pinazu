-- name: GetUsers :many
SELECT id, name, email, additional_info, provider_name, is_online, created_at, updated_at FROM users ORDER BY name;
-- name: GetUserByID :one
SELECT id, name, email, additional_info, provider_name, is_online, created_at, updated_at FROM users WHERE id = $1 LIMIT 1;
-- name: GetUserByEmail :one
SELECT id, name, email, additional_info, provider_name, is_online, created_at, updated_at FROM users WHERE email = $1 LIMIT 1;
-- name: CreateUser :one
INSERT INTO users (name,email,additional_info,password_hash,provider_name)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, email, additional_info, provider_name, is_online, created_at, updated_at;
-- name: ListRolesForUser :many
SELECT * FROM user_role_mapping WHERE user_id = $1 ORDER BY assigned_at DESC;
-- name: AddRoleToUser :one
INSERT INTO user_role_mapping (user_id, role_id, assigned_by)
VALUES ($1, $2, $3)
RETURNING *;
-- name: RemoveRoleFromUser :exec
DELETE FROM user_role_mapping WHERE user_id = $1 AND role_id = $2;
-- name: GetPasswordHashByID :one
SELECT password_hash FROM users WHERE id = $1 LIMIT 1;
-- name: UpdateUserOnlineState :exec
UPDATE users
SET is_online = $1
WHERE id = $2;
-- name: UpdateUser :one
UPDATE users
SET name = $1, email = $2, additional_info = $3, provider_name = $4
WHERE id = $5
RETURNING id, name, email, additional_info, provider_name, is_online, created_at, updated_at;
-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;