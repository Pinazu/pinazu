-- name: GetMessages :many
SELECT * FROM thread_messages WHERE thread_id = $1 ORDER BY created_at ASC;
-- name: GetMessageContents :many
SELECT message FROM thread_messages WHERE thread_id = $1 ORDER BY created_at ASC;
-- name: GetSenderRecipientMessages :many
SELECT message FROM thread_messages WHERE thread_id = $1 AND ((sender_id = $2 AND recipient_id = $3) OR (sender_id = $3 AND recipient_id = $2)) ORDER BY created_at ASC;
-- name: GetMessageByID :one
SELECT * FROM thread_messages WHERE id = $1 LIMIT 1;
-- name: CreateCustomMessage :one
INSERT INTO thread_messages (thread_id, message, sender_type, result_type, stop_reason, sender_id, citations, recipient_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;
-- name: CreateAgentMessage :one
INSERT INTO thread_messages (thread_id, message, sender_type, stop_reason, sender_id, citations, recipient_id)
VALUES ($1, $2, 'assistant', $3, $4, $5, $6)
RETURNING *;
-- name: CreateUserMessage :one
INSERT INTO thread_messages (thread_id, message, sender_type, sender_id, recipient_id)
VALUES ($1, $2, 'user', $3, $4)
RETURNING *;
-- name: CreateResultMessage :one
INSERT INTO thread_messages (thread_id, message, sender_type, result_type, sender_id, recipient_id)
VALUES ($1, $2, "result", $3, $4, $5)
RETURNING *;
-- name: UpdateMessage :one
UPDATE thread_messages
SET message = $1
WHERE id = $2
RETURNING *;
-- name: DeleteMessage :exec
DELETE FROM thread_messages WHERE id = $1;

