package exercise_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
)

type mockExerciseService struct {
	getExercisesFunc func() ([]exercise.Exercise, error)
}

func (m *mockExerciseService) GetExercises() ([]exercise.Exercise, error) {
	return m.getExercisesFunc()
}

func TestExerciseHandler_GetExercises_Success(t *testing.T) {
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

	service := &mockExerciseService{
		getExercisesFunc: func() ([]exercise.Exercise, error) {
			return expected, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	rec := httptest.NewRecorder()

	handler.GetExercises(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", res.StatusCode)
	}

	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", ct)
	}

	var got []exercise.Exercise
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if len(got) != len(expected) {
		t.Fatalf("Expected %d exercises, got %d", len(expected), len(got))
	}
}

func TestExerciseHandler_GetExercises_ServiceError(t *testing.T) {
	service := &mockExerciseService{
		getExercisesFunc: func() ([]exercise.Exercise, error) {
			return nil, errors.New("db exploded")
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	rec := httptest.NewRecorder()

	handler.GetExercises(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", res.StatusCode)
	}
}
