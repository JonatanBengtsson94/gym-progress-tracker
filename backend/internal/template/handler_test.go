package template_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/identity"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/set"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/template"
)

type mockTemplateService struct {
	getTemplatesFunc   func(ctx context.Context, userId uint32) ([]template.TemplateWithLatestWorkout, error)
	createTemplateFunc func(ctx context.Context, tmpl template.Template) (template.Template, error)
}

func (m *mockTemplateService) GetTemplates(ctx context.Context, userId uint32) ([]template.TemplateWithLatestWorkout, error) {
	return m.getTemplatesFunc(ctx, userId)
}

func (m *mockTemplateService) CreateTemplate(ctx context.Context, tmpl template.Template) (template.Template, error) {
	return m.createTemplateFunc(ctx, tmpl)
}

func TestTemplateHandler_CreateTemplate_Success(t *testing.T) {
	created := template.Template{TemplateId: 1, TemplateName: "Pull Day", UserId: 1}

	var got template.Template
	service := &mockTemplateService{
		createTemplateFunc: func(ctx context.Context, tmpl template.Template) (template.Template, error) {
			got = tmpl
			return created, nil
		},
	}

	handler := template.NewTemplateHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/templates", strings.NewReader(`{"template_name":"Pull Day"}`))
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateTemplate(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", ct)
	}
	if got.TemplateName != "Pull Day" || got.UserId != 1 {
		t.Errorf("expected service called with {Pull Day, UserId:1}, got %+v", got)
	}

	var body template.TemplateResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}
	want := template.TemplateResponse{TemplateId: 1, TemplateName: "Pull Day"}
	if body != want {
		t.Errorf("CreateTemplate() response = %+v, want %+v", body, want)
	}
}

func TestTemplateHandler_CreateTemplate_ResponseContainsOnlyExpectedFields(t *testing.T) {
	created := template.Template{TemplateId: 1, TemplateName: "Pull Day", UserId: 1}
	service := &mockTemplateService{
		createTemplateFunc: func(ctx context.Context, tmpl template.Template) (template.Template, error) {
			return created, nil
		},
	}

	handler := template.NewTemplateHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/templates", strings.NewReader(`{"template_name":"Pull Day"}`))
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateTemplate(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	var got map[string]any
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	want := map[string]any{
		"template_id":   float64(1),
		"template_name": "Pull Day",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CreateTemplate() response fields = %v, want exactly %v", got, want)
	}
}

func TestTemplateHandler_CreateTemplate_Unauthorized(t *testing.T) {
	service := &mockTemplateService{
		createTemplateFunc: func(ctx context.Context, tmpl template.Template) (template.Template, error) {
			t.Fatal("CreateTemplate should not be called without an authenticated user")
			return template.Template{}, nil
		},
	}

	handler := template.NewTemplateHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/templates", strings.NewReader(`{"template_name":"Pull Day"}`))
	rec := httptest.NewRecorder()

	handler.CreateTemplate(rec, req)

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestTemplateHandler_CreateTemplate_MalformedBody(t *testing.T) {
	service := &mockTemplateService{
		createTemplateFunc: func(ctx context.Context, tmpl template.Template) (template.Template, error) {
			t.Fatal("CreateTemplate should not be called for a malformed request body")
			return template.Template{}, nil
		},
	}

	handler := template.NewTemplateHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/templates", strings.NewReader(`not-json`))
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateTemplate(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestTemplateHandler_CreateTemplate_NameRequired(t *testing.T) {
	service := &mockTemplateService{
		createTemplateFunc: func(ctx context.Context, tmpl template.Template) (template.Template, error) {
			return template.Template{}, template.ErrTemplateNameRequired
		},
	}

	handler := template.NewTemplateHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/templates", strings.NewReader(`{"template_name":"   "}`))
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateTemplate(rec, req)

	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rec.Result().StatusCode)
	}
}

func TestTemplateHandler_CreateTemplate_AlreadyExists(t *testing.T) {
	service := &mockTemplateService{
		createTemplateFunc: func(ctx context.Context, tmpl template.Template) (template.Template, error) {
			return template.Template{}, template.ErrTemplateAlreadyExists
		},
	}

	handler := template.NewTemplateHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/templates", strings.NewReader(`{"template_name":"Pull Day"}`))
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateTemplate(rec, req)

	if rec.Result().StatusCode != http.StatusConflict {
		t.Fatalf("Expected status 409, got %d", rec.Result().StatusCode)
	}
}

func TestTemplateHandler_CreateTemplate_ServiceError(t *testing.T) {
	service := &mockTemplateService{
		createTemplateFunc: func(ctx context.Context, tmpl template.Template) (template.Template, error) {
			return template.Template{}, errors.New("db exploded")
		},
	}

	handler := template.NewTemplateHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/templates", strings.NewReader(`{"template_name":"Pull Day"}`))
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateTemplate(rec, req)

	if rec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", rec.Result().StatusCode)
	}
}

func newGetTemplatesRequest(userId uint32, authenticated bool) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/templates", nil)
	if authenticated {
		req = req.WithContext(identity.ContextWithUserId(req.Context(), userId))
	}
	return req
}

func TestTemplateHandler_GetTemplates_Success(t *testing.T) {
	startedAt := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	completedAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	serviceTemplates := []template.TemplateWithLatestWorkout{
		{Template: template.Template{TemplateId: 2, TemplateName: "Pull Day", UserId: 1}},
		{
			Template: template.Template{TemplateId: 1, TemplateName: "Push Day", UserId: 1},
			LatestWorkout: &template.LatestWorkout{
				WorkoutId:   7,
				StartedAt:   startedAt,
				CompletedAt: completedAt,
				Sets: []set.Set{
					{Exercise: exercise.Exercise{ExerciseId: 1, ExerciseName: "Bench Press"}, Reps: 8, WeightGrams: 60000},
					{Exercise: exercise.Exercise{ExerciseId: 2, ExerciseName: "Overhead Press"}, Reps: 10, WeightGrams: 30000},
					{Exercise: exercise.Exercise{ExerciseId: 1, ExerciseName: "Bench Press"}, Reps: 6, WeightGrams: 65000},
				},
			},
		},
	}

	var gotUserId uint32
	service := &mockTemplateService{
		getTemplatesFunc: func(ctx context.Context, userId uint32) ([]template.TemplateWithLatestWorkout, error) {
			gotUserId = userId
			return serviceTemplates, nil
		},
	}

	handler := template.NewTemplateHandler(service)

	rec := httptest.NewRecorder()
	handler.GetTemplates(rec, newGetTemplatesRequest(1, true))

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", ct)
	}
	if gotUserId != 1 {
		t.Errorf("expected service to receive userId 1, got %d", gotUserId)
	}

	var body template.TemplatesResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}
	want := template.TemplatesResponse{Templates: []template.TemplateWithLatestWorkoutResponse{
		{TemplateId: 2, TemplateName: "Pull Day"},
		{
			TemplateId:   1,
			TemplateName: "Push Day",
			LatestWorkout: &template.LatestWorkoutResponse{
				WorkoutId:   7,
				StartedAt:   startedAt,
				CompletedAt: completedAt,
				Exercises: []set.ExerciseSetsResponse{
					{ExerciseId: 1, ExerciseName: "Bench Press", Sets: []set.SetResponse{{Reps: 8, WeightGrams: 60000}, {Reps: 6, WeightGrams: 65000}}},
					{ExerciseId: 2, ExerciseName: "Overhead Press", Sets: []set.SetResponse{{Reps: 10, WeightGrams: 30000}}},
				},
			},
		},
	}}
	if !reflect.DeepEqual(body, want) {
		t.Errorf("GetTemplates() response = %+v, want %+v", body, want)
	}
}

func TestTemplateHandler_GetTemplates_ResponseContainsOnlyExpectedFields(t *testing.T) {
	service := &mockTemplateService{
		getTemplatesFunc: func(ctx context.Context, userId uint32) ([]template.TemplateWithLatestWorkout, error) {
			return []template.TemplateWithLatestWorkout{
				{
					Template: template.Template{TemplateId: 1, TemplateName: "Push Day", UserId: 1},
					LatestWorkout: &template.LatestWorkout{
						WorkoutId:   7,
						StartedAt:   time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
						CompletedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
						Sets: []set.Set{
							{Exercise: exercise.Exercise{ExerciseId: 1, ExerciseName: "Bench Press", UserId: 1}, Reps: 8, WeightGrams: 60000, WorkoutId: 7},
						},
					},
				},
				{Template: template.Template{TemplateId: 2, TemplateName: "Pull Day", UserId: 1}},
			}, nil
		},
	}

	handler := template.NewTemplateHandler(service)

	rec := httptest.NewRecorder()
	handler.GetTemplates(rec, newGetTemplatesRequest(1, true))

	res := rec.Result()
	defer res.Body.Close()

	var got map[string]any
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	want := map[string]any{
		"templates": []any{
			map[string]any{
				"template_id":   float64(1),
				"template_name": "Push Day",
				"latest_workout": map[string]any{
					"workout_id":   float64(7),
					"started_at":   "2024-01-15T09:00:00Z",
					"completed_at": "2024-01-15T10:00:00Z",
					"exercises": []any{
						map[string]any{
							"exercise_id":   float64(1),
							"exercise_name": "Bench Press",
							"sets": []any{
								map[string]any{"reps": float64(8), "weight_grams": float64(60000)},
							},
						},
					},
				},
			},
			map[string]any{
				"template_id":    float64(2),
				"template_name":  "Pull Day",
				"latest_workout": nil,
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetTemplates() response fields = %v, want exactly %v", got, want)
	}
}

func TestTemplateHandler_GetTemplates_Empty(t *testing.T) {
	service := &mockTemplateService{
		getTemplatesFunc: func(ctx context.Context, userId uint32) ([]template.TemplateWithLatestWorkout, error) {
			return []template.TemplateWithLatestWorkout{}, nil
		},
	}

	handler := template.NewTemplateHandler(service)

	rec := httptest.NewRecorder()
	handler.GetTemplates(rec, newGetTemplatesRequest(1, true))

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", res.StatusCode)
	}

	var got map[string]any
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}
	want := map[string]any{"templates": []any{}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetTemplates() response = %v, want %v", got, want)
	}
}

func TestTemplateHandler_GetTemplates_Unauthorized(t *testing.T) {
	service := &mockTemplateService{
		getTemplatesFunc: func(ctx context.Context, userId uint32) ([]template.TemplateWithLatestWorkout, error) {
			t.Fatal("GetTemplates should not be called without an authenticated user")
			return nil, nil
		},
	}

	handler := template.NewTemplateHandler(service)

	rec := httptest.NewRecorder()
	handler.GetTemplates(rec, newGetTemplatesRequest(0, false))

	if rec.Result().StatusCode != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", rec.Result().StatusCode)
	}
}

func TestTemplateHandler_GetTemplates_ServiceError(t *testing.T) {
	service := &mockTemplateService{
		getTemplatesFunc: func(ctx context.Context, userId uint32) ([]template.TemplateWithLatestWorkout, error) {
			return nil, errors.New("db exploded")
		},
	}

	handler := template.NewTemplateHandler(service)

	rec := httptest.NewRecorder()
	handler.GetTemplates(rec, newGetTemplatesRequest(1, true))

	if rec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", rec.Result().StatusCode)
	}
}
