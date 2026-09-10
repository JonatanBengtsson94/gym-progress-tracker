package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
)

type mockAuthService struct {
	loginFunc func(ctx context.Context, username, password string) (auth.Session, error)
}

func (m *mockAuthService) Login(ctx context.Context, username, password string) (auth.Session, error) {
	return m.loginFunc(ctx, username, password)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	wantSession := auth.Session{SessionId: uuid.New()}
	var gotUsername, gotPassword string

	service := &mockAuthService{
		loginFunc: func(ctx context.Context, username, password string) (auth.Session, error) {
			gotUsername = username
			gotPassword = password
			return wantSession, nil
		},
	}
	handler := auth.NewAuthHandler(service)

	body := strings.NewReader(`{"username":"alice","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
	if gotUsername != "alice" || gotPassword != "secret" {
		t.Errorf("expected Login called with (alice, secret), got (%s, %s)", gotUsername, gotPassword)
	}

	var got auth.Session
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if got.SessionId != wantSession.SessionId {
		t.Errorf("expected session_id %v, got %v", wantSession.SessionId, got.SessionId)
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	service := &mockAuthService{
		loginFunc: func(ctx context.Context, username, password string) (auth.Session, error) {
			return auth.Session{}, auth.ErrInvalidCredentials
		},
	}
	handler := auth.NewAuthHandler(service)

	body := strings.NewReader(`{"username":"alice","password":"wrong"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestAuthHandler_Login_ServiceError(t *testing.T) {
	service := &mockAuthService{
		loginFunc: func(ctx context.Context, username, password string) (auth.Session, error) {
			return auth.Session{}, errors.New("db exploded")
		},
	}
	handler := auth.NewAuthHandler(service)

	body := strings.NewReader(`{"username":"alice","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Result().StatusCode)
	}
}

func TestAuthHandler_Login_MalformedBody(t *testing.T) {
	service := &mockAuthService{
		loginFunc: func(ctx context.Context, username, password string) (auth.Session, error) {
			t.Fatal("Login should not be called for a malformed request body")
			return auth.Session{}, nil
		},
	}
	handler := auth.NewAuthHandler(service)

	body := strings.NewReader(`not-json`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Result().StatusCode)
	}
}
