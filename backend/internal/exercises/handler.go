package exercises

import (
	"encoding/json"
	"net/http"
)

type ExerciseHandler struct {
	service *ExerciseService
}

func NewExerciseHandler(service *ExerciseService) *ExerciseHandler {
	return &ExerciseHandler{service: service}
}

func (h *ExerciseHandler) GetExercises(w http.ResponseWriter, r *http.Request) {
	exercises, err := h.service.GetExercises()
	if err != nil {
		http.Error(w, "Internal server errror", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exercises)
}
