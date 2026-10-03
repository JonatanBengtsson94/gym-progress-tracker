package template

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/set"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresTemplateRepository struct {
	db *pgxpool.Pool
}

func NewPostgresTemplateRepository(db *pgxpool.Pool) *PostgresTemplateRepository {
	return &PostgresTemplateRepository{db: db}
}

// GetTemplatesByUserId lists the user's templates, each with its latest
// workout (by completed_at, then workout_id) and that workout's sets in the
// order they were logged. Templates are ordered by their latest workout, most
// recent first; templates that have never been used come last, ordered by
// name. The result is never nil, so an empty list serializes as an empty array.
func (r *PostgresTemplateRepository) GetTemplatesByUserId(ctx context.Context, userId uint32) ([]TemplateWithLatestWorkout, error) {
	query := `
		SELECT t.template_id, t.template_name,
			lw.workout_id, lw.started_at, lw.completed_at,
			e.exercise_id, e.exercise_name, s.reps, s.weight_grams
		FROM templates AS t
		LEFT JOIN LATERAL (
			SELECT w.workout_id, w.started_at, w.completed_at
			FROM workouts AS w
			WHERE w.template_id = t.template_id
			ORDER BY w.completed_at DESC, w.workout_id DESC
			LIMIT 1
		) AS lw ON true
		LEFT JOIN sets AS s ON s.workout_id = lw.workout_id
		LEFT JOIN exercises AS e ON e.exercise_id = s.exercise_id
		WHERE t.user_id = $1
		ORDER BY lw.completed_at DESC NULLS LAST, lw.workout_id DESC,
			lower(t.template_name), t.template_id, s.set_id
	`
	rows, err := r.db.Query(ctx, query, userId)
	if err != nil {
		return nil, fmt.Errorf("Get templates failed: %w", err)
	}
	defer rows.Close()

	// Each row is one set of a template's latest workout, or a single row of
	// NULLs past template_name when there is no workout (or it has no sets).
	templates := make([]TemplateWithLatestWorkout, 0)
	for rows.Next() {
		var (
			templateId             uuid.UUID
			templateName           string
			workoutId, exerciseId  *uuid.UUID
			startedAt, completedAt *time.Time
			exerciseName           *string
			reps                   *uint8
			weightGrams            *uint32
		)
		if err := rows.Scan(
			&templateId, &templateName,
			&workoutId, &startedAt, &completedAt,
			&exerciseId, &exerciseName, &reps, &weightGrams,
		); err != nil {
			return nil, fmt.Errorf("Scan templates failed: %w", err)
		}

		if len(templates) == 0 || templates[len(templates)-1].TemplateId != templateId {
			tmpl := TemplateWithLatestWorkout{
				Template: Template{UserId: userId, TemplateId: templateId, TemplateName: templateName},
			}
			if workoutId != nil {
				tmpl.LatestWorkout = &LatestWorkout{
					WorkoutId:   *workoutId,
					StartedAt:   *startedAt,
					CompletedAt: *completedAt,
				}
			}
			templates = append(templates, tmpl)
		}

		if exerciseId != nil {
			latest := templates[len(templates)-1].LatestWorkout
			latest.Sets = append(latest.Sets, set.Set{
				Exercise:    exercise.Exercise{ExerciseId: *exerciseId, ExerciseName: *exerciseName},
				Reps:        *reps,
				WeightGrams: *weightGrams,
				WorkoutId:   latest.WorkoutId,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Get templates failed: %w", err)
	}

	return templates, nil
}

// CreateTemplate stores template under its own id, unless the user already has a template with
// that id or with the same name, ignoring case. Then nothing is stored and that template is
// returned instead, with created false: a retried create gets back what the first one stored, and
// a client that created the template offline learns the one it should use. An id taken by another
// user's template is ErrTemplateIdTaken.
func (r *PostgresTemplateRepository) CreateTemplate(ctx context.Context, template Template) (Template, bool, error) {
	query := `
		INSERT INTO templates (template_id, user_id, template_name)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`
	tag, err := r.db.Exec(ctx, query, template.TemplateId, template.UserId, template.TemplateName)
	if err != nil {
		return Template{}, false, fmt.Errorf("Create template failed: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return template, true, nil
	}

	existing, err := r.findMatchingTemplate(ctx, template)
	if err != nil {
		return Template{}, false, err
	}
	return existing, false, nil
}

// findMatchingTemplate returns the template that kept template from being created: the user's
// own with its id, or else the user's own with its name.
func (r *PostgresTemplateRepository) findMatchingTemplate(ctx context.Context, template Template) (Template, error) {
	query := `
		SELECT template_id, user_id, template_name
		FROM templates
		WHERE template_id = $1 OR (user_id = $2 AND lower(template_name) = lower($3))
		ORDER BY template_id = $1 DESC
	`
	rows, err := r.db.Query(ctx, query, template.TemplateId, template.UserId, template.TemplateName)
	if err != nil {
		return Template{}, fmt.Errorf("Find matching template failed: %w", err)
	}
	defer rows.Close()

	idTaken := false
	for rows.Next() {
		var match Template
		if err := rows.Scan(&match.TemplateId, &match.UserId, &match.TemplateName); err != nil {
			return Template{}, fmt.Errorf("Scan template failed: %w", err)
		}
		// Only the row matching by id can belong to another user.
		if match.UserId != template.UserId {
			idTaken = true
			continue
		}
		return match, nil
	}
	if err := rows.Err(); err != nil {
		return Template{}, fmt.Errorf("Rows iteration failed: %w", err)
	}

	if idTaken {
		return Template{}, ErrTemplateIdTaken
	}
	return Template{}, fmt.Errorf("Create template failed: no template matches %s", template.TemplateId)
}
