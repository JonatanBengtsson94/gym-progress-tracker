package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/config"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/database"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/server"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/user"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/workout"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DB)
	if err != nil {
		slog.Error("Failed to setup database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("Connected to database")

	exerciseRepo, err := exercise.NewPostgresExerciseRepository(ctx, pool)
	if err != nil {
		slog.Error("Failed to create exercise repository", "error", err)
		os.Exit(1)
	}
	exerciseService := exercise.NewExerciseService(exerciseRepo)
	exerciseHandler := exercise.NewExerciseHandler(exerciseService)

	templateRepo := template.NewPostgresTemplateRepository(pool)
	templateService := template.NewTemplateService(templateRepo)
	templateHandler := template.NewTemplateHandler(templateService)

	workoutRepo := workout.NewPostgresWorkoutRepository(pool)
	workoutService := workout.NewWorkoutService(workoutRepo)
	workoutHandler := workout.NewWorkoutHandler(workoutService)

	userRepo := user.NewPostgresUserRepository(pool)
	sessionRepo := auth.NewInMemorySessionRepository(cfg.Session.SessionDuration)
	authService := auth.NewAuthService(userRepo, sessionRepo)
	authHandler := auth.NewAuthHandler(authService)
	authMiddleware := auth.NewAuthMiddleware(authService)

	router := server.NewRouter(exerciseHandler, workoutHandler, templateHandler, authHandler, authMiddleware)

	slog.Info("Listening on port 8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		slog.Error("Server stopped", "error", err)
		os.Exit(1)
	}
}
