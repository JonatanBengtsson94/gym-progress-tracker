package exercise

import (
	"context"
	"strings"
)

type ExerciseRepository interface {
	GetExercises(context.Context, uint32) ([]Exercise, error)
	CreateExercise(context.Context, Exercise) (Exercise, error)
	ModifyExercise(context.Context, Exercise) (Exercise, error)
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
	if strings.TrimSpace(exercise.ExerciseName) == "" {
		return Exercise{}, ErrExerciseNameRequired
	}
	return s.repo.CreateExercise(ctx, exercise)
}

func (s *ExerciseServiceImpl) ModifyExercise(ctx context.Context, exercise Exercise) (Exercise, error) {
	if strings.TrimSpace(exercise.ExerciseName) == "" {
		return Exercise{}, ErrExerciseNameRequired
	}
	return s.repo.ModifyExercise(ctx, exercise)
}
