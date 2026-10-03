package testutil

import (
	"context"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Id returns the UUID whose last group is n in decimal, so Id(101) is
// 00000000-0000-0000-0000-000000000101. Seed files write their fixed ids in
// this form, which lets tests refer to a seeded row by its number.
func Id(n int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", n))
}

// GlobalExerciseId returns the id of the global exercise named name. The
// migrations generate these ids, so tests look them up instead.
func GlobalExerciseId(ctx context.Context, pool *pgxpool.Pool, name string) (uuid.UUID, error) {
	var exerciseId uuid.UUID
	query := `SELECT exercise_id FROM exercises WHERE user_id IS NULL AND exercise_name = $1`
	if err := pool.QueryRow(ctx, query, name).Scan(&exerciseId); err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to look up global exercise %q: %w", name, err)
	}
	return exerciseId, nil
}
