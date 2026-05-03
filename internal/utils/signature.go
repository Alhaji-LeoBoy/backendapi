package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
)

var (
	ErrMissingSecretKey  = errors.New("TICKET_SECRET_KEY environment variable is not set")
	ErrInvalidSignature  = errors.New("invalid ticket signature")
)

// TicketSigner handles HMAC-based ticket signing and verification.
// The signature binds a ticket to its userID, eventID, and name,
// so it cannot be forged or transferred.
type TicketSigner struct {
	key []byte
}

// NewTicketSigner creates a signer from the TICKET_SECRET_KEY env var.
// Returns an error if the key is missing or empty.
func NewTicketSigner() (*TicketSigner, error) {
	key := os.Getenv("TICKET_SECRET_KEY")
	if key == "" {
		return nil, ErrMissingSecretKey
	}
	return &TicketSigner{key: []byte(key)}, nil
}

// NewTicketSignerFromKey creates a signer from an explicit key (useful for tests).
func NewTicketSignerFromKey(key string) *TicketSigner {
	return &TicketSigner{key: []byte(key)}
}

// Sign produces an HMAC-SHA256 hex signature for the given ticket fields.
// The message format is "userID:eventID:name" — this ties the signature
// to a specific user, event, and ticket name.
func (s *TicketSigner) Sign(userID, eventID int64, name string) string {
	message := fmt.Sprintf("%d:%d:%s", userID, eventID, name)
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify checks whether the provided signature matches the expected one.
// Uses hmac.Equal for constant-time comparison to prevent timing attacks.
func (s *TicketSigner) Verify(userID, eventID int64, name, signature string) bool {
	expected, err := hex.DecodeString(s.Sign(userID, eventID, name))
	if err != nil {
		return false
	}
	actual, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	return hmac.Equal(expected, actual)
}

// GenerateSecretKey generates a cryptographically secure random key
// of the given byte length, returned as a hex string.
// Use this to generate a value for TICKET_SECRET_KEY.
func GenerateSecretKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
