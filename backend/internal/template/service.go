package template

import (
	"context"
	"strings"
)

type TemplateRepository interface {
	GetTemplatesByUserId(ctx context.Context, userId uint32) ([]TemplateWithLatestWorkout, error)
	CreateTemplate(ctx context.Context, template Template) (Template, error)
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

func (s *TemplateServiceImpl) CreateTemplate(ctx context.Context, template Template) (Template, error) {
	if strings.TrimSpace(template.TemplateName) == "" {
		return Template{}, ErrTemplateNameRequired
	}
	return s.repo.CreateTemplate(ctx, template)
}
