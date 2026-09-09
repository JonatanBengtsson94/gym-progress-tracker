package exercise

type ExerciseGetter interface {
	GetExercises() ([]Exercise, error)
}

type ExerciseRepository interface {
	ExerciseGetter
}

type ExerciseServiceImpl struct {
	repo ExerciseRepository
}

func NewExerciseService(repo ExerciseRepository) *ExerciseServiceImpl {
	return &ExerciseServiceImpl{repo: repo}
}

func (s *ExerciseServiceImpl) GetExercises() ([]Exercise, error) {
	return s.repo.GetExercises()
}
