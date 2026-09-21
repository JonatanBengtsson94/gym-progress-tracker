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
	"strings"
	"testing"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/server"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/testutil"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/user"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/workout"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	pool, teardown, err := testutil.StartPostgres(
		ctx,
		filepath.Join("..", "..", "..", "database", "migrations", "*.sql"),
		filepath.Join("testdata", "seed.sql"),
	)
	if err != nil {
		log.Fatalf("failed to set up test database: %v", err)
	}
	testPool = pool

	code := m.Run()

	teardown()
	os.Exit(code)
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	ctx := t.Context()

	exerciseRepo, err := exercise.NewPostgresExerciseRepository(ctx, testPool)
	if err != nil {
		t.Fatalf("failed to create exercise repository: %v", err)
	}
	exerciseService := exercise.NewExerciseService(exerciseRepo)
	exerciseHandler := exercise.NewExerciseHandler(exerciseService)

	workoutRepo := workout.NewPostgresWorkoutRepository(testPool)
	workoutService := workout.NewWorkoutService(workoutRepo)
	workoutHandler := workout.NewWorkoutHandler(workoutService)

	templateRepo := template.NewPostgresTemplateRepository(testPool)
	templateService := template.NewTemplateService(templateRepo)
	templateHandler := template.NewTemplateHandler(templateService)

	userRepo := user.NewPostgresUserRepository(testPool)
	userService := user.NewUserService(userRepo)
	userHandler := user.NewUserHandler(userService)
	sessionRepo := auth.NewInMemorySessionRepository(time.Hour)
	authService := auth.NewAuthService(userRepo, sessionRepo)
	authHandler := auth.NewAuthHandler(authService)
	authMiddleware := auth.NewAuthMiddleware(authService)

	return server.NewRouter(userHandler, exerciseHandler, workoutHandler, templateHandler, authHandler, authMiddleware)
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

func TestIntegration_LoginAndGetOwnUser(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	var got user.UserResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode user response: %v", err)
	}

	want := user.UserResponse{UserId: 1, UserName: "alice", FirstName: "Alice", LastName: "Anderson"}
	if got != want {
		t.Errorf("GET /users/me = %+v, want %+v", got, want)
	}
}

func TestIntegration_GetOwnUser_ScopedToSession(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "bob", "secret")

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	var got user.UserResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode user response: %v", err)
	}

	want := user.UserResponse{UserId: 2, UserName: "bob", FirstName: "Bob", LastName: "Brown"}
	if got != want {
		t.Errorf("GET /users/me with bob's session = %+v, want %+v", got, want)
	}
}

func TestIntegration_GetOwnUser_NoToken(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Result().StatusCode)
	}
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

func mustCreateExercise(t *testing.T, router http.Handler, token, name string) exercise.ExerciseResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/exercises", strings.NewReader(fmt.Sprintf(`{"exercise_name":%q}`, name)))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create exercise failed: expected status 201, got %d", res.StatusCode)
	}

	var created exercise.ExerciseResponse
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	return created
}

func TestIntegration_ModifyExerciseAndSeeUpdatedInList(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	created := mustCreateExercise(t, router, token, "Row")

	modifyReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/exercises/%d", created.ExerciseId), strings.NewReader(`{"exercise_name":"Bent Over Row"}`))
	modifyReq.Header.Set("Authorization", "Bearer "+token)
	modifyRec := httptest.NewRecorder()
	router.ServeHTTP(modifyRec, modifyReq)

	modifyRes := modifyRec.Result()
	defer modifyRes.Body.Close()

	if modifyRes.StatusCode != http.StatusOK {
		t.Fatalf("expected modify status 200, got %d", modifyRes.StatusCode)
	}

	var modified exercise.ExerciseResponse
	if err := json.NewDecoder(modifyRes.Body).Decode(&modified); err != nil {
		t.Fatalf("failed to decode modify response: %v", err)
	}
	if modified.ExerciseId != created.ExerciseId || modified.ExerciseName != "Bent Over Row" {
		t.Errorf("unexpected modify response: %+v", modified)
	}

	exercisesReq := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	exercisesReq.Header.Set("Authorization", "Bearer "+token)
	exercisesRec := httptest.NewRecorder()
	router.ServeHTTP(exercisesRec, exercisesReq)

	var exercises []exercise.ExerciseResponse
	if err := json.NewDecoder(exercisesRec.Result().Body).Decode(&exercises); err != nil {
		t.Fatalf("failed to decode exercises response: %v", err)
	}

	found := false
	for _, e := range exercises {
		if e.ExerciseName == "Bent Over Row" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected modified exercise to appear in GET /exercises, got %+v", exercises)
	}
}

func TestIntegration_ModifyExercise_NoToken(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPut, "/exercises/1", strings.NewReader(`{"exercise_name":"Lunge"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_ModifyExercise_GlobalExercise(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	exercisesReq := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	exercisesReq.Header.Set("Authorization", "Bearer "+token)
	exercisesRec := httptest.NewRecorder()
	router.ServeHTTP(exercisesRec, exercisesReq)

	var exercises []exercise.ExerciseResponse
	if err := json.NewDecoder(exercisesRec.Result().Body).Decode(&exercises); err != nil {
		t.Fatalf("failed to decode exercises response: %v", err)
	}

	var benchPressId uint32
	for _, e := range exercises {
		if e.ExerciseName == "Bench Press" {
			benchPressId = e.ExerciseId
			break
		}
	}
	if benchPressId == 0 {
		t.Fatal("expected seeded global exercise \"Bench Press\" to be found")
	}

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/exercises/%d", benchPressId), strings.NewReader(`{"exercise_name":"Bench Press Variant"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_ModifyExercise_NotFound(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	req := httptest.NewRequest(http.MethodPut, "/exercises/999999", strings.NewReader(`{"exercise_name":"Lunge"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_ModifyExercise_DuplicateName(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	created := mustCreateExercise(t, router, token, "Cable Fly")

	// "Custom Test Exercise" is already seeded for alice (user 1).
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/exercises/%d", created.ExerciseId), strings.NewReader(`{"exercise_name":"Custom Test Exercise"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_GetWorkout(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	req := httptest.NewRequest(http.MethodGet, "/workouts/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	var got workout.WorkoutResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode workout response: %v", err)
	}

	if got.WorkoutId != 1 || got.TemplateName != "Push Day" {
		t.Errorf("unexpected workout response: %+v", got)
	}
	if len(got.Exercises) != 1 || got.Exercises[0].ExerciseName != "Bench Press" {
		t.Errorf("expected 1 exercise group for Bench Press, got %+v", got.Exercises)
	}
	if len(got.Exercises[0].Sets) != 1 {
		t.Errorf("expected Bench Press to have 1 set, got %+v", got.Exercises[0].Sets)
	}
}

func TestIntegration_CreateWorkoutAndGetItBack(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	createReq := httptest.NewRequest(http.MethodPost, "/workouts", strings.NewReader(`{
		"template_name": "Full Body",
		"completed_at": "2024-03-01T18:00:00Z",
		"exercises": [{"exercise_id": 1, "sets": [{"reps": 10, "weight_grams": 50000}]}]
	}`))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	createRes := createRec.Result()
	defer createRes.Body.Close()

	if createRes.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d", createRes.StatusCode)
	}

	var created workout.WorkoutResponse
	if err := json.NewDecoder(createRes.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if created.WorkoutId == 0 || created.TemplateId == 0 {
		t.Fatalf("expected generated ids in create response, got %+v", created)
	}
	if created.TemplateName != "Full Body" {
		t.Errorf("expected TemplateName %q, got %q", "Full Body", created.TemplateName)
	}

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/workouts/%d", created.WorkoutId), nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)

	getRes := getRec.Result()
	defer getRes.Body.Close()

	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("expected get status 200, got %d", getRes.StatusCode)
	}

	var got workout.WorkoutResponse
	if err := json.NewDecoder(getRes.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode workout response: %v", err)
	}
	if got.TemplateId != created.TemplateId || got.TemplateName != "Full Body" {
		t.Errorf("unexpected workout response: %+v", got)
	}
	if len(got.Exercises) != 1 || got.Exercises[0].ExerciseId != 1 {
		t.Fatalf("unexpected exercises: %+v", got.Exercises)
	}
	if len(got.Exercises[0].Sets) != 1 || got.Exercises[0].Sets[0].Reps != 10 || got.Exercises[0].Sets[0].WeightGrams != 50000 {
		t.Errorf("unexpected sets: %+v", got.Exercises[0].Sets)
	}
}

func TestIntegration_CreateWorkout_ReusesExistingTemplate(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	// Template 1 ("Push Day") is seeded for alice.
	req := httptest.NewRequest(http.MethodPost, "/workouts", strings.NewReader(`{
		"template_id": 1,
		"completed_at": "2024-03-02T18:00:00Z",
		"exercises": [{"exercise_id": 1, "sets": [{"reps": 6, "weight_grams": 70000}]}]
	}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", res.StatusCode)
	}

	var created workout.WorkoutResponse
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if created.TemplateId != 1 || created.TemplateName != "Push Day" {
		t.Errorf("expected the seeded template to be reused, got %+v", created)
	}
}

func TestIntegration_CreateWorkout_NoToken(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/workouts", strings.NewReader(
		`{"template_id":1,"exercises":[{"exercise_id":1,"sets":[{"reps":6,"weight_grams":70000}]}]}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_CreateWorkout_OtherUsersTemplate(t *testing.T) {
	router := newTestRouter(t)
	// Template 1 belongs to alice, not bob.
	token := mustLogin(t, router, "bob", "secret")

	req := httptest.NewRequest(http.MethodPost, "/workouts", strings.NewReader(
		`{"template_id":1,"exercises":[{"exercise_id":1,"sets":[{"reps":6,"weight_grams":70000}]}]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_CreateWorkout_NoSets(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	req := httptest.NewRequest(http.MethodPost, "/workouts", strings.NewReader(`{"template_id":1,"exercises":[]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_CreateWorkout_RejectsOutOfRangeValues(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	tests := []struct {
		name string
		body string
	}{
		{"negative reps", `{"template_id":1,"exercises":[{"exercise_id":1,"sets":[{"reps":-5,"weight_grams":60000}]}]}`},
		{"negative weight", `{"template_id":1,"exercises":[{"exercise_id":1,"sets":[{"reps":8,"weight_grams":-60000}]}]}`},
		{"negative exercise id", `{"template_id":1,"exercises":[{"exercise_id":-1,"sets":[{"reps":8,"weight_grams":60000}]}]}`},
		{"negative template id", `{"template_id":-1,"exercises":[{"exercise_id":1,"sets":[{"reps":8,"weight_grams":60000}]}]}`},
		{"reps above column range", `{"template_id":1,"exercises":[{"exercise_id":1,"sets":[{"reps":300,"weight_grams":60000}]}]}`},
		{"weight above column range", `{"template_id":1,"exercises":[{"exercise_id":1,"sets":[{"reps":8,"weight_grams":3000000000}]}]}`},
		{"exercise id above column range", `{"template_id":1,"exercises":[{"exercise_id":3000000000,"sets":[{"reps":8,"weight_grams":60000}]}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/workouts", strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Result().StatusCode != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", rec.Result().StatusCode)
			}
		})
	}
}

func TestIntegration_CreateWorkout_DuplicateTemplateName(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	// "Push Day" is already seeded for alice.
	req := httptest.NewRequest(http.MethodPost, "/workouts", strings.NewReader(`{
		"template_name": "Push Day",
		"exercises": [{"exercise_id": 1, "sets": [{"reps": 6, "weight_grams": 70000}]}]
	}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rec.Result().StatusCode)
	}
}

func mustCreateWorkout(t *testing.T, router http.Handler, token, body string) workout.WorkoutResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/workouts", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create workout failed: expected status 201, got %d", res.StatusCode)
	}

	var created workout.WorkoutResponse
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	return created
}

func modifyWorkout(router http.Handler, token string, workoutId any, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/workouts/%v", workoutId), strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func mustGetWorkout(t *testing.T, router http.Handler, token string, workoutId uint32) workout.WorkoutResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/workouts/%d", workoutId), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("get workout failed: expected status 200, got %d", res.StatusCode)
	}

	var got workout.WorkoutResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode workout response: %v", err)
	}

	return got
}

const createPushWorkoutBody = `{
	"template_id": 1,
	"completed_at": "2024-03-01T18:00:00Z",
	"exercises": [{"exercise_id": 1, "sets": [{"reps": 8, "weight_grams": 60000}, {"reps": 6, "weight_grams": 65000}]}]
}`

func TestIntegration_ModifyWorkoutAndGetItBack(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	created := mustCreateWorkout(t, router, token, createPushWorkoutBody)

	rec := modifyWorkout(router, token, created.WorkoutId, `{
		"completed_at": "2024-03-02T19:30:00Z",
		"exercises": [
			{"exercise_id": 1, "sets": [{"reps": 8, "weight_grams": 60000}, {"reps": 6, "weight_grams": 65000}, {"reps": 4, "weight_grams": 70000}]},
			{"exercise_id": 2, "sets": [{"reps": 5, "weight_grams": 100000}]}
		]
	}`)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected modify status 200, got %d", res.StatusCode)
	}

	var modified workout.WorkoutResponse
	if err := json.NewDecoder(res.Body).Decode(&modified); err != nil {
		t.Fatalf("failed to decode modify response: %v", err)
	}
	if modified.WorkoutId != created.WorkoutId || modified.TemplateId != 1 || modified.TemplateName != "Push Day" {
		t.Errorf("unexpected modify response: %+v", modified)
	}

	got := mustGetWorkout(t, router, token, created.WorkoutId)
	if !got.CompletedAt.Equal(time.Date(2024, 3, 2, 19, 30, 0, 0, time.UTC)) {
		t.Errorf("expected CompletedAt to be updated, got %v", got.CompletedAt)
	}
	if got.TemplateId != 1 || got.TemplateName != "Push Day" {
		t.Errorf("expected the template to be unchanged, got %+v", got)
	}
	if len(got.Exercises) != 2 || len(got.Exercises[0].Sets) != 3 || len(got.Exercises[1].Sets) != 1 {
		t.Errorf("expected the sets to be replaced by the submitted ones, got %+v", got.Exercises)
	}
}

func TestIntegration_ModifyWorkout_KeepsCompletedAtWhenOmitted(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	created := mustCreateWorkout(t, router, token, createPushWorkoutBody)

	rec := modifyWorkout(router, token, created.WorkoutId,
		`{"exercises": [{"exercise_id": 1, "sets": [{"reps": 8, "weight_grams": 60000}]}]}`)
	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Result().StatusCode)
	}

	got := mustGetWorkout(t, router, token, created.WorkoutId)
	if !got.CompletedAt.Equal(time.Date(2024, 3, 1, 18, 0, 0, 0, time.UTC)) {
		t.Errorf("expected the original CompletedAt to be kept, got %v", got.CompletedAt)
	}
	if len(got.Exercises) != 1 || len(got.Exercises[0].Sets) != 1 {
		t.Errorf("expected the sets to be replaced, got %+v", got.Exercises)
	}
}

func TestIntegration_ModifyWorkout_IsIdempotent(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	created := mustCreateWorkout(t, router, token, createPushWorkoutBody)

	body := `{
		"completed_at": "2024-03-02T19:30:00Z",
		"exercises": [{"exercise_id": 1, "sets": [{"reps": 5, "weight_grams": 80000}]}]
	}`

	first := modifyWorkout(router, token, created.WorkoutId, body)
	second := modifyWorkout(router, token, created.WorkoutId, body)

	if first.Result().StatusCode != http.StatusOK || second.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected both requests to return 200, got %d and %d", first.Result().StatusCode, second.Result().StatusCode)
	}
	if first.Body.String() != second.Body.String() {
		t.Errorf("expected identical responses, got %q and %q", first.Body.String(), second.Body.String())
	}
}

func TestIntegration_ModifyWorkout_IgnoresTemplateFields(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	created := mustCreateWorkout(t, router, token, createPushWorkoutBody)

	// Sending back a GET response's fields, with a different template, must not swap it.
	rec := modifyWorkout(router, token, created.WorkoutId, `{
		"workout_id": 999,
		"template_id": 12345,
		"template_name": "Sneaky Day",
		"exercises": [{"exercise_id": 1, "exercise_name": "Bench Press", "sets": [{"reps": 8, "weight_grams": 60000}]}]
	}`)
	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Result().StatusCode)
	}

	got := mustGetWorkout(t, router, token, created.WorkoutId)
	if got.WorkoutId != created.WorkoutId || got.TemplateId != 1 || got.TemplateName != "Push Day" {
		t.Errorf("expected workout and template to be unchanged, got %+v", got)
	}
}

func TestIntegration_ModifyWorkout_NoToken(t *testing.T) {
	router := newTestRouter(t)

	rec := modifyWorkout(router, "", 1, `{"exercises":[{"exercise_id":1,"sets":[{"reps":6,"weight_grams":70000}]}]}`)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_ModifyWorkout_NotFound(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	// The last id is beyond what the int4 column can hold.
	for _, workoutId := range []uint32{999999, 3000000000} {
		rec := modifyWorkout(router, token, workoutId, `{"exercises":[{"exercise_id":1,"sets":[{"reps":6,"weight_grams":70000}]}]}`)

		if rec.Result().StatusCode != http.StatusNotFound {
			t.Errorf("workout %d: expected status 404, got %d", workoutId, rec.Result().StatusCode)
		}
	}
}

func TestIntegration_ModifyWorkout_WrongUser(t *testing.T) {
	router := newTestRouter(t)
	aliceToken := mustLogin(t, router, "alice", "secret")
	bobToken := mustLogin(t, router, "bob", "secret")

	created := mustCreateWorkout(t, router, aliceToken, createPushWorkoutBody)

	rec := modifyWorkout(router, bobToken, created.WorkoutId,
		`{"exercises":[{"exercise_id":1,"sets":[{"reps":1,"weight_grams":1000}]}]}`)
	if rec.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Result().StatusCode)
	}

	got := mustGetWorkout(t, router, aliceToken, created.WorkoutId)
	if len(got.Exercises) != 1 || len(got.Exercises[0].Sets) != 2 {
		t.Errorf("expected alice's workout to be untouched, got %+v", got.Exercises)
	}
}

func TestIntegration_ModifyWorkout_InvalidWorkoutId(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	rec := modifyWorkout(router, token, "abc", `{"exercises":[{"exercise_id":1,"sets":[{"reps":6,"weight_grams":70000}]}]}`)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_ModifyWorkout_RejectsInvalidSets(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	created := mustCreateWorkout(t, router, token, createPushWorkoutBody)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"no sets", `{"exercises":[]}`, http.StatusBadRequest},
		{"zero reps", `{"exercises":[{"exercise_id":1,"sets":[{"reps":0,"weight_grams":60000}]}]}`, http.StatusBadRequest},
		{"negative reps", `{"exercises":[{"exercise_id":1,"sets":[{"reps":-5,"weight_grams":60000}]}]}`, http.StatusBadRequest},
		{"weight above column range", `{"exercises":[{"exercise_id":1,"sets":[{"reps":8,"weight_grams":3000000000}]}]}`, http.StatusBadRequest},
		{"unknown exercise", `{"exercises":[{"exercise_id":999999,"sets":[{"reps":8,"weight_grams":60000}]}]}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := modifyWorkout(router, token, created.WorkoutId, tt.body)

			if rec.Result().StatusCode != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Result().StatusCode)
			}
		})
	}

	got := mustGetWorkout(t, router, token, created.WorkoutId)
	if len(got.Exercises) != 1 || len(got.Exercises[0].Sets) != 2 {
		t.Errorf("expected rejected modifies to leave the workout untouched, got %+v", got.Exercises)
	}
}

func TestIntegration_GetWorkout_NoToken(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/workouts/1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_GetWorkout_NotFound(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	req := httptest.NewRequest(http.MethodGet, "/workouts/999999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_GetWorkout_WrongUser(t *testing.T) {
	router := newTestRouter(t)
	// workout 1 belongs to alice (user 1), not bob.
	token := mustLogin(t, router, "bob", "secret")

	req := httptest.NewRequest(http.MethodGet, "/workouts/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_GetWorkout_InvalidWorkoutId(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	req := httptest.NewRequest(http.MethodGet, "/workouts/not-a-number", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_ModifyExercise_NameRequired(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	created := mustCreateExercise(t, router, token, "Face Pull")

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/exercises/%d", created.ExerciseId), strings.NewReader(`{"exercise_name":"   "}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_ModifyExercise_InvalidExerciseId(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	req := httptest.NewRequest(http.MethodPut, "/exercises/abc", strings.NewReader(`{"exercise_name":"Lunge"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_CreateTemplate(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	req := httptest.NewRequest(http.MethodPost, "/templates", strings.NewReader(`{"template_name":"Pull Day"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", res.StatusCode)
	}

	var created template.TemplateResponse
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if created.TemplateName != "Pull Day" || created.TemplateId == 0 {
		t.Errorf("unexpected create response: %+v", created)
	}
}

func TestIntegration_CreateTemplate_NoToken(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/templates", strings.NewReader(`{"template_name":"Pull Day"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_CreateTemplate_NameRequired(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	req := httptest.NewRequest(http.MethodPost, "/templates", strings.NewReader(`{"template_name":"   "}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_CreateTemplate_DuplicateName(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	// "Push Day" is already seeded for alice.
	req := httptest.NewRequest(http.MethodPost, "/templates", strings.NewReader(`{"template_name":"Push Day"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rec.Result().StatusCode)
	}
}
