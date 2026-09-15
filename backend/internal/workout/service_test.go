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
	getWorkoutFunc func(ctx context.Context, userId uint32, workoutId uint32) (workout.Workout, error)
}

func (m *mockWorkoutRepository) GetWorkoutByUserIdAndWorkoutId(ctx context.Context, userId uint32, workoutId uint32) (workout.Workout, error) {
	return m.getWorkoutFunc(ctx, userId, workoutId)
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
