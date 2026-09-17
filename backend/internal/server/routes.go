package server

import (
	"net/http"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/httpx"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/workout"
)

func NewRouter(
	exerciseHandler *exercise.ExerciseHandler,
	workoutHandler *workout.WorkoutHandler,
	templateHandler *template.TemplateHandler,
	authHandler *auth.AuthHandler,
	authMiddleware *auth.AuthMiddleware,
) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /exercises", authMiddleware.RequireAuth(http.HandlerFunc(exerciseHandler.GetExercises)))
	mux.Handle("POST /exercises", authMiddleware.RequireAuth(http.HandlerFunc(exerciseHandler.CreateExercise)))
	mux.Handle("PATCH /exercises", authMiddleware.RequireAuth(http.HandlerFunc(exerciseHandler.ModifyExercise)))
	mux.Handle("GET /workouts/{workoutId}", authMiddleware.RequireAuth(http.HandlerFunc(workoutHandler.GetWorkout)))
	mux.Handle("POST /workouts", authMiddleware.RequireAuth(http.HandlerFunc(workoutHandler.CreateWorkout)))
	mux.Handle("POST /templates", authMiddleware.RequireAuth(http.HandlerFunc(templateHandler.CreateTemplate)))
	mux.HandleFunc("POST /login", authHandler.Login)
	return httpx.LoggingMiddleware(mux)
}
