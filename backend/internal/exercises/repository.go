package exercises

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ExerciseRepository struct {
	db *pgxpool.Pool
}

func NewExercisesRepository(db *pgxpool.Pool) *ExerciseRepository {
	return &ExerciseRepository{db: db}
}

func (r *ExerciseRepository) GetExercises() ([]Exercise, error) {
	rows, err := r.db.Query(context.TODO(), "SELECT exercise_id, exercise_name FROM exercises")
	if err != nil {
		return nil, fmt.Errorf("get exercises failed: %w", err)
	}
	defer rows.Close()

	var exercises []Exercise

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

	return exercises, nil
}
