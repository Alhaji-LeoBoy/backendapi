-- name: GetTicket :one
SELECT * FROM tickets WHERE id = ?;

-- name: ListTickets :many
SELECT * FROM tickets ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountTickets :one
SELECT COUNT(*) FROM tickets;

-- name: GetUserTickets :many
SELECT * FROM tickets WHERE user_id = ? ORDER BY created_at;

-- name: GetEventTickets :many
SELECT * FROM tickets WHERE event_id = ? ORDER BY created_at;

-- name: CreateTicket :one
INSERT INTO tickets (
  user_id,
  event_id,
  name,
  signature,
  price,
  status
)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: PatchTicket :one
UPDATE tickets
SET
  name = COALESCE(sqlc.narg('name'), name),
  price = COALESCE(sqlc.narg('price'), price),
  status = COALESCE(sqlc.narg('status'), status),
  updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg('id') AND user_id = sqlc.arg('user_id')
RETURNING *;

-- name: DeleteTicket :exec
DELETE FROM tickets WHERE id = ? AND user_id = ?;

-- name: AdminDeleteTicket :exec
DELETE FROM tickets WHERE id = ?;

-- name: GetTicketDetailed :one
SELECT
  t.*,
  e.title AS event_title,
  e.start_time AS event_start_time,
  e.image_url AS event_image_url,
  u.username AS user_username,
  u.email AS user_email
FROM tickets t
JOIN events e ON e.id = t.event_id
JOIN users u ON u.id = t.user_id
WHERE t.id = ?;

-- name: GetUserTicketsWithEvent :many
SELECT
  t.*,
  e.title AS event_title,
  e.image_url AS event_image_url
FROM tickets t
JOIN events e ON e.id = t.event_id
WHERE t.user_id = ?
ORDER BY t.created_at DESC;

-- name: ListTicketsByStatusForEvent :many
SELECT * FROM tickets
WHERE event_id = ? AND status = ?
ORDER BY created_at;

-- name: ListTicketsForEventWithUser :many
SELECT
  t.*,
  u.username AS user_username,
  u.email AS user_email
FROM tickets t
JOIN users u ON u.id = t.user_id
WHERE t.event_id = ?
ORDER BY t.created_at;

-- name: ListTicketsBetween :many
SELECT * FROM tickets
WHERE created_at BETWEEN ? AND ?
ORDER BY created_at;
