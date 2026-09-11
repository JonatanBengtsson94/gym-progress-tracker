package auth_test

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/user"
)

type mockUserRepository struct {
	getUserFunc func(ctx context.Context, username string) (user.User, error)
}

func (m *mockUserRepository) GetUserByUsername(ctx context.Context, username string) (user.User, error) {
	return m.getUserFunc(ctx, username)
}

type mockSessionRepository struct {
	getSessionFunc    func(sessionId uuid.UUID) (auth.Session, error)
	createSessionFunc func(userId uint32) auth.Session
}

func (m *mockSessionRepository) GetSessionBySessionId(sessionId uuid.UUID) (auth.Session, error) {
	return m.getSessionFunc(sessionId)
}

func (m *mockSessionRepository) CreateSession(userId uint32) auth.Session {
	return m.createSessionFunc(userId)
}

func TestAuthService_Login_Success(t *testing.T) {
	wantUser := user.User{UserId: 1, UserName: "alice", Password: "secret"}
	wantSession := auth.Session{UserId: 1, SessionId: uuid.New()}

	var gotUserId uint32
	userRepo := &mockUserRepository{
		getUserFunc: func(ctx context.Context, username string) (user.User, error) {
			if username != "alice" {
				t.Errorf("expected username %q, got %q", "alice", username)
			}
			return wantUser, nil
		},
	}
	sessionRepo := &mockSessionRepository{
		createSessionFunc: func(userId uint32) auth.Session {
			gotUserId = userId
			return wantSession
		},
	}

	service := auth.NewAuthService(userRepo, sessionRepo)

	got, err := service.Login(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if gotUserId != wantUser.UserId {
		t.Errorf("expected CreateSession to be called with userId %d, got %d", wantUser.UserId, gotUserId)
	}
	if got != wantSession {
		t.Errorf("Login() = %+v, want %+v", got, wantSession)
	}
}

func TestAuthService_Login_UnknownUser(t *testing.T) {
	userRepo := &mockUserRepository{
		getUserFunc: func(ctx context.Context, username string) (user.User, error) {
			return user.User{}, user.ErrUserNotFound
		},
	}
	sessionRepo := &mockSessionRepository{
		createSessionFunc: func(userId uint32) auth.Session {
			t.Fatal("CreateSession should not be called for an unknown user")
			return auth.Session{}
		},
	}

	service := auth.NewAuthService(userRepo, sessionRepo)

	_, err := service.Login(context.Background(), "ghost", "whatever")
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	userRepo := &mockUserRepository{
		getUserFunc: func(ctx context.Context, username string) (user.User, error) {
			return user.User{UserId: 1, UserName: "alice", Password: "secret"}, nil
		},
	}
	sessionRepo := &mockSessionRepository{
		createSessionFunc: func(userId uint32) auth.Session {
			t.Fatal("CreateSession should not be called for a wrong password")
			return auth.Session{}
		},
	}

	service := auth.NewAuthService(userRepo, sessionRepo)

	_, err := service.Login(context.Background(), "alice", "wrong-password")
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_RepositoryError(t *testing.T) {
	wantErr := errors.New("db exploded")
	userRepo := &mockUserRepository{
		getUserFunc: func(ctx context.Context, username string) (user.User, error) {
			return user.User{}, wantErr
		},
	}
	sessionRepo := &mockSessionRepository{
		createSessionFunc: func(userId uint32) auth.Session {
			t.Fatal("CreateSession should not be called when the repository errors")
			return auth.Session{}
		},
	}

	service := auth.NewAuthService(userRepo, sessionRepo)

	_, err := service.Login(context.Background(), "alice", "secret")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error to wrap %v, got %v", wantErr, err)
	}
	if errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatal("a repository error should not be collapsed into ErrInvalidCredentials")
	}
}

func TestAuthService_ValidateSession_Success(t *testing.T) {
	wantSessionId := uuid.New()
	wantUserId := uint32(7)

	sessionRepo := &mockSessionRepository{
		getSessionFunc: func(sessionId uuid.UUID) (auth.Session, error) {
			if sessionId != wantSessionId {
				t.Errorf("expected sessionId %v, got %v", wantSessionId, sessionId)
			}
			return auth.Session{UserId: wantUserId, SessionId: sessionId}, nil
		},
	}

	service := auth.NewAuthService(&mockUserRepository{}, sessionRepo)

	gotUserId, err := service.ValidateSession(wantSessionId)
	if err != nil {
		t.Fatalf("ValidateSession returned error: %v", err)
	}
	if gotUserId != wantUserId {
		t.Errorf("expected userId %d, got %d", wantUserId, gotUserId)
	}
}

func TestAuthService_ValidateSession_NotFound(t *testing.T) {
	sessionRepo := &mockSessionRepository{
		getSessionFunc: func(sessionId uuid.UUID) (auth.Session, error) {
			return auth.Session{}, auth.ErrSessionNotFound
		},
	}

	service := auth.NewAuthService(&mockUserRepository{}, sessionRepo)

	_, err := service.ValidateSession(uuid.New())
	if !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}
