package server

import (
	"net/http"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/workout"
)

func NewRouter(
	exerciseHandler *exercise.ExerciseHandler,
	workoutHandler *workout.WorkoutHandler,
	authHandler *auth.AuthHandler,
	authMiddleware *auth.AuthMiddleware,
) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /exercises", authMiddleware.RequireAuth(http.HandlerFunc(exerciseHandler.GetExercises)))
	mux.Handle("POST /exercises", authMiddleware.RequireAuth(http.HandlerFunc(exerciseHandler.CreateExercise)))
	mux.Handle("PATCH /exercises", authMiddleware.RequireAuth(http.HandlerFunc(exerciseHandler.ModifyExercise)))
	mux.Handle("GET /workouts/{workoutId}", authMiddleware.RequireAuth(http.HandlerFunc(workoutHandler.GetWorkout)))
	mux.HandleFunc("POST /login", authHandler.Login)
	return mux
}
