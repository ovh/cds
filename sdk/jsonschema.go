package sdk

import (
	"strings"

	"github.com/sguiheux/jsonschema"
)

func GetWorkerModelJsonSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{Anonymous: false}
	wmSchema := reflector.Reflect(&V2WorkerModel{})
	wmDocker := reflector.Reflect(&V2WorkerModelDockerSpec{})
	wmOpenstack := reflector.Reflect(&V2WorkerModelOpenstackSpec{})
	wmVSphere := reflector.Reflect(&V2WorkerModelVSphereSpec{})

	if wmSchema.Definitions == nil {
		wmSchema.Definitions = make(map[string]*jsonschema.Schema)
	}
	wmSchema.Definitions["V2WorkerModelVSphereSpec"] = wmVSphere
	wmSchema.Definitions["V2WorkerModelOpenstackSpec"] = wmOpenstack
	wmSchema.Definitions["V2WorkerModelDockerSpec"] = wmDocker

	propName, _ := wmSchema.Definitions["V2WorkerModel"].Properties.Get("name")
	name := propName.(*jsonschema.Schema)
	name.Pattern = EntityNamePattern

	propOSArch, _ := wmSchema.Definitions["V2WorkerModel"].Properties.Get("osarch")
	osArch := propOSArch.(*jsonschema.Schema)
	for _, r := range OSArchRequirementValues.Values() {
		osArch.Enum = append(osArch.Enum, r)
	}

	return wmSchema
}

func GetActionJsonSchema(publicActionNames []string) *jsonschema.Schema {
	reflector := jsonschema.Reflector{Anonymous: false}
	actionSchema := reflector.Reflect(&V2Action{})

	if actionSchema.Definitions == nil {
		actionSchema.Definitions = make(map[string]*jsonschema.Schema)
	}

	propName, _ := actionSchema.Definitions["V2Action"].Properties.Get("name")
	name := propName.(*jsonschema.Schema)
	name.Pattern = EntityNamePattern

	// Pattern on input/output keys
	propInput, _ := actionSchema.Definitions["V2Action"].Properties.Get("inputs")
	input := propInput.(*jsonschema.Schema)
	input.PatternProperties[EntityActionInputKey] = input.PatternProperties[".*"]
	delete(input.PatternProperties, ".*")

	propOutput, _ := actionSchema.Definitions["V2Action"].Properties.Get("outputs")
	output := propOutput.(*jsonschema.Schema)
	output.PatternProperties[EntityActionInputKey] = output.PatternProperties[".*"]
	delete(output.PatternProperties, ".*")

	// Pattern on step id
	propId, _ := actionSchema.Definitions["ActionStep"].Properties.Get("id")
	stepId := propId.(*jsonschema.Schema)
	stepId.Pattern = EntityActionStepID

	propStepUses, _ := actionSchema.Definitions["ActionStep"].Properties.Get("uses")
	stepUses := propStepUses.(*jsonschema.Schema)
	// Enum on step uses
	if len(publicActionNames) > 0 {
		stepUses.AnyOf = make([]*jsonschema.Schema, 0)
		anyOfEnum := &jsonschema.Schema{}
		for _, actName := range publicActionNames {
			anyOfEnum.Enum = append(anyOfEnum.Enum, "actions/"+actName)
		}
		stepUses.AnyOf = append(stepUses.AnyOf, anyOfEnum)
		stepUses.AnyOf = append(stepUses.AnyOf, &jsonschema.Schema{
			Pattern: "^.cds/actions/.*.(yaml|yml)",
		})
	}
	return actionSchema
}

func GetJobJsonSchema(publicActionNames []string, regionNames []string, workerModels []string) *jsonschema.Schema {
	reflector := jsonschema.Reflector{Anonymous: false}
	jobSchema := reflector.Reflect(&V2Job{})

	propStepUses, _ := jobSchema.Definitions["ActionStep"].Properties.Get("uses")
	stepUses := propStepUses.(*jsonschema.Schema)

	// Enum on step uses
	if len(publicActionNames) > 0 {
		stepUses.AnyOf = make([]*jsonschema.Schema, 0)
		anyOfEnum := &jsonschema.Schema{}
		for _, actName := range publicActionNames {
			anyOfEnum.Enum = append(anyOfEnum.Enum, "actions/"+actName)
		}
		stepUses.AnyOf = append(stepUses.AnyOf, anyOfEnum)
		stepUses.AnyOf = append(stepUses.AnyOf, &jsonschema.Schema{
			Pattern: "^.cds/actions/.*.(yaml|yml)",
		})
	}

	// Enum on region
	propRegion, _ := jobSchema.Definitions["V2Job"].Properties.Get("region")
	regionSchema := propRegion.(*jsonschema.Schema)
	if len(regionNames) > 0 {
		for _, regName := range regionNames {
			regionSchema.Enum = append(regionSchema.Enum, regName)
		}
	}

	propWM, _ := jobSchema.Definitions["V2Job"].Properties.Get("runs-on")
	wmSchema := propWM.(*jsonschema.Schema)
	if len(workerModels) > 0 {
		wmSchema.AnyOf = make([]*jsonschema.Schema, 0)
		enumSchema := &jsonschema.Schema{}
		for _, wmName := range workerModels {
			enumSchema.Enum = append(enumSchema.Enum, wmName)
		}
		wmSchema.AnyOf = append(wmSchema.AnyOf, enumSchema)
		wmSchema.AnyOf = append(wmSchema.AnyOf, &jsonschema.Schema{
			Pattern: "^.cds/worker-models/.*.(yaml|yml)",
		})
	}

	return jobSchema
}

func GetWorkflowJsonSchema(publicActionNames, regionNames, workerModelNames []string) *jsonschema.Schema {
	reflector := jsonschema.Reflector{Anonymous: false}
	workflowSchema := reflector.Reflect(&V2Workflow{})
	workflowOn := reflector.Reflect(&WorkflowOn{
		ModelUpdate:        &WorkflowOnModelUpdate{},
		PullRequest:        &WorkflowOnPullRequest{},
		PullRequestComment: &WorkflowOnPullRequestComment{},
		Push:               &WorkflowOnPush{},
		WorkflowUpdate:     &WorkflowOnWorkflowUpdate{},
	})

	jobSchema := GetJobJsonSchema(publicActionNames, regionNames, workerModelNames)
	actionStepSchema := GetActionJsonSchema(publicActionNames)

	if workflowSchema.Definitions == nil {
		workflowSchema.Definitions = make(map[string]*jsonschema.Schema)
	}

	propName, _ := workflowSchema.Definitions["V2Workflow"].Properties.Get("name")
	name := propName.(*jsonschema.Schema)
	name.Pattern = EntityNamePattern

	// Pattern jobs key
	propsJobs, _ := workflowSchema.Definitions["V2Workflow"].Properties.Get("jobs")
	jobs := propsJobs.(*jsonschema.Schema)
	jobs.PatternProperties[EntityActionInputKey] = jobs.PatternProperties[".*"]
	delete(jobs.PatternProperties, ".*")

	workflowSchema.Definitions["ActionStep"] = actionStepSchema.Definitions["ActionStep"]
	workflowSchema.Definitions["V2Job"] = jobSchema.Definitions["V2Job"]
	workflowSchema.Definitions["WorkflowOn"] = workflowOn.Definitions["WorkflowOn"]
	workflowSchema.Definitions["WorkflowOnPush"] = workflowOn.Definitions["WorkflowOnPush"]
	workflowSchema.Definitions["WorkflowOnPullRequest"] = workflowOn.Definitions["WorkflowOnPullRequest"]
	workflowSchema.Definitions["WorkflowOnPullRequestComment"] = workflowOn.Definitions["WorkflowOnPullRequestComment"]
	workflowSchema.Definitions["WorkflowOnModelUpdate"] = workflowOn.Definitions["WorkflowOnModelUpdate"]
	workflowSchema.Definitions["WorkflowOnWorkflowUpdate"] = workflowOn.Definitions["WorkflowOnWorkflowUpdate"]
	workflowSchema.Definitions["WorkflowOnSchedule"] = workflowOn.Definitions["WorkflowOnSchedule"]
	workflowSchema.Definitions["WorkflowOnRun"] = workflowOn.Definitions["WorkflowOnRun"]

	// Prop On
	propsOn := &jsonschema.Schema{
		OneOf: []*jsonschema.Schema{
			{
				Ref: "#/$defs/WorkflowOn",
			},
			{
				Type: "array",
				Items: &jsonschema.Schema{
					Type: "string",
				},
			},
		},
	}
	workflowSchema.Definitions["V2Workflow"].Properties.Set("on", propsOn)

	return workflowSchema
}

func GetWorkflowTemplateJsonSchema() *jsonschema.Schema {
	reflector := jsonschema.Reflector{Anonymous: false}
	templateSchema := reflector.Reflect(&V2WorkflowTemplate{})

	// Replace spec with type string instead of object
	spec, _ := templateSchema.Definitions["V2WorkflowTemplate"].Properties.Get("spec")
	if schema, ok := spec.(*jsonschema.Schema); ok {
		schema.Type = "string"
		schema.Ref = ""
		templateSchema.Definitions["V2WorkflowTemplate"].Properties.Set("spec", schema)
		delete(templateSchema.Definitions, "WorkflowSpec")
	}

	return templateSchema
}

// Generate documented yaml from jsonschema
func GetYamlFromJsonSchema(schema *jsonschema.Schema) (string, error) {
	var buf strings.Builder
	gen := &YAMLGenerator{
		indent:      "  ",
		commentChar: "#",
	}
	if err := gen.Generate(&buf, schema); err != nil {
		return "", err
	}
	return buf.String(), nil
}
