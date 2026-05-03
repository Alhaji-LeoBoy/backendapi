package tests

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var tokenCols = []string{"id", "user_id", "token_hash", "scope", "expiry", "created_at"}

func TestTokenStore_CreateToken(t *testing.T) {
	s, db, mock := newTokenStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	expiry := now.Add(15 * time.Minute)
	hash := []byte("tokenhash123")

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO tokens (user_id, token_hash, expiry, scope)")).
		WithArgs(int64(1), hash, expiry, "access").
		WillReturnRows(sqlmock.NewRows(tokenCols).
			AddRow(1, int64(1), hash, "access", expiry, now))
	mock.ExpectCommit()

	token, err := s.CreateToken(1, hash, expiry, "access")
	require.NoError(t, err)
	assert.Equal(t, int64(1), token.ID)
	assert.Equal(t, int64(1), token.UserID)
	assert.Equal(t, hash, token.TokenHash)
	assert.Equal(t, "access", token.Scope)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenStore_GetTokenByID(t *testing.T) {
	s, db, mock := newTokenStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	expiry := now.Add(15 * time.Minute)
	hash := []byte("tokenhash123")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, token_hash, scope, expiry, created_at FROM tokens WHERE id = ?1")).
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows(tokenCols).
			AddRow(5, int64(1), hash, "access", expiry, now))

	token, err := s.GetTokenByID(5)
	require.NoError(t, err)
	assert.Equal(t, int64(5), token.ID)
	assert.Equal(t, "access", token.Scope)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenStore_GetTokenByHash(t *testing.T) {
	s, db, mock := newTokenStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	expiry := now.Add(15 * time.Minute)
	hash := []byte("tokenhash123")

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, token_hash, scope, expiry, created_at FROM tokens WHERE token_hash = ?1")).
		WithArgs(hash).
		WillReturnRows(sqlmock.NewRows(tokenCols).
			AddRow(5, int64(1), hash, "access", expiry, now))

	token, err := s.GetTokenByHash(context.Background(), hash)
	require.NoError(t, err)
	assert.Equal(t, int64(5), token.ID)
	assert.Equal(t, hash, token.TokenHash)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenStore_GetUserByTokenHash(t *testing.T) {
	s, db, mock := newTokenStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	hash := []byte("tokenhash123")
	userTokenCols := []string{"id", "username", "email", "password", "bio", "is_admin", "created_at", "updated_at"}

	mock.ExpectQuery(regexp.QuoteMeta("INNER JOIN tokens ON users.id = tokens.user_id")).
		WithArgs(hash, "access").
		WillReturnRows(sqlmock.NewRows(userTokenCols).
			AddRow(1, "alice", "alice@example.com", "pw", sql.NullString{}, false, now, now))

	user, err := s.GetUserByTokenHash(context.Background(), hash, "access")
	require.NoError(t, err)
	assert.Equal(t, int64(1), user.ID)
	assert.Equal(t, "alice", user.Username)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenStore_ListTokensForUser(t *testing.T) {
	s, db, mock := newTokenStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	expiry := now.Add(15 * time.Minute)
	hash1 := []byte("hash1")
	hash2 := []byte("hash2")

	mock.ExpectQuery(regexp.QuoteMeta("FROM tokens WHERE user_id = ?1")).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows(tokenCols).
			AddRow(1, int64(1), hash1, "access", expiry, now).
			AddRow(2, int64(1), hash2, "refresh", expiry, now))

	tokens, err := s.ListTokensForUser(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, tokens, 2)
	assert.Equal(t, "access", tokens[0].Scope)
	assert.Equal(t, "refresh", tokens[1].Scope)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenStore_GetValidTokenByUserAndScope(t *testing.T) {
	s, db, mock := newTokenStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	expiry := now.Add(15 * time.Minute)
	hash := []byte("tokenhash123")

	mock.ExpectQuery(regexp.QuoteMeta("WHERE user_id = ?1")).
		WithArgs(int64(1), "access").
		WillReturnRows(sqlmock.NewRows(tokenCols).
			AddRow(5, int64(1), hash, "access", expiry, now))

	token, err := s.GetValidTokenByUserAndScope(context.Background(), 1, "access")
	require.NoError(t, err)
	assert.Equal(t, int64(5), token.ID)
	assert.Equal(t, "access", token.Scope)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenStore_GetUserByTokenHashWithToken(t *testing.T) {
	s, db, mock := newTokenStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	expiry := now.Add(15 * time.Minute)
	hash := []byte("tokenhash123")
	cols := []string{"id", "username", "email", "bio", "is_admin", "created_at", "updated_at", "scope", "expiry"}

	mock.ExpectQuery(regexp.QuoteMeta("JOIN users u ON u.id = t.user_id")).
		WithArgs(hash).
		WillReturnRows(sqlmock.NewRows(cols).
			AddRow(1, "alice", "alice@example.com", sql.NullString{}, false, now, now, "access", expiry))

	user, err := s.GetUserByTokenHashWithToken(context.Background(), hash)
	require.NoError(t, err)
	assert.Equal(t, int64(1), user.ID)
	assert.Equal(t, "alice", user.Username)
	assert.Equal(t, "access", user.Scope)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenStore_DeleteToken(t *testing.T) {
	s, db, mock := newTokenStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tokens WHERE id = ?1")).
		WithArgs(int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.DeleteToken(context.Background(), 5)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenStore_DeleteTokenByHash(t *testing.T) {
	s, db, mock := newTokenStore(t)
	t.Cleanup(func() { db.Close() })

	hash := []byte("tokenhash123")

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tokens WHERE token_hash = ?1")).
		WithArgs(hash).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.DeleteTokenByHash(context.Background(), hash)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenStore_DeleteAllTokensForUser(t *testing.T) {
	s, db, mock := newTokenStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tokens WHERE user_id = ?1")).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.DeleteAllTokensForUser(context.Background(), 1)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenStore_DeleteExpiredTokens(t *testing.T) {
	s, db, mock := newTokenStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tokens WHERE expiry")).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	err := s.DeleteExpiredTokens(context.Background())
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenStore_DeleteTokensByScope(t *testing.T) {
	s, db, mock := newTokenStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tokens WHERE user_id = ?1 AND scope = ?2")).
		WithArgs(int64(1), "refresh").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	err := s.DeleteTokensByScope(context.Background(), 1, "refresh")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
