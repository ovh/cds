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

type V2WorkflowTemplateParamType string

const (
	V2WorkflowTemplateParamTypeString V2WorkflowTemplateParamType = "string"
	V2WorkflowTemplateParamTypeJson   V2WorkflowTemplateParamType = "json"
)

type V2WorkflowTemplate struct {
	Name        string                        `json:"name" jsonschema_extras:"order=1" jsonschema_description:"Name of the workflow templates"`
	Description string                        `json:"description,omitempty" jsonschema_extras:"order=2" jsonschema_description:"Description of the workflow template"`
	Parameters  []V2WorkflowTemplateParameter `json:"parameters" jsonschema_extras:"order=3" jsonschema_description:"Array of parameters"`
	Spec        WorkflowSpec                  `json:"spec" jsonschema_extras:"order=4,code=true" jsonschema_description:"Workflow definition"`
}

type V2WorkflowTemplateParameter struct {
	Key      string                      `json:"key" jsonschema_extras:"order=1" jsonschema_description:"Name of the parameter"`
	Type     V2WorkflowTemplateParamType `json:"type,omitempty" jsonschema_extras:"order=2" jsonschema_description:"Type of the parameter"`
	Required bool                        `json:"required,omitempty" jsonschema_extras:"order=3" jsonschema_description:"Indicate if the parameter is mandatory"`
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
		return []error{NewErrorFrom(ErrInvalidData, "unable to validate workflow template: "+err.Error())}
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

func (wt V2WorkflowTemplate) Resolve(_ context.Context, w *V2Workflow) (string, error) {
	type innerWorkflow struct {
		Stages       map[string]WorkflowStage `json:"stages,omitempty"`
		Gates        map[string]V2JobGate     `json:"gates,omitempty"`
		Jobs         map[string]V2Job         `json:"jobs"`
		Env          map[string]string        `json:"env,omitempty"`
		Integrations []string                 `json:"integrations,omitempty"`
		VariableSets []string                 `json:"vars,omitempty"`
		Annotations  map[string]string        `json:"annotations,omitempty"`
	}

	if wt.Spec.tpl == nil {
		return "", errors.New("uninitialized workflow spec")
	}

	paramsDef := make(map[string]V2WorkflowTemplateParamType)
	for _, v := range wt.Parameters {
		paramsDef[v.Key] = v.Type
	}

	params := make(map[string]interface{})
	for k, v := range w.Parameters {
		paramType, has := paramsDef[k]
		if has {
			switch paramType {
			case V2WorkflowTemplateParamTypeJson:
				var value interface{}
				if err := json.Unmarshal([]byte(v), &value); err != nil {
					return "", NewErrorFrom(ErrWrongRequest, "unable to unmarshal %s", v)
				}
				params[k] = value
			default:
				params[k] = v
			}
		}
	}

	var buf bytes.Buffer
	if err := wt.Spec.tpl.Execute(&buf, map[string]map[string]interface{}{
		"params": params,
	}); err != nil {
		return "", err
	}

	var in innerWorkflow
	if err := yaml.Unmarshal(buf.Bytes(), &in); err != nil {
		return "", err
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
	// Use workflow template annotations only if
	// they are not already defined by the template
	// or their value is an empty string.
	if len(in.Annotations) > 0 {
		if w.Annotations == nil {
			w.Annotations = make(map[string]string)
		}
		for k, v := range in.Annotations {
			if _, ok := w.Annotations[k]; !ok {
				w.Annotations[k] = v
			}
		}
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
