package exercise_test

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
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

func TestExerciseRepository_GetGlobalExercises(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	exercises, err := repo.GetExercises(ctx, 0)
	if err != nil {
		t.Fatalf("GetExercises returned error: %v", err)
	}

	if len(exercises) != 7 {
		t.Errorf("Expected 7 exercises got: %d", len(exercises))
	}

	expectedNames := []string{
		"Bench Press",
		"Squat",
	}

	for _, expected := range expectedNames {
		found := false
		for _, e := range exercises {
			if e.ExerciseName == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected exercise %q not found in results", expected)
		}
	}
}

func TestExerciseRepository_GetUserExercises(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	exercises, err := repo.GetExercises(ctx, 1)
	if err != nil {
		t.Fatalf("GetExercises returned error: %v", err)
	}

	if len(exercises) != 8 {
		t.Errorf("Expected 8 exercises got: %d", len(exercises))
	}

	expectedNames := []string{
		"Bench Press",
		"Squat",
		"Custom Test Exercise",
	}

	for _, expected := range expectedNames {
		found := false
		for _, e := range exercises {
			if e.ExerciseName == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected exercise %q not found in results", expected)
		}
	}

}

func TestExerciseRepository_GetExercises_UsersDoNotSeeEachOthersCustomExercises(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	user1Exercises, err := repo.GetExercises(ctx, 1)
	if err != nil {
		t.Fatalf("GetExercises returned error: %v", err)
	}

	user2Exercises, err := repo.GetExercises(ctx, 2)
	if err != nil {
		t.Fatalf("GetExercises returned error: %v", err)
	}

	if containsExerciseName(user1Exercises, "Second User Exercise") {
		t.Errorf("User 1 should not see user 2's custom exercise")
	}
	if !containsExerciseName(user1Exercises, "Custom Test Exercise") {
		t.Errorf("User 1 should see their own custom exercise")
	}

	if containsExerciseName(user2Exercises, "Custom Test Exercise") {
		t.Errorf("User 2 should not see user 1's custom exercise")
	}
	if !containsExerciseName(user2Exercises, "Second User Exercise") {
		t.Errorf("User 2 should see their own custom exercise")
	}
}

func TestExerciseRepository_CreateExercise(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	created, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseName: "Lunge", UserId: 1})
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}

	if created.ExerciseId == 0 {
		t.Error("expected a non-zero ExerciseId to be assigned")
	}
	if created.ExerciseName != "Lunge" || created.UserId != 1 {
		t.Errorf("CreateExercise() = %+v, want ExerciseName=Lunge, UserId=1", created)
	}

	exercises, err := repo.GetExercises(ctx, 1)
	if err != nil {
		t.Fatalf("GetExercises returned error: %v", err)
	}
	if !containsExerciseName(exercises, "Lunge") {
		t.Errorf("expected newly created exercise to appear in GetExercises, got %+v", exercises)
	}
}

func TestExerciseRepository_CreateExercise_DuplicateName(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	// "Custom Test Exercise" is already seeded for user 1.
	_, err = repo.CreateExercise(ctx, exercise.Exercise{ExerciseName: "Custom Test Exercise", UserId: 1})
	if !errors.Is(err, exercise.ErrExerciseAlreadyExists) {
		t.Fatalf("Expected ErrExerciseAlreadyExists, got %v", err)
	}
}

func TestExerciseRepository_CreateExercise_DuplicateName_CaseInsensitive(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	_, err = repo.CreateExercise(ctx, exercise.Exercise{ExerciseName: "custom test exercise", UserId: 1})
	if !errors.Is(err, exercise.ErrExerciseAlreadyExists) {
		t.Fatalf("Expected ErrExerciseAlreadyExists for a case-insensitive duplicate, got %v", err)
	}
}

func TestExerciseRepository_CreateExercise_DuplicatesGlobalExercise(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	// "Bench Press" is a seeded global exercise.
	_, err = repo.CreateExercise(ctx, exercise.Exercise{ExerciseName: "Bench Press", UserId: 1})
	if !errors.Is(err, exercise.ErrExerciseAlreadyExists) {
		t.Fatalf("Expected ErrExerciseAlreadyExists, got %v", err)
	}
}

func TestExerciseRepository_CreateExercise_DuplicatesGlobalExercise_CaseInsensitive(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	_, err = repo.CreateExercise(ctx, exercise.Exercise{ExerciseName: "bench press", UserId: 1})
	if !errors.Is(err, exercise.ErrExerciseAlreadyExists) {
		t.Fatalf("Expected ErrExerciseAlreadyExists for a case-insensitive duplicate, got %v", err)
	}
}

func containsExerciseName(exercises []exercise.Exercise, name string) bool {
	for _, e := range exercises {
		if e.ExerciseName == name {
			return true
		}
	}
	return false
}
