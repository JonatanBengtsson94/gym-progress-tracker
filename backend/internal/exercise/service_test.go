package exercise_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
)

type mockExerciseRepository struct {
	getExerciseFunc func(ctx context.Context, userId uint32) ([]exercise.Exercise, error)
}

func (m *mockExerciseRepository) GetExercises(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
	return m.getExerciseFunc(ctx, userId)
}

func TestExerciseService_GetExercises(t *testing.T) {
	ctx := t.Context()
	const wantUserId = 42
	expected := []exercise.Exercise{
		{
			ExerciseId:   1,
			ExerciseName: "Test Exercise 1",
		},
		{
			ExerciseId:   2,
			ExerciseName: "Test Exercise 2",
		},
	}

	var gotUserId uint32
	repo := &mockExerciseRepository{
		getExerciseFunc: func(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
			gotUserId = userId
			return expected, nil
		},
	}

	service := exercise.NewExerciseService(repo)

	got, err := service.GetExercises(ctx, wantUserId)
	if err != nil {
		t.Fatalf("GetExercises returned error: %v", err)
	}

	if gotUserId != wantUserId {
		t.Errorf("expected repo to receive userId %d, got %d", wantUserId, gotUserId)
	}

	if len(got) != len(expected) {
		t.Fatalf("Expected %d exercises, got %d", len(expected), len(got))
	}

	for i, ex := range got {
		if ex != expected[i] {
			t.Errorf("exercise %d: got %+v, want %+v", i, ex, expected[i])
		}
	}
}

func TestExerciseService_GetExercises_RepoError(t *testing.T) {
	ctx := t.Context()
	wantErr := errors.New("db exploded")

	repo := &mockExerciseRepository{
		getExerciseFunc: func(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
			return nil, wantErr
		},
	}

	service := exercise.NewExerciseService(repo)

	_, err := service.GetExercises(ctx, 0)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected error to wrap %v, got %v", wantErr, err)
	}
}
