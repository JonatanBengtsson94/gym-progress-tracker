package exercise

import (
	"cmp"
	"context"
	"slices"
	"strings"
)

type ExerciseRepository interface {
	GetExercisesByUserId(context.Context, uint32) ([]Exercise, error)
	CreateExercise(context.Context, Exercise) (Exercise, error)
	ModifyExercise(context.Context, Exercise) (Exercise, error)
}

type ExerciseServiceImpl struct {
	repo ExerciseRepository
}

func NewExerciseService(repo ExerciseRepository) *ExerciseServiceImpl {
	return &ExerciseServiceImpl{repo: repo}
}

// GetExercises returns the user's own and the global exercises sorted by name, ignoring case.
func (s *ExerciseServiceImpl) GetExercises(ctx context.Context, userId uint32) ([]Exercise, error) {
	exercises, err := s.repo.GetExercisesByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}
	slices.SortFunc(exercises, func(a, b Exercise) int {
		return cmp.Or(
			strings.Compare(strings.ToLower(a.ExerciseName), strings.ToLower(b.ExerciseName)),
			cmp.Compare(a.ExerciseId, b.ExerciseId),
		)
	})
	return exercises, nil
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
