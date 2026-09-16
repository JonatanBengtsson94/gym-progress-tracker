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

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/auth"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/template"
)

type mockTemplateService struct {
	createTemplateFunc func(ctx context.Context, tmpl template.Template) (template.Template, error)
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
	req = req.WithContext(auth.ContextWithUserId(req.Context(), 1))
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
	req = req.WithContext(auth.ContextWithUserId(req.Context(), 1))
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
	req = req.WithContext(auth.ContextWithUserId(req.Context(), 1))
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
	req = req.WithContext(auth.ContextWithUserId(req.Context(), 1))
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
	req = req.WithContext(auth.ContextWithUserId(req.Context(), 1))
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
	req = req.WithContext(auth.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	handler.CreateTemplate(rec, req)

	if rec.Result().StatusCode != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", rec.Result().StatusCode)
	}
}
