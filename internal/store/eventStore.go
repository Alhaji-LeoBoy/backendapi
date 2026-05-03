package store

import (
	"context"
	"database/sql"
	"femProjectSqlc/internal/db"
	"time"
)

type EventStore struct {
	db      *sql.DB
	queries *db.Queries
}

func NewEventStore(db *sql.DB, queries *db.Queries) *EventStore {
	return &EventStore{
		db:      db,
		queries: queries,
	}
}

type EventStoreInterface interface {
	CreateEvent(ctx context.Context, arg db.CreateEventParams) (*db.Event, error)
	GetEvent(ctx context.Context, id int64) (*db.Event, error)
	GetEventWithOwner(ctx context.Context, id int64) (*db.GetEventWithOwnerRow, error)
	GetUserEvents(ctx context.Context, userID int64) ([]db.Event, error)
	GetUserEventsWithStats(ctx context.Context, userID int64) ([]db.GetUserEventsWithStatsRow, error)
	ListEvents(ctx context.Context, arg db.ListEventsParams) ([]db.Event, error)
	CountEvents(ctx context.Context) (int64, error)
	ListEventsBetween(ctx context.Context, arg db.ListEventsBetweenParams) ([]db.Event, error)
	ListEventsWithTicketStats(ctx context.Context) ([]db.ListEventsWithTicketStatsRow, error)
	PatchEvent(ctx context.Context, arg db.PatchEventParams) (*db.Event, error)
	SearchEvents(ctx context.Context, arg db.SearchEventsParams) ([]db.Event, error)
	DeleteEvent(ctx context.Context, id int64, userID int64) error
	AdminDeleteEvent(ctx context.Context, id int64) error
}

func (s *EventStore) CreateEvent(ctx context.Context, arg db.CreateEventParams) (*db.Event, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	event, err := qtx.CreateEvent(ctx, arg)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *EventStore) GetEvent(ctx context.Context, id int64) (*db.Event, error) {
	event, err := s.queries.GetEvent(ctx, id)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *EventStore) GetEventWithOwner(ctx context.Context, id int64) (*db.GetEventWithOwnerRow, error) {
	row, err := s.queries.GetEventWithOwner(ctx, id)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *EventStore) GetUserEvents(ctx context.Context, userID int64) ([]db.Event, error) {
	return s.queries.GetUserEvents(ctx, userID)
}

func (s *EventStore) GetUserEventsWithStats(ctx context.Context, userID int64) ([]db.GetUserEventsWithStatsRow, error) {
	return s.queries.GetUserEventsWithStats(ctx, userID)
}

func (s *EventStore) ListEvents(ctx context.Context, arg db.ListEventsParams) ([]db.Event, error) {
	return s.queries.ListEvents(ctx, arg)
}

func (s *EventStore) CountEvents(ctx context.Context) (int64, error) {
	return s.queries.CountEvents(ctx)
}

func (s *EventStore) ListEventsBetween(ctx context.Context, arg db.ListEventsBetweenParams) ([]db.Event, error) {
	return s.queries.ListEventsBetween(ctx, arg)
}

func (s *EventStore) ListEventsWithTicketStats(ctx context.Context) ([]db.ListEventsWithTicketStatsRow, error) {
	return s.queries.ListEventsWithTicketStats(ctx)
}

func (s *EventStore) PatchEvent(ctx context.Context, arg db.PatchEventParams) (*db.Event, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	event, err := qtx.PatchEvent(ctx, arg)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *EventStore) SearchEvents(ctx context.Context, arg db.SearchEventsParams) ([]db.Event, error) {
	return s.queries.SearchEvents(ctx, arg)
}

func (s *EventStore) DeleteEvent(ctx context.Context, id int64, userID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.DeleteEvent(ctx, db.DeleteEventParams{
		ID:     id,
		UserID: userID,
	}); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *EventStore) AdminDeleteEvent(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.AdminDeleteEvent(ctx, id); err != nil {
		return err
	}

	return tx.Commit()
}

// Helper for handler use
func (s *EventStore) ListEventsBetweenDates(ctx context.Context, from, to time.Time) ([]db.Event, error) {
	return s.queries.ListEventsBetween(ctx, db.ListEventsBetweenParams{
		FromStartTime: from,
		ToStartTime:   to,
	})
}
