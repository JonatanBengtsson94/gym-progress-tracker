package workout_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/set"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/workout"
)

type mockWorkoutService struct {
	getWorkoutFunc func(ctx context.Context, userId uint32, workoutId uint32) (workout.Workout, error)
}

func (m *mockWorkoutService) GetWorkout(ctx context.Context, userId uint32, workoutId uint32) (workout.Workout, error) {
	return m.getWorkoutFunc(ctx, userId, workoutId)
}

func newGetWorkoutRequest(userId uint32, workoutId string, authenticated bool) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/workouts/"+workoutId, nil)
	req.SetPathValue("workoutId", workoutId)
	if authenticated {
		req = req.WithContext(auth.ContextWithUserId(req.Context(), userId))
	}
	return req
}

func TestWorkoutHandler_GetWorkout_Success(t *testing.T) {
	serviceWorkout := workout.Workout{
		WorkoutId:   1,
		CompletedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Template:    template.Template{TemplateId: 1, TemplateName: "Push Day"},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: 1, ExerciseName: "Bench Press"}, Reps: 8, WeightGrams: 60000},
		},
	}

	var gotUserId, gotWorkoutId uint32
	service := &mockWorkoutService{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uint32) (workout.Workout, error) {
			gotUserId = userId
			gotWorkoutId = workoutId
			return serviceWorkout, nil
		},
	}

	handler := workout.NewWorkoutHandler(service)

	req := newGetWorkoutRequest(1, "1", true)
	rec := httptest.NewRecorder()

	handler.GetWorkout(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", ct)
	}
	if gotUserId != 1 {
		t.Errorf("expected service to receive userId 1, got %d", gotUserId)
	}
	if gotWorkoutId != 1 {
		t.Errorf("expected service to receive workoutId 1, got %d", gotWorkoutId)
	}

	var got workout.WorkoutResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if got.WorkoutId != 1 || got.TemplateName != "Push Day" || !got.CompletedAt.Equal(serviceWorkout.CompletedAt) {
		t.Errorf("GetWorkout() = %+v", got)
	}
	if len(got.Sets) != 1 {
		t.Fatalf("expected 1 set, got %d: %+v", len(got.Sets), got.Sets)
	}
	if got.Sets[0].ExerciseId != 1 || got.Sets[0].ExerciseName != "Bench Press" || got.Sets[0].Reps != 8 || got.Sets[0].WeightGrams != 60000 {
		t.Errorf("GetWorkout() set = %+v", got.Sets[0])
	}
}

func TestWorkoutHandler_GetWorkout_ResponseContainsOnlyExpectedFields(t *testing.T) {
	serviceWorkout := workout.Workout{
		WorkoutId:   1,
		CompletedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Template:    template.Template{TemplateId: 1, TemplateName: "Push Day"},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: 1, ExerciseName: "Bench Press"}, Reps: 8, WeightGrams: 60000},
		},
	}

	service := &mockWorkoutService{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uint32) (workout.Workout, error) {
			return serviceWorkout, nil
		},
	}

	handler := workout.NewWorkoutHandler(service)

	req := newGetWorkoutRequest(1, "1", true)
	rec := httptest.NewRecorder()

	handler.GetWorkout(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	var got map[string]any
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	completedAtJSON, err := json.Marshal(serviceWorkout.CompletedAt)
	if err != nil {
		t.Fatalf("Failed to marshal CompletedAt: %v", err)
	}
	var wantCompletedAt string
	if err := json.Unmarshal(completedAtJSON, &wantCompletedAt); err != nil {
		t.Fatalf("Failed to unmarshal CompletedAt: %v", err)
	}

	want := map[string]any{
		"workout_id":    float64(1),
		"template_name": "Push Day",
		"completed_at":  wantCompletedAt,
		"sets": []any{
			map[string]any{
				"exercise_id":   float64(1),
				"exercise_name": "Bench Press",
				"reps":          float64(8),
				"weight_grams":  float64(60000),
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetWorkout() response fields = %v, want exactly %v", got, want)
	}
}

func TestWorkoutHandler_GetWorkout_Unauthorized(t *testing.T) {
	service := &mockWorkoutService{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uint32) (workout.Workout, error) {
			t.Fatal("GetWorkout should not be called without an authenticated user")
			return workout.Workout{}, nil
		},
	}

	handler := workout.NewWorkoutHandler(service)

	req := newGetWorkoutRequest(0, "1", false)
	rec := httptest.NewRecorder()

	handler.GetWorkout(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestWorkoutHandler_GetWorkout_InvalidWorkoutId(t *testing.T) {
	service := &mockWorkoutService{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uint32) (workout.Workout, error) {
			t.Fatal("GetWorkout should not be called for a malformed workout_id")
			return workout.Workout{}, nil
		},
	}

	handler := workout.NewWorkoutHandler(service)

	req := newGetWorkoutRequest(1, "not-a-number", true)
	rec := httptest.NewRecorder()

	handler.GetWorkout(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestWorkoutHandler_GetWorkout_NotFound(t *testing.T) {
	service := &mockWorkoutService{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uint32) (workout.Workout, error) {
			return workout.Workout{}, workout.ErrWorkoutNotFound
		},
	}

	handler := workout.NewWorkoutHandler(service)

	req := newGetWorkoutRequest(1, "999999", true)
	rec := httptest.NewRecorder()

	handler.GetWorkout(rec, req)

	if rec.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", rec.Result().StatusCode)
	}
}

func TestWorkoutHandler_GetWorkout_ServiceError(t *testing.T) {
	service := &mockWorkoutService{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uint32) (workout.Workout, error) {
			return workout.Workout{}, errors.New("db exploded")
		},
	}

	handler := workout.NewWorkoutHandler(service)

	req := newGetWorkoutRequest(1, "1", true)
	rec := httptest.NewRecorder()

	handler.GetWorkout(rec, req)

	if rec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", rec.Result().StatusCode)
	}
}
