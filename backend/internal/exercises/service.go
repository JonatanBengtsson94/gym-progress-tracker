package exercises

type ExerciseService struct {
	repo *ExerciseRepository
}

func NewExerciseService(repo *ExerciseRepository) *ExerciseService {
	return &ExerciseService{repo: repo}
}

func (s *ExerciseService) GetExercises() ([]Exercise, error) {
	return s.repo.GetExercises()
}
