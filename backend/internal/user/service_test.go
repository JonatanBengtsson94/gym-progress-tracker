package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/user"
)

type mockUserRepository struct {
	getUserByUserIdFunc func(ctx context.Context, userId uint32) (user.User, error)
}

func (m *mockUserRepository) GetUserByUserId(ctx context.Context, userId uint32) (user.User, error) {
	return m.getUserByUserIdFunc(ctx, userId)
}

func TestUserService_GetUser(t *testing.T) {
	ctx := t.Context()
	stored := user.User{UserId: 42, UserName: "Test User", FirstName: "Test", LastName: "User"}

	var gotUserId uint32
	repo := &mockUserRepository{
		getUserByUserIdFunc: func(ctx context.Context, userId uint32) (user.User, error) {
			gotUserId = userId
			return stored, nil
		},
	}

	service := user.NewUserService(repo)

	result, err := service.GetUser(ctx, 42)
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}

	if gotUserId != 42 {
		t.Errorf("expected repo to receive userId 42, got %d", gotUserId)
	}
	if result != stored {
		t.Errorf("GetUser() = %+v, want %+v", result, stored)
	}
}

func TestUserService_GetUser_NotFound(t *testing.T) {
	ctx := t.Context()

	repo := &mockUserRepository{
		getUserByUserIdFunc: func(ctx context.Context, userId uint32) (user.User, error) {
			return user.User{}, user.ErrUserNotFound
		},
	}

	service := user.NewUserService(repo)

	_, err := service.GetUser(ctx, 42)
	if !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("Expected ErrUserNotFound, got %v", err)
	}
}
