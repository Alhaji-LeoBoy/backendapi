package store

import (
	"context"
	"database/sql"
	db "femProjectSqlc/internal/db"
	"time"
)

type RequestSessionStore struct {
	db      *sql.DB
	queries *db.Queries
}

func NewRequestSessionStore(database *sql.DB, queries *db.Queries) *RequestSessionStore {
	return &RequestSessionStore{
		db:      database,
		queries: queries,
	}
}

type CreateRequestSessionParams struct {
	UserID           int64
	SessionTokenHash []byte
	IPAddress        sql.NullString
	UserAgent        sql.NullString
	ExpiresAt        time.Time
}

type RequestSessionStoreInterface interface {
	CreateRequestSession(ctx context.Context, arg CreateRequestSessionParams) (*db.RequestSession, error)
	GetRequestSessionByTokenHash(ctx context.Context, tokenHash []byte) (*db.RequestSession, error)
	ListActiveRequestSessionsForUser(ctx context.Context, userID int64) ([]db.RequestSession, error)
	TouchRequestSession(ctx context.Context, tokenHash []byte) error
	RevokeRequestSessionByTokenHash(ctx context.Context, tokenHash []byte) error
	RevokeAllRequestSessionsForUser(ctx context.Context, userID int64) error
	DeleteExpiredOrRevokedRequestSessions(ctx context.Context) error
}

func (s *RequestSessionStore) CreateRequestSession(ctx context.Context, arg CreateRequestSessionParams) (*db.RequestSession, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	session, err := qtx.CreateRequestSession(ctx, db.CreateRequestSessionParams{
		UserID:           arg.UserID,
		SessionTokenHash: arg.SessionTokenHash,
		IpAddress:        arg.IPAddress,
		UserAgent:        arg.UserAgent,
		ExpiresAt:        arg.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *RequestSessionStore) GetRequestSessionByTokenHash(ctx context.Context, tokenHash []byte) (*db.RequestSession, error) {
	session, err := s.queries.GetRequestSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *RequestSessionStore) ListActiveRequestSessionsForUser(ctx context.Context, userID int64) ([]db.RequestSession, error) {
	return s.queries.ListActiveRequestSessionsForUser(ctx, userID)
}

func (s *RequestSessionStore) TouchRequestSession(ctx context.Context, tokenHash []byte) error {
	return s.queries.TouchRequestSession(ctx, tokenHash)
}

func (s *RequestSessionStore) RevokeRequestSessionByTokenHash(ctx context.Context, tokenHash []byte) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.RevokeRequestSessionByTokenHash(ctx, tokenHash); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *RequestSessionStore) RevokeAllRequestSessionsForUser(ctx context.Context, userID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.RevokeAllRequestSessionsForUser(ctx, userID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *RequestSessionStore) DeleteExpiredOrRevokedRequestSessions(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.DeleteExpiredOrRevokedRequestSessions(ctx); err != nil {
		return err
	}

	return tx.Commit()
}
