package workout

import (
	"context"
	"strings"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/set"
)

type WorkoutRepository interface {
	GetWorkoutByUserIdAndWorkoutId(context.Context, uint32, uuid.UUID) (Workout, error)
	GetWorkoutsByUserId(context.Context, uint32) ([]Workout, error)
	GetWorkoutsByUserIdAndTemplateId(context.Context, uint32, uuid.UUID) ([]Workout, error)
	PutWorkout(context.Context, uint32, Workout) (stored Workout, created bool, err error)
}

type WorkoutServiceImpl struct {
	repo WorkoutRepository
}

func NewWorkoutService(repo WorkoutRepository) *WorkoutServiceImpl {
	return &WorkoutServiceImpl{repo: repo}
}

func (s *WorkoutServiceImpl) GetWorkout(ctx context.Context, userId uint32, workoutId uuid.UUID) (Workout, error) {
	return s.repo.GetWorkoutByUserIdAndWorkoutId(ctx, userId, workoutId)
}

func (s *WorkoutServiceImpl) GetWorkouts(ctx context.Context, userId uint32) ([]Workout, error) {
	return s.repo.GetWorkoutsByUserId(ctx, userId)
}

func (s *WorkoutServiceImpl) GetWorkoutsByTemplate(ctx context.Context, userId uint32, templateId uuid.UUID) ([]Workout, error) {
	return s.repo.GetWorkoutsByUserIdAndTemplateId(ctx, userId, templateId)
}

// PutWorkout makes the user's workout with workout.WorkoutId match workout:
// an existing one gets its times and sets replaced and keeps its template,
// and a missing one is created, under its template or a new one named
// Template.TemplateName. created reports which happened.
func (s *WorkoutServiceImpl) PutWorkout(ctx context.Context, userId uint32, workout Workout) (Workout, bool, error) {
	if err := validateSets(workout.Sets); err != nil {
		return Workout{}, false, err
	}
	if workout.StartedAt.IsZero() {
		return Workout{}, false, ErrStartedAtRequired
	}
	if workout.CompletedAt.IsZero() {
		return Workout{}, false, ErrCompletedAtRequired
	}
	if workout.StartedAt.After(workout.CompletedAt) {
		return Workout{}, false, ErrStartedAfterCompleted
	}
	workout.Template.TemplateName = strings.TrimSpace(workout.Template.TemplateName)

	return s.repo.PutWorkout(ctx, userId, workout)
}

func validateSets(sets []set.Set) error {
	if len(sets) == 0 {
		return ErrSetsRequired
	}
	for _, s := range sets {
		if s.Reps == 0 {
			return ErrRepsRequired
		}
	}
	return nil
}
