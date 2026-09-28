package workout

import (
	"errors"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/set"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/template"
)

var ErrWorkoutNotFound = errors.New("workout not found")
var ErrTemplateNotFound = errors.New("template not found")
var ErrExerciseNotFound = errors.New("exercise not found")
var ErrSetsRequired = errors.New("workout must contain at least one set")
var ErrRepsRequired = errors.New("reps must be greater than zero")
var ErrWeightGramsOutOfRange = errors.New("weight_grams is out of range")
var ErrStartedAtRequired = errors.New("started_at is required")
var ErrStartedAfterCompleted = errors.New("started_at must not be after completed_at")

// A Workout with Template.TemplateId unset is stored under a newly created
// template named Template.TemplateName.
type Workout struct {
	WorkoutId   uint32
	StartedAt   time.Time
	CompletedAt time.Time
	Template    template.Template
	Sets        []set.Set
}
