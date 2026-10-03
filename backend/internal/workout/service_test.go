package workout_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/set"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/testutil"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/workout"
)

type mockWorkoutRepository struct {
	getWorkoutFunc    func(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error)
	getWorkoutsFunc   func(ctx context.Context, userId uint32) ([]workout.Workout, error)
	getByTemplateFunc func(ctx context.Context, userId uint32, templateId uuid.UUID) ([]workout.Workout, error)
	createWorkoutFunc func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, bool, error)
	modifyWorkoutFunc func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error)
}

func (m *mockWorkoutRepository) GetWorkoutByUserIdAndWorkoutId(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error) {
	return m.getWorkoutFunc(ctx, userId, workoutId)
}

func (m *mockWorkoutRepository) GetWorkoutsByUserId(ctx context.Context, userId uint32) ([]workout.Workout, error) {
	return m.getWorkoutsFunc(ctx, userId)
}

func (m *mockWorkoutRepository) GetWorkoutsByUserIdAndTemplateId(ctx context.Context, userId uint32, templateId uuid.UUID) ([]workout.Workout, error) {
	return m.getByTemplateFunc(ctx, userId, templateId)
}

func (m *mockWorkoutRepository) CreateWorkout(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, bool, error) {
	return m.createWorkoutFunc(ctx, userId, w)
}

func (m *mockWorkoutRepository) ModifyWorkout(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
	return m.modifyWorkoutFunc(ctx, userId, w)
}

func TestWorkoutService_GetWorkout(t *testing.T) {
	ctx := t.Context()
	const wantUserId = 1
	wantWorkoutId := testutil.Id(42)
	expected := workout.Workout{
		WorkoutId:   wantWorkoutId,
		CompletedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Template:    template.Template{TemplateName: "Push Day"},
		Sets: []set.Set{
			{Reps: 8, WeightGrams: 60000},
		},
	}

	var gotUserId uint32
	var gotWorkoutId uuid.UUID
	repo := &mockWorkoutRepository{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error) {
			gotUserId = userId
			gotWorkoutId = workoutId
			return expected, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	got, err := service.GetWorkout(ctx, wantUserId, wantWorkoutId)
	if err != nil {
		t.Fatalf("GetWorkout returned error: %v", err)
	}

	if gotUserId != wantUserId {
		t.Errorf("expected repo to receive userId %d, got %d", wantUserId, gotUserId)
	}
	if gotWorkoutId != wantWorkoutId {
		t.Errorf("expected repo to receive workoutId %v, got %v", wantWorkoutId, gotWorkoutId)
	}

	if got.WorkoutId != expected.WorkoutId || got.Template.TemplateName != expected.Template.TemplateName || len(got.Sets) != len(expected.Sets) {
		t.Errorf("GetWorkout() = %+v, want %+v", got, expected)
	}
}

func validWorkout() workout.Workout {
	return workout.Workout{
		StartedAt:   time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
		CompletedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Template:    template.Template{TemplateId: testutil.Id(1)},
		Sets:        []set.Set{{Reps: 8, WeightGrams: 60000}},
	}
}

func TestWorkoutService_CreateWorkout(t *testing.T) {
	ctx := t.Context()
	const wantUserId = 1
	toCreate := validWorkout()
	toCreate.WorkoutId = testutil.Id(7)

	var gotUserId uint32
	var gotWorkout workout.Workout
	repo := &mockWorkoutRepository{
		createWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, bool, error) {
			gotUserId = userId
			gotWorkout = w
			return w, true, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	got, created, err := service.CreateWorkout(ctx, wantUserId, toCreate)
	if err != nil {
		t.Fatalf("CreateWorkout returned error: %v", err)
	}

	if gotUserId != wantUserId {
		t.Errorf("expected repo to receive userId %d, got %d", wantUserId, gotUserId)
	}
	if gotWorkout.WorkoutId != toCreate.WorkoutId {
		t.Errorf("expected repo to receive WorkoutId %v unchanged, got %v", toCreate.WorkoutId, gotWorkout.WorkoutId)
	}
	if !gotWorkout.StartedAt.Equal(toCreate.StartedAt) {
		t.Errorf("expected repo to receive StartedAt %v, got %v", toCreate.StartedAt, gotWorkout.StartedAt)
	}
	if !gotWorkout.CompletedAt.Equal(toCreate.CompletedAt) {
		t.Errorf("expected repo to receive CompletedAt %v, got %v", toCreate.CompletedAt, gotWorkout.CompletedAt)
	}
	if got.WorkoutId != toCreate.WorkoutId || !created {
		t.Errorf("CreateWorkout() = workout %v, created %v, want workout %v, created true", got.WorkoutId, created, toCreate.WorkoutId)
	}
}

func TestWorkoutService_CreateWorkout_GeneratesIdWhenOmitted(t *testing.T) {
	ctx := t.Context()

	var got []uuid.UUID
	repo := &mockWorkoutRepository{
		createWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, bool, error) {
			got = append(got, w.WorkoutId)
			return w, true, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	for range 2 {
		if _, _, err := service.CreateWorkout(ctx, 1, validWorkout()); err != nil {
			t.Fatalf("CreateWorkout returned error: %v", err)
		}
	}

	if got[0] == uuid.Nil() || got[0] == got[1] {
		t.Errorf("expected each create without an id to get a new one, got %v", got)
	}
}

func TestWorkoutService_CreateWorkout_ReturnsStoredWorkout(t *testing.T) {
	ctx := t.Context()
	stored := existingWorkout()

	repo := &mockWorkoutRepository{
		createWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, bool, error) {
			return stored, false, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	toCreate := validWorkout()
	toCreate.WorkoutId = stored.WorkoutId

	got, created, err := service.CreateWorkout(ctx, 1, toCreate)
	if err != nil {
		t.Fatalf("CreateWorkout returned error: %v", err)
	}
	if got.WorkoutId != stored.WorkoutId || !got.CompletedAt.Equal(stored.CompletedAt) || created {
		t.Errorf("CreateWorkout() = %+v, created %v, want the stored workout %+v, created false", got, created, stored)
	}
}

func TestWorkoutService_CreateWorkout_ValidationErrors(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name    string
		workout func() workout.Workout
		wantErr error
	}{
		{
			name: "no sets",
			workout: func() workout.Workout {
				w := validWorkout()
				w.Sets = nil
				return w
			},
			wantErr: workout.ErrSetsRequired,
		},
		{
			name: "set with zero reps",
			workout: func() workout.Workout {
				w := validWorkout()
				w.Sets = []set.Set{{Reps: 8}, {Reps: 0}}
				return w
			},
			wantErr: workout.ErrRepsRequired,
		},
		{
			name: "no template id and blank template name",
			workout: func() workout.Workout {
				w := validWorkout()
				w.Template = template.Template{TemplateName: "   "}
				return w
			},
			wantErr: template.ErrTemplateNameRequired,
		},
		{
			name: "no started at",
			workout: func() workout.Workout {
				w := validWorkout()
				w.StartedAt = time.Time{}
				return w
			},
			wantErr: workout.ErrStartedAtRequired,
		},
		{
			name: "started after completed",
			workout: func() workout.Workout {
				w := validWorkout()
				w.StartedAt = w.CompletedAt.Add(time.Second)
				return w
			},
			wantErr: workout.ErrStartedAfterCompleted,
		},
		{
			name: "started in the future with completed at defaulted to now",
			workout: func() workout.Workout {
				w := validWorkout()
				w.StartedAt = time.Now().Add(time.Hour)
				w.CompletedAt = time.Time{}
				return w
			},
			wantErr: workout.ErrStartedAfterCompleted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockWorkoutRepository{
				createWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, bool, error) {
					t.Fatal("CreateWorkout should not reach the repository for an invalid workout")
					return workout.Workout{}, false, nil
				},
			}

			service := workout.NewWorkoutService(repo)

			_, _, err := service.CreateWorkout(ctx, 1, tt.workout())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestWorkoutService_CreateWorkout_TrimsGeneratedTemplateName(t *testing.T) {
	ctx := t.Context()
	toCreate := validWorkout()
	toCreate.Template = template.Template{TemplateName: "  Leg Day  "}

	var gotWorkout workout.Workout
	repo := &mockWorkoutRepository{
		createWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, bool, error) {
			gotWorkout = w
			return w, true, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	if _, _, err := service.CreateWorkout(ctx, 1, toCreate); err != nil {
		t.Fatalf("CreateWorkout returned error: %v", err)
	}

	if gotWorkout.Template.TemplateName != "Leg Day" {
		t.Errorf("expected repo to receive TemplateName %q, got %q", "Leg Day", gotWorkout.Template.TemplateName)
	}
}

func TestWorkoutService_CreateWorkout_DefaultsCompletedAt(t *testing.T) {
	ctx := t.Context()
	toCreate := validWorkout()
	toCreate.CompletedAt = time.Time{}

	var gotWorkout workout.Workout
	repo := &mockWorkoutRepository{
		createWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, bool, error) {
			gotWorkout = w
			return w, true, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	before := time.Now().UTC()
	if _, _, err := service.CreateWorkout(ctx, 1, toCreate); err != nil {
		t.Fatalf("CreateWorkout returned error: %v", err)
	}
	after := time.Now().UTC()

	if gotWorkout.CompletedAt.Before(before) || gotWorkout.CompletedAt.After(after) {
		t.Errorf("expected CompletedAt to default to now, got %v", gotWorkout.CompletedAt)
	}
}

func TestWorkoutService_GetWorkout_RepoError(t *testing.T) {
	ctx := t.Context()
	wantErr := workout.ErrWorkoutNotFound

	repo := &mockWorkoutRepository{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error) {
			return workout.Workout{}, wantErr
		},
	}

	service := workout.NewWorkoutService(repo)

	_, err := service.GetWorkout(ctx, 1, testutil.Id(42))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected error to wrap %v, got %v", wantErr, err)
	}
}

func existingWorkout() workout.Workout {
	return workout.Workout{
		WorkoutId:   testutil.Id(42),
		StartedAt:   time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
		CompletedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Template:    template.Template{TemplateId: testutil.Id(3), TemplateName: "Push Day"},
		Sets:        []set.Set{{Reps: 5, WeightGrams: 50000}},
	}
}

func TestWorkoutService_ModifyWorkout(t *testing.T) {
	ctx := t.Context()
	const wantUserId = 1

	existing := existingWorkout()
	toModify := workout.Workout{
		WorkoutId:   testutil.Id(42),
		StartedAt:   time.Date(2024, 1, 16, 11, 0, 0, 0, time.UTC),
		CompletedAt: time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC),
		Sets:        []set.Set{{Reps: 8, WeightGrams: 60000}, {Reps: 6, WeightGrams: 65000}},
	}

	var gotGetUserId, gotModifyUserId uint32
	var gotGetWorkoutId uuid.UUID
	var gotWorkout workout.Workout
	repo := &mockWorkoutRepository{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error) {
			gotGetUserId = userId
			gotGetWorkoutId = workoutId
			return existing, nil
		},
		modifyWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
			gotModifyUserId = userId
			gotWorkout = w
			return w, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	got, err := service.ModifyWorkout(ctx, wantUserId, toModify)
	if err != nil {
		t.Fatalf("ModifyWorkout returned error: %v", err)
	}

	if gotGetUserId != wantUserId || gotGetWorkoutId != testutil.Id(42) {
		t.Errorf("expected repo lookup for user %d / workout 42, got user %d / workout %v", wantUserId, gotGetUserId, gotGetWorkoutId)
	}
	if gotModifyUserId != wantUserId {
		t.Errorf("expected repo to receive userId %d, got %d", wantUserId, gotModifyUserId)
	}
	if !gotWorkout.StartedAt.Equal(toModify.StartedAt) {
		t.Errorf("expected repo to receive StartedAt %v, got %v", toModify.StartedAt, gotWorkout.StartedAt)
	}
	if !gotWorkout.CompletedAt.Equal(toModify.CompletedAt) {
		t.Errorf("expected repo to receive CompletedAt %v, got %v", toModify.CompletedAt, gotWorkout.CompletedAt)
	}
	if len(gotWorkout.Sets) != 2 {
		t.Errorf("expected repo to receive 2 sets, got %+v", gotWorkout.Sets)
	}
	if got.WorkoutId != testutil.Id(42) {
		t.Errorf("expected WorkoutId 42, got %v", got.WorkoutId)
	}
}

func TestWorkoutService_ModifyWorkout_KeepsExistingTemplate(t *testing.T) {
	ctx := t.Context()
	existing := existingWorkout()

	var gotWorkout workout.Workout
	repo := &mockWorkoutRepository{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error) {
			return existing, nil
		},
		modifyWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
			gotWorkout = w
			return w, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	toModify := validWorkout()
	toModify.WorkoutId = testutil.Id(42)
	toModify.Template = template.Template{TemplateId: testutil.Id(99), TemplateName: "Sneaky Day"}

	if _, err := service.ModifyWorkout(ctx, 1, toModify); err != nil {
		t.Fatalf("ModifyWorkout returned error: %v", err)
	}

	if gotWorkout.Template != existing.Template {
		t.Errorf("expected the existing template %+v to be kept, got %+v", existing.Template, gotWorkout.Template)
	}
}

func TestWorkoutService_ModifyWorkout_KeepsExistingStartedAtWhenOmitted(t *testing.T) {
	ctx := t.Context()
	existing := existingWorkout()

	var gotWorkout workout.Workout
	repo := &mockWorkoutRepository{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error) {
			return existing, nil
		},
		modifyWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
			gotWorkout = w
			return w, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	toModify := validWorkout()
	toModify.WorkoutId = testutil.Id(42)
	toModify.StartedAt = time.Time{}

	if _, err := service.ModifyWorkout(ctx, 1, toModify); err != nil {
		t.Fatalf("ModifyWorkout returned error: %v", err)
	}

	if !gotWorkout.StartedAt.Equal(existing.StartedAt) {
		t.Errorf("expected the existing StartedAt %v to be kept, got %v", existing.StartedAt, gotWorkout.StartedAt)
	}
}

// The ordering check must use the stored value for whichever time is omitted,
// so a change to one time alone can't leave the workout ending before it began.
func TestWorkoutService_ModifyWorkout_StartedAfterCompleted(t *testing.T) {
	ctx := t.Context()
	existing := existingWorkout()

	tests := []struct {
		name   string
		modify func(*workout.Workout)
	}{
		{"both given", func(w *workout.Workout) {
			w.StartedAt = w.CompletedAt.Add(time.Second)
		}},
		{"started at after existing completed at", func(w *workout.Workout) {
			w.StartedAt = existing.CompletedAt.Add(time.Second)
			w.CompletedAt = time.Time{}
		}},
		{"completed at before existing started at", func(w *workout.Workout) {
			w.StartedAt = time.Time{}
			w.CompletedAt = existing.StartedAt.Add(-time.Second)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockWorkoutRepository{
				getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error) {
					return existing, nil
				},
				modifyWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
					t.Fatal("ModifyWorkout should not reach the repository when started_at is after completed_at")
					return workout.Workout{}, nil
				},
			}

			service := workout.NewWorkoutService(repo)

			toModify := validWorkout()
			toModify.WorkoutId = testutil.Id(42)
			tt.modify(&toModify)

			_, err := service.ModifyWorkout(ctx, 1, toModify)
			if !errors.Is(err, workout.ErrStartedAfterCompleted) {
				t.Fatalf("Expected ErrStartedAfterCompleted, got %v", err)
			}
		})
	}
}

func TestWorkoutService_ModifyWorkout_KeepsExistingCompletedAtWhenOmitted(t *testing.T) {
	ctx := t.Context()
	existing := existingWorkout()

	var gotWorkout workout.Workout
	repo := &mockWorkoutRepository{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error) {
			return existing, nil
		},
		modifyWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
			gotWorkout = w
			return w, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	toModify := validWorkout()
	toModify.WorkoutId = testutil.Id(42)
	toModify.CompletedAt = time.Time{}

	if _, err := service.ModifyWorkout(ctx, 1, toModify); err != nil {
		t.Fatalf("ModifyWorkout returned error: %v", err)
	}

	if !gotWorkout.CompletedAt.Equal(existing.CompletedAt) {
		t.Errorf("expected the existing CompletedAt %v to be kept, got %v", existing.CompletedAt, gotWorkout.CompletedAt)
	}
}

func TestWorkoutService_ModifyWorkout_ValidationErrors(t *testing.T) {
	ctx := t.Context()

	tests := []struct {
		name    string
		sets    []set.Set
		wantErr error
	}{
		{"no sets", nil, workout.ErrSetsRequired},
		{"set with zero reps", []set.Set{{Reps: 8}, {Reps: 0}}, workout.ErrRepsRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockWorkoutRepository{
				getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error) {
					t.Fatal("ModifyWorkout should not reach the repository for an invalid workout")
					return workout.Workout{}, nil
				},
				modifyWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
					t.Fatal("ModifyWorkout should not reach the repository for an invalid workout")
					return workout.Workout{}, nil
				},
			}

			service := workout.NewWorkoutService(repo)

			toModify := validWorkout()
			toModify.WorkoutId = testutil.Id(42)
			toModify.Sets = tt.sets

			_, err := service.ModifyWorkout(ctx, 1, toModify)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestWorkoutService_ModifyWorkout_WorkoutNotFound(t *testing.T) {
	ctx := t.Context()
	repo := &mockWorkoutRepository{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error) {
			return workout.Workout{}, workout.ErrWorkoutNotFound
		},
		modifyWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
			t.Fatal("ModifyWorkout should not be called for a workout that does not exist")
			return workout.Workout{}, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	toModify := validWorkout()
	toModify.WorkoutId = testutil.Id(42)

	_, err := service.ModifyWorkout(ctx, 1, toModify)
	if !errors.Is(err, workout.ErrWorkoutNotFound) {
		t.Fatalf("Expected ErrWorkoutNotFound, got %v", err)
	}
}

func TestWorkoutService_ModifyWorkout_RepositoryError(t *testing.T) {
	ctx := t.Context()
	wantErr := errors.New("db exploded")
	repo := &mockWorkoutRepository{
		getWorkoutFunc: func(ctx context.Context, userId uint32, workoutId uuid.UUID) (workout.Workout, error) {
			return existingWorkout(), nil
		},
		modifyWorkoutFunc: func(ctx context.Context, userId uint32, w workout.Workout) (workout.Workout, error) {
			return workout.Workout{}, wantErr
		},
	}

	service := workout.NewWorkoutService(repo)

	toModify := validWorkout()
	toModify.WorkoutId = testutil.Id(42)

	_, err := service.ModifyWorkout(ctx, 1, toModify)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected %v, got %v", wantErr, err)
	}
}

func TestWorkoutService_GetWorkouts(t *testing.T) {
	ctx := t.Context()
	const wantUserId = 1
	expected := []workout.Workout{
		{WorkoutId: testutil.Id(2), CompletedAt: time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC), Template: template.Template{TemplateId: testutil.Id(1), TemplateName: "Push Day"}},
		{WorkoutId: testutil.Id(1), CompletedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), Template: template.Template{TemplateId: testutil.Id(1), TemplateName: "Push Day"}},
	}

	var gotUserId uint32
	repo := &mockWorkoutRepository{
		getWorkoutsFunc: func(ctx context.Context, userId uint32) ([]workout.Workout, error) {
			gotUserId = userId
			return expected, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	got, err := service.GetWorkouts(ctx, wantUserId)
	if err != nil {
		t.Fatalf("GetWorkouts returned error: %v", err)
	}

	if gotUserId != wantUserId {
		t.Errorf("expected repo to receive userId %d, got %d", wantUserId, gotUserId)
	}
	if len(got) != len(expected) || got[0].WorkoutId != testutil.Id(2) || got[1].WorkoutId != testutil.Id(1) {
		t.Errorf("GetWorkouts() = %+v, want %+v (in the repository's order)", got, expected)
	}
}

func TestWorkoutService_GetWorkouts_RepositoryError(t *testing.T) {
	ctx := t.Context()
	wantErr := errors.New("db exploded")
	repo := &mockWorkoutRepository{
		getWorkoutsFunc: func(ctx context.Context, userId uint32) ([]workout.Workout, error) {
			return nil, wantErr
		},
	}

	service := workout.NewWorkoutService(repo)

	_, err := service.GetWorkouts(ctx, 1)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Expected %v, got %v", wantErr, err)
	}
}

func TestWorkoutService_GetWorkoutsByTemplate(t *testing.T) {
	ctx := t.Context()
	const wantUserId = 1
	wantTemplateId := testutil.Id(3)
	expected := []workout.Workout{
		{WorkoutId: testutil.Id(2), CompletedAt: time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC), Template: template.Template{TemplateId: wantTemplateId, TemplateName: "Push Day"}},
		{WorkoutId: testutil.Id(1), CompletedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), Template: template.Template{TemplateId: wantTemplateId, TemplateName: "Push Day"}},
	}

	var gotUserId uint32
	var gotTemplateId uuid.UUID
	repo := &mockWorkoutRepository{
		getByTemplateFunc: func(ctx context.Context, userId uint32, templateId uuid.UUID) ([]workout.Workout, error) {
			gotUserId, gotTemplateId = userId, templateId
			return expected, nil
		},
	}

	service := workout.NewWorkoutService(repo)

	got, err := service.GetWorkoutsByTemplate(ctx, wantUserId, wantTemplateId)
	if err != nil {
		t.Fatalf("GetWorkoutsByTemplate returned error: %v", err)
	}

	if gotUserId != wantUserId || gotTemplateId != wantTemplateId {
		t.Errorf("expected repo to receive userId %d and templateId %v, got %d and %v", wantUserId, wantTemplateId, gotUserId, gotTemplateId)
	}
	if len(got) != len(expected) || got[0].WorkoutId != testutil.Id(2) || got[1].WorkoutId != testutil.Id(1) {
		t.Errorf("GetWorkoutsByTemplate() = %+v, want %+v (in the repository's order)", got, expected)
	}
}

func TestWorkoutService_GetWorkoutsByTemplate_RepositoryErrors(t *testing.T) {
	for _, wantErr := range []error{workout.ErrTemplateNotFound, errors.New("db exploded")} {
		t.Run(wantErr.Error(), func(t *testing.T) {
			repo := &mockWorkoutRepository{
				getByTemplateFunc: func(ctx context.Context, userId uint32, templateId uuid.UUID) ([]workout.Workout, error) {
					return nil, wantErr
				},
			}

			service := workout.NewWorkoutService(repo)

			_, err := service.GetWorkoutsByTemplate(t.Context(), 1, testutil.Id(3))
			if !errors.Is(err, wantErr) {
				t.Fatalf("Expected %v, got %v", wantErr, err)
			}
		})
	}
}
