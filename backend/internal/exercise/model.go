package exercise

import (
	"errors"
	"uuid"
)

var ErrExerciseAlreadyExists = errors.New("exercise already exists")
var ErrExerciseNotFound = errors.New("exercise not found")
var ErrExerciseNameRequired = errors.New("exercise_name is required")
var ErrCannotModifyGlobalExercise = errors.New("cannot modify a global exercise")
var ErrExerciseIdTaken = errors.New("exercise id belongs to another user's exercise")

// Exercise has UserId 0 when it is a global exercise, which every user can see.
type Exercise struct {
	ExerciseId   uuid.UUID
	UserId       uint32
	ExerciseName string
}
