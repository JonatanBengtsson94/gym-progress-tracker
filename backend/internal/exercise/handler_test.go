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
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/identity"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/testutil"
)

type mockExerciseService struct {
	getGlobalFunc      func(ctx context.Context) ([]exercise.Exercise, error)
	getExercisesFunc   func(ctx context.Context, userId uint32) ([]exercise.Exercise, error)
	createExerciseFunc func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error)
	modifyExerciseFunc func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error)
}

func (m *mockExerciseService) GetGlobalExercises(ctx context.Context) ([]exercise.Exercise, error) {
	return m.getGlobalFunc(ctx)
}

func (m *mockExerciseService) GetExercises(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
	return m.getExercisesFunc(ctx, userId)
}

func (m *mockExerciseService) CreateExercise(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
	return m.createExerciseFunc(ctx, ex)
}

func (m *mockExerciseService) ModifyExercise(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
	return m.modifyExerciseFunc(ctx, ex)
}

func TestExerciseHandler_GetExercises_Success(t *testing.T) {
	serviceExercises := []exercise.Exercise{
		{ExerciseId: testutil.Id(1), UserId: 99, ExerciseName: "Test Exercise 1"},
		{ExerciseId: testutil.Id(2), UserId: 99, ExerciseName: "Test Exercise 2"},
	}
	expected := []exercise.ExerciseResponse{
		{ExerciseId: testutil.Id(1), ExerciseName: "Test Exercise 1"},
		{ExerciseId: testutil.Id(2), ExerciseName: "Test Exercise 2"},
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
				{ExerciseId: testutil.Id(1), UserId: 99, ExerciseName: "Test Exercise 1"},
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
		"exercise_id":   "00000000-0000-0000-0000-000000000001",
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

func TestExerciseHandler_GetGlobalExercises_NeedsNoSession(t *testing.T) {
	service := &mockExerciseService{
		getGlobalFunc: func(ctx context.Context) ([]exercise.Exercise, error) {
			return []exercise.Exercise{{ExerciseId: testutil.Id(1), ExerciseName: "Squat (Barbell)"}}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/global-exercises", nil)
	rec := httptest.NewRecorder()

	exercise.NewExerciseHandler(service).GetGlobalExercises(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", res.StatusCode)
	}

	var got exercise.ExercisesResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	want := []exercise.ExerciseResponse{{ExerciseId: testutil.Id(1), ExerciseName: "Squat (Barbell)"}}
	if !reflect.DeepEqual(got.Exercises, want) {
		t.Errorf("GetGlobalExercises() got = %v, want %v", got.Exercises, want)
	}
}

func TestExerciseHandler_GetGlobalExercises_EmptyListIsEmptyArray(t *testing.T) {
	service := &mockExerciseService{
		getGlobalFunc: func(ctx context.Context) ([]exercise.Exercise, error) {
			return nil, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/global-exercises", nil)
	rec := httptest.NewRecorder()

	exercise.NewExerciseHandler(service).GetGlobalExercises(rec, req)

	var got map[string]any
	if err := json.NewDecoder(rec.Result().Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}
	if exercises, ok := got["exercises"].([]any); !ok || len(exercises) != 0 {
		t.Errorf("Expected \"exercises\" to be an empty array, got %#v", got["exercises"])
	}
}

func TestExerciseHandler_GetGlobalExercises_ServiceError(t *testing.T) {
	service := &mockExerciseService{
		getGlobalFunc: func(ctx context.Context) ([]exercise.Exercise, error) {
			return nil, errors.New("db exploded")
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/global-exercises", nil)
	rec := httptest.NewRecorder()

	exercise.NewExerciseHandler(service).GetGlobalExercises(rec, req)

	if rec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", rec.Result().StatusCode)
	}
}

func TestExerciseHandler_CreateExercise_Success(t *testing.T) {
	created := exercise.Exercise{ExerciseId: testutil.Id(1), ExerciseName: "Lunge", UserId: 1}

	var got exercise.Exercise
	service := &mockExerciseService{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
			got = ex
			return created, true, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(`{"exercise_id":"00000000-0000-0000-0000-000000000001","exercise_name":"Lunge"}`))
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
	if got != created {
		t.Errorf("expected service called with %+v, got %+v", created, got)
	}

	var body exercise.ExerciseResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}
	want := exercise.ExerciseResponse{ExerciseId: testutil.Id(1), ExerciseName: "Lunge"}
	if body != want {
		t.Errorf("CreateExercise() response = %+v, want %+v", body, want)
	}
}

func TestExerciseHandler_CreateExercise_WithoutExerciseId(t *testing.T) {
	var got exercise.Exercise
	service := &mockExerciseService{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
			got = ex
			return exercise.Exercise{ExerciseId: testutil.Id(1), ExerciseName: ex.ExerciseName, UserId: ex.UserId}, true, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(`{"exercise_name":"Lunge"}`))
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateExercise(rec, req)

	if rec.Result().StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", rec.Result().StatusCode)
	}
	if got.ExerciseId != uuid.Nil() {
		t.Errorf("expected the service to receive no id, leaving it to pick one, got %v", got.ExerciseId)
	}
}

func TestExerciseHandler_CreateExercise_ExistingExercise(t *testing.T) {
	existing := exercise.Exercise{ExerciseId: testutil.Id(2), ExerciseName: "Lunge", UserId: 1}
	service := &mockExerciseService{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
			return existing, false, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(`{"exercise_id":"00000000-0000-0000-0000-000000000001","exercise_name":"lunge"}`))
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateExercise(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", res.StatusCode)
	}

	var body exercise.ExerciseResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}
	want := exercise.ExerciseResponse{ExerciseId: testutil.Id(2), ExerciseName: "Lunge"}
	if body != want {
		t.Errorf("CreateExercise() response = %+v, want the existing exercise %+v", body, want)
	}
}

func TestExerciseHandler_CreateExercise_InvalidExerciseId(t *testing.T) {
	for _, exerciseId := range []string{`"abc"`, `""`, `1`} {
		t.Run(exerciseId, func(t *testing.T) {
			service := &mockExerciseService{
				createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
					t.Fatal("CreateExercise should not be called for an invalid exercise_id")
					return exercise.Exercise{}, false, nil
				},
			}

			handler := exercise.NewExerciseHandler(service)

			body := `{"exercise_id":` + exerciseId + `,"exercise_name":"Lunge"}`
			req := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(body))
			req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
			rec := httptest.NewRecorder()

			handler.CreateExercise(rec, req)

			if rec.Result().StatusCode != http.StatusBadRequest {
				t.Fatalf("Expected status 400, got %d", rec.Result().StatusCode)
			}
		})
	}
}

func TestExerciseHandler_CreateExercise_ResponseContainsOnlyExpectedFields(t *testing.T) {
	created := exercise.Exercise{ExerciseId: testutil.Id(1), ExerciseName: "Lunge", UserId: 1}
	service := &mockExerciseService{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
			return created, true, nil
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
		"exercise_id":   "00000000-0000-0000-0000-000000000001",
		"exercise_name": "Lunge",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CreateExercise() response fields = %v, want exactly %v", got, want)
	}
}

func TestExerciseHandler_CreateExercise_Unauthorized(t *testing.T) {
	service := &mockExerciseService{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
			t.Fatal("CreateExercise should not be called without an authenticated user")
			return exercise.Exercise{}, false, nil
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
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
			t.Fatal("CreateExercise should not be called for a malformed request body")
			return exercise.Exercise{}, false, nil
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
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
			return exercise.Exercise{}, false, exercise.ErrExerciseNameRequired
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

func TestExerciseHandler_CreateExercise_IdTaken(t *testing.T) {
	service := &mockExerciseService{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
			return exercise.Exercise{}, false, exercise.ErrExerciseIdTaken
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
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
			return exercise.Exercise{}, false, errors.New("db exploded")
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
	modified := exercise.Exercise{ExerciseId: testutil.Id(1), ExerciseName: "Romanian Deadlift", UserId: 1}

	var got exercise.Exercise
	service := &mockExerciseService{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			got = ex
			return modified, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := newModifyExerciseRequest(1, testutil.Id(1).String(), `{"exercise_name":"Romanian Deadlift"}`, true)
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
	if got != modified {
		t.Errorf("expected service called with %+v, got %+v", modified, got)
	}

	var body exercise.ExerciseResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}
	want := exercise.ExerciseResponse{ExerciseId: testutil.Id(1), ExerciseName: "Romanian Deadlift"}
	if body != want {
		t.Errorf("ModifyExercise() response = %+v, want %+v", body, want)
	}
}

func TestExerciseHandler_ModifyExercise_ResponseContainsOnlyExpectedFields(t *testing.T) {
	modified := exercise.Exercise{ExerciseId: testutil.Id(1), ExerciseName: "Romanian Deadlift", UserId: 1}
	service := &mockExerciseService{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			return modified, nil
		},
	}

	handler := exercise.NewExerciseHandler(service)

	req := newModifyExerciseRequest(1, testutil.Id(1).String(), `{"exercise_name":"Romanian Deadlift"}`, true)
	rec := httptest.NewRecorder()

	handler.ModifyExercise(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	var got map[string]any
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	want := map[string]any{
		"exercise_id":   "00000000-0000-0000-0000-000000000001",
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

	req := newModifyExerciseRequest(1, testutil.Id(1).String(), `{"exercise_name":"Lunge"}`, false)
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

	req := newModifyExerciseRequest(1, testutil.Id(1).String(), `not-json`, true)
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

	req := newModifyExerciseRequest(1, testutil.Id(1).String(), `{"exercise_name":"   "}`, true)
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

	req := newModifyExerciseRequest(1, testutil.Id(1).String(), `{"exercise_name":"Lunge"}`, true)
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

	req := newModifyExerciseRequest(1, testutil.Id(1).String(), `{"exercise_name":"Bench Press Variant"}`, true)
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

	req := newModifyExerciseRequest(1, testutil.Id(999).String(), `{"exercise_name":"Lunge"}`, true)
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

	req := newModifyExerciseRequest(1, testutil.Id(1).String(), `{"exercise_name":"Lunge"}`, true)
	rec := httptest.NewRecorder()

	handler.ModifyExercise(rec, req)

	if rec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", rec.Result().StatusCode)
	}
}

func TestExerciseHandler_ModifyExercise_InvalidExerciseId(t *testing.T) {
	for _, exerciseId := range []string{"abc", "1", "00000000-0000-0000-0000-00000000001"} {
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

	req := newModifyExerciseRequest(1, testutil.Id(7).String(), `{"exercise_id":"00000000-0000-0000-0000-000000000009","exercise_name":"Lunge"}`, true)
	rec := httptest.NewRecorder()

	handler.ModifyExercise(rec, req)

	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rec.Result().StatusCode)
	}
	if got.ExerciseId != testutil.Id(7) {
		t.Errorf("expected service to receive exercise id %v from the path, got %v", testutil.Id(7), got.ExerciseId)
	}
}
