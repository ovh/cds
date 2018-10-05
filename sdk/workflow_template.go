package sdk

import (
	"database/sql/driver"
	json "encoding/json"
	"fmt"
)

// WorkflowTemplateRequest struct use for execution request.
type WorkflowTemplateRequest struct {
	Name       string            `json:"name"`
	Parameters map[string]string `json:"parameters"`
}

// WorkflowTemplateResult struct.
type WorkflowTemplateResult struct {
	Workflow  string   `json:"workflow"`
	Pipelines []string `json:"pipelines"`
}

// WorkflowTemplate struct.
type WorkflowTemplate struct {
	ID         int64                      `json:"id" db:"id" `
	Name       string                     `json:"name" db:"name"`
	Parameters WorkflowTemplateParameters `json:"parameters" db:"parameters"`
	Value      string                     `json:"value" db:"value"`
	Pipelines  PipelineTemplates          `json:"pipelines" db:"pipelines"`
}

// ValidateStruct returns workflow template validity.
func (w *WorkflowTemplate) ValidateStruct() error {
	if w.Name == "" || len(w.Value) == 0 {
		return ErrInvalidData
	}

	for _, p := range w.Pipelines {
		if err := p.ValidateStruct(); err != nil {
			return err
		}
	}

	for _, p := range w.Parameters {
		if err := p.ValidateStruct(); err != nil {
			return err
		}
	}

	return nil
}

// CheckParams returns template parameters validity.
func (w *WorkflowTemplate) CheckParams(r WorkflowTemplateRequest) error {
	if r.Name == "" {
		return WrapError(ErrInvalidData, "Name is required")
	}

	for _, p := range w.Parameters {
		v, ok := r.Parameters[p.Key]
		if !ok && p.Required {
			return WrapError(ErrInvalidData, "Param %s is required", p.Key)
		}
		if ok {
			if p.Required && v == "" {
				return WrapError(ErrInvalidData, "Param %s is required", p.Key)
			}
			if p.Type == ParameterTypeBoolean && v != "" && !(v == "true" || v == "false") {
				return WrapError(ErrInvalidData, "Given value it's not a boolean for %s", p.Key)
			}
		}
	}

	return nil
}

// PipelineTemplate struct.
type PipelineTemplate struct {
	Value string `json:"value"`
}

// ValidateStruct returns pipeline template validity.
func (p *PipelineTemplate) ValidateStruct() error {
	if len(p.Value) == 0 {
		return ErrInvalidData
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
		return fmt.Errorf("WorkflowTemplateParameters> type assertion .([]byte) failed") // TODO withstack
	}
	return WrapError(json.Unmarshal(source, w), "cannot unmarshal WorkflowTemplateParameters")
}

// PipelineTemplates struct.
type PipelineTemplates []PipelineTemplate

// Value returns driver.Value from workflow template parameters.
func (p PipelineTemplates) Value() (driver.Value, error) {
	j, err := json.Marshal(p)
	return j, WrapError(err, "cannot marshal PipelineTemplates")
}

// Scan workflow template parameters.
func (p *PipelineTemplates) Scan(src interface{}) error {
	source, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("PipelineTemplates> type assertion .([]byte) failed") // TODO withstack
	}
	return WrapError(json.Unmarshal(source, p), "cannot unmarshal WorkflowTemplateParameters")
}

// ValidateStruct returns pipeline template validity.
func (w *WorkflowTemplateParameter) ValidateStruct() error {
	if w.Key == "" || !w.Type.IsValid() {
		return ErrInvalidData
	}
	return nil
}
