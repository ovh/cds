package exportentities

import (
	"encoding/base64"
	"fmt"

	"github.com/ovh/cds/sdk"
)

// Template is the "as code" representation of a sdk.WorkflowTemplate.
type Template struct {
	Slug         string              `json:"slug" yaml:"slug"`
	Name         string              `json:"name" yaml:"name"`
	Group        string              `json:"group" yaml:"group"`
	Description  string              `json:"description,omitempty" yaml:"description,omitempty"`
	Parameters   []TemplateParameter `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Workflow     string
	Pipelines    []string
	Applications []string
	Environments []string
}

// TemplateParameter is the "as code" representation of a sdk.TemplateParameter.
type TemplateParameter struct {
	Key      string `json:"key" yaml:"key"`
	Type     string `json:"type" yaml:"type"`
	Required bool   `json:"required" yaml:"required"`
}

// Name pattern for template files.
const (
	TemplateWorkflowName    = "workflow.yml"
	TemplatePipelineName    = "%d.pipeline.yml"
	TemplateApplicationName = "%d.application.yml"
	TemplateEnvironmentName = "%d.environment.yml"
)

// NewTemplate creates a new exportable workflow template.
func NewTemplate(wt sdk.WorkflowTemplate) (Template, error) {
	exportedTemplate := Template{
		Slug:         wt.Slug,
		Name:         wt.Name,
		Group:        wt.Group.Name,
		Description:  wt.Description,
		Parameters:   make([]TemplateParameter, len(wt.Parameters)),
		Workflow:     TemplateWorkflowName,
		Pipelines:    make([]string, len(wt.Pipelines)),
		Applications: make([]string, len(wt.Applications)),
		Environments: make([]string, len(wt.Environments)),
	}

	for i, p := range wt.Parameters {
		exportedTemplate.Parameters[i].Key = p.Key
		exportedTemplate.Parameters[i].Type = string(p.Type)
		exportedTemplate.Parameters[i].Required = p.Required
	}

	for i := range wt.Pipelines {
		exportedTemplate.Pipelines[i] = fmt.Sprintf(TemplatePipelineName, i+1)
	}
	for i := range wt.Applications {
		exportedTemplate.Applications[i] = fmt.Sprintf(TemplateApplicationName, i+1)
	}
	for i := range wt.Environments {
		exportedTemplate.Environments[i] = fmt.Sprintf(TemplateEnvironmentName, i+1)
	}

	return exportedTemplate, nil
}

// GetTemplate returns a sdk.WorkflowTemplate.
func (w Template) GetTemplate(wkf []byte, pips, apps, envs [][]byte) sdk.WorkflowTemplate {
	wt := sdk.WorkflowTemplate{
		Slug: w.Slug,
		Name: w.Name,
		Group: &sdk.Group{
			Name: w.Group,
		},
		Description:  w.Description,
		Value:        base64.StdEncoding.EncodeToString(wkf),
		Pipelines:    make([]sdk.PipelineTemplate, len(pips)),
		Applications: make([]sdk.ApplicationTemplate, len(apps)),
		Environments: make([]sdk.EnvironmentTemplate, len(envs)),
	}

	for _, p := range w.Parameters {
		wt.Parameters = append(wt.Parameters, sdk.WorkflowTemplateParameter{
			Key:      p.Key,
			Type:     sdk.TemplateParameterType(p.Type),
			Required: p.Required,
		})
	}

	for i := range pips {
		wt.Pipelines[i].Value = base64.StdEncoding.EncodeToString(pips[i])
	}

	for i := range apps {
		wt.Applications[i].Value = base64.StdEncoding.EncodeToString(apps[i])
	}

	for i := range envs {
		wt.Environments[i].Value = base64.StdEncoding.EncodeToString(envs[i])
	}

	return wt
}
