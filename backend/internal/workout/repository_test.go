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
