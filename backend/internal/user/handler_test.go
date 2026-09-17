package user_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/identity"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/user"
)

type mockUserService struct {
	getUserFunc func(ctx context.Context, userId uint32) (user.User, error)
}

func (m *mockUserService) GetUser(ctx context.Context, userId uint32) (user.User, error) {
	return m.getUserFunc(ctx, userId)
}

func TestUserHandler_GetUserFromSession_Success(t *testing.T) {
	stored := user.User{UserId: 1, UserName: "Test User", Password: "hashed-password", FirstName: "Test", LastName: "User"}

	var gotUserId uint32
	service := &mockUserService{
		getUserFunc: func(ctx context.Context, userId uint32) (user.User, error) {
			gotUserId = userId
			return stored, nil
		},
	}

	handler := user.NewUserHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.GetUserFromSession(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", ct)
	}
	if gotUserId != 1 {
		t.Errorf("expected service called with the session's userId 1, got %d", gotUserId)
	}

	var body user.UserResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}
	want := user.UserResponse{UserId: 1, UserName: "Test User", FirstName: "Test", LastName: "User"}
	if body != want {
		t.Errorf("GetUserFromSession() response = %+v, want %+v", body, want)
	}
}

func TestUserHandler_GetUserFromSession_ResponseContainsOnlyExpectedFields(t *testing.T) {
	stored := user.User{UserId: 1, UserName: "Test User", Password: "hashed-password", FirstName: "Test", LastName: "User"}
	service := &mockUserService{
		getUserFunc: func(ctx context.Context, userId uint32) (user.User, error) {
			return stored, nil
		},
	}

	handler := user.NewUserHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.GetUserFromSession(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	var got map[string]any
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	want := map[string]any{
		"user_id":    float64(1),
		"user_name":  "Test User",
		"first_name": "Test",
		"last_name":  "User",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetUserFromSession() response fields = %v, want exactly %v", got, want)
	}
}

func TestUserHandler_GetUserFromSession_Unauthorized(t *testing.T) {
	service := &mockUserService{
		getUserFunc: func(ctx context.Context, userId uint32) (user.User, error) {
			t.Fatal("GetUser should not be called without an authenticated user")
			return user.User{}, nil
		},
	}

	handler := user.NewUserHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	rec := httptest.NewRecorder()

	handler.GetUserFromSession(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestUserHandler_GetUserFromSession_UserNotFound(t *testing.T) {
	service := &mockUserService{
		getUserFunc: func(ctx context.Context, userId uint32) (user.User, error) {
			return user.User{}, user.ErrUserNotFound
		},
	}

	handler := user.NewUserHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.GetUserFromSession(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestUserHandler_GetUserFromSession_ServiceError(t *testing.T) {
	service := &mockUserService{
		getUserFunc: func(ctx context.Context, userId uint32) (user.User, error) {
			return user.User{}, errors.New("db exploded")
		},
	}

	handler := user.NewUserHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.GetUserFromSession(rec, req)

	if rec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", rec.Result().StatusCode)
	}
}
