package utils

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode"
)

// ValidateEmail checks if the email is well-formed.
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// ValidatePassword enforces minimum strength requirements:
// at least 8 chars, one uppercase, one lowercase, one digit.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	var hasUpper, hasLower, hasDigit bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}
	return nil
}

// ValidateUsername checks username constraints.
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username is required")
	}
	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}
	if len(username) > 30 {
		return fmt.Errorf("username must be at most 30 characters")
	}
	return nil
}

var validTicketStatuses = map[string]bool{
	"active":    true,
	"used":      true,
	"cancelled": true,
	"expired":   true,
	"pending":   true,
}

// ValidateTicketStatus checks if the status is one of the allowed values.
func ValidateTicketStatus(status string) error {
	if !validTicketStatuses[strings.ToLower(status)] {
		return fmt.Errorf("invalid ticket status %q, must be one of: active, used, cancelled, expired, pending", status)
	}
	return nil
}

// ValidatePrice checks that price is non-negative.
func ValidatePrice(price float64) error {
	if price < 0 {
		return fmt.Errorf("price must be non-negative")
	}
	return nil
}
