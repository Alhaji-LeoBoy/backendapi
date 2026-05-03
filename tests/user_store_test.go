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

	dbpkg "femProjectSqlc/internal/db"
	"femProjectSqlc/internal/store"
)

var userCols = []string{"id", "username", "email", "password", "bio", "is_admin", "created_at", "updated_at"}

func TestUserStore_CreateUser(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	params := store.CreateUserParams{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "secret",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO users (username, email, password, bio)")).
		WithArgs(params.Username, params.Email, params.Password, sql.NullString{}).
		WillReturnRows(sqlmock.NewRows(userCols).
			AddRow(1, params.Username, params.Email, params.Password, sql.NullString{}, false, now, now))
	mock.ExpectCommit()

	user, err := s.CreateUser(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(1), user.ID)
	assert.Equal(t, params.Username, user.Username)
	assert.Equal(t, params.Email, user.Email)
	assert.Equal(t, params.Password, user.Password)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_GetUserByID(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 11, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, email, password, bio, is_admin, created_at, updated_at FROM users WHERE id = ?1")).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows(userCols).
			AddRow(7, "bob", "bob@example.com", "pw", sql.NullString{String: "hi", Valid: true}, false, now, now))

	user, err := s.GetUserByID(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, int64(7), user.ID)
	assert.Equal(t, "bob", user.Username)
	assert.Equal(t, "bob@example.com", user.Email)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_GetUserByEmail(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 11, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, email, password, bio, is_admin, created_at, updated_at FROM users WHERE email = ?1")).
		WithArgs("alice@example.com").
		WillReturnRows(sqlmock.NewRows(userCols).
			AddRow(3, "alice", "alice@example.com", "pw", sql.NullString{}, false, now, now))

	user, err := s.GetUserByEmail(context.Background(), "alice@example.com")
	require.NoError(t, err)
	assert.Equal(t, int64(3), user.ID)
	assert.Equal(t, "alice@example.com", user.Email)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_GetUserByUsername(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 11, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, email, password, bio, is_admin, created_at, updated_at FROM users WHERE username = ?1")).
		WithArgs("alice").
		WillReturnRows(sqlmock.NewRows(userCols).
			AddRow(3, "alice", "alice@example.com", "pw", sql.NullString{}, false, now, now))

	user, err := s.GetUserByUsername(context.Background(), "alice")
	require.NoError(t, err)
	assert.Equal(t, int64(3), user.ID)
	assert.Equal(t, "alice", user.Username)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_GetUserByUsernameOrEmail(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 11, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, email, password, bio, is_admin, created_at, updated_at FROM users WHERE username = ?1 OR email = ?2")).
		WithArgs("alice", "alice@example.com").
		WillReturnRows(sqlmock.NewRows(userCols).
			AddRow(3, "alice", "alice@example.com", "pw", sql.NullString{}, false, now, now))

	user, err := s.GetUserByUsernameOrEmail(context.Background(), "alice", "alice@example.com")
	require.NoError(t, err)
	assert.Equal(t, int64(3), user.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_ListUsers(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, email, password, bio, is_admin, created_at, updated_at FROM users ORDER BY created_at DESC")).
		WithArgs(int64(10), int64(0)).
		WillReturnRows(sqlmock.NewRows(userCols).
			AddRow(1, "alice", "a@example.com", "pw1", sql.NullString{}, false, now, now).
			AddRow(2, "bob", "b@example.com", "pw2", sql.NullString{}, false, now, now))

	users, err := s.ListUsers(context.Background(), 10, 0)
	require.NoError(t, err)
	require.Len(t, users, 2)
	assert.Equal(t, int64(1), users[0].ID)
	assert.Equal(t, int64(2), users[1].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_CountUsers(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))

	count, err := s.CountUsers(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(42), count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_CountActiveUsers(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM users WHERE deleted_at IS NULL")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(30))

	count, err := s.CountActiveUsers(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(30), count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_UserExists(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(1))

	exists, err := s.UserExists(context.Background(), 5)
	require.NoError(t, err)
	assert.True(t, exists)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_UsernameExists(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).
		WithArgs("alice").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(1))

	exists, err := s.UsernameExists(context.Background(), "alice")
	require.NoError(t, err)
	assert.True(t, exists)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_EmailExists(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).
		WithArgs("alice@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(0))

	exists, err := s.EmailExists(context.Background(), "alice@example.com")
	require.NoError(t, err)
	assert.False(t, exists)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_PatchUser(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 13, 0, 0, 0, time.UTC)
	params := dbpkg.PatchUserParams{
		Username: sql.NullString{String: "updated", Valid: true},
		Email:    sql.NullString{String: "updated@example.com", Valid: true},
		Password: sql.NullString{},
		Bio:      sql.NullString{},
		ID:       5,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE users")).
		WithArgs(params.Username, params.Email, params.Password, params.Bio, int64(5)).
		WillReturnRows(sqlmock.NewRows(userCols).
			AddRow(5, "updated", "updated@example.com", "pw", sql.NullString{}, false, now, now))
	mock.ExpectCommit()

	user, err := s.PatchUser(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(5), user.ID)
	assert.Equal(t, "updated", user.Username)
	assert.Equal(t, "updated@example.com", user.Email)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_UpdatePassword(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users")).
		WithArgs("newpasshash", int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.UpdatePassword(context.Background(), "newpasshash", 5)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_UpdateEmail(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users")).
		WithArgs("new@example.com", int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.UpdateEmail(context.Background(), "new@example.com", 5)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_DeleteUser(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM users WHERE id = ?1")).
		WithArgs(int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.DeleteUser(context.Background(), 9)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_RestoreUser(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users")).
		WithArgs(int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.RestoreUser(context.Background(), 9)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_SetUserAdmin(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET is_admin")).
		WithArgs(true, int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.SetUserAdmin(context.Background(), true, 5)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_SearchUsers(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("WHERE username LIKE")).
		WithArgs(sql.NullString{String: "ali", Valid: true}, int64(10), int64(0)).
		WillReturnRows(sqlmock.NewRows(userCols).
			AddRow(3, "alice", "alice@example.com", "pw", sql.NullString{}, false, now, now))

	users, err := s.SearchUsers(context.Background(), "ali", 10, 0)
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "alice", users[0].Username)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_GetActiveUsers(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)
	activeCols := []string{"id", "username", "email", "bio", "created_at", "updated_at"}
	mock.ExpectQuery(regexp.QuoteMeta("WHERE deleted_at IS NULL")).
		WithArgs(int64(10), int64(0)).
		WillReturnRows(sqlmock.NewRows(activeCols).
			AddRow(1, "alice", "a@example.com", sql.NullString{}, now, now))

	users, err := s.GetActiveUsers(context.Background(), 10, 0)
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "alice", users[0].Username)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserStore_GetUserWithTokens(t *testing.T) {
	s, db, mock := newUserStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 4, 2, 14, 0, 0, 0, time.UTC)
	tokenCols := []string{
		"id", "username", "email", "password", "bio", "is_admin", "created_at", "updated_at",
		"token_id", "token_hash", "scope", "expiry", "token_created_at",
	}
	mock.ExpectQuery(regexp.QuoteMeta("LEFT JOIN tokens")).
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows(tokenCols).
			AddRow(3, "kate", "k@example.com", "pw", sql.NullString{}, false, now, now,
				sql.NullInt64{Int64: 10, Valid: true}, []byte("hash"), sql.NullString{String: "access", Valid: true}, now, now))

	rows, err := s.GetUserWithTokens(context.Background(), 3)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(3), rows[0].ID)
	assert.Equal(t, "kate", rows[0].Username)
	require.NoError(t, mock.ExpectationsWereMet())
}
