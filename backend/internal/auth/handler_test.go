package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
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

	var got auth.LoginResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	want := auth.LoginResponse{SessionId: wantSession.SessionId.String()}
	if got != want {
		t.Errorf("Login() response = %+v, want %+v", got, want)
	}
}

func TestAuthHandler_Login_ResponseContainsOnlyExpectedFields(t *testing.T) {
	sessionId := uuid.New()
	service := &mockAuthService{
		loginFunc: func(ctx context.Context, username, password string) (auth.Session, error) {
			return auth.Session{
				UserId:    99,
				SessionId: sessionId,
				ExpiresAt: time.Now().Add(time.Hour),
			}, nil
		},
	}
	handler := auth.NewAuthHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	var got map[string]any
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	want := map[string]any{
		"session_id": sessionId.String(),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Login() response fields = %v, want exactly %v", got, want)
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
