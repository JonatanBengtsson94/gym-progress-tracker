package workout

import "context"

type WorkoutRepository interface {
	GetWorkoutByUserIdAndWorkoutId(context.Context, uint32, uint32) (Workout, error)
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
