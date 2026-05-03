-- name: CreateTransaction :one
INSERT INTO transactions (
  user_id,
  event_id,
  ticket_id,
  transaction_id,
  amount,
  card_last4,
  status
)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetTransaction :one
SELECT * FROM transactions WHERE id = ?;

-- name: GetTransactionByTxnID :one
SELECT * FROM transactions WHERE transaction_id = ?;

-- name: GetUserTransactions :many
SELECT
  t.*,
  e.title AS event_title
FROM transactions t
JOIN events e ON e.id = t.event_id
WHERE t.user_id = ?
ORDER BY t.created_at DESC;

-- name: UpdateTransactionTicketID :exec
UPDATE transactions SET ticket_id = ? WHERE id = ?;
