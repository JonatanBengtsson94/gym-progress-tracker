package template

import (
	"errors"
	"time"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/set"
)

var ErrTemplateAlreadyExists = errors.New("template already exists")
var ErrTemplateNameRequired = errors.New("template name is required")
var ErrTemplateIdTaken = errors.New("template id belongs to another user's template")

type Template struct {
	UserId       uint32
	TemplateId   uuid.UUID
	TemplateName string
}

// LatestWorkout is the most recently completed workout logged under a template.
type LatestWorkout struct {
	WorkoutId   uuid.UUID
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
