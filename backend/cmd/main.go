package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/config"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/database"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/server"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DB)
	if err != nil {
		log.Fatalf("Failed to setup database: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to database")

	exerciseRepo := exercise.NewExerciseRepository(pool)
	exerciseService := exercise.NewExerciseService(exerciseRepo)
	exerciseHandler := exercise.NewExerciseHandler(exerciseService)

	router := server.NewRouter(exerciseHandler)

	log.Println("Listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
