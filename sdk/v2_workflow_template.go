package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"text/template"

	"github.com/ovh/cds/sdk/interpolate"
	"github.com/rockbears/yaml"
	"github.com/xeipuuv/gojsonschema"
)

var _ Lintable = V2WorkflowTemplate{}

type V2WorkflowTemplate struct {
	Name        string                     `json:"name"`
	Description string                     `json:"description,omitempty"`
	Parameters  WorkflowTemplateParameters `json:"parameters"`
	Spec        WorkflowSpec               `json:"spec"`
}

type V2WorkflowTemplateGenerateRequest struct {
	Template V2WorkflowTemplate `json:"template"`
	Params   map[string]string  `json:"params"`
}

type V2WorkflowTemplateGenerateResponse struct {
	Error    string `json:"error" cli:"error"`
	Workflow string `json:"workflow" cli:"workflow"`
}

func (wt V2WorkflowTemplate) Lint() (errs []error) {
	schema := GetWorkflowTemplateJsonSchema()
	rawSchema, err := schema.MarshalJSON()
	if err != nil {
		return []error{NewErrorFrom(err, "unable to load workflow schema")}
	}
	schemaLoader := gojsonschema.NewStringLoader(string(rawSchema))

	rawModel, err := json.Marshal(wt)
	if err != nil {
		return []error{NewErrorFrom(err, "unable to marshal workflow")}
	}
	documentLoader := gojsonschema.NewStringLoader(string(rawModel))

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return []error{NewErrorFrom(err, "unable to validate workflow template")}
	}

	for _, e := range result.Errors() {
		errs = append(errs, NewErrorFrom(ErrInvalidData, "yaml validation failed: "+e.String()))
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (wt V2WorkflowTemplate) GetName() string {
	return wt.Name
}

func (wt V2WorkflowTemplate) Resolve(ctx context.Context, w *V2Workflow) (string, error) {
	type innerWorkflow struct {
		Stages       map[string]WorkflowStage `json:"stages,omitempty"`
		Gates        map[string]V2JobGate     `json:"gates,omitempty"`
		Jobs         map[string]V2Job         `json:"jobs"`
		Env          map[string]string        `json:"env,omitempty"`
		Integrations []string                 `json:"integrations,omitempty"`
		VariableSets []string                 `json:"vars,omitempty"`
	}

	if wt.Spec.tpl == nil {
		return "", errors.New("uninitiliazed workflow spec")
	}

	var buf bytes.Buffer
	if err := wt.Spec.tpl.Execute(&buf, map[string]any{
		"params": w.Parameters,
	}); err != nil {
		return "", err
	}

	var in innerWorkflow
	if err := yaml.Unmarshal(buf.Bytes(), &in); err != nil {
		return buf.String(), err
	}

	// fill workflow
	if in.Stages != nil {
		w.Stages = in.Stages
	}
	if in.Gates != nil {
		w.Gates = in.Gates
	}
	if in.Jobs != nil {
		w.Jobs = in.Jobs
	}
	if in.Env != nil {
		w.Env = in.Env
	}
	if in.Integrations != nil {
		w.Integrations = in.Integrations
	}
	if in.VariableSets != nil {
		w.VariableSets = in.VariableSets
	}

	return buf.String(), nil
}

type WorkflowSpec struct {
	tpl *template.Template
	raw json.RawMessage
}

func (t WorkflowSpec) MarshalJSON() ([]byte, error) {
	return t.raw, nil
}

func (t *WorkflowSpec) UnmarshalJSON(data []byte) error {
	t.raw = data
	var strData string
	if err := json.Unmarshal(data, &strData); err != nil {
		return WithStack(err)
	}

	tpl, err := template.New("workflow_template").Funcs(interpolate.InterpolateHelperFuncs).Delims("[[", "]]").Parse(strData)
	if err != nil {
		return WithStack(err)
	}

	t.tpl = tpl
	return nil
}
