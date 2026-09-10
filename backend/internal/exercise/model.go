package exercise

import "errors"

var ErrExerciseAlreadyExists = errors.New("exercise already exists")

type Exercise struct {
	ExerciseId   uint32
	UserId       uint32
	ExerciseName string
}
