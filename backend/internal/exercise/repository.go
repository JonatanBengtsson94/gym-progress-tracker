package exercise

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
