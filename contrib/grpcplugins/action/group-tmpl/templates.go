package main

import (
	"bytes"
	"strings"
	"text/template"
)

var funcMap template.FuncMap = template.FuncMap{
	"title": strings.Title,
	"lower": strings.ToLower,
	"upper": strings.ToUpper,
	"split": strings.Split,
	"join":  strings.Join,
}

const outputBody = `
{
    {{- $configs := .Configs}}
    {{- $subApplications := .SubApplications}}
    "apps": [
        {{range $index, $subApplication := $subApplications}}
        {{ if gt $index 0 }},{{ end }}
           {{ index $configs $subApplication}}
        {{end}}
     ]
}
`

type outputBodyVars struct {
	Configs         map[string]string
	SubApplications []string
}

var outputBodyTemplate *template.Template = template.Must(template.New("outputBody").Funcs(funcMap).Parse(outputBody))

func executeTemplate(tmpl *template.Template, vars interface{}) (string, error) {
	buf := new(bytes.Buffer)
	err := tmpl.Execute(buf, vars)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
