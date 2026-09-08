package main

import (
	"context"
	"fmt"
	"log"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		cfg.DB.Username,
		cfg.DB.Password,
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.DBName,
	)

	pool, err := pgxpool.New(context.TODO(), dsn)
	if err != nil {
		log.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.TODO()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database")

	rows, err := pool.Query(context.TODO(), "SELECT exercise_name FROM exercises")
	if err != nil {
		log.Fatalf("Failed to query exercises: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var exerciseName string
		if err := rows.Scan(&exerciseName); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		fmt.Println(exerciseName)
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Row iteration error: %v", err)
	}
}
