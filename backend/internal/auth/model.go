package auth

import (
	"errors"
	"time"
	"uuid"
)

var ErrSessionNotFound = errors.New("session not found")
var ErrInvalidCredentials = errors.New("invalid credentials")

type Session struct {
	UserId    uint32    `json:"-"`
	SessionId uuid.UUID `json:"session_id"`
	ExpiresAt time.Time `json:"-"`
}
