package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/config"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/database"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/server"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/user"
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

	exerciseRepo, err := exercise.NewPostgresExerciseRepository(ctx, pool)
	if err != nil {
		log.Fatalf("Failed to create exercise repository: %v", err)
	}
	exerciseService := exercise.NewExerciseService(exerciseRepo)
	exerciseHandler := exercise.NewExerciseHandler(exerciseService)

	userRepo := user.NewPostgresUserRepository(pool)
	sessionRepo := auth.NewInMemorySessionRepository(cfg.Session.SessionDuration)
	authService := auth.NewAuthService(userRepo, sessionRepo)
	authHandler := auth.NewAuthHandler(authService)
	authMiddleware := auth.NewAuthMiddleware(authService)

	router := server.NewRouter(exerciseHandler, authHandler, authMiddleware)

	log.Println("Listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
