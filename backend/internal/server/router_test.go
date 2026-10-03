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
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/auth"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/server"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/testutil"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/user"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/workout"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

// The ids of the global exercises the tests log sets for.
var benchPressId, dumbbellBenchPressId uuid.UUID

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

	if benchPressId, err = testutil.GlobalExerciseId(ctx, pool, "Bench Press (Barbell)"); err != nil {
		log.Fatal(err)
	}
	if dumbbellBenchPressId, err = testutil.GlobalExerciseId(ctx, pool, "Bench Press (Dumbbell)"); err != nil {
		log.Fatal(err)
	}

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

	var exercisesBody exercise.ExercisesResponse
	if err := json.NewDecoder(exercisesRes.Body).Decode(&exercisesBody); err != nil {
		t.Fatalf("failed to decode exercises response: %v", err)
	}
	exercises := exercisesBody.Exercises

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

func post(router http.Handler, token, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var body T
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return body
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
	if created.ExerciseName != "Lunge" || created.ExerciseId == uuid.Nil() {
		t.Errorf("unexpected create response: %+v", created)
	}

	exercisesReq := httptest.NewRequest(http.MethodGet, "/exercises", nil)
	exercisesReq.Header.Set("Authorization", "Bearer "+token)
	exercisesRec := httptest.NewRecorder()
	router.ServeHTTP(exercisesRec, exercisesReq)

	exercisesRes := exercisesRec.Result()
	defer exercisesRes.Body.Close()

	var exercisesBody exercise.ExercisesResponse
	if err := json.NewDecoder(exercisesRes.Body).Decode(&exercisesBody); err != nil {
		t.Fatalf("failed to decode exercises response: %v", err)
	}
	exercises := exercisesBody.Exercises

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

func TestIntegration_CreateExercise_ExistingName(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	// alice's "Custom Test Exercise" (1) is seeded, and "Bench Press (Barbell)"
	// is a global exercise. Neither is created again; alice gets it back.
	tests := []struct {
		name string
		want exercise.ExerciseResponse
	}{
		{"custom test exercise", exercise.ExerciseResponse{ExerciseId: testutil.Id(1), ExerciseName: "Custom Test Exercise"}},
		{"bench press (barbell)", exercise.ExerciseResponse{ExerciseId: benchPressId, ExerciseName: "Bench Press (Barbell)"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postExercise(router, token, uuid.NewV7(), tt.name)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", rec.Code)
			}
			if got := decodeBody[exercise.ExerciseResponse](t, rec); got != tt.want {
				t.Errorf("expected the existing exercise %+v, got %+v", tt.want, got)
			}
		})
	}
}

func TestIntegration_CreateExercise_SafeToRetry(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")
	exerciseId := uuid.NewV7()

	first := postExercise(router, token, exerciseId, "Zercher Squat")
	retry := postExercise(router, token, exerciseId, "Zercher Squat")

	if first.Code != http.StatusCreated || retry.Code != http.StatusOK {
		t.Fatalf("expected status 201 and then 200, got %d and %d", first.Code, retry.Code)
	}
	if first.Body.String() != retry.Body.String() {
		t.Errorf("expected the retry to return the same exercise, got %q and %q", first.Body.String(), retry.Body.String())
	}
	want := exercise.ExerciseResponse{ExerciseId: exerciseId, ExerciseName: "Zercher Squat"}
	if got := decodeBody[exercise.ExerciseResponse](t, first); got != want {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}

// Two of alice's devices create the same exercise while offline. The second
// to sync gets the first one's exercise back, and logs its workout against
// that, since the server never stored the exercise under its own id.
func TestIntegration_CreateExercise_OfflineOnTwoDevices(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")
	tabletId, phoneId := uuid.NewV7(), uuid.NewV7()

	if rec := postExercise(router, token, tabletId, "Cossack Squat"); rec.Code != http.StatusCreated {
		t.Fatalf("expected the tablet's exercise to be created, got status %d", rec.Code)
	}

	rec := postExercise(router, token, phoneId, "cossack squat")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected the phone to get the existing exercise with status 200, got %d", rec.Code)
	}
	got := decodeBody[exercise.ExerciseResponse](t, rec)
	if want := (exercise.ExerciseResponse{ExerciseId: tabletId, ExerciseName: "Cossack Squat"}); got != want {
		t.Fatalf("expected the tablet's exercise %+v, got %+v", want, got)
	}

	workoutBody := func(exerciseId uuid.UUID) string {
		return fmt.Sprintf(`{"template_id":%q,"started_at":"2024-06-01T17:00:00Z","completed_at":"2024-06-01T18:00:00Z","exercises":[{"exercise_id":%q,"sets":[{"reps":8,"weight_grams":40000}]}]}`,
			testutil.Id(1), exerciseId)
	}
	workoutId := uuid.NewV7()
	if rec := putWorkout(router, token, workoutId, workoutBody(phoneId)); rec.Code != http.StatusBadRequest {
		t.Errorf("expected the phone's own exercise id to be unknown, got status %d", rec.Code)
	}
	if rec := putWorkout(router, token, workoutId, workoutBody(got.ExerciseId)); rec.Code != http.StatusCreated {
		t.Errorf("expected the workout to be logged against the existing exercise, got status %d", rec.Code)
	}
}

func TestIntegration_CreateExercise_IdOfAnotherUsersExercise(t *testing.T) {
	router := newTestRouter(t)
	aliceToken := mustLogin(t, router, "alice", "secret")
	bobToken := mustLogin(t, router, "bob", "secret")
	exerciseId := uuid.NewV7()

	if rec := postExercise(router, aliceToken, exerciseId, "Alice's Belt Squat"); rec.Code != http.StatusCreated {
		t.Fatalf("expected alice's exercise to be created, got status %d", rec.Code)
	}

	if rec := postExercise(router, bobToken, exerciseId, "Bob's Belt Squat"); rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rec.Code)
	}
}

func postExercise(router http.Handler, token string, exerciseId uuid.UUID, name string) *httptest.ResponseRecorder {
	return post(router, token, "/exercises", fmt.Sprintf(`{"exercise_id":%q,"exercise_name":%q}`, exerciseId, name))
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

	modifyReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/exercises/%v", created.ExerciseId), strings.NewReader(`{"exercise_name":"Bent Over Row"}`))
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

	var exercisesBody exercise.ExercisesResponse
	if err := json.NewDecoder(exercisesRec.Result().Body).Decode(&exercisesBody); err != nil {
		t.Fatalf("failed to decode exercises response: %v", err)
	}
	exercises := exercisesBody.Exercises

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

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/exercises/%v", benchPressId), strings.NewReader(`{"exercise_name":"Bench Press Variant"}`))
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

	req := httptest.NewRequest(http.MethodPut, "/exercises/"+uuid.NewV7().String(), strings.NewReader(`{"exercise_name":"Lunge"}`))
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
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/exercises/%v", created.ExerciseId), strings.NewReader(`{"exercise_name":"Custom Test Exercise"}`))
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

	req := httptest.NewRequest(http.MethodGet, "/workouts/"+testutil.Id(1).String(), nil)
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

	if got.WorkoutId != testutil.Id(1) || got.TemplateName != "Push Day" {
		t.Errorf("unexpected workout response: %+v", got)
	}
	if len(got.Exercises) != 1 || got.Exercises[0].ExerciseName != "Bench Press (Barbell)" {
		t.Errorf("expected 1 exercise group for Bench Press, got %+v", got.Exercises)
	}
	if len(got.Exercises[0].Sets) != 1 {
		t.Errorf("expected Bench Press to have 1 set, got %+v", got.Exercises[0].Sets)
	}
}

func TestIntegration_PutNewWorkoutAndGetItBack(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")
	workoutId := uuid.NewV7()

	rec := putWorkout(router, token, workoutId, fmt.Sprintf(`{
		"template_name": "Full Body",
		"started_at": "2024-03-01T17:00:00Z",
		"completed_at": "2024-03-01T18:00:00Z",
		"exercises": [{"exercise_id": %q, "sets": [{"reps": 10, "weight_grams": 50000}]}]
	}`, benchPressId))

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
	created := decodeBody[workout.WorkoutResponse](t, rec)
	if created.WorkoutId != workoutId || created.TemplateId == uuid.Nil() || created.TemplateName != "Full Body" {
		t.Fatalf("expected workout %v under a new template Full Body, got %+v", workoutId, created)
	}

	got := mustGetWorkout(t, router, token, workoutId)
	if got.TemplateId != created.TemplateId || got.TemplateName != "Full Body" {
		t.Errorf("unexpected workout response: %+v", got)
	}
	if !got.StartedAt.Equal(time.Date(2024, 3, 1, 17, 0, 0, 0, time.UTC)) || !got.CompletedAt.Equal(time.Date(2024, 3, 1, 18, 0, 0, 0, time.UTC)) {
		t.Errorf("expected StartedAt 17:00 and CompletedAt 18:00, got %v and %v", got.StartedAt, got.CompletedAt)
	}
	if len(got.Exercises) != 1 || got.Exercises[0].ExerciseId != benchPressId {
		t.Fatalf("unexpected exercises: %+v", got.Exercises)
	}
	if len(got.Exercises[0].Sets) != 1 || got.Exercises[0].Sets[0].Reps != 10 || got.Exercises[0].Sets[0].WeightGrams != 50000 {
		t.Errorf("unexpected sets: %+v", got.Exercises[0].Sets)
	}
}

func TestIntegration_PutNewWorkout_UnderExistingTemplate(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	// Template 1 ("Push Day") is seeded for alice.
	created := mustCreateWorkout(t, router, token, createPushWorkoutBody())

	if created.TemplateId != testutil.Id(1) || created.TemplateName != "Push Day" {
		t.Errorf("expected the seeded template to be reused, got %+v", created)
	}
}

func TestIntegration_PutWorkout_NoToken(t *testing.T) {
	router := newTestRouter(t)

	rec := putWorkout(router, "", uuid.NewV7(), createPushWorkoutBody())

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestIntegration_PutNewWorkout_OtherUsersTemplate(t *testing.T) {
	router := newTestRouter(t)
	// Template 1 belongs to alice, not bob.
	token := mustLogin(t, router, "bob", "secret")

	rec := putWorkout(router, token, uuid.NewV7(), createPushWorkoutBody())

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestIntegration_PutNewWorkout_WithoutTemplate(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	rec := putWorkout(router, token, uuid.NewV7(), fmt.Sprintf(`{
		"started_at": "2024-03-02T17:00:00Z",
		"completed_at": "2024-03-02T18:00:00Z",
		"exercises": [{"exercise_id": %q, "sets": [{"reps": 6, "weight_grams": 70000}]}]
	}`, benchPressId))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestIntegration_PutNewWorkout_DuplicateTemplateName(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	// "Push Day" is already seeded for alice.
	rec := putWorkout(router, token, uuid.NewV7(), fmt.Sprintf(`{
		"template_name": "Push Day",
		"started_at": "2024-03-02T17:00:00Z",
		"completed_at": "2024-03-02T18:00:00Z",
		"exercises": [{"exercise_id": %q, "sets": [{"reps": 6, "weight_grams": 70000}]}]
	}`, benchPressId))

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rec.Code)
	}
}

func TestIntegration_PutWorkout_RejectsInvalidValues(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	// Each body is valid apart from the one thing its name says, so a 400 is for that.
	body := func(templateId, exerciseId, sets string) string {
		return fmt.Sprintf(`{"template_id":%s,"started_at":"2024-03-02T17:00:00Z","completed_at":"2024-03-02T18:00:00Z","exercises":[{"exercise_id":%s,"sets":[%s]}]}`,
			templateId, exerciseId, sets)
	}
	pushDay, bench := fmt.Sprintf("%q", testutil.Id(1)), fmt.Sprintf("%q", benchPressId)
	tests := []struct {
		name string
		body string
	}{
		{"no sets", fmt.Sprintf(`{"template_id":%s,"started_at":"2024-03-02T17:00:00Z","completed_at":"2024-03-02T18:00:00Z","exercises":[]}`, pushDay)},
		{"zero reps", body(pushDay, bench, `{"reps":0,"weight_grams":60000}`)},
		{"negative reps", body(pushDay, bench, `{"reps":-5,"weight_grams":60000}`)},
		{"negative weight", body(pushDay, bench, `{"reps":8,"weight_grams":-60000}`)},
		{"reps above column range", body(pushDay, bench, `{"reps":300,"weight_grams":60000}`)},
		{"weight above column range", body(pushDay, bench, `{"reps":8,"weight_grams":3000000000}`)},
		{"numeric exercise id", body(pushDay, "1", `{"reps":8,"weight_grams":60000}`)},
		{"numeric template id", body("1", bench, `{"reps":8,"weight_grams":60000}`)},
		{"unknown exercise", body(pushDay, fmt.Sprintf("%q", uuid.NewV7()), `{"reps":8,"weight_grams":60000}`)},
		{"missing started_at", fmt.Sprintf(`{"template_id":%s,"completed_at":"2024-03-02T18:00:00Z","exercises":[{"exercise_id":%s,"sets":[{"reps":8,"weight_grams":60000}]}]}`, pushDay, bench)},
		{"missing completed_at", fmt.Sprintf(`{"template_id":%s,"started_at":"2024-03-02T17:00:00Z","exercises":[{"exercise_id":%s,"sets":[{"reps":8,"weight_grams":60000}]}]}`, pushDay, bench)},
		{"started_at after completed_at", fmt.Sprintf(`{"template_id":%s,"started_at":"2024-03-02T19:00:00Z","completed_at":"2024-03-02T18:00:00Z","exercises":[{"exercise_id":%s,"sets":[{"reps":8,"weight_grams":60000}]}]}`, pushDay, bench)},
	}

	// Both a new workout and a replacement must reject them, leaving the stored one untouched.
	stored := mustCreateWorkout(t, router, token, createPushWorkoutBody())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, workoutId := range []uuid.UUID{uuid.NewV7(), stored.WorkoutId} {
				if rec := putWorkout(router, token, workoutId, tt.body); rec.Code != http.StatusBadRequest {
					t.Errorf("workout %v: expected status 400, got %d", workoutId, rec.Code)
				}
			}
		})
	}

	got := mustGetWorkout(t, router, token, stored.WorkoutId)
	if len(got.Exercises) != 1 || len(got.Exercises[0].Sets) != 2 {
		t.Errorf("expected rejected puts to leave the workout untouched, got %+v", got.Exercises)
	}
}

// A client syncing a workout it logged offline can send it as often as it
// needs to: the first put creates it, and sending it again changes nothing.
func TestIntegration_PutWorkout_SafeToRepeat(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")
	workoutId := uuid.NewV7()

	first := putWorkout(router, token, workoutId, createPushWorkoutBody())
	again := putWorkout(router, token, workoutId, createPushWorkoutBody())

	if first.Code != http.StatusCreated || again.Code != http.StatusOK {
		t.Fatalf("expected status 201 and then 200, got %d and %d", first.Code, again.Code)
	}
	if first.Body.String() != again.Body.String() {
		t.Errorf("expected the same workout back, got %q and %q", first.Body.String(), again.Body.String())
	}

	listed := 0
	for _, w := range mustListWorkouts(t, router, token).Workouts {
		if w.WorkoutId == workoutId {
			listed++
		}
	}
	if listed != 1 {
		t.Errorf("expected the workout to be stored once, found it %d times", listed)
	}
}

func TestIntegration_PutWorkout_ReplacesAndGetItBack(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	created := mustCreateWorkout(t, router, token, createPushWorkoutBody())

	rec := putWorkout(router, token, created.WorkoutId, fmt.Sprintf(`{
		"started_at": "2024-03-01T16:30:00Z",
		"completed_at": "2024-03-02T19:30:00Z",
		"exercises": [
			{"exercise_id": %q, "sets": [{"reps": 8, "weight_grams": 60000}, {"reps": 6, "weight_grams": 65000}, {"reps": 4, "weight_grams": 70000}]},
			{"exercise_id": %q, "sets": [{"reps": 5, "weight_grams": 100000}]}
		]
	}`, benchPressId, dumbbellBenchPressId))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	replaced := decodeBody[workout.WorkoutResponse](t, rec)
	if replaced.WorkoutId != created.WorkoutId || replaced.TemplateId != testutil.Id(1) || replaced.TemplateName != "Push Day" {
		t.Errorf("unexpected put response: %+v", replaced)
	}

	got := mustGetWorkout(t, router, token, created.WorkoutId)
	if !got.StartedAt.Equal(time.Date(2024, 3, 1, 16, 30, 0, 0, time.UTC)) || !got.CompletedAt.Equal(time.Date(2024, 3, 2, 19, 30, 0, 0, time.UTC)) {
		t.Errorf("expected both times to be replaced, got %v and %v", got.StartedAt, got.CompletedAt)
	}
	if got.TemplateId != testutil.Id(1) || got.TemplateName != "Push Day" {
		t.Errorf("expected the template to be unchanged, got %+v", got)
	}
	if len(got.Exercises) != 2 || len(got.Exercises[0].Sets) != 3 || len(got.Exercises[1].Sets) != 1 {
		t.Errorf("expected the sets to be replaced by the submitted ones, got %+v", got.Exercises)
	}
}

func TestIntegration_PutWorkout_IgnoresTemplateFieldsOfExistingWorkout(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	created := mustCreateWorkout(t, router, token, createPushWorkoutBody())

	// Sending back a GET response's fields, with a different template, must not swap it.
	rec := putWorkout(router, token, created.WorkoutId, fmt.Sprintf(`{
		"workout_id": %q,
		"template_id": %q,
		"template_name": "Sneaky Day",
		"started_at": "2024-03-01T17:00:00Z",
		"completed_at": "2024-03-01T18:00:00Z",
		"exercises": [{"exercise_id": %q, "exercise_name": "Bench Press", "sets": [{"reps": 8, "weight_grams": 60000}]}]
	}`, uuid.NewV7(), uuid.NewV7(), benchPressId))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	got := mustGetWorkout(t, router, token, created.WorkoutId)
	if got.WorkoutId != created.WorkoutId || got.TemplateId != testutil.Id(1) || got.TemplateName != "Push Day" {
		t.Errorf("expected workout and template to be unchanged, got %+v", got)
	}
}

func TestIntegration_PutWorkout_IdOfAnotherUsersWorkout(t *testing.T) {
	router := newTestRouter(t)
	aliceToken := mustLogin(t, router, "alice", "secret")
	bobToken := mustLogin(t, router, "bob", "secret")

	created := mustCreateWorkout(t, router, aliceToken, createPushWorkoutBody())

	rec := putWorkout(router, bobToken, created.WorkoutId, fmt.Sprintf(`{
		"template_name": "Bob's Day",
		"started_at": "2024-03-04T17:00:00Z",
		"completed_at": "2024-03-04T18:00:00Z",
		"exercises": [{"exercise_id": %q, "sets": [{"reps": 1, "weight_grams": 1000}]}]
	}`, benchPressId))
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rec.Code)
	}

	got := mustGetWorkout(t, router, aliceToken, created.WorkoutId)
	if len(got.Exercises) != 1 || len(got.Exercises[0].Sets) != 2 {
		t.Errorf("expected alice's workout to be untouched, got %+v", got.Exercises)
	}
}

func TestIntegration_PutWorkout_InvalidWorkoutId(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	// Ids are UUIDs, so the numbers older clients sent are rejected too.
	for _, workoutId := range []string{"abc", "1"} {
		if rec := putWorkout(router, token, workoutId, createPushWorkoutBody()); rec.Code != http.StatusBadRequest {
			t.Errorf("workout %q: expected status 400, got %d", workoutId, rec.Code)
		}
	}
}

func putWorkout(router http.Handler, token string, workoutId any, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/workouts/%v", workoutId), strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// mustCreateWorkout puts body under a new workout id.
func mustCreateWorkout(t *testing.T, router http.Handler, token, body string) workout.WorkoutResponse {
	t.Helper()

	rec := putWorkout(router, token, uuid.NewV7(), body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create workout failed: expected status 201, got %d", rec.Code)
	}
	return decodeBody[workout.WorkoutResponse](t, rec)
}

func mustGetWorkout(t *testing.T, router http.Handler, token string, workoutId uuid.UUID) workout.WorkoutResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/workouts/%v", workoutId), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("get workout failed: expected status 200, got %d", rec.Code)
	}
	return decodeBody[workout.WorkoutResponse](t, rec)
}

func createPushWorkoutBody() string {
	return fmt.Sprintf(`{
	"template_id": %q,
	"started_at": "2024-03-01T17:00:00Z",
	"completed_at": "2024-03-01T18:00:00Z",
	"exercises": [{"exercise_id": %q, "sets": [{"reps": 8, "weight_grams": 60000}, {"reps": 6, "weight_grams": 65000}]}]
}`, testutil.Id(1), benchPressId)
}

// createWorkoutBodyAt returns a create body for a one-hour workout under
// template 1 that completed at completedAt, an RFC 3339 timestamp.
func createWorkoutBodyAt(completedAt string) string {
	completed, err := time.Parse(time.RFC3339, completedAt)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(`{
		"template_id": %q,
		"started_at": %q,
		"completed_at": %q,
		"exercises": [{"exercise_id": %q, "sets": [{"reps": 8, "weight_grams": 60000}]}]
	}`, testutil.Id(1), completed.Add(-time.Hour).Format(time.RFC3339), completedAt, benchPressId)
}

func mustListWorkouts(t *testing.T, router http.Handler, token string) workout.WorkoutsResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/workouts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("list workouts failed: expected status 200, got %d", res.StatusCode)
	}

	var got workout.WorkoutsResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode workouts response: %v", err)
	}

	return got
}

func indexOfWorkout(list workout.WorkoutsResponse, workoutId uuid.UUID) int {
	for i, w := range list.Workouts {
		if w.WorkoutId == workoutId {
			return i
		}
	}
	return -1
}

func TestIntegration_GetWorkouts(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	list := mustListWorkouts(t, router, token)

	// Workout 1 is seeded for alice.
	i := indexOfWorkout(list, testutil.Id(1))
	if i == -1 {
		t.Fatalf("expected the seeded workout 1 to be listed, got %+v", list.Workouts)
	}
	got := list.Workouts[i]
	if got.TemplateId != testutil.Id(1) || got.TemplateName != "Push Day" {
		t.Errorf("unexpected template for workout 1: %+v", got)
	}
	if want := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC); !got.StartedAt.Equal(want) {
		t.Errorf("expected StartedAt %v, got %v", want, got.StartedAt)
	}
	if want := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC); !got.CompletedAt.Equal(want) {
		t.Errorf("expected CompletedAt %v, got %v", want, got.CompletedAt)
	}
}

func TestIntegration_GetWorkouts_ResponseContainsOnlyExpectedFields(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	req := httptest.NewRequest(http.MethodGet, "/workouts", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var raw map[string][]map[string]any
	if err := json.NewDecoder(rec.Result().Body).Decode(&raw); err != nil {
		t.Fatalf("failed to decode workouts response: %v", err)
	}

	if len(raw["workouts"]) == 0 {
		t.Fatal("expected at least the seeded workout to be listed")
	}
	for _, item := range raw["workouts"] {
		if len(item) != 5 {
			t.Errorf("expected exactly workout_id, template_id, template_name, started_at and completed_at, got %v", item)
		}
		if _, ok := item["exercises"]; ok {
			t.Errorf("expected the list to omit exercises, got %v", item)
		}
	}
}

func TestIntegration_GetWorkouts_NewestFirst(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	older := mustCreateWorkout(t, router, token, createWorkoutBodyAt("2032-05-01T10:00:00Z"))
	newer := mustCreateWorkout(t, router, token, createWorkoutBodyAt("2032-05-02T10:00:00Z"))

	list := mustListWorkouts(t, router, token)

	olderAt, newerAt := indexOfWorkout(list, older.WorkoutId), indexOfWorkout(list, newer.WorkoutId)
	if olderAt == -1 || newerAt == -1 {
		t.Fatalf("expected both new workouts to be listed, got %+v", list.Workouts)
	}
	if newerAt > olderAt {
		t.Errorf("expected the newer workout before the older one, got positions %d and %d", newerAt, olderAt)
	}
}

func TestIntegration_GetWorkouts_ReflectsModifiedCompletedAt(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	first := mustCreateWorkout(t, router, token, createWorkoutBodyAt("2033-05-01T10:00:00Z"))
	second := mustCreateWorkout(t, router, token, createWorkoutBodyAt("2033-05-02T10:00:00Z"))

	// Moving the first workout past the second must reorder the list, and
	// editing must not otherwise change where a workout appears.
	rec := putWorkout(router, token, first.WorkoutId, fmt.Sprintf(`{
		"started_at": "2033-05-03T09:00:00Z",
		"completed_at": "2033-05-03T10:00:00Z",
		"exercises": [{"exercise_id": %q, "sets": [{"reps": 8, "weight_grams": 60000}]}]
	}`, benchPressId))
	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("expected modify status 200, got %d", rec.Result().StatusCode)
	}

	list := mustListWorkouts(t, router, token)

	firstAt, secondAt := indexOfWorkout(list, first.WorkoutId), indexOfWorkout(list, second.WorkoutId)
	if firstAt == -1 || secondAt == -1 {
		t.Fatalf("expected both workouts to be listed, got %+v", list.Workouts)
	}
	if firstAt > secondAt {
		t.Errorf("expected the modified workout to move before the other one, got positions %d and %d", firstAt, secondAt)
	}
}

func TestIntegration_GetWorkouts_OnlyOwnWorkouts(t *testing.T) {
	router := newTestRouter(t)
	aliceToken := mustLogin(t, router, "alice", "secret")
	bobToken := mustLogin(t, router, "bob", "secret")

	created := mustCreateWorkout(t, router, aliceToken, createWorkoutBodyAt("2034-05-01T10:00:00Z"))

	if indexOfWorkout(mustListWorkouts(t, router, aliceToken), created.WorkoutId) == -1 {
		t.Error("expected alice to see her own workout")
	}

	bobList := mustListWorkouts(t, router, bobToken)
	if indexOfWorkout(bobList, created.WorkoutId) != -1 || indexOfWorkout(bobList, testutil.Id(1)) != -1 {
		t.Errorf("expected bob not to see alice's workouts, got %+v", bobList.Workouts)
	}
	// Bob has no workouts, which must be an empty array rather than null.
	if bobList.Workouts == nil {
		t.Error("expected an empty list to be an empty array, got null")
	}
}

func TestIntegration_GetWorkouts_NoToken(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/workouts", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_GetWorkout_NoToken(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/workouts/"+testutil.Id(1).String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestIntegration_GetWorkout_NotFound(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	req := httptest.NewRequest(http.MethodGet, "/workouts/"+uuid.NewV7().String(), nil)
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

	req := httptest.NewRequest(http.MethodGet, "/workouts/"+testutil.Id(1).String(), nil)
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

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/exercises/%v", created.ExerciseId), strings.NewReader(`{"exercise_name":"   "}`))
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
	if created.TemplateName != "Pull Day" || created.TemplateId == uuid.Nil() {
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

func TestIntegration_CreateTemplate_ExistingName(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	// Template 1, "Push Day", is seeded for alice.
	rec := postTemplate(router, token, uuid.NewV7(), "push day")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	want := template.TemplateResponse{TemplateId: testutil.Id(1), TemplateName: "Push Day"}
	if got := decodeBody[template.TemplateResponse](t, rec); got != want {
		t.Errorf("expected the existing template %+v, got %+v", want, got)
	}
}

func TestIntegration_CreateTemplate_SafeToRetry(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")
	templateId := uuid.NewV7()

	first := postTemplate(router, token, templateId, "Retry Day")
	retry := postTemplate(router, token, templateId, "Retry Day")

	if first.Code != http.StatusCreated || retry.Code != http.StatusOK {
		t.Fatalf("expected status 201 and then 200, got %d and %d", first.Code, retry.Code)
	}
	if first.Body.String() != retry.Body.String() {
		t.Errorf("expected the retry to return the same template, got %q and %q", first.Body.String(), retry.Body.String())
	}
	want := template.TemplateResponse{TemplateId: templateId, TemplateName: "Retry Day"}
	if got := decodeBody[template.TemplateResponse](t, first); got != want {
		t.Errorf("expected %+v, got %+v", want, got)
	}
}

func TestIntegration_CreateTemplate_IdOfAnotherUsersTemplate(t *testing.T) {
	router := newTestRouter(t)
	// Template 1 belongs to alice.
	token := mustLogin(t, router, "bob", "secret")

	if rec := postTemplate(router, token, testutil.Id(1), "Bob's Day"); rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rec.Code)
	}
}

func postTemplate(router http.Handler, token string, templateId uuid.UUID, name string) *httptest.ResponseRecorder {
	return post(router, token, "/templates", fmt.Sprintf(`{"template_id":%q,"template_name":%q}`, templateId, name))
}

func TestIntegration_CreateTemplateAndSeeItInList(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	createReq := httptest.NewRequest(http.MethodPost, "/templates", strings.NewReader(`{"template_name":"Arm Day"}`))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)

	if createRec.Result().StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d", createRec.Result().StatusCode)
	}

	req := httptest.NewRequest(http.MethodGet, "/templates", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	var body template.TemplatesResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode templates response: %v", err)
	}

	byName := make(map[string]template.TemplateWithLatestWorkoutResponse, len(body.Templates))
	for _, tmpl := range body.Templates {
		byName[tmpl.TemplateName] = tmpl
	}
	// "Push Day" is seeded for alice, with a workout logged under it.
	if pushDay, ok := byName["Push Day"]; !ok || pushDay.LatestWorkout == nil {
		t.Errorf("expected Push Day with a latest workout in GET /templates, got %+v", body.Templates)
	}
	if armDay, ok := byName["Arm Day"]; !ok || armDay.LatestWorkout != nil {
		t.Errorf("expected Arm Day without a latest workout in GET /templates, got %+v", body.Templates)
	}
}

func TestIntegration_GetTemplates_ReturnsLatestWorkout(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	first := mustCreateWorkout(t, router, token, fmt.Sprintf(`{"template_name":"Latest Check","started_at":"2024-05-01T09:00:00Z","completed_at":"2024-05-01T10:00:00Z","exercises":[{"exercise_id":%q,"sets":[{"reps":5,"weight_grams":50000}]}]}`, benchPressId))
	second := mustCreateWorkout(t, router, token, fmt.Sprintf(`{"template_id":%q,"started_at":"2024-05-08T09:00:00Z","completed_at":"2024-05-08T10:00:00Z","exercises":[{"exercise_id":%q,"sets":[{"reps":5,"weight_grams":52500},{"reps":4,"weight_grams":52500}]},{"exercise_id":%q,"sets":[{"reps":10,"weight_grams":20000}]}]}`, first.TemplateId, benchPressId, dumbbellBenchPressId))

	req := httptest.NewRequest(http.MethodGet, "/templates", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	var body template.TemplatesResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode templates response: %v", err)
	}

	var latest *template.LatestWorkoutResponse
	for _, tmpl := range body.Templates {
		if tmpl.TemplateId == first.TemplateId {
			latest = tmpl.LatestWorkout
		}
	}
	if latest == nil {
		t.Fatalf("expected template %v with a latest workout, got %+v", first.TemplateId, body.Templates)
	}
	if latest.WorkoutId != second.WorkoutId {
		t.Errorf("expected latest workout %v, got %v", second.WorkoutId, latest.WorkoutId)
	}
	if !reflect.DeepEqual(latest.Exercises, second.Exercises) {
		t.Errorf("expected latest workout exercises %+v, got %+v", second.Exercises, latest.Exercises)
	}
}

func TestIntegration_GetTemplates_ScopedToUser(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "carol", "secret")

	got := mustGetTemplates(t, router, token)

	// carol's seeded templates, and none of alice's: Carol Legs has a workout,
	// so it comes before the never-used Carol Back.
	if len(got.Templates) != 2 {
		t.Fatalf("expected exactly carol's 2 templates, got %+v", got.Templates)
	}
	legs, back := got.Templates[0], got.Templates[1]
	if legs.TemplateId != testutil.Id(10) || legs.TemplateName != "Carol Legs" || legs.LatestWorkout == nil || legs.LatestWorkout.WorkoutId != testutil.Id(10) {
		t.Errorf("expected Carol Legs (10) with latest workout 10 first, got %+v", legs)
	}
	if back.TemplateId != testutil.Id(11) || back.TemplateName != "Carol Back" || back.LatestWorkout != nil {
		t.Errorf("expected never-used Carol Back (11) second, got %+v", back)
	}
}

func TestIntegration_GetTemplates_OrderFollowsLatestWorkout(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	// Far-future dates keep these two templates ahead of anything other tests log.
	logWorkout := func(templateRef, completedAt string) workout.WorkoutResponse {
		t.Helper()
		body := fmt.Sprintf(`{%s,"started_at":%q,"completed_at":%q,"exercises":[{"exercise_id":%q,"sets":[{"reps":5,"weight_grams":50000}]}]}`,
			templateRef, completedAt, completedAt, benchPressId)
		return mustCreateWorkout(t, router, token, body)
	}
	indexOf := func(templates template.TemplatesResponse, templateId uuid.UUID) int {
		for i, tmpl := range templates.Templates {
			if tmpl.TemplateId == templateId {
				return i
			}
		}
		t.Fatalf("template %v missing from GET /templates: %+v", templateId, templates.Templates)
		return -1
	}

	a := logWorkout(`"template_name":"Order Check A"`, "2099-01-01T10:00:00Z")
	b := logWorkout(`"template_name":"Order Check B"`, "2099-01-02T10:00:00Z")

	before := mustGetTemplates(t, router, token)
	if indexOf(before, b.TemplateId) > indexOf(before, a.TemplateId) {
		t.Errorf("expected B (performed later) before A, got %+v", before.Templates)
	}

	again := logWorkout(fmt.Sprintf(`"template_id":%q`, a.TemplateId), "2099-01-03T10:00:00Z")

	after := mustGetTemplates(t, router, token)
	if indexOf(after, a.TemplateId) > indexOf(after, b.TemplateId) {
		t.Errorf("expected A to move ahead of B after logging a new workout, got %+v", after.Templates)
	}
	if latest := after.Templates[indexOf(after, a.TemplateId)].LatestWorkout; latest == nil || latest.WorkoutId != again.WorkoutId {
		t.Errorf("expected A's latest workout to be %v, got %+v", again.WorkoutId, latest)
	}
}

func TestIntegration_GetTemplates_NoToken(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/templates", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Result().StatusCode)
	}
}

func mustGetTemplates(t *testing.T, router http.Handler, token string) template.TemplatesResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/templates", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected GET /templates status 200, got %d", res.StatusCode)
	}

	var body template.TemplatesResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode templates response: %v", err)
	}
	return body
}

func listWorkoutsByTemplate(router http.Handler, token string, templateId any) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/workouts?template_id=%v", templateId), nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func mustListWorkoutsByTemplate(t *testing.T, router http.Handler, token string, templateId uuid.UUID) workout.WorkoutsResponse {
	t.Helper()

	rec := listWorkoutsByTemplate(router, token, templateId)
	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("list workouts by template failed: expected status 200, got %d", res.StatusCode)
	}

	var got workout.WorkoutsResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode workouts response: %v", err)
	}

	return got
}

func TestIntegration_GetWorkoutsByTemplate(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	older := mustCreateWorkout(t, router, token, createWorkoutBodyAt("2037-05-01T10:00:00Z"))
	newer := mustCreateWorkout(t, router, token, createWorkoutBodyAt("2037-05-02T10:00:00Z"))
	other := mustCreateWorkout(t, router, token, fmt.Sprintf(`{
		"template_name": "Integration By Template",
		"started_at": "2037-05-03T09:00:00Z",
		"completed_at": "2037-05-03T10:00:00Z",
		"exercises": [{"exercise_id": %q, "sets": [{"reps": 8, "weight_grams": 60000}]}]
	}`, benchPressId))

	list := mustListWorkoutsByTemplate(t, router, token, testutil.Id(1))

	olderAt, newerAt := indexOfWorkout(list, older.WorkoutId), indexOfWorkout(list, newer.WorkoutId)
	if olderAt == -1 || newerAt == -1 || indexOfWorkout(list, testutil.Id(1)) == -1 {
		t.Fatalf("expected template 1's workouts to be listed, got %+v", list.Workouts)
	}
	if newerAt > olderAt {
		t.Errorf("expected the newer workout before the older one, got positions %d and %d", newerAt, olderAt)
	}
	if indexOfWorkout(list, other.WorkoutId) != -1 {
		t.Errorf("expected the workout under another template to be excluded, got %+v", list.Workouts)
	}
	for _, w := range list.Workouts {
		if w.TemplateId != testutil.Id(1) || w.TemplateName != "Push Day" {
			t.Errorf("expected only template {1 Push Day}, got %+v", w)
		}
	}

	otherList := mustListWorkoutsByTemplate(t, router, token, other.TemplateId)
	if len(otherList.Workouts) != 1 || otherList.Workouts[0].WorkoutId != other.WorkoutId {
		t.Errorf("expected only workout %v under the new template, got %+v", other.WorkoutId, otherList.Workouts)
	}
}

func TestIntegration_GetWorkoutsByTemplate_ResponseContainsOnlyExpectedFields(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	rec := listWorkoutsByTemplate(router, token, testutil.Id(1))

	var raw map[string][]map[string]any
	if err := json.NewDecoder(rec.Result().Body).Decode(&raw); err != nil {
		t.Fatalf("failed to decode workouts response: %v", err)
	}

	if len(raw["workouts"]) == 0 {
		t.Fatal("expected at least the seeded workout to be listed")
	}
	for _, item := range raw["workouts"] {
		if len(item) != 5 {
			t.Errorf("expected exactly workout_id, template_id, template_name, started_at and completed_at, got %v", item)
		}
	}
}

func TestIntegration_GetWorkoutsByTemplate_EmptyTemplate(t *testing.T) {
	router := newTestRouter(t)
	token := mustLogin(t, router, "alice", "secret")

	req := httptest.NewRequest(http.MethodPost, "/templates", strings.NewReader(`{"template_name": "Never Used"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Result().StatusCode != http.StatusCreated {
		t.Fatalf("create template failed: expected status 201, got %d", rec.Result().StatusCode)
	}
	var created template.TemplateResponse
	if err := json.NewDecoder(rec.Result().Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode template response: %v", err)
	}

	list := mustListWorkoutsByTemplate(t, router, token, created.TemplateId)
	if list.Workouts == nil || len(list.Workouts) != 0 {
		t.Errorf("expected an empty array, got %#v", list.Workouts)
	}
}

func TestIntegration_GetWorkoutsByTemplate_Errors(t *testing.T) {
	router := newTestRouter(t)
	aliceToken := mustLogin(t, router, "alice", "secret")
	bobToken := mustLogin(t, router, "bob", "secret")

	tests := []struct {
		name       string
		token      string
		templateId any
		wantStatus int
	}{
		{"no token", "", testutil.Id(1), http.StatusUnauthorized},
		{"other user's template", bobToken, testutil.Id(1), http.StatusNotFound},
		{"unknown template", aliceToken, uuid.NewV7(), http.StatusNotFound},
		{"numeric id", aliceToken, 1, http.StatusBadRequest},
		{"invalid id", aliceToken, "abc", http.StatusBadRequest},
		{"empty id", aliceToken, "", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := listWorkoutsByTemplate(router, tt.token, tt.templateId)
			if rec.Result().StatusCode != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rec.Result().StatusCode)
			}
		})
	}
}
