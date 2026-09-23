package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/database"
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
		SELECT user_id, username, password_hash
		FROM users
		WHERE username = $1
	`
	err := r.db.QueryRow(ctx, query, username).Scan(&user.UserId, &user.UserName, &user.PasswordHash)
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

func (r *PostgresUserRepository) CreateUser(ctx context.Context, user User) (User, error) {
	query := `
		INSERT INTO users (username, password_hash, first_name, last_name)
		VALUES ($1, $2, $3, $4)
		RETURNING user_id
	`
	err := r.db.QueryRow(ctx, query, user.UserName, user.PasswordHash, user.FirstName, user.LastName).Scan(&user.UserId)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return User{}, ErrUsernameTaken
		}
		return User{}, fmt.Errorf("Create user failed: %w", err)
	}

	return user, nil
}
