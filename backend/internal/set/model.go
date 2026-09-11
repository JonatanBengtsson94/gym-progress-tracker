package set

import "github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"

type Set struct {
	Exercise    exercise.Exercise
	Reps        uint8
	WeightGrams uint32
}
