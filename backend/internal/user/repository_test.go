package user_test

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/testutil"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	pool, teardown, err := testutil.StartPostgres(
		ctx,
		filepath.Join("..", "..", "..", "database", "migrations", "*.sql"),
		filepath.Join("testdata", "seed.sql"),
	)
	if err != nil {
		log.Fatalf("failed to set up test database: %v", err)
	}
	testPool = pool

	code := m.Run()

	teardown()
	os.Exit(code)
}

func TestUserRepository_GetUser_Success(t *testing.T) {
	ctx := t.Context()
	repo := user.NewPostgresUserRepository(testPool)

	got, err := repo.GetUserByUsername(ctx, "Test User")
	if err != nil {
		t.Fatalf("GetUser returned error: %v", err)
	}

	want := user.User{
		UserId:       1,
		UserName:     "Test User",
		PasswordHash: "hashed-password",
	}

	if got != want {
		t.Errorf("GetUser() = %+v, want %+v", got, want)
	}
}

func TestUserRepository_GetUser_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := user.NewPostgresUserRepository(testPool)

	_, err := repo.GetUserByUsername(ctx, "Nonexistent User")
	if !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestUserRepository_GetUserByUserId_Success(t *testing.T) {
	ctx := t.Context()
	repo := user.NewPostgresUserRepository(testPool)

	got, err := repo.GetUserByUserId(ctx, 1)
	if err != nil {
		t.Fatalf("GetUserByUserId returned error: %v", err)
	}

	want := user.User{
		UserId:    1,
		UserName:  "Test User",
		FirstName: "Test",
		LastName:  "User",
	}

	if got != want {
		t.Errorf("GetUserByUserId() = %+v, want %+v", got, want)
	}
}

func TestUserRepository_GetUserByUserId_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := user.NewPostgresUserRepository(testPool)

	_, err := repo.GetUserByUserId(ctx, 999)
	if !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestUserRepository_CreateUser_Success(t *testing.T) {
	ctx := t.Context()
	repo := user.NewPostgresUserRepository(testPool)

	created, err := repo.CreateUser(ctx, user.User{
		UserName:     "New User",
		PasswordHash: "new-hash",
		FirstName:    "New",
		LastName:     "User",
	})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	if created.UserId == 0 {
		t.Fatal("expected CreateUser to return the generated user id")
	}

	got, err := repo.GetUserByUsername(ctx, "New User")
	if err != nil {
		t.Fatalf("GetUserByUsername returned error: %v", err)
	}

	want := user.User{UserId: created.UserId, UserName: "New User", PasswordHash: "new-hash"}
	if got != want {
		t.Errorf("GetUserByUsername() = %+v, want %+v", got, want)
	}
}

func TestUserRepository_CreateUser_UsernameTaken(t *testing.T) {
	ctx := t.Context()
	repo := user.NewPostgresUserRepository(testPool)

	_, err := repo.CreateUser(ctx, user.User{
		UserName:     "Test User",
		PasswordHash: "another-hash",
		FirstName:    "Test",
		LastName:     "User",
	})
	if !errors.Is(err, user.ErrUsernameTaken) {
		t.Fatalf("Expected ErrUsernameTaken, got %v", err)
	}
}
