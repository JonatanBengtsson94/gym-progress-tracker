package server

import (
	"net/http"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/httpx"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/user"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/workout"
)

func NewRouter(
	userHandler *user.UserHandler,
	exerciseHandler *exercise.ExerciseHandler,
	workoutHandler *workout.WorkoutHandler,
	templateHandler *template.TemplateHandler,
	authHandler *auth.AuthHandler,
	authMiddleware *auth.AuthMiddleware,
) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /users/me", authMiddleware.RequireAuth(http.HandlerFunc(userHandler.GetUserFromSession)))
	mux.Handle("GET /exercises", authMiddleware.RequireAuth(http.HandlerFunc(exerciseHandler.GetExercises)))
	mux.Handle("POST /exercises", authMiddleware.RequireAuth(http.HandlerFunc(exerciseHandler.CreateExercise)))
	mux.Handle("PUT /exercises/{exerciseId}", authMiddleware.RequireAuth(http.HandlerFunc(exerciseHandler.ModifyExercise)))
	mux.Handle("GET /workouts/{workoutId}", authMiddleware.RequireAuth(http.HandlerFunc(workoutHandler.GetWorkout)))
	mux.Handle("PUT /workouts/{workoutId}", authMiddleware.RequireAuth(http.HandlerFunc(workoutHandler.ModifyWorkout)))
	mux.Handle("POST /workouts", authMiddleware.RequireAuth(http.HandlerFunc(workoutHandler.CreateWorkout)))
	mux.Handle("POST /templates", authMiddleware.RequireAuth(http.HandlerFunc(templateHandler.CreateTemplate)))
	mux.HandleFunc("POST /login", authHandler.Login)
	return httpx.LoggingMiddleware(mux)
}
