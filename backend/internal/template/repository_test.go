package template_test

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

// The ids of the global exercises the seeded workouts use.
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

func TestTemplateRepository_CreateTemplate(t *testing.T) {
	ctx := t.Context()
	repo := template.NewPostgresTemplateRepository(testPool)

	toCreate := template.Template{TemplateId: uuid.NewV7(), TemplateName: "Pull Day", UserId: 1}
	created, isNew, err := repo.CreateTemplate(ctx, toCreate)
	if err != nil {
		t.Fatalf("CreateTemplate returned error: %v", err)
	}

	if created != toCreate || !isNew {
		t.Errorf("CreateTemplate() = %+v, %v, want %+v, true", created, isNew, toCreate)
	}
}

func TestTemplateRepository_CreateTemplate_Retried(t *testing.T) {
	ctx := t.Context()
	repo := template.NewPostgresTemplateRepository(testPool)

	original := template.Template{TemplateId: uuid.NewV7(), TemplateName: "Retried Day", UserId: 1}
	if _, _, err := repo.CreateTemplate(ctx, original); err != nil {
		t.Fatalf("CreateTemplate returned error: %v", err)
	}

	// A retry gets back what the first attempt stored, even if its name has changed since.
	for _, name := range []string{"Retried Day", "Renamed Day"} {
		retried, isNew, err := repo.CreateTemplate(ctx, template.Template{TemplateId: original.TemplateId, TemplateName: name, UserId: 1})
		if err != nil {
			t.Fatalf("CreateTemplate returned error: %v", err)
		}
		if retried != original || isNew {
			t.Errorf("retry as %q: CreateTemplate() = %+v, %v, want %+v, false", name, retried, isNew, original)
		}
	}
}

func TestTemplateRepository_CreateTemplate_ReturnsExistingTemplateWithSameName(t *testing.T) {
	ctx := t.Context()
	repo := template.NewPostgresTemplateRepository(testPool)

	// "Custom Test Template" is seeded for user 1.
	want := template.Template{TemplateId: testutil.Id(1), TemplateName: "Custom Test Template", UserId: 1}
	for _, name := range []string{"Custom Test Template", "custom test template"} {
		t.Run(name, func(t *testing.T) {
			requestedId := uuid.NewV7()
			got, isNew, err := repo.CreateTemplate(ctx, template.Template{TemplateId: requestedId, TemplateName: name, UserId: 1})
			if err != nil {
				t.Fatalf("CreateTemplate returned error: %v", err)
			}
			if got != want || isNew {
				t.Errorf("CreateTemplate() = %+v, %v, want %+v, false", got, isNew, want)
			}

			var stored int
			if err := testPool.QueryRow(ctx, `SELECT count(*) FROM templates WHERE template_id = $1`, requestedId).Scan(&stored); err != nil {
				t.Fatalf("failed to count templates: %v", err)
			}
			if stored != 0 {
				t.Errorf("expected nothing to be stored under the requested id %v", requestedId)
			}
		})
	}
}

// A retry must get back the template it created, even when its name now belongs
// to another template, or the client would swap its id to the wrong one.
func TestTemplateRepository_CreateTemplate_RetryWinsOverSameName(t *testing.T) {
	ctx := t.Context()
	repo := template.NewPostgresTemplateRepository(testPool)

	// Stored after the template holding the name, so only the lookup's order can pick it.
	first := template.Template{TemplateId: uuid.NewV7(), TemplateName: "Upper A", UserId: 1}
	second := template.Template{TemplateId: uuid.NewV7(), TemplateName: "Upper B", UserId: 1}
	for _, tmpl := range []template.Template{second, first} {
		if _, _, err := repo.CreateTemplate(ctx, tmpl); err != nil {
			t.Fatalf("CreateTemplate returned error: %v", err)
		}
	}

	got, isNew, err := repo.CreateTemplate(ctx, template.Template{TemplateId: first.TemplateId, TemplateName: "upper b", UserId: 1})
	if err != nil {
		t.Fatalf("CreateTemplate returned error: %v", err)
	}
	if got != first || isNew {
		t.Errorf("CreateTemplate() = %+v, %v, want the retried template %+v, false", got, isNew, first)
	}
}

func TestTemplateRepository_CreateTemplate_IdOfAnotherUsersTemplate(t *testing.T) {
	ctx := t.Context()
	repo := template.NewPostgresTemplateRepository(testPool)

	// Template 101 is seeded for user 3.
	_, _, err := repo.CreateTemplate(ctx, template.Template{TemplateId: testutil.Id(101), TemplateName: "Taken Id Template", UserId: 1})
	if !errors.Is(err, template.ErrTemplateIdTaken) {
		t.Fatalf("Expected ErrTemplateIdTaken, got %v", err)
	}
}

func TestTemplateRepository_CreateTemplate_UserNotFound(t *testing.T) {
	ctx := t.Context()
	repo := template.NewPostgresTemplateRepository(testPool)

	_, _, err := repo.CreateTemplate(ctx, template.Template{TemplateId: uuid.NewV7(), TemplateName: "Ghost Template", UserId: 999999})
	if err == nil {
		t.Fatal("expected an error for a nonexistent user_id, got nil")
	}
}

func TestTemplateRepository_CreateTemplate_DifferentUsersCanShareName(t *testing.T) {
	ctx := t.Context()
	repo := template.NewPostgresTemplateRepository(testPool)

	// "Custom Test Template" is seeded for user 1; user 2 should be free to use the same name.
	toCreate := template.Template{TemplateId: uuid.NewV7(), TemplateName: "Custom Test Template", UserId: 2}
	created, isNew, err := repo.CreateTemplate(ctx, toCreate)
	if err != nil {
		t.Fatalf("CreateTemplate returned error: %v", err)
	}
	if created != toCreate || !isNew {
		t.Errorf("CreateTemplate() = %+v, %v, want %+v, true", created, isNew, toCreate)
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
	wantIds := []uuid.UUID{testutil.Id(101), testutil.Id(103), testutil.Id(104), testutil.Id(102)}
	for i, tmpl := range templates {
		if tmpl.TemplateName != wantNames[i] || tmpl.TemplateId != wantIds[i] {
			t.Errorf("templates[%d] = (%v, %q), want (%v, %q)", i, tmpl.TemplateId, tmpl.TemplateName, wantIds[i], wantNames[i])
		}
	}

	pushDay, pullDay, armDay, legDay := templates[0], templates[1], templates[2], templates[3]

	if armDay.LatestWorkout != nil {
		t.Errorf("expected Arm Day without a latest workout, got %+v", armDay.LatestWorkout)
	}
	if legDay.LatestWorkout != nil {
		t.Errorf("expected Leg Day without a latest workout, got %+v", legDay.LatestWorkout)
	}

	if pullDay.LatestWorkout == nil || pullDay.LatestWorkout.WorkoutId != testutil.Id(204) || len(pullDay.LatestWorkout.Sets) != 1 {
		t.Fatalf("expected Pull Day's latest workout to be 204 with 1 set, got %+v", pullDay.LatestWorkout)
	}

	latest := pushDay.LatestWorkout
	if latest == nil {
		t.Fatal("expected Push Day to have a latest workout")
	}
	// 202 and 203 completed at the same time; the higher id wins the tie.
	if latest.WorkoutId != testutil.Id(203) {
		t.Errorf("expected Push Day's latest workout to be 203, got %v", latest.WorkoutId)
	}
	wantStarted := time.Date(2024, 1, 20, 9, 30, 0, 0, time.UTC)
	wantCompleted := time.Date(2024, 1, 20, 10, 0, 0, 0, time.UTC)
	if !latest.StartedAt.Equal(wantStarted) || !latest.CompletedAt.Equal(wantCompleted) {
		t.Errorf("unexpected times: started %v, completed %v", latest.StartedAt, latest.CompletedAt)
	}

	// Sets come back in the order they were logged.
	type setKey struct {
		exerciseId  uuid.UUID
		reps        uint8
		weightGrams uint32
	}
	wantSets := []setKey{{dumbbellBenchPressId, 10, 20000}, {benchPressId, 8, 60000}, {dumbbellBenchPressId, 9, 20000}}
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
