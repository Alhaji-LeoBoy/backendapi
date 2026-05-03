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
)

var ticketCols = []string{"id", "user_id", "event_id", "name", "signature", "price", "status", "created_at", "updated_at"}

func TestTicketStore_CreateTicket(t *testing.T) {
	s, db, mock := newTicketStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	params := dbpkg.CreateTicketParams{
		UserID:    1,
		EventID:   2,
		Name:      "Alice - Go Meetup",
		Signature: "abc123sig",
		Price:     49.99,
		Status:    "active",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO tickets")).
		WithArgs(params.UserID, params.EventID, params.Name, params.Signature, params.Price, params.Status).
		WillReturnRows(sqlmock.NewRows(ticketCols).
			AddRow(1, params.UserID, params.EventID, params.Name, params.Signature, params.Price, params.Status, now, now))
	mock.ExpectCommit()

	ticket, err := s.CreateTicket(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(1), ticket.ID)
	assert.Equal(t, params.Name, ticket.Name)
	assert.Equal(t, params.Signature, ticket.Signature)
	assert.Equal(t, params.Price, ticket.Price)
	assert.Equal(t, "active", ticket.Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketStore_GetTicket(t *testing.T) {
	s, db, mock := newTicketStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, user_id, event_id, name, signature, price, status, created_at, updated_at FROM tickets WHERE id = ?")).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows(ticketCols).
			AddRow(7, 1, 2, "Bob - Concert", "sig456", 75.00, "active", now, now))

	ticket, err := s.GetTicket(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, int64(7), ticket.ID)
	assert.Equal(t, "Bob - Concert", ticket.Name)
	assert.Equal(t, 75.00, ticket.Price)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketStore_GetTicketDetailed(t *testing.T) {
	s, db, mock := newTicketStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	detailedCols := []string{
		"id", "user_id", "event_id", "name", "signature", "price", "status", "created_at", "updated_at",
		"event_title", "event_start_time", "event_image_url", "user_username", "user_email",
	}
	mock.ExpectQuery(regexp.QuoteMeta("JOIN events e ON e.id = t.event_id")).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows(detailedCols).
			AddRow(7, 1, 2, "Bob - Concert", "sig456", 75.00, "active", now, now, "Concert Night", now, "https://example.com/img.jpg", "bob", "bob@example.com"))

	row, err := s.GetTicketDetailed(context.Background(), 7)
	require.NoError(t, err)
	assert.Equal(t, int64(7), row.ID)
	assert.Equal(t, "Concert Night", row.EventTitle)
	assert.Equal(t, "bob", row.UserUsername)
	assert.Equal(t, "bob@example.com", row.UserEmail)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketStore_GetUserTickets(t *testing.T) {
	s, db, mock := newTicketStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("FROM tickets WHERE user_id = ?")).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows(ticketCols).
			AddRow(1, 1, 2, "Ticket A", "sig1", 50.0, "active", now, now).
			AddRow(2, 1, 3, "Ticket B", "sig2", 30.0, "used", now, now))

	tickets, err := s.GetUserTickets(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, tickets, 2)
	assert.Equal(t, "Ticket A", tickets[0].Name)
	assert.Equal(t, "Ticket B", tickets[1].Name)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketStore_GetEventTickets(t *testing.T) {
	s, db, mock := newTicketStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("FROM tickets WHERE event_id = ?")).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows(ticketCols).
			AddRow(1, 1, 2, "Ticket A", "sig1", 50.0, "active", now, now).
			AddRow(3, 4, 2, "Ticket C", "sig3", 50.0, "active", now, now))

	tickets, err := s.GetEventTickets(context.Background(), 2)
	require.NoError(t, err)
	require.Len(t, tickets, 2)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketStore_ListTickets(t *testing.T) {
	s, db, mock := newTicketStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("FROM tickets ORDER BY created_at DESC")).
		WithArgs(int64(0), int64(10)).
		WillReturnRows(sqlmock.NewRows(ticketCols).
			AddRow(1, 1, 2, "Ticket A", "sig1", 50.0, "active", now, now))

	tickets, err := s.ListTickets(context.Background(), dbpkg.ListTicketsParams{Offset: 0, Limit: 10})
	require.NoError(t, err)
	require.Len(t, tickets, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketStore_CountTickets(t *testing.T) {
	s, db, mock := newTicketStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM tickets")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

	count, err := s.CountTickets(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(100), count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketStore_ListTicketsBetween(t *testing.T) {
	s, db, mock := newTicketStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	from := now
	to := now.Add(30 * 24 * time.Hour)

	mock.ExpectQuery(regexp.QuoteMeta("WHERE created_at BETWEEN")).
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows(ticketCols).
			AddRow(1, 1, 2, "Ticket A", "sig1", 50.0, "active", now, now))

	tickets, err := s.ListTicketsBetween(context.Background(), from, to)
	require.NoError(t, err)
	require.Len(t, tickets, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketStore_ListTicketsByStatusForEvent(t *testing.T) {
	s, db, mock := newTicketStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("WHERE event_id = ? AND status = ?")).
		WithArgs(int64(2), "active").
		WillReturnRows(sqlmock.NewRows(ticketCols).
			AddRow(1, 1, 2, "Ticket A", "sig1", 50.0, "active", now, now))

	tickets, err := s.ListTicketsByStatusForEvent(context.Background(), dbpkg.ListTicketsByStatusForEventParams{
		EventID: 2, Status: "active",
	})
	require.NoError(t, err)
	require.Len(t, tickets, 1)
	assert.Equal(t, "active", tickets[0].Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketStore_ListTicketsForEventWithUser(t *testing.T) {
	s, db, mock := newTicketStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	withUserCols := append(ticketCols, "user_username", "user_email")
	mock.ExpectQuery(regexp.QuoteMeta("JOIN users u ON u.id = t.user_id")).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows(withUserCols).
			AddRow(1, 1, 2, "Ticket A", "sig1", 50.0, "active", now, now, "alice", "alice@example.com"))

	rows, err := s.ListTicketsForEventWithUser(context.Background(), 2)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "alice", rows[0].UserUsername)
	assert.Equal(t, "alice@example.com", rows[0].UserEmail)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketStore_PatchTicket(t *testing.T) {
	s, db, mock := newTicketStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	params := dbpkg.PatchTicketParams{
		Name:   sql.NullString{String: "Updated Name", Valid: true},
		Price:  sql.NullFloat64{},
		Status: sql.NullString{String: "used", Valid: true},
		ID:     7,
		UserID: 1,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE tickets")).
		WithArgs(params.Name, params.Price, params.Status, int64(7), int64(1)).
		WillReturnRows(sqlmock.NewRows(ticketCols).
			AddRow(7, 1, 2, "Updated Name", "sig456", 75.00, "used", now, now))
	mock.ExpectCommit()

	ticket, err := s.PatchTicket(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(7), ticket.ID)
	assert.Equal(t, "Updated Name", ticket.Name)
	assert.Equal(t, "used", ticket.Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketStore_DeleteTicket(t *testing.T) {
	s, db, mock := newTicketStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tickets WHERE id = ? AND user_id = ?")).
		WithArgs(int64(7), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.DeleteTicket(context.Background(), 7, 1)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTicketStore_AdminDeleteTicket(t *testing.T) {
	s, db, mock := newTicketStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM tickets WHERE id = ?")).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.AdminDeleteTicket(context.Background(), 7)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
