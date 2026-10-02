package template_test

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/testutil"
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

func TestTemplateRepository_GetTemplatesByUserId(t *testing.T) {
	ctx := t.Context()
	repo := template.NewPostgresTemplateRepository(testPool)

	templates, err := repo.GetTemplatesByUserId(ctx, 3)
	if err != nil {
		t.Fatalf("GetTemplatesByUserId returned error: %v", err)
	}

	// Most recently performed first, then never-used templates by name, ignoring case.
	wantNames := []string{"push Day", "Pull Day", "Arm Day", "Leg Day"}
	if len(templates) != len(wantNames) {
		t.Fatalf("expected %d templates, got %+v", len(wantNames), templates)
	}
	// Checking ids as well as names catches another user's template with a
	// matching name sneaking into the list.
	wantIds := []uint32{101, 103, 104, 102}
	for i, tmpl := range templates {
		if tmpl.TemplateName != wantNames[i] || tmpl.TemplateId != wantIds[i] {
			t.Errorf("templates[%d] = (%d, %q), want (%d, %q)", i, tmpl.TemplateId, tmpl.TemplateName, wantIds[i], wantNames[i])
		}
	}

	pushDay, pullDay, armDay, legDay := templates[0], templates[1], templates[2], templates[3]

	if armDay.LatestWorkout != nil {
		t.Errorf("expected Arm Day without a latest workout, got %+v", armDay.LatestWorkout)
	}
	if legDay.LatestWorkout != nil {
		t.Errorf("expected Leg Day without a latest workout, got %+v", legDay.LatestWorkout)
	}

	if pullDay.LatestWorkout == nil || pullDay.LatestWorkout.WorkoutId != 204 || len(pullDay.LatestWorkout.Sets) != 1 {
		t.Fatalf("expected Pull Day's latest workout to be 204 with 1 set, got %+v", pullDay.LatestWorkout)
	}

	latest := pushDay.LatestWorkout
	if latest == nil {
		t.Fatal("expected Push Day to have a latest workout")
	}
	// 202 and 203 completed at the same time; the higher id wins the tie.
	if latest.WorkoutId != 203 {
		t.Errorf("expected Push Day's latest workout to be 203, got %d", latest.WorkoutId)
	}
	wantStarted := time.Date(2024, 1, 20, 9, 30, 0, 0, time.UTC)
	wantCompleted := time.Date(2024, 1, 20, 10, 0, 0, 0, time.UTC)
	if !latest.StartedAt.Equal(wantStarted) || !latest.CompletedAt.Equal(wantCompleted) {
		t.Errorf("unexpected times: started %v, completed %v", latest.StartedAt, latest.CompletedAt)
	}

	// Sets come back in the order they were logged.
	type setKey struct {
		exerciseId  uint32
		reps        uint8
		weightGrams uint32
	}
	wantSets := []setKey{{2, 10, 20000}, {1, 8, 60000}, {2, 9, 20000}}
	if len(latest.Sets) != len(wantSets) {
		t.Fatalf("expected %d sets, got %+v", len(wantSets), latest.Sets)
	}
	for i, s := range latest.Sets {
		got := setKey{s.Exercise.ExerciseId, s.Reps, s.WeightGrams}
		if got != wantSets[i] {
			t.Errorf("Sets[%d] = %+v, want %+v", i, got, wantSets[i])
		}
		if s.Exercise.ExerciseName == "" {
			t.Errorf("Sets[%d] has an empty ExerciseName", i)
		}
	}
}

func TestTemplateRepository_GetTemplatesByUserId_NoTemplates(t *testing.T) {
	ctx := t.Context()
	repo := template.NewPostgresTemplateRepository(testPool)

	templates, err := repo.GetTemplatesByUserId(ctx, 999999)
	if err != nil {
		t.Fatalf("GetTemplatesByUserId returned error: %v", err)
	}
	if templates == nil || len(templates) != 0 {
		t.Errorf("expected a non-nil empty slice, got %#v", templates)
	}
}

func TestTemplateRepository_GetTemplatesByUserId_QueryError(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	repo := template.NewPostgresTemplateRepository(testPool)

	templates, err := repo.GetTemplatesByUserId(ctx, 3)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected an error wrapping context.Canceled, got %v", err)
	}
	if templates != nil {
		t.Errorf("expected nil templates on error, got %+v", templates)
	}
}
