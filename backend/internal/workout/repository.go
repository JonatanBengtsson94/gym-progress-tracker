package workout

import (
	"context"
	"fmt"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/set"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresWorkoutRepository struct {
	db *pgxpool.Pool
}

func NewPostgresWorkoutRepository(db *pgxpool.Pool) *PostgresWorkoutRepository {
	return &PostgresWorkoutRepository{db: db}
}

func (r *PostgresWorkoutRepository) GetWorkoutByUserIdAndWorkoutId(ctx context.Context, userId uint32, workoutId uint32) (Workout, error) {
	query := `SELECT w.completed_at, t.template_name, s.reps, s.weight_grams, e.exercise_id, e.exercise_name
		FROM workouts AS w
		JOIN templates AS t ON w.template_id = t.template_id
		JOIN sets AS s ON w.workout_id = s.workout_id
		JOIN exercises AS e ON s.exercise_id = e.exercise_id
		WHERE w.workout_id = $1 AND t.user_id = $2`

	rows, err := r.db.Query(ctx, query, workoutId, userId)
	if err != nil {
		return Workout{}, fmt.Errorf("Get workout failed: %w", err)
	}
	defer rows.Close()

	workout := Workout{WorkoutId: workoutId}
	for rows.Next() {
		var set set.Set
		if err := rows.Scan(
			&workout.CompletedAt,
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
	// TODO:
	return []Workout{}, nil
}
