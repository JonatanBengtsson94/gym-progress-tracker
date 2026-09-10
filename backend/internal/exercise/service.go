package exercise

import (
	"context"
)

type ExerciseGetter interface {
	GetExercises(context.Context, uint32) ([]Exercise, error)
}

type ExerciseCreator interface {
	CreateExercise(context.Context, Exercise) (Exercise, error)
}

type ExerciseRepository interface {
	ExerciseGetter
	ExerciseCreator
}

type ExerciseServiceImpl struct {
	repo ExerciseRepository
}

func NewExerciseService(repo ExerciseRepository) *ExerciseServiceImpl {
	return &ExerciseServiceImpl{repo: repo}
}

func (s *ExerciseServiceImpl) GetExercises(ctx context.Context, userId uint32) ([]Exercise, error) {
	return s.repo.GetExercises(ctx, userId)
}

func (s *ExerciseServiceImpl) CreateExercise(ctx context.Context, exercise Exercise) (Exercise, error) {
	return s.repo.CreateExercise(ctx, exercise)
}
