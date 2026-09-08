package server

import (
	"net/http"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercises"
)

func NewRouter(exerciseHandler *exercises.ExerciseHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/exercises", exerciseHandler.GetExercises)
	return mux
}
