package workout

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/httpx"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/identity"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/set"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/template"
)

type WorkoutService interface {
	GetWorkout(context.Context, uint32, uint32) (Workout, error)
	GetWorkouts(context.Context, uint32) ([]Workout, error)
	GetWorkoutsByTemplate(context.Context, uint32, uint32) ([]Workout, error)
	CreateWorkout(context.Context, uint32, Workout) (Workout, error)
	ModifyWorkout(context.Context, uint32, Workout) (Workout, error)
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

type workoutResponseExercise struct {
	ExerciseId   uint32       `json:"exercise_id"`
	ExerciseName string       `json:"exercise_name"`
	Sets         []workoutSet `json:"sets"`
}

type WorkoutResponse struct {
	WorkoutId    uint32                    `json:"workout_id"`
	TemplateId   uint32                    `json:"template_id"`
	TemplateName string                    `json:"template_name"`
	CompletedAt  time.Time                 `json:"completed_at"`
	Exercises    []workoutResponseExercise `json:"exercises"`
}

// toWorkoutResponse groups the workout's flat set list by exercise. Exercises
// keep the order they first appear in, so repeating an exercise later in the
// workout adds to its existing group rather than starting a second one.
func toWorkoutResponse(workout Workout) WorkoutResponse {
	exercises := make([]workoutResponseExercise, 0, len(workout.Sets))
	indexByExerciseId := make(map[uint32]int, len(workout.Sets))

	for _, s := range workout.Sets {
		i, ok := indexByExerciseId[s.Exercise.ExerciseId]
		if !ok {
			i = len(exercises)
			indexByExerciseId[s.Exercise.ExerciseId] = i
			exercises = append(exercises, workoutResponseExercise{
				ExerciseId:   s.Exercise.ExerciseId,
				ExerciseName: s.Exercise.ExerciseName,
			})
		}
		exercises[i].Sets = append(exercises[i].Sets, workoutSet{
			Reps:        s.Reps,
			WeightGrams: s.WeightGrams,
		})
	}

	return WorkoutResponse{
		WorkoutId:    workout.WorkoutId,
		TemplateId:   workout.Template.TemplateId,
		TemplateName: workout.Template.TemplateName,
		CompletedAt:  workout.CompletedAt,
		Exercises:    exercises,
	}
}

func (h *WorkoutHandler) GetWorkout(w http.ResponseWriter, r *http.Request) {
	userId, ok := identity.RequireUserId(w, r)
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

	httpx.WriteJSON(w, http.StatusOK, toWorkoutResponse(workout))
}

type workoutSummary struct {
	WorkoutId    uint32    `json:"workout_id"`
	TemplateId   uint32    `json:"template_id"`
	TemplateName string    `json:"template_name"`
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
		templateId, parseErr := strconv.ParseUint(query.Get("template_id"), 10, 32)
		if parseErr != nil {
			http.Error(w, "Invalid template_id", http.StatusBadRequest)
			return
		}
		workouts, err = h.service.GetWorkoutsByTemplate(r.Context(), userId, uint32(templateId))
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
	ExerciseId uint32       `json:"exercise_id"`
	Sets       []workoutSet `json:"sets"`
}

type createWorkoutRequest struct {
	TemplateId   uint32                   `json:"template_id"`
	TemplateName string                   `json:"template_name"`
	CompletedAt  time.Time                `json:"completed_at"`
	Exercises    []workoutRequestExercise `json:"exercises"`
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
	case errors.Is(err, ErrWorkoutNotFound):
		http.Error(w, "Workout not found", http.StatusNotFound)
	case errors.Is(err, ErrSetsRequired):
		http.Error(w, "exercises must contain at least one set", http.StatusBadRequest)
	case errors.Is(err, ErrRepsRequired):
		http.Error(w, "reps must be greater than zero", http.StatusBadRequest)
	case errors.Is(err, ErrWeightGramsOutOfRange):
		http.Error(w, "weight_grams is out of range", http.StatusBadRequest)
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

func (h *WorkoutHandler) CreateWorkout(w http.ResponseWriter, r *http.Request) {
	userId, ok := identity.RequireUserId(w, r)
	if !ok {
		return
	}

	var req createWorkoutRequest
	if !httpx.DecodeJSONBody(w, r, &req) {
		return
	}

	created, err := h.service.CreateWorkout(r.Context(), userId, Workout{
		CompletedAt: req.CompletedAt,
		Template:    template.Template{TemplateId: req.TemplateId, TemplateName: req.TemplateName},
		Sets:        toSets(req.Exercises),
	})
	if err != nil {
		writeWorkoutError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toWorkoutResponse(created))
}

// modifyWorkoutRequest deliberately has no template fields: a workout keeps
// the template it was logged under, so any template_id or template_name a
// client sends back (for example from a GET response) is ignored.
type modifyWorkoutRequest struct {
	CompletedAt time.Time                `json:"completed_at"`
	Exercises   []workoutRequestExercise `json:"exercises"`
}

func (h *WorkoutHandler) ModifyWorkout(w http.ResponseWriter, r *http.Request) {
	userId, ok := identity.RequireUserId(w, r)
	if !ok {
		return
	}

	workoutId, err := strconv.ParseUint(r.PathValue("workoutId"), 10, 32)
	if err != nil {
		http.Error(w, "Invalid workout_id", http.StatusBadRequest)
		return
	}

	var req modifyWorkoutRequest
	if !httpx.DecodeJSONBody(w, r, &req) {
		return
	}

	modified, err := h.service.ModifyWorkout(r.Context(), userId, Workout{
		WorkoutId:   uint32(workoutId),
		CompletedAt: req.CompletedAt,
		Sets:        toSets(req.Exercises),
	})
	if err != nil {
		writeWorkoutError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toWorkoutResponse(modified))
}
