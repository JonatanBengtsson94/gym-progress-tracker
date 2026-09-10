package user_test

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	migrationFiles, err := filepath.Glob(filepath.Join("..", "..", "..", "database", "migrations", "*.sql"))
	if err != nil {
		log.Fatalf("failed to glob migrations: %v", err)
	}
	sort.Strings(migrationFiles)

	if len(migrationFiles) == 0 {
		log.Fatal("no migration files found")
	}

	seedFile := filepath.Join("testdata", "seed.sql")
	initScripts := append(migrationFiles, seedFile)

	pgContainer, err := postgres.Run(ctx,
		"docker.io/postgres:latest",
		postgres.WithDatabase("testDB"),
		postgres.WithUsername("testUser"),
		postgres.WithPassword("testPass"),
		postgres.WithInitScripts(initScripts...),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)

	if err != nil {
		log.Fatalf("failed to start postgres testcontainer: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}

	testPool, err = pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("failed to connect pool: %v", err)
	}

	if err := testPool.Ping(ctx); err != nil {
		testPool.Close()
		_ = pgContainer.Terminate(ctx)
		log.Fatalf("failed to ping postgres: %v", err)
	}

	code := m.Run()

	testPool.Close()
	_ = pgContainer.Terminate(ctx)
	os.Exit(code)
}

func TestUserRepository_GetUser_Success(t *testing.T) {
	ctx := t.Context()
	repo := user.NewUserRepository(testPool)

	got, err := repo.GetUser(ctx, "Test User")
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
	repo := user.NewUserRepository(testPool)

	_, err := repo.GetUser(ctx, "Nonexistent User")
	if !errors.Is(err, user.ErrUserNotFound) {
		t.Fatalf("Expected ErrUserNotFound, got %v", err)
	}
}
