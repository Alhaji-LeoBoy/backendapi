package store

import (
	"context"
	"database/sql"
	"femProjectSqlc/internal/db"
	"time"
)

type TicketStore struct {
	db      *sql.DB
	queries *db.Queries
}

func NewTicketStore(db *sql.DB, queries *db.Queries) *TicketStore {
	return &TicketStore{
		db:      db,
		queries: queries,
	}
}

type TicketStoreInterface interface {
	CreateTicket(ctx context.Context, arg db.CreateTicketParams) (*db.Ticket, error)
	GetTicket(ctx context.Context, id int64) (*db.Ticket, error)
	GetTicketDetailed(ctx context.Context, id int64) (*db.GetTicketDetailedRow, error)
	GetUserTickets(ctx context.Context, userID int64) ([]db.Ticket, error)
	GetUserTicketsWithEvent(ctx context.Context, userID int64) ([]db.GetUserTicketsWithEventRow, error)
	GetEventTickets(ctx context.Context, eventID int64) ([]db.Ticket, error)
	ListTickets(ctx context.Context, arg db.ListTicketsParams) ([]db.Ticket, error)
	CountTickets(ctx context.Context) (int64, error)
	ListTicketsBetween(ctx context.Context, from, to time.Time) ([]db.Ticket, error)
	ListTicketsByStatusForEvent(ctx context.Context, arg db.ListTicketsByStatusForEventParams) ([]db.Ticket, error)
	ListTicketsForEventWithUser(ctx context.Context, eventID int64) ([]db.ListTicketsForEventWithUserRow, error)
	PatchTicket(ctx context.Context, arg db.PatchTicketParams) (*db.Ticket, error)
	DeleteTicket(ctx context.Context, id int64, userID int64) error
	AdminDeleteTicket(ctx context.Context, id int64) error
}

func (s *TicketStore) CreateTicket(ctx context.Context, arg db.CreateTicketParams) (*db.Ticket, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	ticket, err := qtx.CreateTicket(ctx, arg)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (s *TicketStore) GetTicket(ctx context.Context, id int64) (*db.Ticket, error) {
	ticket, err := s.queries.GetTicket(ctx, id)
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (s *TicketStore) GetTicketDetailed(ctx context.Context, id int64) (*db.GetTicketDetailedRow, error) {
	row, err := s.queries.GetTicketDetailed(ctx, id)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *TicketStore) GetUserTickets(ctx context.Context, userID int64) ([]db.Ticket, error) {
	return s.queries.GetUserTickets(ctx, userID)
}

func (s *TicketStore) GetUserTicketsWithEvent(ctx context.Context, userID int64) ([]db.GetUserTicketsWithEventRow, error) {
	return s.queries.GetUserTicketsWithEvent(ctx, userID)
}

func (s *TicketStore) GetEventTickets(ctx context.Context, eventID int64) ([]db.Ticket, error) {
	return s.queries.GetEventTickets(ctx, eventID)
}

func (s *TicketStore) ListTickets(ctx context.Context, arg db.ListTicketsParams) ([]db.Ticket, error) {
	return s.queries.ListTickets(ctx, arg)
}

func (s *TicketStore) CountTickets(ctx context.Context) (int64, error) {
	return s.queries.CountTickets(ctx)
}

func (s *TicketStore) ListTicketsBetween(ctx context.Context, from, to time.Time) ([]db.Ticket, error) {
	return s.queries.ListTicketsBetween(ctx, db.ListTicketsBetweenParams{
		FromCreatedAt: from,
		ToCreatedAt:   to,
	})
}

func (s *TicketStore) ListTicketsByStatusForEvent(ctx context.Context, arg db.ListTicketsByStatusForEventParams) ([]db.Ticket, error) {
	return s.queries.ListTicketsByStatusForEvent(ctx, arg)
}

func (s *TicketStore) ListTicketsForEventWithUser(ctx context.Context, eventID int64) ([]db.ListTicketsForEventWithUserRow, error) {
	return s.queries.ListTicketsForEventWithUser(ctx, eventID)
}

func (s *TicketStore) PatchTicket(ctx context.Context, arg db.PatchTicketParams) (*db.Ticket, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	ticket, err := qtx.PatchTicket(ctx, arg)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (s *TicketStore) DeleteTicket(ctx context.Context, id int64, userID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.DeleteTicket(ctx, db.DeleteTicketParams{
		ID:     id,
		UserID: userID,
	}); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *TicketStore) AdminDeleteTicket(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.AdminDeleteTicket(ctx, id); err != nil {
		return err
	}

	return tx.Commit()
}
