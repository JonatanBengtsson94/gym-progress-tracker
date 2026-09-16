package template

import "errors"

var ErrTemplateAlreadyExists = errors.New("template already exists")
var ErrTemplateNameRequired = errors.New("template name is required")

type Template struct {
	UserId       uint32
	TemplateId   uint32
	TemplateName string
}
