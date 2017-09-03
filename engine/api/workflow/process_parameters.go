package workflow

import (
	"fmt"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func getNodeJobRunParameters(db gorp.SqlExecutor, j sdk.Job, run *sdk.WorkflowNodeRun, stage *sdk.Stage) ([]sdk.Parameter, error) {
	params, err := getNodeRunBuildParameters(db, run)
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

// GetNodeBuildParameters returns build parameters with default values for cds.version, cds.run, cds.run.number, cds.run.subnumber
func GetNodeBuildParameters(proj *sdk.Project, w *sdk.Workflow, n *sdk.WorkflowNode, pipelineParameters []sdk.Parameter, payload interface{}) ([]sdk.Parameter, error) {
	vars := map[string]string{}
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
	tmp = sdk.ParametersFromPipelineParameters(pipelineParameters)
	for k, v := range tmp {
		vars[k] = v
	}

	// compute payload
	log.Debug("GetNodeBuildParameters> compute payload :%#v", payload)
	errm := &sdk.MultiError{}
	payloadMap, errdump := dump.ToMap(payload, dump.WithLowerCaseFormatter())
	if errdump != nil {
		log.Error("GetNodeBuildParameters> do-dump error: %v", errdump)
		errm.Append(errdump)
	}
	for k, v := range payloadMap {
		tmp[k] = v
	}

	tmp["cds.project"] = w.ProjectKey
	tmp["cds.workflow"] = w.Name
	tmp["cds.pipeline"] = n.Pipeline.Name
	tmp["cds.version"] = fmt.Sprintf("%d.%d", 1, 0)
	tmp["cds.run"] = fmt.Sprintf("%d.%d", 1, 0)
	tmp["cds.run.number"] = fmt.Sprintf("%d", 1)
	tmp["cds.run.subnumber"] = fmt.Sprintf("%d", 0)

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

func getNodeRunBuildParameters(db gorp.SqlExecutor, run *sdk.WorkflowNodeRun) ([]sdk.Parameter, error) {
	//Load workflow run
	w, err := LoadRunByID(db, run.WorkflowRunID)
	if err != nil {
		return nil, sdk.WrapError(err, "getNodeRunParameters> Unable to load workflow run")
	}

	//Load node definition
	n := w.Workflow.GetNode(run.WorkflowNodeID)
	if n == nil {
		return nil, sdk.WrapError(fmt.Errorf("Unable to find node %d in workflow", run.WorkflowNodeID), "getNodeRunParameters>")
	}

	//Load project
	proj, err := project.Load(db, w.Workflow.ProjectKey, nil, project.LoadOptions.WithVariables)
	if err != nil {
		return nil, sdk.WrapError(err, "getNodeRunParameters> Unable to load project")
	}

	//Get node build parameters
	errm := &sdk.MultiError{}
	params, errparam := GetNodeBuildParameters(proj, &w.Workflow, n, run.PipelineParameters, run.Payload)
	if errparam != nil {
		err, ok := errparam.(*sdk.MultiError)
		if ok {
			errm = err
		} else {
			return nil, sdk.WrapError(err, "getNodeRunParameters> Unable to compute node build parameters")
		}
	}

	//override default parameters value
	tmp := sdk.ParametersToMap(params)
	tmp["cds.version"] = fmt.Sprintf("%d.%d", run.Number, run.SubNumber)
	tmp["cds.run"] = fmt.Sprintf("%d.%d", run.Number, run.SubNumber)
	tmp["cds.run.number"] = fmt.Sprintf("%d", run.Number)
	tmp["cds.run.subnumber"] = fmt.Sprintf("%d", run.SubNumber)

	params = []sdk.Parameter{}
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
