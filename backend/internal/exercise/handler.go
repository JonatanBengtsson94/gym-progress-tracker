package exercise

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
)

type ExerciseGetterService interface {
	GetExercises(context.Context, uint32) ([]Exercise, error)
}

type ExerciseCreatorService interface {
	CreateExercise(context.Context, Exercise) (Exercise, error)
}

type ExerciseService interface {
	ExerciseGetterService
	ExerciseCreatorService
}

type ExerciseHandler struct {
	service ExerciseService
}

func NewExerciseHandler(service ExerciseService) *ExerciseHandler {
	return &ExerciseHandler{service: service}
}

type exerciseRequest struct {
	ExerciseName string `json:"exercise_name"`
}

type ExerciseResponse struct {
	ExerciseId   uint32 `json:"exercise_id"`
	ExerciseName string `json:"exercise_name"`
}

func (h *ExerciseHandler) GetExercises(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.UserIdFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	exercises, err := h.service.GetExercises(r.Context(), userId)
	if err != nil {
		http.Error(w, "Internal server errror", http.StatusInternalServerError)
		return
	}

	response := make([]ExerciseResponse, len(exercises))
	for i, e := range exercises {
		response[i] = ExerciseResponse{ExerciseId: e.ExerciseId, ExerciseName: e.ExerciseName}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ExerciseHandler) CreateExercise(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.UserIdFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req exerciseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.ExerciseName) == "" {
		http.Error(w, "exercise_name is required", http.StatusBadRequest)
		return
	}

	exercise := Exercise{ExerciseName: req.ExerciseName, UserId: userId}

	createdExercise, err := h.service.CreateExercise(r.Context(), exercise)
	if errors.Is(err, ErrExerciseAlreadyExists) {
		http.Error(w, "Exercise already exists", http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ExerciseResponse{ExerciseId: createdExercise.ExerciseId, ExerciseName: createdExercise.ExerciseName})
}
