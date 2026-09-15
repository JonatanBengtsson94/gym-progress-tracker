package workout_test

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/workout"
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

func TestWorkoutRepository_GetWorkout(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	got, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, 1)
	if err != nil {
		t.Fatalf("GetWorkoutByUserIdAndWorkoutId returned error: %v", err)
	}

	if got.WorkoutId != 1 {
		t.Errorf("expected WorkoutId 1, got %d", got.WorkoutId)
	}
	if got.Template.TemplateName != "Push Day" {
		t.Errorf("expected TemplateName %q, got %q", "Push Day", got.Template.TemplateName)
	}
	if got.CompletedAt.Format("2006-01-02 15:04:05") != "2024-01-15 10:00:00" {
		t.Errorf("expected CompletedAt %q, got %q", "2024-01-15 10:00:00", got.CompletedAt.Format("2006-01-02 15:04:05"))
	}

	if len(got.Sets) != 2 {
		t.Fatalf("expected 2 sets, got %d: %+v", len(got.Sets), got.Sets)
	}

	for _, s := range got.Sets {
		switch s.Exercise.ExerciseName {
		case "Bench Press":
			if s.Reps != 8 {
				t.Errorf("expected Bench Press Reps 8, got %d", s.Reps)
			}
			if s.WeightGrams != 60000 {
				t.Errorf("expected Bench Press WeightGrams 60000, got %d", s.WeightGrams)
			}
		case "Squat":
			if s.Reps != 5 {
				t.Errorf("expected Squat Reps 5, got %d", s.Reps)
			}
			if s.WeightGrams != 100000 {
				t.Errorf("expected Squat WeightGrams 100000, got %d", s.WeightGrams)
			}
		default:
			t.Errorf("unexpected exercise in workout sets: %q", s.Exercise.ExerciseName)
		}
	}
}

func TestWorkoutRepository_GetWorkout_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	_, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, 999999)
	if !errors.Is(err, workout.ErrWorkoutNotFound) {
		t.Fatalf("Expected ErrWorkoutNotFound, got %v", err)
	}
}

func TestWorkoutRepository_GetWorkout_NoSets(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	// Workout 3 exists but has no sets recorded, so it should look not found.
	_, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, 3)
	if !errors.Is(err, workout.ErrWorkoutNotFound) {
		t.Fatalf("Expected ErrWorkoutNotFound, got %v", err)
	}
}

func TestWorkoutRepository_GetWorkout_WrongUser(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	// Workout 2 belongs to user 2, so it should look not found to user 1.
	_, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, 2)
	if !errors.Is(err, workout.ErrWorkoutNotFound) {
		t.Fatalf("Expected ErrWorkoutNotFound, got %v", err)
	}
}

func TestWorkoutRepository_GetWorkout_UserSeesOwnWorkout(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	got, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 2, 2)
	if err != nil {
		t.Fatalf("GetWorkoutByUserIdAndWorkoutId returned error: %v", err)
	}

	if got.Template.TemplateName != "Pull Day" {
		t.Errorf("expected TemplateName %q, got %q", "Pull Day", got.Template.TemplateName)
	}
	if len(got.Sets) != 1 {
		t.Fatalf("expected 1 set, got %d: %+v", len(got.Sets), got.Sets)
	}
	if got.Sets[0].Exercise.ExerciseName != "Deadlift" {
		t.Errorf("expected exercise %q, got %q", "Deadlift", got.Sets[0].Exercise.ExerciseName)
	}
}
