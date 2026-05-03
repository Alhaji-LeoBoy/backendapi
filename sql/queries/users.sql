-- ==================== User CRUD ====================

-- name: CreateUser :one
INSERT INTO users (username, email, password, bio)
VALUES (?1, ?2, ?3, ?4)
RETURNING id, username, email, password, bio, is_admin, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, username, email, password, bio, is_admin, created_at, updated_at
FROM users
WHERE id = ?1;

-- name: GetUserByEmail :one
SELECT id, username, email, password, bio, is_admin, created_at, updated_at
FROM users
WHERE email = ?1;

-- name: GetUserByUsername :one
SELECT id, username, email, password, bio, is_admin, created_at, updated_at
FROM users
WHERE username = ?1;

-- name: GetUserByUsernameOrEmail :one
SELECT id, username, email, password, bio, is_admin, created_at, updated_at
FROM users
WHERE username = ?1 OR email = ?2;

-- name: ListUsers :many
SELECT id, username, email, password, bio, is_admin, created_at, updated_at
FROM users
ORDER BY created_at DESC
LIMIT ?1 OFFSET ?2;

-- name: SearchUsers :many
SELECT id, username, email, password, bio, is_admin, created_at, updated_at
FROM users
WHERE username LIKE '%' || ?1 || '%'
   OR email LIKE '%' || ?1 || '%'
ORDER BY created_at DESC
LIMIT ?2 OFFSET ?3;

-- name: PatchUser :one
UPDATE users
SET 
    username = COALESCE(sqlc.narg('username'), username),
    email = COALESCE(sqlc.narg('email'), email),
    password = COALESCE(sqlc.narg('password'), password),
    bio = COALESCE(sqlc.narg('bio'), bio),
    updated_at = datetime('now')
WHERE id = sqlc.arg('id')
RETURNING id, username, email, password, bio, is_admin, created_at, updated_at;

-- name: UpdatePassword :exec
UPDATE users
SET password = ?1, updated_at = datetime('now')
WHERE id = ?2;

-- name: UpdateEmail :exec
UPDATE users
SET email = ?1, updated_at = datetime('now')
WHERE id = ?2;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = ?1;

-- ==================== Additional Useful Queries ====================

-- name: UserExists :one
SELECT EXISTS(
    SELECT 1 FROM users WHERE id = ?1
);

-- name: EmailExists :one
SELECT EXISTS(
    SELECT 1 FROM users WHERE email = ?1
);

-- name: UsernameExists :one
SELECT EXISTS(
    SELECT 1 FROM users WHERE username = ?1
);

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: GetUserWithTokens :many
SELECT 
    u.id, u.username, u.email, u.password, u.bio, u.is_admin, u.created_at, u.updated_at,
    t.id as token_id, t.token_hash, t.scope, t.expiry, t.created_at as token_created_at
FROM users u
LEFT JOIN tokens t ON u.id = t.user_id AND t.expiry > datetime('now')
WHERE u.id = ?1;

-- name: BatchGetUsers :many
SELECT id, username, email, bio, is_admin, created_at, updated_at
FROM users
WHERE id IN (SELECT value FROM json_each(?1))
ORDER BY created_at DESC;

-- name: SoftDeleteUser :exec
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at DATETIME;

UPDATE users
SET deleted_at = datetime('now')
WHERE id = ?1 AND deleted_at IS NULL;

-- name: RestoreUser :exec
UPDATE users
SET deleted_at = NULL
WHERE id = ?1;

-- name: GetActiveUsers :many
SELECT id, username, email, bio, created_at, updated_at
FROM users
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT ?1 OFFSET ?2;

-- name: CountActiveUsers :one
SELECT COUNT(*) FROM users WHERE deleted_at IS NULL;

-- name: GetUserByTokenHashWithToken :one
SELECT 
    u.id, u.username, u.email, u.bio, u.is_admin, u.created_at, u.updated_at,
    t.scope, t.expiry
FROM tokens t
JOIN users u ON u.id = t.user_id
WHERE t.token_hash = ?1;

-- name: SetUserAdmin :exec
UPDATE users SET is_admin = ?1, updated_at = datetime('now') WHERE id = ?2;