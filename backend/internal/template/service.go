package template

import (
	"context"
	"strings"
	"uuid"
)

type TemplateRepository interface {
	GetTemplatesByUserId(ctx context.Context, userId uint32) ([]TemplateWithLatestWorkout, error)
	CreateTemplate(ctx context.Context, template Template) (stored Template, created bool, err error)
}

type TemplateServiceImpl struct {
	repo TemplateRepository
}

func NewTemplateService(repo TemplateRepository) *TemplateServiceImpl {
	return &TemplateServiceImpl{repo: repo}
}

func (s *TemplateServiceImpl) GetTemplates(ctx context.Context, userId uint32) ([]TemplateWithLatestWorkout, error) {
	return s.repo.GetTemplatesByUserId(ctx, userId)
}

// CreateTemplate stores template under the id the client chose for it, or a new one if it chose
// none. When the user already has a template with that id or name, that one is returned instead,
// with created false.
func (s *TemplateServiceImpl) CreateTemplate(ctx context.Context, template Template) (Template, bool, error) {
	if strings.TrimSpace(template.TemplateName) == "" {
		return Template{}, false, ErrTemplateNameRequired
	}
	if template.TemplateId == uuid.Nil() {
		template.TemplateId = uuid.NewV7()
	}
	return s.repo.CreateTemplate(ctx, template)
}
