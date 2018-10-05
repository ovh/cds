package workflowtemplate

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/ovh/cds/sdk"
)

func prepareParams(wt *sdk.WorkflowTemplate, r sdk.WorkflowTemplateRequest) interface{} {
	m := map[string]interface{}{}
	for _, p := range wt.Parameters {
		v, ok := r.Parameters[p.Key]
		if ok {
			switch p.Type {
			case sdk.ParameterTypeBoolean:
				m[p.Key] = v == "true"
			default:
				m[p.Key] = v
			}
		}
	}
	return m
}

func executeTemplate(t string, data map[string]interface{}) (string, error) {
	tmpl, err := template.New(fmt.Sprintf("template")).Parse(escapeVars(t))
	if err != nil {
		return "", sdk.WrapError(err, "cannot parse workflow template")
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return "", sdk.WrapError(err, "cannot execute workflow template")
	}

	return unescapeVars(buffer.String()), nil
}

// Execute returns yaml file from template.
func Execute(wt *sdk.WorkflowTemplate, r sdk.WorkflowTemplateRequest) (sdk.WorkflowTemplateResult, error) {
	data := map[string]interface{}{
		"name":   r.Name,
		"params": prepareParams(wt, r),
	}

	v, err := base64.StdEncoding.DecodeString(wt.Value)
	if err != nil {
		return sdk.WorkflowTemplateResult{}, sdk.WrapError(err, "cannot parse workflow template")
	}

	out, err := executeTemplate(string(v), data)
	if err != nil {
		return sdk.WorkflowTemplateResult{}, err
	}

	res := sdk.WorkflowTemplateResult{
		Workflow:  out,
		Pipelines: make([]string, len(wt.Pipelines)),
	}

	for i, p := range wt.Pipelines {
		v, err := base64.StdEncoding.DecodeString(p.Value)
		if err != nil {
			return sdk.WorkflowTemplateResult{}, sdk.WrapError(err, "cannot parse workflow template")
		}

		out, err := executeTemplate(string(v), data)
		if err != nil {
			return sdk.WorkflowTemplateResult{}, err
		}
		res.Pipelines[i] = out
	}

	return res, nil
}

var cdsVarsRegex = regexp.MustCompile("({{[\\.\"a-zA-Z0-9._\\-µ|\\s]+[\\.\"a-zA-Z0-9._\\-µ|\\s]+}})")
var cdsEscapedVarsRegex = regexp.MustCompile("([[[\\.\"a-zA-Z0-9._\\-µ|\\s]+]])")

func escapeVars(input string) string {
	var oldNew []string
	for _, match := range cdsVarsRegex.FindAllStringSubmatch(input, -1) {
		if len(match) > 0 {
			if strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(match[0], "{{")), ".cds") {
				oldNew = append(oldNew, match[0], strings.Replace(
					strings.Replace(match[0], "{{", "[[", -1),
					"}}", "]]", -1))
			}
		}
	}
	return strings.NewReplacer(oldNew...).Replace(input)
}

func unescapeVars(input string) string {
	var oldNew []string
	for _, match := range cdsEscapedVarsRegex.FindAllStringSubmatch(input, -1) {
		if len(match) > 0 {
			oldNew = append(oldNew, match[0], strings.Replace(
				strings.Replace(match[0], "[[", "{{", -1),
				"]]", "}}", -1))
		}
	}
	return strings.NewReplacer(oldNew...).Replace(input)
}
