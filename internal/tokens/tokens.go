package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"time"
)

type Tokens struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserId    int       `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
	ErrInvalidScope = errors.New("invalid scope")
)

// Token plaintext length: 32 random bytes → base32 no-padding → 52 chars
const tokenLength = 52

func GenerateToken(userId int, ttl time.Duration, scope string) (*Tokens, error) {

	if err := ValidateScope(scope); err != nil {
		return nil, err
	}
	token := &Tokens{
		UserId: userId,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

// HashToken returns the SHA-256 hash of a plaintext token.
// Use this consistently for both generation and DB lookups.
func HashToken(plaintext string) []byte {
	hash := sha256.Sum256([]byte(plaintext))
	return hash[:]
}

// ValidateTokenPlaintext checks that the plaintext token has the correct format.
// Call this on any token received from a client before hashing and querying the DB.
func ValidateTokenPlaintext(plaintext string) error {
	if len(plaintext) != tokenLength {
		return ErrInvalidToken
	}
	return nil
}

// ValidateScope checks that the given scope is one of the known valid scopes.
func ValidateScope(scope string) error {
	switch scope {
	case ScopeAccess, ScopeRefresh, ScopeAPI:
		return nil
	default:
		return ErrInvalidScope
	}
}

// IsExpired reports whether a token has passed its expiry time.
func (t *Tokens) IsExpired() bool {
	return time.Now().After(t.Expiry)
}
