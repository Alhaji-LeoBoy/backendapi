package tokens

import (
	"context"
	"femProjectSqlc/internal/db"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// TokenPair holds generated access and refresh tokens
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// TokenStore is a minimal interface for creating tokens
type TokenStore interface {
	CreateToken(userID int64, tokenHash []byte, expiry time.Time, scope string) (db.CreateTokenRow, error)
}

const (
	AccessTokenExpiry  = 30 * time.Minute   // 30 Miinutes
	RefreshTokenExpiry = 7 * 24 * time.Hour // 7 day -> 168 houre
)

// GenerateTokenPair creates access and refresh tokens for a user
func GenerateTokenPair(ctx context.Context, userID int64, store TokenStore) (*TokenPair, error) {

	accessToken, err := GenerateToken(int(userID), AccessTokenExpiry, ScopeAccess)
	if err != nil {
		return nil, err
	}

	refreshToken, err := GenerateToken(int(userID), RefreshTokenExpiry, ScopeRefresh)
	if err != nil {
		return nil, err
	}

	_, err = store.CreateToken(userID, accessToken.Hash, accessToken.Expiry, accessToken.Scope)
	if err != nil {
		return nil, err
	}

	_, err = store.CreateToken(userID, refreshToken.Hash, refreshToken.Expiry, refreshToken.Scope)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken.Plaintext,
		RefreshToken: refreshToken.Plaintext,
	}, nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash verifies a password against a hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
