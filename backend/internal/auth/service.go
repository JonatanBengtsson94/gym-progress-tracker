package auth

import (
	"context"
	"errors"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/user"
)

type UserGetter interface {
	GetUser(context.Context, string) (user.User, error)
}

type UserRepository interface {
	UserGetter
}

type SessionGetter interface {
	GetSession(uuid.UUID) (Session, error)
}

type SessionCreater interface {
	CreateSession(uint32) Session
}

type SessionRepository interface {
	SessionGetter
	SessionCreater
}

type AuthServiceImpl struct {
	userRepo    UserRepository
	sessionRepo SessionRepository
}

func NewAuthService(userRepo UserRepository, sessionRepo SessionRepository) *AuthServiceImpl {
	return &AuthServiceImpl{userRepo: userRepo, sessionRepo: sessionRepo}
}

func (s *AuthServiceImpl) Login(ctx context.Context, username string, password string) (Session, error) {
	u, err := s.userRepo.GetUser(ctx, username)
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
	session, err := s.sessionRepo.GetSession(sessionId)
	if err != nil {
		return 0, err
	}

	return session.UserId, nil
}
