package exercise_test

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/testutil"
)

type mockExerciseRepository struct {
	getGlobalFunc      func(ctx context.Context) ([]exercise.Exercise, error)
	getExerciseFunc    func(ctx context.Context, userId uint32) ([]exercise.Exercise, error)
	createExerciseFunc func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error)
	modifyExerciseFunc func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error)
}

func (m *mockExerciseRepository) GetGlobalExercises(ctx context.Context) ([]exercise.Exercise, error) {
	return m.getGlobalFunc(ctx)
}

func (m *mockExerciseRepository) GetExercisesByUserId(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
	return m.getExerciseFunc(ctx, userId)
}

func (m *mockExerciseRepository) ModifyExercise(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
	return m.modifyExerciseFunc(ctx, ex)
}

func (m *mockExerciseRepository) CreateExercise(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
	return m.createExerciseFunc(ctx, ex)
}

func TestExerciseService_GetExercises(t *testing.T) {
	ctx := t.Context()
	const wantUserId = 42
	expected := []exercise.Exercise{
		{
			ExerciseId:   testutil.Id(1),
			ExerciseName: "Test Exercise 1",
		},
		{
			ExerciseId:   testutil.Id(2),
			ExerciseName: "Test Exercise 2",
		},
	}

	var gotUserId uint32
	repo := &mockExerciseRepository{
		getExerciseFunc: func(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
			gotUserId = userId
			return expected, nil
		},
	}

	service := exercise.NewExerciseService(repo)

	got, err := service.GetExercises(ctx, wantUserId)
	if err != nil {
		t.Fatalf("GetExercises returned error: %v", err)
	}

	if gotUserId != wantUserId {
		t.Errorf("expected repo to receive userId %d, got %d", wantUserId, gotUserId)
	}

	if len(got) != len(expected) {
		t.Fatalf("Expected %d exercises, got %d", len(expected), len(got))
	}

	for i, ex := range got {
		if ex != expected[i] {
			t.Errorf("exercise %d: got %+v, want %+v", i, ex, expected[i])
		}
	}
}

func TestExerciseService_GetExercises_SortsByNameIgnoringCase(t *testing.T) {
	ctx := t.Context()
	repo := &mockExerciseRepository{
		getExerciseFunc: func(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
			return []exercise.Exercise{
				{ExerciseId: testutil.Id(1), ExerciseName: "Squat (Barbell)"},
				{ExerciseId: testutil.Id(4), ExerciseName: "bench press"},
				{ExerciseId: testutil.Id(2), ExerciseName: "Deadlift (Barbell)"},
				{ExerciseId: testutil.Id(3), ExerciseName: "Bench Press"},
			}, nil
		},
	}

	service := exercise.NewExerciseService(repo)

	got, err := service.GetExercises(ctx, 42)
	if err != nil {
		t.Fatalf("GetExercises returned error: %v", err)
	}

	// Names equal apart from case fall back to id so the order is stable.
	wantIds := []uuid.UUID{testutil.Id(3), testutil.Id(4), testutil.Id(2), testutil.Id(1)}
	if len(got) != len(wantIds) {
		t.Fatalf("Expected %d exercises, got %d", len(wantIds), len(got))
	}
	for i, ex := range got {
		if ex.ExerciseId != wantIds[i] {
			t.Errorf("exercise %d: got id %v (%q), want id %v", i, ex.ExerciseId, ex.ExerciseName, wantIds[i])
		}
	}
}

func TestExerciseService_GetExercises_RepoError(t *testing.T) {
	ctx := t.Context()
	wantErr := errors.New("db exploded")

	repo := &mockExerciseRepository{
		getExerciseFunc: func(ctx context.Context, userId uint32) ([]exercise.Exercise, error) {
			return nil, wantErr
		},
	}

	service := exercise.NewExerciseService(repo)

	_, err := service.GetExercises(ctx, 0)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected error to wrap %v, got %v", wantErr, err)
	}
}

func TestExerciseService_GetGlobalExercises_SortsByNameIgnoringCase(t *testing.T) {
	ctx := t.Context()
	repo := &mockExerciseRepository{
		getGlobalFunc: func(ctx context.Context) ([]exercise.Exercise, error) {
			return []exercise.Exercise{
				{ExerciseId: testutil.Id(1), ExerciseName: "Squat (Barbell)"},
				{ExerciseId: testutil.Id(3), ExerciseName: "bench press"},
				{ExerciseId: testutil.Id(2), ExerciseName: "Bench Press"},
			}, nil
		},
	}

	got, err := exercise.NewExerciseService(repo).GetGlobalExercises(ctx)
	if err != nil {
		t.Fatalf("GetGlobalExercises returned error: %v", err)
	}

	wantIds := []uuid.UUID{testutil.Id(2), testutil.Id(3), testutil.Id(1)}
	if len(got) != len(wantIds) {
		t.Fatalf("Expected %d exercises, got %d", len(wantIds), len(got))
	}
	for i, ex := range got {
		if ex.ExerciseId != wantIds[i] {
			t.Errorf("exercise %d: got id %v (%q), want id %v", i, ex.ExerciseId, ex.ExerciseName, wantIds[i])
		}
	}
}

func TestExerciseService_GetGlobalExercises_RepoError(t *testing.T) {
	ctx := t.Context()
	wantErr := errors.New("db exploded")
	repo := &mockExerciseRepository{
		getGlobalFunc: func(ctx context.Context) ([]exercise.Exercise, error) {
			return nil, wantErr
		},
	}

	_, err := exercise.NewExerciseService(repo).GetGlobalExercises(ctx)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected error to wrap %v, got %v", wantErr, err)
	}
}

func TestExerciseService_CreateExercise(t *testing.T) {
	ctx := t.Context()
	input := exercise.Exercise{ExerciseId: testutil.Id(1), ExerciseName: "Lunge", UserId: 42}

	var got exercise.Exercise
	repo := &mockExerciseRepository{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
			got = ex
			return ex, true, nil
		},
	}

	service := exercise.NewExerciseService(repo)

	result, created, err := service.CreateExercise(ctx, input)
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}

	if got != input {
		t.Errorf("expected repo to receive the exercise with its id unchanged, %+v, got %+v", input, got)
	}
	if result != input || !created {
		t.Errorf("CreateExercise() = %+v, %v, want %+v, true", result, created, input)
	}
}

func TestExerciseService_CreateExercise_GeneratesIdWhenOmitted(t *testing.T) {
	ctx := t.Context()

	var got []uuid.UUID
	repo := &mockExerciseRepository{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
			got = append(got, ex.ExerciseId)
			return ex, true, nil
		},
	}

	service := exercise.NewExerciseService(repo)

	for range 2 {
		if _, _, err := service.CreateExercise(ctx, exercise.Exercise{ExerciseName: "Lunge", UserId: 42}); err != nil {
			t.Fatalf("CreateExercise returned error: %v", err)
		}
	}

	if got[0] == uuid.Nil() || got[0] == got[1] {
		t.Errorf("expected each create without an id to get a new one, got %v", got)
	}
}

func TestExerciseService_CreateExercise_ReturnsExistingExercise(t *testing.T) {
	ctx := t.Context()
	existing := exercise.Exercise{ExerciseId: testutil.Id(2), ExerciseName: "Lunge", UserId: 42}

	repo := &mockExerciseRepository{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
			return existing, false, nil
		},
	}

	service := exercise.NewExerciseService(repo)

	result, created, err := service.CreateExercise(ctx, exercise.Exercise{ExerciseId: testutil.Id(1), ExerciseName: "lunge", UserId: 42})
	if err != nil {
		t.Fatalf("CreateExercise returned error: %v", err)
	}
	if result != existing || created {
		t.Errorf("CreateExercise() = %+v, %v, want %+v, false", result, created, existing)
	}
}

func TestExerciseService_CreateExercise_NameRequired(t *testing.T) {
	ctx := t.Context()

	repo := &mockExerciseRepository{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
			t.Fatal("CreateExercise should not be called for an empty exercise_name")
			return exercise.Exercise{}, false, nil
		},
	}

	service := exercise.NewExerciseService(repo)

	_, _, err := service.CreateExercise(ctx, exercise.Exercise{ExerciseName: "   ", UserId: 42})
	if !errors.Is(err, exercise.ErrExerciseNameRequired) {
		t.Fatalf("Expected ErrExerciseNameRequired, got %v", err)
	}
}

func TestExerciseService_CreateExercise_RepoError(t *testing.T) {
	ctx := t.Context()
	wantErr := exercise.ErrExerciseIdTaken

	repo := &mockExerciseRepository{
		createExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, bool, error) {
			return exercise.Exercise{}, false, wantErr
		},
	}

	service := exercise.NewExerciseService(repo)

	_, _, err := service.CreateExercise(ctx, exercise.Exercise{ExerciseName: "Lunge", UserId: 42})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected error to wrap %v, got %v", wantErr, err)
	}
}

func TestExerciseService_ModifyExercise(t *testing.T) {
	ctx := t.Context()
	input := exercise.Exercise{ExerciseId: testutil.Id(1), ExerciseName: "Romanian Deadlift", UserId: 42}
	modified := exercise.Exercise{ExerciseId: testutil.Id(1), ExerciseName: "Romanian Deadlift", UserId: 42}

	var got exercise.Exercise
	repo := &mockExerciseRepository{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			got = ex
			return modified, nil
		},
	}

	service := exercise.NewExerciseService(repo)

	result, err := service.ModifyExercise(ctx, input)
	if err != nil {
		t.Fatalf("ModifyExercise returned error: %v", err)
	}

	if got != input {
		t.Errorf("expected repo to receive %+v, got %+v", input, got)
	}

	if result != modified {
		t.Errorf("ModifyExercise() = %+v, want %+v", result, modified)
	}
}

func TestExerciseService_ModifyExercise_NameRequired(t *testing.T) {
	ctx := t.Context()

	repo := &mockExerciseRepository{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			t.Fatal("ModifyExercise should not be called for an empty exercise_name")
			return exercise.Exercise{}, nil
		},
	}

	service := exercise.NewExerciseService(repo)

	_, err := service.ModifyExercise(ctx, exercise.Exercise{ExerciseId: testutil.Id(1), ExerciseName: "   ", UserId: 42})
	if !errors.Is(err, exercise.ErrExerciseNameRequired) {
		t.Fatalf("Expected ErrExerciseNameRequired, got %v", err)
	}
}

func TestExerciseService_ModifyExercise_RepoError(t *testing.T) {
	ctx := t.Context()
	wantErr := exercise.ErrExerciseNotFound

	repo := &mockExerciseRepository{
		modifyExerciseFunc: func(ctx context.Context, ex exercise.Exercise) (exercise.Exercise, error) {
			return exercise.Exercise{}, wantErr
		},
	}

	service := exercise.NewExerciseService(repo)

	_, err := service.ModifyExercise(ctx, exercise.Exercise{ExerciseId: testutil.Id(1), ExerciseName: "Lunge", UserId: 42})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected error to wrap %v, got %v", wantErr, err)
	}
}
