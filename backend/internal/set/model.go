package set

import (
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/exercise"
)

type Set struct {
	Exercise    exercise.Exercise
	Reps        uint8
	WeightGrams uint32
	WorkoutId   uuid.UUID
}
