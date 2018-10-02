package workflowtemplate

import "reflect"

// Template struct.
type Template struct {
	Params    map[string]reflect.Kind
	Workflow  string   `json:"workflow"`
	Pipelines []string `json:"pipelines"`
}

// Result struct.
type Result struct {
	Workflow  string   `json:"workflow"`
	Pipelines []string `json:"pipelines"`
}

// GetAll returns all existing templates.
func GetAll() []Template {
	return []Template{{
		Params: map[string]reflect.Kind{
			"name":       reflect.String,
			"withDeploy": reflect.Bool,
			"deployWhen": reflect.String,
			"listNodes":  reflect.String,
		},
		Workflow: `
    name: {{.params.name}}
    description: Test simple workflow
    version: v1.0
    workflow:
      Node-1:
        pipeline: Pipeline-1
      {{if .params.withDeploy}}
      Node-2:
        depends_on:
        - Node-1
        when:
        - {{.params.deployWhen}}
        pipeline: Pipeline-1
      {{end}}`,
		Pipelines: []string{`
    version: v1.0
    name: Pipeline-1
    stages:
    - Stage 1
    jobs:
    - job: Job 1
      stage: Stage 1
      steps:
      - script:
        - echo "Hello World!"
      - script:
        - echo "{{.cds.project.name}}"
    `},
	}}
}
