package exercise_test

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/testutil"
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

	if len(exercises) != 48 {
		t.Errorf("Expected 48 exercises got: %d", len(exercises))
	}

	expectedNames := []string{
		"Bench Press (Barbell)",
		"Squat (Barbell)",
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

	if len(exercises) != 49 {
		t.Errorf("Expected 49 exercises got: %d", len(exercises))
	}

	expectedNames := []string{
		"Bench Press (Barbell)",
		"Squat (Barbell)",
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

	toCreate := exercise.Exercise{ExerciseId: uuid.NewV7(), ExerciseName: "Lunge", UserId: 1}
	created, isNew, err := repo.CreateExercise(ctx, toCreate)
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}

	if created != toCreate || !isNew {
		t.Errorf("CreateExercise() = %+v, %v, want %+v, true", created, isNew, toCreate)
	}

	exercises, err := repo.GetExercisesByUserId(ctx, 1)
	if err != nil {
		t.Fatalf("GetExercises returned error: %v", err)
	}
	if !containsExercise(exercises, toCreate.ExerciseId, "Lunge") {
		t.Errorf("expected newly created exercise to appear in GetExercises, got %+v", exercises)
	}
}

func TestExerciseRepository_CreateExercise_Retried(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	original := exercise.Exercise{ExerciseId: uuid.NewV7(), ExerciseName: "Hip Thrust", UserId: 1}
	if _, _, err := repo.CreateExercise(ctx, original); err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}

	// A retry gets back what the first attempt stored, even if its name has changed since.
	for _, name := range []string{"Hip Thrust", "Glute Bridge"} {
		retried, isNew, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseId: original.ExerciseId, ExerciseName: name, UserId: 1})
		if err != nil {
			t.Fatalf("CreateExercise returned error: %v", err)
		}
		if retried != original || isNew {
			t.Errorf("retry as %q: CreateExercise() = %+v, %v, want %+v, false", name, retried, isNew, original)
		}
	}

	exercises, err := repo.GetExercisesByUserId(ctx, 1)
	if err != nil {
		t.Fatalf("GetExercises returned error: %v", err)
	}
	if containsExerciseName(exercises, "Glute Bridge") {
		t.Errorf("expected the retry not to store anything, got %+v", exercises)
	}
}

func TestExerciseRepository_CreateExercise_ReturnsExistingExerciseWithSameName(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}
	benchPressId, err := testutil.GlobalExerciseId(ctx, testPool, "Bench Press (Barbell)")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		want exercise.Exercise
	}{
		// "Custom Test Exercise" is seeded for user 1.
		{"Custom Test Exercise", exercise.Exercise{ExerciseId: testutil.Id(1), ExerciseName: "Custom Test Exercise", UserId: 1}},
		{"custom test exercise", exercise.Exercise{ExerciseId: testutil.Id(1), ExerciseName: "Custom Test Exercise", UserId: 1}},
		{"Bench Press (Barbell)", exercise.Exercise{ExerciseId: benchPressId, ExerciseName: "Bench Press (Barbell)"}},
		{"bench press (barbell)", exercise.Exercise{ExerciseId: benchPressId, ExerciseName: "Bench Press (Barbell)"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestedId := uuid.NewV7()
			got, isNew, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseId: requestedId, ExerciseName: tt.name, UserId: 1})
			if err != nil {
				t.Fatalf("CreateExercise returned error: %v", err)
			}
			if got != tt.want || isNew {
				t.Errorf("CreateExercise() = %+v, %v, want %+v, false", got, isNew, tt.want)
			}
			if exerciseExists(t, requestedId) {
				t.Errorf("expected nothing to be stored under the requested id %v", requestedId)
			}
		})
	}
}

func TestExerciseRepository_CreateExercise_ConcurrentlyUnderTheSameName(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	// Each attempt is a different device with its own id for the exercise.
	const attempts = 8
	results := make(chan exercise.Exercise, attempts)
	created := make(chan bool, attempts)
	var wg sync.WaitGroup
	for range attempts {
		wg.Go(func() {
			stored, isNew, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseId: uuid.NewV7(), ExerciseName: "Landmine Press", UserId: 1})
			if err != nil {
				t.Errorf("CreateExercise returned error: %v", err)
				return
			}
			results <- stored
			created <- isNew
		})
	}
	wg.Wait()
	close(results)
	close(created)

	createdCount := 0
	for isNew := range created {
		if isNew {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Errorf("expected exactly one attempt to create the exercise, got %d", createdCount)
	}

	ids := make(map[uuid.UUID]bool)
	for stored := range results {
		ids[stored.ExerciseId] = true
	}
	if len(ids) != 1 {
		t.Errorf("expected every attempt to end up with the same exercise, got %d different ones", len(ids))
	}
}

// A retry must get back the exercise it created, even when its name now belongs
// to another exercise, or the client would swap its id to the wrong one.
func TestExerciseRepository_CreateExercise_RetryWinsOverSameName(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	// Stored after the exercise holding the name, so only the lookup's order can pick it.
	first := exercise.Exercise{ExerciseId: uuid.NewV7(), ExerciseName: "Pendlay Row (Dumbbell)", UserId: 1}
	second := exercise.Exercise{ExerciseId: uuid.NewV7(), ExerciseName: "Seal Row", UserId: 1}
	for _, e := range []exercise.Exercise{second, first} {
		if _, _, err := repo.CreateExercise(ctx, e); err != nil {
			t.Fatalf("CreateExercise returned error: %v", err)
		}
	}

	got, isNew, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseId: first.ExerciseId, ExerciseName: "seal row", UserId: 1})
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}
	if got != first || isNew {
		t.Errorf("CreateExercise() = %+v, %v, want the retried exercise %+v, false", got, isNew, first)
	}
}

// A global exercise can be added after a user already has one with its name.
// The user's own then wins, since their history is logged against it.
func TestExerciseRepository_CreateExercise_OwnNameWinsOverGlobal(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	globalId := uuid.NewV7()
	if _, err := testPool.Exec(ctx, `INSERT INTO exercises (exercise_id, exercise_name) VALUES ($1, 'Custom Test Exercise')`, globalId); err != nil {
		t.Fatalf("failed to insert global exercise: %v", err)
	}
	t.Cleanup(func() {
		if _, err := testPool.Exec(context.Background(), `DELETE FROM exercises WHERE exercise_id = $1`, globalId); err != nil {
			t.Errorf("failed to delete global exercise: %v", err)
		}
	})

	// "Custom Test Exercise" (1) is seeded for user 1.
	got, isNew, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseId: uuid.NewV7(), ExerciseName: "custom test exercise", UserId: 1})
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}
	want := exercise.Exercise{ExerciseId: testutil.Id(1), ExerciseName: "Custom Test Exercise", UserId: 1}
	if got != want || isNew {
		t.Errorf("CreateExercise() = %+v, %v, want the user's own %+v, false", got, isNew, want)
	}
}

func TestExerciseRepository_CreateExercise_IdOfAnotherUsersExercise(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	// Exercise 2 is seeded for user 2.
	_, _, err = repo.CreateExercise(ctx, exercise.Exercise{ExerciseId: testutil.Id(2), ExerciseName: "Taken Id Exercise", UserId: 1})
	if !errors.Is(err, exercise.ErrExerciseIdTaken) {
		t.Fatalf("Expected ErrExerciseIdTaken, got %v", err)
	}

	exercises, err := repo.GetExercisesByUserId(ctx, 2)
	if err != nil {
		t.Fatalf("GetExercises returned error: %v", err)
	}
	if !containsExercise(exercises, testutil.Id(2), "Second User Exercise") {
		t.Errorf("expected user 2's exercise to be untouched, got %+v", exercises)
	}
}

func TestExerciseRepository_CreateExercise_SameNameWinsOverTakenId(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	// The id belongs to user 2's exercise, but the name to user 1's own, which
	// is all the client needs to switch to.
	got, isNew, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseId: testutil.Id(2), ExerciseName: "Custom Test Exercise", UserId: 1})
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}
	want := exercise.Exercise{ExerciseId: testutil.Id(1), ExerciseName: "Custom Test Exercise", UserId: 1}
	if got != want || isNew {
		t.Errorf("CreateExercise() = %+v, %v, want %+v, false", got, isNew, want)
	}
}

func TestExerciseRepository_CreateExercise_UserNotFound(t *testing.T) {
	ctx := t.Context()
	repo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("NewExerciseRepository returned error: %v", err)
	}

	_, _, err = repo.CreateExercise(ctx, exercise.Exercise{ExerciseId: uuid.NewV7(), ExerciseName: "Ghost Exercise", UserId: 999999})
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

	created, _, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseId: uuid.NewV7(), ExerciseName: "Front Squat", UserId: 1})
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}

	modified, err := repo.ModifyExercise(ctx, exercise.Exercise{ExerciseId: created.ExerciseId, ExerciseName: "Romanian Deadlift", UserId: 1})
	if err != nil {
		t.Fatalf("ModifyExercise returned error: %v", err)
	}

	if modified.ExerciseId != created.ExerciseId {
		t.Errorf("expected ExerciseId %v, got %v", created.ExerciseId, modified.ExerciseId)
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

	benchPressId, err := testutil.GlobalExerciseId(ctx, testPool, "Bench Press (Barbell)")
	if err != nil {
		t.Fatal(err)
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

	_, err = repo.ModifyExercise(ctx, exercise.Exercise{ExerciseId: uuid.NewV7(), ExerciseName: "Ghost Exercise", UserId: 1})
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

	created, _, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseId: uuid.NewV7(), ExerciseName: "Incline Bench", UserId: 1})
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}

	// user 2 does not own this exercise, so it should look not found to them.
	_, err = repo.ModifyExercise(ctx, exercise.Exercise{ExerciseId: created.ExerciseId, ExerciseName: "Wrong User Rename", UserId: 2})
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

	created, _, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseId: uuid.NewV7(), ExerciseName: "Skull Crusher", UserId: 1})
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

	created, _, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseId: uuid.NewV7(), ExerciseName: "Leg Press", UserId: 1})
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

	created, _, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseId: uuid.NewV7(), ExerciseName: "Leg Curl", UserId: 1})
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}

	// "Bench Press (Barbell)" is a seeded global exercise.
	_, err = repo.ModifyExercise(ctx, exercise.Exercise{ExerciseId: created.ExerciseId, ExerciseName: "Bench Press (Barbell)", UserId: 1})
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

	created, _, err := repo.CreateExercise(ctx, exercise.Exercise{ExerciseId: uuid.NewV7(), ExerciseName: "Leg Extension", UserId: 1})
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}

	_, err = repo.ModifyExercise(ctx, exercise.Exercise{ExerciseId: created.ExerciseId, ExerciseName: "bench press (barbell)", UserId: 1})
	if !errors.Is(err, exercise.ErrExerciseAlreadyExists) {
		t.Fatalf("Expected ErrExerciseAlreadyExists for a case-insensitive duplicate, got %v", err)
	}
}

func containsExercise(exercises []exercise.Exercise, exerciseId uuid.UUID, name string) bool {
	for _, e := range exercises {
		if e.ExerciseId == exerciseId && e.ExerciseName == name {
			return true
		}
	}
	return false
}

func exerciseExists(t *testing.T, exerciseId uuid.UUID) bool {
	t.Helper()
	var exists bool
	if err := testPool.QueryRow(t.Context(), `SELECT EXISTS (SELECT 1 FROM exercises WHERE exercise_id = $1)`, exerciseId).Scan(&exists); err != nil {
		t.Fatalf("failed to look up exercise %v: %v", exerciseId, err)
	}
	return exists
}

func containsExerciseName(exercises []exercise.Exercise, name string) bool {
	for _, e := range exercises {
		if e.ExerciseName == name {
			return true
		}
	}
	return false
}
