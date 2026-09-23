package auth

import (
	"context"
	"errors"
	"sync"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/password"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/user"
)

// dummyPasswordHash is compared against when the username does not exist, so
// that a login for an unknown user takes as long as one with a wrong password
// and response times don't reveal which usernames exist.
var dummyPasswordHash = sync.OnceValue(func() string {
	hash, err := password.Hash("dummy-password")
	if err != nil {
		panic(err)
	}
	return hash
})

type UserRepository interface {
	GetUserByUsername(context.Context, string) (user.User, error)
}

type SessionRepository interface {
	GetSessionBySessionId(uuid.UUID) (Session, error)
	CreateSession(uint32) Session
}

type AuthServiceImpl struct {
	userRepo    UserRepository
	sessionRepo SessionRepository
}

func NewAuthService(userRepo UserRepository, sessionRepo SessionRepository) *AuthServiceImpl {
	return &AuthServiceImpl{userRepo: userRepo, sessionRepo: sessionRepo}
}

func (s *AuthServiceImpl) Login(ctx context.Context, username string, plainPassword string) (Session, error) {
	u, err := s.userRepo.GetUserByUsername(ctx, username)
	if errors.Is(err, user.ErrUserNotFound) {
		password.Matches(dummyPasswordHash(), plainPassword)
		return Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, err
	}

	if !password.Matches(u.PasswordHash, plainPassword) {
		return Session{}, ErrInvalidCredentials
	}

	session := s.sessionRepo.CreateSession(u.UserId)

	return session, nil
}

func (s *AuthServiceImpl) ValidateSession(sessionId uuid.UUID) (uint32, error) {
	session, err := s.sessionRepo.GetSessionBySessionId(sessionId)
	if err != nil {
		return 0, err
	}

	return session.UserId, nil
}
