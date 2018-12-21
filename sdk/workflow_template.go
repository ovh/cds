package sdk

import (
	"database/sql/driver"
	json "encoding/json"
	"strings"

	"github.com/ovh/cds/sdk/slug"
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
	Workflow     string
	Pipelines    []string
	Applications []string
	Environments []string
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
	Environments EnvironmentTemplates       `json:"environments" db:"environments"`
	Version      int64                      `json:"version" db:"version"`
	// aggregates
	Group         *Group                 `json:"group,omitempty" db:"-"`
	FirstAudit    *AuditWorkflowTemplate `json:"first_audit,omitempty" db:"-"`
	LastAudit     *AuditWorkflowTemplate `json:"last_audit,omitempty" db:"-"`
	Editable      bool                   `json:"editable,omitempty" db:"-"`
	ChangeMessage string                 `json:"change_message,omitempty" db:"-"`
}

// IsValid returns workflow template validity.
func (w *WorkflowTemplate) IsValid() error {
	w.Slug = slug.Convert(w.Name)
	if !slug.Valid(w.Slug) {
		return WrapError(ErrWrongRequest, "Invalid given name")
	}

	for _, p := range w.Parameters {
		if err := p.IsValid(); err != nil {
			return err
		}
	}

	for _, p := range w.Pipelines {
		if err := p.IsValid(); err != nil {
			return err
		}
	}

	for _, a := range w.Applications {
		if err := a.IsValid(); err != nil {
			return err
		}
	}

	for _, e := range w.Environments {
		if err := e.IsValid(); err != nil {
			return err
		}
	}

	return nil
}

// CheckParams returns template parameters validity.
func (w *WorkflowTemplate) CheckParams(r WorkflowTemplateRequest) error {
	if r.ProjectKey == "" {
		return NewErrorFrom(ErrInvalidData, "Project key is required")
	}
	regexp := NamePatternRegex
	if !regexp.MatchString(r.WorkflowName) {
		return NewErrorFrom(ErrInvalidData, "Invalid given workflow name, should match %s pattern", NamePattern)
	}

	for _, p := range w.Parameters {
		v, ok := r.Parameters[p.Key]
		if !ok && p.Required {
			return NewErrorFrom(ErrInvalidData, "Param %s is required", p.Key)
		}
		if ok {
			if p.Required && v == "" {
				return NewErrorFrom(ErrInvalidData, "Param %s is required", p.Key)
			}
			switch p.Type {
			case ParameterTypeBoolean:
				if v != "" && !(v == "true" || v == "false") {
					return NewErrorFrom(ErrInvalidData, "Given value it's not a boolean for %s", p.Key)
				}
			case ParameterTypeRepository:
				sp := strings.Split(v, "/")
				if len(sp) != 3 {
					return NewErrorFrom(ErrInvalidData, "Given value don't match vcs/repository pattern for %s", p.Key)
				}
			}
		}
	}

	return nil
}

// Update workflow template field from new data.
func (w *WorkflowTemplate) Update(data WorkflowTemplate) {
	w.Name = data.Name
	w.Slug = data.Slug
	w.GroupID = data.GroupID
	w.Description = data.Description
	w.Value = data.Value
	w.Parameters = data.Parameters
	w.Pipelines = data.Pipelines
	w.Applications = data.Applications
	w.Environments = data.Environments
	w.Version = w.Version + 1
}

// WorkflowTemplatesToIDs returns ids of given workflow templates.
func WorkflowTemplatesToIDs(wts []*WorkflowTemplate) []int64 {
	ids := make([]int64, len(wts))
	for i := range wts {
		ids[i] = wts[i].ID
	}
	return ids
}

// WorkflowTemplatesToGroupIDs returns group ids of given workflow templates.
func WorkflowTemplatesToGroupIDs(wts []*WorkflowTemplate) []int64 {
	ids := make([]int64, len(wts))
	for i := range wts {
		ids[i] = wts[i].GroupID
	}
	return ids
}

// PipelineTemplate struct.
type PipelineTemplate struct {
	Value string `json:"value"`
}

// IsValid returns pipeline template validity.
func (p *PipelineTemplate) IsValid() error {
	if len(p.Value) == 0 {
		return NewErrorFrom(ErrInvalidData, "Invalid given pipeline value")
	}
	return nil
}

// ApplicationTemplate struct.
type ApplicationTemplate struct {
	Value string `json:"value"`
}

// IsValid returns application template validity.
func (a *ApplicationTemplate) IsValid() error {
	if len(a.Value) == 0 {
		return NewErrorFrom(ErrInvalidData, "Invalid given application value")
	}
	return nil
}

// EnvironmentTemplate struct.
type EnvironmentTemplate struct {
	Value string `json:"value"`
}

// IsValid returns environment template validity.
func (e *EnvironmentTemplate) IsValid() error {
	if len(e.Value) == 0 {
		return NewErrorFrom(ErrInvalidData, "Invalid given environment value")
	}
	return nil
}

// TemplateParameterType used for template parameter.
type TemplateParameterType string

// Parameter types.
const (
	ParameterTypeString     TemplateParameterType = "string"
	ParameterTypeBoolean    TemplateParameterType = "boolean"
	ParameterTypeRepository TemplateParameterType = "repository"
)

// IsValid returns paramter type validity.
func (t TemplateParameterType) IsValid() bool {
	switch t {
	case ParameterTypeString, ParameterTypeBoolean, ParameterTypeRepository:
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

// EnvironmentTemplates struct.
type EnvironmentTemplates []EnvironmentTemplate

// Value returns driver.Value from workflow template applications.
func (e EnvironmentTemplates) Value() (driver.Value, error) {
	j, err := json.Marshal(e)
	return j, WrapError(err, "cannot marshal EnvironmentTemplates")
}

// Scan environment templates.
func (e *EnvironmentTemplates) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return WithStack(errors.New("type assertion .([]byte) failed"))
	}
	return WrapError(json.Unmarshal(source, e), "cannot unmarshal EnvironmentTemplates")
}

// IsValid returns pipeline template validity.
func (w *WorkflowTemplateParameter) IsValid() error {
	if w.Key == "" || !w.Type.IsValid() {
		return NewErrorFrom(ErrInvalidData, "Invalid given key or type for parameter")
	}
	return nil
}

// WorkflowTemplateInstance struct.
type WorkflowTemplateInstance struct {
	ID                      int64                   `json:"id" db:"id"`
	WorkflowTemplateID      int64                   `json:"workflow_template_id" db:"workflow_template_id"`
	ProjectID               int64                   `json:"project_id" db:"project_id"`
	WorkflowID              *int64                  `json:"workflow_id" db:"workflow_id"`
	WorkflowTemplateVersion int64                   `json:"workflow_template_version" db:"workflow_template_version"`
	Request                 WorkflowTemplateRequest `json:"request" db:"request"`
	WorkflowName            string                  `json:"workflow_name" db:"workflow_name"`
	// aggregates
	FirstAudit *AuditWorkflowTemplateInstance `json:"first_audit,omitempty" db:"-"`
	LastAudit  *AuditWorkflowTemplateInstance `json:"last_audit,omitempty" db:"-"`
	Template   *WorkflowTemplate              `json:"template,omitempty" db:"-"`
	Project    *Project                       `json:"project,omitempty" db:"-"`
	Workflow   *Workflow                      `json:"workflow,omitempty" db:"-"`
}

// WorkflowTemplateInstancesToIDs returns ids of given workflow template instances.
func WorkflowTemplateInstancesToIDs(wtis []*WorkflowTemplateInstance) []int64 {
	ids := make([]int64, len(wtis))
	for i := range wtis {
		ids[i] = wtis[i].ID
	}
	return ids
}

// WorkflowTemplateInstancesToWorkflowIDs returns workflow ids of given workflow template instances.
func WorkflowTemplateInstancesToWorkflowIDs(wtis []*WorkflowTemplateInstance) []int64 {
	ids := make([]int64, len(wtis))
	for i := range wtis {
		if wtis[i].WorkflowID != nil {
			ids[i] = *wtis[i].WorkflowID
		}
	}
	return ids
}

// WorkflowTemplateInstancesToWorkflowTemplateIDs returns workflow template ids of given workflow template instances.
func WorkflowTemplateInstancesToWorkflowTemplateIDs(wtis []*WorkflowTemplateInstance) []int64 {
	ids := make([]int64, len(wtis))
	for i := range wtis {
		ids[i] = wtis[i].WorkflowTemplateID
	}
	return ids
}

// WorkflowTemplateBulk contains info about a template bulk task.
type WorkflowTemplateBulk struct {
	ID                 int64                          `json:"id" db:"id"`
	WorkflowTemplateID int64                          `json:"workflow_template_id" db:"workflow_template_id"`
	Operations         WorkflowTemplateBulkOperations `json:"operations" db:"operations"`
}

// WorkflowTemplateBulkOperation contains one operation of a template bulk task.
type WorkflowTemplateBulkOperation struct {
	Status  OperationStatus         `json:"status"`
	Request WorkflowTemplateRequest `json:"request"`
}

// WorkflowTemplateBulkOperations struct.
type WorkflowTemplateBulkOperations []WorkflowTemplateBulkOperation

// Value returns driver.Value from workflow template bulk operations.
func (w WorkflowTemplateBulkOperations) Value() (driver.Value, error) {
	j, err := json.Marshal(w)
	return j, WrapError(err, "cannot marshal WorkflowTemplateBulkOperations")
}

// Scan pipeline templates.
func (w *WorkflowTemplateBulkOperations) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return WithStack(errors.New("type assertion .([]byte) failed"))
	}
	return WrapError(json.Unmarshal(source, w), "cannot unmarshal WorkflowTemplateBulkOperations")
}
