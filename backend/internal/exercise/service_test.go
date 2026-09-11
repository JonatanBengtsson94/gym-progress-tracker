package exercise_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
)

type mockExerciseRepository struct {
	getExerciseFunc    func(ctx context.Context, userId uint32) ([]exercise.Exercise, error)
	createExerciseFunc func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error)
	modifyExerciseFunc func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error)
}

func (m *mockExerciseRepository) GetExercisesByUserId(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
	return m.getExerciseFunc(ctx, userId)
}

func (m *mockExerciseRepository) ModifyExercise(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
	return m.modifyExerciseFunc(ctx, ex)
}

func (m *mockExerciseRepository) CreateExercise(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
	return m.createExerciseFunc(ctx, ex)
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

func TestExerciseService_CreateExercise(t *testing.T) {
	ctx := t.Context()
	input := exercise.Exercise{ExerciseName: "Lunge", UserId: 42}
	created := exercise.Exercise{ExerciseId: 1, ExerciseName: "Lunge", UserId: 42}

	var got exercise.Exercise
	repo := &mockExerciseRepository{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			got = ex
			return created, nil
		},
	}

	service := exercise.NewExerciseService(repo)

	result, err := service.CreateExercise(ctx, input)
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}

	if got != input {
		t.Errorf("expected repo to receive %+v, got %+v", input, got)
	}

	if result != created {
		t.Errorf("CreateExercise() = %+v, want %+v", result, created)
	}
}

func TestExerciseService_CreateExercise_NameRequired(t *testing.T) {
	ctx := t.Context()

	repo := &mockExerciseRepository{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			t.Fatal("CreateExercise should not be called for an empty exercise_name")
			return exercise.Exercise{}, nil
		},
	}

	service := exercise.NewExerciseService(repo)

	_, err := service.CreateExercise(ctx, exercise.Exercise{ExerciseName: "   ", UserId: 42})
	if !errors.Is(err, exercise.ErrExerciseNameRequired) {
		t.Fatalf("Expected ErrExerciseNameRequired, got %v", err)
	}
}

func TestExerciseService_CreateExercise_RepoError(t *testing.T) {
	ctx := t.Context()
	wantErr := exercise.ErrExerciseAlreadyExists

	repo := &mockExerciseRepository{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			return exercise.Exercise{}, wantErr
		},
	}

	service := exercise.NewExerciseService(repo)

	_, err := service.CreateExercise(ctx, exercise.Exercise{ExerciseName: "Lunge", UserId: 42})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected error to wrap %v, got %v", wantErr, err)
	}
}

func TestExerciseService_ModifyExercise(t *testing.T) {
	ctx := t.Context()
	input := exercise.Exercise{ExerciseId: 1, ExerciseName: "Romanian Deadlift", UserId: 42}
	modified := exercise.Exercise{ExerciseId: 1, ExerciseName: "Romanian Deadlift", UserId: 42}

	var got exercise.Exercise
	repo := &mockExerciseRepository{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			got = ex
			return modified, nil
		},
	}

	service := exercise.NewExerciseService(repo)

	result, err := service.ModifyExercise(ctx, input)
	if err != nil {
		t.Fatalf("ModifyExercise returned error: %v", err)
	}

	if got != input {
		t.Errorf("expected repo to receive %+v, got %+v", input, got)
	}

	if result != modified {
		t.Errorf("ModifyExercise() = %+v, want %+v", result, modified)
	}
}

func TestExerciseService_ModifyExercise_NameRequired(t *testing.T) {
	ctx := t.Context()

	repo := &mockExerciseRepository{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			t.Fatal("ModifyExercise should not be called for an empty exercise_name")
			return exercise.Exercise{}, nil
		},
	}

	service := exercise.NewExerciseService(repo)

	_, err := service.ModifyExercise(ctx, exercise.Exercise{ExerciseId: 1, ExerciseName: "   ", UserId: 42})
	if !errors.Is(err, exercise.ErrExerciseNameRequired) {
		t.Fatalf("Expected ErrExerciseNameRequired, got %v", err)
	}
}

func TestExerciseService_ModifyExercise_RepoError(t *testing.T) {
	ctx := t.Context()
	wantErr := exercise.ErrExerciseNotFound

	repo := &mockExerciseRepository{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			return exercise.Exercise{}, wantErr
		},
	}

	service := exercise.NewExerciseService(repo)

	_, err := service.ModifyExercise(ctx, exercise.Exercise{ExerciseId: 1, ExerciseName: "Lunge", UserId: 42})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected error to wrap %v, got %v", wantErr, err)
	}
}
