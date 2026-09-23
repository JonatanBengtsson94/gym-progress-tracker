// Package password hashes and verifies user passwords with bcrypt so that
// plain text passwords are never persisted.
package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

var ErrPasswordRequired = errors.New("password is required")
var ErrPasswordTooLong = errors.New("password is too long")

// Hash returns a bcrypt hash of plain, suitable for storing in the database.
func Hash(plain string) (string, error) {
	if plain == "" {
		return "", ErrPasswordRequired
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if errors.Is(err, bcrypt.ErrPasswordTooLong) {
		return "", ErrPasswordTooLong
	}
	if err != nil {
		return "", fmt.Errorf("Hash password failed: %w", err)
	}

	return string(hash), nil
}

// Matches reports whether plain is the password that produced hash. A
// malformed hash never matches.
func Matches(hash string, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
