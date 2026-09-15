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
		UserId:   1,
		UserName: "Test User",
		Password: "hashed-password",
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
