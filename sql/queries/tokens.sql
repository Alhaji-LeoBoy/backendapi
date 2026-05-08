-- ==================== Token CRUD ====================

-- name: CreateToken :one
INSERT INTO tokens (user_id, token_hash, expiry, scope)
VALUES (?1, ?2, ?3, ?4)
RETURNING id, user_id, token_hash, scope, expiry, created_at;

-- name: GetTokenByID :one
SELECT id, user_id, token_hash, scope, expiry, created_at
FROM tokens
WHERE id = ?1;

-- name: GetUserByTokenHash :one
SELECT users.id, users.username, users.email, users.password, users.bio,
       users.is_admin, users.created_at, users.updated_at
FROM users
INNER JOIN tokens ON users.id = tokens.user_id
WHERE tokens.token_hash = ?1
  AND tokens.scope = ?2
  AND tokens.expiry > datetime('now');

-- name: GetTokenByHash :one
SELECT id, user_id, token_hash, scope, expiry, created_at
FROM tokens
WHERE token_hash = ?1;

-- name: ListTokensForUser :many
SELECT id, user_id, token_hash, scope, expiry, created_at
FROM tokens
WHERE user_id = ?1
ORDER BY created_at DESC;

-- name: DeleteToken :exec
DELETE FROM tokens
WHERE id = ?1;

-- name: DeleteTokenByHash :exec
DELETE FROM tokens
WHERE token_hash = ?1;

-- name: DeleteAllTokensForUser :exec
DELETE FROM tokens
WHERE user_id = ?1;

-- name: DeleteExpiredTokens :exec
DELETE FROM tokens
WHERE expiry < datetime('now');

-- name: DeleteTokensByScope :exec
DELETE FROM tokens
WHERE user_id = ?1 AND scope = ?2;

-- name: GetValidTokenByUserAndScope :one
SELECT id, user_id, token_hash, scope, expiry, created_at
FROM tokens
WHERE user_id = ?1 
  AND scope = ?2 
  AND expiry > datetime('now')
LIMIT 1;

-- name: ConsumeValidRefreshTokenByHash :one
DELETE FROM tokens
WHERE token_hash = ?1
  AND scope = 'refresh'
  AND expiry > datetime('now')
RETURNING id, user_id, token_hash, scope, expiry, created_at;