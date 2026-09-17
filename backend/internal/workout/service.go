package workout

import (
	"context"
	"strings"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/template"
)

type WorkoutRepository interface {
	GetWorkoutByUserIdAndWorkoutId(context.Context, uint32, uint32) (Workout, error)
	CreateWorkout(context.Context, uint32, Workout) (Workout, error)
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

func (s *WorkoutServiceImpl) CreateWorkout(ctx context.Context, userId uint32, workout Workout) (Workout, error) {
	if len(workout.Sets) == 0 {
		return Workout{}, ErrSetsRequired
	}
	for _, set := range workout.Sets {
		if set.Reps == 0 {
			return Workout{}, ErrRepsRequired
		}
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
