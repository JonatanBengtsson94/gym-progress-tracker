package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/server"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/user"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	migrationFiles, err := filepath.Glob(filepath.Join("..", "..", "..", "database", "migrations", "*.sql"))
	if err != nil {
		log.Fatalf("failed to glob migrations: %v", err)
	}
	sort.Strings(migrationFiles)

	if len(migrationFiles) == 0 {
		log.Fatal("no migration files found")
	}

	seedFile := filepath.Join("testdata", "seed.sql")
	initScripts := append(migrationFiles, seedFile)

	pgContainer, err := postgres.Run(ctx,
		"docker.io/postgres:latest",
		postgres.WithDatabase("testDB"),
		postgres.WithUsername("testUser"),
		postgres.WithPassword("testPass"),
		postgres.WithInitScripts(initScripts...),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("failed to start postgres testcontainer: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}

	testPool, err = pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("failed to connect pool: %v", err)
	}

	if err := testPool.Ping(ctx); err != nil {
		testPool.Close()
		_ = pgContainer.Terminate(ctx)
		log.Fatalf("failed to ping postgres: %v", err)
	}

	code := m.Run()

	testPool.Close()
	_ = pgContainer.Terminate(ctx)
	os.Exit(code)
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	ctx := t.Context()

	exerciseRepo, err := exercise.NewExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("failed to create exercise repository: %v", err)
	}
	exerciseService := exercise.NewExerciseService(exerciseRepo)
	exerciseHandler := exercise.NewExerciseHandler(exerciseService)

	userRepo := user.NewUserRepository(testPool)
	sessionRepo := auth.NewSessionRepository(time.Hour)
	authService := auth.NewAuthService(userRepo, sessionRepo)
	authHandler := auth.NewAuthHandler(authService)
	authMiddleware := auth.NewAuthMiddleware(authService)

	return server.NewRouter(exerciseHandler, authHandler, authMiddleware)
}

func TestIntegration_LoginAndAccessProtectedRoute(t *testing.T) {
	router := newTestRouter(t)

	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"username":"alice","password":"secret"}`))
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)

	loginRes := loginRec.Result()
	defer loginRes.Body.Close()

	if loginRes.StatusCode != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", loginRes.StatusCode)
	}

	var loginResponse auth.LoginResponse
	if err := json.NewDecoder(loginRes.Body).Decode(&loginResponse); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	exercisesReq := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	exercisesReq.Header.Set("Authorization", "Bearer "+loginResponse.SessionId)
	exercisesRec := httptest.NewRecorder()
	router.ServeHTTP(exercisesRec, exercisesReq)

	exercisesRes := exercisesRec.Result()
	defer exercisesRes.Body.Close()

	if exercisesRes.StatusCode != http.StatusOK {
		t.Fatalf("expected exercises status 200, got %d", exercisesRes.StatusCode)
	}

	var exercises []exercise.ExerciseResponse
	if err := json.NewDecoder(exercisesRes.Body).Decode(&exercises); err != nil {
		t.Fatalf("failed to decode exercises response: %v", err)
	}

	found := false
	for _, e := range exercises {
		if e.ExerciseName == "Custom Test Exercise" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find alice's custom exercise, got %+v", exercises)
	}
}

func TestIntegration_AccessProtectedRoute_NoToken(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_Login_WrongPassword(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"username":"alice","password":"wrong"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Result().StatusCode)
	}
}

func mustLogin(t *testing.T, router http.Handler, username, password string) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("login failed: expected status 200, got %d", res.StatusCode)
	}

	var loginResponse auth.LoginResponse
	if err := json.NewDecoder(res.Body).Decode(&loginResponse); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	return loginResponse.SessionId
}

func TestIntegration_CreateExerciseAndSeeItInList(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	createReq := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(`{"exercise_name":"Lunge"}`))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	createRes := createRec.Result()
	defer createRes.Body.Close()

	if createRes.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d", createRes.StatusCode)
	}

	var created exercise.ExerciseResponse
	if err := json.NewDecoder(createRes.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if created.ExerciseName != "Lunge" || created.ExerciseId == 0 {
		t.Errorf("unexpected create response: %+v", created)
	}

	exercisesReq := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	exercisesReq.Header.Set("Authorization", "Bearer "+token)
	exercisesRec := httptest.NewRecorder()
	router.ServeHTTP(exercisesRec, exercisesReq)

	exercisesRes := exercisesRec.Result()
	defer exercisesRes.Body.Close()

	var exercises []exercise.ExerciseResponse
	if err := json.NewDecoder(exercisesRes.Body).Decode(&exercises); err != nil {
		t.Fatalf("failed to decode exercises response: %v", err)
	}

	found := false
	for _, e := range exercises {
		if e.ExerciseName == "Lunge" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected newly created exercise to appear in GET /exercises, got %+v", exercises)
	}
}

func TestIntegration_CreateExercise_NoToken(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(`{"exercise_name":"Lunge"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_CreateExercise_DuplicateName(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	req := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(`{"exercise_name":"Custom Test Exercise"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_CreateExercise_DuplicatesGlobalExercise(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	// "Bench Press" is a seeded global exercise; alice should not be able
	// to create a custom exercise with the same name.
	req := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(`{"exercise_name":"Bench Press"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rec.Result().StatusCode)
	}
}
