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

var eventCols = []string{"id", "user_id", "title", "owner_name", "description", "location", "start_time", "image_url", "price", "created_at", "updated_at"}

func TestEventStore_CreateEvent(t *testing.T) {
	s, db, mock := newEventStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	params := dbpkg.CreateEventParams{
		UserID:      1,
		Title:       "Go Meetup",
		OwnerName:   "alice",
		Description: "Monthly Go meetup",
		Location:    "Tech Hub",
		StartTime:   now.Add(7 * 24 * time.Hour),
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO events")).
		WithArgs(params.UserID, params.Title, params.OwnerName, params.Description, params.Location, params.StartTime, params.ImageUrl, params.Price).
		WillReturnRows(sqlmock.NewRows(eventCols).
			AddRow(1, params.UserID, params.Title, params.OwnerName, params.Description, params.Location, params.StartTime, "", 0.0, now, now))
	mock.ExpectCommit()

	event, err := s.CreateEvent(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(1), event.ID)
	assert.Equal(t, params.Title, event.Title)
	assert.Equal(t, params.OwnerName, event.OwnerName)
	assert.Equal(t, params.Location, event.Location)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEventStore_GetEvent(t *testing.T) {
	s, db, mock := newEventStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("FROM events WHERE id = ?")).
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows(eventCols).
			AddRow(5, 1, "Go Meetup", "alice", "Monthly Go meetup", "Tech Hub", now, "", 0.0, now, now))

	event, err := s.GetEvent(context.Background(), 5)
	require.NoError(t, err)
	assert.Equal(t, int64(5), event.ID)
	assert.Equal(t, "Go Meetup", event.Title)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEventStore_GetEventWithOwner(t *testing.T) {
	s, db, mock := newEventStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	ownerCols := append(eventCols, "owner_username", "owner_email")
	mock.ExpectQuery(regexp.QuoteMeta("JOIN users u ON u.id = e.user_id")).
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows(ownerCols).
			AddRow(5, 1, "Go Meetup", "alice", "Monthly Go meetup", "Tech Hub", now, "", 0.0, now, now, "alice", "alice@example.com"))

	row, err := s.GetEventWithOwner(context.Background(), 5)
	require.NoError(t, err)
	assert.Equal(t, int64(5), row.ID)
	assert.Equal(t, "alice", row.OwnerUsername)
	assert.Equal(t, "alice@example.com", row.OwnerEmail)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEventStore_GetUserEvents(t *testing.T) {
	s, db, mock := newEventStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("FROM events WHERE user_id = ?")).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows(eventCols).
			AddRow(1, 1, "Event A", "alice", "desc", "loc", now, "", 0.0, now, now).
			AddRow(2, 1, "Event B", "alice", "desc2", "loc2", now, "", 0.0, now, now))

	events, err := s.GetUserEvents(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, "Event A", events[0].Title)
	assert.Equal(t, "Event B", events[1].Title)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEventStore_ListEvents(t *testing.T) {
	s, db, mock := newEventStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("FROM events ORDER BY start_time")).
		WithArgs(int64(0), int64(10)).
		WillReturnRows(sqlmock.NewRows(eventCols).
			AddRow(1, 1, "Event A", "alice", "desc", "loc", now, "", 0.0, now, now))

	events, err := s.ListEvents(context.Background(), dbpkg.ListEventsParams{Offset: 0, Limit: 10})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "Event A", events[0].Title)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEventStore_CountEvents(t *testing.T) {
	s, db, mock := newEventStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(*) FROM events")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))

	count, err := s.CountEvents(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(15), count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEventStore_ListEventsBetween(t *testing.T) {
	s, db, mock := newEventStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	from := now
	to := now.Add(30 * 24 * time.Hour)

	mock.ExpectQuery(regexp.QuoteMeta("WHERE start_time BETWEEN")).
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows(eventCols).
			AddRow(1, 1, "Event A", "alice", "desc", "loc", now, "", 0.0, now, now))

	events, err := s.ListEventsBetween(context.Background(), dbpkg.ListEventsBetweenParams{FromStartTime: from, ToStartTime: to})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEventStore_ListEventsWithTicketStats(t *testing.T) {
	s, db, mock := newEventStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	statsCols := append(eventCols, "tickets_count", "revenue")
	mock.ExpectQuery(regexp.QuoteMeta("LEFT JOIN tickets")).
		WillReturnRows(sqlmock.NewRows(statsCols).
			AddRow(1, 1, "Event A", "alice", "desc", "loc", now, "", 0.0, now, now, 10, 499.90))

	rows, err := s.ListEventsWithTicketStats(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(10), rows[0].TicketsCount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEventStore_PatchEvent(t *testing.T) {
	s, db, mock := newEventStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	params := dbpkg.PatchEventParams{
		Title:       sql.NullString{String: "Updated Title", Valid: true},
		OwnerName:   sql.NullString{},
		Description: sql.NullString{},
		Location:    sql.NullString{},
		StartTime:   sql.NullTime{},
		ImageUrl:    sql.NullString{},
		ID:          5,
		UserID:      1,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("UPDATE events")).
		WithArgs(params.Title, params.OwnerName, params.Description, params.Location, params.StartTime, params.ImageUrl, params.Price, int64(5), int64(1)).
		WillReturnRows(sqlmock.NewRows(eventCols).
			AddRow(5, 1, "Updated Title", "alice", "desc", "loc", now, "", 0.0, now, now))
	mock.ExpectCommit()

	event, err := s.PatchEvent(context.Background(), params)
	require.NoError(t, err)
	assert.Equal(t, int64(5), event.ID)
	assert.Equal(t, "Updated Title", event.Title)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEventStore_SearchEvents(t *testing.T) {
	s, db, mock := newEventStore(t)
	t.Cleanup(func() { db.Close() })

	now := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("WHERE title LIKE")).
		WithArgs("%Go%", "%Go%", int64(10), int64(0)).
		WillReturnRows(sqlmock.NewRows(eventCols).
			AddRow(1, 1, "Go Meetup", "alice", "desc", "loc", now, "", 0.0, now, now))

	events, err := s.SearchEvents(context.Background(), dbpkg.SearchEventsParams{
		Title: "%Go%", Location: "%Go%", Limit: 10, Offset: 0,
	})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "Go Meetup", events[0].Title)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEventStore_DeleteEvent(t *testing.T) {
	s, db, mock := newEventStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM events WHERE id = ? AND user_id = ?")).
		WithArgs(int64(5), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.DeleteEvent(context.Background(), 5, 1)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEventStore_AdminDeleteEvent(t *testing.T) {
	s, db, mock := newEventStore(t)
	t.Cleanup(func() { db.Close() })

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM events WHERE id = ?")).
		WithArgs(int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := s.AdminDeleteEvent(context.Background(), 5)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
