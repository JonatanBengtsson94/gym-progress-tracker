package workout_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/identity"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/set"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/testutil"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/workout"
)

// contractWorkout is the workout the contract files describe.
func contractWorkout() workout.Workout {
	benchPress := exercise.Exercise{ExerciseId: uuid.MustParse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d01"), ExerciseName: "Bench Press (Barbell)"}
	return workout.Workout{
		WorkoutId:   uuid.MustParse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d20"),
		StartedAt:   time.Date(2026, 9, 30, 17, 0, 0, 0, time.UTC),
		CompletedAt: time.Date(2026, 9, 30, 18, 0, 0, 0, time.UTC),
		Template:    template.Template{TemplateId: uuid.MustParse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d10"), TemplateName: "Push Day"},
		Sets: []set.Set{
			{Exercise: benchPress, Reps: 8, WeightGrams: 60000},
			{Exercise: benchPress, Reps: 6, WeightGrams: 62500},
		},
	}
}

func TestWorkoutContract_GetWorkout(t *testing.T) {
	service := &mockWorkoutService{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error) {
			return contractWorkout(), nil
		},
	}

	rec := httptest.NewRecorder()
	workout.NewWorkoutHandler(service).GetWorkout(rec, newGetWorkoutRequest(1, contractWorkout().WorkoutId.String(), true))

	testutil.AssertJSONEqual(t, rec.Body.Bytes(), testutil.ContractFile(t, "workout.response.json"))
}

func TestWorkoutContract_GetWorkouts(t *testing.T) {
	service := &mockWorkoutService{
		getWorkoutsFunc: func(ctx context.Context, userId uint32) ([]workout.Workout, error) {
			return []workout.Workout{contractWorkout()}, nil
		},
	}

	rec := httptest.NewRecorder()
	workout.NewWorkoutHandler(service).GetWorkouts(rec, newGetWorkoutsRequest(1, true))

	testutil.AssertJSONEqual(t, rec.Body.Bytes(), testutil.ContractFile(t, "get_workouts.response.json"))
}

func TestWorkoutContract_PutWorkout(t *testing.T) {
	var got workout.Workout
	service := &mockWorkoutService{
		putWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, bool, error) {
			got = w
			return contractWorkout(), true, nil
		},
	}

	want := contractWorkout()
	req := httptest.NewRequest(http.MethodPut, "/workouts/"+want.WorkoutId.String(), bytes.NewReader(testutil.ContractFile(t, "put_workout.request.json")))
	req.SetPathValue("workoutId", want.WorkoutId.String())
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	workout.NewWorkoutHandler(service).PutWorkout(rec, req)

	// The request names exercises and the template by id only.
	want.Template.TemplateName = ""
	for i := range want.Sets {
		want.Sets[i].Exercise.ExerciseName = ""
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected the request to reach the service as %+v, got %+v", want, got)
	}
	testutil.AssertJSONEqual(t, rec.Body.Bytes(), testutil.ContractFile(t, "workout.response.json"))
}
