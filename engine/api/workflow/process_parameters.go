package workflow

import (
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func getNodeJobRunParameters(db gorp.SqlExecutor, j sdk.Job, run *sdk.WorkflowNodeRun, stage *sdk.Stage) ([]sdk.Parameter, error) {
	params, err := getNodeRunParameters(db, run)
	if err != nil {
		return nil, err
	}

	tmp := map[string]string{}

	tmp["cds.stage"] = stage.Name
	tmp["cds.job"] = j.Action.Name
	errm := &sdk.MultiError{}

	for k, v := range tmp {
		s, err := sdk.Interpolate(v, tmp)
		if err != nil {
			errm.Append(err)
			continue
		}
		sdk.AddParameter(&params, k, sdk.StringParameter, s)
	}

	if errm.IsEmpty() {
		return params, nil
	}

	return params, errm
}

func getNodeRunParameters(db gorp.SqlExecutor, run *sdk.WorkflowNodeRun) ([]sdk.Parameter, error) {
	//Load workflow run
	w, err := loadRunByID(db, run.WorkflowRunID)
	if err != nil {
		return nil, sdk.WrapError(err, "getNodeRunParameters> Unable to load workflow run")
	}

	//Load node definition
	n := w.Workflow.GetNode(run.WorkflowNodeID)
	if n == nil {
		return nil, sdk.WrapError(fmt.Errorf("Unable to find node %d in workflow", run.WorkflowNodeID), "getNodeRunParameters>")
	}
	vars := map[string]string{}

	//Load project
	proj, err := project.Load(db, w.Workflow.ProjectKey, nil, project.LoadOptions.WithVariables)
	if err != nil {
		return nil, sdk.WrapError(err, "getNodeRunParameters> Unable to load project")
	}
	tmp := sdk.ParametersFromProjectVariables(proj)
	for k, v := range tmp {
		vars[k] = v
	}

	// compute application variables
	if n.Context != nil && n.Context.Application != nil {
		vars["cds.application"] = n.Context.Application.Name
		tmp := sdk.ParametersFromApplicationVariables(n.Context.Application)
		for k, v := range tmp {
			vars[k] = v
		}
	}

	// compute environment variables
	if n.Context != nil && n.Context.Environment != nil {
		vars["cds.environment"] = n.Context.Environment.Name
		tmp := sdk.ParametersFromEnvironmentVariables(n.Context.Environment)
		for k, v := range tmp {
			vars[k] = v
		}
	}

	// compute pipeline parameters
	tmp = sdk.ParametersFromPipelineParameters(run.PipelineParameters)
	for k, v := range tmp {
		vars[k] = v
	}

	// compute payload
	tmp = sdk.ParametersToMap(run.Payload)

	tmp["cds.project"] = w.Workflow.ProjectKey
	tmp["cds.workflow"] = w.Workflow.Name
	tmp["cds.pipeline"] = n.Pipeline.Name
	tmp["cds.version"] = fmt.Sprintf("%d.%d", run.Number, run.SubNumber)
	tmp["cds.run"] = fmt.Sprintf("%d.%d", run.Number, run.SubNumber)
	tmp["cds.run.number"] = fmt.Sprintf("%d", run.Number)
	tmp["cds.run.subnumber"] = fmt.Sprintf("%d", run.SubNumber)

	errm := &sdk.MultiError{}

	params := []sdk.Parameter{}
	for k, v := range tmp {
		s, err := sdk.Interpolate(v, tmp)
		if err != nil {
			errm.Append(err)
			continue
		}
		sdk.AddParameter(&params, k, sdk.StringParameter, s)
	}

	if errm.IsEmpty() {
		return params, nil
	}

	return params, errm
}
