package workout

import (
	"context"
	"errors"
	"net/http"
	"time"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/httpx"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/identity"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/set"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/template"
)

type WorkoutService interface {
	GetWorkout(context.Context, uint32, uuid.UUID) (Workout, error)
	GetWorkouts(context.Context, uint32) ([]Workout, error)
	GetWorkoutsByTemplate(context.Context, uint32, uuid.UUID) ([]Workout, error)
	PutWorkout(context.Context, uint32, Workout) (stored Workout, created bool, err error)
}

type WorkoutHandler struct {
	service WorkoutService
}

func NewWorkoutHandler(service WorkoutService) *WorkoutHandler {
	return &WorkoutHandler{service: service}
}

type workoutSet struct {
	Reps        uint8  `json:"reps"`
	WeightGrams uint32 `json:"weight_grams"`
}

type WorkoutResponse struct {
	WorkoutId    uuid.UUID                  `json:"workout_id"`
	TemplateId   uuid.UUID                  `json:"template_id"`
	TemplateName string                     `json:"template_name"`
	StartedAt    time.Time                  `json:"started_at"`
	CompletedAt  time.Time                  `json:"completed_at"`
	Exercises    []set.ExerciseSetsResponse `json:"exercises"`
}

// toWorkoutResponse groups the workout's flat set list by exercise.
func toWorkoutResponse(workout Workout) WorkoutResponse {
	return WorkoutResponse{
		WorkoutId:    workout.WorkoutId,
		TemplateId:   workout.Template.TemplateId,
		TemplateName: workout.Template.TemplateName,
		StartedAt:    workout.StartedAt,
		CompletedAt:  workout.CompletedAt,
		Exercises:    set.GroupByExercise(workout.Sets),
	}
}

func (h *WorkoutHandler) GetWorkout(w http.ResponseWriter, r *http.Request) {
	userId, ok := identity.RequireUserId(w, r)
	if !ok {
		return
	}

	workoutId, err := uuid.Parse(r.PathValue("workoutId"))
	if err != nil {
		http.Error(w, "Invalid workout_id", http.StatusBadRequest)
		return
	}

	workout, err := h.service.GetWorkout(r.Context(), userId, workoutId)
	switch {
	case errors.Is(err, ErrWorkoutNotFound):
		http.Error(w, "Workout not found", http.StatusNotFound)
		return
	case err != nil:
		httpx.InternalError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toWorkoutResponse(workout))
}

type workoutSummary struct {
	WorkoutId    uuid.UUID `json:"workout_id"`
	TemplateId   uuid.UUID `json:"template_id"`
	TemplateName string    `json:"template_name"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at"`
}

type WorkoutsResponse struct {
	Workouts []workoutSummary `json:"workouts"`
}

func toWorkoutsResponse(workouts []Workout) WorkoutsResponse {
	summaries := make([]workoutSummary, len(workouts))
	for i, wo := range workouts {
		summaries[i] = workoutSummary{
			WorkoutId:    wo.WorkoutId,
			TemplateId:   wo.Template.TemplateId,
			TemplateName: wo.Template.TemplateName,
			StartedAt:    wo.StartedAt,
			CompletedAt:  wo.CompletedAt}
	}
	return WorkoutsResponse{Workouts: summaries}
}

// GetWorkouts lists the user's workouts, newest first. An optional
// template_id query parameter narrows the list to one of the user's
// templates; an unknown or foreign template is a 404 rather than an empty
// list, so clients can tell it apart from a template with no workouts yet.
func (h *WorkoutHandler) GetWorkouts(w http.ResponseWriter, r *http.Request) {
	userId, ok := identity.RequireUserId(w, r)
	if !ok {
		return
	}

	var workouts []Workout
	var err error
	if query := r.URL.Query(); query.Has("template_id") {
		templateId, parseErr := uuid.Parse(query.Get("template_id"))
		if parseErr != nil {
			http.Error(w, "Invalid template_id", http.StatusBadRequest)
			return
		}
		workouts, err = h.service.GetWorkoutsByTemplate(r.Context(), userId, templateId)
	} else {
		workouts, err = h.service.GetWorkouts(r.Context(), userId)
	}

	switch {
	case errors.Is(err, ErrTemplateNotFound):
		http.Error(w, "Template not found", http.StatusNotFound)
		return
	case err != nil:
		httpx.InternalError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toWorkoutsResponse(workouts))
}

type workoutRequestExercise struct {
	ExerciseId uuid.UUID    `json:"exercise_id"`
	Sets       []workoutSet `json:"sets"`
}

// toSets flattens the exercise groups of a request into a flat list of sets,
// keeping the order they were sent in.
func toSets(exercises []workoutRequestExercise) []set.Set {
	var sets []set.Set
	for _, e := range exercises {
		for _, s := range e.Sets {
			sets = append(sets, set.Set{
				Exercise:    exercise.Exercise{ExerciseId: e.ExerciseId},
				Reps:        s.Reps,
				WeightGrams: s.WeightGrams,
			})
		}
	}
	return sets
}

// writeWorkoutError maps an error from the workout service to an HTTP
// response, falling back to a logged 500 for anything unexpected.
func writeWorkoutError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrWorkoutIdTaken):
		http.Error(w, "workout_id is already in use", http.StatusConflict)
	case errors.Is(err, ErrSetsRequired):
		http.Error(w, "exercises must contain at least one set", http.StatusBadRequest)
	case errors.Is(err, ErrRepsRequired):
		http.Error(w, "reps must be greater than zero", http.StatusBadRequest)
	case errors.Is(err, ErrWeightGramsOutOfRange):
		http.Error(w, "weight_grams is out of range", http.StatusBadRequest)
	case errors.Is(err, ErrStartedAtRequired):
		http.Error(w, "started_at is required", http.StatusBadRequest)
	case errors.Is(err, ErrCompletedAtRequired):
		http.Error(w, "completed_at is required", http.StatusBadRequest)
	case errors.Is(err, ErrStartedAfterCompleted):
		http.Error(w, "started_at must not be after completed_at", http.StatusBadRequest)
	case errors.Is(err, template.ErrTemplateNameRequired):
		http.Error(w, "template_name is required when template_id is omitted", http.StatusBadRequest)
	case errors.Is(err, ErrExerciseNotFound):
		http.Error(w, "Unknown exercise_id in sets", http.StatusBadRequest)
	case errors.Is(err, ErrTemplateNotFound):
		http.Error(w, "Template not found", http.StatusNotFound)
	case errors.Is(err, template.ErrTemplateAlreadyExists):
		http.Error(w, "Template already exists", http.StatusConflict)
	default:
		httpx.InternalError(w, err)
	}
}

// putWorkoutRequest's template fields only matter when the PUT creates the
// workout: an existing workout keeps the template it was logged under, so
// template fields a client sends back (for example from a GET response) are
// ignored.
type putWorkoutRequest struct {
	TemplateId   uuid.UUID                `json:"template_id"`
	TemplateName string                   `json:"template_name"`
	StartedAt    time.Time                `json:"started_at"`
	CompletedAt  time.Time                `json:"completed_at"`
	Exercises    []workoutRequestExercise `json:"exercises"`
}

// PutWorkout makes the user's workout with the workout_id in the path match
// the body, so sending the same body again changes nothing. A workout that
// doesn't exist yet is created, under template_id or a new template named
// template_name, and gets 201 Created; an existing one has its times and sets
// replaced and gets 200 OK. A client that logged workouts offline can PUT each
// under the id it chose, as often as it needs to. A workout_id belonging to
// another user's workout is 409 Conflict.
func (h *WorkoutHandler) PutWorkout(w http.ResponseWriter, r *http.Request) {
	userId, ok := identity.RequireUserId(w, r)
	if !ok {
		return
	}

	workoutId, err := uuid.Parse(r.PathValue("workoutId"))
	if err != nil {
		http.Error(w, "Invalid workout_id", http.StatusBadRequest)
		return
	}

	var req putWorkoutRequest
	if !httpx.DecodeJSONBody(w, r, &req) {
		return
	}

	stored, created, err := h.service.PutWorkout(r.Context(), userId, Workout{
		WorkoutId:   workoutId,
		StartedAt:   req.StartedAt,
		CompletedAt: req.CompletedAt,
		Template:    template.Template{TemplateId: req.TemplateId, TemplateName: req.TemplateName},
		Sets:        toSets(req.Exercises),
	})
	if err != nil {
		writeWorkoutError(w, err)
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpx.WriteJSON(w, status, toWorkoutResponse(stored))
}
