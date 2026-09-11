package exercise

import "errors"

var ErrExerciseAlreadyExists = errors.New("exercise already exists")
var ErrExerciseNotFound = errors.New("exercise not found")
var ErrExerciseNameRequired = errors.New("exercise_name is required")
var ErrCannotModifyGlobalExercise = errors.New("cannot modify a global exercise")

type Exercise struct {
	ExerciseId   uint32
	UserId       uint32
	ExerciseName string
}
