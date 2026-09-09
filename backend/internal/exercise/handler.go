package exercise

import (
	"context"
	"encoding/json"
	"net/http"
)

type ExerciseGetterService interface {
	GetExercises(context.Context, uint32) ([]Exercise, error)
}

type ExerciseService interface {
	ExerciseGetterService
}

type ExerciseHandler struct {
	service ExerciseService
}

func NewExerciseHandler(service ExerciseService) *ExerciseHandler {
	return &ExerciseHandler{service: service}
}

func (h *ExerciseHandler) GetExercises(w http.ResponseWriter, r *http.Request) {
	// TODO: Get userId from authentication
	const userId uint32 = 1

	exercises, err := h.service.GetExercises(r.Context(), userId)
	if err != nil {
		http.Error(w, "Internal server errror", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exercises)
}
