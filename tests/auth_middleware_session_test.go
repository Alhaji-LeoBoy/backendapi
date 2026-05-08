package tests

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"femProjectSqlc/internal/auth"
	dbpkg "femProjectSqlc/internal/db"
	"femProjectSqlc/internal/logger"
	"femProjectSqlc/internal/middleware"
	"femProjectSqlc/internal/store"
	"femProjectSqlc/internal/tokens"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthHandler_Logout_RevokesTokenAndSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	queries := dbpkg.New(db)
	tokenStore := store.NewTokenStore(db, queries)
	requestSessionStore := store.NewRequestSessionStore(db, queries)
	handler := auth.NewHandler(nil, tokenStore, requestSessionStore)

	tokenPlaintext := strings.Repeat("A", 52)
	tokenHash := tokens.HashToken(tokenPlaintext)

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM tokens\s+WHERE token_hash = \?1`).
		WithArgs(tokenHash).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE request_sessions\s+SET revoked_at = datetime\('now'\)\s+WHERE session_token_hash = \?1`).
		WithArgs(tokenHash).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPlaintext)
	rr := httptest.NewRecorder()

	handler.Logout(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Logged out successfully")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMiddleware_ChiAuthenticate_TouchesRequestSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	queries := dbpkg.New(db)
	tokenStore := store.NewTokenStore(db, queries)
	requestSessionStore := store.NewRequestSessionStore(db, queries)
	mw := middleware.New(tokenStore, requestSessionStore, logger.New())

	tokenPlaintext := strings.Repeat("B", 52)
	tokenHash := tokens.HashToken(tokenPlaintext)
	now := time.Now().UTC()
	expiry := now.Add(2 * time.Hour)
	cols := []string{"id", "username", "email", "bio", "is_admin", "created_at", "updated_at", "scope", "expiry"}

	mock.ExpectQuery(regexp.QuoteMeta("JOIN users u ON u.id = t.user_id")).
		WithArgs(tokenHash).
		WillReturnRows(sqlmock.NewRows(cols).
			AddRow(1, "alice", "alice@example.com", nil, false, now, now, tokens.ScopeAccess, expiry))

	mock.ExpectExec(`UPDATE request_sessions\s+SET last_seen_at = datetime\('now'\)`).
		WithArgs(tokenHash).
		WillReturnResult(sqlmock.NewResult(0, 1))

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPlaintext)
	rr := httptest.NewRecorder()

	mw.ChiAuthenticate(next).ServeHTTP(rr, req)

	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusNoContent, rr.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}
