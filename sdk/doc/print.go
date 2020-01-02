package doc

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/template"

	"github.com/ovh/cds/sdk"
)

const sectionTemplate = `+++
title = "{{.Title}}"
+++
{{range .Routes}}
## {{.Title}}

URL         | **` + "`{{.URL}}`" + `**
----------- |----------
Method      | {{.Method}}     
{{- if .QueryParams -}}
{{range .QueryParams}}
Query Parameter | {{.}}
{{- end}}
{{- end}}
Permissions | {{.Permissions}}
{{- if .Scopes}}
Scopes | {{.Scopes}}
{{- end}}
Code        | {{.Code}}    

{{- if .Description}}
### Description
{{ .Description}}
{{- end}}

{{- if .RequestBody}}
### Request Body
` + "```" + `
{{ .RequestBody}}
` + "```" + `
{{- end}}

{{- if .ResponseBody}}
### Response Body
` + "```" + `
{{ .ResponseBody}}
` + "```" + `
{{- end}}
{{end -}}
`

func printSection(name string, docs []Doc, writer io.Writer) error {
	t, err := template.New("routes").Parse(sectionTemplate)
	if err != nil {
		return err
	}

	dataPage := pageTmpl{
		Title:  name,
		Routes: []routeTmpl{},
	}

	sort.Slice(docs, func(i, j int) bool {
		titlea := docTitle(docs[i])
		titleb := docTitle(docs[j])
		return titlea < titleb
	})
	for _, doc := range docs {
		route := routeTmpl{}
		route.Title = docTitle(doc)

		var permissions []string
		var noAuth bool
		for _, v := range doc.Middlewares {
			permissions = append(permissions, fmt.Sprintf("%s: %s", v.Name, strings.Join(v.Value, ",")))
			if v.Name == "Auth" {
				for _, value := range v.Value {
					if value == sdk.FalseString {
						noAuth = true
						break
					}
				}
			}
		}
		if !noAuth {
			permissions = append(permissions, "Auth: true")
		}
		route.Permissions = strings.Join(permissions, " - ")

		route.Scopes = strings.Join(doc.Scopes, ", ")
		route.URL = doc.URL
		route.Method = doc.HTTPOperation
		route.QueryParams = doc.QueryParams
		route.Code = fmt.Sprintf("[%s](https://github.com/ovh/cds/search?q=%%22func+%%28api+*API%%29+%s%%22)\n", doc.Method, doc.Method)
		route.Description = doc.Description
		route.RequestBody = doc.RequestBody
		route.ResponseBody = doc.ResponseBody
		dataPage.Routes = append(dataPage.Routes, route)
	}

	if err := t.Execute(writer, dataPage); err != nil {
		return err
	}

	return nil
}
