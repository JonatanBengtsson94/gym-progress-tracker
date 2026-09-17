package template

import (
	"context"
	"errors"
	"net/http"

	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/httpx"
	"github.com/JonatanBengtsson94/gym-progress-tracker/internal/identity"
)

type TemplateService interface {
	CreateTemplate(context.Context, Template) (Template, error)
}

type TemplateHandler struct {
	service TemplateService
}

func NewTemplateHandler(service TemplateService) *TemplateHandler {
	return &TemplateHandler{service: service}
}

type createTemplateRequest struct {
	TemplateName string `json:"template_name"`
}

type TemplateResponse struct {
	TemplateId   uint32 `json:"template_id"`
	TemplateName string `json:"template_name"`
}

func (h *TemplateHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	userId, ok := identity.RequireUserId(w, r)
	if !ok {
		return
	}

	var req createTemplateRequest
	if !httpx.DecodeJSONBody(w, r, &req) {
		return
	}

	template := Template{TemplateName: req.TemplateName, UserId: userId}

	createdTemplate, err := h.service.CreateTemplate(r.Context(), template)
	switch {
	case errors.Is(err, ErrTemplateNameRequired):
		http.Error(w, "template_name is required", http.StatusBadRequest)
		return
	case errors.Is(err, ErrTemplateAlreadyExists):
		http.Error(w, "Template already exists", http.StatusConflict)
		return
	case err != nil:
		httpx.InternalError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, TemplateResponse{TemplateId: createdTemplate.TemplateId, TemplateName: createdTemplate.TemplateName})
}
