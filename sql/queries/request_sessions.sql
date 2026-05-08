-- name: CreateRequestSession :one
INSERT INTO request_sessions (
    user_id,
    session_token_hash,
    ip_address,
    user_agent,
    expires_at
)
VALUES (?1, ?2, ?3, ?4, ?5)
RETURNING id, user_id, session_token_hash, ip_address, user_agent, expires_at, last_seen_at, created_at, revoked_at;

-- name: GetRequestSessionByTokenHash :one
SELECT id, user_id, session_token_hash, ip_address, user_agent, expires_at, last_seen_at, created_at, revoked_at
FROM request_sessions
WHERE session_token_hash = ?1;

-- name: ListActiveRequestSessionsForUser :many
SELECT id, user_id, session_token_hash, ip_address, user_agent, expires_at, last_seen_at, created_at, revoked_at
FROM request_sessions
WHERE user_id = ?1
  AND revoked_at IS NULL
  AND expires_at > datetime('now')
ORDER BY created_at DESC;

-- name: TouchRequestSession :exec
UPDATE request_sessions
SET last_seen_at = datetime('now')
WHERE session_token_hash = ?1
  AND revoked_at IS NULL
  AND expires_at > datetime('now');

-- name: RevokeRequestSessionByTokenHash :exec
UPDATE request_sessions
SET revoked_at = datetime('now')
WHERE session_token_hash = ?1
  AND revoked_at IS NULL;

-- name: RevokeAllRequestSessionsForUser :exec
UPDATE request_sessions
SET revoked_at = datetime('now')
WHERE user_id = ?1
  AND revoked_at IS NULL;

-- name: DeleteExpiredOrRevokedRequestSessions :exec
DELETE FROM request_sessions
WHERE expires_at < datetime('now')
   OR revoked_at IS NOT NULL;
