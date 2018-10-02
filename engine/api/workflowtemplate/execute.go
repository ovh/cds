package workflowtemplate

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/ovh/cds/sdk"
)

// CheckParams returns template parameters validity.
func (t *Template) CheckParams(r Request) error {
	if r.Name == "" {
		return sdk.ErrInvalidData
	}

	for _, p := range t.Parameters {
		v, ok := r.Parameters[p.Key]
		if !ok && p.Required {
			return sdk.ErrInvalidData
		}
		if ok {
			if p.Required && v == "" {
				return sdk.ErrInvalidData
			}
			if p.Type == Boolean && v != "" && !(v == "true" || v == "false") {
				return sdk.ErrInvalidData
			}
		}
	}

	return nil
}

func (t *Template) prepareParams(r Request) interface{} {
	m := map[string]interface{}{}
	for _, p := range t.Parameters {
		v, ok := r.Parameters[p.Key]
		if ok {
			switch p.Type {
			case Boolean:
				m[p.Key] = v == "true"
			default:
				m[p.Key] = v
			}
		}
	}
	return m
}

// Execute returns yaml file from template.
func (t *Template) Execute(r Request) (Result, error) {
	data := map[string]interface{}{
		"name":   r.Name,
		"params": t.prepareParams(r),
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
