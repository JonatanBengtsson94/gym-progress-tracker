package template_test

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/testutil"
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

func TestTemplateRepository_CreateTemplate(t *testing.T) {
	ctx := t.Context()
	repo := template.NewPostgresTemplateRepository(testPool)

	created, err := repo.CreateTemplate(ctx, template.Template{TemplateName: "Pull Day", UserId: 1})
	if err != nil {
		t.Fatalf("CreateTemplate returned error: %v", err)
	}

	if created.TemplateId == 0 {
		t.Error("expected a non-zero TemplateId to be assigned")
	}
	if created.TemplateName != "Pull Day" {
		t.Errorf("CreateTemplate() = %+v, want TemplateName=Pull Day", created)
	}
}

func TestTemplateRepository_CreateTemplate_DuplicateName(t *testing.T) {
	ctx := t.Context()
	repo := template.NewPostgresTemplateRepository(testPool)

	// "Custom Test Template" is already seeded for user 1.
	_, err := repo.CreateTemplate(ctx, template.Template{TemplateName: "Custom Test Template", UserId: 1})
	if !errors.Is(err, template.ErrTemplateAlreadyExists) {
		t.Fatalf("Expected ErrTemplateAlreadyExists, got %v", err)
	}
}

func TestTemplateRepository_CreateTemplate_DuplicateName_CaseInsensitive(t *testing.T) {
	ctx := t.Context()
	repo := template.NewPostgresTemplateRepository(testPool)

	_, err := repo.CreateTemplate(ctx, template.Template{TemplateName: "custom test template", UserId: 1})
	if !errors.Is(err, template.ErrTemplateAlreadyExists) {
		t.Fatalf("Expected ErrTemplateAlreadyExists for a case-insensitive duplicate, got %v", err)
	}
}

func TestTemplateRepository_CreateTemplate_UserNotFound(t *testing.T) {
	ctx := t.Context()
	repo := template.NewPostgresTemplateRepository(testPool)

	_, err := repo.CreateTemplate(ctx, template.Template{TemplateName: "Ghost Template", UserId: 999999})
	if err == nil {
		t.Fatal("expected an error for a nonexistent user_id, got nil")
	}
}

func TestTemplateRepository_CreateTemplate_DifferentUsersCanShareName(t *testing.T) {
	ctx := t.Context()
	repo := template.NewPostgresTemplateRepository(testPool)

	// "Custom Test Template" is seeded for user 1; user 2 should be free to use the same name.
	created, err := repo.CreateTemplate(ctx, template.Template{TemplateName: "Custom Test Template", UserId: 2})
	if err != nil {
		t.Fatalf("CreateTemplate returned error: %v", err)
	}
	if created.TemplateName != "Custom Test Template" {
		t.Errorf("CreateTemplate() = %+v, want TemplateName=Custom Test Template", created)
	}
}
