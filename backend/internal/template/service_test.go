package template_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/template"
)

type mockTemplateRepository struct {
	getTemplatesFunc   func(ctx context.Context, userId uint32) ([]template.TemplateWithLatestWorkout, error)
	createTemplateFunc func(ctx context.Context, tmpl template.Template) (template.Template, error)
}

func (m *mockTemplateRepository) GetTemplatesByUserId(ctx context.Context, userId uint32) ([]template.TemplateWithLatestWorkout, error) {
	return m.getTemplatesFunc(ctx, userId)
}

func (m *mockTemplateRepository) CreateTemplate(ctx context.Context, tmpl template.Template) (template.Template, error) {
	return m.createTemplateFunc(ctx, tmpl)
}

func TestTemplateService_CreateTemplate(t *testing.T) {
	ctx := t.Context()
	input := template.Template{TemplateName: "Pull Day", UserId: 42}
	created := template.Template{TemplateId: 1, TemplateName: "Pull Day", UserId: 42}

	var gotTemplate template.Template
	repo := &mockTemplateRepository{
		createTemplateFunc: func(ctx context.Context, tmpl template.Template) (template.Template, error) {
			gotTemplate = tmpl
			return created, nil
		},
	}

	service := template.NewTemplateService(repo)

	result, err := service.CreateTemplate(ctx, input)
	if err != nil {
		t.Fatalf("CreateTemplate returned error: %v", err)
	}

	if gotTemplate != input {
		t.Errorf("expected repo to receive %+v, got %+v", input, gotTemplate)
	}
	if result != created {
		t.Errorf("CreateTemplate() = %+v, want %+v", result, created)
	}
}

func TestTemplateService_CreateTemplate_NameRequired(t *testing.T) {
	ctx := t.Context()

	repo := &mockTemplateRepository{
		createTemplateFunc: func(ctx context.Context, tmpl template.Template) (template.Template, error) {
			t.Fatal("CreateTemplate should not be called for an empty template_name")
			return template.Template{}, nil
		},
	}

	service := template.NewTemplateService(repo)

	_, err := service.CreateTemplate(ctx, template.Template{TemplateName: "   ", UserId: 42})
	if !errors.Is(err, template.ErrTemplateNameRequired) {
		t.Fatalf("Expected ErrTemplateNameRequired, got %v", err)
	}
}

func TestTemplateService_CreateTemplate_RepoError(t *testing.T) {
	ctx := t.Context()
	wantErr := template.ErrTemplateAlreadyExists

	repo := &mockTemplateRepository{
		createTemplateFunc: func(ctx context.Context, tmpl template.Template) (template.Template, error) {
			return template.Template{}, wantErr
		},
	}

	service := template.NewTemplateService(repo)

	_, err := service.CreateTemplate(ctx, template.Template{TemplateName: "Pull Day", UserId: 42})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected error to wrap %v, got %v", wantErr, err)
	}
}

func TestTemplateService_GetTemplates(t *testing.T) {
	ctx := t.Context()
	templates := []template.TemplateWithLatestWorkout{
		{Template: template.Template{TemplateId: 1, TemplateName: "Push Day", UserId: 42}, LatestWorkout: &template.LatestWorkout{WorkoutId: 3}},
	}

	var gotUserId uint32
	repo := &mockTemplateRepository{
		getTemplatesFunc: func(ctx context.Context, userId uint32) ([]template.TemplateWithLatestWorkout, error) {
			gotUserId = userId
			return templates, nil
		},
	}

	service := template.NewTemplateService(repo)

	result, err := service.GetTemplates(ctx, 42)
	if err != nil {
		t.Fatalf("GetTemplates returned error: %v", err)
	}

	if gotUserId != 42 {
		t.Errorf("expected repo to receive userId 42, got %d", gotUserId)
	}
	if !reflect.DeepEqual(result, templates) {
		t.Errorf("GetTemplates() = %+v, want %+v", result, templates)
	}
}

func TestTemplateService_GetTemplates_RepoError(t *testing.T) {
	ctx := t.Context()
	wantErr := errors.New("db exploded")

	repo := &mockTemplateRepository{
		getTemplatesFunc: func(ctx context.Context, userId uint32) ([]template.TemplateWithLatestWorkout, error) {
			return nil, wantErr
		},
	}

	service := template.NewTemplateService(repo)

	_, err := service.GetTemplates(ctx, 42)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected error to wrap %v, got %v", wantErr, err)
	}
}
