package workout_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/set"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/workout"
)

type mockWorkoutRepository struct {
	getWorkoutFunc    func(ctx context.Context, userId uint32, workoutId uint32) (workout.Workout, error)
	createWorkoutFunc func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error)
}

func (m *mockWorkoutRepository) GetWorkoutByUserIdAndWorkoutId(ctx context.Context, userId uint32, workoutId uint32) (workout.Workout, error) {
	return m.getWorkoutFunc(ctx, userId, workoutId)
}

func (m *mockWorkoutRepository) CreateWorkout(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
	return m.createWorkoutFunc(ctx, userId, w)
}

func TestWorkoutService_GetWorkout(t *testing.T) {
	ctx := t.Context()
	const wantUserId = 1
	const wantWorkoutId = 42
	expected := workout.Workout{
		WorkoutId:   wantWorkoutId,
		CompletedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Template:    template.Template{TemplateName: "Push Day"},
		Sets: []set.Set{
			{Reps: 8, WeightGrams: 60000},
		},
	}

	var gotUserId, gotWorkoutId uint32
	repo := &mockWorkoutRepository{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uint32) (workout.Workout, error) {
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
		t.Errorf("expected repo to receive workoutId %d, got %d", wantWorkoutId, gotWorkoutId)
	}

	if got.WorkoutId != expected.WorkoutId || got.Template.TemplateName != expected.Template.TemplateName || len(got.Sets) != len(expected.Sets) {
		t.Errorf("GetWorkout() = %+v, want %+v", got, expected)
	}
}

func validWorkout() workout.Workout {
	return workout.Workout{
		CompletedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Template:    template.Template{TemplateId: 1},
		Sets:        []set.Set{{Reps: 8, WeightGrams: 60000}},
	}
}

func TestWorkoutService_CreateWorkout(t *testing.T) {
	ctx := t.Context()
	const wantUserId = 1
	toCreate := validWorkout()

	var gotUserId uint32
	var gotWorkout workout.Workout
	repo := &mockWorkoutRepository{
		createWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
			gotUserId = userId
			gotWorkout = w
			w.WorkoutId = 7
			return w, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	got, err := service.CreateWorkout(ctx, wantUserId, toCreate)
	if err != nil {
		t.Fatalf("CreateWorkout returned error: %v", err)
	}

	if gotUserId != wantUserId {
		t.Errorf("expected repo to receive userId %d, got %d", wantUserId, gotUserId)
	}
	if !gotWorkout.CompletedAt.Equal(toCreate.CompletedAt) {
		t.Errorf("expected repo to receive CompletedAt %v, got %v", toCreate.CompletedAt, gotWorkout.CompletedAt)
	}
	if got.WorkoutId != 7 {
		t.Errorf("expected WorkoutId 7, got %d", got.WorkoutId)
	}
}

func TestWorkoutService_CreateWorkout_ValidationErrors(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name    string
		workout func() workout.Workout
		wantErr error
	}{
		{
			name: "no sets",
			workout: func() workout.Workout {
				w := validWorkout()
				w.Sets = nil
				return w
			},
			wantErr: workout.ErrSetsRequired,
		},
		{
			name: "set with zero reps",
			workout: func() workout.Workout {
				w := validWorkout()
				w.Sets = []set.Set{{Reps: 8}, {Reps: 0}}
				return w
			},
			wantErr: workout.ErrRepsRequired,
		},
		{
			name: "no template id and blank template name",
			workout: func() workout.Workout {
				w := validWorkout()
				w.Template = template.Template{TemplateName: "   "}
				return w
			},
			wantErr: template.ErrTemplateNameRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockWorkoutRepository{
				createWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
					t.Fatal("CreateWorkout should not reach the repository for an invalid workout")
					return workout.Workout{}, nil
				},
			}

			service := workout.NewWorkoutService(repo)

			_, err := service.CreateWorkout(ctx, 1, tt.workout())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestWorkoutService_CreateWorkout_TrimsGeneratedTemplateName(t *testing.T) {
	ctx := t.Context()
	toCreate := validWorkout()
	toCreate.Template = template.Template{TemplateName: "  Leg Day  "}

	var gotWorkout workout.Workout
	repo := &mockWorkoutRepository{
		createWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
			gotWorkout = w
			return w, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	if _, err := service.CreateWorkout(ctx, 1, toCreate); err != nil {
		t.Fatalf("CreateWorkout returned error: %v", err)
	}

	if gotWorkout.Template.TemplateName != "Leg Day" {
		t.Errorf("expected repo to receive TemplateName %q, got %q", "Leg Day", gotWorkout.Template.TemplateName)
	}
}

func TestWorkoutService_CreateWorkout_DefaultsCompletedAt(t *testing.T) {
	ctx := t.Context()
	toCreate := validWorkout()
	toCreate.CompletedAt = time.Time{}

	var gotWorkout workout.Workout
	repo := &mockWorkoutRepository{
		createWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
			gotWorkout = w
			return w, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	before := time.Now().UTC()
	if _, err := service.CreateWorkout(ctx, 1, toCreate); err != nil {
		t.Fatalf("CreateWorkout returned error: %v", err)
	}
	after := time.Now().UTC()

	if gotWorkout.CompletedAt.Before(before) || gotWorkout.CompletedAt.After(after) {
		t.Errorf("expected CompletedAt to default to now, got %v", gotWorkout.CompletedAt)
	}
}

func TestWorkoutService_GetWorkout_RepoError(t *testing.T) {
	ctx := t.Context()
	wantErr := workout.ErrWorkoutNotFound

	repo := &mockWorkoutRepository{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uint32) (workout.Workout, error) {
			return workout.Workout{}, wantErr
		},
	}

	service := workout.NewWorkoutService(repo)

	_, err := service.GetWorkout(ctx, 1, 42)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected error to wrap %v, got %v", wantErr, err)
	}
}
