package auth

import (
	"sync"
	"time"
	"uuid"
)

type InMemorySessionRepository struct {
	mu              sync.RWMutex
	sessions        map[uuid.UUID]Session
	sessionDuration time.Duration
}

func NewSessionRepository(sessionDuration time.Duration) *InMemorySessionRepository {
	return &InMemorySessionRepository{
		sessions:        make(map[uuid.UUID]Session),
		sessionDuration: sessionDuration,
	}
}

func (r *InMemorySessionRepository) GetSession(sessionId uuid.UUID) (Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, ok := r.sessions[sessionId]
	if !ok {
		return Session{}, ErrSessionNotFound
	}

	if time.Now().After(session.ExpiresAt) {
		return Session{}, ErrSessionNotFound
	}

	return session, nil
}

func (r *InMemorySessionRepository) CreateSession(userID uint32) Session {
	r.mu.Lock()
	defer r.mu.Unlock()

	session := Session{
		UserId:    userID,
		SessionId: uuid.New(),
		ExpiresAt: time.Now().Add(r.sessionDuration),
	}

	r.sessions[session.SessionId] = session

	return session
}
