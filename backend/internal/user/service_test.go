package user_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/password"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/user"
)

type mockUserRepository struct {
	getUserByUserIdFunc func(ctx context.Context, userId uint32) (user.User, error)
	createUserFunc      func(ctx context.Context, u user.User) (user.User, error)
}

func (m *mockUserRepository) GetUserByUserId(ctx context.Context, userId uint32) (user.User, error) {
	return m.getUserByUserIdFunc(ctx, userId)
}

func (m *mockUserRepository) CreateUser(ctx context.Context, u user.User) (user.User, error) {
	return m.createUserFunc(ctx, u)
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

func TestUserService_CreateUser_StoresHashedPassword(t *testing.T) {
	ctx := t.Context()

	var stored user.User
	repo := &mockUserRepository{
		createUserFunc: func(ctx context.Context, u user.User) (user.User, error) {
			stored = u
			u.UserId = 7
			return u, nil
		},
	}

	service := user.NewUserService(repo)

	created, err := service.CreateUser(ctx, "alice", "secret", "Alice", "Anderson")
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	if stored.PasswordHash == "secret" {
		t.Fatal("expected the plain text password not to reach the repository")
	}
	if !password.Matches(stored.PasswordHash, "secret") {
		t.Error("expected the stored hash to match the password")
	}
	if stored.UserName != "alice" || stored.FirstName != "Alice" || stored.LastName != "Anderson" {
		t.Errorf("unexpected user passed to repository: %+v", stored)
	}
	if created.UserId != 7 {
		t.Errorf("expected created user id 7, got %d", created.UserId)
	}
}

func TestUserService_CreateUser_MissingFields(t *testing.T) {
	ctx := t.Context()

	repo := &mockUserRepository{
		createUserFunc: func(ctx context.Context, u user.User) (user.User, error) {
			t.Fatal("CreateUser should not reach the repository with missing fields")
			return user.User{}, nil
		},
	}

	service := user.NewUserService(repo)

	cases := []struct {
		name                          string
		username, firstName, lastName string
	}{
		{"missing username", " ", "Alice", "Anderson"},
		{"missing first name", "alice", "", "Anderson"},
		{"missing last name", "alice", "Alice", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.CreateUser(ctx, tc.username, "secret", tc.firstName, tc.lastName)
			if !errors.Is(err, user.ErrUserFieldsRequired) {
				t.Fatalf("expected ErrUserFieldsRequired, got %v", err)
			}
		})
	}
}

func TestUserService_CreateUser_EmptyPassword(t *testing.T) {
	ctx := t.Context()

	repo := &mockUserRepository{
		createUserFunc: func(ctx context.Context, u user.User) (user.User, error) {
			t.Fatal("CreateUser should not reach the repository with an empty password")
			return user.User{}, nil
		},
	}

	service := user.NewUserService(repo)

	_, err := service.CreateUser(ctx, "alice", "", "Alice", "Anderson")
	if !errors.Is(err, password.ErrPasswordRequired) {
		t.Fatalf("expected ErrPasswordRequired, got %v", err)
	}
}

func TestUserService_CreateUser_UsernameTaken(t *testing.T) {
	ctx := t.Context()

	repo := &mockUserRepository{
		createUserFunc: func(ctx context.Context, u user.User) (user.User, error) {
			return user.User{}, user.ErrUsernameTaken
		},
	}

	service := user.NewUserService(repo)

	_, err := service.CreateUser(ctx, "alice", "secret", "Alice", "Anderson")
	if !errors.Is(err, user.ErrUsernameTaken) {
		t.Fatalf("expected ErrUsernameTaken, got %v", err)
	}
}
