package exercise_test

import (
	"errors"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
)

type mockExerciseRepository struct {
	getExerciseFunc func() ([]exercise.Exercise, error)
}

func (m *mockExerciseRepository) GetExercises() ([]exercise.Exercise, error) {
	return m.getExerciseFunc()
}

func TestExerciseService_GetExercises(t *testing.T) {
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

	repo := &mockExerciseRepository{
		getExerciseFunc: func() ([]exercise.Exercise, error) {
			return expected, nil
		},
	}

	service := exercise.NewExerciseService(repo)

	got, err := service.GetExercises()
	if err != nil {
		t.Fatalf("GetExercises returned error: %v", err)
	}
	if len(got) != len(expected) {
		t.Fatalf("Expected %d exercises, got %d", len(expected), len(got))
	}
}

func TestExerciseService_GetExercises_RepoError(t *testing.T) {
	repo := &mockExerciseRepository{
		getExerciseFunc: func() ([]exercise.Exercise, error) {
			return nil, errors.New("db exploded")
		},
	}

	service := exercise.NewExerciseService(repo)

	_, err := service.GetExercises()
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}
