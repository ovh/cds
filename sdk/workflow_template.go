package sdk

import (
	"database/sql/driver"
	json "encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

// WorkflowTemplateRequest struct use for execution request.
type WorkflowTemplateRequest struct {
	ProjectKey   string            `json:"project_key"`
	WorkflowName string            `json:"workflow_name"`
	Parameters   map[string]string `json:"parameters"`
}

// Value returns driver.Value from workflow template request.
func (w WorkflowTemplateRequest) Value() (driver.Value, error) {
	j, err := json.Marshal(w)
	return j, WrapError(err, "cannot marshal WorkflowTemplateRequest")
}

// Scan workflow template request.
func (w *WorkflowTemplateRequest) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return WithStack(errors.New("type assertion .([]byte) failed"))
	}
	return WrapError(json.Unmarshal(source, w), "cannot unmarshal WorkflowTemplateRequest")
}

// WorkflowTemplateResult struct.
type WorkflowTemplateResult struct {
	InstanceID   int64
	Workflow     string
	Pipelines    []string
	Applications []string
}

// WorkflowTemplate struct.
type WorkflowTemplate struct {
	ID           int64                      `json:"id" db:"id" `
	GroupID      int64                      `json:"group_id" db:"group_id"`
	Name         string                     `json:"name" db:"name"`
	Slug         string                     `json:"slug" db:"slug"`
	Description  string                     `json:"description" db:"description"`
	Parameters   WorkflowTemplateParameters `json:"parameters" db:"parameters"`
	Value        string                     `json:"value" db:"value"`
	Pipelines    PipelineTemplates          `json:"pipelines" db:"pipelines"`
	Applications ApplicationTemplates       `json:"applications" db:"applications"`
	Version      int64                      `json:"version" db:"version"`
	// aggregates
	Group      *Group                 `json:"group,omitempty" db:"-"`
	FirstAudit *AuditWorkflowTemplate `json:"first_audit,omitempty" db:"-"`
	LastAudit  *AuditWorkflowTemplate `json:"last_audit,omitempty" db:"-"`
}

// ValidateStruct returns workflow template validity.
func (w *WorkflowTemplate) ValidateStruct() error {
	if w.Name == "" {
		return WithStack(ErrInvalidData)
	}

	for _, p := range w.Parameters {
		if err := p.ValidateStruct(); err != nil {
			return err
		}
	}

	for _, p := range w.Pipelines {
		if err := p.ValidateStruct(); err != nil {
			return err
		}
	}

	for _, a := range w.Applications {
		if err := a.ValidateStruct(); err != nil {
			return err
		}
	}

	return nil
}

// CheckParams returns template parameters validity.
func (w *WorkflowTemplate) CheckParams(r WorkflowTemplateRequest) error {
	if r.ProjectKey == "" {
		return fmt.Errorf("Project key is required")
	}
	if r.WorkflowName == "" {
		return fmt.Errorf("Workflow name is required")
	}

	for _, p := range w.Parameters {
		v, ok := r.Parameters[p.Key]
		if !ok && p.Required {
			return fmt.Errorf("Param %s is required", p.Key)
		}
		if ok {
			if p.Required && v == "" {
				return fmt.Errorf("Param %s is required", p.Key)
			}
			if p.Type == ParameterTypeBoolean && v != "" && !(v == "true" || v == "false") {
				return fmt.Errorf("Given value it's not a boolean for %s", p.Key)
			}
		}
	}

	return nil
}

// WorkflowTemplatesToIDs returns ids of given workflow templates.
func WorkflowTemplatesToIDs(wts []*WorkflowTemplate) []int64 {
	ids := make([]int64, len(wts))
	for i := 0; i < len(wts); i++ {
		ids[i] = wts[i].ID
	}
	return ids
}

// WorkflowTemplatesToGroupIDs returns group ids of given workflow templates.
func WorkflowTemplatesToGroupIDs(wts []*WorkflowTemplate) []int64 {
	ids := make([]int64, len(wts))
	for i := 0; i < len(wts); i++ {
		ids[i] = wts[i].GroupID
	}
	return ids
}

// PipelineTemplate struct.
type PipelineTemplate struct {
	Value string `json:"value"`
}

// ValidateStruct returns pipeline template validity.
func (p *PipelineTemplate) ValidateStruct() error {
	if len(p.Value) == 0 {
		return WithStack(ErrInvalidData)
	}
	return nil
}

// ApplicationTemplate struct.
type ApplicationTemplate struct {
	Value string `json:"value"`
}

// ValidateStruct returns application template validity.
func (a *ApplicationTemplate) ValidateStruct() error {
	if len(a.Value) == 0 {
		return WithStack(ErrInvalidData)
	}
	return nil
}

// TemplateParameterType used for template parameter.
type TemplateParameterType string

// Parameter types.
const (
	ParameterTypeString  TemplateParameterType = "string"
	ParameterTypeBoolean TemplateParameterType = "boolean"
)

// IsValid returns paramter type validity.
func (t TemplateParameterType) IsValid() bool {
	switch t {
	case ParameterTypeString, ParameterTypeBoolean:
		return true
	}
	return false
}

// WorkflowTemplateParameter struct.
type WorkflowTemplateParameter struct {
	Key      string                `json:"key"`
	Type     TemplateParameterType `json:"type"`
	Required bool                  `json:"required"`
}

// WorkflowTemplateParameters struct.
type WorkflowTemplateParameters []WorkflowTemplateParameter

// Value returns driver.Value from workflow template parameters.
func (w WorkflowTemplateParameters) Value() (driver.Value, error) {
	j, err := json.Marshal(w)
	return j, WrapError(err, "cannot marshal WorkflowTemplateParameters")
}

// Scan workflow template parameters.
func (w *WorkflowTemplateParameters) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return WithStack(errors.New("type assertion .([]byte) failed"))
	}
	return WrapError(json.Unmarshal(source, w), "cannot unmarshal WorkflowTemplateParameters")
}

// PipelineTemplates struct.
type PipelineTemplates []PipelineTemplate

// Value returns driver.Value from workflow template pipelines.
func (p PipelineTemplates) Value() (driver.Value, error) {
	j, err := json.Marshal(p)
	return j, WrapError(err, "cannot marshal PipelineTemplates")
}

// Scan pipeline templates.
func (p *PipelineTemplates) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return WithStack(errors.New("type assertion .([]byte) failed"))
	}
	return WrapError(json.Unmarshal(source, p), "cannot unmarshal PipelineTemplates")
}

// ApplicationTemplates struct.
type ApplicationTemplates []ApplicationTemplate

// Value returns driver.Value from workflow template applications.
func (a ApplicationTemplates) Value() (driver.Value, error) {
	j, err := json.Marshal(a)
	return j, WrapError(err, "cannot marshal ApplicationTemplates")
}

// Scan application templates.
func (a *ApplicationTemplates) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return WithStack(errors.New("type assertion .([]byte) failed"))
	}
	return WrapError(json.Unmarshal(source, a), "cannot unmarshal ApplicationTemplates")
}

// ValidateStruct returns pipeline template validity.
func (w *WorkflowTemplateParameter) ValidateStruct() error {
	if w.Key == "" || !w.Type.IsValid() {
		return WithStack(ErrInvalidData)
	}
	return nil
}

// WorkflowTemplateInstance struct.
type WorkflowTemplateInstance struct {
	ID                      int64                   `json:"id" db:"id" `
	WorkflowTemplateID      int64                   `json:"workflow_template_id" db:"workflow_template_id"`
	ProjectID               int64                   `json:"project_id" db:"project_id"`
	WorkflowID              int64                   `json:"workflow_id" db:"workflow_id"`
	WorkflowTemplateVersion int64                   `json:"workflow_template_version" db:"workflow_template_version"`
	Request                 WorkflowTemplateRequest `json:"request" db:"request"`
}
