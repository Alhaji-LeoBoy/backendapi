package tests

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	dbpkg "femProjectSqlc/internal/db"
	"femProjectSqlc/internal/store"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var requestSessionCols = []string{
	"id",
	"user_id",
	"session_token_hash",
	"ip_address",
	"user_agent",
	"expires_at",
	"last_seen_at",
	"created_at",
	"revoked_at",
}

func TestRequestSessionStore_CreateRequestSession(t *testing.T) {
	s, db, mock := newRequestSessionStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	expiresAt := now.Add(30 * time.Minute)
	tokenHash := []byte("access-hash")
	ip := sql.NullString{String: "127.0.0.1", Valid: true}
	ua := sql.NullString{String: "curl/8.0", Valid: true}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO request_sessions")).
		WithArgs(int64(1), tokenHash, ip, ua, expiresAt).
		WillReturnRows(sqlmock.NewRows(requestSessionCols).
			AddRow(1, int64(1), tokenHash, ip.String, ua.String, expiresAt, now, now, nil))
	mock.ExpectCommit()

	session, err := s.CreateRequestSession(context.Background(), store.CreateRequestSessionParams{
		UserID:           1,
		SessionTokenHash: tokenHash,
		IPAddress:        ip,
		UserAgent:        ua,
		ExpiresAt:        expiresAt,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), session.ID)
	assert.Equal(t, int64(1), session.UserID)
	assert.Equal(t, tokenHash, session.SessionTokenHash)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRequestSessionStore_GetRequestSessionByTokenHash(t *testing.T) {
	s, db, mock := newRequestSessionStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	expiresAt := now.Add(30 * time.Minute)
	tokenHash := []byte("access-hash")

	mock.ExpectQuery(regexp.QuoteMeta("FROM request_sessions WHERE session_token_hash = ?1")).
		WithArgs(tokenHash).
		WillReturnRows(sqlmock.NewRows(requestSessionCols).
			AddRow(1, int64(1), tokenHash, "127.0.0.1", "curl/8.0", expiresAt, now, now, nil))

	session, err := s.GetRequestSessionByTokenHash(context.Background(), tokenHash)
	require.NoError(t, err)
	assert.Equal(t, int64(1), session.ID)
	assert.Equal(t, int64(1), session.UserID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRequestSessionStore_ListActiveRequestSessionsForUser(t *testing.T) {
	s, db, mock := newRequestSessionStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	expiresAt := now.Add(30 * time.Minute)
	tokenHash1 := []byte("access-hash-1")
	tokenHash2 := []byte("access-hash-2")

	mock.ExpectQuery(regexp.QuoteMeta("FROM request_sessions")).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows(requestSessionCols).
			AddRow(1, int64(1), tokenHash1, "127.0.0.1", "curl/8.0", expiresAt, now, now, nil).
			AddRow(2, int64(1), tokenHash2, "127.0.0.1", "curl/8.0", expiresAt, now, now, nil))

	sessions, err := s.ListActiveRequestSessionsForUser(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, sessions, 2)
	assert.Equal(t, int64(1), sessions[0].UserID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRequestSessionStore_TouchRequestSession(t *testing.T) {
	s, db, mock := newRequestSessionStore(t)
	t.Cleanup(func() { db.Close() })

	tokenHash := []byte("access-hash")

	mock.ExpectExec(regexp.QuoteMeta("UPDATE request_sessions SET last_seen_at = datetime('now')")).
		WithArgs(tokenHash).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.TouchRequestSession(context.Background(), tokenHash)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRequestSessionStore_RevokeAndDelete(t *testing.T) {
	s, db, mock := newRequestSessionStore(t)
	t.Cleanup(func() { db.Close() })

	tokenHash := []byte("access-hash")

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE request_sessions SET revoked_at = datetime('now') WHERE session_token_hash = ?1")).
		WithArgs(tokenHash).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.RevokeRequestSessionByTokenHash(context.Background(), tokenHash)
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE request_sessions SET revoked_at = datetime('now') WHERE user_id = ?1")).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	err = s.RevokeAllRequestSessionsForUser(context.Background(), 1)
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM request_sessions WHERE expires_at < datetime('now') OR revoked_at IS NOT NULL")).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	err = s.DeleteExpiredOrRevokedRequestSessions(context.Background())
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// compile guard for generated params changes
var _ dbpkg.RequestSession
