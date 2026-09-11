package set

import "context"

type SetRepository interface {
	GetSetsByExerciseId(context.Context, uint32) ([]Set, error)
	GetSetsByWorkoutId(context.Context, uint32) ([]Set, error)
}

type SetServiceImpl struct {
	repo SetRepository
}

func NewSetService(repo SetRepository) *SetServiceImpl {
	return &SetServiceImpl{repo: repo}
}
