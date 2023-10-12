package sdk

import (
	"github.com/invopop/jsonschema"
)

func GetWorkerModelJsonSchema() *jsonschema.Schema {
	wmSchema := jsonschema.Reflect(&V2WorkerModel{})
	wmDocker := jsonschema.Reflect(&V2WorkerModelDockerSpec{})
	wmOpenstack := jsonschema.Reflect(&V2WorkerModelOpenstackSpec{})
	wmVSphere := jsonschema.Reflect(&V2WorkerModelVSphereSpec{})

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
	actionSchema := jsonschema.Reflect(&V2Action{})

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

		for _, actName := range publicActionNames {
			stepUses.Enum = append(stepUses.Enum, "actions/"+actName)
		}
	}
	return actionSchema
}

func GetJobJsonSchema(publicActionNames []string, regionNames []string, workerModels []string) *jsonschema.Schema {
	jobSchema := jsonschema.Reflect(&V2Job{})

	propStepUses, _ := jobSchema.Definitions["ActionStep"].Properties.Get("uses")
	stepUses := propStepUses.(*jsonschema.Schema)
	// Enum on step uses
	if len(publicActionNames) > 0 {
		for _, actName := range publicActionNames {
			stepUses.Enum = append(stepUses.Enum, "actions/"+actName)
		}
	}

	// Enum on region
	propRegion, _ := jobSchema.Definitions["V2Job"].Properties.Get("region")
	regionSchema := propRegion.(*jsonschema.Schema)
	if len(regionNames) > 0 {
		for _, regName := range regionNames {
			regionSchema.Enum = append(regionSchema.Enum, regName)
		}
	}

	propWM, _ := jobSchema.Definitions["V2Job"].Properties.Get("worker_model")
	wmSchema := propWM.(*jsonschema.Schema)
	if len(workerModels) > 0 {
		for _, wmName := range workerModels {
			wmSchema.Enum = append(wmSchema.Enum, wmName)
		}
	}

	return jobSchema
}

func GetWorkflowJsonSchema(publicActionNames, regionNames, workerModelNames []string) *jsonschema.Schema {
	workflowSchema := jsonschema.Reflect(&V2Workflow{})
	workflowOn := jsonschema.Reflect(&WorkflowOn{
		Push:           &WorkflowOnPush{},
		ModelUpdate:    &WorkflowOnModelUpdate{},
		WorkflowUpdate: &WorkflowOnWorkflowUpdate{},
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
	workflowSchema.Definitions["WorkflowOnModelUpdate"] = workflowOn.Definitions["WorkflowOnModelUpdate"]
	workflowSchema.Definitions["WorkflowOnWorkflowUpdate"] = workflowOn.Definitions["WorkflowOnWorkflowUpdate"]

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
