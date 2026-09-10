package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
)

type mockSessionValidator struct {
	validateSessionFunc func(sessionId uuid.UUID) (uint32, error)
}

func (m *mockSessionValidator) ValidateSession(sessionId uuid.UUID) (uint32, error) {
	return m.validateSessionFunc(sessionId)
}

func TestAuthMiddleware_RequireAuth_Success(t *testing.T) {
	wantUserId := uint32(42)
	wantSessionId := uuid.New()

	validator := &mockSessionValidator{
		validateSessionFunc: func(sessionId uuid.UUID) (uint32, error) {
			if sessionId != wantSessionId {
				t.Errorf("expected sessionId %v, got %v", wantSessionId, sessionId)
			}
			return wantUserId, nil
		},
	}
	middleware := auth.NewAuthMiddleware(validator)

	var nextCalled bool
	var gotUserId uint32
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		gotUserId, _ = auth.UserIdFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	req.Header.Set("Authorization", "Bearer "+wantSessionId.String())
	rec := httptest.NewRecorder()

	middleware.RequireAuth(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if gotUserId != wantUserId {
		t.Errorf("expected userId %d in context, got %d", wantUserId, gotUserId)
	}
	if rec.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Result().StatusCode)
	}
}

func TestAuthMiddleware_RequireAuth_MissingHeader(t *testing.T) {
	validator := &mockSessionValidator{
		validateSessionFunc: func(sessionId uuid.UUID) (uint32, error) {
			t.Fatal("ValidateSession should not be called without a token")
			return 0, nil
		},
	}
	middleware := auth.NewAuthMiddleware(validator)

	var nextCalled bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	rec := httptest.NewRecorder()

	middleware.RequireAuth(next).ServeHTTP(rec, req)

	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestAuthMiddleware_RequireAuth_MalformedToken(t *testing.T) {
	validator := &mockSessionValidator{
		validateSessionFunc: func(sessionId uuid.UUID) (uint32, error) {
			t.Fatal("ValidateSession should not be called for a malformed token")
			return 0, nil
		},
	}
	middleware := auth.NewAuthMiddleware(validator)

	var nextCalled bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	req.Header.Set("Authorization", "Bearer not-a-uuid")
	rec := httptest.NewRecorder()

	middleware.RequireAuth(next).ServeHTTP(rec, req)

	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestAuthMiddleware_RequireAuth_SessionNotFound(t *testing.T) {
	validator := &mockSessionValidator{
		validateSessionFunc: func(sessionId uuid.UUID) (uint32, error) {
			return 0, auth.ErrSessionNotFound
		},
	}
	middleware := auth.NewAuthMiddleware(validator)

	var nextCalled bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	req := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	req.Header.Set("Authorization", "Bearer "+uuid.New().String())
	rec := httptest.NewRecorder()

	middleware.RequireAuth(next).ServeHTTP(rec, req)

	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Result().StatusCode)
	}
}
