package server

import (
	"net/http"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
)

func NewRouter(
	exerciseHandler *exercise.ExerciseHandler,
	authHandler *auth.AuthHandler,
	authMiddleware *auth.AuthMiddleware,
) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /exercises", authMiddleware.RequireAuth(http.HandlerFunc(exerciseHandler.GetExercises)))
	mux.HandleFunc("POST /login", authHandler.Login)
	return mux
}
