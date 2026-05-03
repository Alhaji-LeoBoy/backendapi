package store

import (
	"context"
	"database/sql"
	"femProjectSqlc/internal/db"
)

type UserStore struct {
	db      *sql.DB
	queries *db.Queries
}

func NewUserStore(db *sql.DB, queries *db.Queries) *UserStore {
	return &UserStore{
		db:      db,
		queries: queries,
	}
}

type UserStoreInterface interface {
	CreateUser(ctx context.Context, arg CreateUserParams) (*db.CreateUserRow, error)
	GetUserByID(ctx context.Context, id int64) (*db.GetUserByIDRow, error)
	GetUserByEmail(ctx context.Context, email string) (*db.GetUserByEmailRow, error)
	GetUserByUsername(ctx context.Context, username string) (*db.GetUserByUsernameRow, error)
	GetUserByUsernameOrEmail(ctx context.Context, username, email string) (*db.GetUserByUsernameOrEmailRow, error)
	GetUserWithTokens(ctx context.Context, id int64) ([]db.GetUserWithTokensRow, error)
	ListUsers(ctx context.Context, limit, offset int64) ([]db.ListUsersRow, error)
	SearchUsers(ctx context.Context, search string, limit, offset int64) ([]db.SearchUsersRow, error)
	GetActiveUsers(ctx context.Context, limit, offset int64) ([]db.GetActiveUsersRow, error)
	BatchGetUsers(ctx context.Context) ([]db.BatchGetUsersRow, error)
	CountUsers(ctx context.Context) (int64, error)
	CountActiveUsers(ctx context.Context) (int64, error)
	UserExists(ctx context.Context, id int64) (bool, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
	EmailExists(ctx context.Context, email string) (bool, error)
	UpdatePassword(ctx context.Context, password string, id int64) error
	UpdateEmail(ctx context.Context, email string, id int64) error
	PatchUser(ctx context.Context, arg db.PatchUserParams) (*db.PatchUserRow, error)
	DeleteUser(ctx context.Context, id int64) error
	RestoreUser(ctx context.Context, id int64) error
	SetUserAdmin(ctx context.Context, isAdmin bool, id int64) error
}

type CreateUserParams struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *UserStore) CreateUser(ctx context.Context, arg CreateUserParams) (*db.CreateUserRow, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	createdUser, err := qtx.CreateUser(ctx, db.CreateUserParams{
		Username: arg.Username,
		Email:    arg.Email,
		Password: arg.Password,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &createdUser, nil
}

func (s *UserStore) GetUserByID(ctx context.Context, id int64) (*db.GetUserByIDRow, error) {
	user, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) GetUserByEmail(ctx context.Context, email string) (*db.GetUserByEmailRow, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) GetUserByUsername(ctx context.Context, username string) (*db.GetUserByUsernameRow, error) {
	user, err := s.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) GetUserByUsernameOrEmail(ctx context.Context, username, email string) (*db.GetUserByUsernameOrEmailRow, error) {
	user, err := s.queries.GetUserByUsernameOrEmail(ctx, db.GetUserByUsernameOrEmailParams{
		Username: username,
		Email:    email,
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) GetUserWithTokens(ctx context.Context, id int64) ([]db.GetUserWithTokensRow, error) {
	return s.queries.GetUserWithTokens(ctx, id)
}

func (s *UserStore) ListUsers(ctx context.Context, limit, offset int64) ([]db.ListUsersRow, error) {
	return s.queries.ListUsers(ctx, db.ListUsersParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (s *UserStore) SearchUsers(ctx context.Context, search string, limit, offset int64) ([]db.SearchUsersRow, error) {
	return s.queries.SearchUsers(ctx, db.SearchUsersParams{
		Column1: sql.NullString{String: search, Valid: search != ""},
		Limit:   limit,
		Offset:  offset,
	})
}

func (s *UserStore) GetActiveUsers(ctx context.Context, limit, offset int64) ([]db.GetActiveUsersRow, error) {
	return s.queries.GetActiveUsers(ctx, db.GetActiveUsersParams{
		Limit:  limit,
		Offset: offset,
	})
}

func (s *UserStore) BatchGetUsers(ctx context.Context) ([]db.BatchGetUsersRow, error) {
	return s.queries.BatchGetUsers(ctx)
}

func (s *UserStore) CountUsers(ctx context.Context) (int64, error) {
	return s.queries.CountUsers(ctx)
}

func (s *UserStore) CountActiveUsers(ctx context.Context) (int64, error) {
	return s.queries.CountActiveUsers(ctx)
}

func (s *UserStore) UserExists(ctx context.Context, id int64) (bool, error) {
	exists, err := s.queries.UserExists(ctx, id)
	return exists == 1, err
}

func (s *UserStore) UsernameExists(ctx context.Context, username string) (bool, error) {
	exists, err := s.queries.UsernameExists(ctx, username)
	return exists == 1, err
}

func (s *UserStore) EmailExists(ctx context.Context, email string) (bool, error) {
	exists, err := s.queries.EmailExists(ctx, email)
	return exists == 1, err
}

func (s *UserStore) UpdatePassword(ctx context.Context, password string, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.UpdatePassword(ctx, db.UpdatePasswordParams{
		Password: password,
		ID:       id,
	}); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *UserStore) UpdateEmail(ctx context.Context, email string, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.UpdateEmail(ctx, db.UpdateEmailParams{
		Email: email,
		ID:    id,
	}); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *UserStore) PatchUser(ctx context.Context, arg db.PatchUserParams) (*db.PatchUserRow, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	user, err := qtx.PatchUser(ctx, arg)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) DeleteUser(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.DeleteUser(ctx, id); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *UserStore) RestoreUser(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.RestoreUser(ctx, id); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *UserStore) SetUserAdmin(ctx context.Context, isAdmin bool, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := s.queries.WithTx(tx)
	if err := qtx.SetUserAdmin(ctx, db.SetUserAdminParams{
		IsAdmin: isAdmin,
		ID:      id,
	}); err != nil {
		return err
	}

	return tx.Commit()
}
