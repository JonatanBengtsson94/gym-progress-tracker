package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepository struct {
	db *pgxpool.Pool
}

func NewPostgresUserRepository(db *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) GetUserByUsername(ctx context.Context, username string) (User, error) {
	var user User
	query := `
		SELECT user_id, username, password
		FROM users
		WHERE username = $1
	`
	err := r.db.QueryRow(ctx, query, username).Scan(&user.UserId, &user.UserName, &user.Password)
	if errors.Is(err, pgx.ErrNoRows) {
		return user, ErrUserNotFound
	}
	if err != nil {
		return user, fmt.Errorf("GetUserByUsername failed: %w", err)
	}

	return user, nil
}

func (r *PostgresUserRepository) GetUserByUserId(ctx context.Context, userId uint32) (User, error) {
	var user User
	user.UserId = userId
	query := `
		SELECT username, first_name, last_name
		FROM users
		WHERE user_id = $1
	`
	err := r.db.QueryRow(ctx, query, userId).Scan(&user.UserName, &user.FirstName, &user.LastName)
	if errors.Is(err, pgx.ErrNoRows) {
		return user, ErrUserNotFound
	}
	if err != nil {
		return user, fmt.Errorf("GetUserByUserId failed: %w", err)
	}

	return user, nil
}
