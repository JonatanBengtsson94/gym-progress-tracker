package workout

import (
	"context"
	"errors"
	"fmt"
	"math"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/database"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/set"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/template"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// maxInt4 is the largest value the int4 column backing weights can hold.
// Weights beyond it are rejected before reaching the driver, which would
// otherwise fail to encode them and surface as an internal error.
const maxInt4 = math.MaxInt32

type PostgresWorkoutRepository struct {
	db *pgxpool.Pool
}

func NewPostgresWorkoutRepository(db *pgxpool.Pool) *PostgresWorkoutRepository {
	return &PostgresWorkoutRepository{db: db}
}

func (r *PostgresWorkoutRepository) GetWorkoutByUserIdAndWorkoutId(ctx context.Context, userId uint32, workoutId uuid.UUID) (Workout, error) {
	query := `SELECT w.started_at, w.completed_at, t.template_id, t.template_name, s.reps, s.weight_grams, e.exercise_id, e.exercise_name
		FROM workouts AS w
		JOIN templates AS t ON w.template_id = t.template_id
		JOIN sets AS s ON w.workout_id = s.workout_id
		JOIN exercises AS e ON s.exercise_id = e.exercise_id
		WHERE w.workout_id = $1 AND t.user_id = $2
		ORDER BY s.set_id`

	rows, err := r.db.Query(ctx, query, workoutId, userId)
	if err != nil {
		return Workout{}, fmt.Errorf("Get workout failed: %w", err)
	}
	defer rows.Close()

	workout := Workout{WorkoutId: workoutId}
	for rows.Next() {
		var set set.Set
		if err := rows.Scan(
			&workout.StartedAt,
			&workout.CompletedAt,
			&workout.Template.TemplateId,
			&workout.Template.TemplateName,
			&set.Reps, &set.WeightGrams,
			&set.Exercise.ExerciseId, &set.Exercise.ExerciseName); err != nil {
			return Workout{}, fmt.Errorf("Scan sets failed: %w", err)
		}
		workout.Sets = append(workout.Sets, set)
	}
	if err := rows.Err(); err != nil {
		return Workout{}, fmt.Errorf("Row iteration failed: %w", err)
	}
	if len(workout.Sets) == 0 {
		return Workout{}, ErrWorkoutNotFound
	}

	return workout, nil
}

func (r *PostgresWorkoutRepository) GetWorkoutsByUserId(ctx context.Context, userId uint32) ([]Workout, error) {
	query := `SELECT w.workout_id, w.started_at, w.completed_at, t.template_id, t.template_name
		FROM workouts AS w
		JOIN templates AS t ON w.template_id = t.template_id
		WHERE t.user_id = $1
		ORDER BY w.completed_at DESC, w.workout_id DESC`

	return r.listWorkouts(ctx, query, userId)
}

// GetWorkoutsByUserIdAndTemplateId lists the workouts logged under one of the
// user's templates, newest first. A template that doesn't exist or belongs to
// another user is reported as ErrTemplateNotFound, so it can be told apart
// from a template that simply has no workouts yet.
func (r *PostgresWorkoutRepository) GetWorkoutsByUserIdAndTemplateId(ctx context.Context, userId uint32, templateId uuid.UUID) ([]Workout, error) {
	if _, err := findTemplate(ctx, r.db, userId, templateId); err != nil {
		return nil, err
	}

	query := `SELECT w.workout_id, w.started_at, w.completed_at, t.template_id, t.template_name
		FROM workouts AS w
		JOIN templates AS t ON w.template_id = t.template_id
		WHERE t.template_id = $1 AND t.user_id = $2
		ORDER BY w.completed_at DESC, w.workout_id DESC`

	return r.listWorkouts(ctx, query, templateId, userId)
}

// listWorkouts runs a query selecting workout_id, started_at, completed_at,
// template_id and template_name, returning the rows as workouts without sets. The result
// is never nil, so an empty list serializes as an empty array.
func (r *PostgresWorkoutRepository) listWorkouts(ctx context.Context, query string, args ...any) ([]Workout, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("Get workouts failed: %w", err)
	}
	defer rows.Close()

	workouts := make([]Workout, 0)
	for rows.Next() {
		var workout Workout
		if err := rows.Scan(
			&workout.WorkoutId,
			&workout.StartedAt,
			&workout.CompletedAt,
			&workout.Template.TemplateId,
			&workout.Template.TemplateName,
		); err != nil {
			return nil, fmt.Errorf("Scan workouts failed: %w", err)
		}
		workouts = append(workouts, workout)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Rows iteration failed: %w", err)
	}

	return workouts, nil
}

// CreateWorkout stores workout under its own id, unless the user already has a
// workout with that id. Then nothing is stored and that workout is returned
// instead, with created false, so a retried create gets back what the first
// one stored. An id taken by another user's workout is ErrWorkoutIdTaken.
func (r *PostgresWorkoutRepository) CreateWorkout(ctx context.Context, userId uint32, workout Workout) (Workout, bool, error) {
	stored, found, err := r.findOwnWorkout(ctx, userId, workout.WorkoutId)
	if err != nil || found {
		return stored, false, err
	}

	created, err := r.insertWorkout(ctx, userId, workout)
	if errors.Is(err, ErrWorkoutIdTaken) {
		// A retry running alongside this create can store the workout first.
		stored, found, findErr := r.findOwnWorkout(ctx, userId, workout.WorkoutId)
		if findErr != nil || found {
			return stored, false, findErr
		}
	}
	if err != nil {
		return Workout{}, false, err
	}

	return created, true, nil
}

func (r *PostgresWorkoutRepository) findOwnWorkout(ctx context.Context, userId uint32, workoutId uuid.UUID) (Workout, bool, error) {
	workout, err := r.GetWorkoutByUserIdAndWorkoutId(ctx, userId, workoutId)
	if errors.Is(err, ErrWorkoutNotFound) {
		return Workout{}, false, nil
	}
	if err != nil {
		return Workout{}, false, err
	}
	return workout, true, nil
}

// insertWorkout stores workout and its sets in one transaction, creating its
// template first when Template.TemplateId is unset.
func (r *PostgresWorkoutRepository) insertWorkout(ctx context.Context, userId uint32, workout Workout) (Workout, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Workout{}, fmt.Errorf("Begin transaction failed: %w", err)
	}
	defer tx.Rollback(ctx)

	if workout.Template.TemplateId == uuid.Nil() {
		workout.Template, err = createTemplate(ctx, tx, userId, workout.Template.TemplateName)
	} else {
		workout.Template, err = findTemplate(ctx, tx, userId, workout.Template.TemplateId)
	}
	if err != nil {
		return Workout{}, err
	}

	query := `
		INSERT INTO workouts (workout_id, template_id, started_at, completed_at)
		VALUES ($1, $2, $3, $4)
	`
	if _, err := tx.Exec(ctx, query, workout.WorkoutId, workout.Template.TemplateId, workout.StartedAt, workout.CompletedAt); err != nil {
		if database.IsUniqueViolation(err) {
			return Workout{}, ErrWorkoutIdTaken
		}
		return Workout{}, fmt.Errorf("Create workout failed: %w", err)
	}

	workout.Sets, err = insertSets(ctx, tx, userId, workout.WorkoutId, workout.Sets)
	if err != nil {
		return Workout{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Workout{}, fmt.Errorf("Commit transaction failed: %w", err)
	}

	return workout, nil
}

// ModifyWorkout replaces the start and completion times and all sets of an existing
// workout in one transaction, so a failure part-way leaves the old sets in
// place. The workout's template is left as it is. Workouts belonging to other
// users are reported as ErrWorkoutNotFound.
func (r *PostgresWorkoutRepository) ModifyWorkout(ctx context.Context, userId uint32, workout Workout) (Workout, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Workout{}, fmt.Errorf("Begin transaction failed: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		UPDATE workouts AS w
		SET started_at = $1, completed_at = $2, updated_at = CURRENT_TIMESTAMP
		FROM templates AS t
		WHERE w.workout_id = $3 AND w.template_id = t.template_id AND t.user_id = $4
		RETURNING t.template_id, t.template_name
	`
	err = tx.QueryRow(ctx, query, workout.StartedAt, workout.CompletedAt, workout.WorkoutId, userId).Scan(&workout.Template.TemplateId, &workout.Template.TemplateName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Workout{}, ErrWorkoutNotFound
		}
		return Workout{}, fmt.Errorf("Modify workout failed: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM sets WHERE workout_id = $1`, workout.WorkoutId); err != nil {
		return Workout{}, fmt.Errorf("Delete sets failed: %w", err)
	}

	workout.Sets, err = insertSets(ctx, tx, userId, workout.WorkoutId, workout.Sets)
	if err != nil {
		return Workout{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Workout{}, fmt.Errorf("Commit transaction failed: %w", err)
	}

	return workout, nil
}

func createTemplate(ctx context.Context, tx pgx.Tx, userId uint32, templateName string) (template.Template, error) {
	query := `
		INSERT INTO templates (user_id, template_name)
		VALUES ($1, $2)
		RETURNING template_id
	`
	created := template.Template{UserId: userId, TemplateName: templateName}
	if err := tx.QueryRow(ctx, query, userId, templateName).Scan(&created.TemplateId); err != nil {
		if database.IsUniqueViolation(err) {
			return template.Template{}, template.ErrTemplateAlreadyExists
		}
		return template.Template{}, fmt.Errorf("Create template failed: %w", err)
	}

	return created, nil
}

// queryRower is satisfied by both the pool and a transaction, so lookups can
// run either standalone or as part of a larger write.
type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func findTemplate(ctx context.Context, db queryRower, userId uint32, templateId uuid.UUID) (template.Template, error) {
	query := `
		SELECT template_name
		FROM templates
		WHERE template_id = $1 AND user_id = $2
	`
	found := template.Template{UserId: userId, TemplateId: templateId}
	if err := db.QueryRow(ctx, query, templateId, userId).Scan(&found.TemplateName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return template.Template{}, ErrTemplateNotFound
		}
		return template.Template{}, fmt.Errorf("Get template failed: %w", err)
	}

	return found, nil
}

// insertSets stores sets against workoutId, returning them with their
// exercise names resolved. Sets referencing an exercise that is neither
// global nor owned by userId are rejected with ErrExerciseNotFound.
func insertSets(ctx context.Context, tx pgx.Tx, userId uint32, workoutId uuid.UUID, sets []set.Set) ([]set.Set, error) {
	exerciseIds := make([]uuid.UUID, len(sets))
	for i, s := range sets {
		if s.WeightGrams > maxInt4 {
			return nil, ErrWeightGramsOutOfRange
		}
		exerciseIds[i] = s.Exercise.ExerciseId
	}

	names, err := exerciseNames(ctx, tx, userId, exerciseIds)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO sets (exercise_id, workout_id, reps, weight_grams)
		VALUES ($1, $2, $3, $4)
	`
	inserted := make([]set.Set, len(sets))
	for i, s := range sets {
		name, ok := names[s.Exercise.ExerciseId]
		if !ok {
			return nil, ErrExerciseNotFound
		}

		if _, err := tx.Exec(ctx, query, s.Exercise.ExerciseId, workoutId, s.Reps, s.WeightGrams); err != nil {
			return nil, fmt.Errorf("Create set failed: %w", err)
		}

		s.WorkoutId = workoutId
		s.Exercise.ExerciseName = name
		inserted[i] = s
	}

	return inserted, nil
}

func exerciseNames(ctx context.Context, tx pgx.Tx, userId uint32, exerciseIds []uuid.UUID) (map[uuid.UUID]string, error) {
	query := `
		SELECT exercise_id, exercise_name
		FROM exercises
		WHERE exercise_id = ANY($1) AND (user_id IS NULL OR user_id = $2)
	`
	rows, err := tx.Query(ctx, query, exerciseIds, userId)
	if err != nil {
		return nil, fmt.Errorf("Get exercises failed: %w", err)
	}
	defer rows.Close()

	names := make(map[uuid.UUID]string, len(exerciseIds))
	for rows.Next() {
		var exerciseId uuid.UUID
		var exerciseName string
		if err := rows.Scan(&exerciseId, &exerciseName); err != nil {
			return nil, fmt.Errorf("Scan exercise failed: %w", err)
		}
		names[exerciseId] = exerciseName
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Row iteration failed: %w", err)
	}

	return names, nil
}
