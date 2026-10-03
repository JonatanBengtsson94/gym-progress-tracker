package exercise_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/identity"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/testutil"
)

// The exercises the contract files describe.
var (
	contractBenchPress   = exercise.Exercise{ExerciseId: uuid.MustParse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d01"), ExerciseName: "Bench Press (Barbell)"}
	contractZercherSquat = exercise.Exercise{ExerciseId: uuid.MustParse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d02"), ExerciseName: "Zercher Squat", UserId: 1}
)

func TestExerciseContract_GetExercises(t *testing.T) {
	service := &mockExerciseService{
		getExercisesFunc: func(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
			return []exercise.Exercise{contractBenchPress, contractZercherSquat}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	exercise.NewExerciseHandler(service).GetExercises(rec, req)

	testutil.AssertJSONEqual(t, rec.Body.Bytes(), testutil.ContractFile(t, "get_exercises.response.json"))
}

func TestExerciseContract_CreateExercise(t *testing.T) {
	var got exercise.Exercise
	service := &mockExerciseService{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
			got = ex
			return ex, true, nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/exercises", bytes.NewReader(testutil.ContractFile(t, "post_exercises.request.json")))
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	exercise.NewExerciseHandler(service).CreateExercise(rec, req)

	if got != contractZercherSquat {
		t.Errorf("expected the request to reach the service as %+v, got %+v", contractZercherSquat, got)
	}
	testutil.AssertJSONEqual(t, rec.Body.Bytes(), testutil.ContractFile(t, "post_exercises.response.json"))
}
