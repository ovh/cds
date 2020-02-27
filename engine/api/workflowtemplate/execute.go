package workflowtemplate

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/interpolate"
)

func prepareParams(wt sdk.WorkflowTemplate, r sdk.WorkflowTemplateRequest) interface{} {
	m := make(map[string]interface{}, len(wt.Parameters))
	for _, p := range wt.Parameters {
		v, ok := r.Parameters[p.Key]
		if ok {
			switch p.Type {
			case sdk.ParameterTypeBoolean:
				m[p.Key] = v == "true"
			case sdk.ParameterTypeRepository:
				sp := strings.Split(v, "/")
				m[p.Key] = map[string]string{
					"vcs":        sp[0],
					"repository": strings.Join(sp[1:], "/"),
				}
			case sdk.ParameterTypeJSON:
				var res interface{}
				// safely ignore the error because the value of v has been validated on apply submit
				_ = json.Unmarshal([]byte(v), &res)
				m[p.Key] = res
			default:
				m[p.Key] = v
			}
		}
	}
	return m
}

func parseTemplate(templateType string, number int, t string) (*template.Template, error) {
	var id string
	switch templateType {
	case "workflow":
		id = templateType
	default:
		id = fmt.Sprintf("%s.%d", templateType, number)
	}

	tmpl, err := template.New(id).Delims("[[", "]]").Funcs(interpolate.InterpolateHelperFuncs).Parse(t)
	if err != nil {
		reg := regexp.MustCompile(`template: ([0-9a-zA-Z.]+):([0-9]+): (.*)$`)
		submatch := reg.FindStringSubmatch(err.Error())
		if len(submatch) != 4 {
			return nil, sdk.WithStack(err)
		}
		line, err := strconv.Atoi(submatch[2])
		if err != nil {
			return nil, sdk.WithStack(err)
		}
		return nil, sdk.WithStack(sdk.WorkflowTemplateError{
			Type:    templateType,
			Number:  number,
			Line:    line,
			Message: submatch[3],
		})
	}
	return tmpl, nil
}

func executeTemplate(tmpl *template.Template, data map[string]interface{}) (string, error) {
	if data == nil {
		return "", nil
	}
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return "", sdk.NewError(sdk.ErrWrongRequest, sdk.WithStack(err))
	}
	return buffer.String(), nil
}

func decodeTemplateValue(value string) (string, error) {
	v, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", sdk.NewError(sdk.ErrWrongRequest, err)
	}
	return string(v), nil
}

// Parse return a template with parsed content.
func Parse(wt sdk.WorkflowTemplate) (sdk.WorkflowTemplateParsed, error) {
	result := sdk.WorkflowTemplateParsed{
		Pipelines:    make([]*template.Template, len(wt.Pipelines)),
		Applications: make([]*template.Template, len(wt.Applications)),
		Environments: make([]*template.Template, len(wt.Environments)),
	}

	var multiErr sdk.MultiError

	v, err := decodeTemplateValue(wt.Workflow)
	if err != nil {
		return result, err
	}
	result.Workflow, err = parseTemplate("workflow", 0, v)
	if err != nil {
		multiErr.Append(err)
	}

	for i, p := range wt.Pipelines {
		v, err := decodeTemplateValue(p.Value)
		if err != nil {
			return result, err
		}
		result.Pipelines[i], err = parseTemplate("pipeline", i, v)
		if err != nil {
			multiErr.Append(err)
		}
	}

	for i, a := range wt.Applications {
		v, err := decodeTemplateValue(a.Value)
		if err != nil {
			return result, err
		}
		result.Applications[i], err = parseTemplate("application", i, v)
		if err != nil {
			multiErr.Append(err)
		}
	}

	for i, e := range wt.Environments {
		v, err := decodeTemplateValue(e.Value)
		if err != nil {
			return result, err
		}
		result.Environments[i], err = parseTemplate("environment", i, v)
		if err != nil {
			multiErr.Append(err)
		}
	}

	if !multiErr.IsEmpty() {
		var errs []sdk.WorkflowTemplateError
		causes := make([]string, len(multiErr))
		for i, err := range multiErr {
			cause := sdk.Cause(err)
			if e, ok := cause.(sdk.WorkflowTemplateError); ok {
				errs = append(errs, e)
			}
			causes[i] = cause.Error()
		}
		return result, sdk.NewErrorFrom(sdk.Error{
			ID:     sdk.ErrCannotParseTemplate.ID,
			Status: sdk.ErrCannotParseTemplate.Status,
			Data:   errs,
		}, strings.Join(causes, ", "))
	}

	return result, nil
}

// Execute returns yaml file from template.
func Execute(wt sdk.WorkflowTemplate, instance sdk.WorkflowTemplateInstance) (exportentities.WorkflowComponents, error) {
	result := exportentities.WorkflowComponents{
		Pipelines:    make([]exportentities.PipelineV1, len(wt.Pipelines)),
		Applications: make([]exportentities.Application, len(wt.Applications)),
		Environments: make([]exportentities.Environment, len(wt.Environments)),
	}

	data := map[string]interface{}{
		"id":     instance.ID,
		"name":   instance.Request.WorkflowName,
		"params": prepareParams(wt, instance.Request),
	}

	parsedTemplate, err := Parse(wt)
	if err != nil {
		return result, err
	}

	workflowYaml, err := executeTemplate(parsedTemplate.Workflow, data)
	if err != nil {
		return result, err
	}
	result.Workflow, err = exportentities.UnmarshalWorkflow([]byte(workflowYaml))
	if err != nil {
		return result, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot parse generated workflow"))
	}

	for i := range parsedTemplate.Pipelines {
		pipelineYaml, err := executeTemplate(parsedTemplate.Pipelines[i], data)
		if err != nil {
			return result, err
		}
		if err := yaml.Unmarshal([]byte(pipelineYaml), &result.Pipelines[i]); err != nil {
			return result, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot parse generated pipeline"))
		}
	}

	for i := range parsedTemplate.Applications {
		applicationYaml, err := executeTemplate(parsedTemplate.Applications[i], data)
		if err != nil {
			return result, err
		}
		if err := yaml.Unmarshal([]byte(applicationYaml), &result.Applications[i]); err != nil {
			return result, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot parse generated application"))
		}
	}

	for i := range parsedTemplate.Environments {
		environmentYaml, err := executeTemplate(parsedTemplate.Environments[i], data)
		if err != nil {
			return result, err
		}
		if err := yaml.Unmarshal([]byte(environmentYaml), &result.Environments[i]); err != nil {
			return result, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot parse generated environment"))
		}
	}

	return result, nil
}
