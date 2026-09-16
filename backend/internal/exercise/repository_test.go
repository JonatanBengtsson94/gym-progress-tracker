package exercise_test

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/testutil"
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

func TestExerciseRepository_GetGlobalExercises(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	exercises, err := repo.GetExercisesByUserId(ctx, 0)
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
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	exercises, err := repo.GetExercisesByUserId(ctx, 1)
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
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	user1Exercises, err := repo.GetExercisesByUserId(ctx, 1)
	if err != nil {
		t.Fatalf("GetExercises returned error: %v", err)
	}

	user2Exercises, err := repo.GetExercisesByUserId(ctx, 2)
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
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
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

	exercises, err := repo.GetExercisesByUserId(ctx, 1)
	if err != nil {
		t.Fatalf("GetExercises returned error: %v", err)
	}
	if !containsExerciseName(exercises, "Lunge") {
		t.Errorf("expected newly created exercise to appear in GetExercises, got %+v", exercises)
	}
}

func TestExerciseRepository_CreateExercise_DuplicateName(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
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
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
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
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
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
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	_, err = repo.CreateExercise(ctx, exercise.Exercise{ExerciseName: "bench press", UserId: 1})
	if !errors.Is(err, exercise.ErrExerciseAlreadyExists) {
		t.Fatalf("Expected ErrExerciseAlreadyExists for a case-insensitive duplicate, got %v", err)
	}
}

func TestExerciseRepository_CreateExercise_UserNotFound(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	_, err = repo.CreateExercise(ctx, exercise.Exercise{ExerciseName: "Ghost Exercise", UserId: 999999})
	if err == nil {
		t.Fatal("expected an error for a nonexistent user_id, got nil")
	}
}

func TestExerciseRepository_ModifyExercise(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	created, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseName: "Front Squat", UserId: 1})
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}

	modified, err := repo.ModifyExercise(ctx, exercise.Exercise{ExerciseId: created.ExerciseId, ExerciseName: "Romanian Deadlift", UserId: 1})
	if err != nil {
		t.Fatalf("ModifyExercise returned error: %v", err)
	}

	if modified.ExerciseId != created.ExerciseId {
		t.Errorf("expected ExerciseId %d, got %d", created.ExerciseId, modified.ExerciseId)
	}
	if modified.ExerciseName != "Romanian Deadlift" {
		t.Errorf("expected ExerciseName %q, got %q", "Romanian Deadlift", modified.ExerciseName)
	}

	exercises, err := repo.GetExercisesByUserId(ctx, 1)
	if err != nil {
		t.Fatalf("GetExercises returned error: %v", err)
	}
	if !containsExerciseName(exercises, "Romanian Deadlift") {
		t.Errorf("expected modified exercise to appear in GetExercises, got %+v", exercises)
	}
	if containsExerciseName(exercises, "Front Squat") {
		t.Errorf("expected old exercise name to be gone, got %+v", exercises)
	}
}

func TestExerciseRepository_ModifyExercise_GlobalExercise(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	globalExercises, err := repo.GetExercisesByUserId(ctx, 0)
	if err != nil {
		t.Fatalf("GetExercises returned error: %v", err)
	}

	var benchPressId uint32
	for _, e := range globalExercises {
		if e.ExerciseName == "Bench Press" {
			benchPressId = e.ExerciseId
			break
		}
	}
	if benchPressId == 0 {
		t.Fatal("expected seeded global exercise \"Bench Press\" to be found")
	}

	_, err = repo.ModifyExercise(ctx, exercise.Exercise{ExerciseId: benchPressId, ExerciseName: "Bench Press Variant", UserId: 1})
	if !errors.Is(err, exercise.ErrCannotModifyGlobalExercise) {
		t.Fatalf("Expected ErrCannotModifyGlobalExercise, got %v", err)
	}
}

func TestExerciseRepository_ModifyExercise_NotFound(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	_, err = repo.ModifyExercise(ctx, exercise.Exercise{ExerciseId: 999999, ExerciseName: "Ghost Exercise", UserId: 1})
	if !errors.Is(err, exercise.ErrExerciseNotFound) {
		t.Fatalf("Expected ErrExerciseNotFound, got %v", err)
	}
}

func TestExerciseRepository_ModifyExercise_WrongUser(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	created, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseName: "Incline Bench", UserId: 1})
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}

	// user 2 does not own this exercise, so it should look not found to them.
	_, err = repo.ModifyExercise(ctx, exercise.Exercise{ExerciseId: created.ExerciseId, ExerciseName: "Chin Up", UserId: 2})
	if !errors.Is(err, exercise.ErrExerciseNotFound) {
		t.Fatalf("Expected ErrExerciseNotFound, got %v", err)
	}
}

func TestExerciseRepository_ModifyExercise_DuplicateName(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	created, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseName: "Skull Crusher", UserId: 1})
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}

	// "Custom Test Exercise" is already seeded for user 1.
	_, err = repo.ModifyExercise(ctx, exercise.Exercise{ExerciseId: created.ExerciseId, ExerciseName: "Custom Test Exercise", UserId: 1})
	if !errors.Is(err, exercise.ErrExerciseAlreadyExists) {
		t.Fatalf("Expected ErrExerciseAlreadyExists, got %v", err)
	}
}

func TestExerciseRepository_ModifyExercise_DuplicateName_CaseInsensitive(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	created, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseName: "Leg Press", UserId: 1})
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}

	_, err = repo.ModifyExercise(ctx, exercise.Exercise{ExerciseId: created.ExerciseId, ExerciseName: "custom test exercise", UserId: 1})
	if !errors.Is(err, exercise.ErrExerciseAlreadyExists) {
		t.Fatalf("Expected ErrExerciseAlreadyExists for a case-insensitive duplicate, got %v", err)
	}
}

func TestExerciseRepository_ModifyExercise_DuplicatesGlobalExercise(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	created, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseName: "Leg Curl", UserId: 1})
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}

	// "Bench Press" is a seeded global exercise.
	_, err = repo.ModifyExercise(ctx, exercise.Exercise{ExerciseId: created.ExerciseId, ExerciseName: "Bench Press", UserId: 1})
	if !errors.Is(err, exercise.ErrExerciseAlreadyExists) {
		t.Fatalf("Expected ErrExerciseAlreadyExists, got %v", err)
	}
}

func TestExerciseRepository_ModifyExercise_DuplicatesGlobalExercise_CaseInsensitive(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	created, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseName: "Leg Extension", UserId: 1})
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}

	_, err = repo.ModifyExercise(ctx, exercise.Exercise{ExerciseId: created.ExerciseId, ExerciseName: "bench press", UserId: 1})
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
