package template_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/testutil"
)

type mockTemplateRepository struct {
	getTemplatesFunc   func(ctx context.Context, userId uint32) ([]template.TemplateWithLatestWorkout, error)
	createTemplateFunc func(ctx context.Context, tmpl template.Template) (template.Template, bool, error)
}

func (m *mockTemplateRepository) GetTemplatesByUserId(ctx context.Context, userId uint32) ([]template.TemplateWithLatestWorkout, error) {
	return m.getTemplatesFunc(ctx, userId)
}

func (m *mockTemplateRepository) CreateTemplate(ctx context.Context, tmpl template.Template) (template.Template, bool, error) {
	return m.createTemplateFunc(ctx, tmpl)
}

func TestTemplateService_CreateTemplate(t *testing.T) {
	ctx := t.Context()
	input := template.Template{TemplateId: testutil.Id(1), TemplateName: "Pull Day", UserId: 42}

	var gotTemplate template.Template
	repo := &mockTemplateRepository{
		createTemplateFunc: func(ctx context.Context, tmpl template.Template) (template.Template, bool, error) {
			gotTemplate = tmpl
			return tmpl, true, nil
		},
	}

	service := template.NewTemplateService(repo)

	result, created, err := service.CreateTemplate(ctx, input)
	if err != nil {
		t.Fatalf("CreateTemplate returned error: %v", err)
	}

	if gotTemplate != input {
		t.Errorf("expected repo to receive the template with its id unchanged, %+v, got %+v", input, gotTemplate)
	}
	if result != input || !created {
		t.Errorf("CreateTemplate() = %+v, %v, want %+v, true", result, created, input)
	}
}

func TestTemplateService_CreateTemplate_GeneratesIdWhenOmitted(t *testing.T) {
	ctx := t.Context()

	var got []uuid.UUID
	repo := &mockTemplateRepository{
		createTemplateFunc: func(ctx context.Context, tmpl template.Template) (template.Template, bool, error) {
			got = append(got, tmpl.TemplateId)
			return tmpl, true, nil
		},
	}

	service := template.NewTemplateService(repo)

	for range 2 {
		if _, _, err := service.CreateTemplate(ctx, template.Template{TemplateName: "Pull Day", UserId: 42}); err != nil {
			t.Fatalf("CreateTemplate returned error: %v", err)
		}
	}

	if got[0] == uuid.Nil() || got[0] == got[1] {
		t.Errorf("expected each create without an id to get a new one, got %v", got)
	}
}

func TestTemplateService_CreateTemplate_ReturnsExistingTemplate(t *testing.T) {
	ctx := t.Context()
	existing := template.Template{TemplateId: testutil.Id(2), TemplateName: "Pull Day", UserId: 42}

	repo := &mockTemplateRepository{
		createTemplateFunc: func(ctx context.Context, tmpl template.Template) (template.Template, bool, error) {
			return existing, false, nil
		},
	}

	service := template.NewTemplateService(repo)

	result, created, err := service.CreateTemplate(ctx, template.Template{TemplateId: testutil.Id(1), TemplateName: "pull day", UserId: 42})
	if err != nil {
		t.Fatalf("CreateTemplate returned error: %v", err)
	}
	if result != existing || created {
		t.Errorf("CreateTemplate() = %+v, %v, want %+v, false", result, created, existing)
	}
}

func TestTemplateService_CreateTemplate_NameRequired(t *testing.T) {
	ctx := t.Context()

	repo := &mockTemplateRepository{
		createTemplateFunc: func(ctx context.Context, tmpl template.Template) (template.Template, bool, error) {
			t.Fatal("CreateTemplate should not be called for an empty template_name")
			return template.Template{}, false, nil
		},
	}

	service := template.NewTemplateService(repo)

	_, _, err := service.CreateTemplate(ctx, template.Template{TemplateName: "   ", UserId: 42})
	if !errors.Is(err, template.ErrTemplateNameRequired) {
		t.Fatalf("Expected ErrTemplateNameRequired, got %v", err)
	}
}

func TestTemplateService_CreateTemplate_RepoError(t *testing.T) {
	ctx := t.Context()
	wantErr := template.ErrTemplateIdTaken

	repo := &mockTemplateRepository{
		createTemplateFunc: func(ctx context.Context, tmpl template.Template) (template.Template, bool, error) {
			return template.Template{}, false, wantErr
		},
	}

	service := template.NewTemplateService(repo)

	_, _, err := service.CreateTemplate(ctx, template.Template{TemplateName: "Pull Day", UserId: 42})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected error to wrap %v, got %v", wantErr, err)
	}
}

func TestTemplateService_GetTemplates(t *testing.T) {
	ctx := t.Context()
	templates := []template.TemplateWithLatestWorkout{
		{Template: template.Template{TemplateId: testutil.Id(1), TemplateName: "Push Day", UserId: 42}, LatestWorkout: &template.LatestWorkout{WorkoutId: testutil.Id(3)}},
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
