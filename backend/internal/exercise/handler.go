package exercise

import (
	"context"
	"errors"
	"net/http"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/httpx"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/identity"
)

type ExerciseService interface {
	GetGlobalExercises(context.Context) ([]Exercise, error)
	GetExercises(context.Context, uint32) ([]Exercise, error)
	CreateExercise(context.Context, Exercise) (exercise Exercise, created bool, err error)
	ModifyExercise(context.Context, Exercise) (Exercise, error)
}

type ExerciseHandler struct {
	service ExerciseService
}

func NewExerciseHandler(service ExerciseService) *ExerciseHandler {
	return &ExerciseHandler{service: service}
}

type createExerciseRequest struct {
	ExerciseId   uuid.UUID `json:"exercise_id"`
	ExerciseName string    `json:"exercise_name"`
}

type modifyExerciseRequest struct {
	ExerciseName string `json:"exercise_name"`
}

type ExerciseResponse struct {
	ExerciseId   uuid.UUID `json:"exercise_id"`
	ExerciseName string    `json:"exercise_name"`
}

type ExercisesResponse struct {
	Exercises []ExerciseResponse `json:"exercises"`
}

// GetGlobalExercises responds with the exercises every user can see. It needs no session.
func (h *ExerciseHandler) GetGlobalExercises(w http.ResponseWriter, r *http.Request) {
	exercises, err := h.service.GetGlobalExercises(r.Context())
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	writeExercises(w, exercises)
}

// GetExercises responds with the user's own exercises, without the global ones, which are at
// GET /global-exercises.
func (h *ExerciseHandler) GetExercises(w http.ResponseWriter, r *http.Request) {
	userId, ok := identity.RequireUserId(w, r)
	if !ok {
		return
	}

	exercises, err := h.service.GetExercises(r.Context(), userId)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}
	writeExercises(w, exercises)
}

func writeExercises(w http.ResponseWriter, exercises []Exercise) {
	exercisesResponse := make([]ExerciseResponse, len(exercises))
	for i, e := range exercises {
		exercisesResponse[i] = ExerciseResponse{ExerciseId: e.ExerciseId, ExerciseName: e.ExerciseName}
	}

	httpx.WriteJSON(w, http.StatusOK, ExercisesResponse{Exercises: exercisesResponse})
}

// CreateExercise creates one of the user's own exercises, under the exercise_id the client sends
// or a new one if it sends none, and responds 201 Created. It never makes a second exercise with
// the same id or name: if the user already has the exercise_id, or the user or a global exercise
// already has the name, ignoring case, nothing is created and that exercise is returned with
// 200 OK. So retrying a create is safe, and a client that created the exercise offline gets back
// the existing one, whose exercise_id it should use in place of its own. An exercise_id belonging
// to another user's exercise is 409 Conflict.
func (h *ExerciseHandler) CreateExercise(w http.ResponseWriter, r *http.Request) {
	userId, ok := identity.RequireUserId(w, r)
	if !ok {
		return
	}

	var req createExerciseRequest
	if !httpx.DecodeJSONBody(w, r, &req) {
		return
	}

	exercise := Exercise{ExerciseId: req.ExerciseId, ExerciseName: req.ExerciseName, UserId: userId}

	stored, created, err := h.service.CreateExercise(r.Context(), exercise)
	switch {
	case errors.Is(err, ErrExerciseNameRequired):
		http.Error(w, "exercise_name is required", http.StatusBadRequest)
		return
	case errors.Is(err, ErrExerciseIdTaken):
		http.Error(w, "exercise_id is already in use", http.StatusConflict)
		return
	case err != nil:
		httpx.InternalError(w, err)
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpx.WriteJSON(w, status, ExerciseResponse{ExerciseId: stored.ExerciseId, ExerciseName: stored.ExerciseName})
}

func (h *ExerciseHandler) ModifyExercise(w http.ResponseWriter, r *http.Request) {
	userId, ok := identity.RequireUserId(w, r)
	if !ok {
		return
	}

	exerciseId, err := uuid.Parse(r.PathValue("exerciseId"))
	if err != nil {
		http.Error(w, "Invalid exercise_id", http.StatusBadRequest)
		return
	}

	var req modifyExerciseRequest
	if !httpx.DecodeJSONBody(w, r, &req) {
		return
	}

	exercise := Exercise{ExerciseName: req.ExerciseName, ExerciseId: exerciseId, UserId: userId}

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
		httpx.InternalError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ExerciseResponse{ExerciseId: modifiedExercise.ExerciseId, ExerciseName: modifiedExercise.ExerciseName})
}
