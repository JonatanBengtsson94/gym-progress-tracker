package set

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresSetRepository struct {
	db *pgxpool.Pool
}

func NewSetRepository(db *pgxpool.Pool) *PostgresSetRepository {
	return &PostgresSetRepository{db: db}
}

func (r *PostgresSetRepository) GetSetsByExerciseId(ctx context.Context, exerciseId uint32) ([]Set, error) {
	// TODO:
}

func (r *PostgresSetRepository) GetSetsByWorkoutId(ctx context.Context, workoutId uint32) ([]Set, error) {
	// TODO:
}

func (r *PostgresSetRepository) GetSetsByWorkoutIdAndExerciseId(ctx context.Context, workoutId uint32, exerciseId uint32) ([]Set, error) {
	// TODO:
}
