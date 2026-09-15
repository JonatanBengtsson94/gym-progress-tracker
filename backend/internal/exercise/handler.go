package exercise

import (
	"context"
	"errors"
	"net/http"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/httpx"
)

type ExerciseService interface {
	GetExercises(context.Context, uint32) ([]Exercise, error)
	CreateExercise(context.Context, Exercise) (Exercise, error)
	ModifyExercise(context.Context, Exercise) (Exercise, error)
}

type ExerciseHandler struct {
	service ExerciseService
}

func NewExerciseHandler(service ExerciseService) *ExerciseHandler {
	return &ExerciseHandler{service: service}
}

type createExerciseRequest struct {
	ExerciseName string `json:"exercise_name"`
}

type modifyExerciseRequest struct {
	ExerciseName string `json:"exercise_name"`
	ExerciseId   uint32 `json:"exercise_id"`
}

type ExerciseResponse struct {
	ExerciseId   uint32 `json:"exercise_id"`
	ExerciseName string `json:"exercise_name"`
}

func (h *ExerciseHandler) GetExercises(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.RequireUserId(w, r)
	if !ok {
		return
	}

	exercises, err := h.service.GetExercises(r.Context(), userId)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]ExerciseResponse, len(exercises))
	for i, e := range exercises {
		response[i] = ExerciseResponse{ExerciseId: e.ExerciseId, ExerciseName: e.ExerciseName}
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}

func (h *ExerciseHandler) CreateExercise(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.RequireUserId(w, r)
	if !ok {
		return
	}

	var req createExerciseRequest
	if !httpx.DecodeJSONBody(w, r, &req) {
		return
	}

	exercise := Exercise{ExerciseName: req.ExerciseName, UserId: userId}

	createdExercise, err := h.service.CreateExercise(r.Context(), exercise)
	switch {
	case errors.Is(err, ErrExerciseNameRequired):
		http.Error(w, "exercise_name is required", http.StatusBadRequest)
		return
	case errors.Is(err, ErrExerciseAlreadyExists):
		http.Error(w, "Exercise already exists", http.StatusConflict)
		return
	case err != nil:
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, ExerciseResponse{ExerciseId: createdExercise.ExerciseId, ExerciseName: createdExercise.ExerciseName})
}

func (h *ExerciseHandler) ModifyExercise(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.RequireUserId(w, r)
	if !ok {
		return
	}

	var req modifyExerciseRequest
	if !httpx.DecodeJSONBody(w, r, &req) {
		return
	}

	exercise := Exercise{ExerciseName: req.ExerciseName, ExerciseId: req.ExerciseId, UserId: userId}

	modifiedExercise, err := h.service.ModifyExercise(r.Context(), exercise)
	switch {
	case errors.Is(err, ErrExerciseNameRequired):
		http.Error(w, "exercise_name is required", http.StatusBadRequest)
		return
	case errors.Is(err, ErrExerciseAlreadyExists):
		http.Error(w, "Exercise already exists", http.StatusConflict)
		return
	case errors.Is(err, ErrExerciseNotFound):
		http.Error(w, "Exercise not found", http.StatusNotFound)
		return
	case errors.Is(err, ErrCannotModifyGlobalExercise):
		http.Error(w, "Cannot modify a global exercise", http.StatusForbidden)
		return
	case err != nil:
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ExerciseResponse{ExerciseId: modifiedExercise.ExerciseId, ExerciseName: modifiedExercise.ExerciseName})
}
