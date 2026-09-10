package exercise_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
)

type mockExerciseService struct {
	getExercisesFunc func(ctx context.Context, userId uint32) ([]exercise.Exercise, error)
}

func (m *mockExerciseService) GetExercises(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
	return m.getExercisesFunc(ctx, userId)
}

func TestExerciseHandler_GetExercises_Success(t *testing.T) {
	serviceExercises := []exercise.Exercise{
		{ExerciseId: 1, UserId: 99, ExerciseName: "Test Exercise 1"},
		{ExerciseId: 2, UserId: 99, ExerciseName: "Test Exercise 2"},
	}
	expected := []exercise.ExerciseResponse{
		{ExerciseId: 1, ExerciseName: "Test Exercise 1"},
		{ExerciseId: 2, ExerciseName: "Test Exercise 2"},
	}

	service := &mockExerciseService{
		getExercisesFunc: func(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
			return serviceExercises, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	req = req.WithContext(auth.ContextWithUserId(req.Context(), 1))
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

	var got []exercise.ExerciseResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("GetExercises() got = %v, want %v", got, expected)
	}
}

func TestExerciseHandler_GetExercises_ResponseContainsOnlyExpectedFields(t *testing.T) {
	service := &mockExerciseService{
		getExercisesFunc: func(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
			return []exercise.Exercise{
				{ExerciseId: 1, UserId: 99, ExerciseName: "Test Exercise 1"},
			}, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	req = req.WithContext(auth.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.GetExercises(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	var got []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("Expected 1 exercise, got %d", len(got))
	}

	want := map[string]any{
		"exercise_id":   float64(1),
		"exercise_name": "Test Exercise 1",
	}
	if !reflect.DeepEqual(got[0], want) {
		t.Errorf("GetExercises() response fields = %v, want exactly %v", got[0], want)
	}
}

func TestExerciseHandler_GetExercises_ServiceError(t *testing.T) {
	service := &mockExerciseService{
		getExercisesFunc: func(context.Context, uint32) ([]exercise.Exercise, error) {
			return nil, errors.New("db exploded")
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	req = req.WithContext(auth.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.GetExercises(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", res.StatusCode)
	}
}
