package workout

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/httpx"
)

type WorkoutService interface {
	GetWorkout(context.Context, uint32, uint32) (Workout, error)
}

type WorkoutHandler struct {
	service WorkoutService
}

func NewWorkoutHandler(service WorkoutService) *WorkoutHandler {
	return &WorkoutHandler{service: service}
}

type workoutSet struct {
	ExerciseId   uint32 `json:"exercise_id"`
	ExerciseName string `json:"exercise_name"`
	Reps         uint8  `json:"reps"`
	WeightGrams  uint32 `json:"weight_grams"`
}

type WorkoutResponse struct {
	WorkoutId    uint32       `json:"workout_id"`
	TemplateName string       `json:"template_name"`
	CompletedAt  time.Time    `json:"completed_at"`
	Sets         []workoutSet `json:"sets"`
}

func (h *WorkoutHandler) GetWorkout(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.RequireUserId(w, r)
	if !ok {
		return
	}

	workoutId, err := strconv.ParseUint(r.PathValue("workoutId"), 10, 32)
	if err != nil {
		http.Error(w, "Invalid workout_id", http.StatusBadRequest)
		return
	}

	workout, err := h.service.GetWorkout(r.Context(), userId, uint32(workoutId))
	switch {
	case errors.Is(err, ErrWorkoutNotFound):
		http.Error(w, "Workout not found", http.StatusNotFound)
		return
	case err != nil:
		httpx.InternalError(w, err)
		return
	}

	sets := make([]workoutSet, len(workout.Sets))
	for i, s := range workout.Sets {
		sets[i] = workoutSet{
			ExerciseId:   s.Exercise.ExerciseId,
			ExerciseName: s.Exercise.ExerciseName,
			Reps:         s.Reps,
			WeightGrams:  s.WeightGrams,
		}
	}

	httpx.WriteJSON(w, http.StatusOK, WorkoutResponse{
		WorkoutId:    workout.WorkoutId,
		TemplateName: workout.Template.TemplateName,
		CompletedAt:  workout.CompletedAt,
		Sets:         sets,
	})
}
