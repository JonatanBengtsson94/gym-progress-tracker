package server

import (
	"net/http"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
)

func NewRouter(exerciseHandler *exercise.ExerciseHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/exercises", exerciseHandler.GetExercises)
	return mux
}
