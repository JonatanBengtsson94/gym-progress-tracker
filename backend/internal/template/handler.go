package template

import (
	"context"
	"errors"
	"net/http"
	"time"
	"uuid"

	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/httpx"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/identity"
	"github.com/JonatanBengtsson94/gym-progress-tracker/backend/internal/set"
)

type TemplateService interface {
	GetTemplates(context.Context, uint32) ([]TemplateWithLatestWorkout, error)
	CreateTemplate(context.Context, Template) (stored Template, created bool, err error)
}

type TemplateHandler struct {
	service TemplateService
}

func NewTemplateHandler(service TemplateService) *TemplateHandler {
	return &TemplateHandler{service: service}
}

type createTemplateRequest struct {
	TemplateId   uuid.UUID `json:"template_id"`
	TemplateName string    `json:"template_name"`
}

type TemplateResponse struct {
	TemplateId   uuid.UUID `json:"template_id"`
	TemplateName string    `json:"template_name"`
}

type LatestWorkoutResponse struct {
	WorkoutId   uuid.UUID                  `json:"workout_id"`
	StartedAt   time.Time                  `json:"started_at"`
	CompletedAt time.Time                  `json:"completed_at"`
	Exercises   []set.ExerciseSetsResponse `json:"exercises"`
}

type TemplateWithLatestWorkoutResponse struct {
	TemplateId    uuid.UUID              `json:"template_id"`
	TemplateName  string                 `json:"template_name"`
	LatestWorkout *LatestWorkoutResponse `json:"latest_workout"`
}

type TemplatesResponse struct {
	Templates []TemplateWithLatestWorkoutResponse `json:"templates"`
}

func toTemplateWithLatestWorkoutResponse(t TemplateWithLatestWorkout) TemplateWithLatestWorkoutResponse {
	res := TemplateWithLatestWorkoutResponse{TemplateId: t.TemplateId, TemplateName: t.TemplateName}
	if t.LatestWorkout != nil {
		res.LatestWorkout = &LatestWorkoutResponse{
			WorkoutId:   t.LatestWorkout.WorkoutId,
			StartedAt:   t.LatestWorkout.StartedAt,
			CompletedAt: t.LatestWorkout.CompletedAt,
			Exercises:   set.GroupByExercise(t.LatestWorkout.Sets),
		}
	}
	return res
}

// GetTemplates lists the user's templates, each with the latest workout
// logged under it so clients can prefill a new workout from it. Recently
// performed templates come first; a template that has never been used has a
// null latest_workout and is listed after them, by name.

func (h *TemplateHandler) GetTemplates(w http.ResponseWriter, r *http.Request) {
	userId, ok := identity.RequireUserId(w, r)
	if !ok {
		return
	}

	templates, err := h.service.GetTemplates(r.Context(), userId)
	if err != nil {
		httpx.InternalError(w, err)
		return
	}

	templatesResponse := make([]TemplateWithLatestWorkoutResponse, len(templates))
	for i, t := range templates {
		templatesResponse[i] = toTemplateWithLatestWorkoutResponse(t)
	}

	httpx.WriteJSON(w, http.StatusOK, TemplatesResponse{Templates: templatesResponse})
}

// CreateTemplate creates a template for the user, under the template_id the client sends or a new
// one if it sends none, and responds 201 Created. It never makes a second template with the same
// id or name: if the user already has the template_id, or a template with the name, ignoring case,
// nothing is created and that template is returned with 200 OK. So retrying a create is safe, and
// a client that created the template offline gets back the existing one, whose template_id it
// should use in place of its own. A template_id belonging to another user's template is
// 409 Conflict.
func (h *TemplateHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	userId, ok := identity.RequireUserId(w, r)
	if !ok {
		return
	}

	var req createTemplateRequest
	if !httpx.DecodeJSONBody(w, r, &req) {
		return
	}

	template := Template{TemplateId: req.TemplateId, TemplateName: req.TemplateName, UserId: userId}

	stored, created, err := h.service.CreateTemplate(r.Context(), template)
	switch {
	case errors.Is(err, ErrTemplateNameRequired):
		http.Error(w, "template_name is required", http.StatusBadRequest)
		return
	case errors.Is(err, ErrTemplateIdTaken):
		http.Error(w, "template_id is already in use", http.StatusConflict)
		return
	case err != nil:
		httpx.InternalError(w, err)
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpx.WriteJSON(w, status, TemplateResponse{TemplateId: stored.TemplateId, TemplateName: stored.TemplateName})
}
