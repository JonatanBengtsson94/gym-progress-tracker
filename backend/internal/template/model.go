package template

import (
	"errors"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/set"
)

var ErrTemplateAlreadyExists = errors.New("template already exists")
var ErrTemplateNameRequired = errors.New("template name is required")

type Template struct {
	UserId       uint32
	TemplateId   uint32
	TemplateName string
}

// LatestWorkout is the most recently completed workout logged under a template.
type LatestWorkout struct {
	WorkoutId   uint32
	StartedAt   time.Time
	CompletedAt time.Time
	Sets        []set.Set
}

// TemplateWithLatestWorkout has a nil LatestWorkout when no workout has been
// logged under the template yet.
type TemplateWithLatestWorkout struct {
	Template
	LatestWorkout *LatestWorkout
}
