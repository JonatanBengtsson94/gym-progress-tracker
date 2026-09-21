package workout

import (
	"context"
	"strings"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/set"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/template"
)

type WorkoutRepository interface {
	GetWorkoutByUserIdAndWorkoutId(context.Context, uint32, uint32) (Workout, error)
	GetWorkoutsByUserId(context.Context, uint32) ([]Workout, error)
	CreateWorkout(context.Context, uint32, Workout) (Workout, error)
	ModifyWorkout(context.Context, uint32, Workout) (Workout, error)
}

type WorkoutServiceImpl struct {
	repo WorkoutRepository
}

func NewWorkoutService(repo WorkoutRepository) *WorkoutServiceImpl {
	return &WorkoutServiceImpl{repo: repo}
}

func (s *WorkoutServiceImpl) GetWorkout(ctx context.Context, userId uint32, workoutId uint32) (Workout, error) {
	return s.repo.GetWorkoutByUserIdAndWorkoutId(ctx, userId, workoutId)
}

func (s *WorkoutServiceImpl) GetWorkouts(ctx context.Context, userId uint32) ([]Workout, error) {
	return s.repo.GetWorkoutsByUserId(ctx, userId)
}

func (s *WorkoutServiceImpl) CreateWorkout(ctx context.Context, userId uint32, workout Workout) (Workout, error) {
	if err := validateSets(workout.Sets); err != nil {
		return Workout{}, err
	}

	if workout.Template.TemplateId == 0 {
		workout.Template.TemplateName = strings.TrimSpace(workout.Template.TemplateName)
		if workout.Template.TemplateName == "" {
			return Workout{}, template.ErrTemplateNameRequired
		}
	}

	if workout.CompletedAt.IsZero() {
		workout.CompletedAt = time.Now().UTC()
	}

	return s.repo.CreateWorkout(ctx, userId, workout)
}

// ModifyWorkout replaces the completion time and sets of an existing workout.
// The template cannot be changed, so the workout's current one is kept, and an
// omitted CompletedAt keeps the current value rather than defaulting to now.
func (s *WorkoutServiceImpl) ModifyWorkout(ctx context.Context, userId uint32, workout Workout) (Workout, error) {
	if err := validateSets(workout.Sets); err != nil {
		return Workout{}, err
	}

	existing, err := s.repo.GetWorkoutByUserIdAndWorkoutId(ctx, userId, workout.WorkoutId)
	if err != nil {
		return Workout{}, err
	}

	workout.Template = existing.Template
	if workout.CompletedAt.IsZero() {
		workout.CompletedAt = existing.CompletedAt
	}

	return s.repo.ModifyWorkout(ctx, userId, workout)
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
