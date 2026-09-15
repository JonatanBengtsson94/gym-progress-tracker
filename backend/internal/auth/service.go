package auth

import (
	"context"
	"errors"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/user"
)

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

func (s *AuthServiceImpl) Login(ctx context.Context, username string, password string) (Session, error) {
	u, err := s.userRepo.GetUserByUsername(ctx, username)
	if errors.Is(err, user.ErrUserNotFound) {
		return Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, err
	}

	if u.Password != password {
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
