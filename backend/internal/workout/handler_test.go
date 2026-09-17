package workout_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/identity"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/set"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/workout"
)

type mockWorkoutService struct {
	getWorkoutFunc    func(ctx context.Context, userId uint32, workoutId uint32) (workout.Workout, error)
	createWorkoutFunc func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error)
}

func (m *mockWorkoutService) GetWorkout(ctx context.Context, userId uint32, workoutId uint32) (workout.Workout, error) {
	return m.getWorkoutFunc(ctx, userId, workoutId)
}

func (m *mockWorkoutService) CreateWorkout(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
	return m.createWorkoutFunc(ctx, userId, w)
}

func newGetWorkoutRequest(userId uint32, workoutId string, authenticated bool) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/workouts/"+workoutId, nil)
	req.SetPathValue("workoutId", workoutId)
	if authenticated {
		req = req.WithContext(identity.ContextWithUserId(req.Context(), userId))
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
	if len(got.Exercises) != 1 {
		t.Fatalf("expected 1 exercise, got %d: %+v", len(got.Exercises), got.Exercises)
	}
	gotExercise := got.Exercises[0]
	if gotExercise.ExerciseId != 1 || gotExercise.ExerciseName != "Bench Press" {
		t.Errorf("GetWorkout() exercise = %+v", gotExercise)
	}
	if len(gotExercise.Sets) != 1 || gotExercise.Sets[0].Reps != 8 || gotExercise.Sets[0].WeightGrams != 60000 {
		t.Errorf("GetWorkout() sets = %+v", gotExercise.Sets)
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
		"template_id":   float64(1),
		"template_name": "Push Day",
		"completed_at":  wantCompletedAt,
		"exercises": []any{
			map[string]any{
				"exercise_id":   float64(1),
				"exercise_name": "Bench Press",
				"sets": []any{
					map[string]any{
						"reps":         float64(8),
						"weight_grams": float64(60000),
					},
				},
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetWorkout() response fields = %v, want exactly %v", got, want)
	}
}

func TestWorkoutHandler_GetWorkout_GroupsSetsByExercise(t *testing.T) {
	benchPress := exercise.Exercise{ExerciseId: 1, ExerciseName: "Bench Press"}
	squat := exercise.Exercise{ExerciseId: 2, ExerciseName: "Squat"}

	serviceWorkout := workout.Workout{
		WorkoutId:   1,
		CompletedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Template:    template.Template{TemplateId: 1, TemplateName: "Push Day"},
		Sets: []set.Set{
			{Exercise: benchPress, Reps: 8, WeightGrams: 60000},
			{Exercise: squat, Reps: 5, WeightGrams: 100000},
			// Returning to an earlier exercise joins its existing group.
			{Exercise: benchPress, Reps: 6, WeightGrams: 65000},
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

	var got workout.WorkoutResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if len(got.Exercises) != 2 {
		t.Fatalf("expected 2 exercise groups, got %d: %+v", len(got.Exercises), got.Exercises)
	}

	// Groups keep the order the exercises first appear in.
	if got.Exercises[0].ExerciseId != 1 || got.Exercises[1].ExerciseId != 2 {
		t.Errorf("unexpected exercise order: %+v", got.Exercises)
	}

	if len(got.Exercises[0].Sets) != 2 {
		t.Fatalf("expected Bench Press to have 2 sets, got %+v", got.Exercises[0].Sets)
	}
	if got.Exercises[0].Sets[0].Reps != 8 || got.Exercises[0].Sets[1].Reps != 6 {
		t.Errorf("expected Bench Press sets in original order, got %+v", got.Exercises[0].Sets)
	}
	if len(got.Exercises[1].Sets) != 1 || got.Exercises[1].Sets[0].Reps != 5 {
		t.Errorf("expected Squat to have one 5-rep set, got %+v", got.Exercises[1].Sets)
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

func newCreateWorkoutRequest(userId uint32, body string, authenticated bool) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/workouts", strings.NewReader(body))
	if authenticated {
		req = req.WithContext(identity.ContextWithUserId(req.Context(), userId))
	}
	return req
}

func TestWorkoutHandler_CreateWorkout_Success(t *testing.T) {
	completedAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	created := workout.Workout{
		WorkoutId:   7,
		CompletedAt: completedAt,
		Template:    template.Template{TemplateId: 1, TemplateName: "Push Day"},
		Sets: []set.Set{
			{Exercise: exercise.Exercise{ExerciseId: 1, ExerciseName: "Bench Press"}, Reps: 8, WeightGrams: 60000},
		},
	}

	var gotUserId uint32
	var gotWorkout workout.Workout
	service := &mockWorkoutService{
		createWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
			gotUserId = userId
			gotWorkout = w
			return created, nil
		},
	}

	handler := workout.NewWorkoutHandler(service)

	req := newCreateWorkoutRequest(1, `{
		"template_id": 1,
		"completed_at": "2024-01-15T10:00:00Z",
		"exercises": [
			{"exercise_id": 1, "sets": [{"reps": 8, "weight_grams": 60000}]}
		]
	}`, true)
	rec := httptest.NewRecorder()

	handler.CreateWorkout(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", res.StatusCode)
	}

	if gotUserId != 1 {
		t.Errorf("expected service to receive userId 1, got %d", gotUserId)
	}
	if gotWorkout.Template.TemplateId != 1 {
		t.Errorf("expected service to receive TemplateId 1, got %d", gotWorkout.Template.TemplateId)
	}
	if !gotWorkout.CompletedAt.Equal(completedAt) {
		t.Errorf("expected service to receive CompletedAt %v, got %v", completedAt, gotWorkout.CompletedAt)
	}
	if len(gotWorkout.Sets) != 1 {
		t.Fatalf("expected service to receive 1 set, got %d: %+v", len(gotWorkout.Sets), gotWorkout.Sets)
	}
	gotSet := gotWorkout.Sets[0]
	if gotSet.Exercise.ExerciseId != 1 || gotSet.Reps != 8 || gotSet.WeightGrams != 60000 {
		t.Errorf("expected service to receive set {1 8 60000}, got %+v", gotSet)
	}

	var got workout.WorkoutResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}
	if got.WorkoutId != 7 || got.TemplateId != 1 || got.TemplateName != "Push Day" {
		t.Errorf("CreateWorkout() = %+v", got)
	}
	if len(got.Exercises) != 1 || got.Exercises[0].ExerciseName != "Bench Press" {
		t.Errorf("CreateWorkout() exercises = %+v", got.Exercises)
	}
}

func TestWorkoutHandler_CreateWorkout_WithoutTemplateId(t *testing.T) {
	var gotWorkout workout.Workout
	service := &mockWorkoutService{
		createWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
			gotWorkout = w
			return workout.Workout{
				WorkoutId: 7,
				Template:  template.Template{TemplateId: 9, TemplateName: "Leg Day"},
				Sets:      w.Sets,
			}, nil
		},
	}

	handler := workout.NewWorkoutHandler(service)

	req := newCreateWorkoutRequest(1, `{
		"template_name": "Leg Day",
		"exercises": [
			{"exercise_id": 2, "sets": [{"reps": 5, "weight_grams": 100000}]}
		]
	}`, true)
	rec := httptest.NewRecorder()

	handler.CreateWorkout(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", res.StatusCode)
	}
	if gotWorkout.Template.TemplateId != 0 {
		t.Errorf("expected service to receive TemplateId 0, got %d", gotWorkout.Template.TemplateId)
	}
	if gotWorkout.Template.TemplateName != "Leg Day" {
		t.Errorf("expected service to receive TemplateName %q, got %q", "Leg Day", gotWorkout.Template.TemplateName)
	}

	var got workout.WorkoutResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}
	if got.TemplateId != 9 {
		t.Errorf("expected response to expose the generated TemplateId 9, got %d", got.TemplateId)
	}
}

func TestWorkoutHandler_CreateWorkout_Unauthorized(t *testing.T) {
	service := &mockWorkoutService{
		createWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
			t.Fatal("CreateWorkout should not be called without an authenticated user")
			return workout.Workout{}, nil
		},
	}

	handler := workout.NewWorkoutHandler(service)

	req := newCreateWorkoutRequest(0, `{"template_id":1,"exercises":[]}`, false)
	rec := httptest.NewRecorder()

	handler.CreateWorkout(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestWorkoutHandler_CreateWorkout_InvalidBody(t *testing.T) {
	service := &mockWorkoutService{
		createWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
			t.Fatal("CreateWorkout should not be called for a malformed body")
			return workout.Workout{}, nil
		},
	}

	handler := workout.NewWorkoutHandler(service)

	req := newCreateWorkoutRequest(1, `not json`, true)
	rec := httptest.NewRecorder()

	handler.CreateWorkout(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestWorkoutHandler_CreateWorkout_ServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		wantStatus int
	}{
		{"no sets", workout.ErrSetsRequired, http.StatusBadRequest},
		{"zero reps", workout.ErrRepsRequired, http.StatusBadRequest},
		{"weight out of range", workout.ErrWeightGramsOutOfRange, http.StatusBadRequest},
		{"missing template name", template.ErrTemplateNameRequired, http.StatusBadRequest},
		{"unknown exercise", workout.ErrExerciseNotFound, http.StatusBadRequest},
		{"unknown template", workout.ErrTemplateNotFound, http.StatusNotFound},
		{"duplicate template name", template.ErrTemplateAlreadyExists, http.StatusConflict},
		{"unexpected error", errors.New("db exploded"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &mockWorkoutService{
				createWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
					return workout.Workout{}, tt.serviceErr
				},
			}

			handler := workout.NewWorkoutHandler(service)

			req := newCreateWorkoutRequest(1, `{"template_id":1,"exercises":[{"exercise_id":1,"sets":[{"reps":8,"weight_grams":60000}]}]}`, true)
			rec := httptest.NewRecorder()

			handler.CreateWorkout(rec, req)

			if rec.Result().StatusCode != tt.wantStatus {
				t.Fatalf("Expected status %d, got %d", tt.wantStatus, rec.Result().StatusCode)
			}
		})
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
