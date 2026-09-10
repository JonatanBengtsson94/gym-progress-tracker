package auth_test

import (
	"errors"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
)

func TestInMemorySessionRepository_CreateAndGetSession(t *testing.T) {
	repo := auth.NewSessionRepository(time.Hour)

	created := repo.CreateSession(1)

	got, err := repo.GetSession(created.SessionId)
	if err != nil {
		t.Fatalf("GetSession returned error: %v", err)
	}

	if got != created {
		t.Errorf("GetSession() = %+v, want %+v", got, created)
	}
}

func TestInMemorySessionRepository_GetSession_NotFound(t *testing.T) {
	repo := auth.NewSessionRepository(time.Hour)

	_, err := repo.GetSession(uuid.New())
	if !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("Expected ErrSessionNotFound, got %v", err)
	}
}

func TestInMemorySessionRepository_GetSession_Expired(t *testing.T) {
	repo := auth.NewSessionRepository(-time.Minute)

	created := repo.CreateSession(1)

	_, err := repo.GetSession(created.SessionId)
	if !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("Expected ErrSessionNotFound for expired session, got %v", err)
	}
}

func TestInMemorySessionRepository_ConcurrentAccess(t *testing.T) {
	repo := auth.NewSessionRepository(time.Hour)

	var wg sync.WaitGroup
	errs := make(chan error, 20)

	for i := range 20 {
		wg.Add(1)
		go func(userId uint32) {
			defer wg.Done()

			session := repo.CreateSession(userId)

			if _, err := repo.GetSession(session.SessionId); err != nil {
				errs <- err
			}
		}(uint32(i))
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent access returned error: %v", err)
	}
}
