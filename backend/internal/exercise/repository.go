package exercise

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresExerciseRepository struct {
	db              *pgxpool.Pool
	globalExercises []Exercise
}

func NewPostgresExerciseRepository(ctx context.Context, db *pgxpool.Pool) (*PostgresExerciseRepository, error) {
	globalExercises, err := getGlobalExercisesCache(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("Could not create exercise repository: %w", err)
	}
	return &PostgresExerciseRepository{db: db, globalExercises: globalExercises}, nil
}

// GetGlobalExercises returns the exercises every user can see, read once when the repository was created.
func (r *PostgresExerciseRepository) GetGlobalExercises(ctx context.Context) ([]Exercise, error) {
	return slices.Clone(r.globalExercises), nil
}

// GetExercisesByUserId returns the user's own exercises, without the global ones.
func (r *PostgresExerciseRepository) GetExercisesByUserId(ctx context.Context, userId uint32) ([]Exercise, error) {
	query := `
		SELECT exercise_id, exercise_name
		FROM exercises
		WHERE user_id = $1
	`
	rows, err := r.db.Query(ctx, query, userId)
	if err != nil {
		return nil, fmt.Errorf("Get exercises failed: %w", err)
	}
	defer rows.Close()

	var exercises []Exercise

	for rows.Next() {
		exercise := Exercise{UserId: userId}
		if err := rows.Scan(&exercise.ExerciseId, &exercise.ExerciseName); err != nil {
			return nil, fmt.Errorf("Scan exercise failed: %w", err)
		}
		exercises = append(exercises, exercise)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Rows iteration failed: %w", err)
	}

	return exercises, nil
}

// CreateExercise stores exercise under its own id, unless the user can already see an exercise
// with that id, or one of their own or a global exercise has the same name, ignoring case. Then
// nothing is stored and that exercise is returned instead, with created false: a retried create
// gets back what the first one stored, and a client that created the exercise offline learns the
// one it should use. An id taken by another user's exercise is ErrExerciseIdTaken.
func (r *PostgresExerciseRepository) CreateExercise(ctx context.Context, exercise Exercise) (Exercise, bool, error) {
	// No unique index spans the user's and the global exercises, so the insert checks global names itself.
	query := `
		INSERT INTO exercises (exercise_id, user_id, exercise_name)
		SELECT $1::uuid, $2::integer, $3::text
		WHERE NOT EXISTS (
			SELECT 1 FROM exercises WHERE user_id IS NULL AND lower(exercise_name) = lower($3::text)
		)
		ON CONFLICT DO NOTHING
	`
	tag, err := r.db.Exec(ctx, query, exercise.ExerciseId, exercise.UserId, exercise.ExerciseName)
	if err != nil {
		return Exercise{}, false, fmt.Errorf("Create exercise failed: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return exercise, true, nil
	}

	existing, err := r.findMatchingExercise(ctx, exercise)
	if err != nil {
		return Exercise{}, false, err
	}
	return existing, false, nil
}

// findMatchingExercise returns the exercise that kept exercise from being created: the one with
// its id if the user can see it, or else the user's own or, failing that, a global exercise with
// its name.
func (r *PostgresExerciseRepository) findMatchingExercise(ctx context.Context, exercise Exercise) (Exercise, error) {
	query := `
		SELECT exercise_id, COALESCE(user_id, 0), exercise_name
		FROM exercises
		WHERE exercise_id = $1
			OR (lower(exercise_name) = lower($3) AND (user_id = $2 OR user_id IS NULL))
		ORDER BY exercise_id = $1 DESC, user_id NULLS LAST
	`
	rows, err := r.db.Query(ctx, query, exercise.ExerciseId, exercise.UserId, exercise.ExerciseName)
	if err != nil {
		return Exercise{}, fmt.Errorf("Find matching exercise failed: %w", err)
	}
	defer rows.Close()

	idTaken := false
	for rows.Next() {
		var match Exercise
		if err := rows.Scan(&match.ExerciseId, &match.UserId, &match.ExerciseName); err != nil {
			return Exercise{}, fmt.Errorf("Scan exercise failed: %w", err)
		}
		// Only the row matching by id can belong to another user.
		if match.UserId != 0 && match.UserId != exercise.UserId {
			idTaken = true
			continue
		}
		return match, nil
	}
	if err := rows.Err(); err != nil {
		return Exercise{}, fmt.Errorf("Rows iteration failed: %w", err)
	}

	if idTaken {
		return Exercise{}, ErrExerciseIdTaken
	}
	return Exercise{}, fmt.Errorf("Create exercise failed: no exercise matches %s", exercise.ExerciseId)
}

func (r *PostgresExerciseRepository) ModifyExercise(ctx context.Context, exercise Exercise) (Exercise, error) {
	if r.isGlobalExerciseId(exercise.ExerciseId) {
		return Exercise{}, ErrCannotModifyGlobalExercise
	}

	if r.isGlobalNameCollision(exercise.ExerciseName) {
		return Exercise{}, ErrExerciseAlreadyExists
	}

	query := `
		UPDATE exercises
		SET exercise_name = $1
		WHERE exercise_id = $2 AND user_id = $3
		RETURNING exercise_id
	`
	err := r.db.QueryRow(ctx, query, exercise.ExerciseName, exercise.ExerciseId, exercise.UserId).Scan(&exercise.ExerciseId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Exercise{}, ErrExerciseNotFound
		}
		if database.IsUniqueViolation(err) {
			return Exercise{}, ErrExerciseAlreadyExists
		}
		return Exercise{}, fmt.Errorf("Modify exercise failed: %w", err)
	}

	return exercise, nil
}

func getGlobalExercisesCache(ctx context.Context, db *pgxpool.Pool) ([]Exercise, error) {
	query := `
		SELECT exercise_id, exercise_name
		FROM exercises
		WHERE user_id IS NULL
	`
	rows, err := db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("Get global exercises failed: %w", err)
	}
	defer rows.Close()

	globalExercises := make([]Exercise, 0, 8)

	for rows.Next() {
		var exercise Exercise
		if err := rows.Scan(&exercise.ExerciseId, &exercise.ExerciseName); err != nil {
			return nil, fmt.Errorf("Scan exercise failed: %w", err)
		}
		globalExercises = append(globalExercises, exercise)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Rows interation failed: %w", err)
	}

	return globalExercises, nil
}

func (r *PostgresExerciseRepository) isGlobalNameCollision(name string) bool {
	for _, global := range r.globalExercises {
		if strings.EqualFold(global.ExerciseName, name) {
			return true
		}
	}
	return false
}

func (r *PostgresExerciseRepository) isGlobalExerciseId(exerciseId uuid.UUID) bool {
	for _, global := range r.globalExercises {
		if global.ExerciseId == exerciseId {
			return true
		}
	}
	return false
}
