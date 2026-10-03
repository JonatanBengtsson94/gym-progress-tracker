package template_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/exercise"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/identity"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/set"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/template"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/testutil"
)

// The templates the contract files describe.
var (
	contractPushDay = template.Template{TemplateId: uuid.MustParse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d10"), TemplateName: "Push Day", UserId: 1}
	contractLegDay  = template.Template{TemplateId: uuid.MustParse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d11"), TemplateName: "Leg Day", UserId: 1}
)

func TestTemplateContract_GetTemplates(t *testing.T) {
	benchPress := exercise.Exercise{ExerciseId: uuid.MustParse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d01"), ExerciseName: "Bench Press (Barbell)"}
	service := &mockTemplateService{
		getTemplatesFunc: func(ctx context.Context, userId uint32) ([]template.TemplateWithLatestWorkout, error) {
			return []template.TemplateWithLatestWorkout{
				{
					Template: contractPushDay,
					LatestWorkout: &template.LatestWorkout{
						WorkoutId:   uuid.MustParse("0199a5e0-7c1a-7b3e-9f2d-3a8c4e6b1d20"),
						StartedAt:   time.Date(2026, 9, 30, 17, 0, 0, 0, time.UTC),
						CompletedAt: time.Date(2026, 9, 30, 18, 0, 0, 0, time.UTC),
						Sets: []set.Set{
							{Exercise: benchPress, Reps: 8, WeightGrams: 60000},
							{Exercise: benchPress, Reps: 6, WeightGrams: 62500},
						},
					},
				},
				{Template: contractLegDay},
			}, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/templates", nil)
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	template.NewTemplateHandler(service).GetTemplates(rec, req)

	testutil.AssertJSONEqual(t, rec.Body.Bytes(), testutil.ContractFile(t, "get_templates.response.json"))
}

func TestTemplateContract_CreateTemplate(t *testing.T) {
	var got template.Template
	service := &mockTemplateService{
		createTemplateFunc: func(ctx context.Context, tmpl template.Template) (template.Template, bool, error) {
			got = tmpl
			return tmpl, true, nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/templates", bytes.NewReader(testutil.ContractFile(t, "post_templates.request.json")))
	req = req.WithContext(identity.ContextWithUserId(req.Context(), 1))
	rec := httptest.NewRecorder()

	template.NewTemplateHandler(service).CreateTemplate(rec, req)

	if got != contractPushDay {
		t.Errorf("expected the request to reach the service as %+v, got %+v", contractPushDay, got)
	}
	testutil.AssertJSONEqual(t, rec.Body.Bytes(), testutil.ContractFile(t, "post_templates.response.json"))
}
