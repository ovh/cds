package exportentities

import (
	"github.com/ovh/cds/sdk"
)

// Template is the "as code" representation of a sdk.WorkflowTemplate.
type Template struct {
	Slug        string              `json:"slug" yaml:"slug"`
	Name        string              `json:"name" yaml:"name"`
	Group       string              `json:"group" yaml:"group"`
	Description string              `json:"description,omitempty" yaml:"description,omitempty"`
	Parameters  []TemplateParameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type TemplateParameter struct {
	Key      string `json:"key" yaml:"key"`
	Type     string `json:"type" yaml:"type"`
	Required bool   `json:"required" yaml:"required"`
}

// NewTemplate creates a new exportable workflow template.
func NewTemplate(wt sdk.WorkflowTemplate) (Template, error) {
	exportedTemplate := Template{
		Slug:        wt.Slug,
		Name:        wt.Name,
		Group:       wt.Group.Name,
		Description: wt.Description,
		Parameters:  make([]TemplateParameter, len(wt.Parameters)),
	}

	for i, p := range wt.Parameters {
		exportedTemplate.Parameters[i].Key = p.Key
		exportedTemplate.Parameters[i].Type = string(p.Type)
		exportedTemplate.Parameters[i].Required = p.Required
	}

	return exportedTemplate, nil
}

// GetTemplate returns a sdk.WorkflowTemplate.
func (w Template) GetTemplate() sdk.WorkflowTemplate {
	wt := sdk.WorkflowTemplate{
		Slug: w.Slug,
		Name: w.Name,
		Group: &sdk.Group{
			Name: w.Group,
		},
		Description: w.Description,
	}

	for _, p := range w.Parameters {
		wt.Parameters = append(wt.Parameters, sdk.WorkflowTemplateParameter{
			Key:      p.Key,
			Type:     sdk.TemplateParameterType(p.Type),
			Required: p.Required,
		})
	}

	return wt
}
