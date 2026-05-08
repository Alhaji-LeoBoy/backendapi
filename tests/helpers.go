package tests

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	dbpkg "femProjectSqlc/internal/db"
	"femProjectSqlc/internal/store"
)

func newUserStore(t *testing.T) (*store.UserStore, *sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	queries := dbpkg.New(db)
	s := store.NewUserStore(db, queries)
	return s, db, mock
}

func newEventStore(t *testing.T) (*store.EventStore, *sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	queries := dbpkg.New(db)
	s := store.NewEventStore(db, queries)
	return s, db, mock
}

func newTicketStore(t *testing.T) (*store.TicketStore, *sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	queries := dbpkg.New(db)
	s := store.NewTicketStore(db, queries)
	return s, db, mock
}

func newTokenStore(t *testing.T) (*store.TokenStore, *sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	queries := dbpkg.New(db)
	s := store.NewTokenStore(db, queries)
	return s, db, mock
}

func newRequestSessionStore(t *testing.T) (*store.RequestSessionStore, *sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	queries := dbpkg.New(db)
	s := store.NewRequestSessionStore(db, queries)
	return s, db, mock
}
