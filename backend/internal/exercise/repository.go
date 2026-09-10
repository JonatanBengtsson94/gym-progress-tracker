package exercise

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// uniqueViolationCode is the Postgres error code for a unique constraint
// violation (23505), returned e.g. when a user already has an exercise
// with the same name (see uq_exercises_user_name).
const uniqueViolationCode = "23505"

type PostgresExerciseRepository struct {
	db              *pgxpool.Pool
	globalExercises []Exercise
}

func NewExerciseRepository(ctx context.Context, db *pgxpool.Pool) (*PostgresExerciseRepository, error) {
	globalExercises, err := getGlobalExercisesCache(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("Could not create exercise repository: %w", err)
	}
	return &PostgresExerciseRepository{db: db, globalExercises: globalExercises}, nil
}

func (r *PostgresExerciseRepository) GetExercises(ctx context.Context, userId uint32) ([]Exercise, error) {
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

	exercises := make([]Exercise, 0, len(r.globalExercises))

	for rows.Next() {
		var exercise Exercise
		if err := rows.Scan(&exercise.ExerciseId, &exercise.ExerciseName); err != nil {
			return nil, fmt.Errorf("Scan exercise failed: %w", err)
		}
		exercises = append(exercises, exercise)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("Rows interation failed: %w", err)
	}

	allExercises := append(exercises, r.globalExercises...)

	return allExercises, nil
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

func (r *PostgresExerciseRepository) CreateExercise(ctx context.Context, exercise Exercise) (Exercise, error) {
	for _, global := range r.globalExercises {
		if strings.EqualFold(global.ExerciseName, exercise.ExerciseName) {
			return Exercise{}, ErrExerciseAlreadyExists
		}
	}

	query := `
		INSERT INTO exercises (user_id, exercise_name)
		VALUES ($1, $2)
		RETURNING exercise_id
	`
	err := r.db.QueryRow(ctx, query, exercise.UserId, exercise.ExerciseName).Scan(&exercise.ExerciseId)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return Exercise{}, ErrExerciseAlreadyExists
		}
		return Exercise{}, fmt.Errorf("Create exercise failed: %w", err)
	}

	return exercise, nil
}
