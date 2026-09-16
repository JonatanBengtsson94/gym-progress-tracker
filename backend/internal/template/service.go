package template

import (
	"context"
	"strings"
)

type TemplateRepository interface {
	CreateTemplate(ctx context.Context, template Template) (Template, error)
}

type TemplateServiceImpl struct {
	repo TemplateRepository
}

func NewTemplateService(repo TemplateRepository) *TemplateServiceImpl {
	return &TemplateServiceImpl{repo: repo}
}

func (s *TemplateServiceImpl) CreateTemplate(ctx context.Context, template Template) (Template, error) {
	if strings.TrimSpace(template.TemplateName) == "" {
		return Template{}, ErrTemplateNameRequired
	}
	return s.repo.CreateTemplate(ctx, template)
}
