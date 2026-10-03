package workout_test

import (
	"context"
	"errors"
	"log"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/set"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/testutil"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/workout"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

// The ids of the global exercises the tests log sets for.
var benchPressId, dumbbellBenchPressId, smithBenchPressId uuid.UUID

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

	for id, name := range map[*uuid.UUID]string{
		&benchPressId:         "Bench Press (Barbell)",
		&dumbbellBenchPressId: "Bench Press (Dumbbell)",
		&smithBenchPressId:    "Bench Press (Smith Machine)",
	} {
		if *id, err = testutil.GlobalExerciseId(ctx, pool, name); err != nil {
			log.Fatal(err)
		}
	}

	code := m.Run()

	teardown()
	os.Exit(code)
}

func TestWorkoutRepository_GetWorkout(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	got, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, testutil.Id(1))
	if err != nil {
		t.Fatalf("GetWorkoutByUserIdAndWorkoutId returned error: %v", err)
	}

	if got.WorkoutId != testutil.Id(1) {
		t.Errorf("expected WorkoutId 1, got %v", got.WorkoutId)
	}
	if got.Template.TemplateName != "Push Day" {
		t.Errorf("expected TemplateName %q, got %q", "Push Day", got.Template.TemplateName)
	}
	if got.StartedAt.Format("2006-01-02 15:04:05") != "2024-01-15 09:00:00" {
		t.Errorf("expected StartedAt %q, got %q", "2024-01-15 09:00:00", got.StartedAt.Format("2006-01-02 15:04:05"))
	}
	if got.CompletedAt.Format("2006-01-02 15:04:05") != "2024-01-15 10:00:00" {
		t.Errorf("expected CompletedAt %q, got %q", "2024-01-15 10:00:00", got.CompletedAt.Format("2006-01-02 15:04:05"))
	}

	if len(got.Sets) != 2 {
		t.Fatalf("expected 2 sets, got %d: %+v", len(got.Sets), got.Sets)
	}

	for _, s := range got.Sets {
		switch s.Exercise.ExerciseName {
		case "Bench Press (Barbell)":
			if s.Reps != 8 {
				t.Errorf("expected Bench Press Reps 8, got %d", s.Reps)
			}
			if s.WeightGrams != 60000 {
				t.Errorf("expected Bench Press WeightGrams 60000, got %d", s.WeightGrams)
			}
		case "Bench Press (Dumbbell)":
			if s.Reps != 5 {
				t.Errorf("expected Bench Press (Dumbbell) Reps 5, got %d", s.Reps)
			}
			if s.WeightGrams != 100000 {
				t.Errorf("expected Bench Press (Dumbbell) WeightGrams 100000, got %d", s.WeightGrams)
			}
		default:
			t.Errorf("unexpected exercise in workout sets: %q", s.Exercise.ExerciseName)
		}
	}
}

func TestWorkoutRepository_PutWorkout_ExistingTemplate(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	startedAt := time.Date(2024, 2, 1, 8, 15, 0, 0, time.UTC)
	completedAt := time.Date(2024, 2, 1, 9, 30, 0, 0, time.UTC)
	workoutId := uuid.NewV7()
	created, isNew, err := repo.PutWorkout(ctx, 1, workout.Workout{
		WorkoutId:   workoutId,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		Template:    template.Template{TemplateId: testutil.Id(1)},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 8, WeightGrams: 60000},
			{Exercise: exercise.Exercise{ExerciseId: dumbbellBenchPressId}, Reps: 5, WeightGrams: 100000},
		},
	})
	if err != nil {
		t.Fatalf("PutWorkout returned error: %v", err)
	}

	if created.WorkoutId != workoutId || !isNew {
		t.Errorf("expected workout %v to be created, got %v, created %v", workoutId, created.WorkoutId, isNew)
	}
	if created.Template.TemplateId != testutil.Id(1) || created.Template.TemplateName != "Push Day" {
		t.Errorf("expected the existing template to be reused, got %+v", created.Template)
	}
	if len(created.Sets) != 2 {
		t.Fatalf("expected 2 sets, got %d: %+v", len(created.Sets), created.Sets)
	}
	if created.Sets[0].Exercise.ExerciseName != "Bench Press (Barbell)" {
		t.Errorf("expected exercise names to be resolved, got %q", created.Sets[0].Exercise.ExerciseName)
	}

	got, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, created.WorkoutId)
	if err != nil {
		t.Fatalf("GetWorkoutByUserIdAndWorkoutId returned error: %v", err)
	}
	if !got.StartedAt.Equal(startedAt) {
		t.Errorf("expected StartedAt %v, got %v", startedAt, got.StartedAt)
	}
	if !got.CompletedAt.Equal(completedAt) {
		t.Errorf("expected CompletedAt %v, got %v", completedAt, got.CompletedAt)
	}
	if len(got.Sets) != 2 {
		t.Errorf("expected 2 persisted sets, got %d: %+v", len(got.Sets), got.Sets)
	}
}

func TestWorkoutRepository_PutWorkout_GeneratesTemplate(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	created, _, err := repo.PutWorkout(ctx, 1, workout.Workout{
		WorkoutId:   uuid.NewV7(),
		StartedAt:   time.Date(2024, 2, 2, 8, 30, 0, 0, time.UTC),
		CompletedAt: time.Date(2024, 2, 2, 9, 30, 0, 0, time.UTC),
		Template:    template.Template{TemplateName: "Generated Leg Day"},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: dumbbellBenchPressId}, Reps: 5, WeightGrams: 100000},
		},
	})
	if err != nil {
		t.Fatalf("PutWorkout returned error: %v", err)
	}

	if created.Template.TemplateId == uuid.Nil() {
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

func TestWorkoutRepository_PutWorkout_DuplicateTemplateName(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	// "Push Day" already exists for user 1.
	_, _, err := repo.PutWorkout(ctx, 1, workout.Workout{
		WorkoutId:   uuid.NewV7(),
		StartedAt:   time.Date(2024, 2, 3, 8, 30, 0, 0, time.UTC),
		CompletedAt: time.Date(2024, 2, 3, 9, 30, 0, 0, time.UTC),
		Template:    template.Template{TemplateName: "push day"},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 8, WeightGrams: 60000},
		},
	})
	if !errors.Is(err, template.ErrTemplateAlreadyExists) {
		t.Fatalf("Expected ErrTemplateAlreadyExists, got %v", err)
	}
}

func TestWorkoutRepository_PutWorkout_TemplateNotFound(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	// Template 2 belongs to user 2, and the other doesn't exist.
	for _, templateId := range []uuid.UUID{testutil.Id(2), uuid.NewV7()} {
		_, _, err := repo.PutWorkout(ctx, 1, workout.Workout{
			WorkoutId:   uuid.NewV7(),
			StartedAt:   time.Date(2024, 2, 4, 8, 30, 0, 0, time.UTC),
			CompletedAt: time.Date(2024, 2, 4, 9, 30, 0, 0, time.UTC),
			Template:    template.Template{TemplateId: templateId},
			Sets: []set.Set{
				{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 8, WeightGrams: 60000},
			},
		})
		if !errors.Is(err, workout.ErrTemplateNotFound) {
			t.Errorf("template %v: expected ErrTemplateNotFound, got %v", templateId, err)
		}
	}
}

func TestWorkoutRepository_PutWorkout_ExerciseNotFound(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	// Exercise 100 is a custom exercise owned by user 2, and the other doesn't exist.
	for _, exerciseId := range []uuid.UUID{testutil.Id(100), uuid.NewV7()} {
		_, _, err := repo.PutWorkout(ctx, 1, workout.Workout{
			WorkoutId:   uuid.NewV7(),
			StartedAt:   time.Date(2024, 2, 5, 8, 30, 0, 0, time.UTC),
			CompletedAt: time.Date(2024, 2, 5, 9, 30, 0, 0, time.UTC),
			Template:    template.Template{TemplateId: testutil.Id(1)},
			Sets: []set.Set{
				{Exercise: exercise.Exercise{ExerciseId: exerciseId}, Reps: 8, WeightGrams: 60000},
			},
		})
		if !errors.Is(err, workout.ErrExerciseNotFound) {
			t.Errorf("exercise %v: expected ErrExerciseNotFound, got %v", exerciseId, err)
		}
	}
}

func TestWorkoutRepository_PutWorkout_WeightGramsOutOfRange(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	_, _, err := repo.PutWorkout(ctx, 1, workout.Workout{
		WorkoutId:   uuid.NewV7(),
		StartedAt:   time.Date(2024, 2, 7, 8, 30, 0, 0, time.UTC),
		CompletedAt: time.Date(2024, 2, 7, 9, 30, 0, 0, time.UTC),
		Template:    template.Template{TemplateId: testutil.Id(1)},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 8, WeightGrams: math.MaxInt32 + 1},
		},
	})
	if !errors.Is(err, workout.ErrWeightGramsOutOfRange) {
		t.Fatalf("Expected ErrWeightGramsOutOfRange, got %v", err)
	}
}

func TestWorkoutRepository_PutWorkout_RollsBackGeneratedTemplate(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	_, _, err := repo.PutWorkout(ctx, 1, workout.Workout{
		WorkoutId:   uuid.NewV7(),
		StartedAt:   time.Date(2024, 2, 6, 8, 30, 0, 0, time.UTC),
		CompletedAt: time.Date(2024, 2, 6, 9, 30, 0, 0, time.UTC),
		Template:    template.Template{TemplateName: "Rolled Back Day"},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: uuid.NewV7()}, Reps: 8, WeightGrams: 60000},
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

func TestWorkoutRepository_PutWorkout_SameWorkoutTwice(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	toPut := workout.Workout{
		WorkoutId:   uuid.NewV7(),
		StartedAt:   time.Date(2024, 2, 8, 8, 30, 0, 0, time.UTC),
		CompletedAt: time.Date(2024, 2, 8, 9, 30, 0, 0, time.UTC),
		Template:    template.Template{TemplateId: testutil.Id(1)},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 8, WeightGrams: 60000},
		},
	}
	if _, _, err := repo.PutWorkout(ctx, 1, toPut); err != nil {
		t.Fatalf("PutWorkout returned error: %v", err)
	}

	again, isNew, err := repo.PutWorkout(ctx, 1, toPut)
	if err != nil {
		t.Fatalf("PutWorkout returned error: %v", err)
	}
	if isNew || again.WorkoutId != toPut.WorkoutId {
		t.Errorf("expected the second put to replace workout %v, got %v, created %v", toPut.WorkoutId, again.WorkoutId, isNew)
	}

	got, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, toPut.WorkoutId)
	if err != nil {
		t.Fatalf("GetWorkoutByUserIdAndWorkoutId returned error: %v", err)
	}
	if len(got.Sets) != 1 || got.Sets[0].Reps != 8 || !got.CompletedAt.Equal(toPut.CompletedAt) {
		t.Errorf("expected putting the same workout twice to store it once, got %+v", got)
	}
}

func TestWorkoutRepository_PutWorkout_SameWorkoutTwiceWithGeneratedTemplate(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	// The first put creates the template; the second replaces the workout,
	// which keeps its template, so it must not trip over the template's name.
	toCreate := workout.Workout{
		WorkoutId:   uuid.NewV7(),
		StartedAt:   time.Date(2024, 2, 9, 8, 30, 0, 0, time.UTC),
		CompletedAt: time.Date(2024, 2, 9, 9, 30, 0, 0, time.UTC),
		Template:    template.Template{TemplateName: "Retried Template Day"},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 8, WeightGrams: 60000},
		},
	}
	first, _, err := repo.PutWorkout(ctx, 1, toCreate)
	if err != nil {
		t.Fatalf("PutWorkout returned error: %v", err)
	}

	retried, isNew, err := repo.PutWorkout(ctx, 1, toCreate)
	if err != nil {
		t.Fatalf("second PutWorkout returned error: %v", err)
	}
	if isNew || retried.WorkoutId != first.WorkoutId || retried.Template.TemplateId != first.Template.TemplateId {
		t.Errorf("expected the retry to return the stored workout %+v, got %+v, created %v", first, retried, isNew)
	}
}

func TestWorkoutRepository_PutWorkout_ConcurrentRetries(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	toCreate := workout.Workout{
		WorkoutId:   uuid.NewV7(),
		StartedAt:   time.Date(2024, 2, 11, 8, 30, 0, 0, time.UTC),
		CompletedAt: time.Date(2024, 2, 11, 9, 30, 0, 0, time.UTC),
		Template:    template.Template{TemplateId: testutil.Id(1)},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 8, WeightGrams: 60000},
		},
	}

	const attempts = 8
	results := make(chan bool, attempts)
	var wg sync.WaitGroup
	for range attempts {
		wg.Go(func() {
			stored, isNew, err := repo.PutWorkout(ctx, 1, toCreate)
			if err != nil {
				t.Errorf("PutWorkout returned error: %v", err)
				return
			}
			if stored.WorkoutId != toCreate.WorkoutId {
				t.Errorf("expected workout %v, got %v", toCreate.WorkoutId, stored.WorkoutId)
			}
			results <- isNew
		})
	}
	wg.Wait()
	close(results)

	created := 0
	for isNew := range results {
		if isNew {
			created++
		}
	}
	if created != 1 {
		t.Errorf("expected exactly one attempt to create the workout, got %d", created)
	}
}

// A put that finds no workout and then loses the race to create it replaces
// the workout the other request stored, instead of failing.
func TestWorkoutRepository_PutWorkout_CreatedWhileInserting(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)
	workoutId := uuid.NewV7()

	// Another request has inserted the workout but not committed yet.
	tx, err := testPool.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`INSERT INTO workouts (workout_id, template_id, started_at, completed_at) VALUES ($1, $2, '2024-02-12 08:00:00', '2024-02-12 09:00:00')`,
		workoutId, testutil.Id(1),
	); err != nil {
		t.Fatalf("failed to insert workout: %v", err)
	}

	type result struct {
		stored  workout.Workout
		created bool
		err     error
	}
	done := make(chan result, 1)
	go func() {
		stored, created, err := repo.PutWorkout(ctx, 1, workout.Workout{
			WorkoutId:   workoutId,
			StartedAt:   time.Date(2024, 2, 12, 8, 30, 0, 0, time.UTC),
			CompletedAt: time.Date(2024, 2, 12, 9, 30, 0, 0, time.UTC),
			Template:    template.Template{TemplateId: testutil.Id(1)},
			Sets: []set.Set{
				{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 8, WeightGrams: 60000},
			},
		})
		done <- result{stored, created, err}
	}()

	// The put can't see the uncommitted workout, so its insert waits for it.
	waitForLockWait(t)
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	r := <-done
	if r.err != nil {
		t.Fatalf("PutWorkout returned error: %v", r.err)
	}
	if r.created {
		t.Error("expected the put to replace the workout the other request created")
	}

	got, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, workoutId)
	if err != nil {
		t.Fatalf("GetWorkoutByUserIdAndWorkoutId returned error: %v", err)
	}
	if len(got.Sets) != 1 || !got.CompletedAt.Equal(time.Date(2024, 2, 12, 9, 30, 0, 0, time.UTC)) {
		t.Errorf("expected the put's workout to be stored, got %+v", got)
	}
}

// waitForLockWait waits until some query in the test database is blocked on a lock.
func waitForLockWait(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var waiting bool
		if err := testPool.QueryRow(t.Context(),
			`SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock')`,
		).Scan(&waiting); err != nil {
			t.Fatalf("failed to check for lock waits: %v", err)
		}
		if waiting {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for a query to block on a lock")
}

func TestWorkoutRepository_PutWorkout_IdTaken(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	tests := []struct {
		name         string
		workoutId    uuid.UUID
		templateName string
	}{
		// Workout 2 belongs to user 2.
		{"other user's workout", testutil.Id(2), "Taken Id Day"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := repo.PutWorkout(ctx, 1, workout.Workout{
				WorkoutId:   tt.workoutId,
				StartedAt:   time.Date(2024, 2, 10, 8, 30, 0, 0, time.UTC),
				CompletedAt: time.Date(2024, 2, 10, 9, 30, 0, 0, time.UTC),
				Template:    template.Template{TemplateName: tt.templateName},
				Sets: []set.Set{
					{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 8, WeightGrams: 60000},
				},
			})
			if !errors.Is(err, workout.ErrWorkoutIdTaken) {
				t.Fatalf("Expected ErrWorkoutIdTaken, got %v", err)
			}

			var templates int
			if err := testPool.QueryRow(ctx,
				`SELECT count(*) FROM templates WHERE user_id = 1 AND template_name = $1`, tt.templateName,
			).Scan(&templates); err != nil {
				t.Fatalf("failed to count templates: %v", err)
			}
			if templates != 0 {
				t.Errorf("expected the generated template to be rolled back, found %d", templates)
			}
		})
	}
}

func TestWorkoutRepository_GetWorkout_NotFound(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	_, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, uuid.NewV7())
	if !errors.Is(err, workout.ErrWorkoutNotFound) {
		t.Fatalf("Expected ErrWorkoutNotFound, got %v", err)
	}
}

func TestWorkoutRepository_GetWorkout_NoSets(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	// Workout 3 exists but has no sets recorded, so it should look not found.
	_, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, testutil.Id(3))
	if !errors.Is(err, workout.ErrWorkoutNotFound) {
		t.Fatalf("Expected ErrWorkoutNotFound, got %v", err)
	}
}

func TestWorkoutRepository_GetWorkout_WrongUser(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	// Workout 2 belongs to user 2, so it should look not found to user 1.
	_, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, testutil.Id(2))
	if !errors.Is(err, workout.ErrWorkoutNotFound) {
		t.Fatalf("Expected ErrWorkoutNotFound, got %v", err)
	}
}

func TestWorkoutRepository_GetWorkout_UserSeesOwnWorkout(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	got, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 2, testutil.Id(2))
	if err != nil {
		t.Fatalf("GetWorkoutByUserIdAndWorkoutId returned error: %v", err)
	}

	if got.Template.TemplateName != "Pull Day" {
		t.Errorf("expected TemplateName %q, got %q", "Pull Day", got.Template.TemplateName)
	}
	if len(got.Sets) != 1 {
		t.Fatalf("expected 1 set, got %d: %+v", len(got.Sets), got.Sets)
	}
	if got.Sets[0].Exercise.ExerciseName != "Incline Bench Press (Barbell)" {
		t.Errorf("expected exercise %q, got %q", "Incline Bench Press (Barbell)", got.Sets[0].Exercise.ExerciseName)
	}
}

// createModifiableWorkout stores a fresh two-set workout for user 1 under
// template 1, so modify tests never touch the shared seeded workouts.
func createModifiableWorkout(t *testing.T, repo *workout.PostgresWorkoutRepository) workout.Workout {
	t.Helper()

	created, _, err := repo.PutWorkout(t.Context(), 1, workout.Workout{
		WorkoutId:   uuid.NewV7(),
		StartedAt:   time.Date(2024, 3, 1, 17, 0, 0, 0, time.UTC),
		CompletedAt: time.Date(2024, 3, 1, 18, 0, 0, 0, time.UTC),
		Template:    template.Template{TemplateId: testutil.Id(1)},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 8, WeightGrams: 60000},
			{Exercise: exercise.Exercise{ExerciseId: dumbbellBenchPressId}, Reps: 5, WeightGrams: 100000},
		},
	})
	if err != nil {
		t.Fatalf("PutWorkout returned error: %v", err)
	}
	return created
}

func TestWorkoutRepository_PutWorkout_ReplacesExistingWorkout(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)
	created := createModifiableWorkout(t, repo)

	startedAt := time.Date(2024, 3, 2, 18, 0, 0, 0, time.UTC)
	completedAt := time.Date(2024, 3, 2, 19, 15, 0, 0, time.UTC)
	modified, isNew, err := repo.PutWorkout(ctx, 1, workout.Workout{
		WorkoutId:   created.WorkoutId,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		// A workout keeps the template it was logged under.
		Template: template.Template{TemplateName: "Sneaky Day"},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: smithBenchPressId}, Reps: 10, WeightGrams: 40000},
			{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 6, WeightGrams: 70000},
			{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 4, WeightGrams: 75000},
		},
	})
	if err != nil {
		t.Fatalf("PutWorkout returned error: %v", err)
	}

	if modified.WorkoutId != created.WorkoutId || isNew {
		t.Errorf("expected workout %v to be replaced, got %v, created %v", created.WorkoutId, modified.WorkoutId, isNew)
	}
	if modified.Template.TemplateId != testutil.Id(1) || modified.Template.TemplateName != "Push Day" {
		t.Errorf("expected the template to be returned unchanged, got %+v", modified.Template)
	}
	if len(modified.Sets) != 3 || modified.Sets[1].Exercise.ExerciseName != "Bench Press (Barbell)" {
		t.Errorf("expected 3 sets with resolved exercise names, got %+v", modified.Sets)
	}

	got, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, created.WorkoutId)
	if err != nil {
		t.Fatalf("GetWorkoutByUserIdAndWorkoutId returned error: %v", err)
	}
	if !got.StartedAt.Equal(startedAt) {
		t.Errorf("expected StartedAt %v, got %v", startedAt, got.StartedAt)
	}
	if !got.CompletedAt.Equal(completedAt) {
		t.Errorf("expected CompletedAt %v, got %v", completedAt, got.CompletedAt)
	}
	if got.Template.TemplateId != testutil.Id(1) {
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

func TestWorkoutRepository_PutWorkout_ReplacingIsIdempotent(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)
	created := createModifiableWorkout(t, repo)

	toModify := workout.Workout{
		WorkoutId:   created.WorkoutId,
		StartedAt:   time.Date(2024, 3, 2, 18, 15, 0, 0, time.UTC),
		CompletedAt: time.Date(2024, 3, 2, 19, 15, 0, 0, time.UTC),
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 6, WeightGrams: 70000},
		},
	}

	for range 2 {
		if _, _, err := repo.PutWorkout(ctx, 1, toModify); err != nil {
			t.Fatalf("PutWorkout returned error: %v", err)
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

func TestWorkoutRepository_PutWorkout_NewWorkoutWithoutTemplate(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	workoutId := uuid.NewV7()
	_, _, err := repo.PutWorkout(ctx, 1, workout.Workout{
		WorkoutId:   workoutId,
		StartedAt:   time.Date(2024, 3, 2, 18, 15, 0, 0, time.UTC),
		CompletedAt: time.Date(2024, 3, 2, 19, 15, 0, 0, time.UTC),
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 6, WeightGrams: 70000},
		},
	})
	if !errors.Is(err, template.ErrTemplateNameRequired) {
		t.Errorf("expected ErrTemplateNameRequired, got %v", err)
	}
	if _, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, workoutId); !errors.Is(err, workout.ErrWorkoutNotFound) {
		t.Errorf("expected nothing to be stored, got %v", err)
	}
}

func TestWorkoutRepository_PutWorkout_WrongUser(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)
	created := createModifiableWorkout(t, repo)

	// The workout belongs to user 1, so user 2 can neither replace it nor
	// create another under its id.
	_, _, err := repo.PutWorkout(ctx, 2, workout.Workout{
		WorkoutId:   created.WorkoutId,
		StartedAt:   time.Date(2024, 3, 2, 18, 15, 0, 0, time.UTC),
		CompletedAt: time.Date(2024, 3, 2, 19, 15, 0, 0, time.UTC),
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: smithBenchPressId}, Reps: 1, WeightGrams: 1000},
		},
	})
	if !errors.Is(err, workout.ErrWorkoutIdTaken) {
		t.Fatalf("Expected ErrWorkoutIdTaken, got %v", err)
	}

	got, err := repo.GetWorkoutByUserIdAndWorkoutId(ctx, 1, created.WorkoutId)
	if err != nil {
		t.Fatalf("GetWorkoutByUserIdAndWorkoutId returned error: %v", err)
	}
	if len(got.Sets) != 2 || !got.CompletedAt.Equal(created.CompletedAt) {
		t.Errorf("expected the workout to be untouched by another user, got %+v", got)
	}
}

func TestWorkoutRepository_PutWorkout_ReplacingRollsBackOnInvalidSets(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	tests := []struct {
		name    string
		set     set.Set
		wantErr error
	}{
		// Exercise 100 is a custom exercise owned by user 2.
		{"other user's exercise", set.Set{Exercise: exercise.Exercise{ExerciseId: testutil.Id(100)}, Reps: 8, WeightGrams: 60000}, workout.ErrExerciseNotFound},
		{"unknown exercise", set.Set{Exercise: exercise.Exercise{ExerciseId: uuid.NewV7()}, Reps: 8, WeightGrams: 60000}, workout.ErrExerciseNotFound},
		{"weight out of range", set.Set{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 8, WeightGrams: math.MaxInt32 + 1}, workout.ErrWeightGramsOutOfRange},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created := createModifiableWorkout(t, repo)

			_, _, err := repo.PutWorkout(ctx, 1, workout.Workout{
				WorkoutId:   created.WorkoutId,
				StartedAt:   time.Date(2024, 3, 2, 18, 15, 0, 0, time.UTC),
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

// createWorkoutCompletedAt stores a one-hour, one-set workout for user 1
// under template 1.
func createWorkoutCompletedAt(t *testing.T, repo *workout.PostgresWorkoutRepository, completedAt time.Time) workout.Workout {
	t.Helper()

	created, _, err := repo.PutWorkout(t.Context(), 1, workout.Workout{
		WorkoutId:   uuid.NewV7(),
		StartedAt:   completedAt.Add(-time.Hour),
		CompletedAt: completedAt,
		Template:    template.Template{TemplateId: testutil.Id(1)},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 8, WeightGrams: 60000},
		},
	})
	if err != nil {
		t.Fatalf("PutWorkout returned error: %v", err)
	}
	return created
}

func workoutIndex(workouts []workout.Workout, workoutId uuid.UUID) int {
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
	if w.WorkoutId != testutil.Id(2) {
		t.Errorf("expected WorkoutId 2, got %v", w.WorkoutId)
	}
	if w.Template.TemplateId != testutil.Id(2) || w.Template.TemplateName != "Pull Day" {
		t.Errorf("expected template {2 Pull Day}, got %+v", w.Template)
	}
	if want := time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC); !w.StartedAt.Equal(want) {
		t.Errorf("expected StartedAt %v, got %v", want, w.StartedAt)
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

	i := workoutIndex(got, testutil.Id(1))
	if i == -1 {
		t.Fatalf("expected user 1's seeded workout 1 to be listed, got %+v", got)
	}
	if got[i].Template.TemplateName != "Push Day" {
		t.Errorf("expected TemplateName %q, got %q", "Push Day", got[i].Template.TemplateName)
	}
	// Workout 2 belongs to user 2.
	if workoutIndex(got, testutil.Id(2)) != -1 {
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
			t.Errorf("workout %v (%v) is listed after older workout %v (%v)",
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
	got, err := repo.GetWorkoutsByUserIdAndTemplateId(ctx, 2, testutil.Id(2))
	if err != nil {
		t.Fatalf("GetWorkoutsByUserIdAndTemplateId returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 workout, got %d: %+v", len(got), got)
	}
	w := got[0]
	if w.WorkoutId != testutil.Id(2) {
		t.Errorf("expected WorkoutId 2, got %v", w.WorkoutId)
	}
	if w.Template.TemplateId != testutil.Id(2) || w.Template.TemplateName != "Pull Day" {
		t.Errorf("expected template {2 Pull Day}, got %+v", w.Template)
	}
	if want := time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC); !w.StartedAt.Equal(want) {
		t.Errorf("expected StartedAt %v, got %v", want, w.StartedAt)
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
	other, _, err := repo.PutWorkout(ctx, 1, workout.Workout{
		WorkoutId:   uuid.NewV7(),
		StartedAt:   time.Date(2035, 1, 1, 9, 0, 0, 0, time.UTC),
		CompletedAt: time.Date(2035, 1, 1, 10, 0, 0, 0, time.UTC),
		Template:    template.Template{TemplateName: "Leg Day By Template Test"},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: benchPressId}, Reps: 8, WeightGrams: 60000},
		},
	})
	if err != nil {
		t.Fatalf("PutWorkout returned error: %v", err)
	}
	own := createWorkoutCompletedAt(t, repo, time.Date(2035, 1, 2, 10, 0, 0, 0, time.UTC))

	got, err := repo.GetWorkoutsByUserIdAndTemplateId(ctx, 1, testutil.Id(1))
	if err != nil {
		t.Fatalf("GetWorkoutsByUserIdAndTemplateId returned error: %v", err)
	}

	if workoutIndex(got, own.WorkoutId) == -1 || workoutIndex(got, testutil.Id(1)) == -1 {
		t.Errorf("expected template 1's workouts to be listed, got %+v", got)
	}
	if workoutIndex(got, other.WorkoutId) != -1 {
		t.Errorf("expected the workout under another template to be excluded, got %+v", got)
	}
	for _, w := range got {
		if w.Template.TemplateId != testutil.Id(1) || w.Template.TemplateName != "Push Day" {
			t.Errorf("expected only template {1 Push Day}, got %+v", w)
		}
	}

	otherList, err := repo.GetWorkoutsByUserIdAndTemplateId(ctx, 1, other.Template.TemplateId)
	if err != nil {
		t.Fatalf("GetWorkoutsByUserIdAndTemplateId returned error: %v", err)
	}
	if len(otherList) != 1 || otherList[0].WorkoutId != other.WorkoutId {
		t.Errorf("expected only workout %v under the new template, got %+v", other.WorkoutId, otherList)
	}
}

func TestWorkoutRepository_GetWorkoutsByTemplate_NewestFirst(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	completedAt := time.Date(2036, 1, 1, 10, 0, 0, 0, time.UTC)
	older := createWorkoutCompletedAt(t, repo, completedAt.Add(-time.Hour))
	first := createWorkoutCompletedAt(t, repo, completedAt)
	second := createWorkoutCompletedAt(t, repo, completedAt)

	got, err := repo.GetWorkoutsByUserIdAndTemplateId(ctx, 1, testutil.Id(1))
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
			t.Errorf("workout %v (%v) is listed after older workout %v (%v)",
				got[i].WorkoutId, got[i].CompletedAt, got[i-1].WorkoutId, got[i-1].CompletedAt)
		}
	}
}

func TestWorkoutRepository_GetWorkoutsByTemplate_NoWorkouts(t *testing.T) {
	ctx := t.Context()
	repo := workout.NewPostgresWorkoutRepository(testPool)

	var templateId uuid.UUID
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
		templateId uuid.UUID
	}{
		{"unknown template", 1, uuid.NewV7()},
		{"other user's template", 1, testutil.Id(2)},
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
