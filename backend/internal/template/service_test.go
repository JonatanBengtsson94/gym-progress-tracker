package template_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/template"
)

type mockTemplateRepository struct {
	createTemplateFunc func(ctx context.Context, tmpl template.Template) (template.Template, error)
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
