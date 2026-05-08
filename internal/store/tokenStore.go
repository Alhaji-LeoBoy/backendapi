package store

import (
	"context"
	"database/sql"
	"femProjectSqlc/internal/db"
	"time"
)

type TokenStore struct {
	db      *sql.DB
	queries *db.Queries
}

func NewTokenStore(db *sql.DB, queries *db.Queries) *TokenStore {
	return &TokenStore{
		db:      db,
		queries: queries,
	}
}

type TokenStoreInterface interface {
	CreateToken(userID int64, tokenHash []byte, expiry time.Time, scope string) (db.CreateTokenRow, error)
	GetTokenByID(id int64) (db.GetTokenByIDRow, error)
	GetTokenByHash(ctx context.Context, tokenHash []byte) (db.GetTokenByHashRow, error)
	ConsumeValidRefreshTokenByHash(ctx context.Context, tokenHash []byte) (db.ConsumeValidRefreshTokenByHashRow, error)
	GetUserByTokenHash(ctx context.Context, tokenHash []byte, scope string) (db.GetUserByTokenHashRow, error)
	ListTokensForUser(ctx context.Context, userID int64) ([]db.ListTokensForUserRow, error)
	DeleteToken(ctx context.Context, id int64) error
	DeleteTokenByHash(ctx context.Context, tokenHash []byte) error
	DeleteAllTokensForUser(ctx context.Context, userID int64) error
	DeleteExpiredTokens(ctx context.Context) error
	DeleteTokensByScope(ctx context.Context, userID int64, scope string) error
	GetValidTokenByUserAndScope(ctx context.Context, userID int64, scope string) (db.GetValidTokenByUserAndScopeRow, error)
	GetUserByTokenHashWithToken(ctx context.Context, tokenHash []byte) (db.GetUserByTokenHashWithTokenRow, error)
}

// ==================== Token CRUD ====================

func (s *TokenStore) CreateToken(userID int64, tokenHash []byte, expiry time.Time, scope string) (db.CreateTokenRow, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return db.CreateTokenRow{}, err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	createdToken, err := qtx.CreateToken(context.Background(), db.CreateTokenParams{
		UserID:    userID,
		TokenHash: tokenHash,
		Expiry:    expiry,
		Scope:     scope,
	})
	if err != nil {
		return db.CreateTokenRow{}, err
	}

	if err := tx.Commit(); err != nil {
		return db.CreateTokenRow{}, err
	}
	return createdToken, nil
}

func (s *TokenStore) GetTokenByID(id int64) (db.GetTokenByIDRow, error) {
	token, err := s.queries.GetTokenByID(context.Background(), id)
	if err != nil {
		return db.GetTokenByIDRow{}, err
	}
	return token, nil
}

func (s *TokenStore) GetTokenByHash(ctx context.Context, tokenHash []byte) (db.GetTokenByHashRow, error) {
	token, err := s.queries.GetTokenByHash(ctx, tokenHash)
	if err != nil {
		return db.GetTokenByHashRow{}, err
	}
	return token, nil
}

func (s *TokenStore) GetUserByTokenHash(ctx context.Context, tokenHash []byte, scope string) (db.GetUserByTokenHashRow, error) {
	user, err := s.queries.GetUserByTokenHash(ctx, db.GetUserByTokenHashParams{
		TokenHash: tokenHash,
		Scope:     scope,
	})
	if err != nil {
		return db.GetUserByTokenHashRow{}, err
	}
	return user, nil
}

func (s *TokenStore) ListTokensForUser(ctx context.Context, userID int64) ([]db.ListTokensForUserRow, error) {
	tokens, err := s.queries.ListTokensForUser(ctx, userID)
	if err != nil {
		return []db.ListTokensForUserRow{}, err
	}
	return tokens, nil
}

func (s *TokenStore) DeleteToken(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.DeleteToken(ctx, id); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *TokenStore) DeleteTokenByHash(ctx context.Context, tokenHash []byte) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.DeleteTokenByHash(ctx, tokenHash); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *TokenStore) DeleteAllTokensForUser(ctx context.Context, userID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.DeleteAllTokensForUser(ctx, userID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *TokenStore) DeleteExpiredTokens(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.DeleteExpiredTokens(ctx); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *TokenStore) DeleteTokensByScope(ctx context.Context, userID int64, scope string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.DeleteTokensByScope(ctx, db.DeleteTokensByScopeParams{
		UserID: userID,
		Scope:  scope,
	}); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *TokenStore) GetValidTokenByUserAndScope(ctx context.Context, userID int64, scope string) (db.GetValidTokenByUserAndScopeRow, error) {
	token, err := s.queries.GetValidTokenByUserAndScope(ctx, db.GetValidTokenByUserAndScopeParams{
		UserID: userID,
		Scope:  scope,
	})
	if err != nil {
		return db.GetValidTokenByUserAndScopeRow{}, err
	}
	return token, nil
}

func (s *TokenStore) ConsumeValidRefreshTokenByHash(ctx context.Context, tokenHash []byte) (db.ConsumeValidRefreshTokenByHashRow, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return db.ConsumeValidRefreshTokenByHashRow{}, err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	token, err := qtx.ConsumeValidRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return db.ConsumeValidRefreshTokenByHashRow{}, err
	}

	if err := tx.Commit(); err != nil {
		return db.ConsumeValidRefreshTokenByHashRow{}, err
	}
	return token, nil
}

func (s *TokenStore) GetUserByTokenHashWithToken(ctx context.Context, tokenHash []byte) (db.GetUserByTokenHashWithTokenRow, error) {
	user, err := s.queries.GetUserByTokenHashWithToken(ctx, tokenHash)
	if err != nil {
		return db.GetUserByTokenHashWithTokenRow{}, err
	}
	return user, nil

}
