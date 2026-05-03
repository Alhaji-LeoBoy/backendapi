-- name: GetEvent :one
SELECT * FROM events WHERE id = ?;

-- name: ListEvents :many
SELECT * FROM events ORDER BY start_time
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountEvents :one
SELECT COUNT(*) FROM events;

-- name: GetUserEvents :many
SELECT * FROM events WHERE user_id = ? ORDER BY start_time;

-- name: CreateEvent :one
INSERT INTO events (
  user_id,
  title,
  owner_name,
  description,
  location,
  start_time,
  image_url,
  price
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: PatchEvent :one
UPDATE events
SET
  title = COALESCE(sqlc.narg('title'), title),
  owner_name = COALESCE(sqlc.narg('owner_name'), owner_name),
  description = COALESCE(sqlc.narg('description'), description),
  location = COALESCE(sqlc.narg('location'), location),
  start_time = COALESCE(sqlc.narg('start_time'), start_time),
  image_url = COALESCE(sqlc.narg('image_url'), image_url),
  price = COALESCE(sqlc.narg('price'), price),
  updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg('id') AND user_id = sqlc.arg('user_id')
RETURNING *;

-- name: DeleteEvent :exec
DELETE FROM events WHERE id = ? AND user_id = ?;

-- name: AdminDeleteEvent :exec
DELETE FROM events WHERE id = ?;

-- name: GetEventWithOwner :one
SELECT
  e.*,
  u.username AS owner_username,
  u.email AS owner_email
FROM events e
JOIN users u ON u.id = e.user_id
WHERE e.id = ?;

-- name: ListEventsWithTicketStats :many
SELECT
  e.*,
  COUNT(t.id) AS tickets_count,
  COUNT(DISTINCT t.user_id) AS unique_users_count,
  COALESCE(SUM(t.price), 0) AS revenue
FROM events e
LEFT JOIN tickets t ON t.event_id = e.id
GROUP BY e.id
ORDER BY e.start_time;

-- name: GetUserEventsWithStats :many
SELECT
  e.*,
  COUNT(t.id) AS tickets_count,
  COUNT(DISTINCT t.user_id) AS unique_users_count,
  COALESCE(SUM(t.price), 0) AS revenue
FROM events e
LEFT JOIN tickets t ON t.event_id = e.id
WHERE e.user_id = ?
GROUP BY e.id
ORDER BY e.start_time DESC;

-- name: ListEventsBetween :many
SELECT * FROM events
WHERE start_time BETWEEN ? AND ?
ORDER BY start_time;

-- name: SearchEvents :many
SELECT * FROM events
WHERE title LIKE ? OR location LIKE ?
ORDER BY start_time
LIMIT ? OFFSET ?;
