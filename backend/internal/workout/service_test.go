package workout_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/set"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/testutil"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/workout"
)

type mockWorkoutRepository struct {
	getWorkoutFunc    func(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error)
	getWorkoutsFunc   func(ctx context.Context, userId uint32) ([]workout.Workout, error)
	getByTemplateFunc func(ctx context.Context, userId uint32, templateId uuid.UUID) ([]workout.Workout, error)
	putWorkoutFunc    func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, bool, error)
}

func (m *mockWorkoutRepository) GetWorkoutByUserIdAndWorkoutId(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error) {
	return m.getWorkoutFunc(ctx, userId, workoutId)
}

func (m *mockWorkoutRepository) GetWorkoutsByUserId(ctx context.Context, userId uint32) ([]workout.Workout, error) {
	return m.getWorkoutsFunc(ctx, userId)
}

func (m *mockWorkoutRepository) GetWorkoutsByUserIdAndTemplateId(ctx context.Context, userId uint32, templateId uuid.UUID) ([]workout.Workout, error) {
	return m.getByTemplateFunc(ctx, userId, templateId)
}

func (m *mockWorkoutRepository) PutWorkout(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, bool, error) {
	return m.putWorkoutFunc(ctx, userId, w)
}

func TestWorkoutService_GetWorkout(t *testing.T) {
	ctx := t.Context()
	const wantUserId = 1
	wantWorkoutId := testutil.Id(42)
	expected := workout.Workout{
		WorkoutId:   wantWorkoutId,
		CompletedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Template:    template.Template{TemplateName: "Push Day"},
		Sets: []set.Set{
			{Reps: 8, WeightGrams: 60000},
		},
	}

	var gotUserId uint32
	var gotWorkoutId uuid.UUID
	repo := &mockWorkoutRepository{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error) {
			gotUserId = userId
			gotWorkoutId = workoutId
			return expected, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	got, err := service.GetWorkout(ctx, wantUserId, wantWorkoutId)
	if err != nil {
		t.Fatalf("GetWorkout returned error: %v", err)
	}

	if gotUserId != wantUserId {
		t.Errorf("expected repo to receive userId %d, got %d", wantUserId, gotUserId)
	}
	if gotWorkoutId != wantWorkoutId {
		t.Errorf("expected repo to receive workoutId %v, got %v", wantWorkoutId, gotWorkoutId)
	}

	if got.WorkoutId != expected.WorkoutId || got.Template.TemplateName != expected.Template.TemplateName || len(got.Sets) != len(expected.Sets) {
		t.Errorf("GetWorkout() = %+v, want %+v", got, expected)
	}
}

func validWorkout() workout.Workout {
	return workout.Workout{
		StartedAt:   time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
		CompletedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Template:    template.Template{TemplateId: testutil.Id(1)},
		Sets:        []set.Set{{Reps: 8, WeightGrams: 60000}},
	}
}

func TestWorkoutService_GetWorkout_RepoError(t *testing.T) {
	ctx := t.Context()
	wantErr := workout.ErrWorkoutNotFound

	repo := &mockWorkoutRepository{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error) {
			return workout.Workout{}, wantErr
		},
	}

	service := workout.NewWorkoutService(repo)

	_, err := service.GetWorkout(ctx, 1, testutil.Id(42))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected error to wrap %v, got %v", wantErr, err)
	}
}

func TestWorkoutService_PutWorkout(t *testing.T) {
	ctx := t.Context()
	const wantUserId = 1
	toPut := validWorkout()
	toPut.WorkoutId = testutil.Id(42)

	for _, created := range []bool{true, false} {
		var gotUserId uint32
		var gotWorkout workout.Workout
		repo := &mockWorkoutRepository{
			putWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, bool, error) {
				gotUserId = userId
				gotWorkout = w
				return w, created, nil
			},
		}

		service := workout.NewWorkoutService(repo)

		got, gotCreated, err := service.PutWorkout(ctx, wantUserId, toPut)
		if err != nil {
			t.Fatalf("PutWorkout returned error: %v", err)
		}

		if gotUserId != wantUserId {
			t.Errorf("expected repo to receive userId %d, got %d", wantUserId, gotUserId)
		}
		if gotWorkout.WorkoutId != toPut.WorkoutId || gotWorkout.Template != toPut.Template ||
			!gotWorkout.StartedAt.Equal(toPut.StartedAt) || !gotWorkout.CompletedAt.Equal(toPut.CompletedAt) || len(gotWorkout.Sets) != 1 {
			t.Errorf("expected repo to receive %+v unchanged, got %+v", toPut, gotWorkout)
		}
		if got.WorkoutId != toPut.WorkoutId || gotCreated != created {
			t.Errorf("PutWorkout() = %v, created %v, want %v, created %v", got.WorkoutId, gotCreated, toPut.WorkoutId, created)
		}
	}
}

func TestWorkoutService_PutWorkout_TrimsTemplateName(t *testing.T) {
	ctx := t.Context()

	var gotWorkout workout.Workout
	repo := &mockWorkoutRepository{
		putWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, bool, error) {
			gotWorkout = w
			return w, true, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	toPut := validWorkout()
	toPut.WorkoutId = testutil.Id(42)
	toPut.Template = template.Template{TemplateName: "  Leg Day  "}

	if _, _, err := service.PutWorkout(ctx, 1, toPut); err != nil {
		t.Fatalf("PutWorkout returned error: %v", err)
	}

	if gotWorkout.Template.TemplateName != "Leg Day" {
		t.Errorf("expected repo to receive TemplateName %q, got %q", "Leg Day", gotWorkout.Template.TemplateName)
	}
}

func TestWorkoutService_PutWorkout_ValidationErrors(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name    string
		modify  func(*workout.Workout)
		wantErr error
	}{
		{"no sets", func(w *workout.Workout) { w.Sets = nil }, workout.ErrSetsRequired},
		{"set with zero reps", func(w *workout.Workout) { w.Sets = []set.Set{{Reps: 8}, {Reps: 0}} }, workout.ErrRepsRequired},
		{"no started at", func(w *workout.Workout) { w.StartedAt = time.Time{} }, workout.ErrStartedAtRequired},
		{"no completed at", func(w *workout.Workout) { w.CompletedAt = time.Time{} }, workout.ErrCompletedAtRequired},
		{"started after completed", func(w *workout.Workout) { w.StartedAt = w.CompletedAt.Add(time.Second) }, workout.ErrStartedAfterCompleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockWorkoutRepository{
				putWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, bool, error) {
					t.Fatal("PutWorkout should not reach the repository for an invalid workout")
					return workout.Workout{}, false, nil
				},
			}

			service := workout.NewWorkoutService(repo)

			toPut := validWorkout()
			toPut.WorkoutId = testutil.Id(42)
			tt.modify(&toPut)

			_, _, err := service.PutWorkout(ctx, 1, toPut)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestWorkoutService_PutWorkout_RepositoryError(t *testing.T) {
	ctx := t.Context()
	wantErr := workout.ErrWorkoutIdTaken
	repo := &mockWorkoutRepository{
		putWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, bool, error) {
			return workout.Workout{}, false, wantErr
		},
	}

	service := workout.NewWorkoutService(repo)

	toPut := validWorkout()
	toPut.WorkoutId = testutil.Id(42)

	_, _, err := service.PutWorkout(ctx, 1, toPut)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected %v, got %v", wantErr, err)
	}
}

func TestWorkoutService_GetWorkouts(t *testing.T) {
	ctx := t.Context()
	const wantUserId = 1
	expected := []workout.Workout{
		{WorkoutId: testutil.Id(2), CompletedAt: time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC), Template: template.Template{TemplateId: testutil.Id(1), TemplateName: "Push Day"}},
		{WorkoutId: testutil.Id(1), CompletedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), Template: template.Template{TemplateId: testutil.Id(1), TemplateName: "Push Day"}},
	}

	var gotUserId uint32
	repo := &mockWorkoutRepository{
		getWorkoutsFunc: func(ctx context.Context, userId uint32) ([]workout.Workout, error) {
			gotUserId = userId
			return expected, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	got, err := service.GetWorkouts(ctx, wantUserId)
	if err != nil {
		t.Fatalf("GetWorkouts returned error: %v", err)
	}

	if gotUserId != wantUserId {
		t.Errorf("expected repo to receive userId %d, got %d", wantUserId, gotUserId)
	}
	if len(got) != len(expected) || got[0].WorkoutId != testutil.Id(2) || got[1].WorkoutId != testutil.Id(1) {
		t.Errorf("GetWorkouts() = %+v, want %+v (in the repository's order)", got, expected)
	}
}

func TestWorkoutService_GetWorkouts_RepositoryError(t *testing.T) {
	ctx := t.Context()
	wantErr := errors.New("db exploded")
	repo := &mockWorkoutRepository{
		getWorkoutsFunc: func(ctx context.Context, userId uint32) ([]workout.Workout, error) {
			return nil, wantErr
		},
	}

	service := workout.NewWorkoutService(repo)

	_, err := service.GetWorkouts(ctx, 1)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected %v, got %v", wantErr, err)
	}
}

func TestWorkoutService_GetWorkoutsByTemplate(t *testing.T) {
	ctx := t.Context()
	const wantUserId = 1
	wantTemplateId := testutil.Id(3)
	expected := []workout.Workout{
		{WorkoutId: testutil.Id(2), CompletedAt: time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC), Template: template.Template{TemplateId: wantTemplateId, TemplateName: "Push Day"}},
		{WorkoutId: testutil.Id(1), CompletedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), Template: template.Template{TemplateId: wantTemplateId, TemplateName: "Push Day"}},
	}

	var gotUserId uint32
	var gotTemplateId uuid.UUID
	repo := &mockWorkoutRepository{
		getByTemplateFunc: func(ctx context.Context, userId uint32, templateId uuid.UUID) ([]workout.Workout, error) {
			gotUserId, gotTemplateId = userId, templateId
			return expected, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	got, err := service.GetWorkoutsByTemplate(ctx, wantUserId, wantTemplateId)
	if err != nil {
		t.Fatalf("GetWorkoutsByTemplate returned error: %v", err)
	}

	if gotUserId != wantUserId || gotTemplateId != wantTemplateId {
		t.Errorf("expected repo to receive userId %d and templateId %v, got %d and %v", wantUserId, wantTemplateId, gotUserId, gotTemplateId)
	}
	if len(got) != len(expected) || got[0].WorkoutId != testutil.Id(2) || got[1].WorkoutId != testutil.Id(1) {
		t.Errorf("GetWorkoutsByTemplate() = %+v, want %+v (in the repository's order)", got, expected)
	}
}

func TestWorkoutService_GetWorkoutsByTemplate_RepositoryErrors(t *testing.T) {
	for _, wantErr := range []error{workout.ErrTemplateNotFound, errors.New("db exploded")} {
		t.Run(wantErr.Error(), func(t *testing.T) {
			repo := &mockWorkoutRepository{
				getByTemplateFunc: func(ctx context.Context, userId uint32, templateId uuid.UUID) ([]workout.Workout, error) {
					return nil, wantErr
				},
			}

			service := workout.NewWorkoutService(repo)

			_, err := service.GetWorkoutsByTemplate(t.Context(), 1, testutil.Id(3))
			if !errors.Is(err, wantErr) {
				t.Fatalf("Expected %v, got %v", wantErr, err)
			}
		})
	}
}
