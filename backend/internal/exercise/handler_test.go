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

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/identity"
)

type mockExerciseService struct {
	getExercisesFunc   func(ctx context.Context, userId uint32) ([]exercise.Exercise, error)
	createExerciseFunc func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error)
	modifyExerciseFunc func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error)
}

func (m *mockExerciseService) GetExercises(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
	return m.getExercisesFunc(ctx, userId)
}

func (m *mockExerciseService) CreateExercise(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
	return m.createExerciseFunc(ctx, ex)
}

func (m *mockExerciseService) ModifyExercise(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
	return m.modifyExerciseFunc(ctx, ex)
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
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
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

	var got exercise.ExercisesResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if !reflect.DeepEqual(got.Exercises, expected) {
		t.Errorf("GetExercises() got = %v, want %v", got.Exercises, expected)
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
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.GetExercises(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	var got map[string][]map[string]any
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("Expected the envelope to contain only \"exercises\", got %v", got)
	}
	exercises := got["exercises"]
	if len(exercises) != 1 {
		t.Fatalf("Expected 1 exercise, got %d", len(exercises))
	}

	want := map[string]any{
		"exercise_id":   float64(1),
		"exercise_name": "Test Exercise 1",
	}
	if !reflect.DeepEqual(exercises[0], want) {
		t.Errorf("GetExercises() response fields = %v, want exactly %v", exercises[0], want)
	}
}

func TestExerciseHandler_GetExercises_EmptyListIsEmptyArray(t *testing.T) {
	for name, serviceResult := range map[string][]exercise.Exercise{
		"nil":   nil,
		"empty": {},
	} {
		t.Run(name, func(t *testing.T) {
			service := &mockExerciseService{
				getExercisesFunc: func(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
					return serviceResult, nil
				},
			}

			handler := exercise.NewExerciseHandler(service)

			req := httptest.NewRequest(http.MethodGet, "/exercises", nil)
			req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
			rec := httptest.NewRecorder()

			handler.GetExercises(rec, req)

			var got map[string]any
			if err := json.NewDecoder(rec.Result().Body).Decode(&got); err != nil {
				t.Fatalf("Failed to decode response body: %v", err)
			}

			exercises, ok := got["exercises"].([]any)
			if !ok || len(exercises) != 0 {
				t.Errorf("Expected \"exercises\" to be an empty array, got %#v", got["exercises"])
			}
		})
	}
}

func TestExerciseHandler_GetExercises_Unauthorized(t *testing.T) {
	service := &mockExerciseService{
		getExercisesFunc: func(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
			t.Fatal("GetExercises should not be called without an authenticated user")
			return nil, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	rec := httptest.NewRecorder()

	handler.GetExercises(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", rec.Result().StatusCode)
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
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
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
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
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

func TestExerciseHandler_CreateExercise_ResponseContainsOnlyExpectedFields(t *testing.T) {
	created := exercise.Exercise{ExerciseId: 1, ExerciseName: "Lunge", UserId: 1}
	service := &mockExerciseService{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			return created, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(`{"exercise_name":"Lunge"}`))
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateExercise(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	var got map[string]any
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	want := map[string]any{
		"exercise_id":   float64(1),
		"exercise_name": "Lunge",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CreateExercise() response fields = %v, want exactly %v", got, want)
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
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateExercise(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestExerciseHandler_CreateExercise_NameRequired(t *testing.T) {
	service := &mockExerciseService{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			return exercise.Exercise{}, exercise.ErrExerciseNameRequired
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(`{"exercise_name":"   "}`))
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
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
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
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
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateExercise(rec, req)

	if rec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", rec.Result().StatusCode)
	}
}

func newModifyExerciseRequest(userId uint32, exerciseId string, body string, authenticated bool) *http.Request {
	req := httptest.NewRequest(http.MethodPut, "/exercises/"+exerciseId, strings.NewReader(body))
	req.SetPathValue("exerciseId", exerciseId)
	if authenticated {
		req = req.WithContext(identity.ContextWithUserId(req.Context(), userId))
	}
	return req
}

func TestExerciseHandler_ModifyExercise_Success(t *testing.T) {
	modified := exercise.Exercise{ExerciseId: 1, ExerciseName: "Romanian Deadlift", UserId: 1}

	var got exercise.Exercise
	service := &mockExerciseService{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			got = ex
			return modified, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := newModifyExerciseRequest(1, "1", `{"exercise_name":"Romanian Deadlift"}`, true)
	rec := httptest.NewRecorder()

	handler.ModifyExercise(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", ct)
	}
	if got.ExerciseId != 1 || got.ExerciseName != "Romanian Deadlift" || got.UserId != 1 {
		t.Errorf("expected service called with {ExerciseId:1, Romanian Deadlift, UserId:1}, got %+v", got)
	}

	var body exercise.ExerciseResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}
	want := exercise.ExerciseResponse{ExerciseId: 1, ExerciseName: "Romanian Deadlift"}
	if body != want {
		t.Errorf("ModifyExercise() response = %+v, want %+v", body, want)
	}
}

func TestExerciseHandler_ModifyExercise_ResponseContainsOnlyExpectedFields(t *testing.T) {
	modified := exercise.Exercise{ExerciseId: 1, ExerciseName: "Romanian Deadlift", UserId: 1}
	service := &mockExerciseService{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			return modified, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := newModifyExerciseRequest(1, "1", `{"exercise_name":"Romanian Deadlift"}`, true)
	rec := httptest.NewRecorder()

	handler.ModifyExercise(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	var got map[string]any
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	want := map[string]any{
		"exercise_id":   float64(1),
		"exercise_name": "Romanian Deadlift",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ModifyExercise() response fields = %v, want exactly %v", got, want)
	}
}

func TestExerciseHandler_ModifyExercise_Unauthorized(t *testing.T) {
	service := &mockExerciseService{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			t.Fatal("ModifyExercise should not be called without an authenticated user")
			return exercise.Exercise{}, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := newModifyExerciseRequest(1, "1", `{"exercise_name":"Lunge"}`, false)
	rec := httptest.NewRecorder()

	handler.ModifyExercise(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestExerciseHandler_ModifyExercise_MalformedBody(t *testing.T) {
	service := &mockExerciseService{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			t.Fatal("ModifyExercise should not be called for a malformed request body")
			return exercise.Exercise{}, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := newModifyExerciseRequest(1, "1", `not-json`, true)
	rec := httptest.NewRecorder()

	handler.ModifyExercise(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestExerciseHandler_ModifyExercise_NameRequired(t *testing.T) {
	service := &mockExerciseService{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			return exercise.Exercise{}, exercise.ErrExerciseNameRequired
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := newModifyExerciseRequest(1, "1", `{"exercise_name":"   "}`, true)
	rec := httptest.NewRecorder()

	handler.ModifyExercise(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestExerciseHandler_ModifyExercise_AlreadyExists(t *testing.T) {
	service := &mockExerciseService{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			return exercise.Exercise{}, exercise.ErrExerciseAlreadyExists
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := newModifyExerciseRequest(1, "1", `{"exercise_name":"Lunge"}`, true)
	rec := httptest.NewRecorder()

	handler.ModifyExercise(rec, req)

	if rec.Result().StatusCode != http.StatusConflict {
		t.Fatalf("Expected status 409, got %d", rec.Result().StatusCode)
	}
}

func TestExerciseHandler_ModifyExercise_GlobalExercise(t *testing.T) {
	service := &mockExerciseService{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			return exercise.Exercise{}, exercise.ErrCannotModifyGlobalExercise
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := newModifyExerciseRequest(1, "1", `{"exercise_name":"Bench Press Variant"}`, true)
	rec := httptest.NewRecorder()

	handler.ModifyExercise(rec, req)

	if rec.Result().StatusCode != http.StatusForbidden {
		t.Fatalf("Expected status 403, got %d", rec.Result().StatusCode)
	}
}

func TestExerciseHandler_ModifyExercise_NotFound(t *testing.T) {
	service := &mockExerciseService{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			return exercise.Exercise{}, exercise.ErrExerciseNotFound
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := newModifyExerciseRequest(1, "999", `{"exercise_name":"Lunge"}`, true)
	rec := httptest.NewRecorder()

	handler.ModifyExercise(rec, req)

	if rec.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", rec.Result().StatusCode)
	}
}

func TestExerciseHandler_ModifyExercise_ServiceError(t *testing.T) {
	service := &mockExerciseService{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			return exercise.Exercise{}, errors.New("db exploded")
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := newModifyExerciseRequest(1, "1", `{"exercise_name":"Lunge"}`, true)
	rec := httptest.NewRecorder()

	handler.ModifyExercise(rec, req)

	if rec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", rec.Result().StatusCode)
	}
}

func TestExerciseHandler_ModifyExercise_InvalidExerciseId(t *testing.T) {
	for _, exerciseId := range []string{"abc", "-1", "4294967296"} {
		t.Run(exerciseId, func(t *testing.T) {
			service := &mockExerciseService{
				modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
					t.Fatal("ModifyExercise should not be called for an invalid exercise id")
					return exercise.Exercise{}, nil
				},
			}

			handler := exercise.NewExerciseHandler(service)

			req := newModifyExerciseRequest(1, exerciseId, `{"exercise_name":"Lunge"}`, true)
			rec := httptest.NewRecorder()

			handler.ModifyExercise(rec, req)

			if rec.Result().StatusCode != http.StatusBadRequest {
				t.Fatalf("Expected status 400, got %d", rec.Result().StatusCode)
			}
		})
	}
}

func TestExerciseHandler_ModifyExercise_PathIdWinsOverBodyId(t *testing.T) {
	var got exercise.Exercise
	service := &mockExerciseService{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			got = ex
			return ex, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := newModifyExerciseRequest(1, "7", `{"exercise_id":9,"exercise_name":"Lunge"}`, true)
	rec := httptest.NewRecorder()

	handler.ModifyExercise(rec, req)

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Result().StatusCode)
	}
	if got.ExerciseId != 7 {
		t.Errorf("expected service to receive exercise id 7 from the path, got %d", got.ExerciseId)
	}
}
