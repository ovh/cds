package workflowtemplate

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/ovh/cds/sdk"
)

// Execute returns yaml file from template.
func (t *Template) Execute() (Result, error) {
	data := map[string]interface{}{
		"params": map[string]interface{}{
			"name":       "my-workflow",
			"withDeploy": true,
			"deployWhen": "failure",
		},
	}

	var res Result
	for i, v := range append([]string{t.Workflow}, t.Pipelines...) {
		tmpl, err := template.New(fmt.Sprintf("template-%d", i)).Parse(escapeVars(v))
		if err != nil {
			return Result{}, sdk.WrapError(err, "cannot parse workflow template")
		}

		var buffer bytes.Buffer
		if err := tmpl.Execute(&buffer, data); err != nil {
			return Result{}, sdk.WrapError(err, "cannot execute workflow template")
		}

		output := unescapeVars(buffer.String())
		if i == 0 {
			res.Workflow = output
		} else {
			res.Pipelines = append(res.Pipelines, output)
		}
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
