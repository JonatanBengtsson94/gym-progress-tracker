package workout_test

import (
	"context"
	"errors"
	"log"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/set"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/testutil"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/workout"
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

func TestWorkoutRepository_CreateWorkout_ExistingTemplate(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	completedAt := time.Date(2024, 2, 1, 9, 30, 0, 0, time.UTC)
	created, err := repo.CreateWorkout(ctx, 1, workout.Workout{
		CompletedAt: completedAt,
		Template:    template.Template{TemplateId: 1},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: 1}, Reps: 8, WeightGrams: 60000},
			{Exercise: exercise.Exercise{ExerciseId: 2}, Reps: 5, WeightGrams: 100000},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkout returned error: %v", err)
	}

	if created.WorkoutId == 0 {
		t.Error("expected CreateWorkout to return a generated WorkoutId")
	}
	if created.Template.TemplateId != 1 || created.Template.TemplateName != "Push Day" {
		t.Errorf("expected the existing template to be reused, got %+v", created.Template)
	}
	if len(created.Sets) != 2 {
		t.Fatalf("expected 2 sets, got %d: %+v", len(created.Sets), created.Sets)
	}
	if created.Sets[0].Exercise.ExerciseName != "Bench Press" {
		t.Errorf("expected exercise names to be resolved, got %q", created.Sets[0].Exercise.ExerciseName)
	}

	got, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, created.WorkoutId)
	if err != nil {
		t.Fatalf("GetWorkoutByUserIdAndWorkoutId returned error: %v", err)
	}
	if !got.CompletedAt.Equal(completedAt) {
		t.Errorf("expected CompletedAt %v, got %v", completedAt, got.CompletedAt)
	}
	if len(got.Sets) != 2 {
		t.Errorf("expected 2 persisted sets, got %d: %+v", len(got.Sets), got.Sets)
	}
}

func TestWorkoutRepository_CreateWorkout_GeneratesTemplate(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	created, err := repo.CreateWorkout(ctx, 1, workout.Workout{
		CompletedAt: time.Date(2024, 2, 2, 9, 30, 0, 0, time.UTC),
		Template:    template.Template{TemplateName: "Generated Leg Day"},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: 2}, Reps: 5, WeightGrams: 100000},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkout returned error: %v", err)
	}

	if created.Template.TemplateId == 0 {
		t.Fatal("expected CreateWorkout to return a generated TemplateId")
	}

	got, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, created.WorkoutId)
	if err != nil {
		t.Fatalf("GetWorkoutByUserIdAndWorkoutId returned error: %v", err)
	}
	if got.Template.TemplateId != created.Template.TemplateId || got.Template.TemplateName != "Generated Leg Day" {
		t.Errorf("expected workout to be stored under the generated template, got %+v", got.Template)
	}
}

func TestWorkoutRepository_CreateWorkout_DuplicateTemplateName(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	// "Push Day" already exists for user 1.
	_, err := repo.CreateWorkout(ctx, 1, workout.Workout{
		CompletedAt: time.Date(2024, 2, 3, 9, 30, 0, 0, time.UTC),
		Template:    template.Template{TemplateName: "push day"},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: 1}, Reps: 8, WeightGrams: 60000},
		},
	})
	if !errors.Is(err, template.ErrTemplateAlreadyExists) {
		t.Fatalf("Expected ErrTemplateAlreadyExists, got %v", err)
	}
}

func TestWorkoutRepository_CreateWorkout_TemplateNotFound(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	// Template 2 belongs to user 2, and the last id is beyond what the int4
	// column can hold, so no such template can exist either.
	for _, templateId := range []uint32{2, 999999, math.MaxInt32 + 1} {
		_, err := repo.CreateWorkout(ctx, 1, workout.Workout{
			CompletedAt: time.Date(2024, 2, 4, 9, 30, 0, 0, time.UTC),
			Template:    template.Template{TemplateId: templateId},
			Sets: []set.Set{
				{Exercise: exercise.Exercise{ExerciseId: 1}, Reps: 8, WeightGrams: 60000},
			},
		})
		if !errors.Is(err, workout.ErrTemplateNotFound) {
			t.Errorf("template %d: expected ErrTemplateNotFound, got %v", templateId, err)
		}
	}
}

func TestWorkoutRepository_CreateWorkout_ExerciseNotFound(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	// Exercise 100 is a custom exercise owned by user 2, and the last id is
	// beyond what the int4 column can hold.
	for _, exerciseId := range []uint32{100, 999999, math.MaxInt32 + 1} {
		_, err := repo.CreateWorkout(ctx, 1, workout.Workout{
			CompletedAt: time.Date(2024, 2, 5, 9, 30, 0, 0, time.UTC),
			Template:    template.Template{TemplateId: 1},
			Sets: []set.Set{
				{Exercise: exercise.Exercise{ExerciseId: exerciseId}, Reps: 8, WeightGrams: 60000},
			},
		})
		if !errors.Is(err, workout.ErrExerciseNotFound) {
			t.Errorf("exercise %d: expected ErrExerciseNotFound, got %v", exerciseId, err)
		}
	}
}

func TestWorkoutRepository_CreateWorkout_WeightGramsOutOfRange(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	_, err := repo.CreateWorkout(ctx, 1, workout.Workout{
		CompletedAt: time.Date(2024, 2, 7, 9, 30, 0, 0, time.UTC),
		Template:    template.Template{TemplateId: 1},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: 1}, Reps: 8, WeightGrams: math.MaxInt32 + 1},
		},
	})
	if !errors.Is(err, workout.ErrWeightGramsOutOfRange) {
		t.Fatalf("Expected ErrWeightGramsOutOfRange, got %v", err)
	}
}

func TestWorkoutRepository_CreateWorkout_RollsBackGeneratedTemplate(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	_, err := repo.CreateWorkout(ctx, 1, workout.Workout{
		CompletedAt: time.Date(2024, 2, 6, 9, 30, 0, 0, time.UTC),
		Template:    template.Template{TemplateName: "Rolled Back Day"},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: 999999}, Reps: 8, WeightGrams: 60000},
		},
	})
	if !errors.Is(err, workout.ErrExerciseNotFound) {
		t.Fatalf("Expected ErrExerciseNotFound, got %v", err)
	}

	var templates int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM templates WHERE user_id = 1 AND template_name = 'Rolled Back Day'`,
	).Scan(&templates); err != nil {
		t.Fatalf("failed to count templates: %v", err)
	}
	if templates != 0 {
		t.Errorf("expected the generated template to be rolled back, found %d", templates)
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

// createModifiableWorkout stores a fresh two-set workout for user 1 under
// template 1, so modify tests never touch the shared seeded workouts.
func createModifiableWorkout(t *testing.T, repo *workout.PostgresWorkoutRepository) workout.Workout {
	t.Helper()

	created, err := repo.CreateWorkout(t.Context(), 1, workout.Workout{
		CompletedAt: time.Date(2024, 3, 1, 18, 0, 0, 0, time.UTC),
		Template:    template.Template{TemplateId: 1},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: 1}, Reps: 8, WeightGrams: 60000},
			{Exercise: exercise.Exercise{ExerciseId: 2}, Reps: 5, WeightGrams: 100000},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkout returned error: %v", err)
	}
	return created
}

func TestWorkoutRepository_ModifyWorkout(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)
	created := createModifiableWorkout(t, repo)

	completedAt := time.Date(2024, 3, 2, 19, 15, 0, 0, time.UTC)
	modified, err := repo.ModifyWorkout(ctx, 1, workout.Workout{
		WorkoutId:   created.WorkoutId,
		CompletedAt: completedAt,
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: 3}, Reps: 10, WeightGrams: 40000},
			{Exercise: exercise.Exercise{ExerciseId: 1}, Reps: 6, WeightGrams: 70000},
			{Exercise: exercise.Exercise{ExerciseId: 1}, Reps: 4, WeightGrams: 75000},
		},
	})
	if err != nil {
		t.Fatalf("ModifyWorkout returned error: %v", err)
	}

	if modified.WorkoutId != created.WorkoutId {
		t.Errorf("expected WorkoutId %d, got %d", created.WorkoutId, modified.WorkoutId)
	}
	if modified.Template.TemplateId != 1 || modified.Template.TemplateName != "Push Day" {
		t.Errorf("expected the template to be returned unchanged, got %+v", modified.Template)
	}
	if len(modified.Sets) != 3 || modified.Sets[1].Exercise.ExerciseName != "Bench Press" {
		t.Errorf("expected 3 sets with resolved exercise names, got %+v", modified.Sets)
	}

	got, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, created.WorkoutId)
	if err != nil {
		t.Fatalf("GetWorkoutByUserIdAndWorkoutId returned error: %v", err)
	}
	if !got.CompletedAt.Equal(completedAt) {
		t.Errorf("expected CompletedAt %v, got %v", completedAt, got.CompletedAt)
	}
	if got.Template.TemplateId != 1 {
		t.Errorf("expected the template to be unchanged, got %+v", got.Template)
	}
	if len(got.Sets) != 3 {
		t.Fatalf("expected the old sets to be replaced by 3 new ones, got %d: %+v", len(got.Sets), got.Sets)
	}
	wantReps := []uint8{10, 6, 4}
	for i, s := range got.Sets {
		if s.Reps != wantReps[i] {
			t.Errorf("set %d: expected Reps %d, got %d (sets must keep their submitted order)", i, wantReps[i], s.Reps)
		}
	}
}

func TestWorkoutRepository_ModifyWorkout_IsIdempotent(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)
	created := createModifiableWorkout(t, repo)

	toModify := workout.Workout{
		WorkoutId:   created.WorkoutId,
		CompletedAt: time.Date(2024, 3, 2, 19, 15, 0, 0, time.UTC),
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: 1}, Reps: 6, WeightGrams: 70000},
		},
	}

	for range 2 {
		if _, err := repo.ModifyWorkout(ctx, 1, toModify); err != nil {
			t.Fatalf("ModifyWorkout returned error: %v", err)
		}
	}

	got, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, created.WorkoutId)
	if err != nil {
		t.Fatalf("GetWorkoutByUserIdAndWorkoutId returned error: %v", err)
	}
	if len(got.Sets) != 1 {
		t.Errorf("expected repeating the same modify to leave 1 set, got %d: %+v", len(got.Sets), got.Sets)
	}
}

func TestWorkoutRepository_ModifyWorkout_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	// The last id is beyond what the int4 column can hold.
	for _, workoutId := range []uint32{999999, math.MaxInt32 + 1} {
		_, err := repo.ModifyWorkout(ctx, 1, workout.Workout{
			WorkoutId:   workoutId,
			CompletedAt: time.Date(2024, 3, 2, 19, 15, 0, 0, time.UTC),
			Sets: []set.Set{
				{Exercise: exercise.Exercise{ExerciseId: 1}, Reps: 6, WeightGrams: 70000},
			},
		})
		if !errors.Is(err, workout.ErrWorkoutNotFound) {
			t.Errorf("workout %d: expected ErrWorkoutNotFound, got %v", workoutId, err)
		}
	}
}

func TestWorkoutRepository_ModifyWorkout_WrongUser(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)
	created := createModifiableWorkout(t, repo)

	// The workout belongs to user 1, so it should look not found to user 2.
	_, err := repo.ModifyWorkout(ctx, 2, workout.Workout{
		WorkoutId:   created.WorkoutId,
		CompletedAt: time.Date(2024, 3, 2, 19, 15, 0, 0, time.UTC),
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: 3}, Reps: 1, WeightGrams: 1000},
		},
	})
	if !errors.Is(err, workout.ErrWorkoutNotFound) {
		t.Fatalf("Expected ErrWorkoutNotFound, got %v", err)
	}

	got, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, created.WorkoutId)
	if err != nil {
		t.Fatalf("GetWorkoutByUserIdAndWorkoutId returned error: %v", err)
	}
	if len(got.Sets) != 2 || !got.CompletedAt.Equal(created.CompletedAt) {
		t.Errorf("expected the workout to be untouched by another user, got %+v", got)
	}
}

func TestWorkoutRepository_ModifyWorkout_RollsBackOnInvalidSets(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	tests := []struct {
		name    string
		set     set.Set
		wantErr error
	}{
		// Exercise 100 is a custom exercise owned by user 2.
		{"other user's exercise", set.Set{Exercise: exercise.Exercise{ExerciseId: 100}, Reps: 8, WeightGrams: 60000}, workout.ErrExerciseNotFound},
		{"unknown exercise", set.Set{Exercise: exercise.Exercise{ExerciseId: 999999}, Reps: 8, WeightGrams: 60000}, workout.ErrExerciseNotFound},
		{"weight out of range", set.Set{Exercise: exercise.Exercise{ExerciseId: 1}, Reps: 8, WeightGrams: math.MaxInt32 + 1}, workout.ErrWeightGramsOutOfRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created := createModifiableWorkout(t, repo)

			_, err := repo.ModifyWorkout(ctx, 1, workout.Workout{
				WorkoutId:   created.WorkoutId,
				CompletedAt: time.Date(2024, 3, 2, 19, 15, 0, 0, time.UTC),
				Sets:        []set.Set{tt.set},
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Expected %v, got %v", tt.wantErr, err)
			}

			got, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, created.WorkoutId)
			if err != nil {
				t.Fatalf("GetWorkoutByUserIdAndWorkoutId returned error: %v", err)
			}
			if len(got.Sets) != 2 || !got.CompletedAt.Equal(created.CompletedAt) {
				t.Errorf("expected the failed modify to be rolled back, got %+v", got)
			}
		})
	}
}

func TestWorkoutRepository_GetWorkout_IdOutOfRange(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	_, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, math.MaxInt32+1)
	if !errors.Is(err, workout.ErrWorkoutNotFound) {
		t.Fatalf("Expected ErrWorkoutNotFound, got %v", err)
	}
}

// createWorkoutCompletedAt stores a one-set workout for user 1 under template 1.
func createWorkoutCompletedAt(t *testing.T, repo *workout.PostgresWorkoutRepository, completedAt time.Time) workout.Workout {
	t.Helper()

	created, err := repo.CreateWorkout(t.Context(), 1, workout.Workout{
		CompletedAt: completedAt,
		Template:    template.Template{TemplateId: 1},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: 1}, Reps: 8, WeightGrams: 60000},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkout returned error: %v", err)
	}
	return created
}

func workoutIndex(workouts []workout.Workout, workoutId uint32) int {
	for i, w := range workouts {
		if w.WorkoutId == workoutId {
			return i
		}
	}
	return -1
}

func TestWorkoutRepository_GetWorkouts(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	// User 2 only owns seeded workout 2, so the whole list can be asserted.
	got, err := repo.GetWorkoutsByUserId(ctx, 2)
	if err != nil {
		t.Fatalf("GetWorkoutsByUserId returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 workout, got %d: %+v", len(got), got)
	}
	w := got[0]
	if w.WorkoutId != 2 {
		t.Errorf("expected WorkoutId 2, got %d", w.WorkoutId)
	}
	if w.Template.TemplateId != 2 || w.Template.TemplateName != "Pull Day" {
		t.Errorf("expected template {2 Pull Day}, got %+v", w.Template)
	}
	if want := time.Date(2024, 1, 16, 11, 0, 0, 0, time.UTC); !w.CompletedAt.Equal(want) {
		t.Errorf("expected CompletedAt %v, got %v", want, w.CompletedAt)
	}
	if len(w.Sets) != 0 {
		t.Errorf("expected the list to carry no sets, got %+v", w.Sets)
	}
}

func TestWorkoutRepository_GetWorkouts_OnlyOwnWorkouts(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	got, err := repo.GetWorkoutsByUserId(ctx, 1)
	if err != nil {
		t.Fatalf("GetWorkoutsByUserId returned error: %v", err)
	}

	i := workoutIndex(got, 1)
	if i == -1 {
		t.Fatalf("expected user 1's seeded workout 1 to be listed, got %+v", got)
	}
	if got[i].Template.TemplateName != "Push Day" {
		t.Errorf("expected TemplateName %q, got %q", "Push Day", got[i].Template.TemplateName)
	}
	// Workout 2 belongs to user 2.
	if workoutIndex(got, 2) != -1 {
		t.Errorf("expected user 2's workout to be hidden from user 1, got %+v", got)
	}
}

func TestWorkoutRepository_GetWorkouts_NewestFirst(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	older := createWorkoutCompletedAt(t, repo, time.Date(2030, 1, 1, 10, 0, 0, 0, time.UTC))
	newer := createWorkoutCompletedAt(t, repo, time.Date(2030, 1, 2, 10, 0, 0, 0, time.UTC))

	got, err := repo.GetWorkoutsByUserId(ctx, 1)
	if err != nil {
		t.Fatalf("GetWorkoutsByUserId returned error: %v", err)
	}

	olderAt, newerAt := workoutIndex(got, older.WorkoutId), workoutIndex(got, newer.WorkoutId)
	if olderAt == -1 || newerAt == -1 {
		t.Fatalf("expected both new workouts to be listed, got %+v", got)
	}
	if newerAt > olderAt {
		t.Errorf("expected the newer workout before the older one, got positions %d and %d", newerAt, olderAt)
	}

	// The whole list must be ordered, not just the workouts created here.
	for i := 1; i < len(got); i++ {
		if got[i].CompletedAt.After(got[i-1].CompletedAt) {
			t.Errorf("workout %d (%v) is listed after older workout %d (%v)",
				got[i].WorkoutId, got[i].CompletedAt, got[i-1].WorkoutId, got[i-1].CompletedAt)
		}
	}
}

func TestWorkoutRepository_GetWorkouts_TiesBrokenByNewestId(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	completedAt := time.Date(2031, 1, 1, 10, 0, 0, 0, time.UTC)
	first := createWorkoutCompletedAt(t, repo, completedAt)
	second := createWorkoutCompletedAt(t, repo, completedAt)

	got, err := repo.GetWorkoutsByUserId(ctx, 1)
	if err != nil {
		t.Fatalf("GetWorkoutsByUserId returned error: %v", err)
	}

	firstAt, secondAt := workoutIndex(got, first.WorkoutId), workoutIndex(got, second.WorkoutId)
	if firstAt == -1 || secondAt == -1 {
		t.Fatalf("expected both new workouts to be listed, got %+v", got)
	}
	if secondAt > firstAt {
		t.Errorf("expected the later-created workout first when times are equal, got positions %d and %d", secondAt, firstAt)
	}
}

func TestWorkoutRepository_GetWorkouts_NoWorkouts(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	got, err := repo.GetWorkoutsByUserId(ctx, 999)
	if err != nil {
		t.Fatalf("GetWorkoutsByUserId returned error: %v", err)
	}

	// A nil slice would serialize as null further up, so it must be non-nil.
	if got == nil || len(got) != 0 {
		t.Errorf("expected an empty, non-nil slice, got %#v", got)
	}
}

func TestWorkoutRepository_GetWorkoutsByTemplate(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	// User 2 only has seeded workout 2 under template 2, so the whole list can be asserted.
	got, err := repo.GetWorkoutsByUserIdAndTemplateId(ctx, 2, 2)
	if err != nil {
		t.Fatalf("GetWorkoutsByUserIdAndTemplateId returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 workout, got %d: %+v", len(got), got)
	}
	w := got[0]
	if w.WorkoutId != 2 {
		t.Errorf("expected WorkoutId 2, got %d", w.WorkoutId)
	}
	if w.Template.TemplateId != 2 || w.Template.TemplateName != "Pull Day" {
		t.Errorf("expected template {2 Pull Day}, got %+v", w.Template)
	}
	if want := time.Date(2024, 1, 16, 11, 0, 0, 0, time.UTC); !w.CompletedAt.Equal(want) {
		t.Errorf("expected CompletedAt %v, got %v", want, w.CompletedAt)
	}
	if len(w.Sets) != 0 {
		t.Errorf("expected the list to carry no sets, got %+v", w.Sets)
	}
}

func TestWorkoutRepository_GetWorkoutsByTemplate_OnlyThatTemplate(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	// A workout under a second template of user 1 must not show up under template 1.
	other, err := repo.CreateWorkout(ctx, 1, workout.Workout{
		CompletedAt: time.Date(2035, 1, 1, 10, 0, 0, 0, time.UTC),
		Template:    template.Template{TemplateName: "Leg Day By Template Test"},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: 1}, Reps: 8, WeightGrams: 60000},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkout returned error: %v", err)
	}
	own := createWorkoutCompletedAt(t, repo, time.Date(2035, 1, 2, 10, 0, 0, 0, time.UTC))

	got, err := repo.GetWorkoutsByUserIdAndTemplateId(ctx, 1, 1)
	if err != nil {
		t.Fatalf("GetWorkoutsByUserIdAndTemplateId returned error: %v", err)
	}

	if workoutIndex(got, own.WorkoutId) == -1 || workoutIndex(got, 1) == -1 {
		t.Errorf("expected template 1's workouts to be listed, got %+v", got)
	}
	if workoutIndex(got, other.WorkoutId) != -1 {
		t.Errorf("expected the workout under another template to be excluded, got %+v", got)
	}
	for _, w := range got {
		if w.Template.TemplateId != 1 || w.Template.TemplateName != "Push Day" {
			t.Errorf("expected only template {1 Push Day}, got %+v", w)
		}
	}

	otherList, err := repo.GetWorkoutsByUserIdAndTemplateId(ctx, 1, other.Template.TemplateId)
	if err != nil {
		t.Fatalf("GetWorkoutsByUserIdAndTemplateId returned error: %v", err)
	}
	if len(otherList) != 1 || otherList[0].WorkoutId != other.WorkoutId {
		t.Errorf("expected only workout %d under the new template, got %+v", other.WorkoutId, otherList)
	}
}

func TestWorkoutRepository_GetWorkoutsByTemplate_NewestFirst(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	completedAt := time.Date(2036, 1, 1, 10, 0, 0, 0, time.UTC)
	older := createWorkoutCompletedAt(t, repo, completedAt.Add(-time.Hour))
	first := createWorkoutCompletedAt(t, repo, completedAt)
	second := createWorkoutCompletedAt(t, repo, completedAt)

	got, err := repo.GetWorkoutsByUserIdAndTemplateId(ctx, 1, 1)
	if err != nil {
		t.Fatalf("GetWorkoutsByUserIdAndTemplateId returned error: %v", err)
	}

	olderAt, firstAt, secondAt := workoutIndex(got, older.WorkoutId), workoutIndex(got, first.WorkoutId), workoutIndex(got, second.WorkoutId)
	if olderAt == -1 || firstAt == -1 || secondAt == -1 {
		t.Fatalf("expected all new workouts to be listed, got %+v", got)
	}
	// Equal times are broken by the newest id, like the unfiltered list.
	if !(secondAt < firstAt && firstAt < olderAt) {
		t.Errorf("expected order second, first, older; got positions %d, %d, %d", secondAt, firstAt, olderAt)
	}

	for i := 1; i < len(got); i++ {
		if got[i].CompletedAt.After(got[i-1].CompletedAt) {
			t.Errorf("workout %d (%v) is listed after older workout %d (%v)",
				got[i].WorkoutId, got[i].CompletedAt, got[i-1].WorkoutId, got[i-1].CompletedAt)
		}
	}
}

func TestWorkoutRepository_GetWorkoutsByTemplate_NoWorkouts(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	var templateId uint32
	if err := testPool.QueryRow(ctx,
		`INSERT INTO templates (user_id, template_name) VALUES (1, 'Unused Template') RETURNING template_id`,
	).Scan(&templateId); err != nil {
		t.Fatalf("failed to insert template: %v", err)
	}

	got, err := repo.GetWorkoutsByUserIdAndTemplateId(ctx, 1, templateId)
	if err != nil {
		t.Fatalf("GetWorkoutsByUserIdAndTemplateId returned error: %v", err)
	}

	// A nil slice would serialize as null further up, so it must be non-nil.
	if got == nil || len(got) != 0 {
		t.Errorf("expected an empty, non-nil slice, got %#v", got)
	}
}

func TestWorkoutRepository_GetWorkoutsByTemplate_TemplateNotFound(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	tests := []struct {
		name       string
		userId     uint32
		templateId uint32
	}{
		{"unknown template", 1, 9999},
		{"other user's template", 1, 2},
		{"id out of range", 1, math.MaxUint32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repo.GetWorkoutsByUserIdAndTemplateId(ctx, tt.userId, tt.templateId)
			if !errors.Is(err, workout.ErrTemplateNotFound) {
				t.Fatalf("Expected ErrTemplateNotFound, got %v", err)
			}
		})
	}
}
