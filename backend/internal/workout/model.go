package workout

import (
	"errors"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/set"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/template"
)

var ErrWorkoutNotFound = errors.New("workout not found")

type Workout struct {
	WorkoutId   uint32
	CompletedAt time.Time
	Template    template.Template
	Sets        []set.Set
}
