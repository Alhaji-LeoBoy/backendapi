package store

import (
	"context"
	"database/sql"
	db "femProjectSqlc/internal/db"
)

type TransactionStore struct {
	db      *sql.DB
	queries *db.Queries
}

func NewTransactionStore(database *sql.DB, queries *db.Queries) *TransactionStore {
	return &TransactionStore{
		db:      database,
		queries: queries,
	}
}

type TransactionStoreInterface interface {
	CreateTransaction(ctx context.Context, arg db.CreateTransactionParams) (*db.Transaction, error)
	GetTransaction(ctx context.Context, id int64) (*db.Transaction, error)
	GetTransactionByTxnID(ctx context.Context, txnID string) (*db.Transaction, error)
	GetUserTransactions(ctx context.Context, userID int64) ([]db.GetUserTransactionsRow, error)
	UpdateTransactionTicketID(ctx context.Context, arg db.UpdateTransactionTicketIDParams) error
}

func (s *TransactionStore) CreateTransaction(ctx context.Context, arg db.CreateTransactionParams) (*db.Transaction, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	transaction, err := qtx.CreateTransaction(ctx, arg)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (s *TransactionStore) GetTransaction(ctx context.Context, id int64) (*db.Transaction, error) {
	row, err := s.queries.GetTransaction(ctx, id)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *TransactionStore) GetTransactionByTxnID(ctx context.Context, txnID string) (*db.Transaction, error) {
	row, err := s.queries.GetTransactionByTxnID(ctx, txnID)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *TransactionStore) GetUserTransactions(ctx context.Context, userID int64) ([]db.GetUserTransactionsRow, error) {
	return s.queries.GetUserTransactions(ctx, userID)
}

func (s *TransactionStore) UpdateTransactionTicketID(ctx context.Context, arg db.UpdateTransactionTicketIDParams) error {
	return s.queries.UpdateTransactionTicketID(ctx, arg)
}
