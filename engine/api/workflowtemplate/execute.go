package workflowtemplate

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

func prepareParams(wt *sdk.WorkflowTemplate, r sdk.WorkflowTemplateRequest) interface{} {
	m := map[string]interface{}{}
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

	tmpl, err := template.New(id).Delims("[[", "]]").Parse(t)
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
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return "", sdk.WithStack(err)
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

// Execute returns yaml file from template.
func Execute(wt *sdk.WorkflowTemplate, instance *sdk.WorkflowTemplateInstance) (sdk.WorkflowTemplateResult, error) {
	result := sdk.WorkflowTemplateResult{
		Pipelines:    make([]string, len(wt.Pipelines)),
		Applications: make([]string, len(wt.Applications)),
		Environments: make([]string, len(wt.Environments)),
	}

	var data map[string]interface{}
	if instance != nil {
		data = map[string]interface{}{
			"id":     instance.ID,
			"name":   instance.Request.WorkflowName,
			"params": prepareParams(wt, instance.Request),
		}
	}

	var parsingErrs []error

	v, err := decodeTemplateValue(wt.Value)
	if err != nil {
		return result, err
	}
	if tmpl, err := parseTemplate("workflow", 0, v); err != nil {
		parsingErrs = append(parsingErrs, err)
	} else {
		if data != nil {
			result.Workflow, err = executeTemplate(tmpl, data)
			if err != nil {
				return result, err
			}
		}
	}

	for i, p := range wt.Pipelines {
		v, err := decodeTemplateValue(p.Value)
		if err != nil {
			return result, err
		}

		if tmpl, err := parseTemplate("pipeline", i, v); err != nil {
			parsingErrs = append(parsingErrs, err)
		} else {
			result.Pipelines[i], err = executeTemplate(tmpl, data)
			if err != nil {
				return result, err
			}
		}
	}

	for i, a := range wt.Applications {
		v, err := decodeTemplateValue(a.Value)
		if err != nil {
			return result, err
		}

		if tmpl, err := parseTemplate("application", i, v); err != nil {
			parsingErrs = append(parsingErrs, err)
		} else {
			if data != nil {
				result.Applications[i], err = executeTemplate(tmpl, data)
				if err != nil {
					return result, err
				}
			}
		}
	}

	for i, e := range wt.Environments {
		v, err := decodeTemplateValue(e.Value)
		if err != nil {
			return result, err
		}

		if tmpl, err := parseTemplate("environment", i, v); err != nil {
			parsingErrs = append(parsingErrs, err)
		} else {
			if data != nil {
				result.Environments[i], err = executeTemplate(tmpl, data)
				if err != nil {
					return result, err
				}
			}
		}
	}

	if len(parsingErrs) > 0 {
		var errs []sdk.WorkflowTemplateError
		var causes []string
		for _, err := range parsingErrs {
			cause := sdk.Cause(err)
			if e, ok := cause.(sdk.WorkflowTemplateError); ok {
				errs = append(errs, e)
			}
			causes = append(causes, cause.Error())
		}
		return result, sdk.NewErrorFrom(sdk.Error{
			ID:     sdk.ErrCannotParseTemplate.ID,
			Status: sdk.ErrCannotParseTemplate.Status,
			Data:   errs,
		}, strings.Join(causes, ", "))
	}

	return result, nil
}

// Tar returns in buffer the a tar file that contains all generated stuff in template result.
func Tar(wt *sdk.WorkflowTemplate, res sdk.WorkflowTemplateResult, w io.Writer) error {
	tw := tar.NewWriter(w)
	defer func() {
		if err := tw.Close(); err != nil {
			log.Error("%v", sdk.WrapError(err, "Unable to close tar writer"))
		}
	}()

	// add generated workflow to writer
	var wor exportentities.Workflow
	if err := yaml.Unmarshal([]byte(res.Workflow), &wor); err != nil {
		return sdk.NewError(sdk.Error{
			ID:      sdk.ErrWrongRequest.ID,
			Message: "Cannot parse generated workflow",
		}, err)
	}

	// set the workflow template instance path on export
	templatePath := fmt.Sprintf("%s/%s", wt.Group.Name, wt.Slug)
	wor.Template = &templatePath

	bs, err := exportentities.Marshal(wor, exportentities.FormatYAML)
	if err != nil {
		return err
	}
	if err := tw.WriteHeader(&tar.Header{
		Name: fmt.Sprintf("%s.yml", wor.Name),
		Mode: 0644,
		Size: int64(len(bs)),
	}); err != nil {
		return sdk.WrapError(err, "Unable to write header for workflow %s", wor.Name)
	}
	if _, err := io.Copy(tw, bytes.NewBuffer(bs)); err != nil {
		return sdk.WrapError(err, "Unable to copy workflow buffer")
	}

	// add generated pipelines to writer
	for _, p := range res.Pipelines {
		var pip exportentities.PipelineV1
		if err := yaml.Unmarshal([]byte(p), &pip); err != nil {
			return sdk.NewError(sdk.Error{
				ID:      sdk.ErrWrongRequest.ID,
				Message: "Cannot parse generated pipeline",
			}, err)
		}

		bs, err := exportentities.Marshal(pip, exportentities.FormatYAML)
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: fmt.Sprintf("%s.pip.yml", pip.Name),
			Mode: 0644,
			Size: int64(len(bs)),
		}); err != nil {
			return sdk.WrapError(err, "Unable to write header for pipeline %s", pip.Name)
		}
		if _, err := io.Copy(tw, bytes.NewBuffer(bs)); err != nil {
			return sdk.WrapError(err, "Unable to copy pipeline buffer")
		}
	}

	// add generated applications to writer
	for _, a := range res.Applications {
		var app exportentities.Application
		if err := yaml.Unmarshal([]byte(a), &app); err != nil {
			return sdk.NewError(sdk.Error{
				ID:      sdk.ErrWrongRequest.ID,
				Message: "Cannot parse generated application",
			}, err)
		}

		bs, err := exportentities.Marshal(app, exportentities.FormatYAML)
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: fmt.Sprintf("%s.app.yml", app.Name),
			Mode: 0644,
			Size: int64(len(bs)),
		}); err != nil {
			return sdk.WrapError(err, "Unable to write header for application %s", app.Name)
		}
		if _, err := io.Copy(tw, bytes.NewBuffer(bs)); err != nil {
			return sdk.WrapError(err, "Unable to copy application buffer")
		}
	}

	// add generated environments to writer
	for _, e := range res.Environments {
		var env exportentities.Environment
		if err := yaml.Unmarshal([]byte(e), &env); err != nil {
			return sdk.NewError(sdk.Error{
				ID:      sdk.ErrWrongRequest.ID,
				Message: "Cannot parse generated environment",
			}, err)
		}

		bs, err := exportentities.Marshal(env, exportentities.FormatYAML)
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: fmt.Sprintf("%s.env.yml", env.Name),
			Mode: 0644,
			Size: int64(len(bs)),
		}); err != nil {
			return sdk.WrapError(err, "Unable to write header for environment %s", env.Name)
		}
		if _, err := io.Copy(tw, bytes.NewBuffer(bs)); err != nil {
			return sdk.WrapError(err, "Unable to copy environment buffer")
		}
	}

	return nil
}
