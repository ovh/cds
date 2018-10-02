package workflowtemplate

// GetAll returns all existing templates.
func GetAll() []Template {
	return []Template{{
		Parameters: []Parameter{
			{Key: "name", Type: String, Required: true},
			{Key: "withDeploy", Type: Boolean, Required: true},
			{Key: "deployWhen", Type: String},
		},
		Workflow: `
    name: {{.name}}
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
