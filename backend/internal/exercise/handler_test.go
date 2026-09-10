package exercise_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
)

type mockExerciseService struct {
	getExercisesFunc   func(ctx context.Context, userId uint32) ([]exercise.Exercise, error)
	createExerciseFunc func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error)
}

func (m *mockExerciseService) GetExercises(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
	return m.getExercisesFunc(ctx, userId)
}

func (m *mockExerciseService) CreateExercise(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
	return m.createExerciseFunc(ctx, ex)
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

func TestExerciseHandler_CreateExercise_Success(t *testing.T) {
	created := exercise.Exercise{ExerciseId: 1, ExerciseName: "Lunge", UserId: 1}

	var got exercise.Exercise
	service := &mockExerciseService{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			got = ex
			return created, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(`{"exercise_name":"Lunge"}`))
	req = req.WithContext(auth.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateExercise(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", ct)
	}
	if got.ExerciseName != "Lunge" || got.UserId != 1 {
		t.Errorf("expected service called with {Lunge, UserId:1}, got %+v", got)
	}

	var body exercise.ExerciseResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}
	want := exercise.ExerciseResponse{ExerciseId: 1, ExerciseName: "Lunge"}
	if body != want {
		t.Errorf("CreateExercise() response = %+v, want %+v", body, want)
	}
}

func TestExerciseHandler_CreateExercise_Unauthorized(t *testing.T) {
	service := &mockExerciseService{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			t.Fatal("CreateExercise should not be called without an authenticated user")
			return exercise.Exercise{}, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(`{"exercise_name":"Lunge"}`))
	rec := httptest.NewRecorder()

	handler.CreateExercise(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestExerciseHandler_CreateExercise_MalformedBody(t *testing.T) {
	service := &mockExerciseService{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			t.Fatal("CreateExercise should not be called for a malformed request body")
			return exercise.Exercise{}, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(`not-json`))
	req = req.WithContext(auth.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateExercise(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestExerciseHandler_CreateExercise_EmptyName(t *testing.T) {
	service := &mockExerciseService{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			t.Fatal("CreateExercise should not be called for an empty exercise_name")
			return exercise.Exercise{}, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(`{"exercise_name":"   "}`))
	req = req.WithContext(auth.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateExercise(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestExerciseHandler_CreateExercise_AlreadyExists(t *testing.T) {
	service := &mockExerciseService{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			return exercise.Exercise{}, exercise.ErrExerciseAlreadyExists
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(`{"exercise_name":"Lunge"}`))
	req = req.WithContext(auth.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateExercise(rec, req)

	if rec.Result().StatusCode != http.StatusConflict {
		t.Fatalf("Expected status 409, got %d", rec.Result().StatusCode)
	}
}

func TestExerciseHandler_CreateExercise_ServiceError(t *testing.T) {
	service := &mockExerciseService{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			return exercise.Exercise{}, errors.New("db exploded")
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(`{"exercise_name":"Lunge"}`))
	req = req.WithContext(auth.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateExercise(rec, req)

	if rec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", rec.Result().StatusCode)
	}
}
